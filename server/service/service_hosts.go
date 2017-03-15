package service

import (
	"context"

	"github.com/kolide/kolide/server/kolide"
)

func (svc service) ListHosts(ctx context.Context, opt kolide.ListOptions) ([]*kolide.Host, error) {
	return svc.ds.ListHosts(opt)
}

func (svc service) GetHost(ctx context.Context, id uint) (*kolide.Host, error) {
	return svc.ds.Host(id)
}

func (svc service) GetHostSummary(ctx context.Context) (*kolide.HostSummary, error) {
	onlineInterval, err := svc.ExpectedCheckinInterval(ctx)
	if err != nil {
		return nil, err
	}
	online, offline, mia, new, err := svc.ds.GenerateHostStatusStatistics(svc.clock.Now(), onlineInterval)
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
