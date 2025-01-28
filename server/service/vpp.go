package service

import (
	"context"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/docker/go-units"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
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
	TeamID           *uint                     `json:"team_id"`
	AppStoreID       string                    `json:"app_store_id"`
	Platform         fleet.AppleDevicePlatform `json:"platform"`
	SelfService      bool                      `json:"self_service"`
	LabelsIncludeAny []string                  `json:"labels_include_any"`
	LabelsExcludeAny []string                  `json:"labels_exclude_any"`
}

type addAppStoreAppResponse struct {
	Err error `json:"error,omitempty"`
}

func (r addAppStoreAppResponse) error() error { return r.Err }

func addAppStoreAppEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*addAppStoreAppRequest)
	err := svc.AddAppStoreApp(ctx, req.TeamID, fleet.VPPAppTeam{
		VPPAppID:         fleet.VPPAppID{AdamID: req.AppStoreID, Platform: req.Platform},
		SelfService:      req.SelfService,
		LabelsIncludeAny: req.LabelsIncludeAny,
		LabelsExcludeAny: req.LabelsExcludeAny,
	})
	if err != nil {
		return &addAppStoreAppResponse{Err: err}, nil
	}

	return &addAppStoreAppResponse{}, nil
}

func (svc *Service) AddAppStoreApp(ctx context.Context, _ *uint, _ fleet.VPPAppTeam) error {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return fleet.ErrMissingLicense
}

type updateAppStoreAppRequest struct {
	TeamID           *uint    `json:"team_id"`
	SelfService      bool     `json:"self_service"`
	LabelsIncludeAny []string `json:"labels_include_any"`
	LabelsExcludeAny []string `json:"labels_exclude_any"`
}

type updateAppStoreAppResponse struct {
	Err error `json:"error,omitempty"`
}

func (r updateAppStoreAppResponse) error() error { return r.Err }

func updateAppStoreAppEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*updateAppStoreAppRequest)

	if err := svc.UpdateAppStoreApp(ctx, req.TeamID); err != nil {
		return updateAppStoreAppResponse{Err: err}, nil
	}

	return updateAppStoreAppResponse{}, nil
}

func (svc *Service) UpdateAppStoreApp(ctx context.Context, teamID *uint) error {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return fleet.ErrMissingLicense
}

////////////////////////////////////////////////////////////////////////////////
// POST /api/_version_/vpp_tokens
////////////////////////////////////////////////////////////////////////////////

type uploadVPPTokenRequest struct {
	File *multipart.FileHeader
}

func (uploadVPPTokenRequest) DecodeRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	decoded := uploadVPPTokenRequest{}

	err := r.ParseMultipartForm(512 * units.MiB)
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

type uploadVPPTokenResponse struct {
	Err   error             `json:"error,omitempty"`
	Token *fleet.VPPTokenDB `json:"token,omitempty"`
}

func (r uploadVPPTokenResponse) Status() int { return http.StatusAccepted }

func (r uploadVPPTokenResponse) error() error {
	return r.Err
}

func uploadVPPTokenEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*uploadVPPTokenRequest)
	file, err := req.File.Open()
	if err != nil {
		return uploadVPPTokenResponse{Err: err}, nil
	}
	defer file.Close()

	tok, err := svc.UploadVPPToken(ctx, file)
	if err != nil {
		return uploadVPPTokenResponse{Err: err}, nil
	}

	return uploadVPPTokenResponse{Token: tok}, nil
}

func (svc *Service) UploadVPPToken(ctx context.Context, file io.ReadSeeker) (*fleet.VPPTokenDB, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}

////////////////////////////////////////////////////
// PATCH /api/_version_/fleet/vpp_tokens/%d/renew //
////////////////////////////////////////////////////

type patchVPPTokenRenewRequest struct {
	ID   uint `url:"id"`
	File *multipart.FileHeader
}

func (patchVPPTokenRenewRequest) DecodeRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	decoded := patchVPPTokenRenewRequest{}

	err := r.ParseMultipartForm(512 * units.MiB)
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

	id, err := uintFromRequest(r, "id")
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "failed to parse vpp token id")
	}

	decoded.ID = uint(id) //nolint:gosec // dismiss G115

	return &decoded, nil
}

type patchVPPTokenRenewResponse struct {
	Err   error             `json:"error,omitempty"`
	Token *fleet.VPPTokenDB `json:"token,omitempty"`
}

func (r patchVPPTokenRenewResponse) Status() int { return http.StatusAccepted }

func (r patchVPPTokenRenewResponse) error() error {
	return r.Err
}

func patchVPPTokenRenewEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*patchVPPTokenRenewRequest)
	file, err := req.File.Open()
	if err != nil {
		return patchVPPTokenRenewResponse{Err: err}, nil
	}
	defer file.Close()

	tok, err := svc.UpdateVPPToken(ctx, req.ID, file)
	if err != nil {
		return patchVPPTokenRenewResponse{Err: err}, nil
	}

	return patchVPPTokenRenewResponse{Token: tok}, nil
}

func (svc *Service) UpdateVPPToken(ctx context.Context, tokenID uint, token io.ReadSeeker) (*fleet.VPPTokenDB, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}

////////////////////////////////////////////////////
// PATCH /api/_version_/fleet/vpp_tokens/%d/teams //
////////////////////////////////////////////////////

type patchVPPTokensTeamsRequest struct {
	ID      uint   `url:"id"`
	TeamIDs []uint `json:"teams"`
}

type patchVPPTokensTeamsResponse struct {
	Token *fleet.VPPTokenDB `json:"token,omitempty"`
	Err   error             `json:"error,omitempty"`
}

func (r patchVPPTokensTeamsResponse) error() error { return r.Err }

func patchVPPTokensTeams(ctx context.Context, request any, svc fleet.Service) (errorer, error) {
	req := request.(*patchVPPTokensTeamsRequest)

	tok, err := svc.UpdateVPPTokenTeams(ctx, req.ID, req.TeamIDs)
	if err != nil {
		return patchVPPTokensTeamsResponse{Err: err}, nil
	}
	return patchVPPTokensTeamsResponse{Token: tok}, nil
}

func (svc *Service) UpdateVPPTokenTeams(ctx context.Context, tokenID uint, teamIDs []uint) (*fleet.VPPTokenDB, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}

/////////////////////////////////////////
// GET /api/_version_/fleet/vpp_tokens //
/////////////////////////////////////////

type getVPPTokensRequest struct{}

type getVPPTokensResponse struct {
	Tokens []*fleet.VPPTokenDB `json:"vpp_tokens"`
	Err    error               `json:"error,omitempty"`
}

func (r getVPPTokensResponse) error() error { return r.Err }

func getVPPTokens(ctx context.Context, request any, svc fleet.Service) (errorer, error) {
	tokens, err := svc.GetVPPTokens(ctx)
	if err != nil {
		return getVPPTokensResponse{Err: err}, nil
	}

	if tokens == nil {
		tokens = []*fleet.VPPTokenDB{}
	}

	return getVPPTokensResponse{Tokens: tokens}, nil
}

func (svc *Service) GetVPPTokens(ctx context.Context) ([]*fleet.VPPTokenDB, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}

///////////////////////////////////////////////
// DELETE /api/_version_/fleet/vpp_tokens/%d //
///////////////////////////////////////////////

type deleteVPPTokenRequest struct {
	ID uint `url:"id"`
}

type deleteVPPTokenResponse struct {
	Err error `json:"error,omitempty"`
}

func (r deleteVPPTokenResponse) error() error { return r.Err }

func (r deleteVPPTokenResponse) Status() int { return http.StatusNoContent }

func deleteVPPToken(ctx context.Context, request any, svc fleet.Service) (errorer, error) {
	req := request.(*deleteVPPTokenRequest)

	err := svc.DeleteVPPToken(ctx, req.ID)
	if err != nil {
		return deleteVPPTokenResponse{Err: err}, nil
	}

	return deleteVPPTokenResponse{}, nil
}

func (svc *Service) DeleteVPPToken(ctx context.Context, tokenID uint) error {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return fleet.ErrMissingLicense
}
