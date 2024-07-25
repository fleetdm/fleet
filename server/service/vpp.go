package service

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

//////////////////////////////////////////////////////////////////////////////
// Get App Store apps
//////////////////////////////////////////////////////////////////////////////

type getAppStoreAppsRequest struct {
	TeamID uint `query:"team_id"`
}

type getAppStoreAppsResponse struct {
	AppStoreApps []*fleet.VPPApp `json:"app_store_apps"`
	Err          error           `json:"error,omitempty"`
}

func (r getAppStoreAppsResponse) error() error { return r.Err }

func getAppStoreAppsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*getAppStoreAppsRequest)
	apps, err := svc.GetAppStoreApps(ctx, &req.TeamID)
	if err != nil {
		return &getAppStoreAppsResponse{Err: err}, nil
	}

	return &getAppStoreAppsResponse{AppStoreApps: apps}, nil
}

func (svc *Service) GetAppStoreApps(ctx context.Context, teamID *uint) ([]*fleet.VPPApp, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}

//////////////////////////////////////////////////////////////////////////////
// Add App Store apps
//////////////////////////////////////////////////////////////////////////////

type addAppStoreAppRequest struct {
	TeamID     *uint  `json:"team_id"`
	AppStoreID string `json:"app_store_id"`
}

type addAppStoreAppResponse struct {
	Err error `json:"error,omitempty"`
}

func (r addAppStoreAppResponse) error() error { return r.Err }

func addAppStoreAppEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*addAppStoreAppRequest)
	err := svc.AddAppStoreApp(ctx, req.TeamID, req.AppStoreID)
	if err != nil {
		return &addAppStoreAppResponse{Err: err}, nil
	}

	return &addAppStoreAppResponse{}, nil
}

func (svc *Service) AddAppStoreApp(ctx context.Context, teamID *uint, adamID string) error {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return fleet.ErrMissingLicense
}
