package service

import (
	"time"

	"github.com/kolide/kolide-ose/server/kolide"
	"golang.org/x/net/context"
)

func (svc service) ListHosts(ctx context.Context) ([]*kolide.Host, error) {
	return svc.ds.Hosts()
}

func (svc service) GetHost(ctx context.Context, id uint) (*kolide.Host, error) {
	return svc.ds.Host(id)
}

func (svc service) HostStatus(ctx context.Context, host kolide.Host) string {
	if host.UpdatedAt.Add(30 * time.Minute).Before(svc.clock.Now()) {
		return "offline"
	} else {
		return "online"
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
