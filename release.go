package aptcacher

import (
	"fmt"
	"io"
	"strings"
)

var defaultHashAlgorithms = []string{
	"MD5Sum",
	"SHA1",
	"SHA256",
	"SHA512",
}

type Indices map[string]string

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

func GetIndices(r io.Reader) (*Indices, error) {
	release, err := parseDCF(r)
	if err != nil {
		return nil, err
	}

	indices := &Indices{}

	var indicesString []string
	for _, algo := range defaultHashAlgorithms {
		str, ok := (*release)[algo]
		if ok {
			indicesString = str
			break
		}
	}

	if indicesString == nil {
		return nil, fmt.Errorf("No hash field found.")
	}

	for _, line := range indicesString {
		path, hash, err := parseIndex(line)
		if err != nil {
			return nil, err
		}

		(*indices)[path] = hash
	}

	return indices, nil
}

func (oldIndices *Indices) Update(newIndices *Indices) *[]string {
	updated := make([]string, 0)

	for path, newHash := range *newIndices {
		oldHash, ok := (*oldIndices)[path]
		if !ok {
			continue
		}
		if newHash != oldHash {
			updated = append(updated, path)
			(*oldIndices)[path] = newHash
		}
	}

	return &updated
}
