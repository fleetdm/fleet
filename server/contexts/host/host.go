// Package host enables setting and reading
// the current host from context
package host

import (
	"context"
)

type key int

const hostKey key = 0

// NewContext returns a new context carrying the current osquery host.
func NewContext(ctx context.Context, hostID uint) context.Context {
	return context.WithValue(ctx, hostKey, hostID)
}

// FromContext extracts the osquery host from context if present.
func FromContext(ctx context.Context) (uint, bool) {
	host, ok := ctx.Value(hostKey).(uint)
	return host, ok
}
