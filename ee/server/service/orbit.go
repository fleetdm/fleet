package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
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

	// get the script's details
	script, err := svc.ds.GetHostScriptExecutionResult(ctx, execID)
	if err != nil {
		return nil, err
	}
	// ensure it cannot get access to a different host's script
	if script.HostID != host.ID {
		return nil, ctxerr.Wrap(ctx, notFoundError{}, "no script found for this host")
	}
	return script, nil
}

func (svc *Service) SaveHostScriptResult(ctx context.Context, result *fleet.HostScriptResultPayload) error {
	// this is not a user-authenticated endpoint
	svc.authz.SkipAuthorization(ctx)

	host, ok := hostctx.FromContext(ctx)
	if !ok {
		return fleet.OrbitError{Message: "internal error: missing host from request context"}
	}

	// always use the authenticated host's ID as host_id
	result.HostID = host.ID
	return svc.ds.SetHostScriptExecutionResult(ctx, result)
}
