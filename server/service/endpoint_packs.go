package service

import (
	"context"

	"github.com/fleetdm/fleet/server/kolide"
	"github.com/go-kit/kit/endpoint"
)

type packResponse struct {
	kolide.Pack
	QueryCount uint `json:"query_count"`

	// All current hosts in the pack. Hosts which are selected explicty and
	// hosts which are part of a label.
	TotalHostsCount uint `json:"total_hosts_count"`

	// IDs of hosts which were explicitly selected.
	HostIDs  []uint `json:"host_ids"`
	LabelIDs []uint `json:"label_ids"`
}

func packResponseForPack(ctx context.Context, svc kolide.Service, pack kolide.Pack) (*packResponse, error) {
	opts := kolide.ListOptions{}
	queries, err := svc.GetScheduledQueriesInPack(ctx, pack.ID, opts)
	if err != nil {
		return nil, err
	}

	hosts, err := svc.ListExplicitHostsInPack(ctx, pack.ID, opts)
	if err != nil {
		return nil, err
	}

	labels, err := svc.ListLabelsForPack(ctx, pack.ID)
	labelIDs := make([]uint, len(labels))
	for i, label := range labels {
		labelIDs[i] = label.ID
	}
	if err != nil {
		return nil, err
	}

	hostMetrics, err := svc.CountHostsInTargets(ctx, hosts, labelIDs)
	if err != nil {
		return nil, err
	}

	return &packResponse{
		Pack:            pack,
		QueryCount:      uint(len(queries)),
		TotalHostsCount: hostMetrics.TotalHosts,
		HostIDs:         hosts,
		LabelIDs:        labelIDs,
	}, nil
}

////////////////////////////////////////////////////////////////////////////////
// Get Pack
////////////////////////////////////////////////////////////////////////////////

type getPackRequest struct {
	ID uint
}

type getPackResponse struct {
	Pack packResponse `json:"pack,omitempty"`
	Err  error        `json:"error,omitempty"`
}

func (r getPackResponse) error() error { return r.Err }

func makeGetPackEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getPackRequest)

		pack, err := svc.GetPack(ctx, req.ID)
		if err != nil {
			return getPackResponse{Err: err}, nil
		}

		resp, err := packResponseForPack(ctx, svc, *pack)
		if err != nil {
			return getPackResponse{Err: err}, nil
		}

		return getPackResponse{
			Pack: *resp,
		}, nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// List Packs
////////////////////////////////////////////////////////////////////////////////

type listPacksRequest struct {
	ListOptions kolide.ListOptions
}

type listPacksResponse struct {
	Packs []packResponse `json:"packs"`
	Err   error          `json:"error,omitempty"`
}

func (r listPacksResponse) error() error { return r.Err }

func makeListPacksEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listPacksRequest)
		packs, err := svc.ListPacks(ctx, req.ListOptions)
		if err != nil {
			return getPackResponse{Err: err}, nil
		}

		resp := listPacksResponse{Packs: make([]packResponse, len(packs))}
		for i, pack := range packs {
			packResp, err := packResponseForPack(ctx, svc, *pack)
			if err != nil {
				return getPackResponse{Err: err}, nil
			}
			resp.Packs[i] = *packResp
		}
		return resp, nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// Create Pack
////////////////////////////////////////////////////////////////////////////////

type createPackRequest struct {
	payload kolide.PackPayload
}

type createPackResponse struct {
	Pack packResponse `json:"pack,omitempty"`
	Err  error        `json:"error,omitempty"`
}

func (r createPackResponse) error() error { return r.Err }

func makeCreatePackEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createPackRequest)
		pack, err := svc.NewPack(ctx, req.payload)
		if err != nil {
			return createPackResponse{Err: err}, nil
		}

		resp, err := packResponseForPack(ctx, svc, *pack)
		if err != nil {
			return createPackResponse{Err: err}, nil
		}

		return createPackResponse{
			Pack: *resp,
		}, nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// Modify Pack
////////////////////////////////////////////////////////////////////////////////

type modifyPackRequest struct {
	ID      uint
	payload kolide.PackPayload
}

type modifyPackResponse struct {
	Pack packResponse `json:"pack,omitempty"`
	Err  error        `json:"error,omitempty"`
}

func (r modifyPackResponse) error() error { return r.Err }

func makeModifyPackEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(modifyPackRequest)
		pack, err := svc.ModifyPack(ctx, req.ID, req.payload)
		if err != nil {
			return modifyPackResponse{Err: err}, nil
		}

		resp, err := packResponseForPack(ctx, svc, *pack)
		if err != nil {
			return modifyPackResponse{Err: err}, nil
		}

		return modifyPackResponse{
			Pack: *resp,
		}, nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// Delete Pack
////////////////////////////////////////////////////////////////////////////////

type deletePackRequest struct {
	Name string
}

type deletePackResponse struct {
	Err error `json:"error,omitempty"`
}

func (r deletePackResponse) error() error { return r.Err }

func makeDeletePackEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(deletePackRequest)
		err := svc.DeletePack(ctx, req.Name)
		if err != nil {
			return deletePackResponse{Err: err}, nil
		}
		return deletePackResponse{}, nil
	}
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

func makeDeletePackByIDEndpoint(svc kolide.Service) endpoint.Endpoint {
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
// Apply Pack Specs
////////////////////////////////////////////////////////////////////////////////

type applyPackSpecsRequest struct {
	Specs []*kolide.PackSpec `json:"specs"`
}

type applyPackSpecsResponse struct {
	Err error `json:"error,omitempty"`
}

func (r applyPackSpecsResponse) error() error { return r.Err }

func makeApplyPackSpecsEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(applyPackSpecsRequest)
		err := svc.ApplyPackSpecs(ctx, req.Specs)
		if err != nil {
			return applyPackSpecsResponse{Err: err}, nil
		}
		return applyPackSpecsResponse{}, nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// Get Pack Specs
////////////////////////////////////////////////////////////////////////////////

type getPackSpecsResponse struct {
	Specs []*kolide.PackSpec `json:"specs"`
	Err   error              `json:"error,omitempty"`
}

func (r getPackSpecsResponse) error() error { return r.Err }

func makeGetPackSpecsEndpoint(svc kolide.Service) endpoint.Endpoint {
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
	Spec *kolide.PackSpec `json:"specs,omitempty"`
	Err  error            `json:"error,omitempty"`
}

func (r getPackSpecResponse) error() error { return r.Err }

func makeGetPackSpecEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getGenericSpecRequest)
		spec, err := svc.GetPackSpec(ctx, req.Name)
		if err != nil {
			return getPackSpecResponse{Err: err}, nil
		}
		return getPackSpecResponse{Spec: spec}, nil
	}
}
