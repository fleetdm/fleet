package util

import (
	"strings"
	"unicode/utf8"
)

// StripEmojis removes emoji and pictographic characters and certain joiners/variation selectors
// to leave readable text.
func StripEmojis(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r == 0xFE0F || r == 0x200D || r == 0x200C || r == 0x200B {
			continue
		}
		if (r >= 0x1F300 && r <= 0x1FAFF) || (r >= 0x2600 && r <= 0x27BF) {
			continue
		}
		b.WriteRune(r)
	}
	return strings.TrimSpace(b.String())
}

// PlainForSort returns a simplified string without emojis, lowercased, for consistent sorting
func PlainForSort(s string) string {
	return strings.ToLower(StripEmojis(s))
}

// TruncateTitle truncates a string to maxRunes runes and appends "..." if longer.
func TruncateTitle(s string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	count := 0
	for idx := range s {
		if count == maxRunes {
			return s[:idx] + "..."
		}
		count++
	}
	return s
}

// EscapeSingleQuotes escapes single quotes for safe inclusion inside single-quoted shell strings.
func EscapeSingleQuotes(s string) string {
	return strings.ReplaceAll(s, "'", "'\\''")
}

// SafeRuneLen returns runes count of a string.
func SafeRuneLen(s string) int { return utf8.RuneCountInString(s) }
