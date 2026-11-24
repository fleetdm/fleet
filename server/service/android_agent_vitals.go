package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	hostctx "github.com/fleetdm/fleet/v4/server/contexts/host"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

type setAndroidAgentVitalsRequest struct {
	fleet.AndroidAgentVitals
}

type setAndroidAgentVitalsResponse struct {
	Err error `json:"error,omitempty"`
}

func (r setAndroidAgentVitalsResponse) Error() error { return r.Err }

func setAndroidAgentVitalsEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*setAndroidAgentVitalsRequest)
	if err := svc.SetAndroidAgentVitals(ctx, req.AndroidAgentVitals); err != nil {
		return setAndroidAgentVitalsResponse{Err: err}, nil
	}
	return setAndroidAgentVitalsResponse{}, nil
}

func (svc *Service) SetAndroidAgentVitals(ctx context.Context, vitals fleet.AndroidAgentVitals) error {
	// skipauth: This endpoint uses node key authentication instead of user authentication.
	svc.authz.SkipAuthorization(ctx)

	host, ok := hostctx.FromContext(ctx)
	if !ok {
		return ctxerr.New(ctx, "missing host from request context")
	}
	// Orbit node keys are also issued to fleetd on desktop platforms; only the
	// Android agent may write Android vitals.
	if host.Platform != "android" {
		return fleet.NewPermissionError("only Android hosts can report Android agent vitals")
	}
	if vitals.Roaming == nil {
		return fleet.NewInvalidArgumentError("roaming", "required")
	}

	return svc.ds.SetHostMDMAndroidRoaming(ctx, host.UUID, *vitals.Roaming)
}
