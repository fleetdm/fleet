package service

import (
	"context"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/kit/endpoint"
)

// HostResponse is the response struct that contains the full host information
// along with the host online status and the "display text" to be used when
// rendering in the UI.
type HostResponse struct {
	*fleet.Host
	Status      fleet.HostStatus `json:"status"`
	DisplayText string           `json:"display_text"`
	Labels      []fleet.Label    `json:"labels,omitempty"`
}

func hostResponseForHost(ctx context.Context, svc fleet.Service, host *fleet.Host) (*HostResponse, error) {
	return &HostResponse{
		Host:        host,
		Status:      host.Status(time.Now()),
		DisplayText: host.Hostname,
	}, nil
}

// HostDetailResponse is the response struct that contains the full host information
// with the HostDetail details.
type HostDetailResponse struct {
	fleet.HostDetail
	Status      fleet.HostStatus `json:"status"`
	DisplayText string           `json:"display_text"`
}

func hostDetailResponseForHost(ctx context.Context, svc fleet.Service, host *fleet.HostDetail) (*HostDetailResponse, error) {
	return &HostDetailResponse{
		HostDetail:  *host,
		Status:      host.Status(time.Now()),
		DisplayText: host.Hostname,
	}, nil
}

////////////////////////////////////////////////////////////////////////////////
// Get Host
////////////////////////////////////////////////////////////////////////////////

type getHostRequest struct {
	ID uint `url:"id"`
}

type getHostResponse struct {
	Host *HostDetailResponse `json:"host"`
	Err  error               `json:"error,omitempty"`
}

func (r getHostResponse) error() error { return r.Err }

////////////////////////////////////////////////////////////////////////////////
// Get Host By Identifier
////////////////////////////////////////////////////////////////////////////////

type hostByIdentifierRequest struct {
	Identifier string `json:"identifier"`
}

func makeHostByIdentifierEndpoint(svc fleet.Service) endpoint.Endpoint {
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
// Delete Host
////////////////////////////////////////////////////////////////////////////////

type deleteHostRequest struct {
	ID uint `json:"id"`
}

type deleteHostResponse struct {
	Err error `json:"error,omitempty"`
}

func (r deleteHostResponse) error() error { return r.Err }

func makeDeleteHostEndpoint(svc fleet.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(deleteHostRequest)
		err := svc.DeleteHost(ctx, req.ID)
		if err != nil {
			return deleteHostResponse{Err: err}, nil
		}
		return deleteHostResponse{}, nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// Add Hosts to Team
////////////////////////////////////////////////////////////////////////////////

type addHostsToTeamRequest struct {
	TeamID  *uint  `json:"team_id"`
	HostIDs []uint `json:"hosts"`
}

type addHostsToTeamResponse struct {
	Err error `json:"error,omitempty"`
}

func (r addHostsToTeamResponse) error() error { return r.Err }

func makeAddHostsToTeamEndpoint(svc fleet.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(addHostsToTeamRequest)
		err := svc.AddHostsToTeam(ctx, req.TeamID, req.HostIDs)
		if err != nil {
			return addHostsToTeamResponse{Err: err}, nil
		}

		return addHostsToTeamResponse{}, err
	}
}

////////////////////////////////////////////////////////////////////////////////
// Add Hosts to Team by Filter
////////////////////////////////////////////////////////////////////////////////

type addHostsToTeamByFilterRequest struct {
	TeamID  *uint `json:"team_id"`
	Filters struct {
		MatchQuery string           `json:"query"`
		Status     fleet.HostStatus `json:"status"`
		LabelID    *uint            `json:"label_id"`
	} `json:"filters"`
}

type addHostsToTeamByFilterResponse struct {
	Err error `json:"error,omitempty"`
}

func (r addHostsToTeamByFilterResponse) error() error { return r.Err }

func makeAddHostsToTeamByFilterEndpoint(svc fleet.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(addHostsToTeamByFilterRequest)
		listOpt := fleet.HostListOptions{
			ListOptions: fleet.ListOptions{
				MatchQuery: req.Filters.MatchQuery,
			},
			StatusFilter: req.Filters.Status,
		}
		err := svc.AddHostsToTeamByFilter(ctx, req.TeamID, listOpt, req.Filters.LabelID)
		if err != nil {
			return addHostsToTeamByFilterResponse{Err: err}, nil
		}

		return addHostsToTeamByFilterResponse{}, err
	}
}

////////////////////////////////////////////////////////////////////////////////
// Refetch Host
////////////////////////////////////////////////////////////////////////////////

type refetchHostRequest struct {
	ID uint `json:"id"`
}

type refetchHostResponse struct {
	Err error `json:"error,omitempty"`
}

func (r refetchHostResponse) error() error { return r.Err }

func makeRefetchHostEndpoint(svc fleet.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(refetchHostRequest)
		err := svc.RefetchHost(ctx, req.ID)
		if err != nil {
			return refetchHostResponse{Err: err}, nil
		}
		return refetchHostResponse{}, nil
	}
}
