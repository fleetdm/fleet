// Package host enables setting and reading
// the current host from context
package host

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

type key int

const hostKey key = 0

// NewContext returns a new context carrying the current osquery host.
func NewContext(ctx context.Context, host fleet.Host) context.Context {
	return context.WithValue(ctx, hostKey, host)
}

// FromContext extracts the osquery host from context if present.
func FromContext(ctx context.Context) (fleet.Host, bool) {
	host, ok := ctx.Value(hostKey).(fleet.Host)
	return host, ok
}
