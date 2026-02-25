package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

func triggerEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	_, err := svc.AuthenticatedUser(ctx)
	if err != nil {
		return fleet.TriggerResponse{Err: err}, nil
	}
	req := request.(*fleet.TriggerRequest)

	err = svc.TriggerCronSchedule(ctx, req.Name)
	if err != nil {
		return fleet.TriggerResponse{Err: err}, nil
	}

	return fleet.TriggerResponse{}, nil
}
