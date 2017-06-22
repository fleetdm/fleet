// Package host enables setting and reading
// the current host from context
package host

import (
	"context"

	"github.com/kolide/fleet/server/kolide"
)

type key int

const hostKey key = 0

// NewContext returns a new context carrying the current osquery host.
func NewContext(ctx context.Context, host kolide.Host) context.Context {
	return context.WithValue(ctx, hostKey, host)
}

// FromContext extracts the osquery host from context if present.
func FromContext(ctx context.Context) (kolide.Host, bool) {
	host, ok := ctx.Value(hostKey).(kolide.Host)
	return host, ok
}
