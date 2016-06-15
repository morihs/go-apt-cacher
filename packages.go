package aptcacher

import (
	"io"
	"io/ioutil"
	"strings"
)

type PackageIndex map[string]string

func getHash(pkg map[string][]string) string {
	for _, algo := range defaultHashAlgorithms {
		if hash, ok := pkg[algo]; ok && len(hash) == 1 {
			return hash[0]
		}
	}
	return ""
}

func GetPackageIndex(r io.Reader) (PackageIndex, error) {
	lines, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	pkgIndex := make(PackageIndex)
	chunks := strings.Split(string(lines), "\n\n")
	for _, chunk := range chunks {
		pkg, err := parseDCF(chunk)
		if err != nil || len(pkg) == 0 {
			break
		}

		hash := getHash(pkg)
		if remotePath, ok := pkg["Filename"]; ok && len(remotePath) == 1 {
			pkgIndex[remotePath[0]] = hash
		}
	}
	return pkgIndex, nil
}

func (oldPkgIndex PackageIndex) Update(newPkgIndex PackageIndex) []string {
	updated := make([]string, 0)

	for remotePath, newHash := range newPkgIndex {
		oldHash, ok := oldPkgIndex[remotePath]
		if !ok {
			continue
		}
		if newHash != oldHash {
			updated = append(updated, remotePath)
			oldPkgIndex[remotePath] = newHash
		}
	}

	return updated
}
