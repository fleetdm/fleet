package pacman_info

import (
	"bufio"
	"io"
	"strings"

	"golang.org/x/exp/slices"
)

var allowedKeys = []string{
	"name",
	"version",
	"description",
	"groups",
	"build date",
	"install date",
	"install reason",
}

func pacmanParse(reader io.Reader) (any, error) {
	results := make([]map[string]string, 0)
	row := make(map[string]string)

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		// pacman gives an empty line between package info. I'm using this to
		// break out of processing the current package, and start the next.
		if len(line) == 0 && len(row) > 0 {
			results = append(results, row)
			row = make(map[string]string)
			continue
		}

		// We expect pacman to return lines in the following format:
		//`Name            : aalib`
		//`Version         : 1.4rc5-14`
		//`Description     : A portable ASCII art graphic library`...
		// We split each line to get a key -> value pair, then store
		// it into our row until we log the row and start a new one.
		kv := strings.SplitN(line, ":", 2)
		if len(kv) < 2 {
			continue
		}

		var key = strings.ToLower(strings.TrimSpace(kv[0]))
		if slices.Contains(allowedKeys, key) {
			row[key] = strings.TrimSpace(kv[1])
		}
	}

	return results, nil
}
