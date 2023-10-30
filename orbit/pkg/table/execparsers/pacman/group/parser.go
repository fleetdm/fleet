package pacman_group

import (
	"bufio"
	"io"
	"strings"
)

func pacmanParse(reader io.Reader) (any, error) {
	results := make([]map[string]string, 0)

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		// We expect pacman to return lines in the following format:
		// `base-devel autoconf`
		// `gnome baobab`...
		// We split each line by space to get a group and package pair.
		// `<group> <package>`
		data := strings.SplitN(line, " ", 2)
		if len(data) != 2 {
			continue
		}

		row := make(map[string]string)
		row["group"] = strings.TrimSpace(data[0])
		row["package"] = strings.TrimSpace(data[1])

		results = append(results, row)
	}

	return results, nil
}
