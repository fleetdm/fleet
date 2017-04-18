package service

import (
	"context"

	"github.com/kolide/kolide/server/kolide"
)

func (svc service) ResetOptions(ctx context.Context) ([]kolide.Option, error) {
	return svc.ds.ResetOptions()
}

func (svc service) GetOptions(ctx context.Context) ([]kolide.Option, error) {
	opts, err := svc.ds.ListOptions()
	if err != nil {
		return nil, err
	}
	return opts, nil
}

func (svc service) ModifyOptions(ctx context.Context, req kolide.OptionRequest) ([]kolide.Option, error) {
	if err := svc.ds.SaveOptions(req.Options); err != nil {
		return nil, err
	}
	return req.Options, nil
}
