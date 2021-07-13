package service

import (
	"context"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

func (svc *Service) ListActivities(ctx context.Context, opt fleet.ListOptions) ([]*fleet.Activity, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Activity{}, fleet.ActionRead); err != nil {
		return nil, err
	}
	return svc.ds.ListActivities(opt)
}
