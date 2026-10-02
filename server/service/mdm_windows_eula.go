package service

import (
	"context"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/fleet"
)

////////////////////////////////////////////////////////////////////////////////
// POST /setup_experience/windows_eula
////////////////////////////////////////////////////////////////////////////////

type createMDMWindowsEULARequest struct {
	EULA   *multipart.FileHeader
	DryRun bool `query:"dry_run,optional"` // if true, apply validation but do not save changes
}

func (createMDMWindowsEULARequest) DecodeRequest(ctx context.Context, r *http.Request) (any, error) {
	eula, dryRun, err := decodeEULAUpload(r, "windows_eula", fleet.MaxWindowsEULARequestSize)
	if err != nil {
		return nil, err
	}
	return &createMDMWindowsEULARequest{
		EULA:   eula,
		DryRun: dryRun,
	}, nil
}

type createMDMWindowsEULAResponse struct {
	Err error `json:"error,omitempty"`
}

func (r createMDMWindowsEULAResponse) Error() error { return r.Err }

func createMDMWindowsEULAEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*createMDMWindowsEULARequest)
	ff, err := req.EULA.Open()
	if err != nil {
		return createMDMWindowsEULAResponse{Err: err}, nil
	}
	defer ff.Close()

	if err := svc.MDMCreateWindowsEULA(ctx, req.EULA.Filename, ff, req.DryRun); err != nil {
		return createMDMWindowsEULAResponse{Err: err}, nil
	}

	return createMDMWindowsEULAResponse{}, nil
}

func (svc *Service) MDMCreateWindowsEULA(ctx context.Context, name string, file io.ReadSeeker, dryRun bool) error {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return fleet.ErrMissingLicense
}

////////////////////////////////////////////////////////////////////////////////
// GET /setup_experience/windows_eula/:token
////////////////////////////////////////////////////////////////////////////////

type getMDMWindowsEULARequest struct {
	Token string `url:"token"`
}

type getMDMWindowsEULAResponse struct {
	Err error `json:"error,omitempty"`

	// fields used in hijackRender to build the response
	eula *fleet.MDMEULA
}

func (r getMDMWindowsEULAResponse) Error() error { return r.Err }

func (r getMDMWindowsEULAResponse) HijackRender(ctx context.Context, w http.ResponseWriter) {
	// Markdown is downloaded, never rendered as a page in the admin's browser.
	writeEULAFile(ctx, w, r.eula, "text/markdown; charset=utf-8", true)
}

func getMDMWindowsEULAEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*getMDMWindowsEULARequest)

	eula, err := svc.MDMGetWindowsEULABytes(ctx, req.Token)
	if err != nil {
		return getMDMWindowsEULAResponse{Err: err}, nil
	}

	return getMDMWindowsEULAResponse{eula: eula}, nil
}

func (svc *Service) MDMGetWindowsEULABytes(ctx context.Context, token string) (*fleet.MDMEULA, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}

////////////////////////////////////////////////////////////////////////////////
// GET /setup_experience/windows_eula/metadata
////////////////////////////////////////////////////////////////////////////////

type getMDMWindowsEULAMetadataRequest struct{}

type getMDMWindowsEULAMetadataResponse struct {
	*fleet.MDMEULA
	Err error `json:"error,omitempty"`
}

func (r getMDMWindowsEULAMetadataResponse) Error() error { return r.Err }

func getMDMWindowsEULAMetadataEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	eula, err := svc.MDMGetWindowsEULAMetadata(ctx)
	if err != nil && !fleet.IsNotFound(err) {
		return getMDMWindowsEULAMetadataResponse{Err: err}, nil
	}

	if eula == nil {
		// Returned as an error rather than in the response so the server sees
		// the 404 without logging it as an error.
		return nil, newNotFoundError()
	}

	return getMDMWindowsEULAMetadataResponse{MDMEULA: eula}, nil
}

func (svc *Service) MDMGetWindowsEULAMetadata(ctx context.Context) (*fleet.MDMEULA, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}

////////////////////////////////////////////////////////////////////////////////
// DELETE /setup_experience/windows_eula/:token
////////////////////////////////////////////////////////////////////////////////

type deleteMDMWindowsEULARequest struct {
	Token  string `url:"token"`
	DryRun bool   `query:"dry_run,optional"` // if true, apply validation but do not delete
}

type deleteMDMWindowsEULAResponse struct {
	Err error `json:"error,omitempty"`
}

func (r deleteMDMWindowsEULAResponse) Error() error { return r.Err }

func deleteMDMWindowsEULAEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*deleteMDMWindowsEULARequest)
	if err := svc.MDMDeleteWindowsEULA(ctx, req.Token, req.DryRun); err != nil {
		return deleteMDMWindowsEULAResponse{Err: err}, nil
	}
	return deleteMDMWindowsEULAResponse{}, nil
}

func (svc *Service) MDMDeleteWindowsEULA(ctx context.Context, token string, dryRun bool) error {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return fleet.ErrMissingLicense
}
