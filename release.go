package aptcacher

import (
	"bufio"
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

func splitField(line string) (string, string, error) {

	if b := line[0]; b == ' ' || b == '\t' {
		return "", strings.TrimRight(line[1:], " \t"), nil
	}

	split := strings.SplitN(line, ":", 2)
	if len(split) < 2 {
		return "", "", fmt.Errorf("Failed to parse this line: %s", line)
	}

	return strings.Trim(split[0], " \t"), strings.Trim(split[1], " \t"), nil
}

func ParseRelease(r io.Reader) (map[string]([]string), error) {
	release := make(map[string]([]string))
	scanner := bufio.NewScanner(r)

	var currentFieldName string
	for scanner.Scan() {
		line := scanner.Text()
		name, value, err := splitField(line)

		if err != nil {
			return nil, err
		}

		if len(name) > 0 {
			currentFieldName = name
		} else if len(currentFieldName) == 0 {
			err = fmt.Errorf("No field name found in this line or the previous lines: %s", line)
			return nil, err
		}

		if len(value) > 0 {
			release[currentFieldName] = append(release[currentFieldName], value)
		}
	}

	return release, nil
}

func ParseIndex(line string) (string, string, error) {
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

func NewIndices(release *map[string]([]string)) (*Indices, error) {
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
		path, hash, err := ParseIndex(line)
		if err != nil {
			return nil, err
		}

		(*indices)[path] = hash
	}

	return indices, nil
}

func (indeces *Indices) Update(release *map[string]([]string)) {

}
