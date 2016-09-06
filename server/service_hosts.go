package server

import (
	"github.com/kolide/kolide-ose/kolide"
	"golang.org/x/net/context"
)

func (svc service) GetAllHosts(ctx context.Context) ([]*kolide.Host, error) {
	return svc.ds.Hosts()
}

func (svc service) GetHost(ctx context.Context, id uint) (*kolide.Host, error) {
	return svc.ds.Host(id)
}

func (svc service) NewHost(ctx context.Context, p kolide.HostPayload) (*kolide.Host, error) {
	var host kolide.Host

	if p.HostName != nil {
		host.HostName = *p.HostName
	}

	if p.IPAddress != nil {
		host.IPAddress = *p.IPAddress
	}

	if p.NodeKey != nil {
		host.NodeKey = *p.NodeKey
	}

	if p.Platform != nil {
		host.Platform = *p.Platform
	}

	if p.UUID != nil {
		host.UUID = *p.UUID
	}

	_, err := svc.ds.NewHost(&host)
	if err != nil {
		return nil, err
	}
	return &host, nil
}

func (svc service) ModifyHost(ctx context.Context, id uint, p kolide.HostPayload) (*kolide.Host, error) {
	host, err := svc.ds.Host(id)
	if err != nil {
		return nil, err
	}

	if p.HostName != nil {
		host.HostName = *p.HostName
	}

	if p.IPAddress != nil {
		host.IPAddress = *p.IPAddress
	}

	if p.NodeKey != nil {
		host.NodeKey = *p.NodeKey
	}

	if p.Platform != nil {
		host.Platform = *p.Platform
	}

	if p.UUID != nil {
		host.UUID = *p.UUID
	}

	err = svc.ds.SaveHost(host)
	if err != nil {
		return nil, err
	}

	return host, nil
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
