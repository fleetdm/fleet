package service

import (
	"context"

	"github.com/pkg/errors"
)

func (svc service) StatusResultStore(ctx context.Context) error {
	return svc.resultStore.HealthCheck()
}

func (svc service) StatusLiveQuery(ctx context.Context) error {
	cfg, err := svc.AppConfig(ctx)
	if err != nil {
		return errors.Wrap(err, "retreiving app config")
	}

	if cfg.LiveQueryDisabled {
		return errors.New("disabled by administrator")
	}

	return svc.StatusResultStore(ctx)
}
