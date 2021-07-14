package service

import (
	"context"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"time"
)

func (mw loggingMiddleware) GetGlobalScheduledQueries(ctx context.Context, opts fleet.ListOptions) ([]*fleet.ScheduledQuery, error) {
	var (
		err          error
		loggedInUser = "unauthenticated"
	)

	if vc, ok := viewer.FromContext(ctx); ok {
		loggedInUser = vc.Email()
	}

	defer func(begin time.Time) {
		_ = mw.loggerInfo(err).Log(
			"method", "GetGlobalScheduledQueries",
			"err", err,
			"user", loggedInUser,
			"took", time.Since(begin),
		)
	}(time.Now())

	return mw.Service.GetGlobalScheduledQueries(ctx, opts)
}

func (mw loggingMiddleware) ModifyGlobalScheduledQueries(ctx context.Context, id uint, q fleet.ScheduledQueryPayload) (*fleet.ScheduledQuery, error) {
	var (
		err          error
		loggedInUser = "unauthenticated"
	)

	if vc, ok := viewer.FromContext(ctx); ok {
		loggedInUser = vc.Email()
	}

	defer func(begin time.Time) {
		_ = mw.loggerInfo(err).Log(
			"method", "ModifyGlobalScheduledQueries",
			"err", err,
			"user", loggedInUser,
			"took", time.Since(begin),
		)
	}(time.Now())

	return mw.Service.ModifyGlobalScheduledQueries(ctx, id, q)
}

func (mw loggingMiddleware) DeleteGlobalScheduledQueries(ctx context.Context, id uint) error {
	var (
		err          error
		loggedInUser = "unauthenticated"
	)

	if vc, ok := viewer.FromContext(ctx); ok {
		loggedInUser = vc.Email()
	}

	defer func(begin time.Time) {
		_ = mw.loggerInfo(err).Log(
			"method", "DeleteGlobalScheduledQueries",
			"err", err,
			"user", loggedInUser,
			"took", time.Since(begin),
		)
	}(time.Now())

	return mw.Service.DeleteGlobalScheduledQueries(ctx, id)
}

func (mw loggingMiddleware) GlobalScheduleQuery(ctx context.Context, sq *fleet.ScheduledQuery) (*fleet.ScheduledQuery, error) {
	var (
		err          error
		loggedInUser = "unauthenticated"
	)

	if vc, ok := viewer.FromContext(ctx); ok {
		loggedInUser = vc.Email()
	}

	defer func(begin time.Time) {
		_ = mw.loggerInfo(err).Log(
			"method", "GlobalScheduleQuery",
			"err", err,
			"user", loggedInUser,
			"took", time.Since(begin),
		)
	}(time.Now())

	return mw.Service.GlobalScheduleQuery(ctx, sq)
}
