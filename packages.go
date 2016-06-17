package aptcacher

import (
	"io"
	"io/ioutil"
	"strings"
)

type PackagesHashMap map[string]string

func getHash(pkg map[string][]string) string {
	for _, algo := range defaultHashAlgorithms {
		if hash, ok := pkg[algo]; ok && len(hash) == 1 {
			return hash[0]
		}
	}
	return ""
}

func GetPackagesHashMap(r io.Reader) (PackagesHashMap, error) {
	packages, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	pkgHashMap := make(PackagesHashMap)
	chunks := strings.Split(string(packages), "\n\n")
	for _, chunk := range chunks {
		pkg, err := parseDCF(chunk)
		if err != nil || len(pkg) == 0 {
			break
		}

		hash := getHash(pkg)
		if remotePath, ok := pkg["Filename"]; ok && len(remotePath) == 1 {
			pkgHashMap[remotePath[0]] = hash
		}
	}
	return pkgHashMap, nil
}

func (oldPkgHashMap PackagesHashMap) Update(newPkgHashMap PackagesHashMap) []string {
	updated := make([]string, 0)

	for remotePath, newHash := range newPkgHashMap {
		oldHash, ok := oldPkgHashMap[remotePath]
		if !ok {
			continue
		}
		if newHash != oldHash {
			updated = append(updated, remotePath)
			oldPkgHashMap[remotePath] = newHash
		}
	}

	return updated
}
