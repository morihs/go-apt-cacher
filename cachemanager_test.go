package aptcacher

import (
	"io/ioutil"
	"os"
	"testing"
)

func TestCacheManager(t *testing.T) {
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
}
