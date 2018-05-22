package service

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/kolide/fleet/server/kolide"
)

////////////////////////////////////////////////////////////////////////////////
// Get Scheduled Queries In Pack
////////////////////////////////////////////////////////////////////////////////

type getScheduledQueriesInPackRequest struct {
	ID          uint
	ListOptions kolide.ListOptions
}

type scheduledQueryResponse struct {
	kolide.ScheduledQuery
}

type getScheduledQueriesInPackResponse struct {
	Scheduled []scheduledQueryResponse `json:"scheduled"`
	Err       error                    `json:"error,omitempty"`
}

func (r getScheduledQueriesInPackResponse) error() error { return r.Err }

func makeGetScheduledQueriesInPackEndpoint(svc kolide.Service) endpoint.Endpoint {
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
