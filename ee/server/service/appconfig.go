package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

func (svc *Service) HostFeatures(ctx context.Context, host *fleet.Host) (*fleet.Features, error) {
	if host.TeamID != nil {
		features, err := svc.ds.TeamFeatures(ctx, *host.TeamID)
		if err != nil {
			return nil, err
		}
		return features, nil
	}

	appConfig, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return nil, err
	}
	return &appConfig.Features, nil
}
