package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

////////////////////////////////////////////////////////////////////////////////
// List Carves
////////////////////////////////////////////////////////////////////////////////

type listCarvesRequest struct {
	ListOptions fleet.CarveListOptions `url:"carve_options"`
}

type listCarvesResponse struct {
	Carves []fleet.CarveMetadata `json:"carves"`
	Err    error                 `json:"error,omitempty"`
}

func (r listCarvesResponse) error() error { return r.Err }

func listCarvesEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*listCarvesRequest)
	carves, err := svc.ListCarves(ctx, req.ListOptions)
	if err != nil {
		return listCarvesResponse{Err: err}, nil
	}

	resp := listCarvesResponse{}
	for _, carve := range carves {
		resp.Carves = append(resp.Carves, *carve)
	}
	return resp, nil
}

func (svc *Service) ListCarves(ctx context.Context, opt fleet.CarveListOptions) ([]*fleet.CarveMetadata, error) {
	if err := svc.authz.Authorize(ctx, &fleet.CarveMetadata{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	return svc.carveStore.ListCarves(ctx, opt)
}

////////////////////////////////////////////////////////////////////////////////
// Get Carve
////////////////////////////////////////////////////////////////////////////////

type getCarveRequest struct {
	ID int64 `url:"id"`
}

type getCarveResponse struct {
	Carve fleet.CarveMetadata `json:"carve"`
	Err   error               `json:"error,omitempty"`
}

func (r getCarveResponse) error() error { return r.Err }

func getCarveEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*getCarveRequest)
	carve, err := svc.GetCarve(ctx, req.ID)
	if err != nil {
		return getCarveResponse{Err: err}, nil
	}

	return getCarveResponse{Carve: *carve}, nil

}

func (svc *Service) GetCarve(ctx context.Context, id int64) (*fleet.CarveMetadata, error) {
	if err := svc.authz.Authorize(ctx, &fleet.CarveMetadata{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	return svc.carveStore.Carve(ctx, id)
}
