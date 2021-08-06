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

func makeGetPackEndpoint(svc fleet.Service) endpoint.Endpoint {
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
	ListOptions fleet.ListOptions
}

type listPacksResponse struct {
	Packs []packResponse `json:"packs"`
	Err   error          `json:"error,omitempty"`
}

func (r listPacksResponse) error() error { return r.Err }

func makeListPacksEndpoint(svc fleet.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listPacksRequest)
		packs, err := svc.ListPacks(ctx, fleet.PackListOptions{ListOptions: req.ListOptions, IncludeSystemPacks: false})
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
	payload fleet.PackPayload
}

type createPackResponse struct {
	Pack packResponse `json:"pack,omitempty"`
	Err  error        `json:"error,omitempty"`
}

func (r createPackResponse) error() error { return r.Err }

func makeCreatePackEndpoint(svc fleet.Service) endpoint.Endpoint {
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
	payload fleet.PackPayload
}

type modifyPackResponse struct {
	Pack packResponse `json:"pack,omitempty"`
	Err  error        `json:"error,omitempty"`
}

func (r modifyPackResponse) error() error { return r.Err }

func makeModifyPackEndpoint(svc fleet.Service) endpoint.Endpoint {
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

func makeDeletePackEndpoint(svc fleet.Service) endpoint.Endpoint {
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
