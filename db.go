package bitcask

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc64"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"go.uber.org/atomic"
	"google.golang.org/protobuf/proto"
)

const (
	// each file can be 10MB in size
	fileSizeThreshold = 10 * 1024 * 1024
	sizeBufferSize    = 4
)

var (
	ErrNotFound         = errors.New("bitcask: not found")
	ErrChecksumMismatch = errors.New("bitcask: checksum mismatch")
	tombstoneValue      = []byte("\U0001FAA6")
)

type DB struct {
	baseDir           string
	activeFile        *os.File
	data              map[string]*KeyDir
	m                 sync.RWMutex
	offset            *atomic.Int64
	compactionRunning *atomic.Bool
}

// Open creates a new database at the given path and returns a handle to it along with any errors.
func Open(baseDir string) (*DB, error) {
	d := &DB{
		baseDir:           baseDir,
		offset:            atomic.NewInt64(0),
		compactionRunning: atomic.NewBool(false),
	}

	err := os.MkdirAll(baseDir, 0o600)
	if err != nil {
		return nil, err
	}

	// TODO: this needs to be a bit smarter. are there files in the directory already? then populate this from
	// the existing files. If not, then create it fresh.
	d.data = make(map[string]*KeyDir)
	err = d.newActiveFile()
	if err != nil {
		return nil, err
	}

	go d.fileRotation()
	return d, nil
}

func (d *DB) Get(key string) ([]byte, error) {
	// look up the keydir associated with this key. if it's not there, we don't have it.
	d.m.RLock()
	kd, ok := d.data[key]
	d.m.RUnlock()

	if !ok {
		return nil, ErrNotFound
	}

	// kd.Offset tells us where to start reading. The first thing we'll read is the size,
	// in a fixed 4 byte buffer.
	f, err := os.Open(kd.FileId)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	entry, _, err := readProtoFromFile(f, kd.Offset)
	if err != nil {
		return nil, err
	}

	return copyBytes(entry.EntryData.Value), nil
}

func (d *DB) Put(key string, val []byte) error {
	d.m.Lock()
	defer d.m.Unlock()

	startingOffset := d.offset.Load()
	entry, err := makeEntry(key, val, bytes.Equal(tombstoneValue, val))
	if err != nil {
		return err
	}

	currentOffset, err := writeProtoToFile(d.activeFile, entry, startingOffset)
	d.offset.Store(currentOffset)

	// update in memory map, using the starting instead of ending offset
	d.data[key] = &KeyDir{
		FileId:    d.activeFile.Name(),
		Offset:    startingOffset,
		Timestamp: entry.EntryData.Timestamp,
	}

	return d.activeFile.Sync()
}

func (d *DB) Delete(key string) error {
	// write special tombstone record to the active file
	err := d.Put(key, tombstoneValue)
	if err != nil {
		return err
	}

	// remove keydir entry for given key
	d.m.Lock()
	delete(d.data, key)
	d.m.Unlock()

	return nil
}

func (d *DB) List() []string {
	d.m.RLock()
	defer d.m.RUnlock()

	keys := make([]string, 0, len(d.data))
	for k := range d.data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	return keys
}

func (d *DB) Close() error {
	return d.activeFile.Close()
}

func (d *DB) Path() string {
	return d.baseDir
}

func makeEntry(key string, val []byte, tombstone bool) (*Entry, error) {
	ed := &EntryData{
		Timestamp: time.Now().UnixNano(),
		Key:       key,
		Value:     copyBytes(val),
		Tombstone: tombstone,
	}

	edBytes, err := proto.Marshal(ed)
	if err != nil {
		return nil, fmt.Errorf("error marshaling data: %w", err)
	}

	entry := &Entry{
		Crc:       entryDataChecksum(edBytes),
		EntryData: ed,
	}

	return entry, nil
}

func (d *DB) fileRotation() {
	for {
		fi, err := d.activeFile.Stat()
		if err != nil {
			// TODO: handle this more gracefully
			panic(err)
		}

		if fi.Size() < fileSizeThreshold {
			time.Sleep(200 * time.Millisecond)
			continue
		}

		// file has exceeded the threshold, close it and start a new one
		d.m.Lock()
		err = d.activeFile.Sync()
		if err != nil {
			// TODO: handle this more gracefully
			panic(err)
		}

		err = d.activeFile.Close()
		if err != nil {
			// TODO: handle this more gracefully
			panic(err)
		}

		// rename existing file
		fullPath := filepath.Join(d.baseDir, fi.Name())
		parts := strings.Split(fi.Name(), ".")
		newPath := fmt.Sprintf("%s.stable", parts[0])
		err = os.Rename(fullPath, newPath)

		// start a new file
		err = d.newActiveFile()
		if err != nil {
			// TODO: handle this more gracefully
			panic(err)
		}
		d.m.Unlock()

		if !d.compactionRunning.Load() {
			go d.compactStableFiles()
		}
	}
}

func (d *DB) compactStableFiles() {
	d.compactionRunning.Store(true)
	defer d.compactionRunning.Store(false)

	d.m.RLock()
	baseDir := d.baseDir
	d.m.RUnlock()

	toDelete := make(map[string]struct{}, 0)
	latestVersions := make(map[string]*Entry)

	f, err := os.Open(baseDir)
	if err != nil {
		// TODO: handle this more gracefully
		panic(err)
	}

	dirEntries, err := f.ReadDir(0)
	if err != nil {
		// TODO: handle this more gracefully
		panic(err)
	}

	for _, de := range dirEntries {
		if filepath.Ext(de.Name()) != "stable" {
			continue
		}

		sf, err := os.Open(de.Name())
		if err != nil {
			// TODO: handle this more gracefully
			panic(err)
		}

		for {
			var sfOffset int64
			entry, sfOffset, err := readProtoFromFile(sf, sfOffset)
			if err == io.EOF {
				break
			}
			key := entry.EntryData.Key

			d.m.RLock()
			_, ok := d.data[key]
			d.m.RUnlock()

			if !ok {
				continue
			}

			lv, ok := latestVersions[key]
			if !ok || entry.EntryData.Timestamp > lv.EntryData.Timestamp {
				latestVersions[key] = entry
			}
		}

		toDelete[de.Name()] = struct{}{}
	}

	var currentSize, startingOffset, endingOffset int64
	var currentFile *os.File

	for _, entry := range latestVersions {
		if currentSize == 0 {
			newFileName := fmt.Sprintf("%d.stable", time.Now().UTC().UnixNano())
			newFullPath := filepath.Join(baseDir, newFileName)
			currentFile, err = os.OpenFile(newFullPath, os.O_RDWR|os.O_CREATE, 0o600)
			if err != nil {
				// TODO: handle this more gracefully
				panic(err)
			}
		}

		endingOffset, err = writeProtoToFile(currentFile, entry, startingOffset)
		if err != nil {
			// TODO: handle this more gracefully
			panic(err)
		}
		currentSize += endingOffset - startingOffset
		startingOffset = endingOffset

		if currentSize >= fileSizeThreshold {
			err = currentFile.Close()
			if err != nil {
				// TODO: handle this more gracefully
				panic(err)
			}
			currentSize = 0
		}
	}

	for k := range toDelete {
		err := os.Remove(k)
		if err != nil {
			// TODO: handle this more gracefully
			panic(err)
		}
	}
}

// this should be called either at startup or with the lock held
func (d *DB) newActiveFile() error {
	fileName := fmt.Sprintf("%d.active", time.Now().UTC().UnixNano())
	newFullPath := filepath.Join(d.baseDir, fileName)
	f, err := os.OpenFile(newFullPath, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return err
	}

	d.activeFile = f
	d.offset.Store(0)
	return nil
}

func writeProtoToFile(f *os.File, entry *Entry, offset int64) (int64, error) {
	// append to the active file, writing the size first
	data, err := proto.Marshal(entry)
	if err != nil {
		return 0, err
	}

	buf := make([]byte, sizeBufferSize)
	binary.LittleEndian.PutUint32(buf, uint32(len(data)))

	num, err := f.WriteAt(buf, offset)
	if err != nil {
		return 0, err
	}

	// Now that the size is written, append the encoded protobuf
	currentOffset := offset + int64(num)
	num, err = f.WriteAt(data, currentOffset)
	if err != nil {
		return 0, err
	}

	currentOffset += int64(num)
	return currentOffset, nil
}

func readProtoFromFile(f *os.File, offset int64) (*Entry, int64, error) {
	buf := make([]byte, sizeBufferSize)
	num, err := f.ReadAt(buf, offset)
	if err != nil {
		return nil, 0, err
	}

	offset += int64(num)
	readSize := binary.LittleEndian.Uint32(buf)
	msg := make([]byte, readSize)

	// Now that we know how big the proto is, read that out as well
	num, err = f.ReadAt(msg, offset)
	if err != nil {
		return nil, 0, err
	}

	offset += int64(num)
	var entry Entry
	err = proto.Unmarshal(msg, &entry)
	if err != nil {
		return nil, 0, err
	}

	ed, err := proto.Marshal(entry.EntryData)
	if err != nil {
		return nil, 0, err
	}

	// check the crc
	crc := entryDataChecksum(ed)
	if crc != entry.Crc {
		return nil, 0, ErrChecksumMismatch
	}

	return &entry, offset, nil
}

func copyBytes(val []byte) []byte {
	newVal := make([]byte, len(val))
	copy(newVal, val)
	return newVal
}

func entryDataChecksum(ed []byte) uint64 {
	return crc64.Checksum(ed, crc64.MakeTable(crc64.ECMA))
}
