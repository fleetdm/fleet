package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/VividCortex/mysqlerr"
	"github.com/docker/go-units"
	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	nanomdm "github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/go-kit/kit/log/level"
	"github.com/go-sql-driver/mysql"
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
// POST /mdm/setup/eula
////////////////////////////////////////////////////////////////////////////////

type createMDMEULARequest struct {
	EULA *multipart.FileHeader
}

// TODO: We parse the whole body before running svc.authz.Authorize.
// An authenticated but unauthorized user could abuse this.
func (createMDMEULARequest) DecodeRequest(ctx context.Context, r *http.Request) (interface{}, error) {
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

	return &createMDMEULARequest{
		EULA: r.MultipartForm.File["eula"][0],
	}, nil
}

type createMDMEULAResponse struct {
	Err error `json:"error,omitempty"`
}

func (r createMDMEULAResponse) error() error { return r.Err }

func createMDMEULAEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*createMDMEULARequest)
	ff, err := req.EULA.Open()
	if err != nil {
		return createMDMEULAResponse{Err: err}, nil
	}
	defer ff.Close()

	if err := svc.MDMCreateEULA(ctx, req.EULA.Filename, ff); err != nil {
		return createMDMEULAResponse{Err: err}, nil
	}

	return createMDMEULAResponse{}, nil
}

func (svc *Service) MDMCreateEULA(ctx context.Context, name string, file io.ReadSeeker) error {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return fleet.ErrMissingLicense
}

////////////////////////////////////////////////////////////////////////////////
// GET /mdm/setup/eula?token={token}
////////////////////////////////////////////////////////////////////////////////

type getMDMEULARequest struct {
	Token string `url:"token"`
}

type getMDMEULAResponse struct {
	Err error `json:"error,omitempty"`

	// fields used in hijackRender to build the response
	eula *fleet.MDMEULA
}

func (r getMDMEULAResponse) error() error { return r.Err }

func (r getMDMEULAResponse) hijackRender(ctx context.Context, w http.ResponseWriter) {
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

func getMDMEULAEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*getMDMEULARequest)

	eula, err := svc.MDMGetEULABytes(ctx, req.Token)
	if err != nil {
		return getMDMEULAResponse{Err: err}, nil
	}

	return getMDMEULAResponse{eula: eula}, nil
}

func (svc *Service) MDMGetEULABytes(ctx context.Context, token string) (*fleet.MDMEULA, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}

////////////////////////////////////////////////////////////////////////////////
// GET /mdm/setup/eula/{token}/metadata
////////////////////////////////////////////////////////////////////////////////

type getMDMEULAMetadataRequest struct{}

type getMDMEULAMetadataResponse struct {
	*fleet.MDMEULA
	Err error `json:"error,omitempty"`
}

func (r getMDMEULAMetadataResponse) error() error { return r.Err }

func getMDMEULAMetadataEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	eula, err := svc.MDMGetEULAMetadata(ctx)
	if err != nil {
		return getMDMEULAMetadataResponse{Err: err}, nil
	}

	return getMDMEULAMetadataResponse{MDMEULA: eula}, nil
}

func (svc *Service) MDMGetEULAMetadata(ctx context.Context) (*fleet.MDMEULA, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}

////////////////////////////////////////////////////////////////////////////////
// DELETE /mdm/setup/eula
////////////////////////////////////////////////////////////////////////////////

type deleteMDMEULARequest struct {
	Token string `url:"token"`
}

type deleteMDMEULAResponse struct {
	Err error `json:"error,omitempty"`
}

func (r deleteMDMEULAResponse) error() error { return r.Err }

func deleteMDMEULAEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*deleteMDMEULARequest)
	if err := svc.MDMDeleteEULA(ctx, req.Token); err != nil {
		return deleteMDMEULAResponse{Err: err}, nil
	}
	return deleteMDMEULAResponse{}, nil
}

func (svc *Service) MDMDeleteEULA(ctx context.Context, token string) error {
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
	cmd, err := nanomdm.DecodeCommand(rawXMLCmd)
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
		CommandUUID:  cmdMsg.CmdID.Value,
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

////////////////////////////////////////////////////////////////////////////////
// GET /mdm/profiles/summary
////////////////////////////////////////////////////////////////////////////////

type getMDMProfilesSummaryRequest struct {
	TeamID *uint `query:"team_id,optional"`
}

type getMDMProfilesSummaryResponse struct {
	fleet.MDMProfilesSummary
	Err error `json:"error,omitempty"`
}

func (r getMDMProfilesSummaryResponse) error() error { return r.Err }

func getMDMProfilesSummaryEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*getMDMProfilesSummaryRequest)
	res := getMDMProfilesSummaryResponse{}

	as, err := svc.GetMDMAppleProfilesSummary(ctx, req.TeamID)
	if err != nil {
		return &getMDMAppleProfilesSummaryResponse{Err: err}, nil
	}

	ws, err := svc.GetMDMWindowsProfilesSummary(ctx, req.TeamID)
	if err != nil {
		return &getMDMProfilesSummaryResponse{Err: err}, nil
	}

	res.Verified = as.Verified + ws.Verified
	res.Verifying = as.Verifying + ws.Verifying
	res.Failed = as.Failed + ws.Failed
	res.Pending = as.Pending + ws.Pending

	return &res, nil
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
// GET /mdm/profiles/{uuid}
////////////////////////////////////////////////////////////////////////////////

type getMDMConfigProfileRequest struct {
	ProfileUUID string `url:"profile_uuid"`
	Alt         string `query:"alt,optional"`
}

type getMDMConfigProfileResponse struct {
	*fleet.MDMConfigProfilePayload
	Err error `json:"error,omitempty"`
}

func (r getMDMConfigProfileResponse) error() error { return r.Err }

func getMDMConfigProfileEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*getMDMConfigProfileRequest)

	downloadRequested := req.Alt == "media"
	var err error
	if isAppleProfileUUID(req.ProfileUUID) {
		// Apple config profile
		cp, err := svc.GetMDMAppleConfigProfile(ctx, req.ProfileUUID)
		if err != nil {
			return &getMDMConfigProfileResponse{Err: err}, nil
		}

		if downloadRequested {
			return downloadFileResponse{
				content:     cp.Mobileconfig,
				contentType: "application/x-apple-aspen-config",
				filename:    fmt.Sprintf("%s_%s.mobileconfig", time.Now().Format("2006-01-02"), strings.ReplaceAll(cp.Name, " ", "_")),
			}, nil
		}
		return &getMDMConfigProfileResponse{
			MDMConfigProfilePayload: fleet.NewMDMConfigProfilePayloadFromApple(cp),
		}, nil
	}

	if isAppleDeclarationUUID(req.ProfileUUID) {
		// TODO: we could potentially combined with the other service methods
		decl, err := svc.GetMDMAppleDeclaration(ctx, req.ProfileUUID)
		if err != nil {
			return &getMDMConfigProfileResponse{Err: err}, nil
		}

		if downloadRequested {
			return downloadFileResponse{
				content:     decl.RawJSON,
				contentType: "application/json",
				filename:    fmt.Sprintf("%s_%s.json", time.Now().Format("2006-01-02"), strings.ReplaceAll(decl.Name, " ", "_")),
			}, nil
		}
		return &getMDMConfigProfileResponse{
			MDMConfigProfilePayload: fleet.NewMDMConfigProfilePayloadFromAppleDDM(decl),
		}, nil
	}

	// Windows config profile
	cp, err := svc.GetMDMWindowsConfigProfile(ctx, req.ProfileUUID)
	if err != nil {
		return &getMDMConfigProfileResponse{Err: err}, nil
	}

	if downloadRequested {
		return downloadFileResponse{
			content:     cp.SyncML,
			contentType: "application/octet-stream", // not using the XML MIME type as a profile is not valid XML (a list of <Replace> elements)
			filename:    fmt.Sprintf("%s_%s.xml", time.Now().Format("2006-01-02"), strings.ReplaceAll(cp.Name, " ", "_")),
		}, nil
	}
	return &getMDMConfigProfileResponse{
		MDMConfigProfilePayload: fleet.NewMDMConfigProfilePayloadFromWindows(cp),
	}, nil
}

func (svc *Service) GetMDMWindowsConfigProfile(ctx context.Context, profileUUID string) (*fleet.MDMWindowsConfigProfile, error) {
	// first we perform a perform basic authz check
	if err := svc.authz.Authorize(ctx, &fleet.Team{}, fleet.ActionRead); err != nil {
		return nil, err
	}

	cp, err := svc.ds.GetMDMWindowsConfigProfile(ctx, profileUUID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}

	// now we can do a specific authz check based on team id of profile before we
	// return the profile.
	if err := svc.authz.Authorize(ctx, &fleet.MDMConfigProfileAuthz{TeamID: cp.TeamID}, fleet.ActionRead); err != nil {
		return nil, err
	}

	return cp, nil
}

////////////////////////////////////////////////////////////////////////////////
// DELETE /mdm/profiles/{uuid}
////////////////////////////////////////////////////////////////////////////////

type deleteMDMConfigProfileRequest struct {
	ProfileUUID string `url:"profile_uuid"`
}

type deleteMDMConfigProfileResponse struct {
	Err error `json:"error,omitempty"`
}

func (r deleteMDMConfigProfileResponse) error() error { return r.Err }

func deleteMDMConfigProfileEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*deleteMDMConfigProfileRequest)

	var err error
	if isAppleProfileUUID(req.ProfileUUID) {
		err = svc.DeleteMDMAppleConfigProfile(ctx, req.ProfileUUID)
	} else if isAppleDeclarationUUID(req.ProfileUUID) {
		// TODO: we could potentially combined with the other service methods
		err = svc.DeleteMDMAppleDeclaration(ctx, req.ProfileUUID)
	} else {
		err = svc.DeleteMDMWindowsConfigProfile(ctx, req.ProfileUUID)
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
		err := fleet.NewInvalidArgumentError("profile_uuid", fleet.WindowsMDMNotConfiguredMessage).WithStatus(http.StatusBadRequest)
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

	// prevent deleting Fleet-managed profiles (e.g., Windows OS Updates profile controlled by the OS Updates settings)
	fleetNames := mdm.FleetReservedProfileNames()
	if _, ok := fleetNames[prof.Name]; ok {
		err := &fleet.BadRequestError{Message: "Profiles managed by Fleet can't be deleted using this endpoint."}
		return ctxerr.Wrap(ctx, err, "validate profile")
	}

	if err := svc.ds.DeleteMDMWindowsConfigProfile(ctx, profileUUID); err != nil {
		return ctxerr.Wrap(ctx, err)
	}

	// cannot use the profile ID as it is now deleted
	if err := svc.ds.BulkSetPendingMDMHostProfiles(ctx, nil, []uint{teamID}, nil, nil); err != nil {
		return ctxerr.Wrap(ctx, err, "bulk set pending host profiles")
	}

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
func isAppleProfileUUID(profileUUID string) bool {
	return strings.HasPrefix(profileUUID, "a")
}

func isAppleDeclarationUUID(profileUUID string) bool {
	return strings.HasPrefix(profileUUID, fleet.MDMAppleDeclarationUUIDPrefix)
}

////////////////////////////////////////////////////////////////////////////////
// POST /mdm/profiles (Create Apple or Windows MDM Config Profile)
////////////////////////////////////////////////////////////////////////////////

type newMDMConfigProfileRequest struct {
	TeamID  uint
	Profile *multipart.FileHeader
	Labels  []string
}

func (newMDMConfigProfileRequest) DecodeRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	decoded := newMDMConfigProfileRequest{}

	err := r.ParseMultipartForm(512 * units.MiB)
	if err != nil {
		return nil, &fleet.BadRequestError{
			Message:     "failed to parse multipart form",
			InternalErr: err,
		}
	}

	// add team_id
	val, ok := r.MultipartForm.Value["team_id"]
	if !ok || len(val) < 1 {
		// default is no team
		decoded.TeamID = 0
	} else {
		teamID, err := strconv.Atoi(val[0])
		if err != nil {
			return nil, &fleet.BadRequestError{Message: fmt.Sprintf("failed to decode team_id in multipart form: %s", err.Error())}
		}
		decoded.TeamID = uint(teamID)
	}

	// add profile
	fhs, ok := r.MultipartForm.File["profile"]
	if !ok || len(fhs) < 1 {
		return nil, &fleet.BadRequestError{Message: "no file headers for profile"}
	}
	decoded.Profile = fhs[0]

	// add labels
	decoded.Labels = r.MultipartForm.Value["labels"]

	return &decoded, nil
}

type newMDMConfigProfileResponse struct {
	ProfileUUID string `json:"profile_uuid"`
	Err         error  `json:"error,omitempty"`
}

func (r newMDMConfigProfileResponse) error() error { return r.Err }

func newMDMConfigProfileEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*newMDMConfigProfileRequest)

	ff, err := req.Profile.Open()
	if err != nil {
		return &newMDMConfigProfileResponse{Err: err}, nil
	}
	defer ff.Close()

	fileExt := filepath.Ext(req.Profile.Filename)
	profileName := strings.TrimSuffix(filepath.Base(req.Profile.Filename), fileExt)
	isMobileConfig := strings.EqualFold(fileExt, ".mobileconfig")
	isJSON := strings.EqualFold(fileExt, ".json")
	if isMobileConfig || isJSON {
		// Then it's an Apple configuration file
		if isJSON {
			decl, err := svc.NewMDMAppleDeclaration(ctx, req.TeamID, ff, req.Labels, profileName)
			if err != nil {
				return &newMDMConfigProfileResponse{Err: err}, nil
			}

			return &newMDMConfigProfileResponse{
				ProfileUUID: decl.DeclarationUUID,
			}, nil

		}

		cp, err := svc.NewMDMAppleConfigProfile(ctx, req.TeamID, ff, req.Labels)
		if err != nil {
			return &newMDMConfigProfileResponse{Err: err}, nil
		}
		return &newMDMConfigProfileResponse{
			ProfileUUID: cp.ProfileUUID,
		}, nil
	}

	if isWindows := strings.EqualFold(fileExt, ".xml"); isWindows {
		cp, err := svc.NewMDMWindowsConfigProfile(ctx, req.TeamID, profileName, ff, req.Labels)
		if err != nil {
			return &newMDMConfigProfileResponse{Err: err}, nil
		}
		return &newMDMConfigProfileResponse{
			ProfileUUID: cp.ProfileUUID,
		}, nil
	}

	err = svc.NewMDMUnsupportedConfigProfile(ctx, req.TeamID, req.Profile.Filename)
	return &newMDMConfigProfileResponse{Err: err}, nil
}

func (svc *Service) NewMDMUnsupportedConfigProfile(ctx context.Context, teamID uint, filename string) error {
	if err := svc.authz.Authorize(ctx, &fleet.MDMConfigProfileAuthz{TeamID: &teamID}, fleet.ActionWrite); err != nil {
		return ctxerr.Wrap(ctx, err)
	}

	// this is required because we need authorize to return the error, and
	// svc.authz is only available on the concrete Service struct, not on the
	// Service interface so it cannot be done in the endpoint itself.
	return &fleet.BadRequestError{Message: "Couldn't add profile. The file should be a .mobileconfig, XML, or JSON file."}
}

func (svc *Service) NewMDMWindowsConfigProfile(ctx context.Context, teamID uint, profileName string, r io.Reader, labels []string) (*fleet.MDMWindowsConfigProfile, error) {
	if err := svc.authz.Authorize(ctx, &fleet.MDMConfigProfileAuthz{TeamID: &teamID}, fleet.ActionWrite); err != nil {
		return nil, ctxerr.Wrap(ctx, err)
	}

	// check that Windows MDM is enabled - the middleware of that endpoint checks
	// only that any MDM is enabled, maybe it's just macOS
	if err := svc.VerifyMDMWindowsConfigured(ctx); err != nil {
		err := fleet.NewInvalidArgumentError("profile", fleet.WindowsMDMNotConfiguredMessage).WithStatus(http.StatusBadRequest)
		return nil, ctxerr.Wrap(ctx, err, "check windows MDM enabled")
	}

	var teamName string
	if teamID > 0 {
		tm, err := svc.EnterpriseOverrides.TeamByIDOrName(ctx, &teamID, nil)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err)
		}
		teamName = tm.Name
	}

	b, err := io.ReadAll(r)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, &fleet.BadRequestError{
			Message:     "failed to read Windows config profile",
			InternalErr: err,
		})
	}

	cp := fleet.MDMWindowsConfigProfile{
		TeamID: &teamID,
		Name:   profileName,
		SyncML: b,
	}
	if err := cp.ValidateUserProvided(); err != nil {
		// this is not great, but since the validations are shared between the CLI
		// and the API, we must make some changes to error message here.
		msg := err.Error()
		if ix := strings.Index(msg, "To control these settings,"); ix >= 0 {
			msg = strings.TrimSpace(msg[:ix])
		}
		err := &fleet.BadRequestError{Message: "Couldn't upload. " + msg}
		return nil, ctxerr.Wrap(ctx, err, "validate profile")
	}

	labelMap, err := svc.validateProfileLabels(ctx, labels)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "validating labels")
	}
	cp.Labels = labelMap

	newCP, err := svc.ds.NewMDMWindowsConfigProfile(ctx, cp)
	if err != nil {
		var existsErr existsErrorInterface
		if errors.As(err, &existsErr) {
			err = fleet.NewInvalidArgumentError("profile", "Couldn't upload. A configuration profile with this name already exists.").
				WithStatus(http.StatusConflict)
		}
		return nil, ctxerr.Wrap(ctx, err)
	}

	if err := svc.ds.BulkSetPendingMDMHostProfiles(ctx, nil, nil, []string{newCP.ProfileUUID}, nil); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "bulk set pending host profiles")
	}

	var (
		actTeamID   *uint
		actTeamName *string
	)
	if teamID > 0 {
		actTeamID = &teamID
		actTeamName = &teamName
	}
	if err := svc.ds.NewActivity(ctx, authz.UserFromContext(ctx), &fleet.ActivityTypeCreatedWindowsProfile{
		TeamID:      actTeamID,
		TeamName:    actTeamName,
		ProfileName: newCP.Name,
	}); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "logging activity for create mdm windows config profile")
	}

	return newCP, nil
}

func (svc *Service) batchValidateProfileLabels(ctx context.Context, labelNames []string) (map[string]fleet.ConfigurationProfileLabel, error) {
	if len(labelNames) == 0 {
		return nil, nil
	}

	labels, err := svc.ds.LabelIDsByName(ctx, labelNames)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting label IDs by name")
	}

	uniqueNames := make(map[string]bool)
	for _, entry := range labelNames {
		if _, value := uniqueNames[entry]; !value {
			uniqueNames[entry] = true
		}
	}

	if len(labels) != len(uniqueNames) {
		return nil, &fleet.BadRequestError{
			Message:     "some or all the labels provided don't exist",
			InternalErr: fmt.Errorf("names provided: %v", labelNames),
		}
	}

	profLabels := make(map[string]fleet.ConfigurationProfileLabel)
	for labelName, labelID := range labels {
		profLabels[labelName] = fleet.ConfigurationProfileLabel{
			LabelName: labelName,
			LabelID:   labelID,
		}
	}
	return profLabels, nil
}

func (svc *Service) validateProfileLabels(ctx context.Context, labelNames []string) ([]fleet.ConfigurationProfileLabel, error) {
	labelMap, err := svc.batchValidateProfileLabels(ctx, labelNames)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "validating profile labels")
	}

	var profLabels []fleet.ConfigurationProfileLabel
	for _, label := range labelMap {
		profLabels = append(profLabels, label)
	}
	return profLabels, nil
}

////////////////////////////////////////////////////////////////////////////////
// Batch Replace MDM Profiles
////////////////////////////////////////////////////////////////////////////////

type batchSetMDMProfilesRequest struct {
	TeamID        *uint                        `json:"-" query:"team_id,optional"`
	TeamName      *string                      `json:"-" query:"team_name,optional"`
	DryRun        bool                         `json:"-" query:"dry_run,optional"`        // if true, apply validation but do not save changes
	AssumeEnabled *bool                        `json:"-" query:"assume_enabled,optional"` // if true, assume MDM is enabled
	Profiles      backwardsCompatProfilesParam `json:"profiles"`
}

type backwardsCompatProfilesParam []fleet.MDMProfileBatchPayload

func (bcp *backwardsCompatProfilesParam) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return nil
	}

	if lookAhead := bytes.TrimSpace(data); len(lookAhead) > 0 && lookAhead[0] == '[' {
		// use []fleet.MDMProfileBatchPayload to prevent infinite recursion if we
		// use `backwardsCompatProfileSlice`
		var profs []fleet.MDMProfileBatchPayload
		if err := json.Unmarshal(data, &profs); err != nil {
			return fmt.Errorf("unmarshal profile spec. Error using new format: %w", err)
		}
		*bcp = profs
		return nil
	}

	var backwardsCompat map[string][]byte
	if err := json.Unmarshal(data, &backwardsCompat); err != nil {
		return fmt.Errorf("unmarshal profile spec. Error using old format: %w", err)
	}

	*bcp = make(backwardsCompatProfilesParam, 0, len(backwardsCompat))
	for name, contents := range backwardsCompat {
		*bcp = append(*bcp, fleet.MDMProfileBatchPayload{Name: name, Contents: contents})
	}
	return nil
}

type batchSetMDMProfilesResponse struct {
	Err error `json:"error,omitempty"`
}

func (r batchSetMDMProfilesResponse) error() error { return r.Err }

func (r batchSetMDMProfilesResponse) Status() int { return http.StatusNoContent }

func batchSetMDMProfilesEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*batchSetMDMProfilesRequest)
	if err := svc.BatchSetMDMProfiles(
		ctx, req.TeamID, req.TeamName, req.Profiles, req.DryRun, false, req.AssumeEnabled,
	); err != nil {
		return batchSetMDMProfilesResponse{Err: err}, nil
	}
	return batchSetMDMProfilesResponse{}, nil
}

func (svc *Service) BatchSetMDMProfiles(
	ctx context.Context, tmID *uint, tmName *string, profiles []fleet.MDMProfileBatchPayload, dryRun, skipBulkPending bool,
	assumeEnabled *bool,
) error {
	var err error
	if tmID, tmName, err = svc.authorizeBatchProfiles(ctx, tmID, tmName); err != nil {
		return err
	}

	appCfg, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting app config")
	}
	if assumeEnabled != nil {
		appCfg.MDM.WindowsEnabledAndConfigured = *assumeEnabled
	}

	if err := validateProfiles(profiles); err != nil {
		return ctxerr.Wrap(ctx, err, "validating profiles")
	}

	labels := []string{}
	for _, prof := range profiles {
		labels = append(labels, prof.Labels...)
	}
	labelMap, err := svc.batchValidateProfileLabels(ctx, labels)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "validating labels")
	}

	appleProfiles, appleDecls, err := getAppleProfiles(ctx, tmID, appCfg, profiles, labelMap)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "validating macOS profiles")
	}

	windowsProfiles, err := getWindowsProfiles(ctx, tmID, appCfg, profiles, labelMap)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "validating Windows profiles")
	}

	if err := svc.validateCrossPlatformProfileNames(ctx, appleProfiles, windowsProfiles, appleDecls); err != nil {
		return ctxerr.Wrap(ctx, err, "validating cross-platform profile names")
	}

	if dryRun {
		return nil
	}

	if err := svc.ds.BatchSetMDMProfiles(ctx, tmID, appleProfiles, windowsProfiles, appleDecls); err != nil {
		return ctxerr.Wrap(ctx, err, "setting config profiles")
	}

	// set pending status for windows profiles
	winProfUUIDs := []string{}
	for _, p := range windowsProfiles {
		winProfUUIDs = append(winProfUUIDs, p.ProfileUUID)
	}
	if err := svc.ds.BulkSetPendingMDMHostProfiles(ctx, nil, nil, winProfUUIDs, nil); err != nil {
		return ctxerr.Wrap(ctx, err, "bulk set pending windows host profiles")
	}

	// set pending status for apple profiles
	appleProfUUIDs := []string{}
	for _, p := range appleProfiles {
		appleProfUUIDs = append(appleProfUUIDs, p.ProfileUUID)
	}
	if err := svc.ds.BulkSetPendingMDMHostProfiles(ctx, nil, nil, appleProfUUIDs, nil); err != nil {
		return ctxerr.Wrap(ctx, err, "bulk set pending apple host profiles")
	}

	// TODO(roberto): should we generate activities only of any profiles were
	// changed? this is the existing behavior for macOS profiles so I'm
	// leaving it as-is for now.
	if err := svc.ds.NewActivity(ctx, authz.UserFromContext(ctx), &fleet.ActivityTypeEditedMacosProfile{
		TeamID:   tmID,
		TeamName: tmName,
	}); err != nil {
		return ctxerr.Wrap(ctx, err, "logging activity for edited macos profile")
	}
	if err := svc.ds.NewActivity(ctx, authz.UserFromContext(ctx), &fleet.ActivityTypeEditedWindowsProfile{
		TeamID:   tmID,
		TeamName: tmName,
	}); err != nil {
		return ctxerr.Wrap(ctx, err, "logging activity for edited windows profile")
	}
	if err := svc.ds.NewActivity(ctx, authz.UserFromContext(ctx), &fleet.ActivityTypeEditedDeclarationProfile{
		TeamID:   tmID,
		TeamName: tmName,
	}); err != nil {
		return ctxerr.Wrap(ctx, err, "logging activity for edited macos declarations")
	}

	return nil
}

func (svc *Service) validateCrossPlatformProfileNames(ctx context.Context, appleProfiles []*fleet.MDMAppleConfigProfile, windowsProfiles []*fleet.MDMWindowsConfigProfile, appleDecls []*fleet.MDMAppleDeclaration) error {
	// map all profile names to check for duplicates, regardless of platform; key is name, value is one of
	// ".mobileconfig" or ".json" or ".xml"
	extByName := make(map[string]string, len(appleProfiles)+len(windowsProfiles)+len(appleDecls))
	for i, p := range appleProfiles {
		if v, ok := extByName[p.Name]; ok {
			err := fleet.NewInvalidArgumentError(fmt.Sprintf("appleProfiles[%d]", i), fmtDuplicateNameErrMsg(p.Name, ".mobileconfig", v))
			return ctxerr.Wrap(ctx, err, "duplicate mobileconfig profile by name")
		}
		extByName[p.Name] = ".mobileconfig"
	}
	for i, p := range windowsProfiles {
		if v, ok := extByName[p.Name]; ok {
			err := fleet.NewInvalidArgumentError(fmt.Sprintf("windowsProfiles[%d]", i), fmtDuplicateNameErrMsg(p.Name, ".xml", v))
			return ctxerr.Wrap(ctx, err, "duplicate xml by name")
		}
		extByName[p.Name] = ".xml"
	}
	for i, p := range appleDecls {
		if v, ok := extByName[p.Name]; ok {
			err := fleet.NewInvalidArgumentError(fmt.Sprintf("appleDecls[%d]", i), fmtDuplicateNameErrMsg(p.Name, ".json", v))
			return ctxerr.Wrap(ctx, err, "duplicate json by name")
		}
		extByName[p.Name] = ".json"
	}

	return nil
}

func fmtDuplicateNameErrMsg(name, fileType1, fileType2 string) string {
	var part1 string
	switch fileType1 {
	case ".xml":
		part1 = "Windows .xml file name"
	case ".mobileconfig":
		part1 = "macOS .mobileconfig PayloadDisplayName"
	case ".json":
		part1 = "macOS .json file name"
	}

	var part2 string
	switch fileType2 {
	case ".xml":
		part2 = "Windows .xml file name"
	case ".mobileconfig":
		part2 = "macOS .mobileconfig PayloadDisplayName"
	case ".json":
		part2 = "macOS .json file name"
	}

	base := fmt.Sprintf(`Couldn’t edit custom_settings. More than one configuration profile have the same name '%s'`, name)
	detail := ` (%s).`
	switch {
	case part1 == part2:
		return fmt.Sprintf(base+detail, part1)
	case part1 != "" && part2 != "":
		return fmt.Sprintf(base+detail, fmt.Sprintf("%s or %s", part1, part2))
	case part1 != "" || part2 != "":
		return fmt.Sprintf(base+detail, part1+part2)
	default:
		return base + "." // should never happen
	}
}

func (svc *Service) authorizeBatchProfiles(ctx context.Context, tmID *uint, tmName *string) (*uint, *string, error) {
	if tmID != nil && tmName != nil {
		svc.authz.SkipAuthorization(ctx) // so that the error message is not replaced by "forbidden"
		return nil, nil, ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("team_name", "cannot specify both team_id and team_name"))
	}
	if tmID != nil || tmName != nil {
		license, _ := license.FromContext(ctx)
		if !license.IsPremium() {
			field := "team_id"
			if tmName != nil {
				field = "team_name"
			}
			svc.authz.SkipAuthorization(ctx) // so that the error message is not replaced by "forbidden"
			return nil, nil, ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError(field, ErrMissingLicense.Error()))
		}
	}

	// if the team name is provided, load the corresponding team to get its id.
	// vice-versa, if the id is provided, load it to get the name (required for
	// the activity).
	if tmName != nil || tmID != nil {
		tm, err := svc.EnterpriseOverrides.TeamByIDOrName(ctx, tmID, tmName)
		if err != nil {
			return nil, nil, err
		}
		if tmID == nil {
			tmID = &tm.ID
		} else {
			tmName = &tm.Name
		}
	}

	if err := svc.authz.Authorize(ctx, &fleet.MDMConfigProfileAuthz{TeamID: tmID}, fleet.ActionWrite); err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err)
	}

	return tmID, tmName, nil
}

func getAppleProfiles(
	ctx context.Context,
	tmID *uint,
	appCfg *fleet.AppConfig,
	profiles []fleet.MDMProfileBatchPayload,
	labelMap map[string]fleet.ConfigurationProfileLabel,
) ([]*fleet.MDMAppleConfigProfile, []*fleet.MDMAppleDeclaration, error) {
	// any duplicate identifier or name in the provided set results in an error
	profs := make([]*fleet.MDMAppleConfigProfile, 0, len(profiles))
	decls := make([]*fleet.MDMAppleDeclaration, 0, len(profiles))
	// we need to keep track of the names and identifiers to check for duplicates so we will use
	// a map where the key is the name oridentifier and the value is either "mobileconfig" or
	// "declaration" to differentiate between the two types of profiles
	byName, byIdent := make(map[string]string, len(profiles)), make(map[string]string, len(profiles))
	for _, prof := range profiles {
		if mdm.GetRawProfilePlatform(prof.Contents) != "darwin" {
			continue
		}

		// Check for DDM files
		if mdm.GuessProfileExtension(prof.Contents) == "json" {
			rawDecl, err := fleet.GetRawDeclarationValues(prof.Contents)
			if err != nil {
				return nil, nil, err
			}

			if err := rawDecl.ValidateUserProvided(); err != nil {
				return nil, nil, err
			}

			mdmDecl := fleet.NewMDMAppleDeclaration(prof.Contents, tmID, prof.Name, rawDecl.Type, rawDecl.Identifier)
			for _, labelName := range prof.Labels {
				if lbl, ok := labelMap[labelName]; ok {
					declLabel := fleet.ConfigurationProfileLabel{
						LabelName: lbl.LabelName,
						LabelID:   lbl.LabelID,
					}
					mdmDecl.Labels = append(mdmDecl.Labels, declLabel)
				}
			}

			v, ok := byIdent[mdmDecl.Identifier]
			switch {
			case !ok:
				byIdent[mdmDecl.Identifier] = "declaration"
			case v == "mobileconfig":
				return nil, nil, ctxerr.Wrap(ctx,
					fleet.NewInvalidArgumentError(mdmDecl.Identifier, "A configuration profile with this identifier already exists."),
					"duplicate mobileconfig profile by identifier")
			case v == "declaration":
				return nil, nil, ctxerr.Wrap(ctx,
					fleet.NewInvalidArgumentError(mdmDecl.Identifier, "A declaration profile with this identifier already exists."),
					"duplicate declaration profile by identifier")
			default:
				// this should never happen but just in case
				return nil, nil, ctxerr.Wrap(ctx,
					fleet.NewInvalidArgumentError(mdmDecl.Identifier, "A profile with this identifier already exists."),
					"duplicate identifier by identifier")
			}

			decls = append(decls, mdmDecl)

			continue
		}

		mdmProf, err := fleet.NewMDMAppleConfigProfile(prof.Contents, tmID)
		if err != nil {
			return nil, nil, ctxerr.Wrap(ctx,
				fleet.NewInvalidArgumentError(prof.Name, err.Error()),
				"invalid mobileconfig profile")
		}

		for _, labelName := range prof.Labels {
			if lbl, ok := labelMap[labelName]; ok {
				mdmProf.Labels = append(mdmProf.Labels, lbl)
			}
		}

		if err := mdmProf.ValidateUserProvided(); err != nil {
			return nil, nil, ctxerr.Wrap(ctx,
				fleet.NewInvalidArgumentError(prof.Name, err.Error()))
		}

		if mdmProf.Name != prof.Name {
			return nil, nil, ctxerr.Wrap(ctx,
				fleet.NewInvalidArgumentError(prof.Name, fmt.Sprintf("Couldn’t edit custom_settings. The name provided for the profile must match the profile PayloadDisplayName: %q", mdmProf.Name)),
				"duplicate mobileconfig profile by name")
		}

		// TODO: confirm error messages
		if _, ok := byName[mdmProf.Name]; ok {
			return nil, nil, ctxerr.Wrap(ctx,
				fleet.NewInvalidArgumentError(prof.Name, fmt.Sprintf("Couldn’t edit custom_settings. More than one configuration profile have the same name (PayloadDisplayName): %q", mdmProf.Name)),
				"duplicate mobileconfig profile by name")
		}
		byName[mdmProf.Name] = "mobileconfig"

		// TODO: confirm error messages
		if _, ok := byIdent[mdmProf.Identifier]; ok {
			return nil, nil, ctxerr.Wrap(ctx,
				fleet.NewInvalidArgumentError(prof.Name, fmt.Sprintf("Couldn’t edit custom_settings. More than one configuration profile have the same identifier (PayloadIdentifier): %q", mdmProf.Identifier)),
				"duplicate mobileconfig profile by identifier")
		}
		byIdent[mdmProf.Identifier] = "mobileconfig"

		profs = append(profs, mdmProf)
	}

	if !appCfg.MDM.EnabledAndConfigured {
		// NOTE: in order to prevent an error when Fleet MDM is not enabled but no
		// profile is provided, which can happen if a user runs `fleetctl get
		// config` and tries to apply that YAML, as it will contain an empty/null
		// custom_settings key, we just return a success response in this
		// situation.
		if len(profs) == 0 {
			return []*fleet.MDMAppleConfigProfile{}, []*fleet.MDMAppleDeclaration{}, nil
		}

		return nil, nil, ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("mdm", "cannot set custom settings: Fleet MDM is not configured"))
	}

	return profs, decls, nil
}

func getWindowsProfiles(
	ctx context.Context,
	tmID *uint,
	appCfg *fleet.AppConfig,
	profiles []fleet.MDMProfileBatchPayload,
	labelMap map[string]fleet.ConfigurationProfileLabel,
) ([]*fleet.MDMWindowsConfigProfile, error) {
	profs := make([]*fleet.MDMWindowsConfigProfile, 0, len(profiles))

	for _, profile := range profiles {
		if mdm.GetRawProfilePlatform(profile.Contents) != "windows" {
			continue
		}

		mdmProf := &fleet.MDMWindowsConfigProfile{
			TeamID: tmID,
			Name:   profile.Name,
			SyncML: profile.Contents,
		}
		for _, labelName := range profile.Labels {
			if lbl, ok := labelMap[labelName]; ok {
				mdmProf.Labels = append(mdmProf.Labels, lbl)
			}
		}

		if err := mdmProf.ValidateUserProvided(); err != nil {
			return nil, ctxerr.Wrap(ctx,
				fleet.NewInvalidArgumentError(fmt.Sprintf("profiles[%s]", profile.Name), err.Error()))
		}

		profs = append(profs, mdmProf)
	}

	if !appCfg.MDM.WindowsEnabledAndConfigured {
		// NOTE: in order to prevent an error when Fleet MDM is not enabled but no
		// profile is provided, which can happen if a user runs `fleetctl get
		// config` and tries to apply that YAML, as it will contain an empty/null
		// custom_settings key, we just return a success response in this
		// situation.
		if len(profs) == 0 {
			return nil, nil
		}

		return nil, ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("mdm", "cannot set custom settings: Fleet MDM is not configured"))
	}

	return profs, nil
}

func validateProfiles(profiles []fleet.MDMProfileBatchPayload) error {
	for _, profile := range profiles {
		platform := mdm.GetRawProfilePlatform(profile.Contents)
		if platform != "darwin" && platform != "windows" {
			// TODO(roberto): there's ongoing feedback with Marko about improving this message, as it's too windows specific
			return fleet.NewInvalidArgumentError("mdm", "Windows configuration profiles can only have <Replace> or <Add> top level elements.")
		}
	}

	return nil
}

////////////////////////////////////////////////////////////////////////////////
// GET /mdm/profiles (List profiles)
////////////////////////////////////////////////////////////////////////////////

type listMDMConfigProfilesRequest struct {
	TeamID      *uint             `query:"team_id,optional"`
	ListOptions fleet.ListOptions `url:"list_options"`
}

type listMDMConfigProfilesResponse struct {
	Meta     *fleet.PaginationMetadata        `json:"meta"`
	Profiles []*fleet.MDMConfigProfilePayload `json:"profiles"`
	Err      error                            `json:"error,omitempty"`
}

func (r listMDMConfigProfilesResponse) error() error { return r.Err }

func listMDMConfigProfilesEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*listMDMConfigProfilesRequest)

	profs, meta, err := svc.ListMDMConfigProfiles(ctx, req.TeamID, req.ListOptions)
	if err != nil {
		return &listMDMConfigProfilesResponse{Err: err}, nil
	}

	res := listMDMConfigProfilesResponse{Meta: meta, Profiles: profs}
	if profs == nil {
		// return empty json array instead of json null
		res.Profiles = []*fleet.MDMConfigProfilePayload{}
	}
	return &res, nil
}

func (svc *Service) ListMDMConfigProfiles(ctx context.Context, teamID *uint, opt fleet.ListOptions) ([]*fleet.MDMConfigProfilePayload, *fleet.PaginationMetadata, error) {
	if err := svc.authz.Authorize(ctx, &fleet.MDMConfigProfileAuthz{TeamID: teamID}, fleet.ActionRead); err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err)
	}

	if teamID != nil && *teamID > 0 {
		// confirm that team exists
		if _, err := svc.ds.Team(ctx, *teamID); err != nil {
			return nil, nil, ctxerr.Wrap(ctx, err)
		}
	}

	// cursor-based pagination is not supported for profiles
	opt.After = ""
	// custom ordering is not supported, always by name
	opt.OrderKey = "name"
	opt.OrderDirection = fleet.OrderAscending
	// no matching query support
	opt.MatchQuery = ""
	// always include metadata for profiles
	opt.IncludeMetadata = true

	return svc.ds.ListMDMConfigProfiles(ctx, teamID, opt)
}

////////////////////////////////////////////////////////////////////////////////
// Update MDM Disk encryption
////////////////////////////////////////////////////////////////////////////////

type updateMDMDiskEncryptionRequest struct {
	TeamID               *uint `json:"team_id"`
	EnableDiskEncryption bool  `json:"enable_disk_encryption"`
}

type updateMDMDiskEncryptionResponse struct {
	Err error `json:"error,omitempty"`
}

func (r updateMDMDiskEncryptionResponse) error() error { return r.Err }

func (r updateMDMDiskEncryptionResponse) Status() int { return http.StatusNoContent }

func updateMDMDiskEncryptionEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*updateMDMDiskEncryptionRequest)
	if err := svc.UpdateMDMDiskEncryption(ctx, req.TeamID, &req.EnableDiskEncryption); err != nil {
		return updateMDMDiskEncryptionResponse{Err: err}, nil
	}
	return updateMDMDiskEncryptionResponse{}, nil
}

func (svc *Service) UpdateMDMDiskEncryption(ctx context.Context, teamID *uint, enableDiskEncryption *bool) error {
	// TODO(mna): this should all move to the ee package when we remove the
	// `PATCH /api/v1/fleet/mdm/apple/settings` endpoint, but for now it's better
	// leave here so both endpoints can reuse the same logic.

	lic, _ := license.FromContext(ctx)
	if lic == nil || !lic.IsPremium() {
		svc.authz.SkipAuthorization(ctx) // so that the error message is not replaced by "forbidden"
		return ErrMissingLicense
	}

	// for historical reasons (the deprecated PATCH /mdm/apple/settings
	// endpoint), this uses an Apple-specific struct for authorization. Can be improved
	// once we remove the deprecated endpoint.
	if err := svc.authz.Authorize(ctx, fleet.MDMAppleSettingsPayload{TeamID: teamID}, fleet.ActionWrite); err != nil {
		return ctxerr.Wrap(ctx, err)
	}

	if teamID != nil {
		tm, err := svc.EnterpriseOverrides.TeamByIDOrName(ctx, teamID, nil)
		if err != nil {
			return err
		}
		return svc.EnterpriseOverrides.UpdateTeamMDMDiskEncryption(ctx, tm, enableDiskEncryption)
	}
	return svc.updateAppConfigMDMDiskEncryption(ctx, enableDiskEncryption)
}

////////////////////////////////////////////////////////////////////////////////
// POST /hosts/{id:[0-9]+}/configuration_profiles/{profile_uuid}
////////////////////////////////////////////////////////////////////////////////

type resendHostMDMProfileRequest struct {
	HostID      uint   `url:"host_id"`
	ProfileUUID string `url:"profile_uuid"`
}

type resendHostMDMProfileResponse struct {
	Err error `json:"error,omitempty"`
}

func (r resendHostMDMProfileResponse) error() error { return r.Err }

func (r resendHostMDMProfileResponse) Status() int { return http.StatusAccepted }

func resendHostMDMProfileEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*resendHostMDMProfileRequest)

	if err := svc.ResendHostMDMProfile(ctx, req.HostID, req.ProfileUUID); err != nil {
		return resendHostMDMProfileResponse{Err: err}, nil
	}

	return resendHostMDMProfileResponse{}, nil
}

func (svc *Service) ResendHostMDMProfile(ctx context.Context, hostID uint, profileUUID string) error {
	// first we perform a perform basic authz check, we use selective list action to include gitops users
	if err := svc.authz.Authorize(ctx, &fleet.Host{}, fleet.ActionSelectiveList); err != nil {
		return ctxerr.Wrap(ctx, err)
	}

	host, err := svc.ds.HostLite(ctx, hostID)
	if err != nil {
		return ctxerr.Wrap(ctx, err)
	}

	// now we can do a specific authz check based on team id of the host before proceeding
	if err := svc.authz.Authorize(ctx, &fleet.MDMConfigProfileAuthz{TeamID: host.TeamID}, fleet.ActionWrite); err != nil {
		return ctxerr.Wrap(ctx, err)
	}

	var profileTeamID *uint
	var profileName string
	switch {
	case strings.HasPrefix(profileUUID, fleet.MDMAppleProfileUUIDPrefix):
		if err := svc.VerifyMDMAppleConfigured(ctx); err != nil {
			return ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("HostMDMProfile", fleet.AppleMDMNotConfiguredMessage).WithStatus(http.StatusBadRequest), "check apple mdm enabled")
		}
		if host.Platform != "darwin" {
			return ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("HostMDMProfile", "Profile is not compatible with host platform."), "check host platform")
		}
		prof, err := svc.ds.GetMDMAppleConfigProfile(ctx, profileUUID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "getting apple config profile")
		}
		profileTeamID = prof.TeamID
		profileName = prof.Name

	case strings.HasPrefix(profileUUID, fleet.MDMAppleDeclarationUUIDPrefix):
		if err := svc.VerifyMDMAppleConfigured(ctx); err != nil {
			return ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("HostMDMProfile", fleet.AppleMDMNotConfiguredMessage).WithStatus(http.StatusBadRequest), "check apple mdm enabled")
		}
		if host.Platform != "darwin" {
			return ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("HostMDMProfile", "Profile is not compatible with host platform."), "check host platform")
		}
		decl, err := svc.ds.GetMDMAppleDeclaration(ctx, profileUUID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "getting apple declaration")
		}
		profileTeamID = decl.TeamID
		profileName = decl.Name

	case strings.HasPrefix(profileUUID, fleet.MDMWindowsProfileUUIDPrefix):
		if err := svc.VerifyMDMWindowsConfigured(ctx); err != nil {
			return ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("HostMDMProfile", fleet.WindowsMDMNotConfiguredMessage).WithStatus(http.StatusBadRequest), "check windows mdm enabled")
		}
		if host.Platform != "windows" {
			return ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("HostMDMProfile", "Profile is not compatible with host platform."), "check host platform")
		}
		prof, err := svc.ds.GetMDMWindowsConfigProfile(ctx, profileUUID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "getting windows config profile")
		}
		profileTeamID = prof.TeamID
		profileName = prof.Name

	default:
		return ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("HostMDMProfile", "Invalid profile UUID prefix.").WithStatus(http.StatusNotFound), "check profile UUID prefix")
	}

	// check again based on team id of profile before we proceeding
	if err := svc.authz.Authorize(ctx, &fleet.MDMConfigProfileAuthz{TeamID: profileTeamID}, fleet.ActionWrite); err != nil {
		return ctxerr.Wrap(ctx, err, "authorizing profile team")
	}

	status, err := svc.ds.GetHostMDMProfileInstallStatus(ctx, host.UUID, profileUUID)
	if err != nil {
		if fleet.IsNotFound(err) {
			return ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("HostMDMProfile", "Unable to match profile to host.").WithStatus(http.StatusNotFound), "getting host mdm profile status")
		}
		return ctxerr.Wrap(ctx, err, "getting host mdm profile status")
	}
	if status == fleet.MDMDeliveryPending || status == fleet.MDMDeliveryVerifying {
		return ctxerr.Wrap(ctx, fleet.NewInvalidArgumentError("HostMDMProfile", "Couldn’t resend. Configuration profiles with “pending” or “verifying” status can’t be resent.").WithStatus(http.StatusConflict), "check profile status")
	}
	if status != fleet.MDMDeliveryFailed && status != fleet.MDMDeliveryVerified {
		// this should never happen, but just in case
		return ctxerr.Errorf(ctx, "unrecognized profile status %s", status)
	}

	if err := svc.ds.ResendHostMDMProfile(ctx, host.UUID, profileUUID); err != nil {
		return ctxerr.Wrap(ctx, err, "resending host mdm profile")
	}

	if err := svc.ds.NewActivity(ctx, authz.UserFromContext(ctx), &fleet.ActivityTypeResentConfigurationProfile{
		HostID:          &host.ID,
		HostDisplayName: ptr.String(host.DisplayName()),
		ProfileName:     profileName,
	}); err != nil {
		return ctxerr.Wrap(ctx, err, "logging activity for resend config profile")
	}

	return nil
}
