package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/kit/endpoint"
)

////////////////////////////////////////////////////////////////////////////////
// Get Scheduled Queries In Pack
////////////////////////////////////////////////////////////////////////////////

type getScheduledQueriesInPackRequest struct {
	ID          uint
	ListOptions fleet.ListOptions
}

type scheduledQueryResponse struct {
	fleet.ScheduledQuery
}

type getScheduledQueriesInPackResponse struct {
	Scheduled []scheduledQueryResponse `json:"scheduled"`
	Err       error                    `json:"error,omitempty"`
}

func (r getScheduledQueriesInPackResponse) error() error { return r.Err }

func makeGetScheduledQueriesInPackEndpoint(svc fleet.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getScheduledQueriesInPackRequest)
		resp := getScheduledQueriesInPackResponse{Scheduled: []scheduledQueryResponse{}}

		queries, err := svc.GetScheduledQueriesInPack(ctx, req.ID, req.ListOptions)
		if err != nil {
			return getScheduledQueriesInPackResponse{Err: err}, nil
		}

		for _, q := range queries {
			resp.Scheduled = append(resp.Scheduled, scheduledQueryResponse{
				ScheduledQuery: *q,
			})
		}

		return resp, nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// Get Scheduled Query
////////////////////////////////////////////////////////////////////////////////

type getScheduledQueryRequest struct {
	ID uint
}

type getScheduledQueryResponse struct {
	Scheduled *scheduledQueryResponse `json:"scheduled,omitempty"`
	Err       error                   `json:"error,omitempty"`
}

func (r getScheduledQueryResponse) error() error { return r.Err }

func makeGetScheduledQueryEndpoint(svc fleet.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(getScheduledQueryRequest)

		sq, err := svc.GetScheduledQuery(ctx, req.ID)
		if err != nil {
			return getScheduledQueryResponse{Err: err}, nil
		}

		return getScheduledQueryResponse{
			Scheduled: &scheduledQueryResponse{
				ScheduledQuery: *sq,
			},
		}, nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// Schedule Query
////////////////////////////////////////////////////////////////////////////////

type scheduleQueryRequest struct {
	PackID   uint    `json:"pack_id"`
	QueryID  uint    `json:"query_id"`
	Interval uint    `json:"interval"`
	Snapshot *bool   `json:"snapshot"`
	Removed  *bool   `json:"removed"`
	Platform *string `json:"platform"`
	Version  *string `json:"version"`
	Shard    *uint   `json:"shard"`
}

type scheduleQueryResponse struct {
	Scheduled *scheduledQueryResponse `json:"scheduled,omitempty"`
	Err       error                   `json:"error,omitempty"`
}

func (r scheduleQueryResponse) error() error { return r.Err }

func makeScheduleQueryEndpoint(svc fleet.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(scheduleQueryRequest)

		scheduled, err := svc.ScheduleQuery(ctx, &fleet.ScheduledQuery{
			PackID:   req.PackID,
			QueryID:  req.QueryID,
			Interval: req.Interval,
			Snapshot: req.Snapshot,
			Removed:  req.Removed,
			Platform: req.Platform,
			Version:  req.Version,
			Shard:    req.Shard,
		})
		if err != nil {
			return scheduleQueryResponse{Err: err}, nil
		}
		return scheduleQueryResponse{Scheduled: &scheduledQueryResponse{
			ScheduledQuery: *scheduled,
		}}, nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// Modify Scheduled Query
////////////////////////////////////////////////////////////////////////////////

type modifyScheduledQueryRequest struct {
	ID      uint
	payload fleet.ScheduledQueryPayload
}

type modifyScheduledQueryResponse struct {
	Scheduled *scheduledQueryResponse `json:"scheduled,omitempty"`
	Err       error                   `json:"error,omitempty"`
}

func (r modifyScheduledQueryResponse) error() error { return r.Err }

func makeModifyScheduledQueryEndpoint(svc fleet.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(modifyScheduledQueryRequest)

		sq, err := svc.ModifyScheduledQuery(ctx, req.ID, req.payload)
		if err != nil {
			return modifyScheduledQueryResponse{Err: err}, nil
		}

		return modifyScheduledQueryResponse{
			Scheduled: &scheduledQueryResponse{
				ScheduledQuery: *sq,
			},
		}, nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// Delete Scheduled Query
////////////////////////////////////////////////////////////////////////////////

type deleteScheduledQueryRequest struct {
	ID uint
}

type deleteScheduledQueryResponse struct {
	Err error `json:"error,omitempty"`
}

func (r deleteScheduledQueryResponse) error() error { return r.Err }

func makeDeleteScheduledQueryEndpoint(svc fleet.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(deleteScheduledQueryRequest)

		err := svc.DeleteScheduledQuery(ctx, req.ID)
		if err != nil {
			return deleteScheduledQueryResponse{Err: err}, nil
		}

		return deleteScheduledQueryResponse{}, nil
	}
}
