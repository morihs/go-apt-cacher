package aptcacher

import (
	"fmt"
	"io"
	"io/ioutil"
	"strings"
)

var defaultHashAlgorithms = []string{
	"MD5Sum",
	"SHA1",
	"SHA256",
	"SHA512",
}

type ReleaseHashMap map[string]string

func parseIndex(line string) (string, string, error) {
	// line should look like this:
	//                           (Hash)           (Size)                     (Path)
	// ead1cbf42ed119c50bf3aab28b5b6351          8234934 main/binary-amd64/Packages
	fields := strings.Fields(line)

	if len(fields) < 3 {
		err := fmt.Errorf("Failed to parse this fields: %s", line)
		return "", "", err
	}

	// Path, Hash, error
	return fields[2], fields[0], nil
}

func GetReleaseHashMap(r io.Reader) (ReleaseHashMap, error) {
	s, err := ioutil.ReadAll(r)
	dcfMap, err := parseDCF(string(s))
	if err != nil {
		return nil, err
	}

	var releaseString []string
	for _, algo := range defaultHashAlgorithms {
		str, ok := release[algo]
		if ok {
			releaseString = str
			break
		}
	}

	if releaseString == nil {
		return nil, fmt.Errorf("No hash field found.")
	}

	for _, line := range releaseString {
		remotePath, hash, err := parseIndex(line)
		if err != nil {
			return nil, err
		}

		release[remotePath] = hash
	}

	return release, nil
}

func (oldHashMap ReleaseHashMap) Update(newHashMap ReleaseHashMap) []string {
	updated := make([]string, 0)

	for remotePath, newHash := range newHashMap {
		oldHash, ok := oldHashMap[remotePath]
		if !ok {
			continue
		}
		if newHash != oldHash {
			updated = append(updated, remotePath)
			oldHashMap[remotePath] = newHash
		}
	}

	return updated
}
