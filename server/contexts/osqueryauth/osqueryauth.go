// Package osqueryauth provides a context marker indicating that an osquery
// request has been authenticated by the HTTP-level pre-auth middleware via
// the Authorization: NodeKey header. Downstream code uses this to skip
// redundant node-key extraction from the request body.
package osqueryauth

import "context"

type key int

const preAuthedKey key = 0

type preAuthedMarker struct{}

// NewPreAuthedContext returns a ctx marked as pre-authenticated by the
// HTTP-level osquery pre-auth middleware.
func NewPreAuthedContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, preAuthedKey, preAuthedMarker{})
}

// IsPreAuthed reports whether the HTTP-level osquery pre-auth middleware
// successfully authenticated this request.
func IsPreAuthed(ctx context.Context) bool {
	_, ok := ctx.Value(preAuthedKey).(preAuthedMarker)
	return ok
}
