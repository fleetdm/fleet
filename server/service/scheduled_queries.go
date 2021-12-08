package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

////////////////////////////////////////////////////////////////////////////////
// Get Scheduled Queries In Pack
////////////////////////////////////////////////////////////////////////////////

type getScheduledQueriesInPackRequest struct {
	ID uint `url:"id"`
	// TODO(mna): was not set in the old pattern
	ListOptions fleet.ListOptions `url:"list_options"`
}

type scheduledQueryResponse struct {
	fleet.ScheduledQuery
}

type getScheduledQueriesInPackResponse struct {
	Scheduled []scheduledQueryResponse `json:"scheduled"`
	Err       error                    `json:"error,omitempty"`
}

func (r getScheduledQueriesInPackResponse) error() error { return r.Err }

func getScheduledQueriesInPackEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*getScheduledQueriesInPackRequest)
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

func (svc *Service) GetScheduledQueriesInPack(ctx context.Context, id uint, opts fleet.ListOptions) ([]*fleet.ScheduledQuery, error) {
	// Scheduled queries are currently authorized the same as packs.
	if err := svc.authz.Authorize(ctx, &fleet.Pack{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	return svc.ds.ListScheduledQueriesInPack(ctx, id, opts)
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

func scheduleQueryEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*scheduleQueryRequest)

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

func (svc *Service) ScheduleQuery(ctx context.Context, sq *fleet.ScheduledQuery) (*fleet.ScheduledQuery, error) {
	// Scheduled queries are currently authorized the same as packs.
	if err := svc.authz.Authorize(ctx, &fleet.Pack{}, fleet.ActionWrite); err != nil {
		return nil, err
	}

	return svc.unauthorizedScheduleQuery(ctx, sq)
}

////////////////////////////////////////////////////////////////////////////////
// Get Scheduled Query
////////////////////////////////////////////////////////////////////////////////

type getScheduledQueryRequest struct {
	ID uint `url:"id"`
}

type getScheduledQueryResponse struct {
	Scheduled *scheduledQueryResponse `json:"scheduled,omitempty"`
	Err       error                   `json:"error,omitempty"`
}

func (r getScheduledQueryResponse) error() error { return r.Err }

func getScheduledQueryEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*getScheduledQueryRequest)

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

func (svc *Service) GetScheduledQuery(ctx context.Context, id uint) (*fleet.ScheduledQuery, error) {
	// Scheduled queries are currently authorized the same as packs.
	if err := svc.authz.Authorize(ctx, &fleet.Pack{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	return svc.ds.ScheduledQuery(ctx, id)
}
