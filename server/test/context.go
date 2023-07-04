package test

import (
	"context"

	authz_ctx "github.com/fleetdm/fleet/v4/server/contexts/authz"
	hostctx "github.com/fleetdm/fleet/v4/server/contexts/host"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

// UserContext returns a new context with the provided user as the viewer.
func UserContext(ctx context.Context, user *fleet.User) context.Context {
	return viewer.NewContext(ctx, viewer.Viewer{User: user})
}

// HostContext returns a new context with the provided host as the
// device-authenticated host.
func HostContext(ctx context.Context, host *fleet.Host) context.Context {
	authzCtx := &authz_ctx.AuthorizationContext{}
	ctx = authz_ctx.NewContext(ctx, authzCtx)
	ctx = hostctx.NewContext(ctx, host)
	if ac, ok := authz_ctx.FromContext(ctx); ok {
		ac.SetAuthnMethod(authz_ctx.AuthnDeviceToken)
	}
	return ctx
}
