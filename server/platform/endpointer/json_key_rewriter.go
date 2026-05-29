package endpointer

import (
	"bytes"
	"fmt"
	"io"
	"strings"

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
	return fmt.Sprintf("Conflicting field names: cannot specify both `%s` (deprecated) and `%s` in the same request", e.Old, e.New)
}

// pathSep joins JSON object-key path segments in AliasRule.Path. It uses the
// ASCII unit separator control character, which never appears in a JSON object
// key name in practice.
const pathSep = "\x1f"

// AliasRule defines a key-rename rule: the deprecated (old) key name and its
// replacement (new) key name. The struct's json tag uses OldKey (the current
// name), and renameto specifies NewKey (the target name). The rewriter
// accepts both names in requests: OldKey passes through as-is (with
// deprecation tracking) and NewKey is rewritten to OldKey for deserialization.
//
// Path and Scoped together scope a rule to a specific JSON location, which is
// necessary because a rename's NewKey can collide with an unrelated, literal
// JSON field of the same name elsewhere in the document (e.g. the
// `macos_setup`→`setup_experience` rename collides with the per-software
// `setup_experience` install flag). When Scoped is true, the rule applies only
// inside the object reached by the pathSep-joined sequence of (canonical/old)
// object keys from the document root that equals Path — an empty Path then
// denotes the document root. When Scoped is false the rule is unscoped and
// applies at any depth (the legacy behavior, used by directly-constructed
// rules and by RewriteOldToNewKeys).
type AliasRule struct {
	OldKey string
	NewKey string
	Path   string
	Scoped bool
}

// buildIndexes constructs the lookup maps used by the rewriter: old/new key
// name to the rules carrying that name (multiple rules can share a name with
// different paths), plus a canonical map from new key name to old key name
// used to normalize ancestor path segments.
func buildIndexes(rules []AliasRule) (oldIdx, newIdx map[string][]AliasRule, canonical map[string]string) {
	oldIdx = make(map[string][]AliasRule, len(rules))
	newIdx = make(map[string][]AliasRule, len(rules))
	canonical = make(map[string]string, len(rules))
	for _, r := range rules {
		oldIdx[r.OldKey] = append(oldIdx[r.OldKey], r)
		newIdx[r.NewKey] = append(newIdx[r.NewKey], r)
		if _, ok := canonical[r.NewKey]; !ok {
			canonical[r.NewKey] = r.OldKey
		}
	}
	return oldIdx, newIdx, canonical
}

// matchRule selects, from the candidate rules registered under a key name, the
// one that applies at the given JSON path. A scoped rule applies only when its
// Path matches exactly; an unscoped rule applies at any path and is used as a
// fallback. Returns false if no candidate applies here.
func matchRule(cands []AliasRule, path string) (AliasRule, bool) {
	var fallback AliasRule
	haveFallback := false
	for _, c := range cands {
		if c.Scoped {
			if c.Path == path {
				return c, true
			}
			continue
		}
		if !haveFallback {
			fallback = c
			haveFallback = true
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

	// Map from old (deprecated) key to the rules carrying that name.
	oldKeyIndex map[string][]AliasRule
	// Map from new key to the rules carrying that name.
	newKeyIndex map[string][]AliasRule
	// canonical maps a new key name to its old (canonical) name, used to
	// normalize ancestor path segments so scoped matching is independent of
	// whether the input used old or new names for ancestor object keys.
	canonical map[string]string

	// Tracks which deprecated keys have been used (old key -> true).
	usedDeprecated map[string]bool
}

// NewJSONKeyRewriteReader creates a new JSONKeyRewriteReader that wraps the
// given reader and applies the provided alias rules. It reads JSON tokens
// from src, handles bidirectional key aliasing, detects conflicts, and
// writes the result to an internal buffer.
func NewJSONKeyRewriteReader(src io.Reader, rules []AliasRule) *JSONKeyRewriteReader {
	oldIdx, newIdx, canonical := buildIndexes(rules)

	rw := &JSONKeyRewriteReader{
		oldKeyIndex:    oldIdx,
		newKeyIndex:    newIdx,
		canonical:      canonical,
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
	if len(rules) == 0 || len(data) == 0 {
		return data, nil, nil
	}
	oldIdx, newIdx, canonical := buildIndexes(rules)
	rw := &JSONKeyRewriteReader{
		oldKeyIndex:    oldIdx,
		newKeyIndex:    newIdx,
		canonical:      canonical,
		usedDeprecated: make(map[string]bool),
	}
	var buf bytes.Buffer
	if err := rw.rewrite(bytes.NewReader(data), &buf); err != nil {
		return nil, nil, err
	}
	deprecatedKeysMap := make(map[string]string, len(rw.usedDeprecated))
	for k := range rw.usedDeprecated {
		if cands := rw.oldKeyIndex[k]; len(cands) > 0 {
			deprecatedKeysMap[k] = cands[0].NewKey
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
	reversed := make([]AliasRule, len(rules))
	for i, r := range rules {
		// Reversed rules are intentionally unscoped (Scoped left false): the
		// old→new direction is used to rewrite values/subtrees whose paths are
		// relative to that subtree, not the document root, so document-rooted
		// scoping would never match. Unscoped preserves the original global
		// behavior, which is safe here because old→new only renames the old
		// names (a colliding literal already uses the new name and passes
		// through untouched).
		reversed[i] = AliasRule{OldKey: r.NewKey, NewKey: r.OldKey}
	}
	result, _, err := RewriteDeprecatedKeys(data, reversed)
	return result, err
}

// markConflict records that the canonical (old) key for rule has been seen in
// the current object scope and returns an *AliasConflictError if it was
// already present (i.e. both the old and new names appear in the same scope).
func markConflict(keyScopes []map[string]bool, rule AliasRule) error {
	if len(keyScopes) == 0 {
		return nil
	}
	scope := keyScopes[len(keyScopes)-1]
	if scope[rule.OldKey] {
		return &AliasConflictError{Old: rule.OldKey, New: rule.NewKey}
	}
	scope[rule.OldKey] = true
	return nil
}

// rewrite reads tokens from src, rewrites deprecated keys, checks for alias
// conflicts, and writes the transformed JSON to w.
//
// It tracks the current JSON object-key path so that scoped rules apply only
// at their declared location. Path segments are normalized to canonical (old)
// names so a scoped rule matches regardless of whether ancestor object keys
// were written with old or new names.
func (r *JSONKeyRewriteReader) rewrite(src io.Reader, w io.Writer) error {
	dec := jsontext.NewDecoder(src, jsontext.AllowDuplicateNames(true))
	enc := jsontext.NewEncoder(w, jsontext.AllowDuplicateNames(true))

	// Stack of per-object-scope key sets for conflict detection.
	// Pushed on '{', popped on '}'.
	var keyScopes []map[string]bool

	// Current path of canonical object keys to the position being read, and a
	// parallel stack recording whether each open container pushed a segment.
	var pathSegs []string
	var segPushed []bool
	// pendingKey is the canonical name of the most recently read object key,
	// whose value has not yet been consumed. It becomes the path segment if
	// that value is a container.
	pendingKey := ""

	canon := func(k string) string {
		if old, ok := r.canonical[k]; ok {
			return old
		}
		return k
	}
	openContainer := func() {
		if pendingKey != "" {
			pathSegs = append(pathSegs, pendingKey)
			segPushed = append(segPushed, true)
		} else {
			segPushed = append(segPushed, false)
		}
		pendingKey = ""
	}
	closeContainer := func() {
		if n := len(segPushed); n > 0 {
			if segPushed[n-1] {
				pathSegs = pathSegs[:len(pathSegs)-1]
			}
			segPushed = segPushed[:n-1]
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

		switch tok.Kind() {
		case '{':
			keyScopes = append(keyScopes, make(map[string]bool))
			openContainer()
			if err := enc.WriteToken(tok); err != nil {
				return err
			}

		case '[':
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

			if !isKey {
				// String value — pass through unchanged.
				if err := enc.WriteToken(tok); err != nil {
					return err
				}
				pendingKey = ""
				continue
			}

			keyName := tok.String()
			curPath := strings.Join(pathSegs, pathSep)

			// Old (deprecated) key: pass through as-is — the struct expects
			// this name — and track it for deprecation logging.
			if cands, ok := r.oldKeyIndex[keyName]; ok {
				if rule, matched := matchRule(cands, curPath); matched {
					r.usedDeprecated[keyName] = true
					if err := markConflict(keyScopes, rule); err != nil {
						return err
					}
					if err := enc.WriteToken(tok); err != nil {
						return err
					}
					pendingKey = rule.OldKey
					continue
				}
			}

			// New key applicable at this path: rewrite it to OldKey so the
			// struct can deserialize it.
			if cands, ok := r.newKeyIndex[keyName]; ok {
				if rule, matched := matchRule(cands, curPath); matched {
					if err := markConflict(keyScopes, rule); err != nil {
						return err
					}
					if err := enc.WriteToken(jsontext.String(rule.OldKey)); err != nil {
						return err
					}
					pendingKey = rule.OldKey
					continue
				}
			}

			// Not an aliased key applicable here — pass through unchanged.
			if err := enc.WriteToken(tok); err != nil {
				return err
			}
			pendingKey = canon(keyName)

		default:
			// All other tokens: numbers, bools, null — scalar values.
			if err := enc.WriteToken(tok); err != nil {
				return err
			}
			pendingKey = ""
		}
	}
}
