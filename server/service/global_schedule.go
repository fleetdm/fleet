package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
)

// Get Global Schedule
func getGlobalScheduleEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.GetGlobalScheduleRequest)

	gp, err := svc.GetGlobalScheduledQueries(ctx, req.ListOptions)
	if err != nil {
		return fleet.GetGlobalScheduleResponse{Err: err}, nil
	}

	return fleet.GetGlobalScheduleResponse{
		GlobalSchedule: gp,
	}, nil
}

func (svc *Service) GetGlobalScheduledQueries(ctx context.Context, opts fleet.ListOptions) ([]*fleet.ScheduledQuery, error) {
	queries, _, _, _, err := svc.ListQueries(ctx, opts, nil, ptr.Bool(true), false, nil) // teamID == nil means global
	if err != nil {
		return nil, err
	}
	scheduledQueries := make([]*fleet.ScheduledQuery, 0, len(queries))
	for _, query := range queries {
		scheduledQueries = append(scheduledQueries, fleet.ScheduledQueryFromQuery(query))
	}
	return scheduledQueries, nil
}

// Schedule a global query
func globalScheduleQueryEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.GlobalScheduleQueryRequest)

	scheduled, err := svc.GlobalScheduleQuery(ctx, &fleet.ScheduledQuery{
		QueryID:  req.QueryID,
		Interval: req.Interval,
		Snapshot: req.Snapshot,
		Removed:  req.Removed,
		Platform: req.Platform,
		Version:  req.Version,
		Shard:    req.Shard,
	})
	if err != nil {
		return fleet.GlobalScheduleQueryResponse{Err: err}, nil
	}
	return fleet.GlobalScheduleQueryResponse{Scheduled: scheduled}, nil
}

func (svc *Service) GlobalScheduleQuery(ctx context.Context, scheduledQuery *fleet.ScheduledQuery) (*fleet.ScheduledQuery, error) {
	originalQuery, err := svc.ds.Query(ctx, scheduledQuery.QueryID)
	if err != nil {
		setAuthCheckedOnPreAuthErr(ctx)
		return nil, ctxerr.Wrap(ctx, err, "get query")
	}
	if originalQuery.TeamID != nil {
		setAuthCheckedOnPreAuthErr(ctx)
		return nil, ctxerr.New(ctx, "cannot create a global schedule from a team query")
	}
	originalQuery.Name = nameForCopiedQuery(originalQuery.Name)
	newQuery, err := svc.NewQuery(ctx, fleet.ScheduledQueryToQueryPayloadForNewQuery(originalQuery, scheduledQuery))
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "create new query")
	}
	return fleet.ScheduledQueryFromQuery(newQuery), nil
}

// Modify Global Schedule
func modifyGlobalScheduleEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.ModifyGlobalScheduleRequest)

	sq, err := svc.ModifyGlobalScheduledQueries(ctx, req.ID, req.ScheduledQueryPayload)
	if err != nil {
		return fleet.ModifyGlobalScheduleResponse{Err: err}, nil
	}

	return fleet.ModifyGlobalScheduleResponse{
		Scheduled: sq,
	}, nil
}

func (svc *Service) ModifyGlobalScheduledQueries(ctx context.Context, id uint, scheduledQueryPayload fleet.ScheduledQueryPayload) (*fleet.ScheduledQuery, error) {
	query, err := svc.ModifyQuery(ctx, id, fleet.ScheduledQueryPayloadToQueryPayloadForModifyQuery(scheduledQueryPayload))
	if err != nil {
		return nil, err
	}
	return fleet.ScheduledQueryFromQuery(query), nil
}

// Delete Global Schedule
func deleteGlobalScheduleEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.DeleteGlobalScheduleRequest)
	err := svc.DeleteGlobalScheduledQueries(ctx, req.ID)
	if err != nil {
		return fleet.DeleteGlobalScheduleResponse{Err: err}, nil
	}

	return fleet.DeleteGlobalScheduleResponse{}, nil
}

func (svc *Service) DeleteGlobalScheduledQueries(ctx context.Context, id uint) error {
	return svc.DeleteQueryByID(ctx, id)
}
