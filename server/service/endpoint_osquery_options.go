package service

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/kolide/fleet/server/kolide"
)

////////////////////////////////////////////////////////////////////////////////
// Apply Options Spec
////////////////////////////////////////////////////////////////////////////////

type applyOsqueryOptionsSpecRequest struct {
	Spec *kolide.OptionsSpec `json:"spec"`
}

type applyOsqueryOptionsSpecResponse struct {
	Err error `json:"error,omitempty"`
}

func (r applyOsqueryOptionsSpecResponse) error() error { return r.Err }

func makeApplyOsqueryOptionsSpecEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(applyOsqueryOptionsSpecRequest)
		err := svc.ApplyOptionsSpec(ctx, req.Spec)
		if err != nil {
			return applyOsqueryOptionsSpecResponse{Err: err}, nil
		}
		return applyOsqueryOptionsSpecResponse{}, nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// Get Options Spec
////////////////////////////////////////////////////////////////////////////////

type getOsqueryOptionsSpecResponse struct {
	Spec *kolide.OptionsSpec `json:"spec"`
	Err  error               `json:"error,omitempty"`
}

func (r getOsqueryOptionsSpecResponse) error() error { return r.Err }

func makeGetOsqueryOptionsSpecEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		spec, err := svc.GetOptionsSpec(ctx)
		if err != nil {
			return getOsqueryOptionsSpecResponse{Err: err}, nil
		}
		return getOsqueryOptionsSpecResponse{Spec: spec}, nil
	}
}
