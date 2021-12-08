package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/kit/endpoint"
)

////////////////////////////////////////////////////////////////////////////////
// List Hosts in Label
////////////////////////////////////////////////////////////////////////////////

type listHostsInLabelRequest struct {
	ID          uint
	ListOptions fleet.HostListOptions
}

func makeListHostsInLabelEndpoint(svc fleet.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listHostsInLabelRequest)
		hosts, err := svc.ListHostsInLabel(ctx, req.ID, req.ListOptions)
		if err != nil {
			return listLabelsResponse{Err: err}, nil
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
// Delete Label
////////////////////////////////////////////////////////////////////////////////

type deleteLabelRequest struct {
	Name string
}

type deleteLabelResponse struct {
	Err error `json:"error,omitempty"`
}

func (r deleteLabelResponse) error() error { return r.Err }

func makeDeleteLabelEndpoint(svc fleet.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(deleteLabelRequest)
		err := svc.DeleteLabel(ctx, req.Name)
		if err != nil {
			return deleteLabelResponse{Err: err}, nil
		}
		return deleteLabelResponse{}, nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// Delete Label By ID
////////////////////////////////////////////////////////////////////////////////

type deleteLabelByIDRequest struct {
	ID uint
}

type deleteLabelByIDResponse struct {
	Err error `json:"error,omitempty"`
}

func (r deleteLabelByIDResponse) error() error { return r.Err }

func makeDeleteLabelByIDEndpoint(svc fleet.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(deleteLabelByIDRequest)
		err := svc.DeleteLabelByID(ctx, req.ID)
		if err != nil {
			return deleteLabelByIDResponse{Err: err}, nil
		}
		return deleteLabelByIDResponse{}, nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// Apply Label Spec
////////////////////////////////////////////////////////////////////////////////

type applyLabelSpecsRequest struct {
	Specs []*fleet.LabelSpec `json:"specs"`
}

type applyLabelSpecsResponse struct {
	Err error `json:"error,omitempty"`
}

func (r applyLabelSpecsResponse) error() error { return r.Err }

func makeApplyLabelSpecsEndpoint(svc fleet.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(applyLabelSpecsRequest)
		err := svc.ApplyLabelSpecs(ctx, req.Specs)
		if err != nil {
			return applyLabelSpecsResponse{Err: err}, nil
		}
		return applyLabelSpecsResponse{}, nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// Get Label Spec
////////////////////////////////////////////////////////////////////////////////

type getLabelSpecsResponse struct {
	Specs []*fleet.LabelSpec `json:"specs"`
	Err   error              `json:"error,omitempty"`
}

func (r getLabelSpecsResponse) error() error { return r.Err }

func makeGetLabelSpecsEndpoint(svc fleet.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		specs, err := svc.GetLabelSpecs(ctx)
		if err != nil {
			return getLabelSpecsResponse{Err: err}, nil
		}
		return getLabelSpecsResponse{Specs: specs}, nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// Get Label Spec
////////////////////////////////////////////////////////////////////////////////

type getLabelSpecResponse struct {
	Spec *fleet.LabelSpec `json:"specs,omitempty"`
	Err  error            `json:"error,omitempty"`
}

func (r getLabelSpecResponse) error() error { return r.Err }

func makeGetLabelSpecEndpoint(svc fleet.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getGenericSpecRequest)
		spec, err := svc.GetLabelSpec(ctx, req.Name)
		if err != nil {
			return getLabelSpecResponse{Err: err}, nil
		}
		return getLabelSpecResponse{Spec: spec}, nil
	}
}
