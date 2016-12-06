package service

import (
	"time"

	"github.com/go-kit/kit/endpoint"
	"github.com/kolide/kolide-ose/server/kolide"
	"golang.org/x/net/context"
)

const (
	// StatusOnline host is active
	StatusOnline string = "online"
	// StatusOffline no communication with host for OfflineDuration
	StatusOffline string = "offline"
	// StatusMIA no communition with host for MIADuration
	StatusMIA string = "mia"
	// OfflineDuration if a host hasn't been in communition for this
	// period it is considered offline
	OfflineDuration time.Duration = 30 * time.Minute
	// OfflineDuration if a host hasn't been in communition for this
	// period it is considered MIA
	MIADuration time.Duration = 30 * 24 * time.Hour
)

type hostResponse struct {
	kolide.Host
	Status string `json:"status"`
}

////////////////////////////////////////////////////////////////////////////////
// Get Host
////////////////////////////////////////////////////////////////////////////////

type getHostRequest struct {
	ID uint `json:"id"`
}

type getHostResponse struct {
	Host *hostResponse `json:"host"`
	Err  error         `json:"error,omitempty"`
}

func (r getHostResponse) error() error { return r.Err }

func makeGetHostEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getHostRequest)
		host, err := svc.GetHost(ctx, req.ID)
		if err != nil {
			return getHostResponse{Err: err}, nil
		}
		return getHostResponse{&hostResponse{*host, svc.HostStatus(ctx, *host)}, nil}, nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// List Hosts
////////////////////////////////////////////////////////////////////////////////

type listHostsRequest struct {
	ListOptions kolide.ListOptions
}

type listHostsResponse struct {
	Hosts []hostResponse `json:"hosts"`
	Err   error          `json:"error,omitempty"`
}

func (r listHostsResponse) error() error { return r.Err }

func makeListHostsEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listHostsRequest)
		hosts, err := svc.ListHosts(ctx, req.ListOptions)
		if err != nil {
			return listHostsResponse{Err: err}, nil
		}

		resp := listHostsResponse{Hosts: []hostResponse{}}
		for _, host := range hosts {
			resp.Hosts = append(resp.Hosts, hostResponse{*host, svc.HostStatus(ctx, *host)})
		}
		return resp, nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// Delete Host
////////////////////////////////////////////////////////////////////////////////

type deleteHostRequest struct {
	ID uint `json:"id"`
}

type deleteHostResponse struct {
	Err error `json:"error,omitempty"`
}

func (r deleteHostResponse) error() error { return r.Err }

func makeDeleteHostEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(deleteHostRequest)
		err := svc.DeleteHost(ctx, req.ID)
		if err != nil {
			return deleteHostResponse{Err: err}, nil
		}
		return deleteHostResponse{}, nil
	}
}
