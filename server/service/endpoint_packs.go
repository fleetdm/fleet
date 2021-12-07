package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/kit/endpoint"
)

type packResponse struct {
	fleet.Pack
	QueryCount uint `json:"query_count"`

	// All current hosts in the pack. Hosts which are selected explicty and
	// hosts which are part of a label.
	TotalHostsCount uint `json:"total_hosts_count"`

	// IDs of hosts which were explicitly selected.
	HostIDs  []uint `json:"host_ids"`
	LabelIDs []uint `json:"label_ids"`
	TeamIDs  []uint `json:"team_ids"`
}

func packResponseForPack(ctx context.Context, svc fleet.Service, pack fleet.Pack) (*packResponse, error) {
	opts := fleet.ListOptions{}
	queries, err := svc.GetScheduledQueriesInPack(ctx, pack.ID, opts)
	if err != nil {
		return nil, err
	}

	hostMetrics, err := svc.CountHostsInTargets(
		ctx,
		nil,
		fleet.HostTargets{HostIDs: pack.HostIDs, LabelIDs: pack.LabelIDs, TeamIDs: pack.TeamIDs},
	)
	if err != nil {
		return nil, err
	}

	return &packResponse{
		Pack:            pack,
		QueryCount:      uint(len(queries)),
		TotalHostsCount: hostMetrics.TotalHosts,
		HostIDs:         pack.HostIDs,
		LabelIDs:        pack.LabelIDs,
		TeamIDs:         pack.TeamIDs,
	}, nil
}

////////////////////////////////////////////////////////////////////////////////
// Delete Pack By ID
////////////////////////////////////////////////////////////////////////////////

type deletePackByIDRequest struct {
	ID uint
}

type deletePackByIDResponse struct {
	Err error `json:"error,omitempty"`
}

func (r deletePackByIDResponse) error() error { return r.Err }

func makeDeletePackByIDEndpoint(svc fleet.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(deletePackByIDRequest)
		err := svc.DeletePackByID(ctx, req.ID)
		if err != nil {
			return deletePackByIDResponse{Err: err}, nil
		}
		return deletePackByIDResponse{}, nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// Apply Pack Spec
////////////////////////////////////////////////////////////////////////////////

type applyPackSpecsRequest struct {
	Specs []*fleet.PackSpec `json:"specs"`
}

type applyPackSpecsResponse struct {
	Err error `json:"error,omitempty"`
}

func (r applyPackSpecsResponse) error() error { return r.Err }

func makeApplyPackSpecsEndpoint(svc fleet.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(applyPackSpecsRequest)
		_, err := svc.ApplyPackSpecs(ctx, req.Specs)
		if err != nil {
			return applyPackSpecsResponse{Err: err}, nil
		}
		return applyPackSpecsResponse{}, nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// Get Pack Spec
////////////////////////////////////////////////////////////////////////////////

type getPackSpecsResponse struct {
	Specs []*fleet.PackSpec `json:"specs"`
	Err   error             `json:"error,omitempty"`
}

func (r getPackSpecsResponse) error() error { return r.Err }

func makeGetPackSpecsEndpoint(svc fleet.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		specs, err := svc.GetPackSpecs(ctx)
		if err != nil {
			return getPackSpecsResponse{Err: err}, nil
		}
		return getPackSpecsResponse{Specs: specs}, nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// Get Pack Spec
////////////////////////////////////////////////////////////////////////////////

type getPackSpecResponse struct {
	Spec *fleet.PackSpec `json:"specs,omitempty"`
	Err  error           `json:"error,omitempty"`
}

func (r getPackSpecResponse) error() error { return r.Err }

func makeGetPackSpecEndpoint(svc fleet.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getGenericSpecRequest)
		spec, err := svc.GetPackSpec(ctx, req.Name)
		if err != nil {
			return getPackSpecResponse{Err: err}, nil
		}
		return getPackSpecResponse{Spec: spec}, nil
	}
}
