package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
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

	cfg, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "retrieve app config")
	}

	if cfg.ServerSettings.LiveQueryDisabled {
		return ctxerr.New(ctx, "disabled by administrator")
	}

	return svc.StatusResultStore(ctx)
}
