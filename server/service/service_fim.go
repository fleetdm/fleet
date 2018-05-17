package service

import (
	"context"
	"encoding/json"

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

	var arr []string
	if len(config.FIMFileAccesses) > 0 {
		if err = json.Unmarshal([]byte(config.FIMFileAccesses), &arr); err != nil {
			return nil, errors.Wrap(err, "Error reading fim section, fileaccesses must be formatted as an array [\"cassandra\",\"etc\",\"homes\"]")
		}
	}

	result := &kolide.FIMConfig{
		Interval:     uint(config.FIMInterval),
		FilePaths:    paths,
		FileAccesses: arr,
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

	if len(fim.FileAccesses) > 0 {
		fileAccesses, err := json.Marshal(fim.FileAccesses)
		if err != nil {
			return errors.Wrap(err, "Error creating fim section, fileaccesses must be formatted as an array [\"cassandra\",\"etc\",\"homes\"]")
		}
		config.FIMFileAccesses = string(fileAccesses)
	}

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
