package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"
)

const (
	tmpdir = "."
)

type entry struct {
	fd    *os.File
	ready chan struct{}
}

type cacheManager struct {
	base  string
	mu    sync.Mutex
	cache map[string]*entry
}

func (cm *cacheManager) donwload(path string) string {
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

		tmp, err := ioutil.TempFile(tmpdir, "test-")
		if err != nil {
			//TODO
			panic("failed to create a temp file")
		}
		io.Copy(tmp, res.Body)
		e.fd = tmp
		close(e.ready)
	} else {
		cm.mu.Unlock()
		<-e.ready
	}
	return e.fd.Name()
}

func main() {
	flag.Parse()
	args := flag.Args()

	if len(args) != 2 {
		log.Fatal("Wrong number of arguments.")
	}

	cm := cacheManager{base: args[0], cache: make(map[string]*entry)}

	fmt.Println("test", cm.donwload(args[1]))
}
