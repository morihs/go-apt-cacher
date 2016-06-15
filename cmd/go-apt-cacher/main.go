package main

import (
	"flag"
	"log"
	"net/http"
	"time"

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
	go func() { log.Fatal(http.ListenAndServe("localhost:8000", nil)) }()

	timer := time.Tick(3 * time.Second)

	for s := range timer {
		log.Print("Getting Releases...", s)
	}
}
