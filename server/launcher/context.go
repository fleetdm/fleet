package launcher

import (
	newcontext "context"

	"github.com/kolide/fleet/server/contexts/host"
	"github.com/kolide/fleet/server/kolide"
	old "golang.org/x/net/context"
)

type contextKey int

const hostKey contextKey = 0

// newCtx is used to map the old golang.com/net/context which we are forced to use
// because our generated gRPC code uses it, to the new stdlib context, which is used
// by the Fleet application.
func newCtx(ctx old.Context) newcontext.Context {
	if h, ok := ctx.Value(hostKey).(kolide.Host); ok {
		return host.NewContext(newcontext.Background(), h)
	}
	return newcontext.Background()
}

// withHost creates a golang.org/x/net/context containing a host
func withHost(ctx old.Context, h kolide.Host) old.Context {
	return old.WithValue(ctx, hostKey, h)
}

func hostFromContext(ctx old.Context) (kolide.Host, bool) {
	h, ok := ctx.Value(hostKey).(kolide.Host)
	return h, ok
}
