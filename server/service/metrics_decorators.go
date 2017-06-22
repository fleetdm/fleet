package service

import (
	"context"
	"fmt"
	"time"

	"github.com/kolide/fleet/server/kolide"
)

func (mw metricsMiddleware) ListDecorators(ctx context.Context) ([]*kolide.Decorator, error) {
	var (
		decs []*kolide.Decorator
		err  error
	)
	defer func(begin time.Time) {
		lvs := []string{"method", "ListDecorators", "error", fmt.Sprint(err != nil)}
		mw.requestCount.With(lvs...).Add(1)
		mw.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())
	decs, err = mw.Service.ListDecorators(ctx)
	return decs, err
}

func (mw metricsMiddleware) NewDecorator(ctx context.Context, payload kolide.DecoratorPayload) (*kolide.Decorator, error) {
	var (
		dec *kolide.Decorator
		err error
	)
	defer func(begin time.Time) {
		lvs := []string{"method", "NewDecorator", "error", fmt.Sprint(err != nil)}
		mw.requestCount.With(lvs...).Add(1)
		mw.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())
	dec, err = mw.Service.NewDecorator(ctx, payload)
	return dec, err
}

func (mw metricsMiddleware) ModifyDecorator(ctx context.Context, payload kolide.DecoratorPayload) (*kolide.Decorator, error) {
	var (
		dec *kolide.Decorator
		err error
	)
	defer func(begin time.Time) {
		lvs := []string{"method", "ModifyDecorator", "error", fmt.Sprint(err != nil)}
		mw.requestCount.With(lvs...).Add(1)
		mw.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())
	dec, err = mw.Service.ModifyDecorator(ctx, payload)
	return dec, err
}

func (mw metricsMiddleware) DeleteDecorator(ctx context.Context, id uint) error {
	var err error
	defer func(begin time.Time) {
		lvs := []string{"method", "DeleteDecorator", "error", fmt.Sprint(err != nil)}
		mw.requestCount.With(lvs...).Add(1)
		mw.requestLatency.With(lvs...).Observe(time.Since(begin).Seconds())
	}(time.Now())
	err = mw.Service.DeleteDecorator(ctx, id)
	return err
}
