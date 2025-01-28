package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
)

////////////////////////////////////////////////////////////////////////////////
// Get Global Schedule
////////////////////////////////////////////////////////////////////////////////

type getGlobalScheduleRequest struct {
	ListOptions fleet.ListOptions `url:"list_options"`
}

type getGlobalScheduleResponse struct {
	GlobalSchedule []*fleet.ScheduledQuery `json:"global_schedule"`
	Err            error                   `json:"error,omitempty"`
}

func (r getGlobalScheduleResponse) error() error { return r.Err }

func getGlobalScheduleEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*getGlobalScheduleRequest)

	gp, err := svc.GetGlobalScheduledQueries(ctx, req.ListOptions)
	if err != nil {
		return getGlobalScheduleResponse{Err: err}, nil
	}

	return getGlobalScheduleResponse{
		GlobalSchedule: gp,
	}, nil
}

func (svc *Service) GetGlobalScheduledQueries(ctx context.Context, opts fleet.ListOptions) ([]*fleet.ScheduledQuery, error) {
	queries, _, _, err := svc.ListQueries(ctx, opts, nil, ptr.Bool(true), false, nil) // teamID == nil means global
	if err != nil {
		return nil, err
	}
	scheduledQueries := make([]*fleet.ScheduledQuery, 0, len(queries))
	for _, query := range queries {
		scheduledQueries = append(scheduledQueries, fleet.ScheduledQueryFromQuery(query))
	}
	return scheduledQueries, nil
}

////////////////////////////////////////////////////////////////////////////////
// Schedule a global query
////////////////////////////////////////////////////////////////////////////////

type globalScheduleQueryRequest struct {
	QueryID  uint    `json:"query_id"`
	Interval uint    `json:"interval"`
	Snapshot *bool   `json:"snapshot"`
	Removed  *bool   `json:"removed"`
	Platform *string `json:"platform"`
	Version  *string `json:"version"`
	Shard    *uint   `json:"shard"`
}

type globalScheduleQueryResponse struct {
	Scheduled *fleet.ScheduledQuery `json:"scheduled,omitempty"`
	Err       error                 `json:"error,omitempty"`
}

func (r globalScheduleQueryResponse) error() error { return r.Err }

func globalScheduleQueryEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*globalScheduleQueryRequest)

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
		return globalScheduleQueryResponse{Err: err}, nil
	}
	return globalScheduleQueryResponse{Scheduled: scheduled}, nil
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

////////////////////////////////////////////////////////////////////////////////
// Modify Global Schedule
////////////////////////////////////////////////////////////////////////////////

type modifyGlobalScheduleRequest struct {
	ID uint `json:"-" url:"id"`
	fleet.ScheduledQueryPayload
}

type modifyGlobalScheduleResponse struct {
	Scheduled *fleet.ScheduledQuery `json:"scheduled,omitempty"`
	Err       error                 `json:"error,omitempty"`
}

func (r modifyGlobalScheduleResponse) error() error { return r.Err }

func modifyGlobalScheduleEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*modifyGlobalScheduleRequest)

	sq, err := svc.ModifyGlobalScheduledQueries(ctx, req.ID, req.ScheduledQueryPayload)
	if err != nil {
		return modifyGlobalScheduleResponse{Err: err}, nil
	}

	return modifyGlobalScheduleResponse{
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

////////////////////////////////////////////////////////////////////////////////
// Delete Global Schedule
////////////////////////////////////////////////////////////////////////////////

type deleteGlobalScheduleRequest struct {
	ID uint `url:"id"`
}

type deleteGlobalScheduleResponse struct {
	Err error `json:"error,omitempty"`
}

func (r deleteGlobalScheduleResponse) error() error { return r.Err }

func deleteGlobalScheduleEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*deleteGlobalScheduleRequest)
	err := svc.DeleteGlobalScheduledQueries(ctx, req.ID)
	if err != nil {
		return deleteGlobalScheduleResponse{Err: err}, nil
	}

	return deleteGlobalScheduleResponse{}, nil
}

// TODO(lucas): Document new behavior.
func (svc *Service) DeleteGlobalScheduledQueries(ctx context.Context, id uint) error {
	return svc.DeleteQueryByID(ctx, id)
}
