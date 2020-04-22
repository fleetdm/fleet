package service

import (
	"context"
	"time"

	"github.com/kolide/fleet/server/kolide"
)

func (mw loggingMiddleware) GetFIM(ctx context.Context) (cfg *kolide.FIMConfig, err error) {
	defer func(begin time.Time) {
		_ = mw.loggerDebug(err).Log(
			"method", "GetFIM",
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())
	cfg, err = mw.Service.GetFIM(ctx)
	return cfg, err
}

func (mw loggingMiddleware) ModifyFIM(ctx context.Context, fim kolide.FIMConfig) (err error) {
	defer func(begin time.Time) {
		_ = mw.loggerInfo(err).Log(
			"method", "ModifyFIM",
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())
	err = mw.Service.ModifyFIM(ctx, fim)
	return err
}
