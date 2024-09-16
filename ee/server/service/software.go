package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

func (svc *Service) ListSoftware(ctx context.Context, opts fleet.SoftwareListOptions) ([]fleet.Software, *fleet.PaginationMetadata, error) {
	// reuse ListSoftware, but include cve scores in premium version
	opts.IncludeCVEScores = true
	return svc.Service.ListSoftware(ctx, opts)
}

func (svc *Service) SoftwareByID(ctx context.Context, id uint, teamID *uint, _ bool) (*fleet.Software, error) {
	// reuse SoftwareByID, but include cve scores in premium version
	return svc.Service.SoftwareByID(ctx, id, teamID, true)
}

func (svc *Service) ListFleetMaintainedApps(ctx context.Context, teamID uint, opts fleet.ListOptions) ([]fleet.FleetMaintainedAppAvailable, *fleet.PaginationMetadata, error) {
	svc.authz.SkipAuthorization(ctx)

	avail, meta, err := svc.ds.ListAvailableFleetMaintainedApps(ctx, teamID, &opts)
	if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "listing available fleet managed apps")
	}

	return avail, meta, nil
}
