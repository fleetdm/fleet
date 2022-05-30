package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/contexts/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

func (svc *Service) ListDevicePolicies(ctx context.Context, host *fleet.Host) ([]*fleet.HostPolicy, error) {
	if !svc.authz.IsAuthenticatedWith(ctx, authz.AuthnDeviceToken) {
		if err := svc.authz.Authorize(ctx, &fleet.Host{}, fleet.ActionList); err != nil {
			return nil, err
		}

		host, err := svc.ds.HostLite(ctx, host.ID)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "find host for device policies")
		}

		if err := svc.authz.Authorize(ctx, host, fleet.ActionRead); err != nil {
			return nil, err
		}
	}

	return svc.ds.ListPoliciesForHost(ctx, host)
}
