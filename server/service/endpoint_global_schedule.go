package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/kit/endpoint"
)

////////////////////////////////////////////////////////////////////////////////
// Get Global Schedule
////////////////////////////////////////////////////////////////////////////////

type getGlobalScheduleRequest struct {
	ListOptions fleet.ListOptions
}

type getGlobalScheduleResponse struct {
	GlobalSchedule []*fleet.ScheduledQuery `json:"global_schedule"`
	Err            error                   `json:"error,omitempty"`
}

func (r getGlobalScheduleResponse) error() error { return r.Err }

func makeGetGlobalScheduleEndpoint(svc fleet.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getGlobalScheduleRequest)

		gp, err := svc.GetGlobalScheduledQueries(ctx, req.ListOptions)
		if err != nil {
			return getGlobalScheduleResponse{Err: err}, nil
		}

		return getGlobalScheduleResponse{
			GlobalSchedule: gp,
		}, nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// Modify Global Schedule
////////////////////////////////////////////////////////////////////////////////

type modifyGlobalScheduleRequest struct {
	ID      uint
	payload fleet.ScheduledQueryPayload
}

type modifyGlobalScheduleResponse struct {
	Scheduled *fleet.ScheduledQuery `json:"scheduled,omitempty"`
	Err       error                 `json:"error,omitempty"`
}

func (r modifyGlobalScheduleResponse) error() error { return r.Err }

func makeModifyGlobalScheduleEndpoint(svc fleet.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(modifyGlobalScheduleRequest)

		sq, err := svc.ModifyGlobalScheduledQueries(ctx, req.ID, req.payload)
		if err != nil {
			return modifyGlobalScheduleResponse{Err: err}, nil
		}

		return modifyGlobalScheduleResponse{
			Scheduled: sq,
		}, nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// Delete Global Schedule
////////////////////////////////////////////////////////////////////////////////

type deleteGlobalScheduleRequest struct {
	ID uint
}

type deleteGlobalScheduleResponse struct {
	Err error `json:"error,omitempty"`
}

func (r deleteGlobalScheduleResponse) error() error { return r.Err }

func makeDeleteGlobalScheduleEndpoint(svc fleet.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(deleteGlobalScheduleRequest)
		err := svc.DeleteGlobalScheduledQueries(ctx, req.ID)
		if err != nil {
			return deleteGlobalScheduleResponse{Err: err}, nil
		}

		return deleteGlobalScheduleResponse{}, nil
	}
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

func makeGlobalScheduleQueryEndpoint(svc fleet.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(globalScheduleQueryRequest)

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
}
