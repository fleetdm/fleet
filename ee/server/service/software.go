package service

import (
	"context"

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
