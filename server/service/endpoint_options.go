package service

import (
	"github.com/go-kit/kit/endpoint"
	"github.com/kolide/kolide-ose/server/kolide"
	"golang.org/x/net/context"
)

type optionsResponse struct {
	Options []kolide.Option `json:"options,omitempty"`
	Err     error           `json:"error,omitempty"`
}

func (or optionsResponse) error() error { return or.Err }

func makeGetOptionsEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		options, err := svc.GetOptions(ctx)
		if err != nil {
			return optionsResponse{Err: err}, nil
		}
		return optionsResponse{Options: options}, nil
	}
}

func makeModifyOptionsEndpoint(svc kolide.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		payload := request.(kolide.OptionRequest)
		opts, err := svc.ModifyOptions(ctx, payload)
		if err != nil {
			return optionsResponse{Err: err}, nil
		}
		return optionsResponse{Options: opts}, nil
	}
}
