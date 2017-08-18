package service

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/kolide/fleet/server/kolide"
)

type modifyFIMResponse struct {
	Err error `json:"error,omitempty"`
}

func (m modifyFIMResponse) error() error { return m.Err }

func makeModifyFIMEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, req interface{}) (interface{}, error) {
		fimConfig := req.(kolide.FIMConfig)
		var resp modifyFIMResponse
		if err := svc.ModifyFIM(ctx, fimConfig); err != nil {
			resp.Err = err
		}
		return resp, nil
	}
}

type getFIMResponse struct {
	Err     error             `json:"error,omitempty"`
	Payload *kolide.FIMConfig `json:"payload,omitempty"`
}

func makeGetFIMEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, _ interface{}) (interface{}, error) {
		fimConfig, err := svc.GetFIM(ctx)
		if err != nil {
			return getFIMResponse{Err: err}, nil
		}
		return getFIMResponse{Payload: fimConfig}, nil
	}
}
