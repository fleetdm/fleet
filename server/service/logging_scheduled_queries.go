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
