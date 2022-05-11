package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/contexts/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

func (svc *Service) GetHost(ctx context.Context, id uint) (*fleet.HostDetail, error) {
	alreadyAuthd := svc.authz.IsAuthenticatedWith(ctx, authz.AuthnDeviceToken)
	if !alreadyAuthd {
		// First ensure the user has access to list hosts, then check the specific
		// host once team_id is loaded.
		if err := svc.authz.Authorize(ctx, &fleet.Host{}, fleet.ActionList); err != nil {
			return nil, err
		}
	}

	host, err := svc.ds.Host(ctx, id, false)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get host")
	}

	if !alreadyAuthd {
		// Authorize again with team loaded now that we have team_id
		if err := svc.authz.Authorize(ctx, host, fleet.ActionRead); err != nil {
			return nil, err
		}
	}

	hostDetails, err := svc.getHostDetails(ctx, host)
	if err != nil {
		return nil, err
	}

	return hostDetails, nil
}

func (svc *Service) getHostDetails(ctx context.Context, host *fleet.Host) (*fleet.HostDetail, error) {
	if err := svc.ds.LoadHostSoftware(ctx, host); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "load host software")
	}

	labels, err := svc.ds.ListLabelsForHost(ctx, host.ID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get labels for host")
	}

	packs, err := svc.ds.ListPacksForHost(ctx, host.ID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get packs for host")
	}

	policies, err := svc.ds.ListPoliciesForHost(ctx, host)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get policies for host")
	}

	return &fleet.HostDetail{Host: *host, Labels: labels, Packs: packs, Policies: policies}, nil
}

func (svc *Service) HostByIdentifier(ctx context.Context, identifier string) (*fleet.HostDetail, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Host{}, fleet.ActionList); err != nil {
		return nil, err
	}

	host, err := svc.ds.HostByIdentifier(ctx, identifier)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get host by identifier")
	}

	// Authorize again with team loaded now that we have team_id
	if err := svc.authz.Authorize(ctx, host, fleet.ActionRead); err != nil {
		return nil, err
	}

	return svc.getHostDetails(ctx, host)
}
