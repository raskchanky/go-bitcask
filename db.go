package bitcask

import (
	"fmt"
	"hash/crc64"
	"os"
	"path"
	"sync"
	"time"

	"google.golang.org/protobuf/proto"
)

type DB struct {
	path       string
	activeFile *os.File
	data       map[string][]byte // this is likely sub-optimal
	m          sync.RWMutex
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
	f, err := os.OpenFile(path.Join(baseDir, fileName), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}

	d.activeFile = f
	d.data = make(map[string][]byte)
	return d, nil
}

func (d *DB) Get(key []byte) ([]byte, error) {
	return nil, nil
}

func (d *DB) Put(key string, val []byte) error {
	// construct a new entry using the key/val
	entry, err := d.makeEntry(key, val)
	if err != nil {
		return err
	}

	// append to the active file
	err = d.appendToActiveFile(entry)
	if err != nil {
		return err
	}

	// update in memory map
	d.m.Lock()
	d.data[key] = copyBytes(val)
	d.m.Unlock()
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

func (d *DB) appendToActiveFile(entry *Entry) error {
	data, err := proto.Marshal(entry)
	if err != nil {
		return err
	}

	_, err = d.activeFile.Write(data)
	if err != nil {
		return err
	}

	return nil
}

func (d *DB) makeEntry(key string, val []byte) (*Entry, error) {
	ed := &EntryData{
		Timestamp: int32(time.Now().Unix()),
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
