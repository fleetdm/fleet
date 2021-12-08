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

// Scheduled queries are currently authorized the same as packs.

func (svc *Service) GetScheduledQueriesInPack(ctx context.Context, id uint, opts fleet.ListOptions) ([]*fleet.ScheduledQuery, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Pack{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	return svc.ds.ListScheduledQueriesInPack(ctx, id, opts)
}
