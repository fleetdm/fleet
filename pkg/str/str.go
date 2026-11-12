package str

import (
	"strconv"
	"strings"
	"unicode/utf8"
)

// TruncateBytes returns s capped at maxBytes bytes, with marker appended when it had to cut. The returned string including the
// marker never exceeds maxBytes.
//
// The cut lands on a rune boundary, so a valid UTF-8 input stays valid instead of ending in half a character. It does not
// sanitize: an input that is already invalid UTF-8 stays invalid. If maxBytes cannot hold the marker, the marker is dropped
// rather than returning a string that is nothing but marker.
func TruncateBytes(s string, maxBytes int, marker string) string {
	if maxBytes <= 0 {
		return ""
	}
	if len(s) <= maxBytes {
		return s
	}
	cut := maxBytes - len(marker)
	if cut <= 0 {
		cut, marker = maxBytes, ""
	}
	// Step back to the start of a rune so the cut never splits a multi-byte character.
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut] + marker
}

// MaxErrorResponseBytes is the maximum number of bytes captured from a remote
// error response body before the string is truncated.
const MaxErrorResponseBytes = 512 * 1024

// TruncateErrorResponse caps s at MaxErrorResponseBytes bytes. When the string is longer it is cut at a valid UTF-8 boundary
// and " [truncated]" is appended.
func TruncateErrorResponse(s string) string {
	return TruncateBytes(s, MaxErrorResponseBytes, " [truncated]")
}

// TruncateRunes returns s shortened to at most maxRunes characters, preserving the start of the string. Use it before
// storing device- or user-supplied text in a column: utf8mb4 VARCHAR(N) in MySQL counts characters (runes), not bytes,
// so slicing on runes both matches the column constraint and cannot cut a multi-byte character in half and produce invalid UTF-8.
func TruncateRunes(s string, maxRunes int) string {
	if len(s) <= maxRunes {
		// Fast path: a string of at most maxRunes bytes cannot exceed maxRunes characters.
		return s
	}
	if utf8.RuneCountInString(s) <= maxRunes {
		return s
	}
	return string([]rune(s)[:maxRunes])
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
