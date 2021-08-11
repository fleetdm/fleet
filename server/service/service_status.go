package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/pkg/errors"
)

func (svc *Service) StatusResultStore(ctx context.Context) error {
	if err := svc.authz.Authorize(ctx, &fleet.AppConfig{}, fleet.ActionRead); err != nil {
		return err
	}

	return svc.resultStore.HealthCheck()
}

func (svc *Service) StatusLiveQuery(ctx context.Context) error {
	if err := svc.authz.Authorize(ctx, &fleet.AppConfig{}, fleet.ActionRead); err != nil {
		return err
	}

	cfg, err := svc.ds.AppConfig()
	if err != nil {
		return errors.Wrap(err, "retrieve app config")
	}

	if cfg.LiveQueryDisabled {
		return errors.New("disabled by administrator")
	}

	return svc.StatusResultStore(ctx)
}
