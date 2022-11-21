package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"text/template"

	"github.com/docker/go-units"
	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/google/uuid"
	"github.com/groob/plist"
	"github.com/micromdm/micromdm/mdm/appmanifest"
	"github.com/micromdm/nanodep/client"
	"github.com/micromdm/nanodep/godep"
	"github.com/micromdm/nanomdm/mdm"
	"github.com/micromdm/nanomdm/push"
	"github.com/micromdm/nanomdm/storage"
)

type createMDMAppleEnrollmentProfileRequest struct {
	Type       fleet.MDMAppleEnrollmentType `json:"type"`
	DEPProfile *json.RawMessage             `json:"dep_profile"`
}

type createMDMAppleEnrollmentProfileResponse struct {
	EnrollmentProfile *fleet.MDMAppleEnrollmentProfile `json:"enrollment_profile"`
	Err               error                            `json:"error,omitempty"`
}

func (r createMDMAppleEnrollmentProfileResponse) error() error { return r.Err }

func createMDMAppleEnrollmentProfilesEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*createMDMAppleEnrollmentProfileRequest)

	enrollmentProfile, err := svc.NewMDMAppleEnrollmentProfile(ctx, fleet.MDMAppleEnrollmentProfilePayload{
		Type:       req.Type,
		DEPProfile: req.DEPProfile,
	})
	if err != nil {
		return createMDMAppleEnrollmentProfileResponse{
			Err: err,
		}, nil
	}
	return createMDMAppleEnrollmentProfileResponse{
		EnrollmentProfile: enrollmentProfile,
	}, nil
}

func (svc *Service) NewMDMAppleEnrollmentProfile(ctx context.Context, enrollmentPayload fleet.MDMAppleEnrollmentProfilePayload) (*fleet.MDMAppleEnrollmentProfile, error) {
	if err := svc.authz.Authorize(ctx, &fleet.MDMAppleEnrollmentProfile{}, fleet.ActionWrite); err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}

	appConfig, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}

	// generate a token for the profile
	enrollmentPayload.Token = uuid.New().String()

	profile, err := svc.ds.NewMDMAppleEnrollmentProfile(ctx, enrollmentPayload)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}
	if profile.DEPProfile != nil {
		if err := svc.setDEPProfile(ctx, profile, appConfig); err != nil {
			return nil, ctxerr.Wrap(ctx, err)
		}
	}

	profile.EnrollmentURL = svc.mdmAppleEnrollURL(profile.Token, appConfig)

	return profile, nil
}

func (svc *Service) mdmAppleEnrollURL(token string, appConfig *fleet.AppConfig) string {
	return fmt.Sprintf("%s%s?token=%s", appConfig.ServerSettings.ServerURL, apple_mdm.EnrollPath, token)
}

// setDEPProfile define a "DEP profile" on https://mdmenrollment.apple.com and
// sets the returned Profile UUID as the current DEP profile to apply to newly sync DEP devices.
func (svc *Service) setDEPProfile(ctx context.Context, enrollmentProfile *fleet.MDMAppleEnrollmentProfile, appConfig *fleet.AppConfig) error {
	httpClient := fleethttp.NewClient()
	depTransport := client.NewTransport(httpClient.Transport, httpClient, svc.depStorage, nil)
	depClient := client.NewClient(fleethttp.NewClient(), depTransport)

	var depProfileRequest map[string]interface{}
	if err := json.Unmarshal(*enrollmentProfile.DEPProfile, &depProfileRequest); err != nil {
		return fmt.Errorf("invalid DEP profile: %w", err)
	}

	// Override url and configuration_web_url with Fleet's enroll path (publicly accessible address).
	enrollURL := svc.mdmAppleEnrollURL(enrollmentProfile.Token, appConfig)
	depProfileRequest["url"] = enrollURL
	depProfileRequest["configuration_web_url"] = enrollURL
	depProfile, err := json.Marshal(depProfileRequest)
	if err != nil {
		return fmt.Errorf("reserializing DEP profile: %w", err)
	}

	defineProfileRequest, err := client.NewRequestWithContext(
		ctx, apple_mdm.DEPName, svc.depStorage, "POST", "/profile", bytes.NewReader(depProfile),
	)
	if err != nil {
		return fmt.Errorf("create profile request: %w", err)
	}
	defineProfileHTTPResponse, err := depClient.Do(defineProfileRequest)
	if err != nil {
		return fmt.Errorf("exec profile request: %w", err)
	}
	defer defineProfileHTTPResponse.Body.Close()
	if defineProfileHTTPResponse.StatusCode != http.StatusOK {
		return fmt.Errorf("profile request: %s", defineProfileHTTPResponse.Status)
	}
	defineProfileResponseBody, err := io.ReadAll(defineProfileHTTPResponse.Body)
	if err != nil {
		return fmt.Errorf("read profile response: %w", err)
	}
	type depProfileResponseFields struct {
		ProfileUUID string `json:"profile_uuid"`
	}
	defineProfileResponse := depProfileResponseFields{}
	if err := json.Unmarshal(defineProfileResponseBody, &defineProfileResponse); err != nil {
		return fmt.Errorf("parse profile response: %w", err)
	}
	if err := svc.depStorage.StoreAssignerProfile(
		ctx, apple_mdm.DEPName, defineProfileResponse.ProfileUUID,
	); err != nil {
		return fmt.Errorf("set profile UUID: %w", err)
	}
	return nil
}

type listMDMAppleEnrollmentProfilesRequest struct{}

type listMDMAppleEnrollmentProfilesResponse struct {
	EnrollmentProfiles []*fleet.MDMAppleEnrollmentProfile `json:"enrollment_profiles"`
	Err                error                              `json:"error,omitempty"`
}

func (r listMDMAppleEnrollmentProfilesResponse) error() error { return r.Err }

func listMDMAppleEnrollmentsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	enrollmentProfiles, err := svc.ListMDMAppleEnrollmentProfiles(ctx)
	if err != nil {
		return listMDMAppleEnrollmentProfilesResponse{
			Err: err,
		}, nil
	}
	return listMDMAppleEnrollmentProfilesResponse{
		EnrollmentProfiles: enrollmentProfiles,
	}, nil
}

func (svc *Service) ListMDMAppleEnrollmentProfiles(ctx context.Context) ([]*fleet.MDMAppleEnrollmentProfile, error) {
	if err := svc.authz.Authorize(ctx, &fleet.MDMAppleEnrollmentProfile{}, fleet.ActionWrite); err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}

	appConfig, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}

	enrollments, err := svc.ds.ListMDMAppleEnrollmentProfiles(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}
	for i := range enrollments {
		enrollments[i].EnrollmentURL = svc.mdmAppleEnrollURL(enrollments[i].Token, appConfig)
	}
	return enrollments, nil
}

type getMDMAppleCommandResultsRequest struct {
	CommandUUID string `query:"command_uuid,optional"`
}

type getMDMAppleCommandResultsResponse struct {
	Results map[string]*fleet.MDMAppleCommandResult `json:"results,omitempty"`
	Err     error                                   `json:"error,omitempty"`
}

func (r getMDMAppleCommandResultsResponse) error() error { return r.Err }

func getMDMAppleCommandResultsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*getMDMAppleCommandResultsRequest)
	results, err := svc.GetMDMAppleCommandResults(ctx, req.CommandUUID)
	if err != nil {
		return getMDMAppleCommandResultsResponse{
			Err: err,
		}, nil
	}

	return getMDMAppleCommandResultsResponse{
		Results: results,
	}, nil
}

func (svc *Service) GetMDMAppleCommandResults(ctx context.Context, commandUUID string) (map[string]*fleet.MDMAppleCommandResult, error) {
	if err := svc.authz.Authorize(ctx, &fleet.MDMAppleCommandResult{}, fleet.ActionRead); err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}

	results, err := svc.ds.GetMDMAppleCommandResults(ctx, commandUUID)
	if err != nil {
		return nil, err
	}

	return results, nil
}

type uploadAppleInstallerRequest struct {
	Installer *multipart.FileHeader
}

type uploadAppleInstallerResponse struct {
	ID  uint  `json:"installer_id"`
	Err error `json:"error,omitempty"`
}

// TODO(lucas): We parse the whole body before running svc.authz.Authorize.
// An authenticated but unauthorized user could abuse this.
func (uploadAppleInstallerRequest) DecodeRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	err := r.ParseMultipartForm(512 * units.MiB)
	if err != nil {
		return nil, &fleet.BadRequestError{Message: err.Error()}
	}
	installer := r.MultipartForm.File["installer"][0]
	return &uploadAppleInstallerRequest{
		Installer: installer,
	}, nil
}

func (r uploadAppleInstallerResponse) error() error { return r.Err }

func uploadAppleInstallerEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*uploadAppleInstallerRequest)
	ff, err := req.Installer.Open()
	if err != nil {
		return uploadAppleInstallerResponse{Err: err}, nil
	}
	defer ff.Close()
	installer, err := svc.UploadMDMAppleInstaller(ctx, req.Installer.Filename, req.Installer.Size, ff)
	if err != nil {
		return uploadAppleInstallerResponse{Err: err}, nil
	}
	return &uploadAppleInstallerResponse{
		ID: installer.ID,
	}, nil
}

func (svc *Service) UploadMDMAppleInstaller(ctx context.Context, name string, size int64, installer io.Reader) (*fleet.MDMAppleInstaller, error) {
	if err := svc.authz.Authorize(ctx, &fleet.MDMAppleInstaller{}, fleet.ActionWrite); err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}

	appConfig, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}

	token := uuid.New().String()
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}

	url := svc.installerURL(token, appConfig)

	var installerBuf bytes.Buffer
	manifest, err := createManifest(size, io.TeeReader(installer, &installerBuf), url)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}

	inst, err := svc.ds.NewMDMAppleInstaller(ctx, name, size, manifest, installerBuf.Bytes(), token)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}

	return inst, nil
}

func (svc *Service) installerURL(token string, appConfig *fleet.AppConfig) string {
	return fmt.Sprintf("%s%s?token=%s", appConfig.ServerSettings.ServerURL, apple_mdm.InstallerPath, token)
}

func createManifest(size int64, installer io.Reader, url string) (string, error) {
	manifest, err := appmanifest.Create(&readerWithSize{
		Reader: installer,
		size:   size,
	}, url)
	if err != nil {
		return "", fmt.Errorf("create manifest file: %w", err)
	}
	var buf bytes.Buffer
	enc := plist.NewEncoder(&buf)
	enc.Indent("  ")
	if err := enc.Encode(manifest); err != nil {
		return "", fmt.Errorf("encode manifest: %w", err)
	}
	return buf.String(), nil
}

type readerWithSize struct {
	io.Reader
	size int64
}

func (r *readerWithSize) Size() int64 {
	return r.size
}

type getAppleInstallerDetailsRequest struct {
	ID uint `url:"installer_id"`
}

type getAppleInstallerDetailsResponse struct {
	Installer *fleet.MDMAppleInstaller
	Err       error `json:"error,omitempty"`
}

func (r getAppleInstallerDetailsResponse) error() error { return r.Err }

func getAppleInstallerEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*getAppleInstallerDetailsRequest)
	installer, err := svc.GetMDMAppleInstallerByID(ctx, req.ID)
	if err != nil {
		return getAppleInstallerDetailsResponse{Err: err}, nil
	}
	return &getAppleInstallerDetailsResponse{
		Installer: installer,
	}, nil
}

func (svc *Service) GetMDMAppleInstallerByID(ctx context.Context, id uint) (*fleet.MDMAppleInstaller, error) {
	if err := svc.authz.Authorize(ctx, &fleet.MDMAppleInstaller{}, fleet.ActionWrite); err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}

	inst, err := svc.ds.MDMAppleInstallerDetailsByID(ctx, id)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}
	return inst, nil
}

type deleteAppleInstallerDetailsRequest struct {
	ID uint `url:"installer_id"`
}

type deleteAppleInstallerDetailsResponse struct {
	Err error `json:"error,omitempty"`
}

func (r deleteAppleInstallerDetailsResponse) error() error { return r.Err }

func deleteAppleInstallerEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*deleteAppleInstallerDetailsRequest)
	if err := svc.DeleteMDMAppleInstaller(ctx, req.ID); err != nil {
		return deleteAppleInstallerDetailsResponse{Err: err}, nil
	}
	return &deleteAppleInstallerDetailsResponse{}, nil
}

func (svc *Service) DeleteMDMAppleInstaller(ctx context.Context, id uint) error {
	if err := svc.authz.Authorize(ctx, &fleet.MDMAppleInstaller{}, fleet.ActionWrite); err != nil {
		return ctxerr.Wrap(ctx, err)
	}

	if err := svc.ds.DeleteMDMAppleInstaller(ctx, id); err != nil {
		return ctxerr.Wrap(ctx, err)
	}
	return nil
}

type listMDMAppleDevicesRequest struct{}

type listMDMAppleDevicesResponse struct {
	Devices []fleet.MDMAppleDevice `json:"devices"`
	Err     error                  `json:"error,omitempty"`
}

func (r listMDMAppleDevicesResponse) error() error { return r.Err }

func listMDMAppleDevicesEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	devices, err := svc.ListMDMAppleDevices(ctx)
	if err != nil {
		return listMDMAppleDevicesResponse{Err: err}, nil
	}
	return &listMDMAppleDevicesResponse{
		Devices: devices,
	}, nil
}

func (svc *Service) ListMDMAppleDevices(ctx context.Context) ([]fleet.MDMAppleDevice, error) {
	if err := svc.authz.Authorize(ctx, &fleet.MDMAppleDevice{}, fleet.ActionWrite); err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}

	return svc.ds.MDMAppleListDevices(ctx)
}

type listMDMAppleDEPDevicesRequest struct{}

type listMDMAppleDEPDevicesResponse struct {
	Devices []fleet.MDMAppleDEPDevice `json:"devices"`
	Err     error                     `json:"error,omitempty"`
}

func (r listMDMAppleDEPDevicesResponse) error() error { return r.Err }

func listMDMAppleDEPDevicesEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	devices, err := svc.ListMDMAppleDEPDevices(ctx)
	if err != nil {
		return listMDMAppleDEPDevicesResponse{Err: err}, nil
	}
	return &listMDMAppleDEPDevicesResponse{
		Devices: devices,
	}, nil
}

func (svc *Service) ListMDMAppleDEPDevices(ctx context.Context) ([]fleet.MDMAppleDEPDevice, error) {
	if err := svc.authz.Authorize(ctx, &fleet.MDMAppleDEPDevice{}, fleet.ActionWrite); err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}
	depClient := godep.NewClient(svc.depStorage, fleethttp.NewClient())

	// TODO(lucas): Use cursors and limit to fetch in multiple requests.
	// This single-request version supports up to 1000 devices (max to return in one call).
	fetchDevicesResponse, err := depClient.FetchDevices(ctx, apple_mdm.DEPName, godep.WithLimit(1000))
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}

	devices := make([]fleet.MDMAppleDEPDevice, len(fetchDevicesResponse.Devices))
	for i := range fetchDevicesResponse.Devices {
		devices[i] = fleet.MDMAppleDEPDevice{Device: fetchDevicesResponse.Devices[i]}
	}
	return devices, nil
}

type enqueueMDMAppleCommandRequest struct {
	Command   string   `json:"command"`
	DeviceIDs []string `json:"device_ids"`
	NoPush    bool     `json:"no_push"`
}

type enqueueMDMAppleCommandResponse struct {
	status int                        `json:"-"`
	Result fleet.CommandEnqueueResult `json:"result"`
	Err    error                      `json:"error,omitempty"`
}

func (r enqueueMDMAppleCommandResponse) error() error { return r.Err }
func (r enqueueMDMAppleCommandResponse) Status() int  { return r.status }

func enqueueMDMAppleCommandEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*enqueueMDMAppleCommandRequest)
	rawCommand, err := base64.RawStdEncoding.DecodeString(req.Command)
	if err != nil {
		return enqueueMDMAppleCommandResponse{Err: err}, nil
	}
	command, err := mdm.DecodeCommand(rawCommand)
	if err != nil {
		return enqueueMDMAppleCommandResponse{Err: err}, nil
	}
	status, result, err := svc.EnqueueMDMAppleCommand(ctx, &fleet.MDMAppleCommand{Command: command}, req.DeviceIDs, req.NoPush)
	if err != nil {
		return enqueueMDMAppleCommandResponse{Err: err}, nil
	}
	return enqueueMDMAppleCommandResponse{
		status: status,
		Result: *result,
	}, nil
}

func (svc *Service) EnqueueMDMAppleCommand(
	ctx context.Context,
	command *fleet.MDMAppleCommand,
	deviceIDs []string,
	noPush bool,
) (status int, result *fleet.CommandEnqueueResult, err error) {
	if err := svc.authz.Authorize(ctx, command, fleet.ActionWrite); err != nil {
		return 0, nil, ctxerr.Wrap(ctx, err)
	}
	return rawCommandEnqueue(ctx, svc.mdmStorage, svc.mdmPushService, command.Command, deviceIDs, noPush, svc.logger)
}

// rawCommandEnqueue enqueues a command to be executed on the given devices.
//
// This method was extracted from:
// https://github.com/fleetdm/nanomdm/blob/a261f081323c80fb7f6575a64ac1a912dffe44ba/http/api/api.go#L134-L261
// NOTE(lucas): At the time, I found no way to reuse Fleet's gokit middlewares with a raw http.Handler
// like api.RawCommandEnqueueHandler.
func rawCommandEnqueue(
	ctx context.Context,
	enqueuer storage.CommandEnqueuer,
	pusher push.Pusher,
	command *mdm.Command,
	deviceIDs []string,
	noPush bool,
	logger kitlog.Logger,
) (status int, result *fleet.CommandEnqueueResult, err error) {
	output := fleet.CommandEnqueueResult{
		Status:      make(fleet.EnrolledAPIResults),
		NoPush:      noPush,
		CommandUUID: command.CommandUUID,
		RequestType: command.Command.RequestType,
	}

	logger = kitlog.With(
		logger,
		"command_uuid", command.CommandUUID,
		"request_type", command.Command.RequestType,
	)
	logs := []interface{}{
		"msg", "enqueue",
	}
	idErrs, err := enqueuer.EnqueueCommand(ctx, deviceIDs, command)
	ct := len(deviceIDs) - len(idErrs)
	if err != nil {
		logs = append(logs, "err", err)
		output.CommandError = err.Error()
		if len(idErrs) == 0 {
			// we assume if there were no ID-specific errors but
			// there was a general error then all IDs failed
			ct = 0
		}
	}
	logs = append(logs, "count", ct)
	if len(idErrs) > 0 {
		logs = append(logs, "errs", len(idErrs))
	}
	if err != nil || len(idErrs) > 0 {
		level.Info(logger).Log(logs...)
	} else {
		level.Debug(logger).Log(logs...)
	}
	// loop through our command errors, if any, and add to output
	for id, err := range idErrs {
		if err != nil {
			output.Status[id] = &fleet.EnrolledAPIResult{
				CommandError: err.Error(),
			}
		}
	}
	// optionally send pushes
	pushResp := make(map[string]*push.Response)
	var pushErr error
	if !noPush {
		pushResp, pushErr = pusher.Push(ctx, deviceIDs)
		if err != nil {
			level.Info(logger).Log("msg", "push", "err", err)
			output.PushError = err.Error()
		}
	} else {
		pushErr = nil
	}
	// loop through our push errors, if any, and add to output
	var pushCt, pushErrCt int
	for id, resp := range pushResp {
		if _, ok := output.Status[id]; ok {
			output.Status[id].PushResult = resp.Id
		} else {
			output.Status[id] = &fleet.EnrolledAPIResult{
				PushResult: resp.Id,
			}
		}
		if resp.Err != nil {
			output.Status[id].PushError = resp.Err.Error()
			pushErrCt++
		} else {
			pushCt++
		}
	}
	logs = []interface{}{
		"msg", "push",
		"count", pushCt,
	}
	if pushErr != nil {
		logs = append(logs, "err", pushErr)
	}
	if pushErrCt > 0 {
		logs = append(logs, "errs", pushErrCt)
	}
	if pushErr != nil || pushErrCt > 0 {
		level.Info(logger).Log(logs...)
	} else {
		level.Debug(logger).Log(logs...)
	}
	// generate response codes depending on if everything succeeded, failed, or parially succedded
	header := http.StatusInternalServerError
	if (len(idErrs) > 0 || err != nil || (!noPush && (pushErrCt > 0 || pushErr != nil))) && (ct > 0 || (!noPush && (pushCt > 0))) {
		header = http.StatusMultiStatus
	} else if (len(idErrs) == 0 && err == nil && (noPush || (pushErrCt == 0 && pushErr == nil))) && (ct >= 1 && (noPush || (pushCt >= 1))) {
		header = http.StatusOK
	}
	return header, &output, nil
}

type mdmAppleEnrollRequest struct {
	Token string `query:"token"`
}

func (r mdmAppleEnrollResponse) error() error { return r.Err }

type mdmAppleEnrollResponse struct {
	Err error `json:"error,omitempty"`

	// Profile field is used in hijackRender for the response.
	Profile []byte
}

func (r mdmAppleEnrollResponse) hijackRender(ctx context.Context, w http.ResponseWriter) {
	w.Header().Set("Content-Length", strconv.FormatInt(int64(len(r.Profile)), 10))
	w.Header().Set("Content-Type", "application/x-apple-aspen-config")

	// OK to just log the error here as writing anything on
	// `http.ResponseWriter` sets the status code to 200 (and it can't be
	// changed.) Clients should rely on matching content-length with the
	// header provided.
	if n, err := w.Write(r.Profile); err != nil {
		logging.WithExtras(ctx, "err", err, "written", n)
	}
}

func mdmAppleEnrollEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*mdmAppleEnrollRequest)

	profile, err := svc.GetMDMAppleEnrollmentProfileByToken(ctx, req.Token)
	if err != nil {
		return mdmAppleEnrollResponse{Err: err}, nil
	}
	return mdmAppleEnrollResponse{
		Profile: profile,
	}, nil
}

func (svc *Service) GetMDMAppleEnrollmentProfileByToken(ctx context.Context, token string) (profile []byte, err error) {
	// skipauth: The enroll profile endpoint is unauthenticated.
	svc.authz.SkipAuthorization(ctx)

	_, err = svc.ds.GetMDMAppleEnrollmentProfileByToken(ctx, token)
	if err != nil {
		if fleet.IsNotFound(err) {
			return nil, fleet.NewAuthFailedError("enrollment profile not found")
		}
		return nil, ctxerr.Wrap(ctx, err, "get enrollment profile")
	}

	appConfig, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}

	// TODO(lucas): Actually use enrollment (when we define which configuration we want to define
	// on enrollments).
	mobileconfig, err := generateEnrollmentProfileMobileconfig(
		appConfig.OrgInfo.OrgName,
		appConfig.ServerSettings.ServerURL,
		svc.config.MDMApple.SCEP.Challenge,
		svc.mdmPushCertTopic,
	)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}
	return mobileconfig, nil
}

// enrollmentProfileMobileconfigTemplate is the template Fleet uses to assemble a .mobileconfig enrollment profile to serve to devices.
//
// During a profile replacement, the system updates payloads with the same PayloadIdentifier and
// PayloadUUID in the old and new profiles.
var enrollmentProfileMobileconfigTemplate = template.Must(template.New("").Parse(`
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>PayloadContent</key>
	<array>
		<dict>
			<key>PayloadContent</key>
			<dict>
				<key>Key Type</key>
				<string>RSA</string>
				<key>Challenge</key>
				<string>{{ .SCEPChallenge }}</string>
				<key>Key Usage</key>
				<integer>5</integer>
				<key>Keysize</key>
				<integer>2048</integer>
				<key>URL</key>
				<string>{{ .SCEPURL }}</string>
				<key>Subject</key>
				<array>
					<array><array><string>O</string><string>FleetDM</string></array></array>
					<array><array><string>CN</string><string>FleetDM Identity</string></array></array>
				</array>
			</dict>
			<key>PayloadIdentifier</key>
			<string>com.fleetdm.fleet.mdm.apple.scep</string>
			<key>PayloadType</key>
			<string>com.apple.security.scep</string>
			<key>PayloadUUID</key>
			<string>BCA53F9D-5DD2-494D-98D3-0D0F20FF6BA1</string>
			<key>PayloadVersion</key>
			<integer>1</integer>
		</dict>
		<dict>
			<key>AccessRights</key>
			<integer>8191</integer>
			<key>CheckOutWhenRemoved</key>
			<true/>
			<key>IdentityCertificateUUID</key>
			<string>BCA53F9D-5DD2-494D-98D3-0D0F20FF6BA1</string>
			<key>PayloadIdentifier</key>
			<string>com.fleetdm.fleet.mdm.apple.mdm</string>
			<key>PayloadType</key>
			<string>com.apple.mdm</string>
			<key>PayloadUUID</key>
			<string>29713130-1602-4D27-90C9-B822A295E44E</string>
			<key>PayloadVersion</key>
			<integer>1</integer>
			<key>ServerCapabilities</key>
			<array>
				<string>com.apple.mdm.per-user-connections</string>
			</array>
			<key>ServerURL</key>
			<string>{{ .ServerURL }}</string>
			<key>SignMessage</key>
			<true/>
			<key>Topic</key>
			<string>{{ .Topic }}</string>
		</dict>
	</array>
	<key>PayloadDisplayName</key>
	<string>{{ .Organization }} Enrollment</string>
	<key>PayloadIdentifier</key>
	<string>com.fleetdm.fleet.mdm.apple</string>
	<key>PayloadScope</key>
	<string>System</string>
	<key>PayloadType</key>
	<string>Configuration</string>
	<key>PayloadUUID</key>
	<string>5ACABE91-CE30-4C05-93E3-B235C152404E</string>
	<key>PayloadVersion</key>
	<integer>1</integer>
</dict>
</plist>`))

func generateEnrollmentProfileMobileconfig(orgName, fleetURL, scepChallenge, topic string) ([]byte, error) {
	scepURL := fleetURL + apple_mdm.SCEPPath
	serverURL := fleetURL + apple_mdm.MDMPath

	var buf bytes.Buffer
	if err := enrollmentProfileMobileconfigTemplate.Execute(&buf, struct {
		Organization  string
		SCEPURL       string
		SCEPChallenge string
		Topic         string
		ServerURL     string
	}{
		Organization:  orgName,
		SCEPURL:       scepURL,
		SCEPChallenge: scepChallenge,
		Topic:         topic,
		ServerURL:     serverURL,
	}); err != nil {
		return nil, fmt.Errorf("execute template: %w", err)
	}
	return buf.Bytes(), nil
}

type mdmAppleGetInstallerRequest struct {
	Token string `query:"token"`
}

func (r mdmAppleGetInstallerResponse) error() error { return r.Err }

type mdmAppleGetInstallerResponse struct {
	Err error `json:"error,omitempty"`

	// head is used by hijackRender for the response.
	head bool
	// Name field is used in hijackRender for the response.
	name string
	// Size field is used in hijackRender for the response.
	size int64
	// Installer field is used in hijackRender for the response.
	installer []byte
}

func (r mdmAppleGetInstallerResponse) hijackRender(ctx context.Context, w http.ResponseWriter) {
	w.Header().Set("Content-Length", strconv.FormatInt(r.size, 10))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment;filename="%s"`, r.name))

	if r.head {
		w.WriteHeader(http.StatusOK)
		return
	}

	// OK to just log the error here as writing anything on
	// `http.ResponseWriter` sets the status code to 200 (and it can't be
	// changed.) Clients should rely on matching content-length with the
	// header provided
	if n, err := w.Write(r.installer); err != nil {
		logging.WithExtras(ctx, "err", err, "bytes_copied", n)
	}
}

func mdmAppleGetInstallerEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*mdmAppleGetInstallerRequest)
	installer, err := svc.GetMDMAppleInstallerByToken(ctx, req.Token)
	if err != nil {
		return mdmAppleGetInstallerResponse{Err: err}, nil
	}
	return mdmAppleGetInstallerResponse{
		head:      false,
		name:      installer.Name,
		size:      installer.Size,
		installer: installer.Installer,
	}, nil
}

func (svc *Service) GetMDMAppleInstallerByToken(ctx context.Context, token string) (*fleet.MDMAppleInstaller, error) {
	// skipauth: The installer endpoint uses token authentication.
	svc.authz.SkipAuthorization(ctx)

	installer, err := svc.ds.MDMAppleInstaller(ctx, token)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}
	return installer, nil
}

type mdmAppleHeadInstallerRequest struct {
	Token string `query:"token"`
}

func mdmAppleHeadInstallerEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*mdmAppleHeadInstallerRequest)
	installer, err := svc.GetMDMAppleInstallerDetailsByToken(ctx, req.Token)
	if err != nil {
		return mdmAppleGetInstallerResponse{Err: err}, nil
	}
	return mdmAppleGetInstallerResponse{
		head: true,
		name: installer.Name,
		size: installer.Size,
	}, nil
}

func (svc *Service) GetMDMAppleInstallerDetailsByToken(ctx context.Context, token string) (*fleet.MDMAppleInstaller, error) {
	// skipauth: The installer endpoint uses token authentication.
	svc.authz.SkipAuthorization(ctx)

	installer, err := svc.ds.MDMAppleInstallerDetailsByToken(ctx, token)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}
	return installer, nil
}

type listMDMAppleInstallersRequest struct{}

type listMDMAppleInstallersResponse struct {
	Installers []fleet.MDMAppleInstaller `json:"installers"`
	Err        error                     `json:"error,omitempty"`
}

func (r listMDMAppleInstallersResponse) error() error { return r.Err }

func listMDMAppleInstallersEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	installers, err := svc.ListMDMAppleInstallers(ctx)
	if err != nil {
		return listMDMAppleInstallersResponse{
			Err: err,
		}, nil
	}
	return listMDMAppleInstallersResponse{
		Installers: installers,
	}, nil
}

func (svc *Service) ListMDMAppleInstallers(ctx context.Context) ([]fleet.MDMAppleInstaller, error) {
	if err := svc.authz.Authorize(ctx, &fleet.MDMAppleInstaller{}, fleet.ActionWrite); err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}

	appConfig, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}

	installers, err := svc.ds.ListMDMAppleInstallers(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}
	for i := range installers {
		installers[i].URL = svc.installerURL(installers[i].URLToken, appConfig)
	}
	return installers, nil
}
