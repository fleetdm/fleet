package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

func (svc Service) GetHost(ctx context.Context, id uint) (*fleet.HostDetail, error) {
	// First ensure the user has access to list hosts, then check the specific
	// host once team_id is loaded.
	if err := svc.authz.Authorize(ctx, &fleet.Host{}, fleet.ActionList); err != nil {
		return nil, err
	}

	host, err := svc.ds.Host(ctx, id, false)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get host")
	}

	// Authorize again with team loaded now that we have team_id
	if err := svc.authz.Authorize(ctx, host, fleet.ActionRead); err != nil {
		return nil, err
	}

	return svc.getHostDetails(ctx, host)
}

func (svc Service) HostByIdentifier(ctx context.Context, identifier string) (*fleet.HostDetail, error) {
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

func (svc Service) getHostDetails(ctx context.Context, host *fleet.Host) (*fleet.HostDetail, error) {
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

func (svc Service) DeleteHost(ctx context.Context, id uint) error {
	if err := svc.authz.Authorize(ctx, &fleet.Host{}, fleet.ActionList); err != nil {
		return err
	}

	host, err := svc.ds.Host(ctx, id, false)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get host for delete")
	}

	// Authorize again with team loaded now that we have team_id
	if err := svc.authz.Authorize(ctx, host, fleet.ActionWrite); err != nil {
		return err
	}

	return svc.ds.DeleteHost(ctx, id)
}

func (svc *Service) FlushSeenHosts(ctx context.Context) error {
	// No authorization check because this is used only internally.
	hostIDs := svc.seenHostSet.getAndClearHostIDs()
	return svc.ds.MarkHostsSeen(ctx, hostIDs, svc.clock.Now())
}

func (svc Service) AddHostsToTeam(ctx context.Context, teamID *uint, hostIDs []uint) error {
	// This is currently treated as a "team write". If we ever give users
	// besides global admins permissions to modify team hosts, we will need to
	// check that the user has permissions for both the source and destination
	// teams.
	if err := svc.authz.Authorize(ctx, &fleet.Host{TeamID: teamID}, fleet.ActionWrite); err != nil {
		return err
	}

	return svc.ds.AddHostsToTeam(ctx, teamID, hostIDs)
}

func (svc Service) AddHostsToTeamByFilter(ctx context.Context, teamID *uint, opt fleet.HostListOptions, lid *uint) error {
	// This is currently treated as a "team write". If we ever give users
	// besides global admins permissions to modify team hosts, we will need to
	// check that the user has permissions for both the source and destination
	// teams.
	if err := svc.authz.Authorize(ctx, &fleet.Host{TeamID: teamID}, fleet.ActionWrite); err != nil {
		return err
	}
	hostIDs, err := svc.hostIDsFromFilters(ctx, opt, lid)
	if err != nil {
		return err
	}
	if len(hostIDs) == 0 {
		return nil
	}

	// Apply the team to the selected hosts.
	return svc.ds.AddHostsToTeam(ctx, teamID, hostIDs)
}

func (svc Service) hostIDsFromFilters(ctx context.Context, opt fleet.HostListOptions, lid *uint) ([]uint, error) {
	filter, err := processHostFilters(ctx, opt, lid)
	if err != nil {
		return nil, err
	}

	// Load hosts, either from label if provided or from all hosts.
	var hosts []*fleet.Host
	if lid != nil {
		hosts, err = svc.ds.ListHostsInLabel(ctx, filter, *lid, opt)
	} else {
		hosts, err = svc.ds.ListHosts(ctx, filter, opt)
	}
	if err != nil {
		return nil, err
	}

	if len(hosts) == 0 {
		return nil, nil
	}

	hostIDs := make([]uint, 0, len(hosts))
	for _, h := range hosts {
		hostIDs = append(hostIDs, h.ID)
	}
	return hostIDs, nil
}

func (svc Service) countHostFromFilters(ctx context.Context, labelID *uint, opt fleet.HostListOptions) (int, error) {
	filter, err := processHostFilters(ctx, opt, nil)
	if err != nil {
		return 0, err
	}

	var count int
	if labelID != nil {
		count, err = svc.ds.CountHostsInLabel(ctx, filter, *labelID, opt)
	} else {
		count, err = svc.ds.CountHosts(ctx, filter, opt)
	}
	if err != nil {
		return 0, err
	}

	return count, nil
}

func processHostFilters(ctx context.Context, opt fleet.HostListOptions, lid *uint) (fleet.TeamFilter, error) {
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return fleet.TeamFilter{}, fleet.ErrNoContext
	}
	filter := fleet.TeamFilter{User: vc.User, IncludeObserver: true}

	if opt.StatusFilter != "" && lid != nil {
		return fleet.TeamFilter{}, fleet.NewInvalidArgumentError("status", "may not be provided with label_id")
	}

	opt.PerPage = fleet.PerPageUnlimited
	return filter, nil
}

func (svc *Service) RefetchHost(ctx context.Context, id uint) error {
	if err := svc.authz.Authorize(ctx, &fleet.Host{}, fleet.ActionList); err != nil {
		return err
	}

	host, err := svc.ds.Host(ctx, id, false)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "find host for refetch")
	}

	if err := svc.authz.Authorize(ctx, host, fleet.ActionRead); err != nil {
		return err
	}

	host.RefetchRequested = true
	if err := svc.ds.SaveHost(ctx, host); err != nil {
		return ctxerr.Wrap(ctx, err, "save host")
	}

	return nil
}
