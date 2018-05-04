package service

import (
	"context"

	"github.com/kolide/fleet/server/kolide"
	"github.com/pkg/errors"
)

func (svc service) ApplyOptionsSpec(ctx context.Context, spec *kolide.OptionsSpec) error {
	err := svc.ds.ApplyOptions(spec)
	if err != nil {
		return errors.Wrap(err, "apply options")
	}
	return nil
}

func (svc service) GetOptionsSpec(ctx context.Context) (*kolide.OptionsSpec, error) {
	spec, err := svc.ds.GetOptions()
	if err != nil {
		return nil, errors.Wrap(err, "get options from datastore")
	}

	return spec, nil
}
