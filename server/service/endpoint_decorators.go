package service

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/kolide/fleet/server/kolide"
)

type listDecoratorResponse struct {
	Decorators []*kolide.Decorator `json:"decorators"`
	Err        error               `json:"error,omitempty"`
}

func (r listDecoratorResponse) error() error { return r.Err }

func makeListDecoratorsEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		decs, err := svc.ListDecorators(ctx)
		if err != nil {
			return listDecoratorResponse{Err: err}, nil
		}
		return listDecoratorResponse{Decorators: decs}, nil
	}
}

type newDecoratorRequest struct {
	Payload kolide.DecoratorPayload `json:"payload"`
}

type decoratorResponse struct {
	Decorator *kolide.Decorator `json:"decorator,omitempty"`
	Err       error             `json:"error,omitempty"`
}

func (r decoratorResponse) error() error { return r.Err }

func makeNewDecoratorEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		r := request.(newDecoratorRequest)
		dec, err := svc.NewDecorator(ctx, r.Payload)
		if err != nil {
			return decoratorResponse{Err: err}, nil
		}
		return decoratorResponse{Decorator: dec}, nil
	}
}

type deleteDecoratorRequest struct {
	ID uint
}

type deleteDecoratorResponse struct {
	Err error `json:"error,omitempty"`
}

func (r deleteDecoratorResponse) error() error { return r.Err }

func makeDeleteDecoratorEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		r := request.(deleteDecoratorRequest)
		err := svc.DeleteDecorator(ctx, r.ID)

		if err != nil {
			return deleteDecoratorResponse{Err: err}, nil
		}
		return deleteDecoratorResponse{}, nil
	}
}

func makeModifyDecoratorEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		r := request.(newDecoratorRequest)
		dec, err := svc.ModifyDecorator(ctx, r.Payload)
		if err != nil {
			return decoratorResponse{Err: err}, nil
		}
		return decoratorResponse{Decorator: dec}, nil

	}
}
