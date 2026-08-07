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
// By default a renamed key produces a clean split: the old-named key keeps an
// all-old subtree and the new-named key gets an all-new subtree (via
// RewriteOldToNewKeys), so each subtree is internally single-named. A renamed
// leaf at the top level (or under a non-renamed key) is instead duplicated in
// place, so both names appear as siblings with the same value.
//
// A rule with Inline set opts into "merged" duplication for that container: its
// old-named subtree additionally carries the new-named copies of any nested
// renamed containers, so both names appear together on the same object (e.g.
// "abm_tokens" holding both "macos_team" and "macos_fleet"). Leaf renames
// inside an inlined subtree are still kept single-named per container (so
// "macos_team" holds "team_id" while its sibling "macos_fleet" holds
// "fleet_id") rather than cross-contaminating both id names into one object.
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
	compact := len(opts) > 0 && opts[0].Compact
	return duplicateJSONKeys(data, rules, compact)
}

// duplicateJSONKeys is the recursive core of DuplicateJSONKeys.
//
// An Inline container is the only recursive case: its old-named subtree is
// re-run through this function so nested renames surface there too — exactly as
// they did before the container itself was renamed. That recursion needs no
// special mode because the default rules already produce the right shape:
// nested renamed *containers* split cleanly into old/new siblings (their values
// are consumed whole by ReadValue, so their leaves are never duplicated in
// place), while nested renamed *leaves* are duplicated in place. The new-named
// subtree is always a clean RewriteOldToNewKeys copy.
func duplicateJSONKeys(data []byte, rules []AliasRule, compact bool) []byte {
	if len(rules) == 0 || len(data) == 0 {
		return data
	}

	oldToNew := make(map[string]string, len(rules))
	newToOld := make(map[string]string, len(rules))
	inlineOld := make(map[string]struct{}, len(rules))
	for _, r := range rules {
		oldToNew[r.OldKey] = r.NewKey
		newToOld[r.NewKey] = r.OldKey
		if r.Inline {
			inlineOld[r.OldKey] = struct{}{}
		}
	}

	var buf bytes.Buffer
	dec := jsontext.NewDecoder(bytes.NewReader(data), jsontext.AllowDuplicateNames(true))
	encOpts := []jsontext.Options{jsontext.AllowDuplicateNames(true)}
	if !compact {
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
				if len(scopes) > 0 && newToOld[keyName] != "" {
					scopes[len(scopes)-1].naturalNew[keyName] = true
				}

				// Check if this key is deprecated and should generate a duplicate.
				newKey, shouldDuplicate := oldToNew[keyName]
				if shouldDuplicate {
					// Write the old key.
					if err := enc.WriteToken(tok); err != nil {
						return data
					}

					// Read the raw value.
					val, err := dec.ReadValue()
					if err != nil {
						return data
					}

					// Old-named subtree. By default it is written as-is (the
					// value already uses old names from json.Marshal). An Inline
					// container instead re-runs the duplicator over its value so
					// nested renames also surface under the old name, the way
					// they did before this container was renamed.
					if _, ok := inlineOld[keyName]; ok && startsWithContainer(val) {
						// compact is irrelevant here: the result is re-encoded
						// by the outer encoder, which applies its own indent.
						oldVal := duplicateJSONKeys([]byte(val), rules, true)
						if err := enc.WriteValue(jsontext.Value(oldVal)); err != nil {
							return data
						}
					} else if err := enc.WriteValue(val); err != nil {
						return data
					}

					// New-named sibling: a clean, fully new-named copy. For a
					// scalar this is the same value, which yields an in-place
					// duplicate (both old and new key on the same object).
					newVal, renameErr := RewriteOldToNewKeys([]byte(val), rules)
					if renameErr != nil {
						newVal = []byte(val) // fall back to original value on error
					}
					if len(scopes) > 0 {
						scopes[len(scopes)-1].pending = append(
							scopes[len(scopes)-1].pending,
							pendingDup{newKey: newKey, value: jsontext.Value(newVal)},
						)
					}
				} else { // !shouldDuplicate (no old key match) — just write the key as-is
					if err := enc.WriteToken(tok); err != nil {
						return data
					}
				}
			} else { // !isKey — string value, not a key — just write as-is
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

// startsWithContainer reports whether the JSON value v is an object or array
// (as opposed to a scalar: string, number, bool, or null).
func startsWithContainer(v []byte) bool {
	for _, b := range v {
		switch b {
		case ' ', '\t', '\n', '\r':
			continue
		case '{', '[':
			return true
		default:
			return false
		}
	}
	return false
}
