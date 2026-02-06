package util

import "strings"

// StatusMatches compares status to any of the provided names (case-insensitive, trimmed).
func StatusMatches(status string, names ...string) bool {
	ls := strings.ToLower(strings.TrimSpace(status))
	for _, n := range names {
		if ls == strings.ToLower(strings.TrimSpace(n)) {
			return true
		}
	}
	return false
}
