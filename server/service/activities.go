package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

////////////////////////////////////////////////////////////////////////////////
// Get activities
////////////////////////////////////////////////////////////////////////////////

type listActivitiesRequest struct {
	ListOptions fleet.ListOptions `url:"list_options"`
}

type listActivitiesResponse struct {
	Activities []*fleet.Activity `json:"activities"`
	Err        error             `json:"error,omitempty"`
}

func (r listActivitiesResponse) error() error { return r.Err }

func listActivitiesEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*listActivitiesRequest)
	activities, err := svc.ListActivities(ctx, req.ListOptions)
	if err != nil {
		return listActivitiesResponse{Err: err}, nil
	}

	return listActivitiesResponse{Activities: activities}, nil
}

// ListActivities returns a slice of activities for the whole organization
func (svc *Service) ListActivities(ctx context.Context, opt fleet.ListOptions) ([]*fleet.Activity, error) {
	if err := svc.authz.Authorize(ctx, &fleet.Activity{}, fleet.ActionRead); err != nil {
		return nil, err
	}
	return svc.ds.ListActivities(ctx, opt)
}
