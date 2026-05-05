package util

import "regexp"

// CveIDPattern is regexp matches to `CVE-\d{4}-\d{4,}`
var CveIDPattern = regexp.MustCompile(`CVE-\d{4}-\d{4,}`)

// UniqueStrings eliminates duplication from []string
func UniqueStrings(s []string) []string {
	if len(s) == 0 {
		return nil
	}
	m := make(map[string]struct{}, len(s))
	for _, v := range s {
		m[v] = struct{}{}
	}
	uniq := make([]string, 0, len(m))
	for v := range m {
		uniq = append(uniq, v)
	}
	return uniq
}
