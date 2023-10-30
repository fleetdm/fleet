package apt

import (
	"bufio"
	"io"
	"strings"
)

func aptParse(reader io.Reader) (any, error) {
	results := make([]map[string]string, 0)

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		// We expect apt to return lines in the following format:
		// `base-files/jammy-updates 12ubuntu4.3 amd64 [upgradable from: 12ubuntu4.2]`
		// We split on the forward slash, then on spaces to get the following output:
		// `<package name>/<source> <update version> <arch> [upgradable from: <current version>]`
		pair := strings.Split(line, "/")
		if len(pair) < 2 {
			continue
		}

		packageName := strings.ToLower(strings.TrimSpace(pair[0]))
		if len(packageName) < 1 {
			continue
		}

		values := strings.Split(pair[1], " ")
		if len(values) < 6 {
			continue
		}

		row := make(map[string]string)
		row["package"] = packageName
		row["sources"] = strings.TrimSpace(values[0])
		row["update_version"] = strings.TrimSpace(values[1])
		row["current_version"] = strings.TrimRight(values[5], "]")

		results = append(results, row)
	}

	return results, nil
}
