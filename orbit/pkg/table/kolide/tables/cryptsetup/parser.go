package cryptsetup

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// parseStatus parses the output from `cryptsetup status`. This is a
// pretty simple key, value format, but does have a free form first
// line. It's not clear if this is going to be stable, or change
// across versions.
func parseStatus(rawdata []byte) (map[string]interface{}, error) {
	var data map[string]interface{}

	if len(rawdata) == 0 {
		return nil, errors.New("No data")
	}

	scanner := bufio.NewScanner(bytes.NewReader(rawdata))
	firstLine := true
	for scanner.Scan() {
		line := scanner.Text()
		if firstLine {
			var err error
			data, err = parseFirstLine(line)
			if err != nil {
				return nil, err
			}

			firstLine = false
			continue
		}

		kv := strings.SplitN(line, ": ", 2)

		// blank lines, or other unexpected input can just be skipped.
		if len(kv) < 2 {
			continue
		}

		data[strings.ReplaceAll(strings.TrimSpace(kv[0]), " ", "_")] = strings.TrimSpace(kv[1])
	}

	return data, nil
}

// regexp for the first line of the status output.
var firstLineRegexp = regexp.MustCompile(`^(?:Device (.*) (not found))|(?:(.*?) is ([a-z]+)(?:\.| and is (in use)))`)

// parseFirstLine parses the first line of the status output. This
// appears to be a free form string indicating several pieces of
// information. It is parsed with a single regexp. (See tests for
// examples)
func parseFirstLine(line string) (map[string]interface{}, error) {
	if line == "" {
		return nil, fmt.Errorf("Invalid first line")
	}

	m := firstLineRegexp.FindAllStringSubmatch(line, -1)
	if len(m) != 1 {
		return nil, fmt.Errorf("Failed to match first line: %s", line)
	}
	if len(m[0]) != 6 {
		return nil, fmt.Errorf("Got %d matches. Expected 6. Failed to match first line: %s", len(m[0]), line)
	}

	data := make(map[string]interface{}, 3)

	// check for $1 and $2 for the error condition
	if m[0][1] != "" && m[0][2] != "" {
		data["short_name"] = m[0][1]
		data["status"] = strings.ReplaceAll(m[0][2], " ", "_")
		data["mounted"] = strconv.FormatBool(false)
		return data, nil
	}

	if m[0][3] != "" && m[0][4] != "" {
		data["display_name"] = m[0][3]
		data["status"] = strings.ReplaceAll(m[0][4], " ", "_")
		data["mounted"] = strconv.FormatBool(m[0][5] != "")
		return data, nil
	}

	return nil, fmt.Errorf("Unknown first line: %s", line)
}
