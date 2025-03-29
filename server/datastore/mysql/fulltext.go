package mysql

import (
	"regexp"
	"strings"
)

var mysqlFTSSymbolRegexp = regexp.MustCompile("[-+@]+")

// queryMinLength returns true if the query argument is longer than a "short" word.
// What defines a "short" word is MySQL's "ft_min_word_len" VARIABLE, generally set
// to 4 by default in Fleet deployments.
//
// TODO(lucas): Remove this method on #2627.
func queryMinLength(query string) bool {
	return countLongestTerm(query) >= 3
}

func countLongestTerm(query string) int {
	maxSize := 0
	for _, q := range strings.Split(query, " ") {
		if len(q) > maxSize {
			maxSize = len(q)
		}
	}
	return maxSize
}

// transformQuery replaces occurrences of characters that are treated specially
// by the MySQL FTS engine to try to make the search more user-friendly
func transformQuery(query string) string {
	return transformQueryWithSuffix(query, "*")
}

func transformQueryWithSuffix(query, suffix string) string {
	return strings.TrimSpace(
		mysqlFTSSymbolRegexp.ReplaceAllLiteralString(query, " "),
	) + suffix
}
