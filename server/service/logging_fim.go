package service

import (
	"context"
	"time"

	"github.com/kolide/fleet/server/kolide"
)

func (lm loggingMiddleware) GetFIM(ctx context.Context) (cfg *kolide.FIMConfig, err error) {
	defer func(begin time.Time) {
		lm.logger.Log(
			"method", "GetFIM",
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())
	cfg, err = lm.Service.GetFIM(ctx)
	return cfg, err
}

func (lm loggingMiddleware) ModifyFIM(ctx context.Context, fim kolide.FIMConfig) (err error) {
	defer func(begin time.Time) {
		lm.logger.Log(
			"method", "ModifyFIM",
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())
	err = lm.Service.ModifyFIM(ctx, fim)
	return err
}
