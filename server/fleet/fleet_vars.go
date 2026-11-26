package fleet

import (
	"iter"
	"strings"
)

const (
	ServerVarPrefix = "FLEET_VAR_"
)

// ContainsPrefixVars scans a string for variables in the form of $VAR
// and ${VAR} that begin with prefix, and return an array of those
// variables with the prefix removed. A bare $FLEET_SECRET_ is skipped, see
// IsBarePrefix.
func ContainsPrefixVars(text, prefix string) []string {
	vars := []string{}
	for name, end := range shellVars(text) {
		if strings.HasPrefix(name, prefix) && !IsBarePrefix(text, name, prefix, end) {
			vars = append(vars, strings.TrimPrefix(name, prefix))
		}
	}
	return vars
}

// IsBarePrefix reports whether name, ending at byte end of text, is only the
// $FLEET_SECRET_ prefix written as prose, such as "$FLEET_SECRET_ in CI". The
// braced form, or another variable, a brace, markup, an escape or a quote right
// after the prefix, keeps it a reference. Other prefixes are always references.
func IsBarePrefix(text, name, prefix string, end int) bool {
	if prefix != ServerSecretPrefix || name != prefix || text[end-1] == '}' {
		return false
	}
	return end >= len(text) || !strings.ContainsRune(`${<>&\]"'`, rune(text[end]))
}

// ContainsVar returns true if text contains the given variableName in the form of
// $VAR and ${VAR} (based on os.Expand). Returns false if variableName is empty.
func ContainsVar(text, variableName string) bool {
	if variableName == "" {
		return false
	}
	for name := range shellVars(text) {
		if name == variableName {
			return true
		}
	}
	return false
}

// shellVars yields each $VAR and ${VAR} name in text and the byte where it
// ends, scanning like os.Expand, which doesn't report positions.
func shellVars(text string) iter.Seq2[string, int] {
	return func(yield func(string, int) bool) {
		// ${} is all ASCII, so bytes are fine for this operation.
		for j := 0; j < len(text); j++ {
			if text[j] == '$' && j+1 < len(text) {
				name, w := getShellName(text[j+1:])
				if !yield(name, j+1+w) {
					return
				}
				j += w
			}
		}
	}
}

// MaybeExpand conditionally replaces ${var} or $var in the string based on the mapping function.
// Only replaces the variable with the mapper string if it returns true.
// The mapper returning false will leave the original variable unchanged.
// Based on os.Expand
func MaybeExpand(s string, mapping func(string, int, int) (string, bool)) string {
	var buf []byte
	// ${} is all ASCII, so bytes are fine for this operation.
	i := 0
	for j := 0; j < len(s); j++ {
		if s[j] == '$' && j+1 < len(s) {
			if buf == nil {
				buf = make([]byte, 0, len(s))
			}
			buf = append(buf, s[i:j]...)
			name, w := getShellName(s[j+1:])
			if name == "" {
				// Unlike the original Expand, don't
				// eat invalid syntax, just leave it
				// and pass over.
				w = 0
				buf = append(buf, s[j])
			} else {
				replacement, shouldReplace := mapping(name, j, j+w+1)
				if shouldReplace {
					buf = append(buf, replacement...)
				} else {
					// We aren't replacing this
					// reference, pass over it.
					w = 0
					buf = append(buf, s[j])
				}
			}
			j += w
			i = j + 1
		}
	}
	if buf == nil {
		return s
	}
	return string(buf) + s[i:]
}

// isShellSpecialVar reports whether the character identifies a special
// shell variable such as $*.
func isShellSpecialVar(c uint8) bool {
	switch c {
	case '*', '#', '$', '@', '!', '?', '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return true
	}
	return false
}

// isAlphaNum reports whether the byte is an ASCII letter, number, or underscore.
func isAlphaNum(c uint8) bool {
	return c == '_' || '0' <= c && c <= '9' || 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z'
}

// getShellName returns the name that begins the string and the number of bytes
// consumed to extract it. If the name is enclosed in {}, it's part of a ${}
// expansion and two more bytes are needed than the length of the name.
func getShellName(s string) (string, int) {
	switch {
	case s[0] == '{':
		if len(s) > 2 && isShellSpecialVar(s[1]) && s[2] == '}' {
			return s[1:2], 3
		}
		// Scan to closing brace
		for i := 1; i < len(s); i++ {
			if s[i] == '}' {
				if i == 1 {
					return "", 2 // Bad syntax; eat "${}"
				}
				return s[1:i], i + 1
			}
		}
		return "", 1 // Bad syntax; eat "${"
	case isShellSpecialVar(s[0]):
		return s[0:1], 1
	}
	// Scan alphanumerics.
	var i int
	// nolint:revive
	for i = 0; i < len(s) && isAlphaNum(s[i]); i++ {
	}
	return s[:i], i
}
