package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

func (mw validationMiddleware) NewAppConfig(ctx context.Context, payload fleet.AppConfig) (*fleet.AppConfig, error) {
	invalid := &fleet.InvalidArgumentError{}
	if payload.ServerSettings.ServerURL == "" {
		invalid.Append("server_url", "missing required argument")
	}
	// explicitly removed check for "https" scheme
	if invalid.HasErrors() {
		return nil, ctxerr.Wrap(ctx, invalid)
	}
	return mw.Service.NewAppConfig(ctx, payload)
}
