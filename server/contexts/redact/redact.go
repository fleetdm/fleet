// Package redact tracks secret values expanded while serving a request so they
// can be scrubbed from what is sent back to the caller. Validation runs on
// secret-expanded content, so any error that quotes that content would
// otherwise leak the secret.
package redact

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"io"
	"maps"
	"slices"
	"strconv"
	"sync"
)

type key int

const secretsKey key = 0

// Placeholder replaces every recorded secret value.
const Placeholder = "[redacted]"

type secrets struct {
	mu   sync.Mutex
	vals map[string]struct{}
}

// NewContext returns a context that records expanded secret values.
func NewContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, secretsKey, &secrets{vals: make(map[string]struct{})})
}

// Record adds vals to the context's secrets. It is a no-op when the context was
// not created with NewContext (e.g. cron jobs).
func Record(ctx context.Context, vals ...string) {
	s, ok := ctx.Value(secretsKey).(*secrets)
	if !ok {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, v := range vals {
		if v != "" {
			s.vals[v] = struct{}{}
		}
	}
}

// values returns the recorded secrets, longest first so a secret containing
// another is replaced whole rather than leaving a fragment.
func values(ctx context.Context) []string {
	s, ok := ctx.Value(secretsKey).(*secrets)
	if !ok {
		return nil
	}
	s.mu.Lock()
	vals := slices.Collect(maps.Keys(s.vals))
	s.mu.Unlock()
	slices.SortFunc(vals, func(a, b string) int { return cmp.Compare(len(b), len(a)) })
	return vals
}

// ScrubJSON replaces recorded secrets in an encoded JSON document. Each value is
// matched as is and as quoted by %q and %+q, since errors format values both
// ways, and each of those raw and JSON-encoded (as written by encoding/json,
// including HTML escaping).
func ScrubJSON(ctx context.Context, doc []byte) []byte {
	for _, v := range values(ctx) {
		for _, form := range []string{v, unquoted(strconv.Quote(v)), unquoted(strconv.QuoteToASCII(v))} {
			if enc, err := json.Marshal(form); err == nil {
				doc = bytes.ReplaceAll(doc, []byte(unquoted(string(enc))), []byte(Placeholder))
			}
			doc = bytes.ReplaceAll(doc, []byte(form), []byte(Placeholder))
		}
	}
	return doc
}

func unquoted(s string) string { return s[1 : len(s)-1] }

// JSONWriter wraps w so that each Write is scrubbed with ScrubJSON. It relies
// on json.Encoder writing each encoded value in a single Write.
func JSONWriter(ctx context.Context, w io.Writer) io.Writer {
	return jsonWriter{ctx: ctx, w: w}
}

type jsonWriter struct {
	ctx context.Context
	w   io.Writer
}

func (jw jsonWriter) Write(p []byte) (int, error) {
	if _, err := jw.w.Write(ScrubJSON(jw.ctx, p)); err != nil {
		return 0, err
	}
	return len(p), nil
}
