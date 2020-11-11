package service

import (
	"context"

	"github.com/fleetdm/fleet/server/kolide"
)

func (svc service) ListHosts(ctx context.Context, opt kolide.HostListOptions) ([]*kolide.Host, error) {
	return svc.ds.ListHosts(opt)
}

func (svc service) GetHost(ctx context.Context, id uint) (*kolide.Host, error) {
	return svc.ds.Host(id)
}

func (svc service) HostByIdentifier(ctx context.Context, identifier string) (*kolide.Host, error) {
	return svc.ds.HostByIdentifier(identifier)
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
