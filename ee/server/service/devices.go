package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

func (svc *Service) ListDevicePolicies(ctx context.Context, host *fleet.Host) ([]*fleet.HostPolicy, error) {
	return svc.ds.ListPoliciesForHost(ctx, host)
}
