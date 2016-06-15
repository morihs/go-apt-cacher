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
		if path, ok := pkg["Filename"]; ok && len(path) == 1 {
			pkgIndex[path[0]] = hash
		}
	}
	return pkgIndex, nil
}
