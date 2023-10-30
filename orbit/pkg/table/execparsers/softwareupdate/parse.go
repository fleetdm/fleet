//go:build darwin
// +build darwin

package softwareupdate

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

func (p *parser) parseSoftwareupdate(reader io.Reader) (any, error) {
	results := make([]map[string]string, 0)

	p.scanner = bufio.NewScanner(reader)
	for p.scanner.Scan() {
		currentLine := strings.TrimSpace(p.scanner.Text())

		if strings.Contains(currentLine, "No new software available") {
			// This is our indication that the device is up-to-date: there should be no recommended
			// updates. Return this data early.
			results = append(results, map[string]string{
				"UpToDate": "true",
			})
			break
		}

		// There are some header lines (e.g. `Software Update Tool`) that we can safely discard.
		// We only care about pairs of lines, where the first line begins in the following way.
		if !strings.HasPrefix(currentLine, "* Label:") {
			continue
		}

		// Software updates are listed in the following format:
		// * Label: <title>
		//     Title: <title>, Version: <version>, Size: <size>, Recommended: YES|?, Action: <action>,
		label := strings.TrimSpace(strings.TrimPrefix(currentLine, "* Label:"))
		labelAttributes, err := p.parseUpdate(label)
		if err != nil {
			return results, fmt.Errorf("could not parse software update data for label %s: %w", label, err)
		}

		results = append(results, labelAttributes)
	}

	return results, nil
}

func (p *parser) parseUpdate(label string) (map[string]string, error) {
	result := make(map[string]string)
	result["Label"] = label

	// Get the next line
	if !p.scanner.Scan() {
		return result, fmt.Errorf("software update data missing for label %s", label)
	}
	updateDataStr := strings.TrimSuffix(strings.TrimSpace(p.scanner.Text()), ",")

	// Add each update attribute to the result
	updateData := strings.Split(updateDataStr, ",")
	for _, attr := range updateData {
		keyValPair := strings.SplitN(attr, ":", 2)
		if len(keyValPair) < 2 {
			return result, fmt.Errorf("software update data has malformed attribute %s", attr)
		}
		result[strings.TrimSpace(keyValPair[0])] = strings.TrimSpace(keyValPair[1])
	}

	return result, nil
}
