package aptcacher

import (
	"compress/gzip"
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
	localPath string
	ready     chan struct{}
}

type Repo struct {
	base    string
	release []string
}

type CacheManager struct {
	base string

	mu       sync.Mutex
	cache    map[string]*Entry
	indices  Indices
	pkgIndex PackageIndex
}

func New(base string) *CacheManager {
	return &CacheManager{
		base:     base,
		cache:    make(map[string]*Entry),
		indices:  make(Indices),
		pkgIndex: make(PackageIndex),
	}
}

func (cm *CacheManager) Cache(remotePath string) *Entry {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	// https://blog.golang.org/context

	entry, ok := cm.cache[remotePath]
	if !ok {
		entry = &Entry{ready: make(chan struct{})}
		cm.cache[remotePath] = entry
		cm.mu.Unlock()

		res, err := http.Get(cm.base + remotePath)
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

		localPath := tmp.Name()
		entry.localPath = localPath

		if isRelease(remotePath) {
			go cm.UpdateIndices(localPath)
		} else if isPackages(remotePath) {
			go cm.UpdatePackageIndex(localPath)
		}

		stat, err := tmp.Stat()
		if err != nil {
			//TODO
			panic("failed to stat a temp file")
		}
		entry.size = stat.Size()

		close(e.ready)
	} else {
		cm.mu.Unlock()
		<-entry.ready
	}

	return entry
}

func isRelease(remotePath string) bool {
	return path.Base(remotePath) == "Release"
}

func (cm *CacheManager) UpdateIndices(localPath string) {
	r, _ := os.Open(localPath)
	indices, err := GetIndices(r)
	if err != nil {
		return
	}
	updated := cm.indices.Update(indices)

	for _, pkgIndex := range updated {
		cm.Invalidate(pkgIndex)
		cm.Cache(pkgIndex)
	}
}

func isPackages(remotePath string) bool {
	return path.Base(remotePath) == "Packages.gz"
}

func (cm *CacheManager) UpdatePackageIndex(remotePath string) {
	r, _ := os.Open(remotePath)
	//wrap with gzip.Reader to parse Packages.gz
	gr, _ := gzip.NewReader(r)
	pkgIndex, err := GetPackageIndex(gr)
	if err != nil {
		return
	}
	updated := cm.pkgIndex.Update(pkgIndex)

	for _, pkgIndex := range updated {
		cm.Invalidate(pkgIndex)
	}
}

func (cm *CacheManager) Invalidate(remotePath string) {
	//TODO
	cm.mu.Lock()
	defer cm.mu.Unlock()

	delete(cm.cache, remotePath)
}

func (cm *CacheManager) Serve(w http.ResponseWriter, req *http.Request) {
	e := cm.Cache(req.URL.Path)
	http.ServeFile(w, req, e.localPath)
}
