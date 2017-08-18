package service

import (
	"context"
	"fmt"
	"time"

	"github.com/kolide/fleet/server/kolide"
)

func (mw metricsMiddleware) GetFIM(ctx context.Context) (cfg *kolide.FIMConfig, err error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "GetFIM", "error", fmt.Sprint(err != nil)}
		mw.requestCount.With(lvs...).Add(1)
		mw.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())
	cfg, err = mw.Service.GetFIM(ctx)
	return cfg, err
}

func (mw metricsMiddleware) ModifyFIM(ctx context.Context, fim kolide.FIMConfig) (err error) {
	defer func(begin time.Time) {
		lvs := []string{"method", "ModifyFIM", "error", fmt.Sprint(err != nil)}
		mw.requestCount.With(lvs...).Add(1)
		mw.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())
	err = mw.Service.ModifyFIM(ctx, fim)
	return err
}
