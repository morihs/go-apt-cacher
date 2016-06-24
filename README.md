[![Build Status](https://travis-ci.org/cybozu-go/go-apt-cacher.svg?branch=master)](https://travis-ci.org/cybozu-go/go-apt-cacher)

go-apt-cacher is a caching reverse proxy built specifically for Debian (APT)
repositories.

Features
--------

* Automatic checksum validation for cached files

    Cached files will **never** be broken!

* Reverse proxy for http and https repositories
* LRU-based cache eviction
* Smart caching strategy specialized for APT

Install
-------

Use Go 1.6 or better.

```
go get github.com/cybozu-go/go-apt-cacher/cmd/go-apt-cacher
```

License
-------

[MIT](https://opensource.org/licenses/MIT)
