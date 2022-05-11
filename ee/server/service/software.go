package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

func (svc Service) ListSoftware(ctx context.Context, opt fleet.SoftwareListOptions) ([]fleet.Software, error) {
	if err := svc.authz.Authorize(ctx, &fleet.AuthzSoftwareInventory{
		TeamID: opt.TeamID,
	}, fleet.ActionRead); err != nil {
		return nil, err
	}

	// default sort order to hosts_count descending
	if opt.OrderKey == "" {
		opt.OrderKey = "hosts_count"
		opt.OrderDirection = fleet.OrderDescending
	}
	opt.WithHostCounts = true

	softwares, err := svc.ds.ListSoftware(ctx, opt)
	if err != nil {
		return nil, err
	}

	return softwares, nil
}

func (svc *Service) SoftwareByID(ctx context.Context, id uint) (*fleet.Software, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Host{}, fleet.ActionList); err != nil {
		return nil, err
	}

	return svc.ds.SoftwareByID(ctx, id)
}
