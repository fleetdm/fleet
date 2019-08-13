package service

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/kolide/fleet/server/kolide"
)

type statusResultStoreResponse struct {
	Err error `json:"error,omitempty"`
}

func (m statusResultStoreResponse) error() error { return m.Err }

func makeStatusResultStoreEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, req interface{}) (interface{}, error) {
		var resp statusResultStoreResponse
		if err := svc.StatusResultStore(ctx); err != nil {
			resp.Err = err
		}
		return resp, nil
	}
}
