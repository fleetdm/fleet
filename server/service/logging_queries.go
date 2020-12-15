package service

import (
	"context"
	"time"

	"github.com/fleetdm/fleet/server/contexts/viewer"
	"github.com/fleetdm/fleet/server/kolide"
)

func (mw loggingMiddleware) GetQuerySpec(ctx context.Context, name string) (spec *kolide.QuerySpec, err error) {
	defer func(begin time.Time) {
		_ = mw.loggerDebug(err).Log(
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
		_ = mw.loggerDebug(err).Log(
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
		_ = mw.loggerDebug(err).Log(
			"method", "ApplyQuerySpecs",
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())
	err = mw.Service.ApplyQuerySpecs(ctx, specs)
	return err
}

func (mw loggingMiddleware) ListQueries(ctx context.Context, opt kolide.ListOptions) ([]*kolide.Query, error) {
	var (
		loggedInUser = "unauthenticated"
		err          error
	)
	if vc, ok := viewer.FromContext(ctx); ok {

		loggedInUser = vc.Username()
	}
	defer func(begin time.Time) {
		_ = mw.loggerDebug(err).Log(
			"method", "ListQueries",
			"err", err,
			"user", loggedInUser,
			"took", time.Since(begin),
		)
	}(time.Now())
	query, err := mw.Service.ListQueries(ctx, opt)
	return query, err
}
func (mw loggingMiddleware) GetQuery(ctx context.Context, id uint) (*kolide.Query, error) {
	var (
		loggedInUser = "unauthenticated"
		err          error
	)
	if vc, ok := viewer.FromContext(ctx); ok {

		loggedInUser = vc.Username()
	}
	defer func(begin time.Time) {
		_ = mw.loggerDebug(err).Log(
			"method", "GetQuery",
			"err", err,
			"user", loggedInUser,
			"took", time.Since(begin),
		)
	}(time.Now())
	query, err := mw.Service.GetQuery(ctx, id)
	return query, err
}
func (mw loggingMiddleware) NewQuery(ctx context.Context, p kolide.QueryPayload) (*kolide.Query, error) {
	var (
		query        *kolide.Query
		loggedInUser = "unauthenticated"
		err          error
	)
	if vc, ok := viewer.FromContext(ctx); ok {

		loggedInUser = vc.Username()
	}
	defer func(begin time.Time) {
		if query == nil {
			_ = mw.loggerInfo(err).Log(
				"method", "NewQuery",
				"err", err,
				"user", loggedInUser,
				"took", time.Since(begin),
			)
			return
		}
		_ = mw.loggerInfo(err).Log(
			"method", "NewQuery",
			"err", err,
			"user", loggedInUser,
			"name", query.Name,
			"sql", query.Query,
			"took", time.Since(begin),
		)
	}(time.Now())
	query, err = mw.Service.NewQuery(ctx, p)
	return query, err
}
func (mw loggingMiddleware) ModifyQuery(ctx context.Context, id uint, p kolide.QueryPayload) (*kolide.Query, error) {
	var (
		query        *kolide.Query
		loggedInUser = "unauthenticated"
		err          error
	)
	if vc, ok := viewer.FromContext(ctx); ok {

		loggedInUser = vc.Username()
	}
	defer func(begin time.Time) {
		if query == nil {
			_ = mw.loggerInfo(err).Log(
				"method", "ModifyQuery",
				"err", err,
				"user", loggedInUser,
				"took", time.Since(begin),
			)
			return
		}
		_ = mw.loggerInfo(err).Log(
			"method", "ModifyQuery",
			"err", err,
			"user", loggedInUser,
			"name", query.Name,
			"sql", query.Query,
			"took", time.Since(begin),
		)
	}(time.Now())
	query, err = mw.Service.ModifyQuery(ctx, id, p)
	return query, err
}
func (mw loggingMiddleware) DeleteQuery(ctx context.Context, name string) error {
	var (
		loggedInUser = "unauthenticated"
		err          error
	)
	if vc, ok := viewer.FromContext(ctx); ok {

		loggedInUser = vc.Username()
	}
	defer func(begin time.Time) {
		_ = mw.loggerInfo(err).Log(
			"method", "DeleteQuery",
			"err", err,
			"name", name,
			"user", loggedInUser,
			"took", time.Since(begin),
		)
	}(time.Now())
	err = mw.Service.DeleteQuery(ctx, name)
	return err
}
