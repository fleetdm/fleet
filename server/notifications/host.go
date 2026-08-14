package notifications

import "context"

// hostIDKey is the context key type for the authenticated host ID. The
// notifications bounded context never sees a *fleet.Host: the device auth
// middleware injected from server/service resolves the host and stores just
// its ID here, since fleet.Host is defined outside this bounded context.
type hostIDKey int

const authenticatedHostIDKey hostIDKey = 0

// NewHostContext returns a new context carrying the authenticated host's ID.
func NewHostContext(ctx context.Context, hostID uint) context.Context {
	return context.WithValue(ctx, authenticatedHostIDKey, hostID)
}

// HostIDFromContext extracts the authenticated host's ID from context if present.
func HostIDFromContext(ctx context.Context) (uint, bool) {
	hostID, ok := ctx.Value(authenticatedHostIDKey).(uint)
	return hostID, ok
}
