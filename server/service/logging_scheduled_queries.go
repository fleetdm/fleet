package service

import (
	"context"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

func (mw loggingMiddleware) GetScheduledQueriesInPack(ctx context.Context, id uint, opts fleet.ListOptions) ([]*fleet.ScheduledQuery, error) {
	var (
		queries []*fleet.ScheduledQuery
		err     error
	)

	defer func(begin time.Time) {
		_ = mw.loggerDebug(err).Log(
			"method", "GetScheduledQueriesInPack",
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	queries, err = mw.Service.GetScheduledQueriesInPack(ctx, id, opts)
	return queries, err
}

func (mw loggingMiddleware) GetScheduledQuery(ctx context.Context, id uint) (*fleet.ScheduledQuery, error) {
	var (
		query *fleet.ScheduledQuery
		err   error
	)

	defer func(begin time.Time) {
		_ = mw.loggerDebug(err).Log(
			"method", "GetScheduledQuery",
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	query, err = mw.Service.GetScheduledQuery(ctx, id)
	return query, err
}

//these ones too
func (mw loggingMiddleware) ScheduleQuery(ctx context.Context, sq *fleet.ScheduledQuery) (*fleet.ScheduledQuery, error) {
	var (
		query        *fleet.ScheduledQuery
		err          error
		loggedInUser = "unauthenticated"
	)

	if vc, ok := viewer.FromContext(ctx); ok {

		loggedInUser = vc.Email()
	}

	defer func(begin time.Time) {
		_ = mw.loggerInfo(err).Log(
			"method", "ScheduleQuery",
			"err", err,
			"user", loggedInUser,
			"took", time.Since(begin),
		)
	}(time.Now())

	query, err = mw.Service.ScheduleQuery(ctx, sq)
	return query, err
}

func (mw loggingMiddleware) DeleteScheduledQuery(ctx context.Context, id uint) error {
	var (
		err          error
		loggedInUser = "unauthenticated"
	)

	if vc, ok := viewer.FromContext(ctx); ok {

		loggedInUser = vc.Email()
	}

	defer func(begin time.Time) {
		_ = mw.loggerInfo(err).Log(
			"method", "DeleteScheduledQuery",
			"err", err,
			"user", loggedInUser,
			"took", time.Since(begin),
		)
	}(time.Now())

	err = mw.Service.DeleteScheduledQuery(ctx, id)
	return err
}

func (mw loggingMiddleware) ModifyScheduledQuery(ctx context.Context, id uint, p fleet.ScheduledQueryPayload) (*fleet.ScheduledQuery, error) {
	var (
		query        *fleet.ScheduledQuery
		err          error
		loggedInUser = "unauthenticated"
	)

	if vc, ok := viewer.FromContext(ctx); ok {
		loggedInUser = vc.Email()
	}

	defer func(begin time.Time) {
		_ = mw.loggerInfo(err).Log(
			"method", "ModifyScheduledQuery",
			"err", err,
			"user", loggedInUser,
			"took", time.Since(begin),
		)
	}(time.Now())

	query, err = mw.Service.ModifyScheduledQuery(ctx, id, p)
	return query, err
}
