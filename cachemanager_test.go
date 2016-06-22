package aptcacher

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"io/ioutil"
	"os"
	"path/filepath"
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

func TestCacheManagerLoad(t *testing.T) {
	t.Parallel()

	files := map[string][]byte{
		"a":    []byte{'a'},
		"bc":   []byte{'b', 'c'},
		"def":  []byte{'d', 'e', 'f'},
		"ghij": []byte{'g', 'h', 'i', 'j'},
	}

	dir, err := ioutil.TempDir("", "gotest")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	for k, v := range files {
		err := ioutil.WriteFile(filepath.Join(dir, k), v, 0644)
		if err != nil {
			t.Fatal(err)
		}
	}

	cm := NewCacheManager(dir, 0)
	cm.Load()

	f, err := cm.Lookup(&FileInfo{path: "a"})
	if err != nil {
		t.Error(err)
	}
	f.Close()
	f, err = cm.Lookup(&FileInfo{path: "bc"})
	if err != nil {
		t.Error(err)
	}
	f.Close()
	f, err = cm.Lookup(&FileInfo{path: "def"})
	if err != nil {
		t.Error(err)
	}
	f.Close()

	sha1sum := sha1.Sum(files["ghij"])
	f, err = cm.Lookup(&FileInfo{
		path:    "ghij",
		sha1sum: sha1sum[:],
	})
	if err != nil {
		t.Fatal(err)
	}

	data, err := ioutil.ReadAll(f)
	f.Close()
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Compare(files["ghij"], data) != 0 {
		t.Error(`bytes.Compare(files["ghij"], data) != 0`)
	}
}
