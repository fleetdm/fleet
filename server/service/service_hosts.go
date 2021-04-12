package service

import (
	"context"

	"github.com/fleetdm/fleet/server/kolide"
	"github.com/pkg/errors"
)

func (svc service) ListHosts(ctx context.Context, opt kolide.HostListOptions) ([]*kolide.Host, error) {
	return svc.ds.ListHosts(opt)
}

func (svc service) GetHost(ctx context.Context, id uint) (*kolide.HostDetail, error) {
	host, err := svc.ds.Host(id)
	if err != nil {
		return nil, errors.Wrap(err, "get host")
	}

	return svc.getHostDetails(ctx, host)
}

func (svc service) HostByIdentifier(ctx context.Context, identifier string) (*kolide.HostDetail, error) {
	host, err := svc.ds.HostByIdentifier(identifier)
	if err != nil {
		return nil, errors.Wrap(err, "get host by identifier")
	}

	return svc.getHostDetails(ctx, host)
}

func (svc service) getHostDetails(ctx context.Context, host *kolide.Host) (*kolide.HostDetail, error) {
	labels, err := svc.ds.ListLabelsForHost(host.ID)
	if err != nil {
		return nil, errors.Wrap(err, "get labels for host")
	}

	packPtrs, err := svc.ds.ListPacksForHost(host.ID)
	if err != nil {
		return nil, errors.Wrap(err, "get packs for host")
	}

	// TODO refactor List* APIs to be consistent so we don't have to do this
	// transformation
	packs := make([]kolide.Pack, 0, len(packPtrs))
	for _, p := range packPtrs {
		packs = append(packs, *p)
	}

	return &kolide.HostDetail{Host: *host, Labels: labels, Packs: packs}, nil
}

func (svc service) GetHostSummary(ctx context.Context) (*kolide.HostSummary, error) {
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

func (svc service) DeleteHost(ctx context.Context, id uint) error {
	return svc.ds.DeleteHost(id)
}

func (svc *service) FlushSeenHosts(ctx context.Context) error {
	hostIDs := svc.seenHostSet.getAndClearHostIDs()
	return svc.ds.MarkHostsSeen(hostIDs, svc.clock.Now())
}
