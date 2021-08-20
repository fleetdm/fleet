package service

import (
	"context"
	"fmt"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

func (mw metricsMiddleware) NewAppConfig(ctx context.Context, p fleet.AppConfig) (*fleet.AppConfig, error) {
	var (
		info *fleet.AppConfig
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

func (mw metricsMiddleware) AppConfig(ctx context.Context) (*fleet.AppConfig, error) {
	var (
		info *fleet.AppConfig
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

func (mw metricsMiddleware) ModifyAppConfig(ctx context.Context, p []byte) (*fleet.AppConfig, error) {
	var (
		info *fleet.AppConfig
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
