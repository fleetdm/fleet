package service

import (
	"context"
	"time"

	"github.com/fleetdm/fleet/server/kolide"
	"github.com/go-kit/kit/endpoint"
)

// HostResponse is the response struct that contains the full host information
// along with the host online status and the "display text" to be used when
// rendering in the UI.
type HostResponse struct {
	kolide.Host
	Status      kolide.HostStatus `json:"status"`
	DisplayText string            `json:"display_text"`
	Labels      []kolide.Label    `json:"labels,omitempty"`
}

func hostResponseForHost(ctx context.Context, svc kolide.Service, host *kolide.Host) (*HostResponse, error) {
	return &HostResponse{
		Host:        *host,
		Status:      host.Status(time.Now()),
		DisplayText: host.HostName,
	}, nil
}

// HostDetailresponse is the response struct that contains the full host information
// with the HostDetail details.
type HostDetailResponse struct {
	kolide.HostDetail
	Status      kolide.HostStatus `json:"status"`
	DisplayText string            `json:"display_text"`
}

func hostDetailResponseForHost(ctx context.Context, svc kolide.Service, host *kolide.HostDetail) (*HostDetailResponse, error) {
	return &HostDetailResponse{
		HostDetail:  *host,
		Status:      host.Status(time.Now()),
		DisplayText: host.HostName,
	}, nil
}

////////////////////////////////////////////////////////////////////////////////
// Get Host
////////////////////////////////////////////////////////////////////////////////

type getHostRequest struct {
	ID uint `json:"id"`
}

type getHostResponse struct {
	Host *HostDetailResponse `json:"host"`
	Err  error               `json:"error,omitempty"`
}

func (r getHostResponse) error() error { return r.Err }

func makeGetHostEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getHostRequest)
		host, err := svc.GetHost(ctx, req.ID)
		if err != nil {
			return getHostResponse{Err: err}, nil
		}

		resp, err := hostDetailResponseForHost(ctx, svc, host)
		if err != nil {
			return getHostResponse{Err: err}, nil
		}

		return getHostResponse{
			Host: resp,
		}, nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// Get Host By Identifier
////////////////////////////////////////////////////////////////////////////////

type hostByIdentifierRequest struct {
	Identifier string `json:"identifier"`
}

func makeHostByIdentifierEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(hostByIdentifierRequest)
		host, err := svc.HostByIdentifier(ctx, req.Identifier)
		if err != nil {
			return getHostResponse{Err: err}, nil
		}

		resp, err := hostDetailResponseForHost(ctx, svc, host)
		if err != nil {
			return getHostResponse{Err: err}, nil
		}

		return getHostResponse{
			Host: resp,
		}, nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// List Hosts
////////////////////////////////////////////////////////////////////////////////

type listHostsRequest struct {
	ListOptions kolide.HostListOptions
}

type listHostsResponse struct {
	Hosts []HostResponse `json:"hosts"`
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

		hostResponses := make([]HostResponse, len(hosts))
		for i, host := range hosts {
			h, err := hostResponseForHost(ctx, svc, host)
			if err != nil {
				return listHostsResponse{Err: err}, nil
			}

			hostResponses[i] = *h
		}
		return listHostsResponse{Hosts: hostResponses}, nil
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
