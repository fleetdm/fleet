package str

import (
	"strconv"
	"strings"
	"unicode/utf8"
)

// MaxErrorResponseBytes is the maximum number of bytes captured from a remote
// error response body before the string is truncated.
const MaxErrorResponseBytes = 512 * 1024

// TruncateErrorResponse caps s at MaxErrorResponseBytes bytes. When the string
// is longer it is cut at a valid UTF-8 boundary and " [truncated]" is appended.
func TruncateErrorResponse(s string) string {
	if len(s) <= MaxErrorResponseBytes {
		return s
	}
	cut := s[:MaxErrorResponseBytes]
	// Step back from the cut point to ensure we end on a valid rune boundary.
	for !utf8.ValidString(cut) {
		cut = cut[:len(cut)-1]
	}
	return cut + " [truncated]"
}

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
