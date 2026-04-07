package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

func (svc *Service) ListAPIEndpoints(ctx context.Context, opts fleet.ListOptions) ([]fleet.APIEndpoint, *fleet.PaginationMetadata, int, error) {
	if err := svc.authz.Authorize(ctx, &fleet.User{}, fleet.ActionWrite); err != nil {
		return nil, nil, 0, ctxerr.Wrap(ctx, err, "authorize list API endpoints")
	}

	opts.IncludeMetadata = true
	return svc.ds.ListAPIEndpoints(ctx, opts)
}
