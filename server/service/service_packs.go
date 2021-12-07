package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

func (svc *Service) GetPackSpec(ctx context.Context, name string) (*fleet.PackSpec, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Pack{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	return svc.ds.GetPackSpec(ctx, name)
}

func (svc *Service) ListPacksForHost(ctx context.Context, hid uint) ([]*fleet.Pack, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Pack{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	return svc.ds.ListPacksForHost(ctx, hid)
}
