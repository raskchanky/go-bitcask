package bitcask

import (
	"bytes"
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

func TestGet(t *testing.T) {
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

	val, err := d.Get("lol")
	if err == nil {
		t.Fatal("expected an error for an unknown key")
	}
	if err != ErrNotFound {
		t.Fatal("expected a not found error for an unknown key")
	}
	if val != nil {
		t.Fatal("expected a nil byte slice for an unknown key")
	}

	val, err = d.Get("foo")
	if err != nil {
		t.Fatalf("expected a nil error but got %v\n", err)
	}
	if !bytes.Equal(val, []byte("bar")) {
		t.Fatalf("expected val to be %q, but it was %q", "bar", string(val))
	}
}

func TestDelete(t *testing.T) {
	d, err := testDB(t)
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = d.Close() }()
	defer func() { _ = os.RemoveAll(d.Path()) }()

	// add a kv pair
	err = d.Put("foo", []byte("bar"))
	if err != nil {
		t.Fatal(err)
	}

	// make sure it's there
	val, err := d.Get("foo")
	if err != nil {
		t.Fatalf("expected a nil error but got %v\n", err)
	}
	if !bytes.Equal(val, []byte("bar")) {
		t.Fatalf("expected val to be %q, but it was %q", "bar", string(val))
	}

	// delete it
	err = d.Delete("foo")
	if err != nil {
		t.Fatal(err)
	}

	// shouldn't be able to find it any more
	val, err = d.Get("foo")
	if err == nil {
		t.Fatal("expected an error for a deleted key")
	}
	if err != ErrNotFound {
		t.Fatal("expected a not found error for a deleted key")
	}
	if val != nil {
		t.Fatal("expected a nil byte slice for a deleted key")
	}
}
