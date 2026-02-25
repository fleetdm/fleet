package service

import (
	"context"
	"io"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/platform/endpointer"
	platform_http "github.com/fleetdm/fleet/v4/server/platform/http"
)

// Get App Store apps
func getAppStoreAppsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.GetAppStoreAppsRequest)
	apps, err := svc.GetAppStoreApps(ctx, &req.TeamID)
	if err != nil {
		return &fleet.GetAppStoreAppsResponse{Err: err}, nil
	}

	return &fleet.GetAppStoreAppsResponse{AppStoreApps: apps}, nil
}

func (svc *Service) GetAppStoreApps(ctx context.Context, teamID *uint) ([]*fleet.VPPApp, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}

// Add App Store apps
func addAppStoreAppEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.AddAppStoreAppRequest)
	titleID, err := svc.AddAppStoreApp(ctx, req.TeamID, fleet.VPPAppTeam{
		VPPAppID:             fleet.VPPAppID{AdamID: req.AppStoreID, Platform: req.Platform},
		SelfService:          req.SelfService,
		LabelsIncludeAny:     req.LabelsIncludeAny,
		LabelsExcludeAny:     req.LabelsExcludeAny,
		AddAutoInstallPolicy: req.AutomaticInstall,
		Categories:           req.Categories,
		Configuration:        req.Configuration,
	})
	if err != nil {
		return &fleet.AddAppStoreAppResponse{Err: err}, nil
	}

	return &fleet.AddAppStoreAppResponse{TitleID: titleID}, nil
}

func (svc *Service) AddAppStoreApp(ctx context.Context, _ *uint, _ fleet.VPPAppTeam) (uint, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return 0, fleet.ErrMissingLicense
}

// Update App Store apps
func updateAppStoreAppEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.UpdateAppStoreAppRequest)

	updatedApp, activity, err := svc.UpdateAppStoreApp(ctx, req.TitleID, req.TeamID, fleet.AppStoreAppUpdatePayload{
		SelfService:      req.SelfService,
		LabelsIncludeAny: req.LabelsIncludeAny,
		LabelsExcludeAny: req.LabelsExcludeAny,
		Categories:       req.Categories,
		Configuration:    req.Configuration,
		DisplayName:      req.DisplayName,
		SoftwareAutoUpdateConfig: fleet.SoftwareAutoUpdateConfig{
			AutoUpdateEnabled:   req.AutoUpdateEnabled,
			AutoUpdateStartTime: req.AutoUpdateStartTime,
			AutoUpdateEndTime:   req.AutoUpdateEndTime,
		},
	})
	if err != nil {
		return fleet.UpdateAppStoreAppResponse{Err: err}, nil
	}

	if req.AutoUpdateEnabled != nil {
		// Update AutoUpdateConfig separately
		err = svc.UpdateSoftwareTitleAutoUpdateConfig(ctx, req.TitleID, req.TeamID, fleet.SoftwareAutoUpdateConfig{
			AutoUpdateEnabled:   req.AutoUpdateEnabled,
			AutoUpdateStartTime: req.AutoUpdateStartTime,
			AutoUpdateEndTime:   req.AutoUpdateEndTime,
		})
		if err != nil {
			return fleet.UpdateAppStoreAppResponse{Err: err}, nil
		}
	}

	// Re-fetch the software title to get the updated auto-update config.
	updatedTitle, err := svc.SoftwareTitleByID(ctx, req.TitleID, req.TeamID)
	if err != nil {
		return fleet.UpdateAppStoreAppResponse{Err: err}, nil
	}
	if updatedTitle.AutoUpdateEnabled != nil {
		activity.AutoUpdateEnabled = updatedTitle.AutoUpdateEnabled
		if *updatedTitle.AutoUpdateEnabled {
			activity.AutoUpdateStartTime = updatedTitle.AutoUpdateStartTime
			activity.AutoUpdateEndTime = updatedTitle.AutoUpdateEndTime
		}
	}

	if err := svc.NewActivity(ctx, authz.UserFromContext(ctx), activity); err != nil {
		return fleet.UpdateAppStoreAppResponse{Err: err}, nil
	}

	return fleet.UpdateAppStoreAppResponse{AppStoreApp: updatedApp}, nil
}

func (svc *Service) UpdateAppStoreApp(ctx context.Context, titleID uint, teamID *uint, payload fleet.AppStoreAppUpdatePayload) (*fleet.VPPAppStoreApp, *fleet.ActivityEditedAppStoreApp, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, nil, fleet.ErrMissingLicense
}

// POST /api/_version_/vpp_tokens
type decodeUploadVPPTokenRequest struct{}

func (decodeUploadVPPTokenRequest) DecodeRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	decoded := fleet.UploadVPPTokenRequest{}

	err := r.ParseMultipartForm(platform_http.MaxMultipartFormSize)
	if err != nil {
		return nil, &fleet.BadRequestError{
			Message:     "failed to parse multipart form",
			InternalErr: err,
		}
	}

	if r.MultipartForm.File["token"] == nil || len(r.MultipartForm.File["token"]) == 0 {
		return nil, &fleet.BadRequestError{
			Message:     "token multipart field is required",
			InternalErr: err,
		}
	}

	decoded.File = r.MultipartForm.File["token"][0]

	return &decoded, nil
}

func uploadVPPTokenEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.UploadVPPTokenRequest)
	file, err := req.File.Open()
	if err != nil {
		return fleet.UploadVPPTokenResponse{Err: err}, nil
	}
	defer file.Close()

	tok, err := svc.UploadVPPToken(ctx, file)
	if err != nil {
		return fleet.UploadVPPTokenResponse{Err: err}, nil
	}

	return fleet.UploadVPPTokenResponse{Token: tok}, nil
}

func (svc *Service) UploadVPPToken(ctx context.Context, file io.ReadSeeker) (*fleet.VPPTokenDB, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}

// PATCH /api/_version_/fleet/vpp_tokens/%d/renew //
type decodePatchVPPTokenRenewRequest struct{}

func (decodePatchVPPTokenRenewRequest) DecodeRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	decoded := fleet.PatchVPPTokenRenewRequest{}

	err := r.ParseMultipartForm(platform_http.MaxMultipartFormSize)
	if err != nil {
		return nil, &fleet.BadRequestError{
			Message:     "failed to parse multipart form",
			InternalErr: err,
		}
	}

	if r.MultipartForm.File["token"] == nil || len(r.MultipartForm.File["token"]) == 0 {
		return nil, &fleet.BadRequestError{
			Message:     "token multipart field is required",
			InternalErr: err,
		}
	}

	decoded.File = r.MultipartForm.File["token"][0]

	id, err := endpointer.UintFromRequest(r, "id")
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "failed to parse vpp token id")
	}

	decoded.ID = uint(id) //nolint:gosec // dismiss G115

	return &decoded, nil
}

func patchVPPTokenRenewEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.PatchVPPTokenRenewRequest)
	file, err := req.File.Open()
	if err != nil {
		return fleet.PatchVPPTokenRenewResponse{Err: err}, nil
	}
	defer file.Close()

	tok, err := svc.UpdateVPPToken(ctx, req.ID, file)
	if err != nil {
		return fleet.PatchVPPTokenRenewResponse{Err: err}, nil
	}

	return fleet.PatchVPPTokenRenewResponse{Token: tok}, nil
}

func (svc *Service) UpdateVPPToken(ctx context.Context, tokenID uint, token io.ReadSeeker) (*fleet.VPPTokenDB, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}

// PATCH /api/_version_/fleet/vpp_tokens/%d/teams //
func patchVPPTokensTeams(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.PatchVPPTokensTeamsRequest)

	tok, err := svc.UpdateVPPTokenTeams(ctx, req.ID, req.TeamIDs)
	if err != nil {
		return fleet.PatchVPPTokensTeamsResponse{Err: err}, nil
	}
	return fleet.PatchVPPTokensTeamsResponse{Token: tok}, nil
}

func (svc *Service) UpdateVPPTokenTeams(ctx context.Context, tokenID uint, teamIDs []uint) (*fleet.VPPTokenDB, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}

// GET /api/_version_/fleet/vpp_tokens //
func getVPPTokens(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	tokens, err := svc.GetVPPTokens(ctx)
	if err != nil {
		return fleet.GetVPPTokensResponse{Err: err}, nil
	}

	if tokens == nil {
		tokens = []*fleet.VPPTokenDB{}
	}

	return fleet.GetVPPTokensResponse{Tokens: tokens}, nil
}

func (svc *Service) GetVPPTokens(ctx context.Context) ([]*fleet.VPPTokenDB, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}

// DELETE /api/_version_/fleet/vpp_tokens/%d //
func deleteVPPToken(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.DeleteVPPTokenRequest)

	err := svc.DeleteVPPToken(ctx, req.ID)
	if err != nil {
		return fleet.DeleteVPPTokenResponse{Err: err}, nil
	}

	return fleet.DeleteVPPTokenResponse{}, nil
}

func (svc *Service) DeleteVPPToken(ctx context.Context, tokenID uint) error {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return fleet.ErrMissingLicense
}
