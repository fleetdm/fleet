package endpointer

import (
	"bytes"
	"fmt"
	"io"
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
// It works by buffering output and scanning through the JSON byte stream with
// a minimal state machine that understands JSON structure well enough to
// identify object keys (strings immediately followed by ':') vs. string values.
//
// The rewriter does NOT fully parse JSON—it only needs to track enough state
// to distinguish keys from values and handle string escaping correctly.
type JSONKeyRewriteReader struct {
	src   io.Reader
	rules []AliasRule

	// Map from old key to rule index for fast lookup.
	oldKeyIndex map[string]int

	// Output buffer: rewritten bytes ready to be read by the caller.
	outBuf bytes.Buffer

	// Input buffer for reading from src.
	inBuf []byte

	// Tracks which deprecated keys have been used (old key -> true).
	usedDeprecated map[string]bool

	// err stores a sticky error from src or from conflict detection.
	err error

	// --- State machine fields ---

	// inString is true when we are inside a JSON string.
	inString bool
	// escape is true when the previous character was a backslash inside a string.
	escape bool
	// stringBuf captures the current string content (without quotes) when we
	// are inside a string, so we can check if it's a key that needs rewriting.
	stringBuf bytes.Buffer
	// depth tracks brace/bracket nesting to scope conflict detection.
	// Not currently used for scoping but reserved for future use.
	depth int

	// expectColon is set after we finish reading a string. If the next
	// non-whitespace character is ':', we know the string was a key.
	expectColon bool
	// pendingString holds the raw string (including quotes) that we just
	// finished reading, held until we determine if it's a key or value.
	pendingString []byte
	// pendingContent holds the unquoted content of the pending string.
	pendingContent string

	// keysInCurrentObject tracks keys seen in the current object scope for
	// conflict detection. Maps from new key name -> true.
	// We use a stack of maps for nested objects.
	keyScopes []map[string]bool
}

// NewJSONKeyRewriteReader creates a new JSONKeyRewriteReader that wraps the
// given reader and applies the provided alias rules.
func NewJSONKeyRewriteReader(src io.Reader, rules []AliasRule) *JSONKeyRewriteReader {
	idx := make(map[string]int, len(rules))
	for i, r := range rules {
		idx[r.OldKey] = i
	}
	return &JSONKeyRewriteReader{
		src:            src,
		rules:          rules,
		oldKeyIndex:    idx,
		inBuf:          make([]byte, 4096),
		usedDeprecated: make(map[string]bool),
	}
}

// UsedDeprecatedKeys returns the list of deprecated key names that were
// encountered during reading. This should be called after the reader has been
// fully consumed (i.e., after json.Decoder.Decode or similar has returned).
func (r *JSONKeyRewriteReader) UsedDeprecatedKeys() []string {
	keys := make([]string, 0, len(r.usedDeprecated))
	for k := range r.usedDeprecated {
		keys = append(keys, k)
	}
	return keys
}

// Read implements io.Reader. It reads from the underlying source, rewrites
// deprecated keys, and writes the result to p.
func (r *JSONKeyRewriteReader) Read(p []byte) (int, error) {
	// If we have buffered output, drain it first.
	if r.outBuf.Len() > 0 {
		return r.outBuf.Read(p)
	}

	// If we've already hit an error, return it.
	if r.err != nil {
		return 0, r.err
	}

	// Read more data from source.
	n, err := r.src.Read(r.inBuf)
	if n > 0 {
		if processErr := r.process(r.inBuf[:n]); processErr != nil {
			r.err = processErr
			// Flush what we have and then return the error on next read.
			if r.outBuf.Len() > 0 {
				return r.outBuf.Read(p)
			}
			return 0, r.err
		}
	}

	if err != nil {
		// Before returning EOF, flush any pending string as a value (non-key).
		if err == io.EOF && r.pendingString != nil {
			r.outBuf.Write(r.pendingString)
			r.pendingString = nil
			r.pendingContent = ""
			r.expectColon = false
		}
		if r.outBuf.Len() > 0 {
			read, _ := r.outBuf.Read(p)
			// Only return the src error if we've drained the buffer.
			if r.outBuf.Len() == 0 {
				return read, err
			}
			return read, nil
		}
		return 0, err
	}

	if r.outBuf.Len() > 0 {
		return r.outBuf.Read(p)
	}

	// No output produced yet, try again.
	return 0, nil
}

// process scans the input bytes through the state machine and produces output.
func (r *JSONKeyRewriteReader) process(data []byte) error {
	for _, b := range data {
		if r.inString {
			r.stringBuf.WriteByte(b)
			if r.escape {
				r.escape = false
				continue
			}
			if b == '\\' {
				r.escape = true
				continue
			}
			if b == '"' {
				// End of string. Capture the content (without the closing quote).
				raw := r.stringBuf.Bytes()
				// raw includes everything inside the string plus the closing quote.
				// The content is raw[:len(raw)-1] (without the closing quote).
				content := string(raw[:len(raw)-1])

				// Build the full quoted string for output.
				quoted := make([]byte, 0, len(raw)+1)
				quoted = append(quoted, '"')
				quoted = append(quoted, raw...)

				r.inString = false
				r.stringBuf.Reset()

				// We now need to determine if this string is a key or a value.
				// A key is a string followed by ':'. We can't know until we
				// see the next non-whitespace character.
				// Hold the string in pending state.
				r.pendingString = quoted
				r.pendingContent = content
				r.expectColon = true
			}
			continue
		}

		// Not inside a string.
		if r.expectColon {
			if isJSONWhitespace(b) {
				// Buffer whitespace between a potential key and ':'.
				r.outBuf.WriteByte(b)
				continue
			}
			if b == ':' {
				// The pending string IS a key.
				if err := r.emitKey(r.pendingContent, r.pendingString); err != nil {
					return err
				}
				r.outBuf.WriteByte(b)
				r.pendingString = nil
				r.pendingContent = ""
				r.expectColon = false
				continue
			}
			// Not a colon — the pending string was a value, not a key.
			r.outBuf.Write(r.pendingString)
			r.pendingString = nil
			r.pendingContent = ""
			r.expectColon = false
			// Fall through to handle the current byte.
		}

		switch b {
		case '"':
			r.inString = true
			r.escape = false
			r.stringBuf.Reset()
		case '{':
			r.depth++
			r.keyScopes = append(r.keyScopes, make(map[string]bool))
			r.outBuf.WriteByte(b)
		case '}':
			r.depth--
			if len(r.keyScopes) > 0 {
				r.keyScopes = r.keyScopes[:len(r.keyScopes)-1]
			}
			r.outBuf.WriteByte(b)
		case '[':
			r.outBuf.WriteByte(b)
		case ']':
			r.outBuf.WriteByte(b)
		default:
			r.outBuf.WriteByte(b)
		}
	}
	return nil
}

// emitKey handles a JSON key: checks if it needs rewriting, tracks deprecated
// usage, and detects conflicts.
func (r *JSONKeyRewriteReader) emitKey(content string, original []byte) error {
	// Determine the canonical (new) key name for conflict detection.
	canonicalKey := content
	rewritten := false

	if idx, ok := r.oldKeyIndex[content]; ok {
		// This is a deprecated key — rewrite it.
		rule := r.rules[idx]
		canonicalKey = rule.NewKey
		r.usedDeprecated[content] = true
		rewritten = true
	}

	// Check for alias conflict in the current object scope.
	if len(r.keyScopes) > 0 {
		scope := r.keyScopes[len(r.keyScopes)-1]
		if scope[canonicalKey] {
			// Conflict! Both the old and new key exist in this object.
			// Find the rule to report the error with proper names.
			for _, rule := range r.rules {
				if rule.NewKey == canonicalKey {
					return &AliasConflictError{Old: rule.OldKey, New: rule.NewKey}
				}
			}
			// Shouldn't happen, but just in case:
			return &AliasConflictError{Old: content, New: canonicalKey}
		}
		scope[canonicalKey] = true
	}

	if rewritten {
		// Write the rewritten key.
		rule := r.rules[r.oldKeyIndex[content]]
		r.outBuf.WriteByte('"')
		r.outBuf.WriteString(rule.NewKey)
		r.outBuf.WriteByte('"')
	} else {
		// Write the original key unchanged.
		r.outBuf.Write(original)
	}

	return nil
}

func isJSONWhitespace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}
