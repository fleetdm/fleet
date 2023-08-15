package service

import (
	"context"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

func (svc *Service) GetHost(ctx context.Context, id uint, opts fleet.HostDetailOptions) (*fleet.HostDetail, error) {
	// reuse GetHost, but include premium details
	opts.IncludeCVEScores = true
	opts.IncludePolicies = true
	return svc.Service.GetHost(ctx, id, opts)
}

func (svc *Service) HostByIdentifier(ctx context.Context, identifier string, opts fleet.HostDetailOptions) (*fleet.HostDetail, error) {
	// reuse HostByIdentifier, but include premium options
	opts.IncludeCVEScores = true
	opts.IncludePolicies = true
	return svc.Service.HostByIdentifier(ctx, identifier, opts)
}

func (svc *Service) RunHostScript(ctx context.Context, request *fleet.HostScriptRequestPayload, waitForResult time.Duration) (*fleet.HostScriptResult, error) {
	// must load the host (lite is enough, just for the team) to authorize
	// with the proper team id. We cannot first authorize if the user can list
	// hosts, because the user could have a write-only role (e.g. gitops).
	host, err := svc.ds.HostLite(ctx, request.HostID)
	if err != nil {
		// if error is because the host does not exist, check first if the user
		// had access to run a script (to prevent leaking valid host ids).
		if fleet.IsNotFound(err) {
			if err := svc.authz.Authorize(ctx, &fleet.HostScriptResult{}, fleet.ActionWrite); err != nil {
				return nil, err
			}
		}
		svc.authz.SkipAuthorization(ctx)
		return nil, ctxerr.Wrap(ctx, err, "get host lite")
	}
	if err := svc.authz.Authorize(ctx, &fleet.HostScriptResult{TeamID: host.TeamID}, fleet.ActionWrite); err != nil {
		return nil, err
	}

	if request.ScriptContents == "" {
		return nil, fleet.NewInvalidArgumentError("script_contents", "a script to execute is required")
	}

	// create the script execution request
	script, err := svc.ds.NewHostScriptExecutionRequest(ctx, request)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "create script execution request")
	}
	if waitForResult <= 0 {
		// async execution, return
		return script, nil
	}

	ctx, cancel := context.WithTimeout(ctx, waitForResult)
	defer cancel()

	// check quickly for a result (1/100 of the wait), then every 1/10 of the
	// wait.
	initialDelay, checkInterval := waitForResult/100, waitForResult/10
	after := time.NewTimer(initialDelay)
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-after.C:
			result, err := svc.ds.GetHostScriptExecutionResult(ctx, script.ExecutionID)
			if err != nil {
				return nil, ctxerr.Wrap(ctx, err, "get script execution result")
			}
			if result.ExitCode.Valid {
				// a result was received from the host, return
				return result, nil
			}
			after.Reset(checkInterval)
		}
	}
}
