package main

import (
	"flag"
	"net/http"
	"os"
	"time"

	aptcacher "github.com/cybozu-go/go-apt-cacher"
	"github.com/cybozu-go/log"
)

func main() {
	flag.Parse()
	args := flag.Args()

	if len(args) < 1 {
		log.Error("Usage: go-apt-cacher REPO")
		os.Exit(1)
	}

	cm := aptcacher.New(args[0])
	http.HandleFunc("/", cm.Serve)
	go func() { log.Fatal(http.ListenAndServe(":3142", nil)) }()

	//http://qiita.com/ruiu/items/1ea0c72088ad8f2b841e
	timer := time.Tick(3 * time.Second)

	for s := range timer {
		// update Releases
	}
}
