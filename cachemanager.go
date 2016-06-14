package aptcacher

import (
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
)

const (
	defaultTempdir = "."
	defaultListen  = "localhost:8000"
)

type Entry struct {
	filepath string
	size     int64
	ready    chan struct{}
}

type Repo struct {
	base    string
	release []string
}

type CacheManager struct {
	base  string
	mu    sync.Mutex
	cache map[string]*Entry
}

func New(base string) *CacheManager {
	return &CacheManager{base: base, cache: make(map[string]*Entry)}
}

func (cm *CacheManager) Cache(path string) *Entry {
	cm.mu.Lock()
	e := cm.cache[path]
	if e == nil {
		e = &Entry{ready: make(chan struct{})}
		cm.cache[path] = e
		cm.mu.Unlock()

		res, err := http.Get(cm.base + path)
		if err != nil {
			//TODO
			log.Fatal(err)
			panic("failed to get")
		}
		defer res.Body.Close()

		tmp, err := ioutil.TempFile(defaultTempdir, "test-")
		if err != nil {
			//TODO
			panic("failed to create a temp file")
		}

		io.Copy(tmp, res.Body)

		e.filepath = tmp.Name()

		stat, err := tmp.Stat()
		if err != nil {
			//TODO
			panic("failed to stat a temp file")
		}
		e.size = stat.Size()

		close(e.ready)
	} else {
		cm.mu.Unlock()
		<-e.ready
	}

	return e
}

func (cm *CacheManager) Serve(w http.ResponseWriter, req *http.Request) {
	e := cm.Cache(req.URL.Path)
	http.ServeFile(w, req, e.filepath)
}
