package service

import (
	"context"
	"time"

	"github.com/kolide/kolide/server/kolide"
)

func (mw loggingMiddleware) ListDecorators(ctx context.Context) ([]*kolide.Decorator, error) {
	var (
		decs []*kolide.Decorator
		err  error
	)
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "ListDecorators",
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())
	decs, err = mw.Service.ListDecorators(ctx)
	return decs, err
}

func (mw loggingMiddleware) NewDecorator(ctx context.Context, payload kolide.DecoratorPayload) (*kolide.Decorator, error) {
	var (
		dec *kolide.Decorator
		err error
	)
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "NewDecorator",
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())
	dec, err = mw.Service.NewDecorator(ctx, payload)
	return dec, err
}

func (mw loggingMiddleware) ModifyDecorator(ctx context.Context, payload kolide.DecoratorPayload) (*kolide.Decorator, error) {
	var (
		dec *kolide.Decorator
		err error
	)
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "ModifyDecorator",
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())
	dec, err = mw.Service.ModifyDecorator(ctx, payload)
	return dec, err
}

func (mw loggingMiddleware) DeleteDecorator(ctx context.Context, id uint) error {
	var err error
	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "DeleteDecorator",
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())
	err = mw.Service.DeleteDecorator(ctx, id)
	return err
}
