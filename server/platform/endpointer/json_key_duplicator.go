package endpointer

import (
	"bytes"
)

// DuplicateJSONKeys takes marshaled JSON and, for each AliasRule, duplicates
// keys so that both the new and old (deprecated) names appear in the output.
// For example, if a rule maps OldKey:"team_id" → NewKey:"fleet_id", and the
// JSON contains "fleet_id": 42, the output will contain both "fleet_id": 42
// and "team_id": 42.
//
// If the old key already exists in the same object scope, the duplication is
// skipped for that key (to avoid producing duplicate keys when the source
// struct already has both).
//
// The function operates on a byte-level state machine similar to
// JSONKeyRewriteReader — it tracks strings, escapes, nesting depth, and
// key vs. value positions to correctly identify JSON object keys without
// fully parsing the document.
func DuplicateJSONKeys(data []byte, rules []AliasRule) []byte {
	if len(rules) == 0 || len(data) == 0 {
		return data
	}

	// Build lookup: newKey -> oldKey
	newToOld := make(map[string]string, len(rules))
	for _, r := range rules {
		newToOld[r.NewKey] = r.OldKey
	}

	// First pass: scan to find which keys exist in each object scope.
	// We need this to know when to skip duplication.
	existingKeys := scanExistingKeys(data, rules)

	// Second pass: produce output with duplicated keys.
	return duplicatePass(data, newToOld, existingKeys)
}

// scopeID identifies an object scope by its nesting depth and the byte offset
// where the opening '{' was found.
type scopeID struct {
	depth  int
	offset int
}

// scanExistingKeys scans the JSON data and returns a set of (scopeID, keyName)
// pairs representing all keys that already exist in each object scope.
// Only keys relevant to the rules are tracked.
func scanExistingKeys(data []byte, rules []AliasRule) map[scopeID]map[string]bool {
	// Build set of interesting keys (both old and new names).
	interesting := make(map[string]bool, len(rules)*2)
	for _, r := range rules {
		interesting[r.OldKey] = true
		interesting[r.NewKey] = true
	}

	result := make(map[scopeID]map[string]bool)

	// State machine
	depth := 0
	// Stack of scope IDs (object scopes only). Array scopes don't get pushed.
	var scopeStack []scopeID
	// Track whether we're expecting a key (after '{' or ',' at object level).
	// We use a stack to track this per nesting level.
	type scopeState struct {
		isObject   bool
		expectKey  bool
		afterColon bool
	}
	var stateStack []scopeState

	i := 0
	for i < len(data) {
		b := data[i]

		switch b {
		case '"':
			// Extract the string value (extractString consumes the whole string).
			str, end := extractString(data, i)
			if end < 0 {
				// Malformed JSON, bail out.
				return result
			}

			// It's a key if we're in an object scope and expecting a key.
			if len(stateStack) > 0 {
				top := &stateStack[len(stateStack)-1]
				if top.isObject && top.expectKey && !top.afterColon {
					// This is a key.
					if interesting[str] && len(scopeStack) > 0 {
						sid := scopeStack[len(scopeStack)-1]
						if result[sid] == nil {
							result[sid] = make(map[string]bool)
						}
						result[sid][str] = true
					}
					top.expectKey = false
					top.afterColon = false
				}
			}
			i = end + 1
			continue

		case ':':
			if len(stateStack) > 0 {
				stateStack[len(stateStack)-1].afterColon = true
			}

		case ',':
			if len(stateStack) > 0 {
				top := &stateStack[len(stateStack)-1]
				if top.isObject {
					top.expectKey = true
					top.afterColon = false
				}
			}

		case '{':
			depth++
			sid := scopeID{depth: depth, offset: i}
			scopeStack = append(scopeStack, sid)
			stateStack = append(stateStack, scopeState{isObject: true, expectKey: true})

		case '}':
			if len(scopeStack) > 0 {
				scopeStack = scopeStack[:len(scopeStack)-1]
			}
			if len(stateStack) > 0 {
				stateStack = stateStack[:len(stateStack)-1]
			}
			// After closing a value, the next thing in parent scope is a comma or closing bracket.
			if len(stateStack) > 0 {
				stateStack[len(stateStack)-1].afterColon = false
			}
			depth--

		case '[':
			depth++
			stateStack = append(stateStack, scopeState{isObject: false})
			// Arrays don't produce object scopes, but we push a dummy scope
			// so the stack stays aligned.
			scopeStack = append(scopeStack, scopeID{depth: -1, offset: i})

		case ']':
			if len(scopeStack) > 0 {
				scopeStack = scopeStack[:len(scopeStack)-1]
			}
			if len(stateStack) > 0 {
				stateStack = stateStack[:len(stateStack)-1]
			}
			if len(stateStack) > 0 {
				stateStack[len(stateStack)-1].afterColon = false
			}
			depth--
		}

		i++
	}

	return result
}

// duplicatePass does the second pass: copies data to output, and after each
// key-value pair where the key is a NewKey (per rules), inserts a duplicate
// with the OldKey name.
func duplicatePass(data []byte, newToOld map[string]string, existingKeys map[scopeID]map[string]bool) []byte {
	out := bytes.NewBuffer(make([]byte, 0, len(data)+len(data)/4))

	depth := 0
	var scopeStack []scopeID

	type scopeState struct {
		isObject   bool
		expectKey  bool
		afterColon bool
	}
	var stateStack []scopeState

	i := 0
	for i < len(data) {
		b := data[i]

		switch b {
		case '"':
			str, end := extractString(data, i)
			if end < 0 {
				// Malformed; copy rest and return.
				out.Write(data[i:])
				return out.Bytes()
			}

			isKey := false
			if len(stateStack) > 0 {
				top := &stateStack[len(stateStack)-1]
				if top.isObject && top.expectKey && !top.afterColon {
					isKey = true
					top.expectKey = false
				}
			}

			if isKey {
				// Write the key (including quotes) verbatim.
				out.Write(data[i : end+1])
				i = end + 1

				// Check if this key should trigger duplication.
				oldKey, shouldDuplicate := newToOld[str]
				if shouldDuplicate && len(scopeStack) > 0 {
					sid := scopeStack[len(scopeStack)-1]
					// Skip if old key already exists in this scope.
					if keys, ok := existingKeys[sid]; ok && keys[oldKey] {
						shouldDuplicate = false
					}
				}

				if shouldDuplicate {
					// We need to copy the colon + value, then insert the duplicate.
					// Find the colon, copy it.
					colonIdx := skipWhitespace(data, i)
					if colonIdx < len(data) && data[colonIdx] == ':' {
						out.Write(data[i : colonIdx+1])
						i = colonIdx + 1
					}

					// Copy the value.
					valStart := skipWhitespace(data, i)
					valEnd := findValueEnd(data, valStart)
					if valEnd < 0 {
						out.Write(data[i:])
						return out.Bytes()
					}
					out.Write(data[i:valEnd])
					// Extract just the value portion (with leading whitespace preserved).
					valuePortion := data[i:valEnd]
					i = valEnd

					// Now insert the duplicate: comma + old key + colon + value.
					out.WriteByte(',')
					out.WriteByte('"')
					out.WriteString(oldKey)
					out.WriteByte('"')
					out.WriteByte(':')
					out.Write(valuePortion)

					// Update state: after emitting the value, we mark afterColon=false.
					if len(stateStack) > 0 {
						stateStack[len(stateStack)-1].afterColon = false
					}
				}
				continue
			}

			// Not a key — just a string value; copy verbatim.
			out.Write(data[i : end+1])
			i = end + 1
			continue

		case ':':
			if len(stateStack) > 0 {
				stateStack[len(stateStack)-1].afterColon = true
			}
			out.WriteByte(b)

		case ',':
			if len(stateStack) > 0 {
				top := &stateStack[len(stateStack)-1]
				if top.isObject {
					top.expectKey = true
					top.afterColon = false
				}
			}
			out.WriteByte(b)

		case '{':
			depth++
			sid := scopeID{depth: depth, offset: i}
			scopeStack = append(scopeStack, sid)
			stateStack = append(stateStack, scopeState{isObject: true, expectKey: true})
			out.WriteByte(b)

		case '}':
			if len(scopeStack) > 0 {
				scopeStack = scopeStack[:len(scopeStack)-1]
			}
			if len(stateStack) > 0 {
				stateStack = stateStack[:len(stateStack)-1]
			}
			if len(stateStack) > 0 {
				stateStack[len(stateStack)-1].afterColon = false
			}
			depth--
			out.WriteByte(b)

		case '[':
			depth++
			stateStack = append(stateStack, scopeState{isObject: false})
			scopeStack = append(scopeStack, scopeID{depth: -1, offset: i})
			out.WriteByte(b)

		case ']':
			if len(scopeStack) > 0 {
				scopeStack = scopeStack[:len(scopeStack)-1]
			}
			if len(stateStack) > 0 {
				stateStack = stateStack[:len(stateStack)-1]
			}
			if len(stateStack) > 0 {
				stateStack[len(stateStack)-1].afterColon = false
			}
			depth--
			out.WriteByte(b)

		default:
			out.WriteByte(b)
		}

		i++
	}

	return out.Bytes()
}

// extractString extracts the unescaped content of a JSON string starting at
// position i (which should point to the opening '"'). It returns the string
// content and the index of the closing '"'. Returns ("", -1) if malformed.
func extractString(data []byte, i int) (string, int) {
	if i >= len(data) || data[i] != '"' {
		return "", -1
	}

	j := i + 1
	var buf bytes.Buffer
	for j < len(data) {
		if data[j] == '\\' {
			if j+1 < len(data) {
				// For our purposes we just need the raw key name.
				// We handle the common escapes; unicode escapes are
				// passed through as-is since they're unlikely in key names.
				switch data[j+1] {
				case '"', '\\', '/':
					buf.WriteByte(data[j+1])
				case 'n':
					buf.WriteByte('\n')
				case 't':
					buf.WriteByte('\t')
				case 'r':
					buf.WriteByte('\r')
				case 'b':
					buf.WriteByte('\b')
				case 'f':
					buf.WriteByte('\f')
				default:
					// e.g. \uXXXX — just copy raw bytes
					buf.WriteByte(data[j])
					buf.WriteByte(data[j+1])
				}
				j += 2
				continue
			}
			return "", -1
		}
		if data[j] == '"' {
			return buf.String(), j
		}
		buf.WriteByte(data[j])
		j++
	}
	return "", -1
}

// skipWhitespace returns the index of the first non-whitespace byte at or
// after position i.
func skipWhitespace(data []byte, i int) int {
	for i < len(data) && (data[i] == ' ' || data[i] == '\t' || data[i] == '\n' || data[i] == '\r') {
		i++
	}
	return i
}

// findValueEnd finds the end of a JSON value starting at position i.
// The value may be a string, number, boolean, null, object, or array.
// It returns the index one past the last byte of the value.
// Returns -1 on malformed input.
func findValueEnd(data []byte, i int) int {
	if i >= len(data) {
		return -1
	}

	switch data[i] {
	case '"':
		// String value
		_, end := extractString(data, i)
		if end < 0 {
			return -1
		}
		return end + 1

	case '{':
		return findMatchingBrace(data, i, '{', '}')
	case '[':
		return findMatchingBrace(data, i, '[', ']')

	case 't': // true
		if i+4 <= len(data) && string(data[i:i+4]) == "true" {
			return i + 4
		}
		return -1
	case 'f': // false
		if i+5 <= len(data) && string(data[i:i+5]) == "false" {
			return i + 5
		}
		return -1
	case 'n': // null
		if i+4 <= len(data) && string(data[i:i+4]) == "null" {
			return i + 4
		}
		return -1

	default:
		// Number: digits, '-', '.', 'e', 'E', '+'
		j := i
		for j < len(data) {
			c := data[j]
			if (c >= '0' && c <= '9') || c == '-' || c == '+' || c == '.' || c == 'e' || c == 'E' {
				j++
			} else {
				break
			}
		}
		if j == i {
			return -1
		}
		return j
	}
}

// findMatchingBrace finds the index one past the matching closing brace/bracket
// for the opening one at position i. It handles nested braces and strings.
func findMatchingBrace(data []byte, i int, open, close byte) int {
	if i >= len(data) || data[i] != open {
		return -1
	}

	depth := 0
	inStr := false
	esc := false
	for j := i; j < len(data); j++ {
		if inStr {
			if esc {
				esc = false
				continue
			}
			if data[j] == '\\' {
				esc = true
				continue
			}
			if data[j] == '"' {
				inStr = false
			}
			continue
		}

		switch data[j] {
		case '"':
			inStr = true
		case open:
			depth++
		case close:
			depth--
			if depth == 0 {
				return j + 1
			}
		}
	}
	return -1
}
