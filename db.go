package bitcask

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc64"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"go.uber.org/atomic"
	"google.golang.org/protobuf/proto"
)

const (
	// each file can be 10MB in size
	fileSizeThreshold = int64(10 * 1024)
	sizeBufferSize    = 4
)

/*
TODO:
- Consider replacing protobufs with some other serialization format, e.g. msgpack and following the file format laid
  out in the paper more closely. I'm not entirely clear how much value there is here, other than adherence to the
  paper. Off the cuff, it feels like a wash.
- Implement hint files as described in the paper.
- Test file rotation
- Test stable file compaction
- Test concurrency
- Test performance
- Better error handling
*/

var (
	ErrNotFound         = errors.New("bitcask: not found")
	ErrChecksumMismatch = errors.New("bitcask: checksum mismatch")
	tombstoneValue      = []byte("\U0001FAA6")
	filenameRegex       = regexp.MustCompile(`^bitcask\.[0-9]+\.(active|stable)$`)
)

type DB struct {
	baseDir           string
	activeFile        *os.File
	data              map[string]*KeyDir
	m                 sync.RWMutex
	offset            *atomic.Int64
	compactionRunning *atomic.Bool
	compactionCh      chan struct{}
	skipRotation      *atomic.Bool
	skipCompaction    *atomic.Bool
	errCh             chan error
	writeCount        int
}

// Open creates a new database at the given path and returns a handle to it along with any errors.
func Open(baseDir string) (*DB, error) {
	d := &DB{
		baseDir:           baseDir,
		offset:            atomic.NewInt64(0),
		compactionRunning: atomic.NewBool(false),
		data:              make(map[string]*KeyDir),
		compactionCh:      make(chan struct{}),
		skipRotation:      atomic.NewBool(false),
		skipCompaction:    atomic.NewBool(false),
		errCh:             make(chan error, 1024),
	}

	fi, err := os.Stat(baseDir)
	if err == nil && !fi.IsDir() {
		return nil, errors.New(fmt.Sprintf("%s is not a directory", baseDir))
	}

	err = os.MkdirAll(baseDir, 0o700)
	if err != nil {
		return nil, err
	}

	err = d.populateData()
	if err != nil {
		return nil, err
	}

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

	err = d.activeFile.Sync()
	if err != nil {
		return err
	}

	d.writeCount++

	if !d.skipRotation.Load() && d.writeCount > 100 {
		d.writeCount = 0
		go d.fileRotation(fileSizeThreshold)
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
	// Wait for compaction to finish if it's in progress
	if d.compactionRunning.Load() {
		<-d.compactionCh
	}

	close(d.compactionCh)
	close(d.errCh)

	d.m.Lock()
	err := d.activeFile.Close()
	d.m.Unlock()

	if err != nil {
		return err
	}

	return nil
}

func (d *DB) Path() string {
	d.m.RLock()
	defer d.m.RUnlock()
	return d.baseDir
}

func (d *DB) Size() (int64, error) {
	baseDir := d.Path()
	var size int64

	f, err := os.Open(baseDir)
	if err != nil {
		return 0, err
	}
	defer func() { _ = f.Close() }()

	dirEntries, err := f.ReadDir(0)
	if err != nil {
		return 0, err
	}

	if len(dirEntries) == 0 {
		return 0, nil
	}

	for _, de := range dirEntries {
		if !filenameRegex.MatchString(de.Name()) {
			continue
		}

		dePath := filepath.Join(baseDir, de.Name())
		fi, err := os.Stat(dePath)
		if err != nil {
			return 0, err
		}

		size += fi.Size()
	}

	return size, nil
}

// populateData checks to see if there are files in the base directory. If so, it uses them to populate
// a new KeyDir. If not, it creates a new active file.
// TODO: this should be using hint files if they're available
func (d *DB) populateData() error {
	d.m.Lock()
	defer d.m.Unlock()

	var activeFile string
	f, err := os.Open(d.baseDir)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	dirEntries, err := f.ReadDir(0)
	if err != nil {
		return err
	}

	if len(dirEntries) == 0 {
		return d.newActiveFile()
	}

	for _, de := range dirEntries {
		if !filenameRegex.MatchString(de.Name()) {
			// TODO: should this be logged somehow?
			continue
		}

		dePath := filepath.Join(d.baseDir, de.Name())
		sf, err := os.Open(dePath)
		if err != nil {
			return err
		}

		sfOffset := int64(0)
		// TODO: this would probably work better with some kind of buffering, e.g. read in
		// 4K at a time from the file and process that, rather than reading record at a time.
		// One bummer is there's no way to know where the boundary of a protobuf is. Even if we
		// followed the file format described in the paper more closely, it's not entirely clear
		// if that would be better than protobufs for this particular problem (I'm guessing not).
		// It's also not clear if this is an actual bottleneck or an imaginary one.
		for {
			entry, newOffset, err := readProtoFromFile(sf, sfOffset)
			if err != nil {
				if err == io.EOF {
					break
				} else {
					return err
				}
			}

			// deleted values should get removed if they've already been populated from
			// previous records or not added if this is the first time we're seeing them.
			if bytes.Equal(entry.EntryData.Value, tombstoneValue) {
				_, ok := d.data[entry.EntryData.Key]

				if ok {
					delete(d.data, entry.EntryData.Key)
				}
			} else {
				kd := &KeyDir{
					Timestamp: entry.EntryData.Timestamp,
					FileId:    dePath,
					Offset:    sfOffset,
				}

				d.data[entry.EntryData.Key] = kd
			}

			sfOffset = newOffset
		}

		if filepath.Ext(dePath) == ".active" {
			d.offset.Store(sfOffset)
			activeFile = dePath
		}

		_ = sf.Close()
	}

	if activeFile != "" {
		err = d.storeActiveFile(activeFile)
		if err != nil {
			return err
		}
	}

	return nil
}

func (d *DB) fileRotation(sizeThreshold int64) {
	d.m.Lock()
	defer d.m.Unlock()

	log.Println("starting active file rotation")

	fi, err := d.activeFile.Stat()
	if err != nil {
		log.Printf("Error during file rotation: %v", err)
		d.errCh <- err
	}

	if fi.Size() < sizeThreshold {
		return
	}

	// file has exceeded the threshold, close it and start a new one
	err = d.activeFile.Sync()
	if err != nil {
		log.Printf("Error during file rotation: %v", err)
		d.errCh <- err
	}

	err = d.activeFile.Close()
	if err != nil {
		log.Printf("Error during file rotation: %v", err)
		d.errCh <- err
	}

	// rename existing file
	fullPath := filepath.Join(d.baseDir, fi.Name())
	parts := strings.Split(fi.Name(), ".")
	newName := fmt.Sprintf("%s.%s.stable", parts[0], parts[1])
	newPath := filepath.Join(d.baseDir, newName)
	err = os.Rename(fullPath, newPath)

	// start a new file
	err = d.newActiveFile()
	if err != nil {
		log.Printf("Error during file rotation: %v", err)
		d.errCh <- err
	}

	if !d.skipCompaction.Load() && !d.compactionRunning.Load() {
		go d.compactStableFiles()
	}
}

// TODO: this should be writing out hint files
func (d *DB) compactStableFiles() {
	if d.compactionRunning.Load() {
		return
	}

	d.compactionRunning.Store(true)
	defer func() {
		d.compactionCh <- struct{}{}
		d.compactionRunning.Store(false)
	}()

	log.Println("starting stable file compaction")

	d.m.RLock()
	baseDir := d.baseDir
	d.m.RUnlock()

	toDelete := make(map[string]struct{})
	latestVersions := make(map[string]*Entry)

	f, err := os.Open(baseDir)
	if err != nil {
		log.Printf("Error during file compaction: %v", err)
		d.errCh <- err
	}
	defer func() { _ = f.Close() }()

	dirEntries, err := f.ReadDir(0)
	if err != nil {
		log.Printf("Error during file compaction: %v", err)
		d.errCh <- err
	}

	for _, de := range dirEntries {
		if ok, err := filepath.Match("bitcask.[0-9]*.*", de.Name()); !ok {
			if err != nil {
				log.Printf("Error during file compaction: %v", err)
				d.errCh <- err
			}
			continue
		}

		dePath := filepath.Join(baseDir, de.Name())
		if filepath.Ext(dePath) != ".stable" {
			continue
		}

		sf, err := os.Open(dePath)
		if err != nil {
			log.Printf("Error during file compaction: %v", err)
			d.errCh <- err
		}

		for {
			var sfOffset int64
			entry, sfOffset, err := readProtoFromFile(sf, sfOffset)
			if err != nil {
				if err == io.EOF {
					break
				} else {
					log.Printf("Error during file compaction: %v", err)
					d.errCh <- err
				}
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

		toDelete[dePath] = struct{}{}
		_ = sf.Close()
	}

	var currentSize, startingOffset, endingOffset int64
	var currentFile *os.File

	for _, entry := range latestVersions {
		if currentSize == 0 {
			newFileName := fmt.Sprintf("bitcask.%d.stable", time.Now().UTC().UnixNano())
			newFullPath := filepath.Join(baseDir, newFileName)
			currentFile, err = os.OpenFile(newFullPath, os.O_RDWR|os.O_CREATE, 0o600)
			if err != nil {
				log.Printf("Error during file compaction: %v", err)
				d.errCh <- err
			}
		}

		endingOffset, err = writeProtoToFile(currentFile, entry, startingOffset)
		if err != nil {
			log.Printf("Error during file compaction: %v", err)
			d.errCh <- err
		}
		currentSize += endingOffset - startingOffset
		startingOffset = endingOffset

		if currentSize >= fileSizeThreshold {
			err = currentFile.Close()
			if err != nil {
				log.Printf("Error during file compaction: %v", err)
				d.errCh <- err
			}
			currentSize = 0
		}
	}

	for k := range toDelete {
		err := os.Remove(k)
		if err != nil {
			log.Printf("Error during file compaction: %v", err)
			d.errCh <- err
		}
	}
}

// this should be called either at startup or with the lock held
func (d *DB) newActiveFile() error {
	fileName := fmt.Sprintf("bitcask.%d.active", time.Now().UTC().UnixNano())
	newFullPath := filepath.Join(d.baseDir, fileName)
	f, err := os.OpenFile(newFullPath, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return err
	}

	d.activeFile = f
	d.offset.Store(0)
	return nil
}

// this should be called either at startup or with the lock held
func (d *DB) storeActiveFile(path string) error {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return err
	}

	d.activeFile = f
	return nil
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

	return &Entry{
		Crc:       entryDataChecksum(edBytes),
		EntryData: ed,
	}, nil
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
