package service

import (
	"context"
	"fmt"
	"time"

	"github.com/kolide/fleet/server/kolide"
)

func (mw metricsMiddleware) SaveLicense(ctx context.Context, jwtToken string) (*kolide.License, error) {
	var (
		lic *kolide.License
		err error
	)
	defer func(begin time.Time) {
		lvs := []string{"method", "SaveLicense", "error", fmt.Sprint(err != nil)}
		mw.requestCount.With(lvs...).Add(1)
		mw.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())
	lic, err = mw.Service.SaveLicense(ctx, jwtToken)
	return lic, err
}

func (mw metricsMiddleware) License(ctx context.Context) (*kolide.License, error) {
	var (
		lic *kolide.License
		err error
	)
	defer func(begin time.Time) {
		lvs := []string{"method", "License", "error", fmt.Sprint(err != nil)}
		mw.requestCount.With(lvs...).Add(1)
		mw.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())
	lic, err = mw.Service.License(ctx)
	return lic, err

}
