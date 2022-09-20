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
	"github.com/google/uuid"
	"github.com/groob/plist"
	"github.com/micromdm/micromdm/mdm/appmanifest"
	"github.com/micromdm/nanodep/client"
	"github.com/micromdm/nanodep/godep"
	"github.com/micromdm/nanomdm/cryptoutil"
	nanomdm_log "github.com/micromdm/nanomdm/log"
	"github.com/micromdm/nanomdm/mdm"
	"github.com/micromdm/nanomdm/push"
	"github.com/micromdm/nanomdm/storage"
)

type createMDMAppleEnrollmentRequest struct {
	Name      string           `json:"name"`
	DEPConfig *json.RawMessage `json:"dep_config"`
}

type createMDMAppleEnrollmentResponse struct {
	Enrollment *fleet.MDMAppleEnrollment `json:"enrollment"`
	Err        error                     `json:"error,omitempty"`
}

func (r createMDMAppleEnrollmentResponse) error() error { return r.Err }

func createMDMAppleEnrollmentEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	req := request.(*createMDMAppleEnrollmentRequest)
	enrollment, err := svc.NewMDMAppleEnrollment(ctx, fleet.MDMAppleEnrollmentPayload{
		Name:      req.Name,
		DEPConfig: req.DEPConfig,
	})
	if err != nil {
		return createMDMAppleEnrollmentResponse{
			Err: err,
		}, nil
	}
	return createMDMAppleEnrollmentResponse{
		Enrollment: enrollment,
	}, nil
}

func (svc *Service) NewMDMAppleEnrollment(ctx context.Context, enrollmentPayload fleet.MDMAppleEnrollmentPayload) (*fleet.MDMAppleEnrollment, error) {
	if err := svc.authz.Authorize(ctx, &fleet.MDMAppleEnrollment{}, fleet.ActionWrite); err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}

	enrollment, err := svc.ds.NewMDMAppleEnrollment(ctx, enrollmentPayload)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}
	if enrollment.DEPConfig != nil {
		if err := svc.setDEPProfile(ctx, enrollment); err != nil {
			return nil, ctxerr.Wrap(ctx, err)
		}
	}
	enrollment.URL = svc.mdmAppleEnrollURL(enrollment.ID)
	return enrollment, nil
}

func (svc *Service) mdmAppleEnrollURL(enrollmentID uint) string {
	return fmt.Sprintf("https://%s%s?id=%d", svc.config.MDMApple.ServerAddress, apple_mdm.EnrollPath, enrollmentID)
}

// setDEPProfile define a "DEP profile" on https://mdmenrollment.apple.com and
// sets the returned Profile UUID as the current DEP profile to apply to newly sync DEP devices.
func (svc *Service) setDEPProfile(ctx context.Context, enrollment *fleet.MDMAppleEnrollment) error {
	httpClient := fleethttp.NewClient()
	depTransport := client.NewTransport(httpClient.Transport, httpClient, svc.depStorage, nil)
	depClient := client.NewClient(fleethttp.NewClient(), depTransport)

	// TODO(lucas): Currently overriding the `url` and `configuration_web_url`.
	// We need to actually expose configuration.
	var depProfileRequest map[string]interface{}
	if err := json.Unmarshal(*enrollment.DEPConfig, &depProfileRequest); err != nil {
		return fmt.Errorf("invalid DEP profile: %w", err)
	}
	enrollURL := svc.mdmAppleEnrollURL(enrollment.ID)
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

type listMDMAppleEnrollmentsRequest struct{}

type listMDMAppleEnrollmentsResponse struct {
	Enrollments []fleet.MDMAppleEnrollment `json:"enrollments"`
	Err         error                      `json:"error,omitempty"`
}

func (r listMDMAppleEnrollmentsResponse) error() error { return r.Err }

func listMDMAppleEnrollmentsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (interface{}, error) {
	enrollments, err := svc.ListMDMAppleEnrollments(ctx)
	if err != nil {
		return listMDMAppleEnrollmentsResponse{
			Err: err,
		}, nil
	}
	return listMDMAppleEnrollmentsResponse{
		Enrollments: enrollments,
	}, nil
}

func (svc *Service) ListMDMAppleEnrollments(ctx context.Context) ([]fleet.MDMAppleEnrollment, error) {
	if err := svc.authz.Authorize(ctx, &fleet.MDMAppleEnrollment{}, fleet.ActionWrite); err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}

	enrollments, err := svc.ds.ListMDMAppleEnrollments(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}
	for i := range enrollments {
		enrollments[i].URL = svc.mdmAppleEnrollURL(enrollments[i].ID)
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

	urlToken, err := uuid.NewRandom()
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}
	var installerBuf bytes.Buffer
	url := svc.installerURL(urlToken.String())
	manifest, err := createManifest(size, io.TeeReader(installer, &installerBuf), url)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}
	inst, err := svc.ds.NewMDMAppleInstaller(ctx, name, size, manifest, installerBuf.Bytes(), urlToken.String())
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}
	return inst, nil
}

func (svc *Service) installerURL(urlToken string) string {
	return "https://" + svc.config.MDMApple.ServerAddress + apple_mdm.InstallerPath + "?token=" + urlToken
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

	// TODO(lucas): Use cursors and limit.
	fetchDevicesResponse, err := depClient.FetchDevices(ctx, apple_mdm.DEPName)
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
	ctx context.Context, command *fleet.MDMAppleCommand, deviceIDs []string, noPush bool,
) (status int, result *fleet.CommandEnqueueResult, err error) {
	if err := svc.authz.Authorize(ctx, command, fleet.ActionWrite); err != nil {
		return 0, nil, ctxerr.Wrap(ctx, err)
	}
	return rawCommandEnqueue(ctx, svc.mdmStorage, svc.mdmPushService, command.Command, deviceIDs, noPush, svc.mdmLogger)
}

// Copied from https://github.com/fleetdm/nanomdm/blob/a261f081323c80fb7f6575a64ac1a912dffe44ba/http/api/api.go#L134-L261
// NOTE(lucas): I found no way to reuse Fleet's gokit middlewares with a raw http.Handler like api.RawCommandEnqueueHandler.
func rawCommandEnqueue(
	ctx context.Context,
	enqueuer storage.CommandEnqueuer,
	pusher push.Pusher,
	command *mdm.Command,
	deviceIDs []string,
	noPush bool,
	logger nanomdm_log.Logger,
) (status int, result *fleet.CommandEnqueueResult, err error) {
	output := fleet.CommandEnqueueResult{
		Status:      make(fleet.EnrolledAPIResults),
		NoPush:      noPush,
		CommandUUID: command.CommandUUID,
		RequestType: command.Command.RequestType,
	}

	logger = logger.With(
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
		logger.Info(logs...)
	} else {
		logger.Debug(logs...)
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
			logger.Info("msg", "push", "err", err)
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
		logger.Info(logs...)
	} else {
		logger.Debug(logs...)
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
	EnrollmentID uint `query:"id"`
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
	profile, err := svc.GetMDMAppleEnrollProfile(ctx, req.EnrollmentID)
	if err != nil {
		return mdmAppleEnrollResponse{Err: err}, nil
	}
	return mdmAppleEnrollResponse{
		Profile: profile,
	}, nil
}

func (svc *Service) GetMDMAppleEnrollProfile(ctx context.Context, enrollmentID uint) (profile []byte, err error) {
	// skipauth: The enroll profile endpoint is unauthenticated.
	svc.authz.SkipAuthorization(ctx)

	topic, err := cryptoutil.TopicFromPEMCert(svc.config.MDMApple.MDM.PushCert.PEMCert)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}
	_, err = svc.ds.MDMAppleEnrollment(ctx, enrollmentID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}
	// TODO(lucas): Actually use enrollment (when we define which configuration we want to define
	// on enrollments).
	mobileConfig, err := generateMobileConfig(
		"https://"+svc.config.MDMApple.ServerAddress+apple_mdm.SCEPPath,
		"https://"+svc.config.MDMApple.ServerAddress+apple_mdm.MDMPath,
		svc.config.MDMApple.SCEP.Challenge,
		topic,
	)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}
	fmt.Printf("%s\n", mobileConfig)
	return mobileConfig, nil
}

// mobileConfigTemplate is the template Fleet uses to assemble a .mobileconfig enroll profile to serve to devices.
//
// TODO(lucas): Tweak the remaining configuration.
// Downloaded from:
// https://github.com/micromdm/nanomdm/blob/3b1eb0e4e6538b6644633b18dedc6d8645853cb9/docs/enroll.mobileconfig
//
// TODO(lucas): Support enroll profile signing?
var mobileConfigTemplate = template.Must(template.New(".mobileconfig").Parse(`
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
				<string>{{ .SCEPServerURL }}</string>
			</dict>
			<key>PayloadIdentifier</key>
			<string>com.github.micromdm.scep</string>
			<key>PayloadType</key>
			<string>com.apple.security.scep</string>
			<key>PayloadUUID</key>
			<string>CB90E976-AD44-4B69-8108-8095E6260978</string>
			<key>PayloadVersion</key>
			<integer>1</integer>
		</dict>
		<dict>
			<key>AccessRights</key>
			<integer>8191</integer>
			<key>CheckOutWhenRemoved</key>
			<true/>
			<key>IdentityCertificateUUID</key>
			<string>CB90E976-AD44-4B69-8108-8095E6260978</string>
			<key>PayloadIdentifier</key>
			<string>com.github.micromdm.nanomdm.mdm</string>
			<key>PayloadType</key>
			<string>com.apple.mdm</string>
			<key>PayloadUUID</key>
			<string>96B11019-B54C-49DC-9480-43525834DE7B</string>
			<key>PayloadVersion</key>
			<integer>1</integer>
			<key>ServerCapabilities</key>
			<array>
				<string>com.apple.mdm.per-user-connections</string>
			</array>
			<key>ServerURL</key>
			<string>{{ .MDMServerURL }}</string>
			<key>SignMessage</key>
			<true/>
			<key>Topic</key>
			<string>{{ .Topic }}</string>
		</dict>
	</array>
	<key>PayloadDisplayName</key>
	<string>Enrollment Profile</string>
	<key>PayloadIdentifier</key>
	<string>com.github.micromdm.nanomdm</string>
	<key>PayloadScope</key>
	<string>System</string>
	<key>PayloadType</key>
	<string>Configuration</string>
	<key>PayloadUUID</key>
	<string>F9760DD4-F2D1-4F29-8D2C-48D52DD0A9B3</string>
	<key>PayloadVersion</key>
	<integer>1</integer>
</dict>
</plist>`))

func generateMobileConfig(scepServerURL, mdmServerURL, scepChallenge, topic string) ([]byte, error) {
	var contents bytes.Buffer
	if err := mobileConfigTemplate.Execute(&contents, struct {
		SCEPServerURL string
		MDMServerURL  string
		SCEPChallenge string
		Topic         string
	}{
		SCEPServerURL: scepServerURL,
		MDMServerURL:  mdmServerURL,
		SCEPChallenge: scepChallenge,
		Topic:         topic,
	}); err != nil {
		return nil, fmt.Errorf("execute template: %w", err)
	}
	return contents.Bytes(), nil
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
		head:      true,
		name:      installer.Name,
		size:      installer.Size,
		installer: installer.Installer,
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

	installers, err := svc.ds.ListMDMAppleInstallers(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}
	for i := range installers {
		installers[i].URL = svc.installerURL(installers[i].URLToken)
	}
	return installers, nil
}
