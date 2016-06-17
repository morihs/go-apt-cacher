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
	release, err := parseDCF(string(s))
	if err != nil {
		return nil, err
	}

	var relHash []string
	for _, algo := range defaultHashAlgorithms {
		if str, ok := release[algo]; ok {
			relHash = str
			break
		}
	}

	if relHash == nil {
		return nil, fmt.Errorf("No hash field found.")
	}

	relHashMap := make(ReleaseHashMap)
	for _, line := range relHash {
		remotePath, hash, err := parseIndex(line)
		if err != nil {
			return nil, err
		}

		relHashMap[remotePath] = hash
	}

	return relHashMap, nil
}

func (oldRelHashMap ReleaseHashMap) Update(newRelHashMap ReleaseHashMap) []string {
	updated := make([]string, 0)

	for remotePath, newHash := range newRelHashMap {
		oldHash, ok := oldRelHashMap[remotePath]
		if !ok {
			continue
		}
		if newHash != oldHash {
			updated = append(updated, remotePath)
			oldRelHashMap[remotePath] = newHash
		}
	}

	return updated
}
