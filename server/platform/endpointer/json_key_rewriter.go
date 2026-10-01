package endpointer

import (
	"bytes"
	"encoding/json/jsontext"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
)

// AliasConflictError is returned when both the deprecated and new field names
// are specified in the same JSON object scope. For example, if "team_id" is
// renamed to "fleet_id", and a request contains both, this error is returned.
type AliasConflictError struct {
	Old string
	New string
}

func (e *AliasConflictError) Error() string {
	return fmt.Sprintf("Conflicting field names: cannot specify both `%s` (deprecated) and `%s` in the same request", e.Old, e.New)
}

// AliasRule defines a key-rename rule: the deprecated (old) key name and its
// replacement (new) key name. The struct's json tag uses OldKey (the current
// name), and renameto specifies NewKey (the target name). The rewriter
// accepts both names in requests: OldKey passes through as-is (with
// deprecation tracking) and NewKey is rewritten to OldKey for deserialization.
type AliasRule struct {
	OldKey string
	NewKey string
	// Inline opts a renamed container into "merged" response duplication:
	// instead of the default clean split (the old key holds an all-old subtree
	// and the new key an all-new one), the old key's subtree also carries the
	// new-named copies of any nested renamed containers — so both names appear
	// together on the same object. Set via the `,inline` option on the
	// `renameto` struct tag (e.g. `renameto:"ab_tokens,inline"`). It only
	// affects response encoding (DuplicateJSONKeys); request decoding ignores
	// it.
	Inline bool
	// Scope restricts the rule to keys sitting directly inside an object that was reached through one of these key
	// names. Empty means the rule applies at any depth, which is how every rule behaved before scoping existed. Set via
	// the `renamescope` struct tag.
	Scope []string
}

// appliesIn reports whether the rule may rewrite a key sitting directly inside an object reached through enclosingKey.
// The root object of a document has an empty enclosingKey, which only unscoped rules match.
func (r AliasRule) appliesIn(enclosingKey string) bool {
	return len(r.Scope) == 0 || slices.Contains(r.Scope, enclosingKey)
}

// key returns a comparable identity for the rule, for deduplication.
func (r AliasRule) key() string {
	return strings.Join(append([]string{r.OldKey, r.NewKey, strconv.FormatBool(r.Inline)}, r.Scope...), "\x00")
}

// aliasIndex groups rules by key name. A name can carry more than one rule when the same name is renamed differently in
// different objects, so lookups are resolved against the enclosing key.
type aliasIndex map[string][]AliasRule

func newAliasIndex(rules []AliasRule, keyOf func(AliasRule) string) aliasIndex {
	idx := make(aliasIndex, len(rules))
	for _, r := range rules {
		k := keyOf(r)
		idx[k] = append(idx[k], r)
	}
	return idx
}

// lookup returns the rule registered under key that applies inside enclosingKey. A scoped rule wins over an unscoped one
// so that declaration order can't decide which of two same-named rules matches.
func (idx aliasIndex) lookup(key, enclosingKey string) (AliasRule, bool) {
	var fallback AliasRule
	var haveFallback bool
	for _, r := range idx[key] {
		if len(r.Scope) == 0 {
			if !haveFallback {
				fallback, haveFallback = r, true
			}
			continue
		}
		if r.appliesIn(enclosingKey) {
			return r, true
		}
	}
	return fallback, haveFallback
}

// JSONKeyRewriteReader is a streaming io.Reader that handles
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

	// Rules indexed by old (deprecated) key name.
	oldKeyIndex aliasIndex
	// Rules indexed by new key name.
	newKeyIndex aliasIndex

	// rootKey is the object key the document being rewritten was nested under, empty for a whole document. It seeds
	// scope matching so a fragment lifted out of a larger document still resolves scoped rules.
	rootKey string

	// Tracks which deprecated keys have been used (old key -> true).
	usedDeprecated map[string]bool
}

// NewJSONKeyRewriteReader creates a new JSONKeyRewriteReader that wraps the
// given reader and applies the provided alias rules. It reads JSON tokens
// from src, handles bidirectional key aliasing, detects conflicts, and
// writes the result to an internal buffer.
func NewJSONKeyRewriteReader(src io.Reader, rules []AliasRule) *JSONKeyRewriteReader {
	rw := &JSONKeyRewriteReader{
		oldKeyIndex:    newAliasIndex(rules, func(r AliasRule) string { return r.OldKey }),
		newKeyIndex:    newAliasIndex(rules, func(r AliasRule) string { return r.NewKey }),
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

// RewriteDeprecatedKeys handles JSON key aliasing in data using
// the provided alias rules. It rewrites NewKey→OldKey (so the struct can
// deserialize), passes through OldKey as-is, and returns an error if both
// appear in the same scope (alias conflict) or the JSON is malformed.
//
// This is useful when a request body is captured as json.RawMessage and later
// decoded into a struct with `renameto` tags — the rewriter in MakeDecoder
// won't have seen the inner fields, so this function can be called before the
// deferred unmarshal.
func RewriteDeprecatedKeys(data []byte, rules []AliasRule) ([]byte, map[string]string, error) {
	return rewriteDeprecatedKeysIn(data, rules, "")
}

// rewriteDeprecatedKeysIn is RewriteDeprecatedKeys for a fragment that was nested under rootKey in a larger document, so
// that rules scoped to that key still apply. rootKey is empty for a whole document.
func rewriteDeprecatedKeysIn(data []byte, rules []AliasRule, rootKey string) ([]byte, map[string]string, error) {
	if len(rules) == 0 || len(data) == 0 {
		return data, nil, nil
	}
	rw := &JSONKeyRewriteReader{
		oldKeyIndex:    newAliasIndex(rules, func(r AliasRule) string { return r.OldKey }),
		newKeyIndex:    newAliasIndex(rules, func(r AliasRule) string { return r.NewKey }),
		rootKey:        rootKey,
		usedDeprecated: make(map[string]bool),
	}
	var buf bytes.Buffer
	if err := rw.rewrite(bytes.NewReader(data), &buf); err != nil {
		return nil, nil, err
	}
	deprecatedKeysMap := make(map[string]string, len(rw.usedDeprecated))
	for k := range rw.usedDeprecated {
		if rules := rw.oldKeyIndex[k]; len(rules) > 0 {
			deprecatedKeysMap[k] = rules[0].NewKey
		}
	}
	return buf.Bytes(), deprecatedKeysMap, nil
}

// RewriteOldToNewKeys is the reverse of RewriteDeprecatedKey; it takes
// the rules and reverses them before translating keys.
// Use this in situations where a payload was rewritten from new to old keys
// for deserialization, but you want to return a response with the new keys
// for forward compatibility.
func RewriteOldToNewKeys(data []byte, rules []AliasRule) ([]byte, error) {
	return rewriteOldToNewKeysIn(data, rules, "")
}

// rewriteOldToNewKeysIn is RewriteOldToNewKeys for a fragment that was nested under rootKey in a larger document.
func rewriteOldToNewKeysIn(data []byte, rules []AliasRule, rootKey string) ([]byte, error) {
	reversed := make([]AliasRule, len(rules))
	for i, r := range rules {
		// Inline is intentionally not preserved: this only renames keys, it
		// never duplicates them.
		reversed[i] = AliasRule{OldKey: r.NewKey, NewKey: r.OldKey, Scope: r.Scope}
	}
	result, _, err := rewriteDeprecatedKeysIn(data, reversed, rootKey)
	return result, err
}

// softwareScopeKey marks the JSON object/array container whose contents are
// the values of TeamSpec.Software — i.e. SoftwarePackageSpec /
// TeamSpecAppStoreApp / MaintainedAppSpec items. The literal `setup_experience`
// install flag on those items collides with the `macos_setup`↔`setup_experience`
// rename on the MDM section, so renames are skipped under this subtree. See
// https://github.com/fleetdm/fleet/issues/44970.
const softwareScopeKey = "software"

// rewrite reads tokens from src, rewrites deprecated keys, checks for alias
// conflicts, and writes the transformed JSON to w.
func (r *JSONKeyRewriteReader) rewrite(src io.Reader, w io.Writer) error {
	dec := jsontext.NewDecoder(src, jsontext.AllowDuplicateNames(true))
	enc := jsontext.NewEncoder(w, jsontext.AllowDuplicateNames(true))

	// Stack of per-object-scope key sets for conflict detection.
	// Pushed on '{', popped on '}'.
	var keyScopes []map[string]bool

	// Track whether we are currently inside the `software` subtree.
	// pendingKey is the most-recent object key whose value has not yet been
	// read; openContainer/closeContainer maintain softwareDepth by checking
	// whether the container being opened lives under `software`.
	pendingKey := ""
	softwareDepth := 0

	// enclosing records, per open container, the object key that container was reached through, so scoped rules can be
	// resolved. Array elements inherit the array's own key.
	var enclosing []string
	currentEnclosing := func() string {
		if len(enclosing) == 0 {
			return r.rootKey
		}
		return enclosing[len(enclosing)-1]
	}
	openContainer := func() {
		if softwareDepth > 0 || pendingKey == softwareScopeKey {
			softwareDepth++
		}
		key := pendingKey
		if key == "" {
			key = currentEnclosing()
		}
		enclosing = append(enclosing, key)
		pendingKey = ""
	}
	closeContainer := func() {
		if softwareDepth > 0 {
			softwareDepth--
		}
		if len(enclosing) > 0 {
			enclosing = enclosing[:len(enclosing)-1]
		}
	}

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
			openContainer()
			if err := enc.WriteToken(tok); err != nil {
				return err
			}

		case '}':
			if len(keyScopes) > 0 {
				keyScopes = keyScopes[:len(keyScopes)-1]
			}
			closeContainer()
			if err := enc.WriteToken(tok); err != nil {
				return err
			}

		case '[':
			openContainer()
			if err := enc.WriteToken(tok); err != nil {
				return err
			}

		case ']':
			closeContainer()
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

				// Inside the `software` subtree, all inner keys are literal —
				// most notably each item's `setup_experience` is a bool
				// install flag, not the renamed `macos_setup` container. Pass
				// them through untouched.
				if softwareDepth > 0 {
					pendingKey = keyName
					if err := enc.WriteToken(tok); err != nil {
						return err
					}
					continue
				}

				// Use OldKey as the canonical key for scope tracking.
				// Both OldKey (pass-through) and NewKey (rewrite) resolve
				// to the same canonical key for conflict detection.

				if rule, ok := r.oldKeyIndex.lookup(keyName, currentEnclosing()); ok {
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

					pendingKey = keyName
					// Write the key as-is (old name, which the struct expects).
					if err := enc.WriteToken(tok); err != nil {
						return err
					}
				} else if rule, ok := r.newKeyIndex.lookup(keyName, currentEnclosing()); ok {
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

					pendingKey = canonicalKey
					// Write the rewritten (old) key.
					if err := enc.WriteToken(jsontext.String(canonicalKey)); err != nil {
						return err
					}
				} else {
					// Not an aliased key — pass through unchanged.
					pendingKey = keyName
					if err := enc.WriteToken(tok); err != nil {
						return err
					}
				}
			} else {
				// String value — pass through unchanged.
				pendingKey = ""
				if err := enc.WriteToken(tok); err != nil {
					return err
				}
			}

		default:
			// All other tokens: numbers, bools, null — scalar values.
			pendingKey = ""
			if err := enc.WriteToken(tok); err != nil {
				return err
			}
		}
	}
}
