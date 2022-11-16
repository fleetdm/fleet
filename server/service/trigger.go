package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

type getTriggerRequest struct {
	Name string `query:"name"`
}

type getTriggerResponse struct {
	Err error `json:"error,omitempty"`
}

func (r getTriggerResponse) error() error { return r.Err }

func triggerEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	_, err := svc.AuthenticatedUser(ctx)
	if err != nil {
		return getTriggerResponse{Err: err}, nil
	}
	req := request.(*getTriggerRequest)

	err = svc.TriggerCronSchedule(ctx, req.Name)
	if err != nil {
		return getTriggerResponse{Err: err}, nil
	}

	return getTriggerResponse{}, nil
}
