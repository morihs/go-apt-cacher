package aptcacher

import (
	"fmt"
	"strings"
)

type DCFMap map[string][]string

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

func parseDCF(dcf string) (DCFMap, error) {
	dcfMap := make(DCFMap)

	var currentFieldName string
	for _, line := range strings.Split(dcf, "\n") {
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
			dcfMap[currentFieldName] = append(dcfMap[currentFieldName], value)
		}
	}

	return dcfMap, nil
}
