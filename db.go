package bitcask

import (
	"errors"
	"fmt"
	"hash/crc64"
	"os"
	"path"
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
)

type DB struct {
	path       string
	activeFile *os.File
	data       map[string]*KeyDir
	m          sync.RWMutex
	offset     *atomic.Int64
}

// Open creates a new database at the given path and returns a handle to it along with any errors.
func Open(baseDir string) (*DB, error) {
	d := &DB{
		path: baseDir,
	}

	err := os.MkdirAll(baseDir, 0o600)
	if err != nil {
		return nil, err
	}

	fileName := fmt.Sprintf("%d.active", time.Now().UTC().UnixNano())
	f, err := os.OpenFile(path.Join(baseDir, fileName), os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return nil, err
	}

	d.activeFile = f
	d.data = make(map[string]*KeyDir)
	d.offset = atomic.NewInt64(0)
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
	entry, err := d.makeEntry(key, val)
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

func (d *DB) Delete(key []byte) error {
	return nil
}

func (d *DB) List() [][]byte {
	return nil
}

func (d *DB) Close() error {
	return d.activeFile.Close()
}

func (d *DB) Path() string {
	return d.path
}

func (d *DB) appendToActiveFile(entry *Entry) (int, error) {
	data, err := proto.Marshal(entry)
	if err != nil {
		return 0, err
	}

	return d.activeFile.WriteAt(data, d.offset.Load())
}

func (d *DB) makeEntry(key string, val []byte) (*Entry, error) {
	ed := &EntryData{
		Timestamp: time.Now().UnixNano(),
		KeySize:   int64(len(key)),
		ValueSize: int64(len(val)),
		Key:       key,
		Value:     copyBytes(val),
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

func copyBytes(val []byte) []byte {
	newVal := make([]byte, len(val))
	copy(newVal, val)
	return newVal
}

func entryDataChecksum(ed []byte) uint64 {
	return crc64.Checksum(ed, crc64.MakeTable(crc64.ECMA))
}
