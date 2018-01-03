package service

import (
	"github.com/kolide/fleet/server/kolide"
	"github.com/pkg/errors"
)

func (svc service) ApplyOptionsSpec(spec *kolide.OptionsSpec) error {
	err := svc.ds.ApplyOptions(spec)
	return errors.Wrap(err, "apply options")
}

func (svc service) GetOptionsSpec() (*kolide.OptionsSpec, error) {
	spec, err := svc.ds.GetOptions()
	if err != nil {
		return nil, errors.Wrap(err, "get options from datastore")
	}

	return spec, nil
}
