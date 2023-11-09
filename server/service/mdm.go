package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/VividCortex/mysqlerr"
	"github.com/docker/go-units"
	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/go-kit/kit/log/level"
	"github.com/go-sql-driver/mysql"
	"github.com/micromdm/nanomdm/mdm"
)

////////////////////////////////////////////////////////////////////////////////
// GET /mdm/apple
////////////////////////////////////////////////////////////////////////////////

type getAppleMDMResponse struct {
	*fleet.AppleMDM
	Err error `json:"error,omitempty"`
}

func (r getAppleMDMResponse) error() error { return r.Err }

func getAppleMDMEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	appleMDM, err := svc.GetAppleMDM(ctx)
	if err != nil {
		return getAppleMDMResponse{Err: err}, nil
	}

	return getAppleMDMResponse{AppleMDM: appleMDM}, nil
}

func (svc *Service) GetAppleMDM(ctx context.Context) (*fleet.AppleMDM, error) {
	if err := svc.authz.Authorize(ctx, &fleet.AppleMDM{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	// if there is no apple mdm config, fail with a 404
	if !svc.config.MDM.IsAppleAPNsSet() {
		return nil, newNotFoundError()
	}

	apns, _, _, err := svc.config.MDM.AppleAPNs()
	if err != nil {
		return nil, err
	}

	appleMDM := &fleet.AppleMDM{
		CommonName: apns.Leaf.Subject.CommonName,
		Issuer:     apns.Leaf.Issuer.CommonName,
		RenewDate:  apns.Leaf.NotAfter,
	}
	if apns.Leaf.SerialNumber != nil {
		appleMDM.SerialNumber = apns.Leaf.SerialNumber.String()
	}

	return appleMDM, nil
}

////////////////////////////////////////////////////////////////////////////////
// GET /mdm/apple_bm
////////////////////////////////////////////////////////////////////////////////

type getAppleBMResponse struct {
	*fleet.AppleBM
	Err error `json:"error,omitempty"`
}

func (r getAppleBMResponse) error() error { return r.Err }

func getAppleBMEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	appleBM, err := svc.GetAppleBM(ctx)
	if err != nil {
		return getAppleBMResponse{Err: err}, nil
	}

	return getAppleBMResponse{AppleBM: appleBM}, nil
}

func (svc *Service) GetAppleBM(ctx context.Context) (*fleet.AppleBM, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}

////////////////////////////////////////////////////////////////////////////////
// GET /mdm/apple/request_csr
////////////////////////////////////////////////////////////////////////////////

type requestMDMAppleCSRRequest struct {
	EmailAddress string `json:"email_address"`
	Organization string `json:"organization"`
}

type requestMDMAppleCSRResponse struct {
	*fleet.AppleCSR
	Err error `json:"error,omitempty"`
}

func (r requestMDMAppleCSRResponse) error() error { return r.Err }

func requestMDMAppleCSREndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*requestMDMAppleCSRRequest)

	csr, err := svc.RequestMDMAppleCSR(ctx, req.EmailAddress, req.Organization)
	if err != nil {
		return requestMDMAppleCSRResponse{Err: err}, nil
	}
	return requestMDMAppleCSRResponse{
		AppleCSR: csr,
	}, nil
}

func (svc *Service) RequestMDMAppleCSR(ctx context.Context, email, org string) (*fleet.AppleCSR, error) {
	if err := svc.authz.Authorize(ctx, &fleet.AppleCSR{}, fleet.ActionWrite); err != nil {
		return nil, err
	}

	if err := fleet.ValidateEmail(email); err != nil {
		if strings.TrimSpace(email) == "" {
			return nil, ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("email_address", "missing email address"))
		}
		return nil, ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("email_address", fmt.Sprintf("invalid email address: %v", err)))
	}
	if strings.TrimSpace(org) == "" {
		return nil, ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("organization", "missing organization"))
	}

	// create the raw SCEP CA cert and key (creating before the CSR signing
	// request so that nothing can fail after the request is made, except for the
	// network during the response of course)
	scepCACert, scepCAKey, err := apple_mdm.NewSCEPCACertKey()
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "generate SCEP CA cert and key")
	}

	// create the APNs CSR
	apnsCSR, apnsKey, err := apple_mdm.GenerateAPNSCSRKey(email, org)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "generate APNs CSR")
	}

	// request the signed APNs CSR from fleetdm.com
	client := fleethttp.NewClient(fleethttp.WithTimeout(10 * time.Second))
	if err := apple_mdm.GetSignedAPNSCSR(client, apnsCSR); err != nil {
		if ferr, ok := err.(apple_mdm.FleetWebsiteError); ok {
			status := http.StatusBadGateway
			if ferr.Status >= 400 && ferr.Status <= 499 {
				// TODO: fleetdm.com returns a genereric "Bad
				// Request" message, we should coordinate and
				// stablish a response schema from which we can get
				// the invalid field and use
				// fleet.NewInvalidArgumentError instead
				//
				// For now, since we have already validated
				// everything else, we assume that a 4xx
				// response is an email with an invalid domain
				return nil, ctxerr.Wrap(
					ctx,
					fleet.NewInvalidArgumentError(
						"email_address",
						fmt.Sprintf("this email address is not valid: %v", err),
					),
				)
			}
			return nil, ctxerr.Wrap(
				ctx,
				fleet.NewUserMessageError(
					fmt.Errorf("FleetDM CSR request failed: %w", err),
					status,
				),
			)
		}

		return nil, ctxerr.Wrap(ctx, err, "get signed CSR")
	}

	// PEM-encode the cert and keys
	scepCACertPEM := apple_mdm.EncodeCertPEM(scepCACert)
	scepCAKeyPEM := apple_mdm.EncodePrivateKeyPEM(scepCAKey)
	apnsKeyPEM := apple_mdm.EncodePrivateKeyPEM(apnsKey)

	return &fleet.AppleCSR{
		APNsKey:  apnsKeyPEM,
		SCEPCert: scepCACertPEM,
		SCEPKey:  scepCAKeyPEM,
	}, nil
}

func (svc *Service) VerifyMDMAppleConfigured(ctx context.Context) error {
	appCfg, err := svc.ds.AppConfig(ctx)
	if err != nil {
		// skipauth: Authorization is currently for user endpoints only.
		svc.authz.SkipAuthorization(ctx)
		return err
	}
	if !appCfg.MDM.EnabledAndConfigured {
		// skipauth: Authorization is currently for user endpoints only.
		svc.authz.SkipAuthorization(ctx)
		return fleet.ErrMDMNotConfigured
	}

	return nil
}

////////////////////////////////////////////////////////////////////////////////
// POST /mdm/apple/setup/eula
////////////////////////////////////////////////////////////////////////////////

type createMDMAppleEULARequest struct {
	EULA *multipart.FileHeader
}

// TODO: We parse the whole body before running svc.authz.Authorize.
// An authenticated but unauthorized user could abuse this.
func (createMDMAppleEULARequest) DecodeRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	err := r.ParseMultipartForm(512 * units.MiB)
	if err != nil {
		return nil, &fleet.BadRequestError{
			Message:     "failed to parse multipart form",
			InternalErr: err,
		}
	}

	if r.MultipartForm.File["eula"] == nil {
		return nil, &fleet.BadRequestError{
			Message:     "eula multipart field is required",
			InternalErr: err,
		}
	}

	return &createMDMAppleEULARequest{
		EULA: r.MultipartForm.File["eula"][0],
	}, nil
}

type createMDMAppleEULAResponse struct {
	Err error `json:"error,omitempty"`
}

func (r createMDMAppleEULAResponse) error() error { return r.Err }

func createMDMAppleEULAEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*createMDMAppleEULARequest)
	ff, err := req.EULA.Open()
	if err != nil {
		return createMDMAppleEULAResponse{Err: err}, nil
	}
	defer ff.Close()

	if err := svc.MDMAppleCreateEULA(ctx, req.EULA.Filename, ff); err != nil {
		return createMDMAppleEULAResponse{Err: err}, nil
	}

	return createMDMAppleEULAResponse{}, nil
}

func (svc *Service) MDMAppleCreateEULA(ctx context.Context, name string, file io.ReadSeeker) error {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return fleet.ErrMissingLicense
}

////////////////////////////////////////////////////////////////////////////////
// GET /mdm/apple/setup/eula?token={token}
////////////////////////////////////////////////////////////////////////////////

type getMDMAppleEULARequest struct {
	Token string `url:"token"`
}

type getMDMAppleEULAResponse struct {
	Err error `json:"error,omitempty"`

	// fields used in hijackRender to build the response
	eula *fleet.MDMAppleEULA
}

func (r getMDMAppleEULAResponse) error() error { return r.Err }

func (r getMDMAppleEULAResponse) hijackRender(ctx context.Context, w http.ResponseWriter) {
	w.Header().Set("Content-Length", strconv.Itoa(len(r.eula.Bytes)))
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	// OK to just log the error here as writing anything on
	// `http.ResponseWriter` sets the status code to 200 (and it can't be
	// changed.) Clients should rely on matching content-length with the
	// header provided
	if n, err := w.Write(r.eula.Bytes); err != nil {
		logging.WithExtras(ctx, "err", err, "bytes_copied", n)
	}
}

func getMDMAppleEULAEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*getMDMAppleEULARequest)

	eula, err := svc.MDMAppleGetEULABytes(ctx, req.Token)
	if err != nil {
		return getMDMAppleEULAResponse{Err: err}, nil
	}

	return getMDMAppleEULAResponse{eula: eula}, nil
}

func (svc *Service) MDMAppleGetEULABytes(ctx context.Context, token string) (*fleet.MDMAppleEULA, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}

////////////////////////////////////////////////////////////////////////////////
// GET /mdm/apple/setup/eula/{token}/metadata
////////////////////////////////////////////////////////////////////////////////

type getMDMAppleEULAMetadataRequest struct{}

type getMDMAppleEULAMetadataResponse struct {
	*fleet.MDMAppleEULA
	Err error `json:"error,omitempty"`
}

func (r getMDMAppleEULAMetadataResponse) error() error { return r.Err }

func getMDMAppleEULAMetadataEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	eula, err := svc.MDMAppleGetEULAMetadata(ctx)
	if err != nil {
		return getMDMAppleEULAMetadataResponse{Err: err}, nil
	}

	return getMDMAppleEULAMetadataResponse{MDMAppleEULA: eula}, nil
}

func (svc *Service) MDMAppleGetEULAMetadata(ctx context.Context) (*fleet.MDMAppleEULA, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}

////////////////////////////////////////////////////////////////////////////////
// DELETE /mdm/apple/setup/eula
////////////////////////////////////////////////////////////////////////////////

type deleteMDMAppleEULARequest struct {
	Token string `url:"token"`
}

type deleteMDMAppleEULAResponse struct {
	Err error `json:"error,omitempty"`
}

func (r deleteMDMAppleEULAResponse) error() error { return r.Err }

func deleteMDMAppleEULAEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*deleteMDMAppleEULARequest)
	if err := svc.MDMAppleDeleteEULA(ctx, req.Token); err != nil {
		return deleteMDMAppleEULAResponse{Err: err}, nil
	}
	return deleteMDMAppleEULAResponse{}, nil
}

func (svc *Service) MDMAppleDeleteEULA(ctx context.Context, token string) error {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return fleet.ErrMissingLicense
}

////////////////////////////////////////////////////////////////////////////////
// Windows MDM Middleware
////////////////////////////////////////////////////////////////////////////////

func (svc *Service) VerifyMDMWindowsConfigured(ctx context.Context) error {
	appCfg, err := svc.ds.AppConfig(ctx)
	if err != nil {
		// skipauth: Authorization is currently for user endpoints only.
		svc.authz.SkipAuthorization(ctx)
		return err
	}

	// Windows MDM configuration setting
	if !appCfg.MDM.WindowsEnabledAndConfigured {
		// skipauth: Authorization is currently for user endpoints only.
		svc.authz.SkipAuthorization(ctx)
		return fleet.ErrMDMNotConfigured
	}

	return nil
}

////////////////////////////////////////////////////////////////////////////////
// Apple or Windows MDM Middleware
////////////////////////////////////////////////////////////////////////////////

func (svc *Service) VerifyMDMAppleOrWindowsConfigured(ctx context.Context) error {
	appCfg, err := svc.ds.AppConfig(ctx)
	if err != nil {
		// skipauth: Authorization is currently for user endpoints only.
		svc.authz.SkipAuthorization(ctx)
		return err
	}

	// Apple or Windows MDM configuration setting
	if !appCfg.MDM.EnabledAndConfigured && !appCfg.MDM.WindowsEnabledAndConfigured {
		// skipauth: Authorization is currently for user endpoints only.
		svc.authz.SkipAuthorization(ctx)
		return fleet.ErrMDMNotConfigured
	}

	return nil
}

////////////////////////////////////////////////////////////////////////////////
// Run Apple or Windows MDM Command
////////////////////////////////////////////////////////////////////////////////

type runMDMCommandRequest struct {
	Command   string   `json:"command"`
	HostUUIDs []string `json:"host_uuids"`
}

type runMDMCommandResponse struct {
	*fleet.CommandEnqueueResult
	Err error `json:"error,omitempty"`
}

func (r runMDMCommandResponse) error() error { return r.Err }

func runMDMCommandEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*runMDMCommandRequest)
	result, err := svc.RunMDMCommand(ctx, req.Command, req.HostUUIDs)
	if err != nil {
		return runMDMCommandResponse{Err: err}, nil
	}
	return runMDMCommandResponse{
		CommandEnqueueResult: result,
	}, nil
}

func (svc *Service) RunMDMCommand(ctx context.Context, rawBase64Cmd string, hostUUIDs []string) (result *fleet.CommandEnqueueResult, err error) {
	hosts, err := svc.authorizeAllHostsTeams(ctx, hostUUIDs, fleet.ActionWrite, &fleet.MDMCommandAuthz{})
	if err != nil {
		return nil, err
	}
	if len(hosts) == 0 {
		err := fleet.NewInvalidArgumentError("host_uuids", "No hosts targeted. Make sure you provide a valid UUID.").WithStatus(http.StatusNotFound)
		return nil, ctxerr.Wrap(ctx, err, "no host received")
	}

	platforms := make(map[string]bool)
	for _, h := range hosts {
		if !h.MDMInfo.IsFleetEnrolled() {
			err := fleet.NewInvalidArgumentError("host_uuids", "Can't run the MDM command because one or more hosts have MDM turned off. Run the following command to see a list of hosts with MDM on: fleetctl get hosts --mdm.").WithStatus(http.StatusPreconditionFailed)
			return nil, ctxerr.Wrap(ctx, err, "check host mdm enrollment")
		}
		platforms[h.FleetPlatform()] = true
	}
	if len(platforms) != 1 {
		err := fleet.NewInvalidArgumentError("host_uuids", "All hosts must be on the same platform.")
		return nil, ctxerr.Wrap(ctx, err, "check host platform")
	}

	// it's a for loop but at this point it's guaranteed that the map has a single value.
	var commandPlatform string
	for platform := range platforms {
		commandPlatform = platform
	}
	if commandPlatform != "windows" && commandPlatform != "darwin" {
		err := fleet.NewInvalidArgumentError("host_uuids", "Invalid platform. You can only run MDM commands on Windows or macOS hosts.")
		return nil, ctxerr.Wrap(ctx, err, "check host platform")
	}

	// check that the platform-specific MDM is enabled (not sure this check can
	// ever happen, since we verify that the hosts are enrolled, but just to be
	// safe)
	switch commandPlatform {
	case "windows":
		if err := svc.VerifyMDMWindowsConfigured(ctx); err != nil {
			err := fleet.NewInvalidArgumentError("host_uuids", fleet.WindowsMDMNotConfiguredMessage).WithStatus(http.StatusBadRequest)
			return nil, ctxerr.Wrap(ctx, err, "check windows MDM enabled")
		}
	default:
		if err := svc.VerifyMDMAppleConfigured(ctx); err != nil {
			err := fleet.NewInvalidArgumentError("host_uuids", fleet.AppleMDMNotConfiguredMessage).WithStatus(http.StatusBadRequest)
			return nil, ctxerr.Wrap(ctx, err, "check macOS MDM enabled")
		}
	}

	// We're supporting both padded and unpadded base64.
	rawXMLCmd, err := server.Base64DecodePaddingAgnostic(rawBase64Cmd)
	if err != nil {
		err = fleet.NewInvalidArgumentError("command", "unable to decode base64 command").WithStatus(http.StatusBadRequest)
		return nil, ctxerr.Wrap(ctx, err, "decode base64 command")
	}

	// the rest is platform-specific (validation of command payload, enqueueing, etc.)
	switch commandPlatform {
	case "windows":
		return svc.enqueueMicrosoftMDMCommand(ctx, rawXMLCmd, hostUUIDs)
	default:
		return svc.enqueueAppleMDMCommand(ctx, rawXMLCmd, hostUUIDs)
	}
}

var appleMDMPremiumCommands = map[string]bool{
	"EraseDevice": true,
	"DeviceLock":  true,
}

func (svc *Service) enqueueAppleMDMCommand(ctx context.Context, rawXMLCmd []byte, deviceIDs []string) (result *fleet.CommandEnqueueResult, err error) {
	cmd, err := mdm.DecodeCommand(rawXMLCmd)
	if err != nil {
		err = fleet.NewInvalidArgumentError("command", "unable to decode plist command").WithStatus(http.StatusUnsupportedMediaType)
		return nil, ctxerr.Wrap(ctx, err, "decode plist command")
	}

	if appleMDMPremiumCommands[strings.TrimSpace(cmd.Command.RequestType)] {
		lic, err := svc.License(ctx)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "get license")
		}
		if !lic.IsPremium() {
			return nil, fleet.ErrMissingLicense
		}
	}

	if err := svc.mdmAppleCommander.EnqueueCommand(ctx, deviceIDs, string(rawXMLCmd)); err != nil {
		// if at least one UUID enqueued properly, return success, otherwise return
		// error
		var apnsErr *apple_mdm.APNSDeliveryError
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &apnsErr) {
			if len(apnsErr.FailedUUIDs) < len(deviceIDs) {
				// some hosts properly received the command, so return success, with the list
				// of failed uuids.
				return &fleet.CommandEnqueueResult{
					CommandUUID: cmd.CommandUUID,
					RequestType: cmd.Command.RequestType,
					FailedUUIDs: apnsErr.FailedUUIDs,
				}, nil
			}
			// push failed for all hosts
			err := fleet.NewBadGatewayError("Apple push notificiation service", err)
			return nil, ctxerr.Wrap(ctx, err, "enqueue command")

		} else if errors.As(err, &mysqlErr) {
			// enqueue may fail with a foreign key constraint error 1452 when one of
			// the hosts provided is not enrolled in nano_enrollments. Detect when
			// that's the case and add information to the error.
			if mysqlErr.Number == mysqlerr.ER_NO_REFERENCED_ROW_2 {
				err := fleet.NewInvalidArgumentError(
					"device_ids",
					fmt.Sprintf("at least one of the hosts is not enrolled in MDM or is not an elegible device: %v", err),
				).WithStatus(http.StatusBadRequest)
				return nil, ctxerr.Wrap(ctx, err, "enqueue command")
			}
		}

		return nil, ctxerr.Wrap(ctx, err, "enqueue command")
	}
	return &fleet.CommandEnqueueResult{
		CommandUUID: cmd.CommandUUID,
		RequestType: cmd.Command.RequestType,
		Platform:    "darwin",
	}, nil
}

func (svc *Service) enqueueMicrosoftMDMCommand(ctx context.Context, rawXMLCmd []byte, deviceIDs []string) (result *fleet.CommandEnqueueResult, err error) {
	cmdMsg, err := fleet.ParseWindowsMDMCommand(rawXMLCmd)
	if err != nil {
		err = fleet.NewInvalidArgumentError("command", err.Error())
		return nil, ctxerr.Wrap(ctx, err, "decode SyncML command")
	}

	if cmdMsg.IsPremium() {
		lic, err := svc.License(ctx)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "get license")
		}
		if !lic.IsPremium() {
			return nil, fleet.ErrMissingLicense
		}
	}

	winCmd := &fleet.MDMWindowsCommand{
		// TODO: using the provided ID to mimic Apple, but seems better if
		// we're full in control of it, what we should do?
		CommandUUID:  cmdMsg.CmdID,
		RawCommand:   rawXMLCmd,
		TargetLocURI: cmdMsg.GetTargetURI(),
	}
	if err := svc.ds.MDMWindowsInsertCommandForHosts(ctx, deviceIDs, winCmd); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "insert pending windows mdm command")
	}

	return &fleet.CommandEnqueueResult{
		CommandUUID: winCmd.CommandUUID,
		RequestType: winCmd.TargetLocURI,
		Platform:    "windows",
	}, nil
}

////////////////////////////////////////////////////////////////////////////////
// GET /mdm/commandresults
////////////////////////////////////////////////////////////////////////////////

type getMDMCommandResultsRequest struct {
	CommandUUID string `query:"command_uuid,optional"`
}

type getMDMCommandResultsResponse struct {
	Results []*fleet.MDMCommandResult `json:"results,omitempty"`
	Err     error                     `json:"error,omitempty"`
}

func (r getMDMCommandResultsResponse) error() error { return r.Err }

func getMDMCommandResultsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*getMDMCommandResultsRequest)
	results, err := svc.GetMDMCommandResults(ctx, req.CommandUUID)
	if err != nil {
		return getMDMCommandResultsResponse{
			Err: err,
		}, nil
	}

	return getMDMCommandResultsResponse{
		Results: results,
	}, nil
}

func (svc *Service) GetMDMCommandResults(ctx context.Context, commandUUID string) ([]*fleet.MDMCommandResult, error) {
	// first, authorize that the user has the right to list hosts
	if err := svc.authz.Authorize(ctx, &fleet.Host{}, fleet.ActionList); err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}

	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, fleet.ErrNoContext
	}

	// check that command exists first, to return 404 on invalid commands
	// (the command may exist but have no results yet).
	p, err := svc.ds.GetMDMCommandPlatform(ctx, commandUUID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}

	var results []*fleet.MDMCommandResult
	switch p {
	case "darwin":
		results, err = svc.ds.GetMDMAppleCommandResults(ctx, commandUUID)
	case "windows":
		results, err = svc.ds.GetMDMWindowsCommandResults(ctx, commandUUID)
	default:
		// this should never happen, but just in case
		level.Debug(svc.logger).Log("msg", "unknown MDM command platform", "platform", p)
	}

	if err != nil {
		return nil, err
	}

	// now we can load the hosts (lite) corresponding to those command results,
	// and do the final authorization check with the proper team(s). Include observers,
	// as they are able to view command results for their teams' hosts.
	filter := fleet.TeamFilter{User: vc.User, IncludeObserver: true}
	hostUUIDs := make([]string, len(results))
	for i, res := range results {
		hostUUIDs[i] = res.HostUUID
	}
	hosts, err := svc.ds.ListHostsLiteByUUIDs(ctx, filter, hostUUIDs)
	if err != nil {
		return nil, err
	}
	if len(hosts) == 0 {
		// do not return 404 here, as it's possible for a command to not have
		// results yet
		return nil, nil
	}

	// collect the team IDs and verify that the user has access to view commands
	// on all affected teams. Index the hosts by uuid for easly lookup as
	// afterwards we'll want to store the hostname on the returned results.
	hostsByUUID := make(map[string]*fleet.Host, len(hosts))
	teamIDs := make(map[uint]bool)
	for _, h := range hosts {
		var id uint
		if h.TeamID != nil {
			id = *h.TeamID
		}
		teamIDs[id] = true
		hostsByUUID[h.UUID] = h
	}

	var commandAuthz fleet.MDMCommandAuthz
	for tmID := range teamIDs {
		commandAuthz.TeamID = &tmID
		if tmID == 0 {
			commandAuthz.TeamID = nil
		}

		if err := svc.authz.Authorize(ctx, commandAuthz, fleet.ActionRead); err != nil {
			return nil, ctxerr.Wrap(ctx, err)
		}
	}

	// add the hostnames to the results
	for _, res := range results {
		if h := hostsByUUID[res.HostUUID]; h != nil {
			res.Hostname = hostsByUUID[res.HostUUID].Hostname
		}
	}
	return results, nil
}

////////////////////////////////////////////////////////////////////////////////
// GET /mdm/commands
////////////////////////////////////////////////////////////////////////////////

type listMDMCommandsRequest struct {
	ListOptions fleet.ListOptions `url:"list_options"`
}

type listMDMCommandsResponse struct {
	Results []*fleet.MDMCommand `json:"results"`
	Err     error               `json:"error,omitempty"`
}

func (r listMDMCommandsResponse) error() error { return r.Err }

func listMDMCommandsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*listMDMCommandsRequest)
	results, err := svc.ListMDMCommands(ctx, &fleet.MDMCommandListOptions{
		ListOptions: req.ListOptions,
	})
	if err != nil {
		return listMDMCommandsResponse{
			Err: err,
		}, nil
	}

	return listMDMCommandsResponse{
		Results: results,
	}, nil
}

func (svc *Service) ListMDMCommands(ctx context.Context, opts *fleet.MDMCommandListOptions) ([]*fleet.MDMCommand, error) {
	// first, authorize that the user has the right to list hosts
	if err := svc.authz.Authorize(ctx, &fleet.Host{}, fleet.ActionList); err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}

	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, fleet.ErrNoContext
	}

	// get the list of commands so we know what hosts (and therefore what teams)
	// we're dealing with. Including the observers as they are allowed to view
	// MDM Apple commands.
	results, err := svc.ds.ListMDMCommands(ctx, fleet.TeamFilter{
		User:            vc.User,
		IncludeObserver: true,
	}, opts)
	if err != nil {
		return nil, err
	}

	// collect the different team IDs and verify that the user has access to view
	// commands on all affected teams, do not assume that ListMDMCommands
	// only returned hosts that the user is authorized to view the command
	// results of (that is, always verify with our rego authz policy).
	teamIDs := make(map[uint]bool)
	for _, res := range results {
		var id uint
		if res.TeamID != nil {
			id = *res.TeamID
		}
		teamIDs[id] = true
	}

	// instead of returning an authz error if the user is not authorized for a
	// team, we remove those commands from the results (as we want to return
	// whatever the user is allowed to see). Since this can only be done after
	// retrieving the list of commands, this may result in returning less results
	// than requested, but it's ok - it's expected that the results retrieved
	// from the datastore will all be authorized for the user.
	var commandAuthz fleet.MDMCommandAuthz
	var authzErr error
	for tmID := range teamIDs {
		commandAuthz.TeamID = &tmID
		if tmID == 0 {
			commandAuthz.TeamID = nil
		}
		if err := svc.authz.Authorize(ctx, commandAuthz, fleet.ActionRead); err != nil {
			if authzErr == nil {
				authzErr = err
			}
			teamIDs[tmID] = false
		}
	}

	if authzErr != nil {
		level.Error(svc.logger).Log("err", "unauthorized to view some team commands", "details", authzErr)

		// filter-out the teams that the user is not allowed to view
		allowedResults := make([]*fleet.MDMCommand, 0, len(results))
		for _, res := range results {
			var id uint
			if res.TeamID != nil {
				id = *res.TeamID
			}
			if teamIDs[id] {
				allowedResults = append(allowedResults, res)
			}
		}
		results = allowedResults
	}

	return results, nil
}

////////////////////////////////////////////////////////////////////////////////
// GET /mdm/disk_encryption/summary
////////////////////////////////////////////////////////////////////////////////

type getMDMDiskEncryptionSummaryRequest struct {
	TeamID *uint `query:"team_id,optional"`
}

type getMDMDiskEncryptionSummaryResponse struct {
	*fleet.MDMDiskEncryptionSummary
	Err error `json:"error,omitempty"`
}

func (r getMDMDiskEncryptionSummaryResponse) error() error { return r.Err }

func getMDMDiskEncryptionSummaryEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*getMDMDiskEncryptionSummaryRequest)

	des, err := svc.GetMDMDiskEncryptionSummary(ctx, req.TeamID)
	if err != nil {
		return getMDMDiskEncryptionSummaryResponse{Err: err}, nil
	}

	return &getMDMDiskEncryptionSummaryResponse{
		MDMDiskEncryptionSummary: des,
	}, nil
}

func (svc *Service) GetMDMDiskEncryptionSummary(ctx context.Context, teamID *uint) (*fleet.MDMDiskEncryptionSummary, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}

// authorizeAllHostsTeams is a helper function that loads the hosts
// corresponding to the hostUUIDs and authorizes the context user to execute
// the specified authzAction (e.g. fleet.ActionWrite) for all the hosts' teams
// with the specified authorizer, which is typically a struct that can set a
// TeamID field and defines an authorization subject, such as
// fleet.MDMCommandAuthz.
//
// On success, the list of hosts is returned (which may be empty, it is up to
// the caller to return an error if needed when no hosts are found).
func (svc *Service) authorizeAllHostsTeams(ctx context.Context, hostUUIDs []string, authzAction any, authorizer fleet.TeamIDSetter) ([]*fleet.Host, error) {
	// load hosts (lite) by uuids, check that the user has the rights to run
	// commands for every affected team.
	if err := svc.authz.Authorize(ctx, &fleet.Host{}, fleet.ActionList); err != nil {
		return nil, err
	}

	// here we use a global admin as filter because we want to get all hosts that
	// correspond to those uuids. Only after we get those hosts will we check
	// authorization for the current user, for all teams affected by that host.
	// Without this, only hosts that the user can view would be returned and the
	// actual authorization check might only be done on a subset of the requsted
	// hosts.
	filter := fleet.TeamFilter{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}}
	hosts, err := svc.ds.ListHostsLiteByUUIDs(ctx, filter, hostUUIDs)
	if err != nil {
		return nil, err
	}

	// collect the team IDs and verify that the user has access to run commands
	// on all affected teams.
	teamIDs := make(map[uint]bool, len(hosts))
	for _, h := range hosts {
		var id uint
		if h.TeamID != nil {
			id = *h.TeamID
		}
		teamIDs[id] = true
	}

	for tmID := range teamIDs {
		authzTeamID := &tmID
		if tmID == 0 {
			authzTeamID = nil
		}
		authorizer.SetTeamID(authzTeamID)

		if err := svc.authz.Authorize(ctx, authorizer, authzAction); err != nil {
			return nil, ctxerr.Wrap(ctx, err)
		}
	}
	return hosts, nil
}

////////////////////////////////////////////////////////////////////////////////
// DELETE /mdm/profiles/{id_or_uuid}
////////////////////////////////////////////////////////////////////////////////

type deleteMDMConfigProfileRequest struct {
	ProfileIDOrUUID string `url:"profile_id_or_uuid"`
}

type deleteMDMConfigProfileResponse struct {
	Err error `json:"error,omitempty"`
}

func (r deleteMDMConfigProfileResponse) error() error { return r.Err }

func deleteMDMConfigProfileEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*deleteMDMConfigProfileRequest)

	appleID, isApple := isAppleProfileID(req.ProfileIDOrUUID)
	var err error
	if isApple {
		err = svc.DeleteMDMAppleConfigProfile(ctx, appleID)
	} else {
		err = svc.DeleteMDMWindowsConfigProfile(ctx, req.ProfileIDOrUUID)
	}
	return &deleteMDMConfigProfileResponse{Err: err}, nil
}

func (svc *Service) DeleteMDMWindowsConfigProfile(ctx context.Context, profileUUID string) error {
	// first we perform a perform basic authz check
	if err := svc.authz.Authorize(ctx, &fleet.Team{}, fleet.ActionRead); err != nil {
		return ctxerr.Wrap(ctx, err)
	}

	// check that Windows MDM is enabled - the middleware of that endpoint checks
	// only that any MDM is enabled, maybe it's just macOS
	if err := svc.VerifyMDMWindowsConfigured(ctx); err != nil {
		err := fleet.NewInvalidArgumentError("profile_id", fleet.WindowsMDMNotConfiguredMessage).WithStatus(http.StatusBadRequest)
		return ctxerr.Wrap(ctx, err, "check windows MDM enabled")
	}

	prof, err := svc.ds.GetMDMWindowsConfigProfile(ctx, profileUUID)
	if err != nil {
		return ctxerr.Wrap(ctx, err)
	}

	var teamName string
	teamID := *prof.TeamID
	if teamID >= 1 {
		tm, err := svc.EnterpriseOverrides.TeamByIDOrName(ctx, &teamID, nil)
		if err != nil {
			return ctxerr.Wrap(ctx, err)
		}
		teamName = tm.Name
	}

	// now we can do a specific authz check based on team id of profile before we delete the profile
	if err := svc.authz.Authorize(ctx, &fleet.MDMConfigProfileAuthz{TeamID: prof.TeamID}, fleet.ActionWrite); err != nil {
		return ctxerr.Wrap(ctx, err)
	}

	// TODO: do we have Fleet-specific profiles for Windows that we'd want to prevent the user from deleting?

	if err := svc.ds.DeleteMDMWindowsConfigProfile(ctx, profileUUID); err != nil {
		return ctxerr.Wrap(ctx, err)
	}

	// TODO: integrate the call to bulk-update host profiles affected by this deletion (see Apple's implementation)
	// (part of https://github.com/fleetdm/fleet/issues/14364)

	var (
		actTeamID   *uint
		actTeamName *string
	)
	if teamID > 0 {
		actTeamID = &teamID
		actTeamName = &teamName
	}
	if err := svc.ds.NewActivity(ctx, authz.UserFromContext(ctx), &fleet.ActivityTypeDeletedWindowsProfile{
		TeamID:      actTeamID,
		TeamName:    actTeamName,
		ProfileName: prof.Name,
	}); err != nil {
		return ctxerr.Wrap(ctx, err, "logging activity for delete mdm windows config profile")
	}

	return nil
}

// returns the numeric Apple profile ID and true if it is an Apple identifier,
// or 0 and false otherwise.
func isAppleProfileID(profileIDOrUUID string) (uint, bool) {
	// parsing as 32 bits as that's the maximum value of the DB column (and can
	// be safely converted to uint).
	id, err := strconv.ParseUint(profileIDOrUUID, 10, 32)
	if err != nil {
		return 0, false
	}
	return uint(id), true
}
