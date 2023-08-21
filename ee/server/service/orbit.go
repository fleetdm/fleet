package service

import (
	"context"

	hostctx "github.com/fleetdm/fleet/v4/server/contexts/host"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

func (svc *Service) GetHostScript(ctx context.Context, execID string) (*fleet.HostScriptResult, error) {
	// this is not a user-authenticated endpoint
	svc.authz.SkipAuthorization(ctx)

	host, ok := hostctx.FromContext(ctx)
	if !ok {
		return nil, fleet.OrbitError{Message: "internal error: missing host from request context"}
	}

	// TODO(mna): implement...
	_ = host
	return nil, nil
}

func (svc *Service) SaveHostScriptResult(ctx context.Context, result *fleet.HostScriptResult) error {
	// this is not a user-authenticated endpoint
	svc.authz.SkipAuthorization(ctx)

	host, ok := hostctx.FromContext(ctx)
	if !ok {
		return fleet.OrbitError{Message: "internal error: missing host from request context"}
	}

	// TODO(mna): implement...
	_ = host
	return nil
}
