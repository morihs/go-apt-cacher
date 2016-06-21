package aptcacher

import (
	"bytes"
	"crypto/md5"
	"io/ioutil"
	"os"
	"testing"
)

func TestCacheManager(t *testing.T) {
	t.Parallel()

	dir, err := ioutil.TempDir("", "gotest")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	cm := NewCacheManager(dir, 0)

	err = cm.Insert([]byte{'a'}, "path/to/a")
	if err != nil {
		t.Fatal(err)
	}
	if cm.Len() != 1 {
		t.Error(`cm.Len() != 1`)
	}
	if cm.used != 1 {
		t.Error(`cm.used != 1`)
	}

	// overwrite
	err = cm.Insert([]byte{'a'}, "path/to/a")
	if err != nil {
		t.Fatal(err)
	}
	if cm.Len() != 1 {
		t.Error(`cm.Len() != 1`)
	}
	if cm.used != 1 {
		t.Error(`cm.used != 1`)
	}

	err = cm.Insert([]byte{'b', 'c'}, "path/to/bc")
	if err != nil {
		t.Fatal(err)
	}
	if cm.Len() != 2 {
		t.Error(`cm.Len() != 2`)
	}
	if cm.used != 3 {
		t.Error(`cm.used != 3`)
	}

	data := []byte{'d', 'a', 't', 'a'}
	md5sum := md5.Sum(data)

	err = cm.Insert(data, "data")
	if err != nil {
		t.Fatal(err)
	}

	f, err := cm.Lookup(&FileInfo{
		path:   "data",
		md5sum: md5sum[:],
	})
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	data2, err := ioutil.ReadAll(f)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Compare(data, data2) != 0 {
		t.Error(`bytes.Compare(data, data2) != 0`)
	}

	_, err = cm.Lookup(&FileInfo{
		path:   "data",
		md5sum: []byte{},
	})
	if err != ErrNotFound {
		t.Error(`err != ErrNotFound`)
	}

	err = cm.Delete("data")
	if err != nil {
		t.Fatal(err)
	}
	if cm.Len() != 2 {
		t.Error(`cm.Len() != 2`)
	}
	if cm.used != 3 {
		t.Error(`cm.used != 3`)
	}
}

func TestCacheManagerLRU(t *testing.T) {
	t.Parallel()

	dir, err := ioutil.TempDir("", "gotest")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	cm := NewCacheManager(dir, 3)

	err = cm.Insert([]byte{'a'}, "path/to/a")
	if err != nil {
		t.Fatal(err)
	}
	err = cm.Insert([]byte{'b', 'c'}, "path/to/bc")
	if err != nil {
		t.Fatal(err)
	}
	if cm.used != 3 {
		t.Error(`cm.used != 3`)
	}

	// a and bc will be purged
	err = cm.Insert([]byte{'d', 'e'}, "path/to/de")
	if err != nil {
		t.Fatal(err)
	}
	if cm.Len() != 1 {
		t.Error(`cm.Len() != 1`)
	}
	if cm.used != 2 {
		t.Error(`cm.used != 2`)
	}

	_, err = cm.Lookup(&FileInfo{path: "path/to/a"})
	if err != ErrNotFound {
		t.Error(`err != ErrNotFound`)
	}
	_, err = cm.Lookup(&FileInfo{path: "path/to/bc"})
	if err != ErrNotFound {
		t.Error(`err != ErrNotFound`)
	}

	err = cm.Insert([]byte{'a'}, "path/to/a")
	if err != nil {
		t.Fatal(err)
	}

	// touch de
	_, err = cm.Lookup(&FileInfo{path: "path/to/de"})
	if err != nil {
		t.Error(err)
	}

	// a will be purged
	err = cm.Insert([]byte{'f'}, "path/to/f")
	if err != nil {
		t.Fatal(err)
	}

	_, err = cm.Lookup(&FileInfo{path: "path/to/a"})
	if err != ErrNotFound {
		t.Error(`err != ErrNotFound`)
	}
	_, err = cm.Lookup(&FileInfo{path: "path/to/de"})
	if err != nil {
		t.Error(err)
	}
	_, err = cm.Lookup(&FileInfo{path: "path/to/f"})
	if err != nil {
		t.Error(err)
	}
}
