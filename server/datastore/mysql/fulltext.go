package mysql

import (
	"regexp"
	"strings"
)

var mysqlFTSSymbolRegexp = regexp.MustCompile("[-+]+")

// queryMinLength returns true if the query argument is longer than a "short" word.
// What defines a "short" word is MySQL's "ft_min_word_len" VARIABLE, generally set
// to 4 by default in Fleet deployments.
func queryMinLength(query string) bool {
	// TODO(lucas): Change to 4 (on a separate ticket/PR).
	// There's currently no bug because we always append the truncation operation "*".
	// From Oracle docs: "If a word is specified with the truncation operator, it is not
	// stripped from a boolean query, even if it is too short or a stopword."
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
	return transformQueryWithSuffix(query, "*")
}

func transformQueryWithSuffix(query, suffix string) string {
	return strings.TrimSpace(
		mysqlFTSSymbolRegexp.ReplaceAllLiteralString(query, " "),
	) + suffix
}
