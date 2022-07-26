package bitcask

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
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

func TestCRUD(t *testing.T) {
	d, err := testDB(t)
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = d.Close() }()
	defer func() { _ = os.RemoveAll(d.Path()) }()

	// add 5 kv pairs
	for i := 0; i < 5; i++ {
		err = d.Put(fmt.Sprintf("key-%d", i), []byte(fmt.Sprintf("value-%d", i)))
		if err != nil {
			t.Fatal(err)
		}
	}

	// check to make sure they're all there
	for i := 0; i < 5; i++ {
		k := fmt.Sprintf("key-%d", i)
		v := []byte(fmt.Sprintf("value-%d", i))

		val, err := d.Get(k)
		if err != nil {
			t.Fatalf("expected a nil error but got %v\n", err)
		}
		if !bytes.Equal(val, v) {
			t.Fatalf("expected value for %q to be %q, but it was %q", k, string(v), string(val))
		}
	}

	// try to get something that doesn't exist and it should error
	val, err := d.Get("lol")
	if err == nil {
		t.Fatal("expected an error for a deleted key")
	}
	if err != ErrNotFound {
		t.Fatal("expected a not found error for a deleted key")
	}
	if val != nil {
		t.Fatal("expected a nil byte slice for a deleted key")
	}

	// delete one of the keys
	err = d.Delete("key-2")
	if err != nil {
		t.Fatal(err)
	}

	// shouldn't be able to find it any more
	val, err = d.Get("key-2")
	if err == nil {
		t.Fatal("expected an error for a deleted key")
	}
	if err != ErrNotFound {
		t.Fatal("expected a not found error for a deleted key")
	}
	if val != nil {
		t.Fatal("expected a nil byte slice for a deleted key")
	}

	// listing keys should return the remaining 4, in sorted order
	keys := d.List()
	expectedKeys := []string{"key-0", "key-1", "key-3", "key-4"}
	if !cmp.Equal(keys, expectedKeys, cmpopts.EquateEmpty(), cmpopts.SortSlices(func(a string, b string) bool { return a < b })) {
		t.Fatalf("expected %v to equal %v but it didn't", keys, expectedKeys)
	}
}
