package service

import (
	"context"
	"fmt"
	"time"

	"github.com/kolide/kolide/server/kolide"
)

func (mw metricsMiddleware) ImportConfig(ctx context.Context, cfg *kolide.ImportConfig) (*kolide.ImportConfigResponse, error) {
	var (
		resp *kolide.ImportConfigResponse
		err  error
	)
	defer func(begin time.Time) {
		lvs := []string{"method", "ImportConfig", "error", fmt.Sprint(err != nil)}
		mw.requestCount.With(lvs...).Add(1)
		mw.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())
	resp, err = mw.Service.ImportConfig(ctx, cfg)
	return resp, err
}
