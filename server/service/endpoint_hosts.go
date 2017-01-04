package service

import (
	"time"

	"github.com/go-kit/kit/endpoint"
	"github.com/kolide/kolide-ose/server/kolide"
	"golang.org/x/net/context"
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
		return getHostResponse{
			&hostResponse{
				Host:   *host,
				Status: host.Status(time.Now()),
			},
			nil,
		}, nil
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
			resp.Hosts = append(
				resp.Hosts,
				hostResponse{
					Host:   *host,
					Status: host.Status(time.Now()),
				},
			)
		}
		return resp, nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// Get Host Summary
////////////////////////////////////////////////////////////////////////////////

type getHostSummaryResponse struct {
	kolide.HostSummary
	Err error `json:"error,omitempty"`
}

func (r getHostSummaryResponse) error() error { return r.Err }

func makeGetHostSummaryEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		summary, err := svc.GetHostSummary(ctx)
		if err != nil {
			return getHostSummaryResponse{Err: err}, nil
		}

		resp := getHostSummaryResponse{
			HostSummary: *summary,
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
