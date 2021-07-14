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
	payload fleet.GlobalSchedulePayload
}

type modifyGlobalScheduleResponse struct {
	GlobalSchedule []*fleet.ScheduledQuery `json:"global_schedule"`
	Err            error                   `json:"error,omitempty"`
}

func (r modifyGlobalScheduleResponse) error() error { return r.Err }

func makeModifyGlobalScheduleEndpoint(svc fleet.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(modifyGlobalScheduleRequest)

		sqs, err := svc.ModifyGlobalScheduledQueries(ctx, req.payload.GlobalSchedule)
		if err != nil {
			return modifyGlobalScheduleResponse{Err: err}, nil
		}

		return modifyGlobalScheduleResponse{
			GlobalSchedule: sqs,
		}, nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// Delete Global Schedule
////////////////////////////////////////////////////////////////////////////////

type deleteGlobalScheduleRequest struct{}

type deleteGlobalScheduleResponse struct {
	Err error `json:"error,omitempty"`
}

func (r deleteGlobalScheduleResponse) error() error { return r.Err }

func makeDeleteGlobalScheduleEndpoint(svc fleet.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		err := svc.DeleteGlobalScheduledQueries(ctx)
		if err != nil {
			return deleteGlobalScheduleResponse{Err: err}, nil
		}

		return deleteGlobalScheduleResponse{}, nil
	}
}
