// Package osqueryauth provides a context marker indicating that an osquery
// request has been authenticated by the HTTP-level pre-auth middleware via
// the Authorization: NodeKey header. Downstream code uses this to skip
// redundant node-key extraction from the request body.
package osqueryauth

import "context"

type key int

const (
	preAuthedKey key = iota
	debugKey
)

type preAuthedMarker struct{}

type debugMarker struct{}

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

// NewDebugContext marks ctx as belonging to a host with debug logging
// enabled. The HTTP pre-auth middleware sets this when AuthenticateHost
// reports the debug flag, so the endpoint-layer authenticatedHost
// passthrough can apply the same request/response debug logging that the
// legacy body-auth path applies.
func NewDebugContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, debugKey, debugMarker{})
}

// IsDebug reports whether the request was authenticated for a host with
// debug logging enabled.
func IsDebug(ctx context.Context) bool {
	_, ok := ctx.Value(debugKey).(debugMarker)
	return ok
}
