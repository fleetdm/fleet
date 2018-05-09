package service

import (
	"context"
	"time"

	"github.com/kolide/fleet/server/kolide"
)

func (mw loggingMiddleware) GetQuerySpec(ctx context.Context, name string) (spec *kolide.QuerySpec, err error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "GetQuerySpec",
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())
	spec, err = mw.Service.GetQuerySpec(ctx, name)
	return spec, err
}

func (mw loggingMiddleware) GetQuerySpecs(ctx context.Context) (specs []*kolide.QuerySpec, err error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "GetQuerySpecs",
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())
	specs, err = mw.Service.GetQuerySpecs(ctx)
	return specs, err
}

func (mw loggingMiddleware) ApplyQuerySpecs(ctx context.Context, specs []*kolide.QuerySpec) (err error) {
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "ApplyQuerySpecs",
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())
	err = mw.Service.ApplyQuerySpecs(ctx, specs)
	return err
}
