package service

import (
	"context"

	"github.com/fleetdm/fleet/server/kolide"
	"github.com/pkg/errors"
)

func (svc Service) ListHosts(ctx context.Context, opt kolide.HostListOptions) ([]*kolide.Host, error) {
	return svc.ds.ListHosts(opt)
}

func (svc Service) GetHost(ctx context.Context, id uint) (*kolide.HostDetail, error) {
	host, err := svc.ds.Host(id)
	if err != nil {
		return nil, errors.Wrap(err, "get host")
	}

	return svc.getHostDetails(ctx, host)
}

func (svc Service) HostByIdentifier(ctx context.Context, identifier string) (*kolide.HostDetail, error) {
	host, err := svc.ds.HostByIdentifier(identifier)
	if err != nil {
		return nil, errors.Wrap(err, "get host by identifier")
	}

	return svc.getHostDetails(ctx, host)
}

func (svc Service) getHostDetails(ctx context.Context, host *kolide.Host) (*kolide.HostDetail, error) {
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

	return &kolide.HostDetail{Host: *host, Labels: labels, Packs: packs}, nil
}

func (svc Service) GetHostSummary(ctx context.Context) (*kolide.HostSummary, error) {
	online, offline, mia, new, err := svc.ds.GenerateHostStatusStatistics(svc.clock.Now())
	if err != nil {
		return nil, err
	}
	return &kolide.HostSummary{
		OnlineCount:  online,
		OfflineCount: offline,
		MIACount:     mia,
		NewCount:     new,
	}, nil
}

func (svc Service) DeleteHost(ctx context.Context, id uint) error {
	return svc.ds.DeleteHost(id)
}

func (svc *Service) FlushSeenHosts(ctx context.Context) error {
	hostIDs := svc.seenHostSet.getAndClearHostIDs()
	return svc.ds.MarkHostsSeen(hostIDs, svc.clock.Now())
}

func (svc Service) AddHostsToTeam(ctx context.Context, teamID *uint, hostIDs []uint) error {
	return svc.ds.AddHostsToTeam(teamID, hostIDs)
}

func (svc Service) AddHostsToTeamByFilter(ctx context.Context, teamID *uint, opt kolide.HostListOptions, lid *uint) error {
	if opt.StatusFilter != "" && lid != nil {
		return kolide.NewInvalidArgumentError("status", "may not be provided with label_id")
	}

	opt.PerPage = kolide.PerPageUnlimited

	// Load hosts, either from label if provided or from all hosts.
	var hosts []*kolide.Host
	var err error
	if lid != nil {
		hosts, err = svc.ds.ListHostsInLabel(*lid, opt)
	} else {
		hosts, err = svc.ds.ListHosts(opt)
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

	host.RefetchRequested = true
	if err := svc.ds.SaveHost(host); err != nil {
		return errors.Wrap(err, "save host")
	}

	return nil
}
