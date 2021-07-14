package service

import (
	"context"
	"database/sql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/pkg/errors"
)

func (svc *Service) GetGlobalScheduledQueries(ctx context.Context, opts fleet.ListOptions) ([]*fleet.ScheduledQuery, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Pack{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	gp, err := svc.ds.EnsureGlobalPack()
	if err != nil {
		return nil, err
	}

	return svc.ds.ListScheduledQueriesInPack(gp.ID, opts)
}

func (svc *Service) ModifyGlobalScheduledQueries(ctx context.Context, queries []fleet.GlobalScheduleQueryPayload) ([]*fleet.ScheduledQuery, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Pack{}, fleet.ActionWrite); err != nil {
		return nil, err
	}

	gp, err := svc.ds.EnsureGlobalPack()
	if err != nil {
		return nil, err
	}

	var sqs []*fleet.ScheduledQuery

	for _, p := range queries {
		if p.QueryID == nil {
			return nil, fleet.NewError(
				fleet.ErrNoQueryIDNeeded,
				"All queries need an ID to modify the global schedule",
			)
		}

		q, err := svc.ds.Query(*p.QueryID)
		if err != nil {
			return nil, err
		}

		if p.Interval == nil {

		}
		sq, err := svc.ds.ScheduledQueryByQueryID(*p.QueryID)
		if err != nil {
			if errors.Cause(err) != sql.ErrNoRows {
				return nil, err
			}
			sq = &fleet.ScheduledQuery{
				PackID:      gp.ID,
				Name:        q.Name,
				QueryID:     q.ID,
				QueryName:   q.Name,
				Query:       q.Query,
				Description: q.Description,
			}
		}

		sq.PackID = gp.ID

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
		sqs = append(sqs, sq)
	}

	return svc.ds.ReplaceScheduledQueriesInPack(gp.ID, sqs)
}

func (svc *Service) DeleteGlobalScheduledQueries(ctx context.Context) error {
	if err := svc.authz.Authorize(ctx, &fleet.Pack{}, fleet.ActionWrite); err != nil {
		return err
	}

	gp, err := svc.ds.EnsureGlobalPack()
	if err != nil {
		return err
	}

	sqs, err := svc.ds.ListScheduledQueriesInPack(gp.ID, fleet.ListOptions{})
	if err != nil {
		return err
	}

	var ids []uint
	for _, sq := range sqs {
		ids = append(ids, sq.ID)
	}
	return svc.ds.DeleteScheduledQueries(ids)
}
