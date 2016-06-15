package aptcacher

import (
	"fmt"
	"io"
	"io/ioutil"
	"strings"
)

type PackageIndex map[string]string

func GetPackageIndex(r io.Reader) (PackageIndex, error) {
	lines, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	chunks := strings.Split(string(lines), "\n\n")

	for _, chunk := range chunks {
		pkg, e := parseDCF(chunk)
		if e != nil || len(pkg) == 0 {
			break
		}
		fmt.Println(pkg)
	}
	return nil, nil
}
