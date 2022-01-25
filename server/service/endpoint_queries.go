package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/kit/endpoint"
)

////////////////////////////////////////////////////////////////////////////////
// Apply Query Spec
////////////////////////////////////////////////////////////////////////////////

type applyQuerySpecsRequest struct {
	Specs []*fleet.QuerySpec `json:"specs"`
}

type applyQuerySpecsResponse struct {
	Err error `json:"error,omitempty"`
}

func (r applyQuerySpecsResponse) error() error { return r.Err }

func makeApplyQuerySpecsEndpoint(svc fleet.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(applyQuerySpecsRequest)
		err := svc.ApplyQuerySpecs(ctx, req.Specs)
		if err != nil {
			return applyQuerySpecsResponse{Err: err}, nil
		}
		return applyQuerySpecsResponse{}, nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// Get Query Spec
////////////////////////////////////////////////////////////////////////////////

type getQuerySpecsResponse struct {
	Specs []*fleet.QuerySpec `json:"specs"`
	Err   error              `json:"error,omitempty"`
}

func (r getQuerySpecsResponse) error() error { return r.Err }

func makeGetQuerySpecsEndpoint(svc fleet.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		specs, err := svc.GetQuerySpecs(ctx)
		if err != nil {
			return getQuerySpecsResponse{Err: err}, nil
		}
		return getQuerySpecsResponse{Specs: specs}, nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// Get Query Spec
////////////////////////////////////////////////////////////////////////////////

type getQuerySpecResponse struct {
	Spec *fleet.QuerySpec `json:"specs,omitempty"`
	Err  error            `json:"error,omitempty"`
}

func (r getQuerySpecResponse) error() error { return r.Err }

func makeGetQuerySpecEndpoint(svc fleet.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getGenericSpecRequest)
		spec, err := svc.GetQuerySpec(ctx, req.Name)
		if err != nil {
			return getQuerySpecResponse{Err: err}, nil
		}
		return getQuerySpecResponse{Spec: spec}, nil
	}
}
