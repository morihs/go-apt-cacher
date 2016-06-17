package aptcacher

import (
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"sync"
)

const (
	defaultTempdir = "."
	defaultListen  = "localhost:8000"
)

type Entry struct {
	ready chan struct{}
}

type Repo struct {
	base    string
	release []string
}

type CacheManager struct {
	base string

	mu      sync.Mutex
	cache   map[string]*Entry
	relHash ReleaseHashMap
	pkgHash PackagesHashMap
}

func New(base string) *CacheManager {
	return &CacheManager{
		base:    base,
		cache:   make(map[string]*Entry),
		relHash: make(ReleaseHashMap),
		pkgHash: make(PackagesHashMap),
	}
}

func (cm *CacheManager) Cache(pkgPath string) *Entry {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	// https://blog.golang.org/context

	entry, ok := cm.cache[pkgPath]
	if !ok {
		entry = &Entry{ready: make(chan struct{})}
		cm.cache[pkgPath] = entry
		cm.mu.Unlock()

		res, err := http.Get(cm.base + pkgPath)
		if err != nil {
			//TODO
			panic("failed to get")
		}
		defer res.Body.Close()

		tmp, err := ioutil.TempFile(defaultTempdir, "test-")
		if err != nil {
			//TODO
			panic("failed to create a temp file")
		}

		io.Copy(tmp, res.Body)

		if err := os.MkdirAll(path.Dir(pkgPath), 0755); err != nil {
			fmt.Println(err)
		}

		if err := os.Rename(tmp.Name(), pkgPath); err != nil {
			fmt.Println(err)
		}

		if isRelease(pkgPath) {
			go cm.UpdateReleaseHashMap(pkgPath)
		} else if isPackages(pkgPath) {
			go cm.UpdatePackagesHashMap(pkgPath)
		}

		close(entry.ready)
	} else {
		cm.mu.Unlock()
		<-entry.ready
	}

	return entry
}

func isRelease(pkgPath string) bool {
	return path.Base(pkgPath) == "Release"
}

func (cm *CacheManager) UpdateReleaseHashMap(pkgPath string) {
	r, _ := os.Open(pkgPath)
	relHash, err := GetReleaseHashMap(r)
	if err != nil {
		return
	}
	updated := cm.relHash.Update(relHash)

	for _, pkgHash := range updated {
		cm.Invalidate(pkgHash)
		cm.Cache(pkgHash)
	}
}

func isPackages(pkgPath string) bool {
	return path.Base(pkgPath) == "Packages.gz"
}

func (cm *CacheManager) UpdatePackagesHashMap(pkgPath string) {
	r, _ := os.Open(pkgPath)
	//wrap with gzip.Reader to parse Packages.gz
	gzipReader, _ := gzip.NewReader(r)
	pkgHash, err := GetPackagesHashMap(gzipReader)
	if err != nil {
		return
	}
	updated := cm.pkgHash.Update(pkgHash)

	for _, pkgHash := range updated {
		cm.Invalidate(pkgHash)
	}
}

func (cm *CacheManager) Invalidate(pkgPath string) {
	//TODO
	cm.mu.Lock()
	defer cm.mu.Unlock()

	delete(cm.cache, pkgPath)
}

func (cm *CacheManager) Serve(w http.ResponseWriter, req *http.Request) {
	cm.Cache(req.URL.Path)
	http.ServeFile(w, req, req.URL.Path)
}
