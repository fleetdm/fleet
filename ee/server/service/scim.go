package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

func (svc *Service) ScimDetails(ctx context.Context) (fleet.ScimDetails, error) {
	err := svc.authz.Authorize(ctx, &fleet.ScimUser{}, fleet.ActionRead)
	if err != nil {
		return fleet.ScimDetails{}, err
	}

	request, err := svc.ds.ScimLastRequest(ctx)
	if err != nil {
		return fleet.ScimDetails{}, ctxerr.Wrap(ctx, err, "scim details")
	}
	return fleet.ScimDetails{
		LastRequest: request,
	}, nil
}
