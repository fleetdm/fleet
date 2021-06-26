package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/pkg/errors"
)

// Scheduled queries are currently authorized the same as packs.

func (svc *Service) GetScheduledQueriesInPack(ctx context.Context, id uint, opts fleet.ListOptions) ([]*fleet.ScheduledQuery, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Pack{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	return svc.ds.ListScheduledQueriesInPack(id, opts)
}

func (svc *Service) GetScheduledQuery(ctx context.Context, id uint) (*fleet.ScheduledQuery, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Pack{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	return svc.ds.ScheduledQuery(id)
}

func (svc *Service) ScheduleQuery(ctx context.Context, sq *fleet.ScheduledQuery) (*fleet.ScheduledQuery, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Pack{}, fleet.ActionWrite); err != nil {
		return nil, err
	}

	// Fill in the name with query name if it is unset (because the UI
	// doesn't provide a way to set it)
	if sq.Name == "" {
		query, err := svc.ds.Query(sq.QueryID)
		if err != nil {
			return nil, errors.Wrap(err, "lookup name for query")
		}

		packQueries, err := svc.ds.ListScheduledQueriesInPack(sq.PackID, fleet.ListOptions{})
		if err != nil {
			return nil, errors.Wrap(err, "find existing scheduled queries")
		}
		_ = packQueries

		sq.Name = findNextNameForQuery(query.Name, packQueries)
		sq.QueryName = query.Name
	} else if sq.QueryName == "" {
		query, err := svc.ds.Query(sq.QueryID)
		if err != nil {
			return nil, errors.Wrap(err, "lookup name for query")
		}
		sq.QueryName = query.Name
	}
	return svc.ds.NewScheduledQuery(sq)
}

// Add "-1" suffixes to the query name until it is unique
func findNextNameForQuery(name string, scheduled []*fleet.ScheduledQuery) string {
	for _, q := range scheduled {
		if name == q.Name {
			return findNextNameForQuery(name+"-1", scheduled)
		}
	}
	return name
}

func (svc *Service) ModifyScheduledQuery(ctx context.Context, id uint, p fleet.ScheduledQueryPayload) (*fleet.ScheduledQuery, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Pack{}, fleet.ActionWrite); err != nil {
		return nil, err
	}

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

func (svc *Service) DeleteScheduledQuery(ctx context.Context, id uint) error {
	if err := svc.authz.Authorize(ctx, &fleet.Pack{}, fleet.ActionWrite); err != nil {
		return err
	}

	return svc.ds.DeleteScheduledQuery(id)
}
