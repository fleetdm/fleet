package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/kit/endpoint"
)

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
