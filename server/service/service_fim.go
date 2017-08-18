package service

import (
	"context"

	"github.com/kolide/fleet/server/kolide"
	"github.com/pkg/errors"
)

func (svc service) GetFIM(ctx context.Context) (*kolide.FIMConfig, error) {
	config, err := svc.ds.AppConfig()
	if err != nil {
		return nil, errors.Wrap(err, "getting fim config")
	}
	paths, err := svc.ds.FIMSections()
	if err != nil {
		return nil, errors.Wrap(err, "getting fim paths")
	}
	result := &kolide.FIMConfig{
		Interval:  uint(config.FIMInterval),
		FilePaths: paths,
	}
	return result, nil
}

// ModifyFIM will remove existing FIM settings and replace it
func (svc service) ModifyFIM(ctx context.Context, fim kolide.FIMConfig) error {
	if err := svc.ds.ClearFIMSections(); err != nil {
		return errors.Wrap(err, "updating fim")
	}
	config, err := svc.ds.AppConfig()
	if err != nil {
		return errors.Wrap(err, "updating fim")
	}
	config.FIMInterval = int(fim.Interval)
	for sectionName, paths := range fim.FilePaths {
		section := kolide.FIMSection{
			SectionName: sectionName,
			Paths:       paths,
		}
		if _, err := svc.ds.NewFIMSection(&section); err != nil {
			return errors.Wrap(err, "creating fim section")
		}
	}
	return svc.ds.SaveAppConfig(config)
}
