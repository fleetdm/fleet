package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
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

func getGlobalScheduleEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
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
	if err := svc.authz.Authorize(ctx, &fleet.Pack{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	gp, err := svc.ds.EnsureGlobalPack(ctx)
	if err != nil {
		return nil, err
	}

	return svc.ds.ListScheduledQueriesInPack(ctx, gp.ID, opts)
}

////////////////////////////////////////////////////////////////////////////////
// Global Schedule Query
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

func globalScheduleQueryEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
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

func (svc *Service) GlobalScheduleQuery(ctx context.Context, sq *fleet.ScheduledQuery) (*fleet.ScheduledQuery, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Pack{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	gp, err := svc.ds.EnsureGlobalPack(ctx)
	if err != nil {
		return nil, err
	}
	sq.PackID = gp.ID

	return svc.ScheduleQuery(ctx, sq)
}
