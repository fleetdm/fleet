package service

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
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
	UninstallScript   string
	LabelsIncludeAny  []string
	LabelsExcludeAny  []string
	AutomaticInstall  bool
}

type updateSoftwareInstallerRequest struct {
	TitleID           uint `url:"id"`
	File              *multipart.FileHeader
	TeamID            *uint
	InstallScript     *string
	PreInstallQuery   *string
	PostInstallScript *string
	UninstallScript   *string
	SelfService       *bool
	LabelsIncludeAny  []string
	LabelsExcludeAny  []string
}

type uploadSoftwareInstallerResponse struct {
	Err error `json:"error,omitempty"`
}

// TODO: We parse the whole body before running svc.authz.Authorize.
// An authenticated but unauthorized user could abuse this.
func (updateSoftwareInstallerRequest) DecodeRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	decoded := updateSoftwareInstallerRequest{}

	// populate software title ID since we're overriding the decoder that would do it for us
	titleID, err := uint32FromRequest(r, "id")
	if err != nil {
		return nil, badRequestErr("intFromRequest", err)
	}
	decoded.TitleID = uint(titleID)

	err = r.ParseMultipartForm(512 * units.MiB)
	if err != nil {
		var mbe *http.MaxBytesError
		if errors.As(err, &mbe) {
			return nil, &fleet.BadRequestError{
				Message:     "The maximum file size is 3 GB.",
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

	// unlike for uploadSoftwareInstallerRequest, every field is optional, including the file upload
	if r.MultipartForm.File["software"] != nil || len(r.MultipartForm.File["software"]) > 0 {
		decoded.File = r.MultipartForm.File["software"][0]
		if decoded.File.Size > fleet.MaxSoftwareInstallerSize {
			// Should never happen here since the request's body is limited to the maximum size.
			return nil, &fleet.BadRequestError{
				Message: "The maximum file size is 3 GB.",
			}
		}
	}

	// default is no team
	val, ok := r.MultipartForm.Value["team_id"]
	if ok {
		teamID, err := strconv.ParseUint(val[0], 10, 32)
		if err != nil {
			return nil, &fleet.BadRequestError{Message: fmt.Sprintf("Invalid team_id: %s", val[0])}
		}
		decoded.TeamID = ptr.Uint(uint(teamID))
	}

	installScriptMultipart, ok := r.MultipartForm.Value["install_script"]
	if ok && len(installScriptMultipart) > 0 {
		decoded.InstallScript = &installScriptMultipart[0]
	}

	preinstallQueryMultipart, ok := r.MultipartForm.Value["pre_install_query"]
	if ok && len(preinstallQueryMultipart) > 0 {
		decoded.PreInstallQuery = &preinstallQueryMultipart[0]
	}

	postInstallScriptMultipart, ok := r.MultipartForm.Value["post_install_script"]
	if ok && len(postInstallScriptMultipart) > 0 {
		decoded.PostInstallScript = &postInstallScriptMultipart[0]
	}

	uninstallScriptMultipart, ok := r.MultipartForm.Value["uninstall_script"]
	if ok && len(uninstallScriptMultipart) > 0 {
		decoded.UninstallScript = &uninstallScriptMultipart[0]
	}

	val, ok = r.MultipartForm.Value["self_service"]
	if ok && len(val) > 0 && val[0] != "" {
		parsed, err := strconv.ParseBool(val[0])
		if err != nil {
			return nil, &fleet.BadRequestError{Message: fmt.Sprintf("failed to decode self_service bool in multipart form: %s", err.Error())}
		}
		decoded.SelfService = &parsed
	}

	// decode labels
	var inclAny, exclAny []string
	var existsInclAny, existsExclAny bool

	inclAny, existsInclAny = r.MultipartForm.Value[string(fleet.LabelsIncludeAny)]
	switch {
	case !existsInclAny:
		decoded.LabelsIncludeAny = nil
	case len(inclAny) == 1 && inclAny[0] == "":
		decoded.LabelsIncludeAny = []string{}
	default:
		decoded.LabelsIncludeAny = inclAny
	}

	exclAny, existsExclAny = r.MultipartForm.Value[string(fleet.LabelsExcludeAny)]
	switch {
	case !existsExclAny:
		decoded.LabelsExcludeAny = nil
	case len(exclAny) == 1 && exclAny[0] == "":
		decoded.LabelsExcludeAny = []string{}
	default:
		decoded.LabelsExcludeAny = exclAny
	}

	return &decoded, nil
}

func updateSoftwareInstallerEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*updateSoftwareInstallerRequest)

	payload := &fleet.UpdateSoftwareInstallerPayload{
		TitleID:           req.TitleID,
		TeamID:            req.TeamID,
		InstallScript:     req.InstallScript,
		PreInstallQuery:   req.PreInstallQuery,
		PostInstallScript: req.PostInstallScript,
		UninstallScript:   req.UninstallScript,
		SelfService:       req.SelfService,
		LabelsIncludeAny:  req.LabelsIncludeAny,
		LabelsExcludeAny:  req.LabelsExcludeAny,
	}
	if req.File != nil {
		ff, err := req.File.Open()
		if err != nil {
			return uploadSoftwareInstallerResponse{Err: err}, nil
		}
		defer ff.Close()

		tfr, err := fleet.NewTempFileReader(ff, nil)
		if err != nil {
			return uploadSoftwareInstallerResponse{Err: err}, nil
		}
		defer tfr.Close()

		payload.InstallerFile = tfr
		payload.Filename = req.File.Filename
	}

	installer, err := svc.UpdateSoftwareInstaller(ctx, payload)
	if err != nil {
		return uploadSoftwareInstallerResponse{Err: err}, nil
	}

	return getSoftwareInstallerResponse{SoftwareInstaller: installer}, nil
}

func (svc *Service) UpdateSoftwareInstaller(ctx context.Context, payload *fleet.UpdateSoftwareInstallerPayload) (*fleet.SoftwareInstaller, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}

// TODO: We parse the whole body before running svc.authz.Authorize.
// An authenticated but unauthorized user could abuse this.
func (uploadSoftwareInstallerRequest) DecodeRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	decoded := uploadSoftwareInstallerRequest{}

	err := r.ParseMultipartForm(512 * units.MiB)
	if err != nil {
		var mbe *http.MaxBytesError
		if errors.As(err, &mbe) {
			return nil, &fleet.BadRequestError{
				Message:     "The maximum file size is 3 GB.",
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
	if decoded.File.Size > fleet.MaxSoftwareInstallerSize {
		// Should never happen here since the request's body is limited to the
		// maximum size.
		return nil, &fleet.BadRequestError{
			Message: "The maximum file size is 3 GB.",
		}
	}

	// default is no team
	val, ok := r.MultipartForm.Value["team_id"]
	if ok {
		teamID, err := strconv.ParseUint(val[0], 10, 32)
		if err != nil {
			return nil, &fleet.BadRequestError{Message: fmt.Sprintf("Invalid team_id: %s", val[0])}
		}
		decoded.TeamID = ptr.Uint(uint(teamID))
	}

	val, ok = r.MultipartForm.Value["install_script"]
	if ok && len(val) > 0 {
		decoded.InstallScript = val[0]
	}

	val, ok = r.MultipartForm.Value["uninstall_script"]
	if ok && len(val) > 0 {
		decoded.UninstallScript = val[0]
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

	// decode labels
	var inclAny, exclAny []string
	var existsInclAny, existsExclAny bool

	inclAny, existsInclAny = r.MultipartForm.Value[string(fleet.LabelsIncludeAny)]
	switch {
	case !existsInclAny:
		decoded.LabelsIncludeAny = nil
	case len(inclAny) == 1 && inclAny[0] == "":
		decoded.LabelsIncludeAny = []string{}
	default:
		decoded.LabelsIncludeAny = inclAny
	}

	exclAny, existsExclAny = r.MultipartForm.Value[string(fleet.LabelsExcludeAny)]
	switch {
	case !existsExclAny:
		decoded.LabelsExcludeAny = nil
	case len(exclAny) == 1 && exclAny[0] == "":
		decoded.LabelsExcludeAny = []string{}
	default:
		decoded.LabelsExcludeAny = exclAny
	}

	val, ok = r.MultipartForm.Value["automatic_install"]
	if ok && len(val) > 0 && val[0] != "" {
		parsed, err := strconv.ParseBool(val[0])
		if err != nil {
			return nil, &fleet.BadRequestError{Message: fmt.Sprintf("failed to decode automatic_install bool in multipart form: %s", err.Error())}
		}
		decoded.AutomaticInstall = parsed
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

	tfr, err := fleet.NewTempFileReader(ff, nil)
	if err != nil {
		return uploadSoftwareInstallerResponse{Err: err}, nil
	}
	defer tfr.Close()

	payload := &fleet.UploadSoftwareInstallerPayload{
		TeamID:            req.TeamID,
		InstallScript:     req.InstallScript,
		PreInstallQuery:   req.PreInstallQuery,
		PostInstallScript: req.PostInstallScript,
		InstallerFile:     tfr,
		Filename:          req.File.Filename,
		SelfService:       req.SelfService,
		UninstallScript:   req.UninstallScript,
		LabelsIncludeAny:  req.LabelsIncludeAny,
		LabelsExcludeAny:  req.LabelsExcludeAny,
		AutomaticInstall:  req.AutomaticInstall,
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

type downloadSoftwareInstallerRequest struct {
	TitleID uint   `url:"title_id"`
	Token   string `url:"token"`
}

func getSoftwareInstallerEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*getSoftwareInstallerRequest)

	payload, err := svc.DownloadSoftwareInstaller(ctx, false, req.Alt, req.TitleID, req.TeamID)
	if err != nil {
		return orbitDownloadSoftwareInstallerResponse{Err: err}, nil
	}

	return orbitDownloadSoftwareInstallerResponse{payload: payload}, nil
}

func getSoftwareInstallerTokenEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*getSoftwareInstallerRequest)

	token, err := svc.GenerateSoftwareInstallerToken(ctx, req.Alt, req.TitleID, req.TeamID)
	if err != nil {
		return getSoftwareInstallerTokenResponse{Err: err}, nil
	}
	return getSoftwareInstallerTokenResponse{Token: token}, nil
}

func downloadSoftwareInstallerEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*downloadSoftwareInstallerRequest)

	meta, err := svc.GetSoftwareInstallerTokenMetadata(ctx, req.Token, req.TitleID)
	if err != nil {
		return orbitDownloadSoftwareInstallerResponse{Err: err}, nil
	}

	payload, err := svc.DownloadSoftwareInstaller(ctx, true, "media", meta.TitleID, &meta.TeamID)
	if err != nil {
		return orbitDownloadSoftwareInstallerResponse{Err: err}, nil
	}

	return orbitDownloadSoftwareInstallerResponse{payload: payload}, nil
}

func (svc *Service) GenerateSoftwareInstallerToken(ctx context.Context, _ string, _ uint, _ *uint) (string, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return "", fleet.ErrMissingLicense
}

func (svc *Service) GetSoftwareInstallerTokenMetadata(ctx context.Context, _ string, _ uint) (*fleet.SoftwareInstallerTokenMetadata,
	error,
) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}

func (svc *Service) GetSoftwareInstallerMetadata(ctx context.Context, _ bool, _ uint, _ *uint) (*fleet.SoftwareInstaller, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}

type getSoftwareInstallerResponse struct {
	SoftwareInstaller *fleet.SoftwareInstaller `json:"software_installer,omitempty"`
	Err               error                    `json:"error,omitempty"`
}

func (r getSoftwareInstallerResponse) error() error { return r.Err }

type getSoftwareInstallerTokenResponse struct {
	Err   error  `json:"error,omitempty"`
	Token string `json:"token"`
}

func (r getSoftwareInstallerTokenResponse) error() error { return r.Err }

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

func (svc *Service) DownloadSoftwareInstaller(ctx context.Context, _ bool, _ string, _ uint,
	_ *uint) (*fleet.DownloadSoftwareInstallerPayload,
	error,
) {
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

func (svc *Service) GetVPPTokenIfCanInstallVPPApps(ctx context.Context, appleDevice bool, host *fleet.Host) (string, error) {
	return "", fleet.ErrMissingLicense // called downstream of auth checks so doesn't need skipauth
}

func (svc *Service) InstallVPPAppPostValidation(ctx context.Context, host *fleet.Host, vppApp *fleet.VPPApp, token string, selfService bool, policyID *uint) (string, error) {
	return "", fleet.ErrMissingLicense // called downstream of auth checks so doesn't need skipauth
}

type uninstallSoftwareRequest struct {
	HostID          uint `url:"host_id"`
	SoftwareTitleID uint `url:"software_title_id"`
}

func uninstallSoftwareTitleEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*uninstallSoftwareRequest)

	err := svc.UninstallSoftwareTitle(ctx, req.HostID, req.SoftwareTitleID)
	if err != nil {
		return installSoftwareResponse{Err: err}, nil
	}

	return installSoftwareResponse{}, nil
}

func (svc *Service) UninstallSoftwareTitle(ctx context.Context, _ uint, _ uint) error {
	// skipauth: No authorization check needed due to implementation returning only license error.
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
	TeamName string                            `json:"-" query:"team_name,optional"`
	DryRun   bool                              `json:"-" query:"dry_run,optional"` // if true, apply validation but do not save changes
	Software []*fleet.SoftwareInstallerPayload `json:"software"`
}

type batchSetSoftwareInstallersResponse struct {
	RequestUUID string `json:"request_uuid"`
	Err         error  `json:"error,omitempty"`
}

func (r batchSetSoftwareInstallersResponse) error() error { return r.Err }
func (r batchSetSoftwareInstallersResponse) Status() int  { return http.StatusAccepted }

func batchSetSoftwareInstallersEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*batchSetSoftwareInstallersRequest)
	requestUUID, err := svc.BatchSetSoftwareInstallers(ctx, req.TeamName, req.Software, req.DryRun)
	if err != nil {
		return batchSetSoftwareInstallersResponse{Err: err}, nil
	}
	return batchSetSoftwareInstallersResponse{RequestUUID: requestUUID}, nil
}

func (svc *Service) BatchSetSoftwareInstallers(ctx context.Context, tmName string, payloads []*fleet.SoftwareInstallerPayload, dryRun bool) (string, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return "", fleet.ErrMissingLicense
}

type batchSetSoftwareInstallersResultRequest struct {
	RequestUUID string `url:"request_uuid"`
	TeamName    string `query:"team_name,optional"`
	DryRun      bool   `query:"dry_run,optional"` // if true, apply validation but do not save changes
}

type batchSetSoftwareInstallersResultResponse struct {
	Status   string                          `json:"status"`
	Message  string                          `json:"message"`
	Packages []fleet.SoftwarePackageResponse `json:"packages"`

	Err error `json:"error,omitempty"`
}

func (r batchSetSoftwareInstallersResultResponse) error() error { return r.Err }

func batchSetSoftwareInstallersResultEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*batchSetSoftwareInstallersResultRequest)
	status, message, packages, err := svc.GetBatchSetSoftwareInstallersResult(ctx, req.TeamName, req.RequestUUID, req.DryRun)
	if err != nil {
		return batchSetSoftwareInstallersResultResponse{Err: err}, nil
	}
	return batchSetSoftwareInstallersResultResponse{
		Status:   status,
		Message:  message,
		Packages: packages,
	}, nil
}

func (svc *Service) GetBatchSetSoftwareInstallersResult(ctx context.Context, tmName string, requestUUID string, dryRun bool) (string, string, []fleet.SoftwarePackageResponse, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return "", "", nil, fleet.ErrMissingLicense
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
// VPP App Store Apps Batch Install
//////////////////////////////////////////////////////////////////////////////

type batchAssociateAppStoreAppsRequest struct {
	TeamName string                  `json:"-" query:"team_name,optional"`
	DryRun   bool                    `json:"-" query:"dry_run,optional"`
	Apps     []fleet.VPPBatchPayload `json:"app_store_apps"`
}

func (b *batchAssociateAppStoreAppsRequest) DecodeBody(ctx context.Context, r io.Reader, u url.Values, c []*x509.Certificate) error {
	if err := json.NewDecoder(r).Decode(b); err != nil {
		var typeErr *json.UnmarshalTypeError
		if errors.As(err, &typeErr) {
			return ctxerr.Wrap(ctx, fleet.NewUserMessageError(fmt.Errorf("Couldn't edit software. %q must be a %s, found %s", typeErr.Field, typeErr.Type.String(), typeErr.Value), http.StatusBadRequest))
		}
	}

	return nil
}

type batchAssociateAppStoreAppsResponse struct {
	Apps []fleet.VPPAppResponse `json:"app_store_apps"`
	Err  error                  `json:"error,omitempty"`
}

func (r batchAssociateAppStoreAppsResponse) error() error { return r.Err }

func (r batchAssociateAppStoreAppsResponse) Status() int { return http.StatusNoContent }

func batchAssociateAppStoreAppsEndpoint(ctx context.Context, request any, svc fleet.Service) (errorer, error) {
	req := request.(*batchAssociateAppStoreAppsRequest)
	apps, err := svc.BatchAssociateVPPApps(ctx, req.TeamName, req.Apps, req.DryRun)
	if err != nil {
		return batchAssociateAppStoreAppsResponse{Err: err}, nil
	}
	return batchAssociateAppStoreAppsResponse{Apps: apps}, nil
}

func (svc *Service) BatchAssociateVPPApps(ctx context.Context, teamName string, payloads []fleet.VPPBatchPayload, dryRun bool) ([]fleet.VPPAppResponse, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}
