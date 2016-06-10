package main

import (
	"flag"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"
)

const (
	defaultTempdir = "."
	defaultListen  = "localhost:8000"
)

type entry struct {
	filepath string
	size     int64
	ready    chan struct{}
}

type repo struct {
	base    string
	release []string
}

type cacheManager struct {
	base  string
	mu    sync.Mutex
	cache map[string]*entry
}

func (cm *cacheManager) get(path string) *entry {
	cm.mu.Lock()
	e := cm.cache[path]
	if e == nil {
		e = &entry{ready: make(chan struct{})}
		cm.cache[path] = e
		cm.mu.Unlock()

		res, err := http.Get(cm.base + path)
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

func (cm *cacheManager) download(w http.ResponseWriter, req *http.Request) {
	e := cm.get(req.URL.Path)
	http.ServeFile(w, req, e.filepath)
}

func main() {
	flag.Parse()
	args := flag.Args()

	if len(args) < 2 {
		log.Fatal("Wrong number of arguments.")
	}

	//repo := &repo{base: args[0], release: args[1:]}
	cm := cacheManager{base: args[0], cache: make(map[string]*entry)}
	http.HandleFunc("/", cm.download)

	timer := time.Tick(1 * time.Minute)
	for range timer {
		cm.get("dists/jessie/Release")
	}

	log.Fatal(http.ListenAndServe(defaultListen, nil))
}
