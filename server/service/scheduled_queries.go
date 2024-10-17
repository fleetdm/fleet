package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////
// All API endpoints in this file are used for 2017 packs functionality.
////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////

////////////////////////////////////////////////////////////////////////////////
// Get Scheduled Queries In Pack
////////////////////////////////////////////////////////////////////////////////

type getScheduledQueriesInPackRequest struct {
	ID          uint              `url:"id"`
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

func getScheduledQueriesInPackEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
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
	if err := svc.authz.Authorize(ctx, &fleet.Pack{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	return svc.ds.ListScheduledQueriesInPackWithStats(ctx, id, opts)
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

func scheduleQueryEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
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

func (svc *Service) unauthorizedScheduleQuery(ctx context.Context, sq *fleet.ScheduledQuery) (*fleet.ScheduledQuery, error) {
	if sq.Interval < 1 || sq.Interval > fleet.MaxScheduledQueryInterval {
		return nil, ctxerr.Wrap(ctx, &fleet.BadRequestError{
			Message: "invalid scheduled query interval",
		})
	}

	// Fill in the name with query name if it is unset (because the UI
	// doesn't provide a way to set it)
	if sq.Name == "" {
		query, err := svc.ds.Query(ctx, sq.QueryID)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "lookup name for query")
		}

		packQueries, err := svc.ds.ListScheduledQueriesInPackWithStats(ctx, sq.PackID, fleet.ListOptions{})
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "find existing scheduled queries")
		}

		sq.Name = findNextNameForQuery(query.Name, packQueries)
		sq.QueryName = query.Name
	} else if sq.QueryName == "" {
		query, err := svc.ds.Query(ctx, sq.QueryID)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "lookup name for query")
		}
		sq.QueryName = query.Name
	}

	return svc.ds.NewScheduledQuery(ctx, sq)
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

func getScheduledQueryEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
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

////////////////////////////////////////////////////////////////////////////////
// Modify Scheduled Query
////////////////////////////////////////////////////////////////////////////////

type modifyScheduledQueryRequest struct {
	ID uint `json:"-" url:"id"`
	fleet.ScheduledQueryPayload
}

type modifyScheduledQueryResponse struct {
	Scheduled *scheduledQueryResponse `json:"scheduled,omitempty"`
	Err       error                   `json:"error,omitempty"`
}

func (r modifyScheduledQueryResponse) error() error { return r.Err }

func modifyScheduledQueryEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*modifyScheduledQueryRequest)

	sq, err := svc.ModifyScheduledQuery(ctx, req.ID, req.ScheduledQueryPayload)
	if err != nil {
		return modifyScheduledQueryResponse{Err: err}, nil
	}

	return modifyScheduledQueryResponse{
		Scheduled: &scheduledQueryResponse{
			ScheduledQuery: *sq,
		},
	}, nil
}

func (svc *Service) ModifyScheduledQuery(ctx context.Context, id uint, p fleet.ScheduledQueryPayload) (*fleet.ScheduledQuery, error) {
	// Scheduled queries are currently authorized the same as packs.
	if err := svc.authz.Authorize(ctx, &fleet.Pack{}, fleet.ActionWrite); err != nil {
		return nil, err
	}

	return svc.unauthorizedModifyScheduledQuery(ctx, id, p)
}

func (svc *Service) unauthorizedModifyScheduledQuery(ctx context.Context, id uint, p fleet.ScheduledQueryPayload) (*fleet.ScheduledQuery, error) {
	if p.Interval != nil {
		if *p.Interval < 1 || *p.Interval > fleet.MaxScheduledQueryInterval {
			return nil, ctxerr.New(ctx, "invalid scheduled query interval")
		}
	}

	sq, err := svc.ds.ScheduledQuery(ctx, id)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting scheduled query to modify")
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
			val := uint(p.Shard.Int64) //nolint:gosec // dismiss G115
			sq.Shard = &val
		} else {
			sq.Shard = nil
		}
	}

	return svc.ds.SaveScheduledQuery(ctx, sq)
}

////////////////////////////////////////////////////////////////////////////////
// Delete Scheduled Query
////////////////////////////////////////////////////////////////////////////////

type deleteScheduledQueryRequest struct {
	ID uint `url:"id"`
}

type deleteScheduledQueryResponse struct {
	Err error `json:"error,omitempty"`
}

func (r deleteScheduledQueryResponse) error() error { return r.Err }

func deleteScheduledQueryEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*deleteScheduledQueryRequest)

	err := svc.DeleteScheduledQuery(ctx, req.ID)
	if err != nil {
		return deleteScheduledQueryResponse{Err: err}, nil
	}

	return deleteScheduledQueryResponse{}, nil
}

func (svc *Service) DeleteScheduledQuery(ctx context.Context, id uint) error {
	// Scheduled queries are currently authorized the same as packs.
	if err := svc.authz.Authorize(ctx, &fleet.Pack{}, fleet.ActionWrite); err != nil {
		return err
	}

	return svc.ds.DeleteScheduledQuery(ctx, id)
}
