package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/maintainedapps"
)

// DecodeRequest implements the RequestDecoder interface to support base64-encoded
// script fields. This allows bypassing WAF rules that may block requests containing
// shell/PowerShell script patterns. When the X-Fleet-Scripts-Encoded header is set
// to "base64", the script fields are decoded from base64.
type decodeAddFleetMaintainedAppRequest struct{}

func (decodeAddFleetMaintainedAppRequest) DecodeRequest(ctx context.Context, r *http.Request) (any, error) {
	var req fleet.AddFleetMaintainedAppRequest

	// Decode JSON body
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, &fleet.BadRequestError{
			Message:     "failed to decode request body",
			InternalErr: err,
		}
	}

	// Check if scripts are base64 encoded
	if isScriptsEncoded(r) {
		var err error
		if req.InstallScript, err = decodeBase64Script(req.InstallScript); err != nil {
			return nil, fleet.NewInvalidArgumentError("install_script", "invalid base64 encoding")
		}
		if req.UninstallScript, err = decodeBase64Script(req.UninstallScript); err != nil {
			return nil, fleet.NewInvalidArgumentError("uninstall_script", "invalid base64 encoding")
		}
		if req.PostInstallScript, err = decodeBase64Script(req.PostInstallScript); err != nil {
			return nil, fleet.NewInvalidArgumentError("post_install_script", "invalid base64 encoding")
		}
		if req.PreInstallQuery, err = decodeBase64Script(req.PreInstallQuery); err != nil {
			return nil, fleet.NewInvalidArgumentError("pre_install_query", "invalid base64 encoding")
		}
	}

	return &req, nil
}

func addFleetMaintainedAppEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.AddFleetMaintainedAppRequest)
	ctx, cancel := context.WithTimeout(ctx, maintained_apps.InstallerTimeout)
	defer cancel()
	titleId, err := svc.AddFleetMaintainedApp(
		ctx,
		req.TeamID,
		req.AppID,
		req.InstallScript,
		req.PreInstallQuery,
		req.PostInstallScript,
		req.UninstallScript,
		req.SelfService,
		req.AutomaticInstall,
		req.LabelsIncludeAny,
		req.LabelsExcludeAny,
	)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			err = fleet.NewGatewayTimeoutError("Couldn't add. Request timeout. Please make sure your server and load balancer timeout is long enough.", err)
		}

		return &fleet.AddFleetMaintainedAppResponse{Err: err}, nil
	}
	return &fleet.AddFleetMaintainedAppResponse{SoftwareTitleID: titleId}, nil
}

func (svc *Service) AddFleetMaintainedApp(ctx context.Context, _ *uint, _ uint, _, _, _, _ string, _ bool, _ bool, _, _ []string) (uint, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return 0, fleet.ErrMissingLicense
}

func listFleetMaintainedAppsEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.ListFleetMaintainedAppsRequest)

	apps, meta, err := svc.ListFleetMaintainedApps(ctx, req.TeamID, req.ListOptions)
	if err != nil {
		return fleet.ListFleetMaintainedAppsResponse{Err: err}, nil
	}

	listResp := fleet.ListFleetMaintainedAppsResponse{
		FleetMaintainedApps: apps,
		Meta:                meta,
	}

	return listResp, nil
}

func (svc *Service) ListFleetMaintainedApps(ctx context.Context, teamID *uint, opts fleet.ListOptions) ([]fleet.MaintainedApp, *fleet.PaginationMetadata, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, nil, fleet.ErrMissingLicense
}

func getFleetMaintainedApp(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.GetFleetMaintainedAppRequest)

	app, err := svc.GetFleetMaintainedApp(ctx, req.AppID, req.TeamID)
	if err != nil {
		return fleet.GetFleetMaintainedAppResponse{Err: err}, nil
	}

	return fleet.GetFleetMaintainedAppResponse{FleetMaintainedApp: app}, nil
}

func (svc *Service) GetFleetMaintainedApp(ctx context.Context, appID uint, teamID *uint) (*fleet.MaintainedApp, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}
