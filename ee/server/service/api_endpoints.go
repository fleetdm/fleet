package service

import (
	"context"

	apiendpoints "github.com/fleetdm/fleet/v4/server/api_endpoints"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

func (svc *Service) ListAPIEndpoints(ctx context.Context) ([]fleet.APIEndpoint, error) {
	if err := svc.authz.Authorize(ctx, &fleet.APIEndpoint{}, fleet.ActionRead); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "authorize list API endpoints")
	}
	return apiendpoints.GetAPIEndpoints(), nil
}
