package service

import (
	"context"
	"time"

	"github.com/kolide/fleet/server/kolide"
)

func (mw loggingMiddleware) GetScheduledQueriesInPack(ctx context.Context, id uint, opts kolide.ListOptions) ([]*kolide.ScheduledQuery, error) {
	var (
		queries []*kolide.ScheduledQuery
		err     error
	)

	defer func(begin time.Time) {
		_ = mw.logger.Log(
			"method", "GetScheduledQueriesInPack",
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	queries, err = mw.Service.GetScheduledQueriesInPack(ctx, id, opts)
	return queries, err
}

func (mw loggingMiddleware) GetScheduledQuery(ctx context.Context, id uint) (*kolide.ScheduledQuery, error) {
	var (
		query *kolide.ScheduledQuery
		err   error
	)

	defer func(begin time.Time) {
		_ = mw.logger.Log(
			"method", "GetScheduledQuery",
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	query, err = mw.Service.GetScheduledQuery(ctx, id)
	return query, err
}

func (mw loggingMiddleware) ScheduleQuery(ctx context.Context, sq *kolide.ScheduledQuery) (*kolide.ScheduledQuery, error) {
	var (
		query *kolide.ScheduledQuery
		err   error
	)

	defer func(begin time.Time) {
		_ = mw.logger.Log(
			"method", "ScheduleQuery",
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	query, err = mw.Service.ScheduleQuery(ctx, sq)
	return query, err
}

func (mw loggingMiddleware) DeleteScheduledQuery(ctx context.Context, id uint) error {
	var (
		err error
	)

	defer func(begin time.Time) {
		_ = mw.logger.Log(
			"method", "DeleteScheduledQuery",
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	err = mw.Service.DeleteScheduledQuery(ctx, id)
	return err
}

func (mw loggingMiddleware) ModifyScheduledQuery(ctx context.Context, id uint, p kolide.ScheduledQueryPayload) (*kolide.ScheduledQuery, error) {
	var (
		query *kolide.ScheduledQuery
		err   error
	)

	defer func(begin time.Time) {
		_ = mw.logger.Log(
			"method", "ModifyScheduledQuery",
			"err", err,
			"took", time.Since(begin),
		)
	}(time.Now())

	query, err = mw.Service.ModifyScheduledQuery(ctx, id, p)
	return query, err
}
