package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

type addFleetMaintainedAppRequest struct {
	TeamID *uint `json:"team_id"`
	AppID  uint  `json:"fleet_maintained_app_id"`
}

type addFleetMaintainedAppResponse struct {
	Err error `json:"error,omitempty"`
}

func (r addFleetMaintainedAppResponse) error() error { return r.Err }

func addFleetMaintainedAppEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*addFleetMaintainedAppRequest)
	err := svc.AddFleetMaintainedApp(ctx, req.TeamID, req.AppID)
	if err != nil {
		return &addFleetMaintainedAppResponse{Err: err}, nil
	}
	return &addFleetMaintainedAppResponse{}, nil
}

func (svc *Service) AddFleetMaintainedApp(ctx context.Context, teamID *uint, appID uint) error {
	if err := svc.authz.Authorize(ctx, &fleet.SoftwareInstaller{TeamID: teamID}, fleet.ActionWrite); err != nil {
		return err
	}

	return nil
}
