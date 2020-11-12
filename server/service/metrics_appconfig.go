package service

import (
	"context"
	"fmt"
	"time"

	"github.com/fleetdm/fleet/server/kolide"
)

func (mw metricsMiddleware) NewAppConfig(ctx context.Context, p kolide.AppConfigPayload) (*kolide.AppConfig, error) {
	var (
		info *kolide.AppConfig
		err  error
	)
	defer func(begin time.Time) {
		lvs := []string{"method", "NewOrgInfo", "error", fmt.Sprint(err != nil)}
		mw.requestCount.With(lvs...).Add(1)
		mw.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())
	info, err = mw.Service.NewAppConfig(ctx, p)
	return info, err
}

func (mw metricsMiddleware) AppConfig(ctx context.Context) (*kolide.AppConfig, error) {
	var (
		info *kolide.AppConfig
		err  error
	)
	defer func(begin time.Time) {
		lvs := []string{"method", "OrgInfo", "error", fmt.Sprint(err != nil)}
		mw.requestCount.With(lvs...).Add(1)
		mw.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())
	info, err = mw.Service.AppConfig(ctx)
	return info, err
}

func (mw metricsMiddleware) ModifyAppConfig(ctx context.Context, p kolide.AppConfigPayload) (*kolide.AppConfig, error) {
	var (
		info *kolide.AppConfig
		err  error
	)
	defer func(begin time.Time) {
		lvs := []string{"method", "ModifyOrgInfo", "error", fmt.Sprint(err != nil)}
		mw.requestCount.With(lvs...).Add(1)
		mw.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())
	info, err = mw.Service.ModifyAppConfig(ctx, p)
	return info, err
}
