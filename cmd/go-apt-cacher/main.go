package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/cybozu-go/go-apt-cacher"
)

func main() {
	flag.Parse()
	args := flag.Args()

	if len(args) < 2 {
		log.Fatal("Wrong number of arguments.")
	}

	cm := aptcacher.New(args[0])
	http.HandleFunc("/", cm.Serve)
	log.Fatal(http.ListenAndServe("localhost:8000", nil))
}
