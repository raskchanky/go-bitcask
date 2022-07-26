package bitcask

import (
	"bytes"
	"errors"
	"fmt"
	"hash/crc64"
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
)

var (
	ErrNotFound         = errors.New("bitcask: not found")
	ErrChecksumMismatch = errors.New("bitcask: checksum mismatch")
	tombstoneValue      = []byte("\U0001FAA6")
)

type DB struct {
	baseDir    string
	activeFile *os.File
	data       map[string]*KeyDir
	m          sync.RWMutex
	offset     *atomic.Int64
}

// Open creates a new database at the given path and returns a handle to it along with any errors.
func Open(baseDir string) (*DB, error) {
	d := &DB{
		baseDir: baseDir,
		offset:  atomic.NewInt64(0),
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
	d.m.RLock()
	defer d.m.RUnlock()

	kd, ok := d.data[key]
	if !ok {
		return nil, ErrNotFound
	}

	buf := make([]byte, kd.ValueSize)
	// TODO: right now this reads straight from the active file but it should indirect to whatever file kd.FileId points to
	_, err := d.activeFile.ReadAt(buf, kd.ValueOffset)
	if err != nil {
		return nil, err
	}

	var entry Entry
	err = proto.Unmarshal(buf, &entry)
	if err != nil {
		return nil, err
	}

	ed, err := proto.Marshal(entry.EntryData)
	if err != nil {
		return nil, err
	}

	// check the crc
	crc := entryDataChecksum(ed)
	if crc != entry.Crc {
		return nil, ErrChecksumMismatch
	}

	return copyBytes(entry.EntryData.Value), nil
}

func (d *DB) Put(key string, val []byte) error {
	d.m.Lock()
	defer d.m.Unlock()

	currentOffset := d.offset.Load()

	// construct a new entry using the key/val
	entry, err := d.makeEntry(key, val, bytes.Equal(tombstoneValue, val))
	if err != nil {
		return err
	}

	// append to the active file
	num, err := d.appendToActiveFile(entry)
	if err != nil {
		return err
	}
	d.offset.Add(int64(num))

	// update in memory map
	d.data[key] = &KeyDir{
		FileId:      d.activeFile.Name(),
		ValueSize:   int64(num),
		ValueOffset: currentOffset,
		Timestamp:   time.Now().UnixNano(),
	}

	return nil
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

func (d *DB) appendToActiveFile(entry *Entry) (int, error) {
	data, err := proto.Marshal(entry)
	if err != nil {
		return 0, err
	}

	// TODO: we might need to Sync() here
	return d.activeFile.WriteAt(data, d.offset.Load())
}

func (d *DB) makeEntry(key string, val []byte, tombstone bool) (*Entry, error) {
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
			time.Sleep(10 * time.Millisecond)
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

		go d.compactStableFiles()
	}
}

func (d *DB) compactStableFiles() {
	// d.m.RLock()
	// baseDir := d.baseDir
	// d.m.RUnlock()
	//
	// f, err := os.Open(baseDir)
	// if err != nil {
	// 	// TODO: handle this more gracefully
	// 	panic(err)
	// }
	//
	// dirEntries, err := f.ReadDir(0)
	// if err != nil {
	// 	// TODO: handle this more gracefully
	// 	panic(err)
	// }
	//
	// for _, de := range dirEntries {
	// 	if !strings.HasSuffix(de.Name(), ".stable") {
	// 		continue
	// 	}
	//
	// 	sf, err := os.Open(de.Name())
	// 	if err != nil {
	// 		// TODO: handle this more gracefully
	// 		panic(err)
	// 	}
	// }
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

func copyBytes(val []byte) []byte {
	newVal := make([]byte, len(val))
	copy(newVal, val)
	return newVal
}

func entryDataChecksum(ed []byte) uint64 {
	return crc64.Checksum(ed, crc64.MakeTable(crc64.ECMA))
}
