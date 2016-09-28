package service

import (
	"fmt"
	"time"

	"github.com/kolide/kolide-ose/server/kolide"
	"golang.org/x/net/context"
)

func (mw metricsMiddleware) NewOrgInfo(ctx context.Context, p kolide.OrgInfoPayload) (*kolide.OrgInfo, error) {
	var (
		info *kolide.OrgInfo
		err  error
	)
	defer func(begin time.Time) {
		lvs := []string{"method", "NewOrgInfo", "error", fmt.Sprint(err != nil)}
		mw.requestCount.With(lvs...).Add(1)
		mw.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())
	info, err = mw.Service.NewOrgInfo(ctx, p)
	return info, err
}

func (mw metricsMiddleware) OrgInfo(ctx context.Context) (*kolide.OrgInfo, error) {
	var (
		info *kolide.OrgInfo
		err  error
	)
	defer func(begin time.Time) {
		lvs := []string{"method", "OrgInfo", "error", fmt.Sprint(err != nil)}
		mw.requestCount.With(lvs...).Add(1)
		mw.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())
	info, err = mw.Service.OrgInfo(ctx)
	return info, err
}

func (mw metricsMiddleware) ModifyOrgInfo(ctx context.Context, p kolide.OrgInfoPayload) (*kolide.OrgInfo, error) {
	var (
		info *kolide.OrgInfo
		err  error
	)
	defer func(begin time.Time) {
		lvs := []string{"method", "ModifyOrgInfo", "error", fmt.Sprint(err != nil)}
		mw.requestCount.With(lvs...).Add(1)
		mw.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())
	info, err = mw.Service.ModifyOrgInfo(ctx, p)
	return info, err
}
