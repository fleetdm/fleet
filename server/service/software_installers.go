package service

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"

	authzctx "github.com/fleetdm/fleet/v4/server/contexts/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	hostctx "github.com/fleetdm/fleet/v4/server/contexts/host"
	"github.com/fleetdm/fleet/v4/server/contexts/installersize"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/platform/endpointer"
	platform_http "github.com/fleetdm/fleet/v4/server/platform/http"

	"github.com/fleetdm/fleet/v4/server/ptr"
)

// TODO: We parse the whole body before running svc.authz.Authorize.
// An authenticated but unauthorized user could abuse this.
type decodeUpdateSoftwareInstallerRequest struct{}

func (decodeUpdateSoftwareInstallerRequest) DecodeRequest(ctx context.Context, r *http.Request) (any, error) {
	decoded := fleet.UpdateSoftwareInstallerRequest{}

	// populate software title ID since we're overriding the decoder that would do it for us
	titleID, err := uint32FromRequest(r, "id")
	if err != nil {
		return nil, endpointer.BadRequestErr("IntFromRequest", err)
	}
	decoded.TitleID = uint(titleID)

	maxInstallerSize := installersize.FromContext(ctx)
	err = parseMultipartForm(ctx, r, platform_http.MaxMultipartFormSize)
	if err != nil {
		var mbe *http.MaxBytesError
		if errors.As(err, &mbe) {
			return nil, &fleet.BadRequestError{
				Message:     fmt.Sprintf("The maximum file size is %s.", installersize.Human(maxInstallerSize)),
				InternalErr: err,
			}
		}
		var nerr net.Error
		if errors.As(err, &nerr) && nerr.Timeout() {
			return nil, fleet.NewUserMessageError(
				ctxerr.New(ctx, "Couldn't add. Please ensure your internet connection speed is sufficient and stable."),
				http.StatusRequestTimeout,
			)
		}
		return nil, &fleet.BadRequestError{
			Message:     "failed to parse multipart form: " + err.Error(),
			InternalErr: err,
		}
	}

	// unlike for fleet.UploadSoftwareInstallerRequest, every field is optional, including the file upload
	if r.MultipartForm.File["software"] != nil || len(r.MultipartForm.File["software"]) > 0 {
		decoded.File = r.MultipartForm.File["software"][0]
		if decoded.File.Size > maxInstallerSize {
			// Should never happen here since the request's body is limited to the maximum size.
			return nil, &fleet.BadRequestError{
				Message: fmt.Sprintf("The maximum file size is %s.", installersize.Human(maxInstallerSize)),
			}
		}
	}

	// default is no team
	val, ok := r.MultipartForm.Value["fleet_id"]
	if ok {
		fleetID, err := strconv.ParseUint(val[0], 10, 32)
		if err != nil {
			return nil, &fleet.BadRequestError{Message: fmt.Sprintf("Invalid fleet_id: %s", val[0])}
		}
		decoded.TeamID = ptr.Uint(uint(fleetID))
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

	// decode labels and categories
	var inclAny, exclAny, categories []string
	var existsInclAny, existsExclAny, existsCategories bool

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

	categories, existsCategories = r.MultipartForm.Value["categories"]
	switch {
	case !existsCategories:
		decoded.Categories = nil
	case len(categories) == 1 && categories[0] == "":
		decoded.Categories = []string{}
	default:
		decoded.Categories = categories
	}

	displayNameMultiPart, existsDisplayName := r.MultipartForm.Value["display_name"]
	if existsDisplayName && len(displayNameMultiPart) > 0 {
		decoded.DisplayName = ptr.String(displayNameMultiPart[0])
		if len(*decoded.DisplayName) > fleet.SoftwareTitleDisplayNameMaxLength {
			return nil, &fleet.BadRequestError{
				Message: "The maximum display name length is 255 characters.",
			}
		}
	}

	// Check if scripts are base64 encoded (to bypass WAF rules that block script patterns)
	if isScriptsEncoded(r) {
		if decoded.InstallScript != nil {
			decodedScript, err := decodeBase64Script(*decoded.InstallScript)
			if err != nil {
				return nil, &fleet.BadRequestError{Message: "invalid base64 encoding for install_script"}
			}
			decoded.InstallScript = &decodedScript
		}
		if decoded.UninstallScript != nil {
			decodedScript, err := decodeBase64Script(*decoded.UninstallScript)
			if err != nil {
				return nil, &fleet.BadRequestError{Message: "invalid base64 encoding for uninstall_script"}
			}
			decoded.UninstallScript = &decodedScript
		}
		if decoded.PreInstallQuery != nil {
			decodedScript, err := decodeBase64Script(*decoded.PreInstallQuery)
			if err != nil {
				return nil, &fleet.BadRequestError{Message: "invalid base64 encoding for pre_install_query"}
			}
			decoded.PreInstallQuery = &decodedScript
		}
		if decoded.PostInstallScript != nil {
			decodedScript, err := decodeBase64Script(*decoded.PostInstallScript)
			if err != nil {
				return nil, &fleet.BadRequestError{Message: "invalid base64 encoding for post_install_script"}
			}
			decoded.PostInstallScript = &decodedScript
		}
	}

	return &decoded, nil
}

func updateSoftwareInstallerEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.UpdateSoftwareInstallerRequest)

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
		Categories:        req.Categories,
		DisplayName:       req.DisplayName,
	}
	if req.File != nil {
		ff, err := req.File.Open()
		if err != nil {
			return fleet.UploadSoftwareInstallerResponse{Err: err}, nil
		}
		defer ff.Close()

		tfr, err := fleet.NewTempFileReader(ff, nil)
		if err != nil {
			return fleet.UploadSoftwareInstallerResponse{Err: err}, nil
		}
		defer tfr.Close()

		payload.InstallerFile = tfr
		payload.Filename = req.File.Filename
	}

	installer, err := svc.UpdateSoftwareInstaller(ctx, payload)
	if err != nil {
		return fleet.UploadSoftwareInstallerResponse{Err: err}, nil
	}

	return fleet.GetSoftwareInstallerResponse{SoftwareInstaller: installer}, nil
}

func (svc *Service) UpdateSoftwareInstaller(ctx context.Context, payload *fleet.UpdateSoftwareInstallerPayload) (*fleet.SoftwareInstaller, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}

// TODO: We parse the whole body before running svc.authz.Authorize.
// An authenticated but unauthorized user could abuse this.
type decodeUploadSoftwareInstallerRequest struct{}

func (decodeUploadSoftwareInstallerRequest) DecodeRequest(ctx context.Context, r *http.Request) (any, error) {
	decoded := fleet.UploadSoftwareInstallerRequest{}

	maxInstallerSize := installersize.FromContext(ctx)
	err := parseMultipartForm(ctx, r, platform_http.MaxMultipartFormSize)
	if err != nil {
		var mbe *http.MaxBytesError
		if errors.As(err, &mbe) {
			return nil, &fleet.BadRequestError{
				Message:     fmt.Sprintf("The maximum file size is %s.", installersize.Human(maxInstallerSize)),
				InternalErr: err,
			}
		}
		var nerr net.Error
		if errors.As(err, &nerr) && nerr.Timeout() {
			return nil, fleet.NewUserMessageError(
				ctxerr.New(ctx, "Couldn't add. Please ensure your internet connection speed is sufficient and stable."),
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
	if decoded.File.Size > maxInstallerSize {
		// Should never happen here since the request's body is limited to the
		// maximum size.
		return nil, &fleet.BadRequestError{
			Message: fmt.Sprintf("The maximum file size is %s.", installersize.Human(maxInstallerSize)),
		}
	}

	// default is no team
	val, ok := r.MultipartForm.Value["fleet_id"]
	if ok {
		fleetID, err := strconv.ParseUint(val[0], 10, 32)
		if err != nil {
			return nil, &fleet.BadRequestError{Message: fmt.Sprintf("Invalid fleet_id: %s", val[0])}
		}
		decoded.TeamID = ptr.Uint(uint(fleetID))
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

	// Check if scripts are base64 encoded (to bypass WAF rules that block script patterns)
	if isScriptsEncoded(r) {
		var err error
		if decoded.InstallScript, err = decodeBase64Script(decoded.InstallScript); err != nil {
			return nil, &fleet.BadRequestError{Message: "invalid base64 encoding for install_script"}
		}
		if decoded.UninstallScript, err = decodeBase64Script(decoded.UninstallScript); err != nil {
			return nil, &fleet.BadRequestError{Message: "invalid base64 encoding for uninstall_script"}
		}
		if decoded.PreInstallQuery, err = decodeBase64Script(decoded.PreInstallQuery); err != nil {
			return nil, &fleet.BadRequestError{Message: "invalid base64 encoding for pre_install_query"}
		}
		if decoded.PostInstallScript, err = decodeBase64Script(decoded.PostInstallScript); err != nil {
			return nil, &fleet.BadRequestError{Message: "invalid base64 encoding for post_install_script"}
		}
	}

	return &decoded, nil
}

func uploadSoftwareInstallerEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.UploadSoftwareInstallerRequest)
	ff, err := req.File.Open()
	if err != nil {
		return fleet.UploadSoftwareInstallerResponse{Err: err}, nil
	}
	defer ff.Close()

	tfr, err := fleet.NewTempFileReader(ff, nil)
	if err != nil {
		return fleet.UploadSoftwareInstallerResponse{Err: err}, nil
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

	installer, err := svc.UploadSoftwareInstaller(ctx, payload)
	if err != nil {
		return fleet.UploadSoftwareInstallerResponse{Err: err}, nil
	}

	return &fleet.UploadSoftwareInstallerResponse{SoftwarePackage: installer}, nil
}

func (svc *Service) UploadSoftwareInstaller(ctx context.Context, payload *fleet.UploadSoftwareInstallerPayload) (*fleet.SoftwareInstaller, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}

func deleteSoftwareInstallerEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.DeleteSoftwareInstallerRequest)
	err := svc.DeleteSoftwareInstaller(ctx, req.TitleID, req.TeamID)
	if err != nil {
		return fleet.DeleteSoftwareInstallerResponse{Err: err}, nil
	}
	return fleet.DeleteSoftwareInstallerResponse{}, nil
}

func (svc *Service) DeleteSoftwareInstaller(ctx context.Context, titleID uint, teamID *uint) error {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return fleet.ErrMissingLicense
}

func getSoftwareInstallerEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.GetSoftwareInstallerRequest)

	payload, err := svc.DownloadSoftwareInstaller(ctx, false, req.Alt, req.TitleID, req.TeamID)
	if err != nil {
		return fleet.OrbitDownloadSoftwareInstallerResponse{Err: err}, nil
	}

	return fleet.OrbitDownloadSoftwareInstallerResponse{Payload: payload}, nil
}

func getSoftwareInstallerTokenEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.GetSoftwareInstallerRequest)

	token, err := svc.GenerateSoftwareInstallerToken(ctx, req.Alt, req.TitleID, req.TeamID)
	if err != nil {
		return fleet.GetSoftwareInstallerTokenResponse{Err: err}, nil
	}
	return fleet.GetSoftwareInstallerTokenResponse{Token: token}, nil
}

func downloadSoftwareInstallerEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.DownloadSoftwareInstallerRequest)

	meta, err := svc.GetSoftwareInstallerTokenMetadata(ctx, req.Token, req.TitleID)
	if err != nil {
		return fleet.OrbitDownloadSoftwareInstallerResponse{Err: err}, nil
	}

	payload, err := svc.DownloadSoftwareInstaller(ctx, true, "media", meta.TitleID, &meta.TeamID)
	if err != nil {
		return fleet.OrbitDownloadSoftwareInstallerResponse{Err: err}, nil
	}

	return fleet.OrbitDownloadSoftwareInstallerResponse{Payload: payload}, nil
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

func (svc *Service) DownloadSoftwareInstaller(ctx context.Context, _ bool, _ string, _ uint,
	_ *uint) (*fleet.DownloadSoftwareInstallerPayload,
	error,
) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}

// Request to install software in a host
func installSoftwareTitleEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.InstallSoftwareRequest)

	err := svc.InstallSoftwareTitle(ctx, req.HostID, req.SoftwareTitleID)
	if err != nil {
		return fleet.InstallSoftwareResponse{Err: err}, nil
	}

	return fleet.InstallSoftwareResponse{}, nil
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

func (svc *Service) InstallVPPAppPostValidation(ctx context.Context, host *fleet.Host, vppApp *fleet.VPPApp, token string, opts fleet.HostSoftwareInstallOptions) (string, error) {
	return "", fleet.ErrMissingLicense // called downstream of auth checks so doesn't need skipauth
}

// Uninstall software
func uninstallSoftwareTitleEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.UninstallSoftwareRequest)

	err := svc.UninstallSoftwareTitle(ctx, req.HostID, req.SoftwareTitleID)
	if err != nil {
		return fleet.InstallSoftwareResponse{Err: err}, nil
	}

	return fleet.InstallSoftwareResponse{}, nil
}

func (svc *Service) UninstallSoftwareTitle(ctx context.Context, _ uint, _ uint) error {
	// skipauth: No authorization check needed due to implementation returning only license error.
	svc.authz.SkipAuthorization(ctx)
	return fleet.ErrMissingLicense
}

// Get software uninstall results (host details and self service)
func getDeviceSoftwareInstallResultsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	_, ok := hostctx.FromContext(ctx)
	if !ok {
		err := ctxerr.Wrap(ctx, fleet.NewAuthRequiredError("internal error: missing host from request context"))
		return fleet.GetSoftwareInstallResultsResponse{Err: err}, nil
	}

	req := request.(*fleet.GetDeviceSoftwareInstallResultsRequest)
	results, err := svc.GetSoftwareInstallResults(ctx, req.InstallUUID)
	if err != nil {
		return fleet.GetSoftwareInstallResultsResponse{Err: err}, nil
	}

	return &fleet.GetSoftwareInstallResultsResponse{Results: results}, nil
}

func getSoftwareInstallResultsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.GetSoftwareInstallResultsRequest)

	results, err := svc.GetSoftwareInstallResults(ctx, req.InstallUUID)
	if err != nil {
		return fleet.GetSoftwareInstallResultsResponse{Err: err}, nil
	}

	return &fleet.GetSoftwareInstallResultsResponse{Results: results}, nil
}

func (svc *Service) GetSoftwareInstallResults(ctx context.Context, resultUUID string) (*fleet.HostSoftwareInstallerResult, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}

// Get software uninstall results from My device page
func getDeviceSoftwareUninstallResultsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	host, ok := hostctx.FromContext(ctx)
	if !ok {
		err := ctxerr.Wrap(ctx, fleet.NewAuthRequiredError("internal error: missing host from request context"))
		return fleet.GetSoftwareInstallResultsResponse{Err: err}, nil
	}

	req := request.(*fleet.GetDeviceSoftwareUninstallResultsRequest)
	scriptResult, err := svc.GetSelfServiceUninstallScriptResult(ctx, host, req.ExecutionID)
	if err != nil {
		return fleet.GetScriptResultResponse{Err: err}, nil
	}

	return setUpGetScriptResultResponse(scriptResult), nil
}

func (svc *Service) GetSelfServiceUninstallScriptResult(ctx context.Context, host *fleet.Host, execID string) (*fleet.HostScriptResult, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}

// Batch replace software installers
func batchSetSoftwareInstallersEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.BatchSetSoftwareInstallersRequest)
	requestUUID, err := svc.BatchSetSoftwareInstallers(ctx, req.TeamName, req.Software, req.DryRun)
	if err != nil {
		return fleet.BatchSetSoftwareInstallersResponse{Err: err}, nil
	}
	return fleet.BatchSetSoftwareInstallersResponse{RequestUUID: requestUUID}, nil
}

func (svc *Service) BatchSetSoftwareInstallers(ctx context.Context, tmName string, payloads []*fleet.SoftwareInstallerPayload, dryRun bool) (string, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return "", fleet.ErrMissingLicense
}

func batchSetSoftwareInstallersResultEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.BatchSetSoftwareInstallersResultRequest)
	status, message, packages, err := svc.GetBatchSetSoftwareInstallersResult(ctx, req.TeamName, req.RequestUUID, req.DryRun)
	if err != nil {
		return fleet.BatchSetSoftwareInstallersResultResponse{Err: err}, nil
	}
	return fleet.BatchSetSoftwareInstallersResultResponse{
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

// Self Service Install
func submitSelfServiceSoftwareInstall(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	host, ok := hostctx.FromContext(ctx)
	if !ok {
		err := ctxerr.Wrap(ctx, fleet.NewAuthRequiredError("internal error: missing host from request context"))
		return fleet.SubmitSelfServiceSoftwareInstallResponse{Err: err}, nil
	}

	req := request.(*fleet.FleetSelfServiceSoftwareInstallRequest)
	if err := svc.SelfServiceInstallSoftwareTitle(ctx, host, req.SoftwareTitleID); err != nil {
		return fleet.SubmitSelfServiceSoftwareInstallResponse{Err: err}, nil
	}

	return fleet.SubmitSelfServiceSoftwareInstallResponse{}, nil
}

func (svc *Service) SelfServiceInstallSoftwareTitle(ctx context.Context, host *fleet.Host, softwareTitleID uint) error {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return fleet.ErrMissingLicense
}

func submitDeviceSoftwareUninstall(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	host, ok := hostctx.FromContext(ctx)
	if !ok {
		err := ctxerr.Wrap(ctx, fleet.NewAuthRequiredError("internal error: missing host from request context"))
		return fleet.SubmitDeviceSoftwareUninstallResponse{Err: err}, nil
	}

	req := request.(*fleet.FleetDeviceSoftwareUninstallRequest)
	if err := svc.UninstallSoftwareTitle(ctx, host.ID, req.SoftwareTitleID); err != nil {
		return fleet.SubmitDeviceSoftwareUninstallResponse{Err: err}, nil
	}

	return fleet.SubmitDeviceSoftwareUninstallResponse{}, nil
}

func (svc *Service) HasSelfServiceSoftwareInstallers(ctx context.Context, host *fleet.Host) (bool, error) {
	alreadyAuthenticated := svc.authz.IsAuthenticatedWith(ctx, authzctx.AuthnDeviceToken) ||
		svc.authz.IsAuthenticatedWith(ctx, authzctx.AuthnDeviceCertificate) ||
		svc.authz.IsAuthenticatedWith(ctx, authzctx.AuthnDeviceURL)
	if !alreadyAuthenticated {
		if err := svc.authz.Authorize(ctx, host, fleet.ActionRead); err != nil {
			return false, err
		}
	}

	return svc.ds.HasSelfServiceSoftwareInstallers(ctx, host.Platform, host.TeamID)
}

// VPP App Store Apps Batch Install
func batchAssociateAppStoreAppsEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.BatchAssociateAppStoreAppsRequest)
	apps, err := svc.BatchAssociateVPPApps(ctx, req.TeamName, req.Apps, req.DryRun)
	if err != nil {
		return fleet.BatchAssociateAppStoreAppsResponse{Err: err}, nil
	}
	return fleet.BatchAssociateAppStoreAppsResponse{Apps: apps}, nil
}

func (svc *Service) BatchAssociateVPPApps(ctx context.Context, teamName string, payloads []fleet.VPPBatchPayload, dryRun bool) ([]fleet.VPPAppResponse, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}

func getInHouseAppManifestEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.GetInHouseAppManifestRequest)
	manifest, err := svc.GetInHouseAppManifest(ctx, req.TitleID, req.TeamID)
	if err != nil {
		return &fleet.GetInHouseAppManifestResponse{Err: err}, nil
	}

	return &fleet.GetInHouseAppManifestResponse{Manifest: manifest}, nil
}

func (svc *Service) GetInHouseAppManifest(ctx context.Context, titleID uint, teamID *uint) ([]byte, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}

func getInHouseAppPackageEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.GetInHouseAppPackageRequest)
	file, err := svc.GetInHouseAppPackage(ctx, req.TitleID, req.TeamID)
	if err != nil {
		return &fleet.GetInHouseAppPackageResponse{Err: err}, nil
	}

	return &fleet.GetInHouseAppPackageResponse{Payload: file}, nil
}

func (svc *Service) GetInHouseAppPackage(ctx context.Context, titleID uint, teamID *uint) (*fleet.DownloadSoftwareInstallerPayload, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}
