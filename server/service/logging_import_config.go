package service

import (
	"context"
	"time"

	"github.com/kolide/kolide/server/kolide"
)

func (mw loggingMiddleware) ImportConfig(ctx context.Context, cfg *kolide.ImportConfig) (*kolide.ImportConfigResponse, error) {
	var (
		resp *kolide.ImportConfigResponse
		err  error
	)

	defer func(begin time.Time) {
		_ = mw.logger.Log(
			"method", "ImportConfig",
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	resp, err = mw.Service.ImportConfig(ctx, cfg)
	return resp, err

}
