package service

import (
	"context"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/kit/endpoint"
)

////////////////////////////////////////////////////////////////////////////////
// Get activities
////////////////////////////////////////////////////////////////////////////////

type listActivitiesRequest struct {
	ListOptions fleet.ListOptions
}

type listActivitiesResponse struct {
	Activities []*fleet.Activity `json:"activities"`
	Err        error             `json:"error,omitempty"`
}

func (r listActivitiesResponse) error() error { return r.Err }

func makeListActivitiesEndpoint(svc fleet.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(listActivitiesRequest)
		activities, err := svc.ListActivities(ctx, req.ListOptions)
		if err != nil {
			return listActivitiesResponse{Err: err}, nil
		}

		return listActivitiesResponse{Activities: activities}, err
	}
}
