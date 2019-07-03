package mysql

import (
	"regexp"
	"strings"
)

var mysqlFTSSymbolRegexp = regexp.MustCompile("[-+]+")

func queryMinLength(query string) bool {
	return countLongestTerm(query) >= 3
}

func countLongestTerm(query string) int {
	max := 0
	for _, q := range strings.Split(query, " ") {
		if len(q) > max {
			max = len(q)
		}
	}
	return max
}

// transformQuery replaces occurrences of characters that are treated specially
// by the MySQL FTS engine to try to make the search more user-friendly
func transformQuery(query string) string {
	return strings.TrimSpace(
		mysqlFTSSymbolRegexp.ReplaceAllLiteralString(query, " "),
	) + "*"
}
