// Package nodeauth provides a context key for storing a host node-key
// authenticator so that request decoders can authenticate before reading
// potentially large request bodies.
package nodeauth

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

type key int

const nodeAuthKey key = 0

// NewContext stores a fleet.HostAuthenticator in ctx.
func NewContext(ctx context.Context, svc fleet.HostAuthenticator) context.Context {
	return context.WithValue(ctx, nodeAuthKey, svc)
}

// FromContext retrieves the fleet.HostAuthenticator from ctx, or nil if not set.
func FromContext(ctx context.Context) fleet.HostAuthenticator {
	svc, _ := ctx.Value(nodeAuthKey).(fleet.HostAuthenticator)
	return svc
}
