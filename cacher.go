package aptcacher

// This file implements core logics to download and cache APT
// repository items.

import (
	"bytes"
	"flag"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"sync"
	"time"

	"github.com/cybozu-go/log"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
	"golang.org/x/net/context/ctxhttp"
)

const (
	// DefaultCheckInterval is the default interval to check
	// updates of Release/InRelease files.  Default is 15 seconds.
	DefaultCheckInterval = 15

	// DefaultCachePeriod is the default period to cache bad HTTP
	// response statuses.  Default is 3 seconds.
	DefaultCachePeriod = 3

	requestTimeout = 30 * time.Minute
)

var (
	checkInterval = flag.Int("interval", DefaultCheckInterval,
		"check interval for Release/InRelease")
	cachePeriod = flag.Int("cacheperiod", DefaultCachePeriod,
		"cache period for bad HTTP response statuses")
)

// Cacher downloads and caches APT indices and deb files.
type Cacher struct {
	meta   *Storage
	items  *Storage
	um     URLMap
	ctx    context.Context
	client *http.Client

	fiLock sync.RWMutex
	info   map[string]*FileInfo

	dlLock     sync.RWMutex
	dlChannels map[string]chan struct{}
	results    map[string]int
}

// NewCacher constructs Cacher.
//
// meta is a pointer to Storage to store meta data files.
// cache is a pointer to Storage to cache debs and other files.
func NewCacher(ctx context.Context, meta, cache *Storage, um URLMap) (*Cacher, error) {
	if err := meta.Load(); err != nil {
		return nil, errors.Wrap(err, "meta.Load")
	}
	if err := cache.Load(); err != nil {
		return nil, errors.Wrap(err, "cache.Load")
	}

	c := &Cacher{
		meta:       meta,
		items:      cache,
		um:         um,
		ctx:        ctx,
		client:     &http.Client{},
		info:       make(map[string]*FileInfo),
		dlChannels: make(map[string]chan struct{}),
	}

	metas := meta.ListAll()
	for _, fi := range metas {
		f, err := meta.Lookup(fi)
		if err != nil {
			return nil, errors.Wrap(err, "meta.Lookup")
		}
		fil, err := ExtractFileInfo(fi.path, f)
		f.Close()
		if err != nil {
			return nil, errors.Wrap(err, "ExtractFileInfo("+fi.path+")")
		}
		for _, fi2 := range fil {
			c.info[fi2.path] = fi2
		}
	}

	// add meta files w/o checksums (Release, Release.pgp, and InRelease).
	for _, fi := range metas {
		if _, ok := c.info[fi.path]; !ok {
			c.info[fi.path] = fi
			c.maintMeta(fi.path)
		}
	}

	return c, nil
}

func (c *Cacher) maintMeta(p string) {
	switch path.Base(p) {
	case "Release":
		go c.maintRelease(p, true)
	case "InRelease":
		go c.maintRelease(p, false)
	}
}

func (c *Cacher) maintRelease(p string, withGPG bool) {
	ticker := time.NewTicker(time.Duration(*checkInterval) * time.Second)
	defer ticker.Stop()

	if log.Enabled(log.LvDebug) {
		log.Debug("maintRelease", map[string]interface{}{
			"_path": p,
		})
	}

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			ch1 := c.Download(p, nil)
			if withGPG {
				ch2 := c.Download(p+".gpg", nil)
				<-ch2
			}
			<-ch1
		}
	}
}

// Download downloads an item and caches it.
//
// If valid is not nil, the downloaded data is validated against it.
//
// The caller receives a channel that will be closed when the item
// is downloaded and cached.  If prefix of p is not registered
// in URLMap, nil is returned.
//
// Note that download may fail, or just invalidated soon.
// Users of this method should retry if the item is not cached
// or invalidated.
func (c *Cacher) Download(p string, valid *FileInfo) <-chan struct{} {
	u := c.um.URL(p)
	if u == nil {
		return nil
	}

	c.dlLock.Lock()
	defer c.dlLock.Unlock()

	ch, ok := c.dlChannels[p]
	if ok {
		return ch
	}

	ch = make(chan struct{})
	c.dlChannels[p] = ch
	go c.download(p, u, valid)
	return ch
}

// download is a goroutine to download an item.
func (c *Cacher) download(p string, u *url.URL, valid *FileInfo) {
	statusCode := http.StatusInternalServerError

	defer func() {
		c.dlLock.Lock()
		ch := c.dlChannels[p]
		delete(c.dlChannels, p)
		c.results[p] = statusCode
		c.dlLock.Unlock()
		close(ch)

		// invalidate result cache after some interval
		go func(ctx context.Context) {
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Duration(*cachePeriod) * time.Second):
			}
			c.dlLock.Lock()
			delete(c.results, p)
			c.dlLock.Unlock()
		}(c.ctx)
	}()

	ctx, cancel := context.WithTimeout(c.ctx, requestTimeout)
	defer cancel()

	resp, err := ctxhttp.Get(ctx, c.client, u.String())
	if err != nil {
		log.Warn("GET failed", map[string]interface{}{
			"_url": u.String(),
			"_err": err.Error(),
		})
		return
	}
	body, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()

	statusCode = resp.StatusCode
	if statusCode != 200 {
		return
	}

	if err != nil {
		log.Warn("GET failed", map[string]interface{}{
			"_url": u.String(),
			"_err": err.Error(),
		})
		return
	}

	fi := MakeFileInfo(p, body)
	if valid != nil && !valid.Same(fi) {
		log.Warn("downloaded data is not valid", map[string]interface{}{
			"_url": u.String(),
		})
		return
	}

	storage := c.items
	var fil []*FileInfo
	if IsMeta(p) {
		storage = c.meta
		fil, err = ExtractFileInfo(p, bytes.NewReader(body))
		if err != nil {
			log.Error("invalid meta data", map[string]interface{}{
				"_path": p,
				"_err":  err.Error(),
			})
			return
		}
	}

	c.fiLock.Lock()
	defer c.fiLock.Unlock()

	if err := storage.Insert(body, fi); err != nil {
		log.Error("could not save an item", map[string]interface{}{
			"_path": p,
			"_err":  err.Error(),
		})
		return
	}

	for _, fi2 := range fil {
		c.info[fi2.path] = fi2
	}
	if IsMeta(p) {
		_, ok := c.info[p]
		if !ok {
			// As this is the first time that downloaded meta file p,
			c.maintMeta(p)
		}
	}
	c.info[p] = fi
	log.Info("downloaded and cached", map[string]interface{}{
		"_path": p,
	})
}

// Get looks up a cached item, and if not found, downloads it
// from the upstream server.
//
// The return values are cached HTTP status code of the response from
// an upstream server, a pointer to os.File for the cache file,
// and error.
func (c *Cacher) Get(p string) (statusCode int, f *os.File, err error) {
	u := c.um.URL(p)
	if u == nil {
		return http.StatusNotFound, nil, nil
	}

	storage := c.items
	if IsMeta(p) {
		if !IsSupported(p) {
			// return 404 for unsupported compression algorithms
			return http.StatusNotFound, nil, nil
		}
		storage = c.meta
	}

RETRY:
	c.fiLock.RLock()
	fi, ok := c.info[p]
	c.fiLock.RUnlock()

	if ok {
		f, err := storage.Lookup(fi)
		switch err {
		case nil:
			return http.StatusOK, f, nil
		case ErrNotFound:
		default:
			log.Error("lookup failure", map[string]interface{}{
				"_err": err.Error(),
			})
			return http.StatusInternalServerError, nil, err
		}
	}

	// not found in storage.
	c.dlLock.RLock()
	ch, chOk := c.dlChannels[p]
	result, resultOk := c.results[p]
	c.dlLock.RUnlock()

	if resultOk && result != http.StatusOK {
		return result, nil, nil
	}
	if chOk {
		<-ch
	} else {
		<-c.Download(p, fi)
	}
	goto RETRY
}
