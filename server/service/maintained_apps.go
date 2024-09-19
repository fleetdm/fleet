package service

import (
	"context"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

type addFleetMaintainedAppRequest struct {
	TeamID            *uint  `json:"team_id"`
	AppID             uint   `json:"fleet_maintained_app_id"`
	InstallScript     string `json:"install_script"`
	PreInstallQuery   string `json:"pre_install_query"`
	PostInstallScript string `json:"post_install_script"`
	SelfService       bool   `json:"self_service"`
}

type addFleetMaintainedAppResponse struct {
	Err error `json:"error,omitempty"`
}

func (r addFleetMaintainedAppResponse) error() error { return r.Err }

func addFleetMaintainedAppEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*addFleetMaintainedAppRequest)
	ctx, cancel := context.WithTimeout(ctx, 15*time.Minute)
	defer cancel()
	err := svc.AddFleetMaintainedApp(ctx, req.TeamID, req.AppID, req.InstallScript, req.PreInstallQuery, req.PostInstallScript, req.SelfService)
	if err != nil {
		return &addFleetMaintainedAppResponse{Err: err}, nil
	}
	return &addFleetMaintainedAppResponse{}, nil
}

func (svc *Service) AddFleetMaintainedApp(ctx context.Context, teamID *uint, appID uint, installScript, preInstallQuery, postInstallScript string, selfService bool) error {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return fleet.ErrMissingLicense
}
