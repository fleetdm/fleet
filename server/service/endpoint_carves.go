package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/kit/endpoint"
)

////////////////////////////////////////////////////////////////////////////////
// Begin File Carve
////////////////////////////////////////////////////////////////////////////////

type carveBeginRequest struct {
	NodeKey    string `json:"node_key"`
	BlockCount int64  `json:"block_count"`
	BlockSize  int64  `json:"block_size"`
	CarveSize  int64  `json:"carve_size"`
	CarveId    string `json:"carve_id"`
	RequestId  string `json:"request_id"`
}

type carveBeginResponse struct {
	SessionId string `json:"session_id"`
	Success   bool   `json:"success,omitempty"`
	Err       error  `json:"error,omitempty"`
}

func (r carveBeginResponse) error() error { return r.Err }

func makeCarveBeginEndpoint(svc fleet.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(carveBeginRequest)

		payload := fleet.CarveBeginPayload{
			BlockCount: req.BlockCount,
			BlockSize:  req.BlockSize,
			CarveSize:  req.CarveSize,
			CarveId:    req.CarveId,
			RequestId:  req.RequestId,
		}

		carve, err := svc.CarveBegin(ctx, payload)
		if err != nil {
			return carveBeginResponse{Err: err}, nil
		}

		return carveBeginResponse{SessionId: carve.SessionId, Success: true}, nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// Receive Block for File Carve
////////////////////////////////////////////////////////////////////////////////

type carveBlockRequest struct {
	NodeKey   string `json:"node_key"`
	BlockId   int64  `json:"block_id"`
	SessionId string `json:"session_id"`
	RequestId string `json:"request_id"`
	Data      []byte `json:"data"`
}

type carveBlockResponse struct {
	Success bool  `json:"success,omitempty"`
	Err     error `json:"error,omitempty"`
}

func (r carveBlockResponse) error() error { return r.Err }

func makeCarveBlockEndpoint(svc fleet.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(carveBlockRequest)

		payload := fleet.CarveBlockPayload{
			SessionId: req.SessionId,
			RequestId: req.RequestId,
			BlockId:   req.BlockId,
			Data:      req.Data,
		}

		err := svc.CarveBlock(ctx, payload)
		if err != nil {
			return carveBlockResponse{Err: err}, nil
		}

		return carveBlockResponse{Success: true}, nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// Get Carve
////////////////////////////////////////////////////////////////////////////////

type getCarveRequest struct {
	ID int64
}

type getCarveResponse struct {
	Carve fleet.CarveMetadata `json:"carve"`
	Err   error               `json:"error,omitempty"`
}

func (r getCarveResponse) error() error { return r.Err }

func makeGetCarveEndpoint(svc fleet.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getCarveRequest)
		carve, err := svc.GetCarve(ctx, req.ID)
		if err != nil {
			return getCarveResponse{Err: err}, nil
		}

		return getCarveResponse{Carve: *carve}, nil

	}
}

////////////////////////////////////////////////////////////////////////////////
// List Carves
////////////////////////////////////////////////////////////////////////////////

type listCarvesRequest struct {
	ListOptions fleet.CarveListOptions
}

type listCarvesResponse struct {
	Carves []fleet.CarveMetadata `json:"carves"`
	Err    error                 `json:"error,omitempty"`
}

func (r listCarvesResponse) error() error { return r.Err }

func makeListCarvesEndpoint(svc fleet.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listCarvesRequest)
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
}

////////////////////////////////////////////////////////////////////////////////
// Get Carve Block
////////////////////////////////////////////////////////////////////////////////

type getCarveBlockRequest struct {
	ID      int64
	BlockId int64
}

type getCarveBlockResponse struct {
	Data []byte `json:"data"`
	Err  error  `json:"error,omitempty"`
}

func (r getCarveBlockResponse) error() error { return r.Err }

func makeGetCarveBlockEndpoint(svc fleet.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getCarveBlockRequest)
		data, err := svc.GetBlock(ctx, req.ID, req.BlockId)
		if err != nil {
			return getCarveBlockResponse{Err: err}, nil
		}

		return getCarveBlockResponse{Data: data}, nil
	}
}
