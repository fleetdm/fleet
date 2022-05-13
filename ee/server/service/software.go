package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

func (svc Service) ListSoftware(ctx context.Context, opts fleet.SoftwareListOptions) ([]fleet.Software, error) {
	opts.IncludeCVEScores = true
	return svc.Service.ListSoftware(ctx, opts)
}

func (svc *Service) SoftwareByID(ctx context.Context, id uint, includeCVEScores bool) (*fleet.Software, error) {
	return svc.Service.SoftwareByID(ctx, id, true)
}
