package aptcacher

// This file provides utilities for debian repository indices.

import (
	"encoding/hex"
	"io"
	"path"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

// IsMeta returns true if p points a debian repository index file
// containing checksums for other files.
func IsMeta(p string) bool {
	base := path.Base(p)

	// https://wiki.debian.org/RepositoryFormat#Compression_of_indices
	switch {
	case strings.HasSuffix(base, ".gz"):
		base = base[0 : len(base)-3]
	case strings.HasSuffix(base, ".bz2"):
		base = base[0 : len(base)-4]
	case strings.HasSuffix(base, ".xz"):
		base = base[0 : len(base)-3]
	case strings.HasSuffix(base, ".lzma"):
		base = base[0 : len(base)-5]
	case strings.HasSuffix(base, ".lz"):
		base = base[0 : len(base)-3]
	}

	switch base {
	case "Release", "Release.gpg", "InRelease":
		return true
	case "Packages", "Sources", "Index":
		return true
	}

	return false
}

func parseChecksum(l string) (p string, size uint64, csum []byte, err error) {
	flds := strings.Fields(l)
	if len(flds) != 3 {
		err = errors.New("invalid checksum line: " + l)
		return
	}

	n, err_ := strconv.ParseUint(flds[1], 10, 64)
	if err_ != nil {
		err = err_
		return
	}
	d, err_ := hex.DecodeString(flds[0])
	if err_ != nil {
		err = err_
		return
	}

	p = flds[2]
	size = n
	csum = d
	return
}

// GetFilesFromRelease parses Release or InRelease file and
// returns a list of *FileInfo pointed in the file.
//
// p is the local path.
func GetFilesFromRelease(p string, r io.Reader) ([]*FileInfo, error) {
	dir := path.Dir(p)

	d, err := NewParser(r).Read()
	if err != nil {
		return nil, errors.Wrap(err, "NewParser(r).Read()")
	}

	md5sums := d["MD5Sum"]
	sha1sums := d["SHA1"]
	sha256sums := d["SHA256"]

	if len(md5sums) == 0 && len(sha1sums) == 0 && len(sha256sums) == 0 {
		return nil, nil
	}

	m := make(map[string]*FileInfo)

	for _, l := range md5sums {
		p, size, csum, err := parseChecksum(l)
		p = path.Join(dir, path.Clean(p))
		if err != nil {
			return nil, errors.Wrap(err, "parseChecksum for md5sums")
		}

		fi := &FileInfo{
			path:   p,
			size:   size,
			md5sum: csum,
		}
		m[p] = fi
	}

	for _, l := range sha1sums {
		p, size, csum, err := parseChecksum(l)
		p = path.Join(dir, path.Clean(p))
		if err != nil {
			return nil, errors.Wrap(err, "parseChecksum for sha1sums")
		}

		fi, ok := m[p]
		if ok {
			fi.sha1sum = csum
		} else {
			fi := &FileInfo{
				path:    p,
				size:    size,
				sha1sum: csum,
			}
			m[p] = fi
		}
	}

	for _, l := range sha256sums {
		p, size, csum, err := parseChecksum(l)
		p = path.Join(dir, path.Clean(p))
		if err != nil {
			return nil, errors.Wrap(err, "parseChecksum for sha256sums")
		}

		fi, ok := m[p]
		if ok {
			fi.sha256sum = csum
		} else {
			fi := &FileInfo{
				path:      p,
				size:      size,
				sha256sum: csum,
			}
			m[p] = fi
		}
	}

	l := make([]*FileInfo, 0, len(m))
	for _, fi := range m {
		l = append(l, fi)
	}
	return l, nil
}

// GetFilesFromPackages parses Packages file and returns
// a list of *FileInfo pointed in the file.
//
// p is the local path.
func GetFilesFromPackages(p string, r io.Reader) ([]*FileInfo, error) {
	prefix := strings.SplitN(p, "/", 2)[0]

	var l []*FileInfo
	parser := NewParser(r)

	for {
		d, err := parser.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, errors.Wrap(err, "parser.Read")
		}

		filename, ok := d["Filename"]
		if !ok {
			return nil, errors.New("no Filename in " + p)
		}
		p := path.Join(prefix, path.Clean(filename[0]))

		strsize, ok := d["Size"]
		if !ok {
			return nil, errors.New("no Size in " + p)
		}
		size, err := strconv.ParseUint(strsize[0], 10, 64)
		if err != nil {
			return nil, err
		}

		fi := &FileInfo{
			path: p,
			size: size,
		}
		if csum, ok := d["MD5sum"]; ok {
			b, err := hex.DecodeString(csum[0])
			if err != nil {
				return nil, err
			}
			fi.md5sum = b
		}
		if csum, ok := d["SHA1"]; ok {
			b, err := hex.DecodeString(csum[0])
			if err != nil {
				return nil, err
			}
			fi.sha1sum = b
		}
		if csum, ok := d["SHA256"]; ok {
			b, err := hex.DecodeString(csum[0])
			if err != nil {
				return nil, err
			}
			fi.sha256sum = b
		}
		l = append(l, fi)
	}

	return l, nil
}

// GetFilesFromSources parses Sources file and returns
// a list of *FileInfo pointed in the file.
//
// p is the local path.
func GetFilesFromSources(p string, r io.Reader) ([]*FileInfo, error) {
	prefix := strings.SplitN(p, "/", 2)[0]

	var l []*FileInfo
	parser := NewParser(r)

	for {
		d, err := parser.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, errors.Wrap(err, "parser.Read")
		}

		dir, ok := d["Directory"]
		if !ok {
			return nil, errors.New("no Directory in " + p)
		}
		files, ok := d["Files"]
		if !ok {
			return nil, errors.New("no Files in " + p)
		}

		m := make(map[string]*FileInfo)
		for _, l := range files {
			fname, size, csum, err := parseChecksum(l)
			if err != nil {
				return nil, errors.Wrap(err, "parseChecksum for Files")
			}

			fpath := path.Join(prefix, dir[0], fname)
			fi := &FileInfo{
				path:   fpath,
				size:   size,
				md5sum: csum,
			}
			m[fpath] = fi
		}

		for _, l := range d["Checksums-Sha1"] {
			fname, _, csum, err := parseChecksum(l)
			if err != nil {
				return nil, errors.Wrap(err, "parseChecksum for Checksums-Sha1")
			}

			fpath := path.Join(prefix, dir[0], fname)
			fi, ok := m[fpath]
			if !ok {
				return nil, errors.New("mismatch between Files and Checksums-Sha1 in " + p)
			}
			fi.sha1sum = csum
		}

		for _, l := range d["Checksums-Sha256"] {
			fname, _, csum, err := parseChecksum(l)
			if err != nil {
				return nil, errors.Wrap(err, "parseChecksum for Checksums-Sha256")
			}

			fpath := path.Join(prefix, dir[0], fname)
			fi, ok := m[fpath]
			if !ok {
				return nil, errors.New("mismatch between Files and Checksums-Sha256 in " + p)
			}
			fi.sha256sum = csum
		}

		for _, fi := range m {
			l = append(l, fi)
		}
	}

	return l, nil
}

// GetFilesFromIndex parses i18n/Index file and returns
// a list of *FileInfo pointed in the file.
//
// p is the local path.
func GetFilesFromIndex(p string, r io.Reader) ([]*FileInfo, error) {
	return GetFilesFromRelease(p, r)
}