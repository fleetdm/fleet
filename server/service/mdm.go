package service

import (
	"context"
	"encoding/xml"
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
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/logging"
	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
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
	DeviceIDs []string `json:"device_ids"`
}

type runMDMCommandResponse struct {
	*fleet.CommandEnqueueResult
	Platform string `json:"platform"`
	Err      error  `json:"error,omitempty"`
}

func (r runMDMCommandResponse) error() error { return r.Err }

func runMDMCommandEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (errorer, error) {
	req := request.(*runMDMCommandRequest)
	result, err := svc.RunMDMCommand(ctx, req.Command, req.DeviceIDs)
	if err != nil {
		return runMDMCommandResponse{Err: err}, nil
	}
	return runMDMCommandResponse{
		CommandEnqueueResult: result,
	}, nil
}

func (svc *Service) RunMDMCommand(ctx context.Context, rawBase64Cmd string, deviceIDs []string) (result *fleet.CommandEnqueueResult, err error) {
	// TODO(mna): epic mentions "GitOps users can run commands", check that it is what we want
	// (how can gitops run a command?). This complicates a lot of things, as gitops has no read
	// permissions at all (e.g. HostByIdentifier call in fleetctl, list permission here, etc.).

	// TODO(mna): this doesn't work as it assumes a Microsoft Authz (it could be macOS, we don't know
	// until we've checked the hosts). I think that now that this is a platform-agnostic endpoint,
	// authorization would be the same regardless of platform. Confirm this and we can reuse a single
	// MDMApple/Microsoft/CommandAuthz struct.
	hosts, err := svc.authorizeAllHostsTeams(ctx, deviceIDs, false, fleet.ActionWrite, &fleet.MDMMicrosoftCommandAuthz{})
	if err != nil {
		return nil, err
	}
	if len(hosts) == 0 {
		err := fleet.NewInvalidArgumentError("device_ids", "No hosts targeted. Make sure you provide a valid UUID.").WithStatus(http.StatusNotFound)
		return nil, ctxerr.Wrap(ctx, err, "no host received")
	}

	platforms := make(map[string]bool)
	for _, h := range hosts {
		// TODO(mna): hosts lite does not have mdm info, add it to hosts lite or
		// create a new get non-lite hosts by uuids?
		if !h.MDMInfo.IsFleetEnrolled() {
			err := fleet.NewInvalidArgumentError("device_ids", "Can't run the MDM command because one or more hosts have MDM turned off. Run the following command to see a list of hosts with MDM on: fleetctl get hosts --mdm.").WithStatus(http.StatusPreconditionFailed)
			return nil, ctxerr.Wrap(ctx, err, "check host mdm enrollment")
		}
		platforms[h.FleetPlatform()] = true
	}
	if len(platforms) != 1 {
		err := fleet.NewInvalidArgumentError("device_ids", "All hosts must be on the same platform.")
		return nil, ctxerr.Wrap(ctx, err, "check host platform")
	}

	// it's a for loop but at this point it's guaranteed that the map has a single value.
	var commandPlatform string
	for platform := range platforms {
		commandPlatform = platform
	}
	if commandPlatform != "windows" && commandPlatform != "darwin" {
		err := fleet.NewInvalidArgumentError("device_ids", "Invalid platform. You can only run MDM commands on Windows or macOS hosts.")
		return nil, ctxerr.Wrap(ctx, err, "check host platform")
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
		return svc.enqueueMicrosoftMDMCommand(ctx, rawXMLCmd, deviceIDs)
	default:
		return svc.enqueueAppleMDMCommand(ctx, rawXMLCmd, deviceIDs)
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
					fmt.Sprintf("at least one of the hosts is not enrolled in MDM: %v", err),
				).WithStatus(http.StatusConflict)
				return nil, ctxerr.Wrap(ctx, err, "enqueue command")
			}
		}

		return nil, ctxerr.Wrap(ctx, err, "enqueue command")
	}
	return &fleet.CommandEnqueueResult{
		CommandUUID: cmd.CommandUUID,
		RequestType: cmd.Command.RequestType,
	}, nil
}

func (svc *Service) enqueueMicrosoftMDMCommand(ctx context.Context, rawXMLCmd []byte, deviceIDs []string) (result *fleet.CommandEnqueueResult, err error) {
	// a command is a SyncML message
	var cmdMsg fleet.SyncML
	if err := xml.Unmarshal(rawXMLCmd, &cmdMsg); err != nil {
		err = fleet.NewInvalidArgumentError("command", fmt.Sprintf("The payload isn't valid XML: %v", err))
		return nil, ctxerr.Wrap(ctx, err, "decode SyncML command")
	}
	if err := cmdMsg.IsValidHeader(); err != nil {
		err = fleet.NewInvalidArgumentError("command", fmt.Sprintf("The payload isn't a valid MDM command: %v", err))
		return nil, ctxerr.Wrap(ctx, err, "validate SyncML header")
	}
	if err := cmdMsg.IsValidBody(); err != nil {
		err = fleet.NewInvalidArgumentError("command", fmt.Sprintf("The payload isn't a valid MDM command: %v", err))
		return nil, ctxerr.Wrap(ctx, err, "validate SyncML body")
	}

	// any non-exec commands provided?
	if len(cmdMsg.SyncBody.Add)+len(cmdMsg.SyncBody.Alert)+len(cmdMsg.SyncBody.Atomic)+len(cmdMsg.SyncBody.Delete)+
		len(cmdMsg.SyncBody.Get)+len(cmdMsg.SyncBody.Replace)+len(cmdMsg.SyncBody.Status)+len(cmdMsg.SyncBody.Results)+
		len(cmdMsg.SyncBody.Raw) > 0 {
		err = fleet.NewInvalidArgumentError("command", "You can run only <Exec> command type.")
		return nil, ctxerr.Wrap(ctx, err, "validate SyncML commands")
	}
	if len(cmdMsg.SyncBody.Exec) != 1 {
		err = fleet.NewInvalidArgumentError("command", "You can run only a single <Exec> command.")
		return nil, ctxerr.Wrap(ctx, err, "validate SyncML Exec commands")
	}

	cmd := cmdMsg.SyncBody.Exec[0]
	if cmd.IsPremium() {
		lic, err := svc.License(ctx)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "get license")
		}
		if !lic.IsPremium() {
			return nil, fleet.ErrMissingLicense
		}
	}

	// TODO(mna): enqueue in the windows mdm command table

	return &fleet.CommandEnqueueResult{}, nil
}
