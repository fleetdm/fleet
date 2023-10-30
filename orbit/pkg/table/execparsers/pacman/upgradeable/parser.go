package pacman_upgradeable

import (
	"bufio"
	"io"
	"regexp"
	"strings"
)

var lineRegexp = regexp.MustCompile(`(.*) (.*) -> (.*)`)

func pacmanParse(reader io.Reader) (any, error) {
	results := make([]map[string]string, 0)

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		// We expect pacman to return lines in the following format:
		// `abseil-cpp 20220623.0-1 -> 20220623.1-1`
		// `adwaita-icon-theme 42.0+r1+gc144c3d75-1 -> 43-2`
		// `alsa-ucm-conf 1.2.7.2-1 -> 1.2.8-1`...
		// We split each line by a regex pattern to get the package and versions.
		// `<package> <current_version> -> <upgrade_version>`
		data := lineRegexp.FindStringSubmatch(line)
		if len(data) != 4 {
			continue
		}

		row := make(map[string]string)
		row["package"] = strings.TrimSpace(data[1])
		row["current_version"] = strings.TrimSpace(data[2])
		row["upgrade_version"] = strings.TrimSpace(data[3])

		results = append(results, row)
	}

	return results, nil
}
