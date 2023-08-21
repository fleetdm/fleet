package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

type getScriptResultRequest struct {
	ID uint `url:"id"`
}

type getScriptResultResponse struct {
	*fleet.ScriptResult
	Err error `json:"error,omitempty"`
}

func (r getScriptResultResponse) error() error { return r.Err }

func getScriptResultEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*getScriptResultRequest)
	scriptResults, err := svc.GetScriptResult(ctx, req.ID)
	if err != nil {
		return getScriptResultResponse{Err: err}, nil
	}

	return getScriptResultResponse{
		ScriptResult: scriptResults,
	}, nil
}

func (svc *Service) GetScriptResult(ctx context.Context, scriptID uint) (*fleet.ScriptResult, error) {
	if err := svc.authz.Authorize(ctx, &fleet.ScriptResult{}, fleet.ActionList); err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}

	scriptResult, err := svc.ds.GetScriptResult(ctx, scriptID)
	if err != nil {
		return nil, err
	}

	return scriptResult, nil
}
