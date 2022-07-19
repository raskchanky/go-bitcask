package bitcask

type DB struct {
	path string
}

func Open(path string) (*DB, error) {
	return nil, nil
}

func (d *DB) Get(key []byte) ([]byte, error) {
	return nil, nil
}

func (d *DB) Put(key []byte, val []byte) error {
	return nil
}

func (d *DB) Delete(key []byte) error {
	return nil
}

func (d *DB) List() [][]byte {
	return nil
}

func (d *DB) Close() error {
	return nil
}
