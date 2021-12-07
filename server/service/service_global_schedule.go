package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
)

func (svc *Service) ModifyGlobalScheduledQueries(
	ctx context.Context,
	id uint,
	query fleet.ScheduledQueryPayload,
) (*fleet.ScheduledQuery, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Pack{}, fleet.ActionWrite); err != nil {
		return nil, err
	}

	gp, err := svc.ds.EnsureGlobalPack(ctx)
	if err != nil {
		return nil, err
	}

	query.PackID = ptr.Uint(gp.ID)

	return svc.ModifyScheduledQuery(ctx, id, query)
}

func (svc *Service) DeleteGlobalScheduledQueries(ctx context.Context, id uint) error {
	if err := svc.authz.Authorize(ctx, &fleet.Pack{}, fleet.ActionWrite); err != nil {
		return err
	}

	return svc.DeleteScheduledQuery(ctx, id)
}
