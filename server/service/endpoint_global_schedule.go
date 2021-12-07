package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/kit/endpoint"
)

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
