package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"strconv"

	"github.com/docker/go-units"
	authzctx "github.com/fleetdm/fleet/v4/server/contexts/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	hostctx "github.com/fleetdm/fleet/v4/server/contexts/host"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
)

type uploadSoftwareInstallerRequest struct {
	File              *multipart.FileHeader
	TeamID            *uint
	InstallScript     string
	PreInstallQuery   string
	PostInstallScript string
	SelfService       bool
}

type uploadSoftwareInstallerResponse struct {
	Err error `json:"error,omitempty"`
}

// MaxSoftwareInstallerSize is the maximum size allowed for software
// installers. This is enforced by the endpoint that uploads installers.
const MaxSoftwareInstallerSize = 500 * units.MiB

// TODO: We parse the whole body before running svc.authz.Authorize.
// An authenticated but unauthorized user could abuse this.
func (uploadSoftwareInstallerRequest) DecodeRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	decoded := uploadSoftwareInstallerRequest{}
	err := r.ParseMultipartForm(512 * units.MiB)
	if err != nil {
		var mbe *http.MaxBytesError
		if errors.As(err, &mbe) {
			return nil, &fleet.BadRequestError{
				Message:     "The maximum file size is 500 MB.",
				InternalErr: err,
			}
		}
		var nerr net.Error
		if errors.As(err, &nerr) && nerr.Timeout() {
			return nil, fleet.NewUserMessageError(
				ctxerr.New(ctx, "Couldn't upload. Please ensure your internet connection speed is sufficient and stable."),
				http.StatusRequestTimeout,
			)
		}
		return nil, &fleet.BadRequestError{
			Message:     "failed to parse multipart form: " + err.Error(),
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
	if decoded.File.Size > MaxSoftwareInstallerSize {
		// Should never happen here since the request's body is limited to the
		// maximum size.
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

	val, ok = r.MultipartForm.Value["self_service"]
	if ok && len(val) > 0 && val[0] != "" {
		parsed, err := strconv.ParseBool(val[0])
		if err != nil {
			return nil, &fleet.BadRequestError{Message: fmt.Sprintf("failed to decode self_service bool in multipart form: %s", err.Error())}
		}
		decoded.SelfService = parsed
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
		SelfService:       req.SelfService,
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
	TeamID  *uint `query:"team_id"`
	TitleID uint  `url:"title_id"`
}

type deleteSoftwareInstallerResponse struct {
	Err error `json:"error,omitempty"`
}

func (r deleteSoftwareInstallerResponse) error() error { return r.Err }
func (r deleteSoftwareInstallerResponse) Status() int  { return http.StatusNoContent }

func deleteSoftwareInstallerEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*deleteSoftwareInstallerRequest)
	err := svc.DeleteSoftwareInstaller(ctx, req.TitleID, req.TeamID)
	if err != nil {
		return deleteSoftwareInstallerResponse{Err: err}, nil
	}
	return deleteSoftwareInstallerResponse{}, nil
}

func (svc *Service) DeleteSoftwareInstaller(ctx context.Context, titleID uint, teamID *uint) error {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return fleet.ErrMissingLicense
}

type getSoftwareInstallerRequest struct {
	Alt     string `query:"alt,optional"`
	TeamID  *uint  `query:"team_id"`
	TitleID uint   `url:"title_id"`
}

type getSoftwareInstallerResponse struct {
	// meta *fleet.SoftwareInstaller // NOTE: API design currently only supports downloading the
	Err error `json:"error,omitempty"`
}

func (r getSoftwareInstallerResponse) error() error { return r.Err }

func getSoftwareInstallerEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*getSoftwareInstallerRequest)

	downloadRequested := req.Alt == "media"
	if !downloadRequested {
		// TODO: confirm error handling
		return getSoftwareInstallerResponse{Err: &fleet.BadRequestError{Message: "only alt=media is supported"}}, nil
	}

	payload, err := svc.DownloadSoftwareInstaller(ctx, req.TitleID, req.TeamID)
	if err != nil {
		return orbitDownloadSoftwareInstallerResponse{Err: err}, nil
	}

	return orbitDownloadSoftwareInstallerResponse{payload: payload}, nil
}

func (svc *Service) GetSoftwareInstallerMetadata(ctx context.Context, titleID uint, teamID *uint) (*fleet.SoftwareInstaller, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}

type orbitDownloadSoftwareInstallerResponse struct {
	Err error `json:"error,omitempty"`
	// fields used by hijackRender for the response.
	payload *fleet.DownloadSoftwareInstallerPayload
}

func (r orbitDownloadSoftwareInstallerResponse) error() error { return r.Err }

func (r orbitDownloadSoftwareInstallerResponse) hijackRender(ctx context.Context, w http.ResponseWriter) {
	w.Header().Set("Content-Length", strconv.Itoa(int(r.payload.Size)))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment;filename="%s"`, r.payload.Filename))

	// OK to just log the error here as writing anything on
	// `http.ResponseWriter` sets the status code to 200 (and it can't be
	// changed.) Clients should rely on matching content-length with the
	// header provided
	if n, err := io.Copy(w, r.payload.Installer); err != nil {
		logging.WithExtras(ctx, "err", err, "bytes_copied", n)
	}
	r.payload.Installer.Close()
}

func (svc *Service) DownloadSoftwareInstaller(ctx context.Context, titleID uint, teamID *uint) (*fleet.DownloadSoftwareInstallerPayload, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}

/////////////////////////////////////////////////////////////////////////////////
// Request to install software in a host
/////////////////////////////////////////////////////////////////////////////////

type installSoftwareRequest struct {
	HostID          uint `url:"host_id"`
	SoftwareTitleID uint `url:"software_title_id"`
}

type installSoftwareResponse struct {
	Err error `json:"error,omitempty"`
}

func (r installSoftwareResponse) error() error { return r.Err }

func (r installSoftwareResponse) Status() int { return http.StatusAccepted }

func installSoftwareTitleEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*installSoftwareRequest)

	err := svc.InstallSoftwareTitle(ctx, req.HostID, req.SoftwareTitleID)
	if err != nil {
		return installSoftwareResponse{Err: err}, nil
	}

	return installSoftwareResponse{}, nil
}

func (svc *Service) InstallSoftwareTitle(ctx context.Context, hostID uint, softwareTitleID uint) error {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return fleet.ErrMissingLicense
}

type getSoftwareInstallResultsRequest struct {
	InstallUUID string `url:"install_uuid"`
}

type getSoftwareInstallResultsResponse struct {
	Err     error                              `json:"error,omitempty"`
	Results *fleet.HostSoftwareInstallerResult `json:"results,omitempty"`
}

func (r getSoftwareInstallResultsResponse) error() error { return r.Err }

func getSoftwareInstallResultsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*getSoftwareInstallResultsRequest)

	results, err := svc.GetSoftwareInstallResults(ctx, req.InstallUUID)
	if err != nil {
		return getSoftwareInstallResultsResponse{Err: err}, nil
	}

	return &getSoftwareInstallResultsResponse{Results: results}, nil
}

func (svc *Service) GetSoftwareInstallResults(ctx context.Context, resultUUID string) (*fleet.HostSoftwareInstallerResult, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}

////////////////////////////////////////////////////////////////////////////////
// Batch replace software installers
////////////////////////////////////////////////////////////////////////////////

type batchSetSoftwareInstallersRequest struct {
	TeamName string                           `json:"-" query:"team_name"`
	DryRun   bool                             `json:"-" query:"dry_run,optional"` // if true, apply validation but do not save changes
	Software []fleet.SoftwareInstallerPayload `json:"software"`
}

type batchSetSoftwareInstallersResponse struct {
	Err error `json:"error,omitempty"`
}

func (r batchSetSoftwareInstallersResponse) error() error { return r.Err }

func (r batchSetSoftwareInstallersResponse) Status() int { return http.StatusNoContent }

func batchSetSoftwareInstallersEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*batchSetSoftwareInstallersRequest)
	if err := svc.BatchSetSoftwareInstallers(ctx, req.TeamName, req.Software, req.DryRun); err != nil {
		return batchSetSoftwareInstallersResponse{Err: err}, nil
	}
	return batchSetSoftwareInstallersResponse{}, nil
}

func (svc *Service) BatchSetSoftwareInstallers(ctx context.Context, tmName string, payloads []fleet.SoftwareInstallerPayload, dryRun bool) error {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return fleet.ErrMissingLicense
}

//////////////////////////////////////////////////////////////////////////////
// Self Service Install
//////////////////////////////////////////////////////////////////////////////

type fleetSelfServiceSoftwareInstallRequest struct {
	Token           string `url:"token"`
	SoftwareTitleID uint   `url:"software_title_id"`
}

func (r *fleetSelfServiceSoftwareInstallRequest) deviceAuthToken() string {
	return r.Token
}

type submitSelfServiceSoftwareInstallResponse struct {
	Err error `json:"error,omitempty"`
}

func (r submitSelfServiceSoftwareInstallResponse) error() error { return r.Err }
func (r submitSelfServiceSoftwareInstallResponse) Status() int  { return http.StatusAccepted }

func submitSelfServiceSoftwareInstall(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	host, ok := hostctx.FromContext(ctx)
	if !ok {
		err := ctxerr.Wrap(ctx, fleet.NewAuthRequiredError("internal error: missing host from request context"))
		return submitSelfServiceSoftwareInstallResponse{Err: err}, nil
	}

	req := request.(*fleetSelfServiceSoftwareInstallRequest)
	if err := svc.SelfServiceInstallSoftwareTitle(ctx, host, req.SoftwareTitleID); err != nil {
		return submitSelfServiceSoftwareInstallResponse{Err: err}, nil
	}

	return submitSelfServiceSoftwareInstallResponse{}, nil
}

func (svc *Service) SelfServiceInstallSoftwareTitle(ctx context.Context, host *fleet.Host, softwareTitleID uint) error {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return fleet.ErrMissingLicense
}

func (svc *Service) HasSelfServiceSoftwareInstallers(ctx context.Context, host *fleet.Host) (bool, error) {
	alreadyAuthenticated := svc.authz.IsAuthenticatedWith(ctx, authzctx.AuthnDeviceToken)
	if !alreadyAuthenticated {
		if err := svc.authz.Authorize(ctx, host, fleet.ActionRead); err != nil {
			return false, err
		}
	}

	return svc.ds.HasSelfServiceSoftwareInstallers(ctx, host.Platform, host.TeamID)
}

//////////////////////////////////////////////////////////////////////////////
// Get App Store Software
//////////////////////////////////////////////////////////////////////////////

type getAppStoreSoftwareRequest struct {
	TeamID uint `url:"team_id"`
}

type getAppStoreSoftwareResponse struct {
	Err error `json:"error,omitempty"`
}

func (r getAppStoreSoftwareResponse) error() error { return r.Err }

func getAppStoreSoftwareEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*getAppStoreSoftwareRequest)
	err := svc.GetAppStoreSoftware(ctx, &req.TeamID)
	if err != nil {
		return &getAppStoreSoftwareResponse{Err: err}, nil
	}

	return &getAppStoreSoftwareResponse{}, nil
}

func (svc *Service) GetAppStoreSoftware(ctx context.Context, teamID *uint) error {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return fleet.ErrMissingLicense
}
