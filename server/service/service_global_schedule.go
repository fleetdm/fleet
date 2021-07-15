package service

import (
	"context"
	"database/sql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/pkg/errors"
)

func (svc *Service) GlobalScheduleQuery(ctx context.Context, sq *fleet.ScheduledQuery) (*fleet.ScheduledQuery, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Pack{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	gp, err := svc.ds.EnsureGlobalPack()
	if err != nil {
		return nil, err
	}
	sq.PackID = gp.ID

	return svc.ScheduleQuery(ctx, sq)
}

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

func (svc *Service) ModifyGlobalScheduledQueries(
	ctx context.Context,
	queryID uint,
	query fleet.ScheduledQueryPayload,
) (*fleet.ScheduledQuery, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Pack{}, fleet.ActionWrite); err != nil {
		return nil, err
	}

	gp, err := svc.ds.EnsureGlobalPack()
	if err != nil {
		return nil, err
	}

	q, err := svc.ds.Query(queryID)
	if err != nil {
		return nil, err
	}

	sq, err := svc.ds.ScheduledQueryByQueryAndPack(queryID, gp.ID)
	if err != nil {
		if errors.Cause(err) != sql.ErrNoRows {
			return nil, err
		}
		sq, err = svc.ds.NewScheduledQuery(&fleet.ScheduledQuery{
			PackID:      gp.ID,
			Name:        q.Name,
			QueryID:     q.ID,
			QueryName:   q.Name,
			Query:       q.Query,
			Description: q.Description,
		})
		if err != nil {
			return nil, err
		}
	}

	return svc.ModifyScheduledQuery(ctx, sq.ID, query)
}

func (svc *Service) DeleteGlobalScheduledQueries(ctx context.Context, id uint) error {
	if err := svc.authz.Authorize(ctx, &fleet.Pack{}, fleet.ActionWrite); err != nil {
		return err
	}

	return svc.DeleteScheduledQuery(ctx, id)
}
