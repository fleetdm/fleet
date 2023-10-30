package dsregcmd

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
)

// Section headers span multiple lines, they look like:
// +----------------------------------------------------------------------+
// | Device State                                                         |
// +----------------------------------------------------------------------+
//
// Because we're using a bufio.Scanner, we can't match the entire header in one shot.
// Instead we match the first line, and then the section title.
var startHeaderRegex = regexp.MustCompile(`^\s*\+\-+\+\s*$`)
var titleRegex = regexp.MustCompile(`^\s*\|\s*(.+?)\s*\|\s*$`)

// Capture output like:
//
//	IsDeviceJoined : NO
var lineRegex = regexp.MustCompile(`^\s*(.*?)\s*:\s*(.*?)\s*$`)

func parseDsreg(reader io.Reader) (any, error) {
	results := make(map[string]map[string]interface{})

	var currentSectionHeader string

	// Read the output line by line
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()

		// Check if we've found a section header. If so, grab the next line and get the section title.
		if startHeaderRegex.MatchString(line) {
			if ok := scanner.Scan(); !ok {
				return nil, fmt.Errorf("failed to read second section header line")
			}
			line = scanner.Text()
			m := titleRegex.FindStringSubmatch(line)
			if len(m) < 1 {
				return nil, fmt.Errorf("failed to parse section header, no second line: %s", line)
			}
			currentSectionHeader = m[1]
			results[currentSectionHeader] = make(map[string]interface{})

			// Consume the last line of the section header.
			if ok := scanner.Scan(); !ok {
				return nil, fmt.Errorf("failed to read third section header line")
			} else {
				line := scanner.Text()
				if !startHeaderRegex.MatchString(line) {
					return nil, fmt.Errorf("third section header line mismatch: %s", line)
				}
			}
			continue
		}

		// Check if we've found a line
		if m := lineRegex.FindStringSubmatch(line); len(m) > 0 {
			if currentSectionHeader == "" {
				return nil, fmt.Errorf("Found line before section header: %s", line)
			}

			// Add the key/value pair to the results
			results[currentSectionHeader][m[1]] = m[2]

			continue
		}
	}

	return results, nil
}
