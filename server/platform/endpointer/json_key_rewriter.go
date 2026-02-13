package endpointer

import (
	"bytes"
	"fmt"
	"io"

	"github.com/go-json-experiment/json/jsontext"
)

// AliasConflictError is returned when both the deprecated and new field names
// are specified in the same JSON object scope. For example, if "team_id" is
// renamed to "fleet_id", and a request contains both, this error is returned.
type AliasConflictError struct {
	Old string
	New string
}

func (e *AliasConflictError) Error() string {
	return fmt.Sprintf("Conflicting field names: cannot specify both %q (deprecated) and %q in the same request", e.Old, e.New)
}

// AliasRule defines a key-rename rule: the deprecated (old) key name and its
// replacement (new) key name. The struct's json tag uses OldKey (the current
// name), and renameto specifies NewKey (the target name). The rewriter
// accepts both names in requests: OldKey passes through as-is (with
// deprecation tracking) and NewKey is rewritten to OldKey for deserialization.
type AliasRule struct {
	OldKey string
	NewKey string
}

// JSONKeyRewriteReader is a streaming io.Reader that handles bidirectional
// JSON key aliasing while reading. It:
//
//   - Passes through OldKey (deprecated) names as-is (the struct expects them)
//     and tracks them in usedDeprecated for deprecation logging.
//   - Rewrites NewKey names to OldKey so the struct can deserialize them.
//   - Detects alias conflicts: if both OldKey and NewKey appear in the same
//     JSON object scope, it returns an *AliasConflictError.
//
// It uses jsontext.Decoder/Encoder for token-level processing, delegating all
// JSON lexing (string escaping, unicode, whitespace) to the library.
type JSONKeyRewriteReader struct {
	reader  *bytes.Reader
	initErr error

	// Map from old (deprecated) key to its AliasRule for fast lookup.
	oldKeyIndex map[string]AliasRule
	// Map from new key to its AliasRule for fast lookup.
	newKeyIndex map[string]AliasRule

	// Tracks which deprecated keys have been used (old key -> true).
	usedDeprecated map[string]bool
}

// NewJSONKeyRewriteReader creates a new JSONKeyRewriteReader that wraps the
// given reader and applies the provided alias rules. It reads JSON tokens
// from src, handles bidirectional key aliasing, detects conflicts, and
// writes the result to an internal buffer.
func NewJSONKeyRewriteReader(src io.Reader, rules []AliasRule) *JSONKeyRewriteReader {
	oldIdx := make(map[string]AliasRule, len(rules))
	newIdx := make(map[string]AliasRule, len(rules))
	for _, r := range rules {
		oldIdx[r.OldKey] = r
		newIdx[r.NewKey] = r
	}

	rw := &JSONKeyRewriteReader{
		oldKeyIndex:    oldIdx,
		newKeyIndex:    newIdx,
		usedDeprecated: make(map[string]bool),
	}

	var buf bytes.Buffer
	if err := rw.rewrite(src, &buf); err != nil {
		rw.initErr = err
		return rw
	}
	rw.reader = bytes.NewReader(buf.Bytes())
	return rw
}

// UsedDeprecatedKeys returns the list of deprecated key names that were
// encountered during reading. This should be called after the reader has been
// fully consumed (i.e., after json.Decoder.Decode or similar has returned),
// which guarantees the background goroutine has finished.
func (r *JSONKeyRewriteReader) UsedDeprecatedKeys() []string {
	keys := make([]string, 0, len(r.usedDeprecated))
	for k := range r.usedDeprecated {
		keys = append(keys, k)
	}
	return keys
}

// Close closes the reader end of the pipe to unblock the transform goroutine
// if the consumer stops reading early.
func (r *JSONKeyRewriteReader) Close() error {
	return nil
}

// Read implements io.Reader by reading from the pipe.
func (r *JSONKeyRewriteReader) Read(p []byte) (int, error) {
	if r.initErr != nil {
		return 0, r.initErr
	}
	if r.reader == nil {
		return 0, io.EOF
	}
	return r.reader.Read(p)
}

// RewriteDeprecatedKeys handles bidirectional JSON key aliasing in data using
// the provided alias rules. It rewrites NewKey→OldKey (so the struct can
// deserialize), passes through OldKey as-is, and returns an error if both
// appear in the same scope (alias conflict) or the JSON is malformed.
//
// This is useful when a request body is captured as json.RawMessage and later
// decoded into a struct with `renameto` tags — the rewriter in MakeDecoder
// won't have seen the inner fields, so this function can be called before the
// deferred unmarshal.
func RewriteDeprecatedKeys(data []byte, rules []AliasRule) ([]byte, error) {
	if len(rules) == 0 || len(data) == 0 {
		return data, nil
	}
	oldIdx := make(map[string]AliasRule, len(rules))
	newIdx := make(map[string]AliasRule, len(rules))
	for _, r := range rules {
		oldIdx[r.OldKey] = r
		newIdx[r.NewKey] = r
	}
	rw := &JSONKeyRewriteReader{
		oldKeyIndex:    oldIdx,
		newKeyIndex:    newIdx,
		usedDeprecated: make(map[string]bool),
	}
	var buf bytes.Buffer
	if err := rw.rewrite(bytes.NewReader(data), &buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// rewrite reads tokens from src, rewrites deprecated keys, checks for alias
// conflicts, and writes the transformed JSON to w.
func (r *JSONKeyRewriteReader) rewrite(src io.Reader, w io.Writer) error {
	dec := jsontext.NewDecoder(src, jsontext.AllowDuplicateNames(true))
	enc := jsontext.NewEncoder(w, jsontext.AllowDuplicateNames(true))

	// Stack of per-object-scope key sets for conflict detection.
	// Pushed on '{', popped on '}'.
	var keyScopes []map[string]bool

	for {
		tok, err := dec.ReadToken()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		kind := tok.Kind()

		switch kind {
		case '{':
			keyScopes = append(keyScopes, make(map[string]bool))
			if err := enc.WriteToken(tok); err != nil {
				return err
			}

		case '}':
			if len(keyScopes) > 0 {
				keyScopes = keyScopes[:len(keyScopes)-1]
			}
			if err := enc.WriteToken(tok); err != nil {
				return err
			}

		case '"':
			// Determine if this string is an object key by checking the
			// decoder's stack: inside an object ('{') at an odd length
			// means we just read a key (name).
			isKey := false
			depth := dec.StackDepth()
			if depth > 0 {
				parentKind, length := dec.StackIndex(depth)
				// length is odd after reading a name (names and values
				// are counted separately).
				if parentKind == '{' && length%2 == 1 {
					isKey = true
				}
			}

			if isKey {
				keyName := tok.String()

				// Use OldKey as the canonical key for scope tracking.
				// Both OldKey (pass-through) and NewKey (rewrite) resolve
				// to the same canonical key for conflict detection.

				if rule, ok := r.oldKeyIndex[keyName]; ok {
					// This is an OldKey (deprecated name). Pass through
					// as-is — the struct expects this name. Track it for
					// deprecation logging.
					canonicalKey := rule.OldKey
					r.usedDeprecated[keyName] = true

					// Conflict detection.
					if len(keyScopes) > 0 {
						scope := keyScopes[len(keyScopes)-1]
						if scope[canonicalKey] {
							return &AliasConflictError{Old: rule.OldKey, New: rule.NewKey}
						}
						scope[canonicalKey] = true
					}

					// Write the key as-is (old name, which the struct expects).
					if err := enc.WriteToken(tok); err != nil {
						return err
					}
				} else if rule, ok := r.newKeyIndex[keyName]; ok {
					// This is a NewKey. Rewrite it to OldKey so the
					// struct can deserialize it.
					canonicalKey := rule.OldKey

					// Conflict detection.
					if len(keyScopes) > 0 {
						scope := keyScopes[len(keyScopes)-1]
						if scope[canonicalKey] {
							return &AliasConflictError{Old: rule.OldKey, New: rule.NewKey}
						}
						scope[canonicalKey] = true
					}

					// Write the rewritten (old) key.
					if err := enc.WriteToken(jsontext.String(canonicalKey)); err != nil {
						return err
					}
				} else {
					// Not an aliased key — pass through unchanged.
					if err := enc.WriteToken(tok); err != nil {
						return err
					}
				}
			} else {
				// String value — pass through unchanged.
				if err := enc.WriteToken(tok); err != nil {
					return err
				}
			}

		default:
			// All other tokens: [, ], numbers, bools, null — pass through.
			if err := enc.WriteToken(tok); err != nil {
				return err
			}
		}
	}
}
