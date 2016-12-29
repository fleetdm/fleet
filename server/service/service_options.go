package service

import (
	"github.com/kolide/kolide-ose/server/kolide"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
)

func (svc service) GetOptions(ctx context.Context) ([]kolide.Option, error) {
	opts, err := svc.ds.ListOptions()
	if err != nil {
		return nil, errors.Wrap(err, "options service")
	}
	return opts, nil
}

func (svc service) ModifyOptions(ctx context.Context, req kolide.OptionRequest) ([]kolide.Option, error) {
	if err := svc.ds.SaveOptions(req.Options); err != nil {
		return nil, errors.Wrap(err, "modify options service")
	}
	return req.Options, nil
}
