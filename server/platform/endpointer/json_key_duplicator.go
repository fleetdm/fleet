package endpointer

import (
	"bytes"
	"io"

	"github.com/go-json-experiment/json/jsontext"
)

// DuplicateJSONKeysOpts controls optional behavior of DuplicateJSONKeys.
type DuplicateJSONKeysOpts struct {
	// Compact disables pretty-printing. By default, output is indented with
	// two spaces to match the standard API response format.
	Compact bool
}

// DuplicateJSONKeys takes marshaled JSON and, for each AliasRule, duplicates
// keys so that both the old (native) and new names appear in the output.
// For example, if a rule maps OldKey:"team_id" → NewKey:"fleet_id", and the
// JSON contains "team_id": 42, the output will contain both "team_id": 42
// and "fleet_id": 42.
//
// If the new key already exists in the same object scope, the duplication is
// skipped for that key (to avoid producing duplicate keys when the source
// struct already has both, or when the function is called more than once).
//
// The function uses jsontext.Decoder/Encoder for token-level processing,
// delegating all JSON lexing (string escaping, unicode, nesting) to the
// library. Duplicates are deferred until the closing '}' of each object so
// that naturally-occurring new keys can be detected and skipped.
func DuplicateJSONKeys(data []byte, rules []AliasRule, opts ...DuplicateJSONKeysOpts) []byte {
	if len(rules) == 0 || len(data) == 0 {
		return data
	}

	oldToNew := make(map[string]string, len(rules))
	for _, r := range rules {
		oldToNew[r.OldKey] = r.NewKey
	}

	var buf bytes.Buffer
	dec := jsontext.NewDecoder(bytes.NewReader(data), jsontext.AllowDuplicateNames(true))
	encOpts := []jsontext.Options{jsontext.AllowDuplicateNames(true)}
	if len(opts) == 0 || !opts[0].Compact {
		encOpts = append(encOpts, jsontext.WithIndent("  "))
	}
	enc := jsontext.NewEncoder(&buf, encOpts...)

	// pendingDup holds a key-value pair that should be inserted as a
	// duplicate at the end of the current object scope (before '}'),
	// unless the new key was found naturally in the same scope.
	type pendingDup struct {
		newKey string
		value  jsontext.Value
	}

	// Per-object-scope state: pending duplicates and naturally-seen new keys.
	type scopeState struct {
		pending    []pendingDup
		naturalNew map[string]bool
	}
	var scopes []scopeState

	for {
		tok, err := dec.ReadToken()
		if err != nil {
			if err == io.EOF {
				break
			}
			// On any error, return the original data unchanged.
			return data
		}

		kind := tok.Kind()

		switch kind {
		case '{':
			scopes = append(scopes, scopeState{naturalNew: make(map[string]bool)})
			if err := enc.WriteToken(tok); err != nil {
				return data
			}

		case '}':
			// Before closing the object, emit any pending duplicates whose
			// new key was not seen naturally in this scope.
			if len(scopes) > 0 {
				scope := scopes[len(scopes)-1]
				for _, dup := range scope.pending {
					if scope.naturalNew[dup.newKey] {
						continue // new key exists naturally; skip duplicate
					}
					if err := enc.WriteToken(jsontext.String(dup.newKey)); err != nil {
						return data
					}
					if err := enc.WriteValue(dup.value); err != nil {
						return data
					}
				}
				scopes = scopes[:len(scopes)-1]
			}
			if err := enc.WriteToken(tok); err != nil {
				return data
			}

		case '"':
			// Determine if this string is an object key.
			isKey := false
			depth := dec.StackDepth()
			if depth > 0 {
				parentKind, length := dec.StackIndex(depth)
				if parentKind == '{' && length%2 == 1 {
					isKey = true
				}
			}

			if isKey {
				keyName := tok.String()

				// Track new keys that appear naturally.
				if len(scopes) > 0 {
					for _, newKey := range oldToNew {
						if keyName == newKey {
							scopes[len(scopes)-1].naturalNew[newKey] = true
						}
					}
				}

				// Check if this old key should generate a duplicate.
				newKey, shouldDuplicate := oldToNew[keyName]
				if shouldDuplicate {
					// Write the old key.
					if err := enc.WriteToken(tok); err != nil {
						return data
					}

					// Read the value, capturing it for deferred duplication.
					val, err := dec.ReadValue()
					if err != nil {
						return data
					}

					// Recursively duplicate keys within the value.
					processedVal := DuplicateJSONKeys([]byte(val), rules, opts...)

					// Write the value for the old key.
					if err := enc.WriteValue(jsontext.Value(processedVal)); err != nil {
						return data
					}

					// Defer the duplicate for emission at '}'.
					if len(scopes) > 0 {
						scopes[len(scopes)-1].pending = append(
							scopes[len(scopes)-1].pending,
							pendingDup{newKey: newKey, value: jsontext.Value(processedVal)},
						)
					}
				} else {
					if err := enc.WriteToken(tok); err != nil {
						return data
					}
				}
			} else {
				// String value — pass through.
				if err := enc.WriteToken(tok); err != nil {
					return data
				}
			}

		default:
			// All other tokens: [, ], numbers, bools, null — pass through.
			if err := enc.WriteToken(tok); err != nil {
				return data
			}
		}
	}

	return buf.Bytes()
}
