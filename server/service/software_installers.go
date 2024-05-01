package service

import (
	"context"
	"fmt"
	"mime/multipart"
	"net/http"
	"strconv"

	"github.com/docker/go-units"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
)

type uploadSoftwareInstallerRequest struct {
	File              *multipart.FileHeader
	TeamID            *uint
	InstallScript     string
	PreInstallQuery   string
	PostInstallScript string
}

type uploadSoftwareInstallerResponse struct {
	Err error `json:"error,omitempty"`
}

// TODO: We parse the whole body before running svc.authz.Authorize.
// An authenticated but unauthorized user could abuse this.
func (uploadSoftwareInstallerRequest) DecodeRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	decoded := uploadSoftwareInstallerRequest{}
	err := r.ParseMultipartForm(512 * units.MiB)
	if err != nil {
		return nil, &fleet.BadRequestError{
			Message:     "failed to parse multipart form",
			InternalErr: err,
		}
	}

	if r.MultipartForm.File["software"] == nil || len(r.MultipartForm.File["software"]) == 0 {
		return nil, &fleet.BadRequestError{
			Message:     "software multipart field is required",
			InternalErr: err,
		}
	}

	decoded.File = r.MultipartForm.File["software"][0]

	if decoded.File.Size > 500*units.MiB {
		// TODO: Should we try to assess the size earlier in the request processing (before parsing the form)?
		return nil, &fleet.BadRequestError{
			Message: "The maximum file size is 500 MB.",
		}
	}

	// default is no team
	val, ok := r.MultipartForm.Value["team_id"]
	if ok && len(val) > 0 {
		teamID, err := strconv.Atoi(val[0])
		if err != nil {
			return nil, &fleet.BadRequestError{Message: fmt.Sprintf("failed to decode team_id in multipart form: %s", err.Error())}
		}
		decoded.TeamID = ptr.Uint(uint(teamID))
	}

	val, ok = r.MultipartForm.Value["install_script"]
	if ok && len(val) > 0 {
		decoded.InstallScript = val[0]
	}

	val, ok = r.MultipartForm.Value["pre_install_query"]
	if ok && len(val) > 0 {
		decoded.PreInstallQuery = val[0]
	}

	val, ok = r.MultipartForm.Value["post_install_script"]
	if ok && len(val) > 0 {
		decoded.PostInstallScript = val[0]
	}

	return &decoded, nil
}

func (r uploadSoftwareInstallerResponse) error() error { return r.Err }

func uploadSoftwareInstallerEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*uploadSoftwareInstallerRequest)
	ff, err := req.File.Open()
	if err != nil {
		return uploadSoftwareInstallerResponse{Err: err}, nil
	}
	defer ff.Close()

	payload := &fleet.UploadSoftwareInstallerPayload{
		TeamID:            req.TeamID,
		InstallScript:     req.InstallScript,
		PreInstallQuery:   req.PreInstallQuery,
		PostInstallScript: req.PostInstallScript,
		InstallerFile:     ff,
		Filename:          req.File.Filename,
	}

	if err := svc.UploadSoftwareInstaller(ctx, payload); err != nil {
		return uploadSoftwareInstallerResponse{Err: err}, nil
	}
	return &uploadSoftwareInstallerResponse{}, nil
}

func (svc *Service) UploadSoftwareInstaller(ctx context.Context, payload *fleet.UploadSoftwareInstallerPayload) error {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return fleet.ErrMissingLicense
}

type deleteSoftwareInstallerRequest struct {
	ID uint `url:"id"`
}

type deleteSoftwareInstallerResponse struct {
	Err error `json:"error,omitempty"`
}

func (r deleteSoftwareInstallerResponse) error() error { return r.Err }
func (r deleteSoftwareInstallerResponse) Status() int  { return http.StatusNoContent }

func deleteSoftwareInstallerEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*deleteSoftwareInstallerRequest)
	err := svc.DeleteSoftwareInstaller(ctx, req.ID)
	if err != nil {
		return deleteSoftwareInstallerResponse{Err: err}, nil
	}
	return deleteSoftwareInstallerResponse{}, nil
}

func (svc *Service) DeleteSoftwareInstaller(ctx context.Context, id uint) error {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return fleet.ErrMissingLicense
}
