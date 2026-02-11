package endpointer

import (
	"bytes"
	"fmt"
	"io"

	"github.com/go-json-experiment/json/jsontext"
)

// AliasConflictError is returned when both the deprecated and new field names
// are specified in the same JSON object scope. For example, if "team_id" is
// deprecated in favor of "fleet_id", and a request contains both, this error
// is returned.
type AliasConflictError struct {
	Old string
	New string
}

func (e *AliasConflictError) Error() string {
	return fmt.Sprintf("Conflicting field names: cannot specify both %q (deprecated) and %q in the same request", e.Old, e.New)
}

// AliasRule defines a key-rename rule: the deprecated (old) key name and its
// replacement (new) key name. The rewriter will rewrite OldKey to NewKey when
// found in the JSON stream.
type AliasRule struct {
	OldKey string
	NewKey string
}

// JSONKeyRewriteReader is a streaming io.Reader that rewrites deprecated JSON
// object keys to their new names while reading. It also:
//
//   - Tracks which deprecated keys were encountered (for deprecation logging).
//   - Detects alias conflicts: if both the deprecated and new key appear in the
//     same JSON object scope, it returns an *AliasConflictError.
//
// Internally it uses an io.Pipe with a background goroutine that reads tokens
// from the source using jsontext.Decoder and writes rewritten tokens via
// jsontext.Encoder. This keeps the streaming io.Reader interface while
// delegating all JSON lexing (string escaping, unicode, whitespace) to the
// standard library.
type JSONKeyRewriteReader struct {
	reader  *bytes.Reader
	initErr error

	// Map from old key to its AliasRule for fast lookup.
	oldKeyIndex map[string]AliasRule

	// Tracks which deprecated keys have been used (old key -> true).
	// Written by the goroutine, read after the pipe is drained (wg.Wait).
	usedDeprecated map[string]bool
}

// NewJSONKeyRewriteReader creates a new JSONKeyRewriteReader that wraps the
// given reader and applies the provided alias rules. A background goroutine
// reads JSON tokens from src, rewrites deprecated keys, detects conflicts,
// and writes the result to the returned reader.
func NewJSONKeyRewriteReader(src io.Reader, rules []AliasRule) *JSONKeyRewriteReader {
	idx := make(map[string]AliasRule, len(rules))
	for _, r := range rules {
		idx[r.OldKey] = r
	}

	rw := &JSONKeyRewriteReader{
		oldKeyIndex:    idx,
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

// rewrite reads tokens from src, rewrites deprecated keys, checks for alias
// conflicts, and writes the transformed JSON to w.
func (r *JSONKeyRewriteReader) rewrite(src io.Reader, w io.Writer) error {
	dec := jsontext.NewDecoder(src)
	enc := jsontext.NewEncoder(w)

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

				// Check if this is a deprecated key that needs rewriting.
				if rule, ok := r.oldKeyIndex[keyName]; ok {
					canonicalKey := rule.NewKey
					r.usedDeprecated[keyName] = true

					// Conflict detection.
					if len(keyScopes) > 0 {
						scope := keyScopes[len(keyScopes)-1]
						if scope[canonicalKey] {
							return &AliasConflictError{Old: rule.OldKey, New: rule.NewKey}
						}
						scope[canonicalKey] = true
					}

					// Write the rewritten (new) key.
					if err := enc.WriteToken(jsontext.String(canonicalKey)); err != nil {
						return err
					}
				} else {
					// Not a deprecated key — check for conflict with a
					// previously-rewritten key in this scope.
					canonicalKey := keyName
					if len(keyScopes) > 0 {
						scope := keyScopes[len(keyScopes)-1]
						if scope[canonicalKey] {
							// This new key was already seen via a rewritten old key.
							for _, rule := range r.oldKeyIndex {
								if rule.NewKey == canonicalKey {
									return &AliasConflictError{Old: rule.OldKey, New: rule.NewKey}
								}
							}
						}
						scope[canonicalKey] = true
					}

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
