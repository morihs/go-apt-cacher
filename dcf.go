package aptcacher

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

func parseDCFField(line string) (string, string, error) {

	if b := line[0]; b == ' ' || b == '\t' {
		return "", strings.TrimRight(line[1:], " \t"), nil
	}

	split := strings.SplitN(line, ":", 2)
	if len(split) < 2 {
		return "", "", fmt.Errorf("Failed to parse this line: %s", line)
	}

	return strings.Trim(split[0], " \t"), strings.Trim(split[1], " \t"), nil
}

func parseDCF(r io.Reader) (*map[string]([]string), error) {
	release := make(map[string]([]string))
	scanner := bufio.NewScanner(r)

	var currentFieldName string
	for scanner.Scan() {
		line := scanner.Text()

		// abort reading if the line is blank
		if line == "" {
			break
		}

		name, value, err := parseDCFField(line)

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

	return &release, nil
}
