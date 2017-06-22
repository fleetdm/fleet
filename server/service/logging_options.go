package service

import (
	"context"
	"time"

	"github.com/kolide/fleet/server/kolide"
)

func (mw loggingMiddleware) GetOptions(ctx context.Context) ([]kolide.Option, error) {
	var (
		options []kolide.Option
		err     error
	)

	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "GetOptions",
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())
	options, err = mw.Service.GetOptions(ctx)
	return options, err
}

func (mw loggingMiddleware) ModifyOptions(ctx context.Context, req kolide.OptionRequest) ([]kolide.Option, error) {
	var (
		options []kolide.Option
		err     error
	)

	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "ModifyOptions",
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())
	options, err = mw.Service.ModifyOptions(ctx, req)
	return options, err
}

func (mw loggingMiddleware) ResetOptions(ctx context.Context) ([]kolide.Option, error) {
	var (
		options []kolide.Option
		err     error
	)
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "ResetOptions",
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())
	options, err = mw.Service.ResetOptions(ctx)
	return options, err
}
