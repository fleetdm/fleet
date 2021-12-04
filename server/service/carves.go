package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
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

////////////////////////////////////////////////////////////////////////////////
// Get Carve Block
////////////////////////////////////////////////////////////////////////////////

type getCarveBlockRequest struct {
	ID      int64 `url:"id"`
	BlockId int64 `url:"block_id"`
}

type getCarveBlockResponse struct {
	Data []byte `json:"data"`
	Err  error  `json:"error,omitempty"`
}

func (r getCarveBlockResponse) error() error { return r.Err }

func getCarveBlockEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*getCarveBlockRequest)
	data, err := svc.GetBlock(ctx, req.ID, req.BlockId)
	if err != nil {
		return getCarveBlockResponse{Err: err}, nil
	}

	return getCarveBlockResponse{Data: data}, nil
}

func (svc *Service) GetBlock(ctx context.Context, carveId, blockId int64) ([]byte, error) {
	if err := svc.authz.Authorize(ctx, &fleet.CarveMetadata{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	metadata, err := svc.carveStore.Carve(ctx, carveId)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get carve by name")
	}

	if metadata.Expired {
		return nil, errors.New("cannot get block for expired carve")
	}

	if blockId > metadata.MaxBlock {
		return nil, fmt.Errorf("block %d not yet available", blockId)
	}

	data, err := svc.carveStore.GetBlock(ctx, metadata, blockId)
	if err != nil {
		return nil, ctxerr.Wrapf(ctx, err, "get block %d", blockId)
	}

	return data, nil
}
