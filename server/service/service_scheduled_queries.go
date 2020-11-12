package service

import (
	"context"

	"github.com/fleetdm/fleet/server/kolide"
	"github.com/pkg/errors"
)

func (svc service) GetScheduledQueriesInPack(ctx context.Context, id uint, opts kolide.ListOptions) ([]*kolide.ScheduledQuery, error) {
	return svc.ds.ListScheduledQueriesInPack(id, opts)
}

func (svc service) GetScheduledQuery(ctx context.Context, id uint) (*kolide.ScheduledQuery, error) {
	return svc.ds.ScheduledQuery(id)
}

func (svc service) ScheduleQuery(ctx context.Context, sq *kolide.ScheduledQuery) (*kolide.ScheduledQuery, error) {
	// Fill in the name with query name if it is unset (because the UI
	// doesn't provide a way to set it)
	if sq.Name == "" {
		query, err := svc.ds.Query(sq.QueryID)
		if err != nil {
			return nil, errors.Wrap(err, "lookup name for query")
		}
		sq.Name = query.Name
		sq.QueryName = query.Name
	}
	return svc.ds.NewScheduledQuery(sq)
}

func (svc service) ModifyScheduledQuery(ctx context.Context, id uint, p kolide.ScheduledQueryPayload) (*kolide.ScheduledQuery, error) {
	sq, err := svc.GetScheduledQuery(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "getting scheduled query to modify")
	}

	if p.PackID != nil {
		sq.PackID = *p.PackID
	}

	if p.QueryID != nil {
		sq.QueryID = *p.QueryID
	}

	if p.Interval != nil {
		sq.Interval = *p.Interval
	}

	if p.Snapshot != nil {
		sq.Snapshot = p.Snapshot
	}

	if p.Removed != nil {
		sq.Removed = p.Removed
	}

	if p.Platform != nil {
		sq.Platform = p.Platform
	}

	if p.Version != nil {
		sq.Version = p.Version
	}

	if p.Shard != nil {
		if p.Shard.Valid {
			val := uint(p.Shard.Int64)
			sq.Shard = &val
		} else {
			sq.Shard = nil
		}
	}

	return svc.ds.SaveScheduledQuery(sq)
}

func (svc service) DeleteScheduledQuery(ctx context.Context, id uint) error {
	return svc.ds.DeleteScheduledQuery(id)
}
