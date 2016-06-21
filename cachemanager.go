package aptcacher

import (
	"container/heap"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	"github.com/cybozu-go/log"
	"github.com/pkg/errors"
)

var (
	// ErrNotFound is returned by CacheManager.Lookup for non-existing items.
	ErrNotFound = errors.New("Not Found")
)

// entry represents an item in the cache.
type entry struct {
	*FileInfo
	size uint64

	// for container/heap.
	// atime is used as priorities.
	atime uint64
	index int
}

// CacheManager manages cache entries in APT repositories.
//
// Caches will be dropped in LRU fashion when the total size of items
// exceeds the capacity.
type CacheManager struct {
	dir      string // directory for cache items
	capacity uint64

	mu     sync.Mutex
	used   uint64
	cache  map[string]*entry
	lru    []*entry // for container/heap
	lclock uint64   // ditto
}

// New creates a CacheManager.
//
// dir is the directory for cached items.
// capacity is the maximum total size (bytes) of items in the cache.
// If capacity is zero, items will not be evicted.
func NewCacheManager(dir string, capacity uint64) *CacheManager {
	if !filepath.IsAbs(dir) {
		panic("dir must be an absolute path")
	}
	return &CacheManager{
		dir:      dir,
		cache:    make(map[string]*entry),
		capacity: capacity,
	}
}

// Len implements heap.Interface.
func (cm *CacheManager) Len() int {
	return len(cm.lru)
}

// Less implements heap.Interface.
func (cm *CacheManager) Less(i, j int) bool {
	return cm.lru[i].atime < cm.lru[j].atime
}

// Swap implements heap.Interface.
func (cm *CacheManager) Swap(i, j int) {
	cm.lru[i], cm.lru[j] = cm.lru[j], cm.lru[i]
	cm.lru[i].index = i
	cm.lru[j].index = j
}

// Push implements heap.Interface.
func (cm *CacheManager) Push(x interface{}) {
	e, ok := x.(*entry)
	if !ok {
		panic("CacheManager.Push: wrong type")
	}
	n := len(cm.lru)
	e.index = n
	cm.lru = append(cm.lru, e)
}

// Pop implements heap.Interface.
func (cm *CacheManager) Pop() interface{} {
	n := len(cm.lru)
	e := cm.lru[n-1]
	e.index = -1 // for safety
	cm.lru = cm.lru[0 : n-1]
	return e
}

// maint removes unused items from cache until used < capacity.
// cm.mu lock must be acquired beforehand.
func (cm *CacheManager) maint() {
	for cm.capacity > 0 && cm.used > cm.capacity {
		e := heap.Pop(cm).(*entry)
		delete(cm.cache, e.Path())
		cm.used -= e.size
		if err := os.Remove(filepath.Join(cm.dir, e.Path())); err != nil {
			log.Error("CacheManager.maint", map[string]interface{}{
				"_err": err.Error(),
			})
		}
		log.Info("CacheManager.maint removed", map[string]interface{}{
			"_path": e.Path(),
		})
	}
}

func readData(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return ioutil.ReadAll(f)
}

// Load loads existing items in filesystem.
func (cm *CacheManager) Load() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	wf := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		subpath, err := filepath.Rel(cm.dir, path)
		if err != nil {
			return err
		}
		if _, ok := cm.cache[subpath]; ok {
			return nil
		}

		data, err := readData(path)
		if err != nil {
			return err
		}

		size := uint64(info.Size())
		md5sum := md5.Sum(data)
		sha1sum := sha1.Sum(data)
		sha256sum := sha256.Sum256(data)
		e := &entry{
			FileInfo: &FileInfo{
				path:      subpath,
				md5sum:    md5sum[:],
				sha1sum:   sha1sum[:],
				sha256sum: sha256sum[:],
			},
			size:  size,
			atime: cm.lclock,
			index: len(cm.lru),
		}
		cm.used += size
		cm.lclock++
		cm.lru = append(cm.lru, e)
		cm.cache[subpath] = e
		log.Debug("CacheManager.Load", map[string]interface{}{
			"_path": subpath,
		})
		return nil
	}

	if err := filepath.Walk(cm.dir, wf); err != nil {
		return err
	}
	heap.Init(cm)

	cm.maint()

	return nil
}

// Insert inserts or updates a cache item.
func (cm *CacheManager) Insert(data []byte, path string) error {
	if len(path) == 0 {
		return errors.New("CacheManager.Insert: zero-length path")
	}

	f, err := ioutil.TempFile(cm.dir, "_tmp")
	if err != nil {
		return errors.Wrap(err, "CacheManager.Insert")
	}
	defer func() {
		f.Close()
		os.Remove(f.Name())
	}()

	_, err = f.Write(data)
	if err != nil {
		return errors.Wrap(err, "CacheManager.Insert")
	}
	err = f.Sync()
	if err != nil {
		return errors.Wrap(err, "CacheManager.Insert")
	}

	md5sum := md5.Sum(data)
	sha1sum := sha1.Sum(data)
	sha256sum := sha256.Sum256(data)
	destpath := filepath.Join(cm.dir, path)
	dirpath := filepath.Dir(destpath)

	_, err = os.Stat(dirpath)
	switch {
	case os.IsNotExist(err):
		err = os.MkdirAll(dirpath, 0755)
		if err != nil {
			return errors.Wrap(err, "CacheManager.Insert")
		}
	case err != nil:
		return errors.Wrap(err, "CacheManager.Insert")
	}

	cm.mu.Lock()
	defer cm.mu.Unlock()

	if existing, ok := cm.cache[path]; ok {
		err = os.Remove(destpath)
		if err != nil {
			return errors.Wrap(err, "CacheManager.Insert")
		}
		cm.used -= existing.size
		heap.Remove(cm, existing.index)
		delete(cm.cache, path)
		log.Info("deleted existing item", map[string]interface{}{
			"_path": path,
		})
	}

	err = os.Rename(f.Name(), destpath)
	if err != nil {
		return errors.Wrap(err, "CacheManager.Insert")
	}

	size := uint64(len(data))
	e := &entry{
		FileInfo: &FileInfo{
			path:      path,
			md5sum:    md5sum[:],
			sha1sum:   sha1sum[:],
			sha256sum: sha256sum[:],
		},
		size:  size,
		atime: cm.lclock,
	}
	cm.used += size
	cm.lclock++
	heap.Push(cm, e)
	cm.cache[path] = e

	cm.maint()

	return nil
}

// Lookup looks up an item in the cache.
// If no item matching fi is found, ErrNotFound is returned.
//
// The caller is responsible to close the retured os.File.
func (cm *CacheManager) Lookup(fi *FileInfo) (*os.File, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	e, ok := cm.cache[fi.path]
	if !ok {
		return nil, ErrNotFound
	}

	if !fi.Same(e.FileInfo) {
		// checksum mismatch
		return nil, ErrNotFound
	}

	e.atime = cm.lclock
	cm.lclock++
	heap.Fix(cm, e.index)
	return os.Open(filepath.Join(cm.dir, fi.path))
}

// Delete deletes an item from the cache.
func (cm *CacheManager) Delete(path string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	e, ok := cm.cache[path]
	if !ok {
		return nil
	}

	err := os.Remove(filepath.Join(cm.dir, path))
	if err != nil {
		return err
	}

	delete(cm.cache, path)
	heap.Remove(cm, e.index)
	return nil
}
