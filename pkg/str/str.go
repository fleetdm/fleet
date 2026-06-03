package str

import (
	"strconv"
	"strings"
)

func SplitAndTrim(s string, delimiter string, removeEmpty bool) []string {
	parts := strings.Split(s, delimiter)
	cleaned := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if !removeEmpty || part != "" {
			cleaned = append(cleaned, part)
		}
	}
	return cleaned
}

// ParseUintList parses a comma-separated string of unsigned integers, trimming
// whitespace around each value and silently skipping values that cannot be
// parsed. Returns nil for an empty input.
func ParseUintList(s string) []uint {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]uint, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if v, err := strconv.ParseUint(p, 10, 0); err == nil {
			result = append(result, uint(v))
		}
	}
	return result
}

// ParseStringList parses a comma-separated string into a slice of strings,
// trimming whitespace and dropping empty values. Returns nil for an empty
// input.
func ParseStringList(s string) []string {
	if s == "" {
		return nil
	}
	return SplitAndTrim(s, ",", true)
}
