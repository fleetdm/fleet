package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/kit/endpoint"
)

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
