package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

// HostFeatures retrieves the features enabled for a given host.
//
// - If the host belongs to a team, it will use the team's features. When a team
// is created the features are mixed with the global config features, but from
// that point on they are independent of whatever the global config is.
// - If the host doesn't belong to a team, the app config features are used.
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
