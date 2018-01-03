package service

import (
	"context"

	"github.com/kolide/fleet/server/kolide"
)

func (svc service) GetScheduledQueriesInPack(ctx context.Context, id uint, opts kolide.ListOptions) ([]*kolide.ScheduledQuery, error) {
	return svc.ds.ListScheduledQueriesInPack(id, opts)
}
