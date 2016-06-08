package main

import (
	"flag"
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

type entry struct {
	filepath string
	ready    chan struct{}
}

type cacheManager struct {
	base  string
	mu    sync.Mutex
	cache map[string]*entry
}

func (cm *cacheManager) download(w http.ResponseWriter, req *http.Request) {
	cm.mu.Lock()
	path := req.URL.Path
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
		close(e.ready)
	} else {
		cm.mu.Unlock()
		<-e.ready
	}

	http.ServeFile(w, req, e.filepath)
}

func main() {
	flag.Parse()
	args := flag.Args()

	if len(args) != 1 {
		log.Fatal("Wrong number of arguments.")
	}

	cm := cacheManager{base: args[0], cache: make(map[string]*entry)}

	http.HandleFunc("/", cm.download)

	log.Fatal(http.ListenAndServe(defaultListen, nil))
}
