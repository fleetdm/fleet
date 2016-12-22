package service

import (
	"github.com/kolide/kolide-ose/server/kolide"
	"golang.org/x/net/context"
)

func (svc service) ListHosts(ctx context.Context, opt kolide.ListOptions) ([]*kolide.Host, error) {
	return svc.ds.ListHosts(opt)
}

func (svc service) GetHost(ctx context.Context, id uint) (*kolide.Host, error) {
	return svc.ds.Host(id)
}

func (svc service) HostStatus(ctx context.Context, host kolide.Host) string {
	switch {
	case host.UpdatedAt.Add(MIADuration).Before(svc.clock.Now()):
		return StatusMIA
	case host.UpdatedAt.Add(OfflineDuration).Before(svc.clock.Now()):
		return StatusOffline
	default:
		return StatusOnline
	}
}

func (svc service) DeleteHost(ctx context.Context, id uint) error {
	host, err := svc.ds.Host(id)
	if err != nil {
		return err
	}

	err = svc.ds.DeleteHost(host)
	if err != nil {
		return err
	}

	return nil
}
