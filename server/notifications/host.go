package notifications

import "context"

// This context carries a host ID rather than a host: server/service owns
// device authentication and resolves the *fleet.Host, a type this context
// can't depend on, so its middleware hands over just the ID.
type hostIDKey int

const authenticatedHostIDKey hostIDKey = 0

func NewHostContext(ctx context.Context, hostID uint) context.Context {
	return context.WithValue(ctx, authenticatedHostIDKey, hostID)
}

func HostIDFromContext(ctx context.Context) (uint, bool) {
	hostID, ok := ctx.Value(authenticatedHostIDKey).(uint)
	return hostID, ok
}
