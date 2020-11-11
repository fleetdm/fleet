package service

import (
	"context"
	"time"

	"github.com/fleetdm/fleet/server/contexts/viewer"
	"github.com/fleetdm/fleet/server/kolide"
)

func (mw loggingMiddleware) NewLabel(ctx context.Context, p kolide.LabelPayload) (*kolide.Label, error) {
	var (
		label        *kolide.Label
		err          error
		loggedInUser = "unauthenticated"
	)

	if vc, ok := viewer.FromContext(ctx); ok {

		loggedInUser = vc.Username()
	}

	defer func(begin time.Time) {
		_ = mw.loggerInfo(err).Log(
			"method", "NewLabel",
			"err", err,
			"user", loggedInUser,
			"took", time.Since(begin),
		)
	}(time.Now())

	label, err = mw.Service.NewLabel(ctx, p)
	return label, err
}

func (mw loggingMiddleware) ModifyLabel(ctx context.Context, id uint, p kolide.ModifyLabelPayload) (*kolide.Label, error) {
	var (
		label        *kolide.Label
		err          error
		loggedInUser = "unauthenticated"
	)

	if vc, ok := viewer.FromContext(ctx); ok {

		loggedInUser = vc.Username()
	}

	defer func(begin time.Time) {
		_ = mw.loggerInfo(err).Log(
			"method", "ModifyLabel",
			"err", err,
			"user", loggedInUser,
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
		_ = mw.loggerDebug(err).Log(
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
		_ = mw.loggerDebug(err).Log(
			"method", "GetLabel",
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	label, err = mw.Service.GetLabel(ctx, id)
	return label, err
}

func (mw loggingMiddleware) DeleteLabel(ctx context.Context, name string) error {
	var (
		err          error
		loggedInUser = "unauthenticated"
	)

	if vc, ok := viewer.FromContext(ctx); ok {

		loggedInUser = vc.Username()
	}

	defer func(begin time.Time) {
		_ = mw.loggerInfo(err).Log(
			"method", "DeleteLabel",
			"err", err,
			"user", loggedInUser,
			"took", time.Since(begin),
		)
	}(time.Now())

	err = mw.Service.DeleteLabel(ctx, name)
	return err
}

func (mw loggingMiddleware) GetLabelSpec(ctx context.Context, name string) (spec *kolide.LabelSpec, err error) {
	defer func(begin time.Time) {
		_ = mw.loggerDebug(err).Log(
			"method", "GetLabelSpec",
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())
	spec, err = mw.Service.GetLabelSpec(ctx, name)
	return spec, err
}

func (mw loggingMiddleware) GetLabelSpecs(ctx context.Context) (specs []*kolide.LabelSpec, err error) {
	defer func(begin time.Time) {
		_ = mw.loggerDebug(err).Log(
			"method", "GetLabelSpecs",
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())
	specs, err = mw.Service.GetLabelSpecs(ctx)
	return specs, err
}

func (mw loggingMiddleware) ApplyLabelSpecs(ctx context.Context, specs []*kolide.LabelSpec) (err error) {
	var (
		loggedInUser = "unauthenticated"
	)

	if vc, ok := viewer.FromContext(ctx); ok {

		loggedInUser = vc.Username()
	}

	defer func(begin time.Time) {
		_ = mw.loggerInfo(err).Log(
			"method", "ApplyLabelSpecs",
			"err", err,
			"user", loggedInUser,
			"took", time.Since(begin),
		)
	}(time.Now())
	err = mw.Service.ApplyLabelSpecs(ctx, specs)
	return err
}
