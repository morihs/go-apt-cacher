package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/net/context"

	"github.com/BurntSushi/toml"
	aptcacher "github.com/cybozu-go/go-apt-cacher"
	"github.com/cybozu-go/log"
)

const (
	defaultConfigPath = "/etc/go-apt-cacher.toml"
	defaultAddress    = ":3142"
)

var (
	configPath    = flag.String("f", defaultConfigPath, "configuration file name.")
	listenAddress = flag.String("l", defaultAddress, "listen address.")
)

func main() {
	flag.Parse()

	var config aptcacher.CacherConfig
	md, err := toml.DecodeFile(*configPath, &config)
	if err != nil {
		log.ErrorExit(err)
	}
	if len(md.Undecoded()) > 0 {
		log.Error("invalid config keys", map[string]interface{}{
			"_keys": fmt.Sprintf("%#v", md.Undecoded()),
		})
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cacher, err := aptcacher.NewCacher(ctx, &config)
	if err != nil {
		log.ErrorExit(err)
	}

	l, err := net.Listen("tcp", *listenAddress)
	if err != nil {
		log.ErrorExit(err)
	}

	done := make(chan error, 1)
	go func() {
		done <- aptcacher.Serve(ctx, l, cacher)
	}()

	sig := make(chan os.Signal, 10)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	signal.Stop(sig)
	cancel()
	if err := <-done; err != nil {
		log.Error(err.Error(), nil)
	}
}
