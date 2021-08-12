package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/pkg/errors"
)

func (svc Service) ListHosts(ctx context.Context, opt fleet.HostListOptions) ([]*fleet.Host, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Host{}, fleet.ActionList); err != nil {
		return nil, err
	}

	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, fleet.ErrNoContext
	}
	filter := fleet.TeamFilter{User: vc.User, IncludeObserver: true}

	return svc.ds.ListHosts(filter, opt)
}

func (svc Service) GetHost(ctx context.Context, id uint) (*fleet.HostDetail, error) {
	// First ensure the user has access to list hosts, then check the specific
	// host once team_id is loaded.
	if err := svc.authz.Authorize(ctx, &fleet.Host{}, fleet.ActionList); err != nil {
		return nil, err
	}

	host, err := svc.ds.Host(id)
	if err != nil {
		return nil, errors.Wrap(err, "get host")
	}

	// Authorize again with team loaded now that we have team_id
	if err := svc.authz.Authorize(ctx, host, fleet.ActionRead); err != nil {
		return nil, err
	}

	return svc.getHostDetails(ctx, host)
}

func (svc Service) HostByIdentifier(ctx context.Context, identifier string) (*fleet.HostDetail, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Host{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	host, err := svc.ds.HostByIdentifier(identifier)
	if err != nil {
		return nil, errors.Wrap(err, "get host by identifier")
	}

	// Authorize again with team loaded now that we have team_id
	if err := svc.authz.Authorize(ctx, host, fleet.ActionRead); err != nil {
		return nil, err
	}

	return svc.getHostDetails(ctx, host)
}

func (svc Service) getHostDetails(ctx context.Context, host *fleet.Host) (*fleet.HostDetail, error) {
	if err := svc.ds.LoadHostSoftware(host); err != nil {
		return nil, errors.Wrap(err, "load host software")
	}

	labels, err := svc.ds.ListLabelsForHost(host.ID)
	if err != nil {
		return nil, errors.Wrap(err, "get labels for host")
	}

	packs, err := svc.ds.ListPacksForHost(host.ID)
	if err != nil {
		return nil, errors.Wrap(err, "get packs for host")
	}

	return &fleet.HostDetail{Host: *host, Labels: labels, Packs: packs}, nil
}

func (svc Service) GetHostSummary(ctx context.Context) (*fleet.HostSummary, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Host{}, fleet.ActionList); err != nil {
		return nil, err
	}
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, fleet.ErrNoContext
	}
	filter := fleet.TeamFilter{User: vc.User, IncludeObserver: true}

	online, offline, mia, new, err := svc.ds.GenerateHostStatusStatistics(filter, svc.clock.Now())
	if err != nil {
		return nil, err
	}
	return &fleet.HostSummary{
		OnlineCount:  online,
		OfflineCount: offline,
		MIACount:     mia,
		NewCount:     new,
	}, nil
}

func (svc Service) DeleteHost(ctx context.Context, id uint) error {
	if err := svc.authz.Authorize(ctx, &fleet.Host{}, fleet.ActionWrite); err != nil {
		return err
	}

	host, err := svc.ds.Host(id)
	if err != nil {
		return errors.Wrap(err, "get host for delete")
	}

	// Authorize again with team loaded now that we have team_id
	if err := svc.authz.Authorize(ctx, host, fleet.ActionWrite); err != nil {
		return err
	}

	return svc.ds.DeleteHost(id)
}

func (svc *Service) FlushSeenHosts(ctx context.Context) error {
	// No authorization check because this is used only internally.

	hostIDs := svc.seenHostSet.getAndClearHostIDs()
	return svc.ds.MarkHostsSeen(hostIDs, svc.clock.Now())
}

func (svc Service) AddHostsToTeam(ctx context.Context, teamID *uint, hostIDs []uint) error {
	// This is currently treated as a "team write". If we ever give users
	// besides global admins permissions to modify team hosts, we will need to
	// check that the user has permissions for both the source and destination
	// teams.
	if err := svc.authz.Authorize(ctx, &fleet.Host{}, fleet.ActionWrite); err != nil {
		return err
	}

	return svc.ds.AddHostsToTeam(teamID, hostIDs)
}

func (svc Service) AddHostsToTeamByFilter(ctx context.Context, teamID *uint, opt fleet.HostListOptions, lid *uint) error {
	// This is currently treated as a "team write". If we ever give users
	// besides global admins permissions to modify team hosts, we will need to
	// check that the user has permissions for both the source and destination
	// teams.
	if err := svc.authz.Authorize(ctx, &fleet.Host{}, fleet.ActionWrite); err != nil {
		return err
	}
	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return fleet.ErrNoContext
	}
	filter := fleet.TeamFilter{User: vc.User, IncludeObserver: true}

	if opt.StatusFilter != "" && lid != nil {
		return fleet.NewInvalidArgumentError("status", "may not be provided with label_id")
	}

	opt.PerPage = fleet.PerPageUnlimited

	// Load hosts, either from label if provided or from all hosts.
	var hosts []*fleet.Host
	var err error
	if lid != nil {
		hosts, err = svc.ds.ListHostsInLabel(filter, *lid, opt)
	} else {
		hosts, err = svc.ds.ListHosts(filter, opt)
	}
	if err != nil {
		return err
	}

	if len(hosts) == 0 {
		return nil
	}

	hostIDs := make([]uint, 0, len(hosts))
	for _, h := range hosts {
		hostIDs = append(hostIDs, h.ID)
	}

	// Apply the team to the selected hosts.
	return svc.ds.AddHostsToTeam(teamID, hostIDs)
}

func (svc *Service) RefetchHost(ctx context.Context, id uint) error {
	host, err := svc.ds.Host(id)
	if err != nil {
		return errors.Wrap(err, "find host for refetch")
	}

	if err := svc.authz.Authorize(ctx, host, fleet.ActionRead); err != nil {
		return err
	}

	host.RefetchRequested = true
	if err := svc.ds.SaveHost(host); err != nil {
		return errors.Wrap(err, "save host")
	}

	return nil
}
