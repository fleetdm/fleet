package netskope

import "strings"

// parseNsdiagText parses the `Key:: Value.` lines that `nsdiag -f` prints into a
// column-keyed map. Lines without a `::` separator, and keys not in
// nsdiagKeyToColumn, are skipped.
func parseNsdiagText(b []byte) map[string]string {
	result := make(map[string]string)
	for line := range strings.SplitSeq(string(b), "\n") {
		key, val, found := strings.Cut(line, "::")
		if !found {
			continue
		}
		col, ok := nsdiagKeyToColumn[strings.TrimSpace(strings.ToLower(key))]
		if !ok {
			continue
		}
		// nsdiag terminates every value with a period.
		val = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(val), "."))
		result[col] = normalizeNsdiagValue(val)
	}
	return result
}

// normalizeNsdiagValue uppercases the boolean values nsdiag reports with
// inconsistent casing (`FALSE` in some fields, `false` in others) so a single
// comparison works across fields. Only the literal words are matched: numeric
// values such as a gateway IP or a `0`/`1` field are passed through unchanged.
func normalizeNsdiagValue(val string) string {
	switch {
	case strings.EqualFold(val, "true"):
		return "TRUE"
	case strings.EqualFold(val, "false"):
		return "FALSE"
	}
	return val
}
