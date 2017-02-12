package service

import (
	"time"

	"github.com/kolide/kolide/server/kolide"
	"golang.org/x/net/context"
)

func (mw loggingMiddleware) ModifyLabel(ctx context.Context, id uint, p kolide.ModifyLabelPayload) (*kolide.Label, error) {
	var (
		label *kolide.Label
		err   error
	)

	defer func(begin time.Time) {
		mw.logger.Log(
			"method", "ModifyLabel",
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	label, err = mw.Service.ModifyLabel(ctx, id, p)
	return label, err
}

func (mw loggingMiddleware) ListLabels(ctx context.Context, opt kolide.ListOptions) ([]*kolide.Label, error) {
	var (
		labels []*kolide.Label
		err    error
	)

	defer func(begin time.Time) {
		_ = mw.logger.Log(
			"method", "ListLabels",
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	labels, err = mw.Service.ListLabels(ctx, opt)
	return labels, err
}

func (mw loggingMiddleware) GetLabel(ctx context.Context, id uint) (*kolide.Label, error) {
	var (
		label *kolide.Label
		err   error
	)

	defer func(begin time.Time) {
		_ = mw.logger.Log(
			"method", "GetLabel",
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	label, err = mw.Service.GetLabel(ctx, id)
	return label, err
}

func (mw loggingMiddleware) NewLabel(ctx context.Context, p kolide.LabelPayload) (*kolide.Label, error) {
	var (
		label *kolide.Label
		err   error
	)

	defer func(begin time.Time) {
		_ = mw.logger.Log(
			"method", "NewLabel",
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	label, err = mw.Service.NewLabel(ctx, p)
	return label, err
}

func (mw loggingMiddleware) DeleteLabel(ctx context.Context, id uint) error {
	var (
		err error
	)

	defer func(begin time.Time) {
		_ = mw.logger.Log(
			"method", "DeleteLabel",
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	err = mw.Service.DeleteLabel(ctx, id)
	return err
}
