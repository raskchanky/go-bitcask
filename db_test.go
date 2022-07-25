package bitcask

import (
	"os"
	"testing"
)

func testDB(t *testing.T) (*DB, error) {
	t.Helper()

	dir, err := os.MkdirTemp("", "*")
	if err != nil {
		t.Fatal(err)
	}

	return Open(dir)
}

func TestOpen(t *testing.T) {
	dir, err := os.MkdirTemp("", "*")
	if err != nil {
		t.Fatal(err)
	}
	d, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = os.RemoveAll(dir) }()
	if err != nil {
		t.Fatal(err)
	}

	if d == nil {
		t.Fatalf("expected a non-nil database but got a nil one")
	}

	err = d.Close()
	if err != nil {
		t.Fatal(err)
	}
}

func TestPut(t *testing.T) {
	d, err := testDB(t)
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = d.Close() }()
	defer func() { _ = os.RemoveAll(d.Path()) }()

	err = d.Put("foo", []byte("bar"))
	if err != nil {
		t.Fatal(err)
	}
}
