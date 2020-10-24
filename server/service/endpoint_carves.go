package service

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/kolide/fleet/server/kolide"
)

////////////////////////////////////////////////////////////////////////////////
// Begin File Carve
////////////////////////////////////////////////////////////////////////////////

type carveBeginRequest struct {
	NodeKey    string `json:"node_key"`
	BlockCount int    `json:"block_count"`
	BlockSize  int    `json:"block_size"`
	CarveSize  int    `json:"carve_size"`
	CarveId    string `json:"carve_id"`
	RequestId  string `json:"request_id"`
}

type carveBeginResponse struct {
	SessionId string `json:"session_id"`
	Success   bool   `json:"success,omitempty"`
	Err       error  `json:"error,omitempty"`
}

func (r carveBeginResponse) error() error { return r.Err }

func makeCarveBeginEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(carveBeginRequest)

		payload := kolide.CarveBeginPayload{
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
	BlockId   int    `json:"block_id"`
	SessionId string `json:"session_id"`
	RequestId string `json:"request_id"`
	Data      string `json:"data"`
}

type carveBlockResponse struct {
	Success bool  `json:"success,omitempty"`
	Err     error `json:"error,omitempty"`
}

func (r carveBlockResponse) error() error { return r.Err }

func makeCarveBlockEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(carveBlockRequest)

		payload := kolide.CarveBlockPayload{
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
// List Carves
////////////////////////////////////////////////////////////////////////////////

type carveResponse struct {
	kolide.CarveMetadata
}

type listCarvesRequest struct {
	ListOptions kolide.ListOptions
}

type listCarvesResponse struct {
	Carves []carveResponse `json:"carves"`
	Err    error           `json:"error,omitempty"`
}

func (r listCarvesResponse) error() error { return r.Err }

func makeListCarvesEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listCarvesRequest)
		carves, err := svc.ListCarves(ctx, req.ListOptions)
		if err != nil {
			return listCarvesResponse{Err: err}, nil
		}

		resp := listCarvesResponse{}
		for _, carve := range carves {
			resp.Carves = append(resp.Carves, carveResponse{*carve})
		}
		return resp, nil
	}
}
