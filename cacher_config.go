package aptcacher

const (
	defaultCheckInterval = 15
	defaultCachePeriod   = 3
	defaultCacheCapacity = 1
)

// CacherConfig is a struct to read TOML configurations.
//
// Use https://github.com/BurntSushi/toml as follows:
//
//    var config CacherConfig
//    md, err := toml.DecodeFile("/path/to/config.toml", &config)
//    if err != nil {
//        ...
//    }
type CacherConfig struct {
	// CheckInterval specifies interval in seconds to check updates for
	// Release/InRelease files.
	//
	// Default is 15 seconds.
	CheckInterval int `toml:"check_interval"`

	// CachePeriod specifies the period to cache bad HTTP response statuses.
	//
	// Default is 3 seconds.
	CachePeriod int `toml:"cache_period"`

	// MetaDirectory specifies a directory to store APT meta data files.
	//
	// This must differ from CacheDirectory.
	MetaDirectory string `toml:"meta_dir"`

	// CacheDirectory specifies a directory to cache non-meta data files.
	//
	// This must differ from MetaDirectory.
	CacheDirectory string `toml:"cache_dir"`

	// CacheCapacity specifies how many bytes can be stored in CacheDirectory.
	//
	// Unit is GiB.  Default is 1 GiB.
	CacheCapacity int `toml:"cache_capacity"`

	// Mapping specifies mapping between prefixes and APT URLs.
	Mapping map[string]string `toml:"mapping"`
}
