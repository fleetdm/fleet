package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"path"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	hostctx "github.com/fleetdm/fleet/v4/server/contexts/host"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/go-kit/log/level"
)

// Ping device endpoint
// NOTE: we're intentionally not reading the capabilities header in this
// endpoint as is unauthenticated and we don't want to trust whatever comes in
// there.
func devicePingEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	svc.DisableAuthForPing(ctx)
	return fleet.DevicePingResponse{}, nil
}

func (svc *Service) DisableAuthForPing(ctx context.Context) {
	// skipauth: this endpoint is intentionally public to allow devices to ping
	// the server and among other things, get the fleet.Capabilities header to
	// determine which capabilities are enabled in the server.
	svc.authz.SkipAuthorization(ctx)
}

// Fleet Desktop endpoints
// getFleetDesktopEndpoint is meant to be the only API endpoint used by Fleet Desktop. This
// endpoint should not include any kind of identifying information about the host.
func getFleetDesktopEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	sum, err := svc.GetFleetDesktopSummary(ctx)
	if err != nil {
		return fleet.FleetDesktopResponse{Err: err}, nil
	}
	return fleet.FleetDesktopResponse{DesktopSummary: sum}, nil
}

func (svc *Service) GetFleetDesktopSummary(ctx context.Context) (fleet.DesktopSummary, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return fleet.DesktopSummary{}, fleet.ErrMissingLicense
}

// Get Current Device's Host
func getDeviceHostEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.GetDeviceHostRequest)
	host, ok := hostctx.FromContext(ctx)
	if !ok {
		err := ctxerr.Wrap(ctx, fleet.NewAuthRequiredError("internal error: missing host from request context"))
		return fleet.GetDeviceHostResponse{Err: err}, nil
	}

	// must still load the full host details, as it returns more information
	opts := fleet.HostDetailOptions{
		IncludeCVEScores: false,
		IncludePolicies:  false,
		ExcludeSoftware:  req.ExcludeSoftware,
	}
	hostDetails, err := svc.GetHost(ctx, host.ID, opts)
	if err != nil {
		return fleet.GetDeviceHostResponse{Err: err}, nil
	}

	resp, err := hostDetailResponseForHost(ctx, svc, hostDetails)
	if err != nil {
		return fleet.GetDeviceHostResponse{Err: err}, nil
	}

	// the org logo URL config is required by the frontend to render the page;
	// we need to be careful with what we return from AppConfig in the response
	// as this is a weakly authenticated endpoint (with the device auth token).
	ac, err := svc.AppConfigObfuscated(ctx)
	if err != nil {
		return fleet.GetDeviceHostResponse{Err: err}, nil
	}

	license, err := svc.License(ctx)
	if err != nil {
		return fleet.GetDeviceHostResponse{Err: err}, nil
	}

	// Scrub sensitive data from the host response for iOS and iPadOS devices
	if authzCtx, ok := authz.FromContext(ctx); ok && authzCtx.AuthnMethod() == authz.AuthnDeviceURL {
		if host.Platform == "ios" || host.Platform == "ipados" {
			resp.HardwareSerial = ""
			resp.UUID = ""
			resp.PrimaryMac = ""
			resp.TeamName = nil
			resp.MDM.Profiles = nil
			resp.Labels = nil
			resp.Hostname = ""
			resp.ComputerName = ""
			resp.DisplayText = ""
			resp.DisplayName = ""

			// Scrub sensitive data from the license response
			scrubbedLicense := *license
			scrubbedLicense.Organization = ""
			scrubbedLicense.DeviceCount = 0
			scrubbedLicense.Expiration = time.Time{}
			license = &scrubbedLicense
		}
	}

	resp.DEPAssignedToFleet = ptr.Bool(false)
	if ac.MDM.EnabledAndConfigured && license.IsPremium() {
		hdep, err := svc.GetHostDEPAssignment(ctx, host)
		if err != nil && !fleet.IsNotFound(err) {
			return fleet.GetDeviceHostResponse{Err: err}, nil
		}
		resp.DEPAssignedToFleet = ptr.Bool(hdep.IsDEPAssignedToFleet())
	}

	softwareInventoryEnabled := ac.Features.EnableSoftwareInventory
	requireAllSoftware := ac.MDM.MacOSSetup.RequireAllSoftware
	var conditionalAccessEnabled bool
	if resp.TeamID != nil {
		// load the team to get the device's team's software inventory config.
		tm, err := svc.GetTeam(ctx, *resp.TeamID)
		if err != nil && !fleet.IsNotFound(err) {
			return fleet.GetDeviceHostResponse{Err: err}, nil
		}
		if tm != nil {
			softwareInventoryEnabled = tm.Config.Features.EnableSoftwareInventory // TODO: We should look for opportunities to fix the confusing name of the `global_config` object in the API response. Also, how can we better clarify/document the expected order of precedence for team and global feature flags?
			requireAllSoftware = tm.Config.MDM.MacOSSetup.RequireAllSoftware
			conditionalAccessEnabled = ac.ConditionalAccess.OktaConfigured() && tm.Config.Integrations.ConditionalAccessEnabled.Valid && tm.Config.Integrations.ConditionalAccessEnabled.Value
		}
	}

	hasSelfService := false
	if softwareInventoryEnabled {
		hasSelfService, err = svc.HasSelfServiceSoftwareInstallers(ctx, host)
		if err != nil {
			return fleet.GetDeviceHostResponse{Err: err}, nil
		}
	}

	deviceGlobalConfig := fleet.DeviceGlobalConfig{
		MDM: fleet.DeviceGlobalMDMConfig{
			// TODO(mna): It currently only returns the Apple enabled and configured,
			// regardless of the platform of the device. See
			// https://github.com/fleetdm/fleet/pull/19304#discussion_r1618792410.
			EnabledAndConfigured: ac.MDM.EnabledAndConfigured,
			RequireAllSoftware:   requireAllSoftware,
		},
		Features: fleet.DeviceFeatures{
			EnableSoftwareInventory:       softwareInventoryEnabled,
			EnableConditionalAccess:       conditionalAccessEnabled,
			EnableConditionalAccessBypass: ac.ConditionalAccess != nil && ac.ConditionalAccess.BypassEnabled(),
		},
	}

	return fleet.GetDeviceHostResponse{
		Host:                      resp,
		OrgLogoURL:                ac.OrgInfo.OrgLogoURL,
		OrgLogoURLLightBackground: ac.OrgInfo.OrgLogoURLLightBackground,
		OrgContactURL:             ac.OrgInfo.ContactURL,
		License:                   *license,
		GlobalConfig:              deviceGlobalConfig,
		SelfService:               hasSelfService,
	}, nil
}

func (svc *Service) GetHostDEPAssignment(ctx context.Context, host *fleet.Host) (*fleet.HostDEPAssignment, error) {
	alreadyAuthd := svc.authz.IsAuthenticatedWith(ctx, authz.AuthnDeviceToken) ||
		svc.authz.IsAuthenticatedWith(ctx, authz.AuthnDeviceCertificate) ||
		svc.authz.IsAuthenticatedWith(ctx, authz.AuthnDeviceURL)
	if !alreadyAuthd {
		if err := svc.authz.Authorize(ctx, host, fleet.ActionRead); err != nil {
			return nil, err
		}
	}
	return svc.ds.GetHostDEPAssignment(ctx, host.ID)
}

// AuthenticateDevice returns the host identified by the device authentication
// token, along with a boolean indicating if debug logging is enabled for that
// host.
func (svc *Service) AuthenticateDevice(ctx context.Context, authToken string) (*fleet.Host, bool, error) {
	const deviceAuthTokenTTL = time.Hour
	// skipauth: Authorization is currently for user endpoints only.
	svc.authz.SkipAuthorization(ctx)

	if authToken == "" {
		return nil, false, ctxerr.Wrap(ctx, fleet.NewAuthRequiredError("authentication error: missing device authentication token"))
	}

	host, err := svc.ds.LoadHostByDeviceAuthToken(ctx, authToken, deviceAuthTokenTTL)
	switch {
	case err == nil:
		// OK
	case fleet.IsNotFound(err):
		return nil, false, ctxerr.Wrap(ctx, fleet.NewAuthRequiredError("authentication error: invalid device authentication token"))
	default:
		return nil, false, ctxerr.Wrap(ctx, err, "authenticate device")
	}

	// iOS/iPadOS must use certificate authentication.
	if host.Platform == "ios" || host.Platform == "ipados" {
		return nil, false, ctxerr.Wrap(ctx, fleet.NewAuthRequiredError("authentication error: iOS and iPadOS devices must use certificate authentication"))
	}

	return host, svc.debugEnabledForHost(ctx, host.ID), nil
}

// AuthenticateDeviceByCertificate returns the host identified by the certificate
// serial number and host UUID. This is used for iOS/iPadOS devices accessing the
// My Device page via client certificate authentication. The certificate must match
// the host's identity certificate, and the host must be iOS or iPadOS.
func (svc *Service) AuthenticateDeviceByCertificate(ctx context.Context, certSerial uint64, hostUUID string) (*fleet.Host, bool, error) {
	// skipauth: Authorization is currently for user endpoints only.
	svc.authz.SkipAuthorization(ctx)

	if certSerial == 0 {
		return nil, false, ctxerr.Wrap(ctx, fleet.NewAuthRequiredError("authentication error: missing certificate serial"))
	}

	if hostUUID == "" {
		return nil, false, ctxerr.Wrap(ctx, fleet.NewAuthRequiredError("authentication error: missing host UUID"))
	}

	// Look up the MDM SCEP certificate by serial number to get the device UUID
	certDeviceUUID, err := svc.ds.GetMDMSCEPCertBySerial(ctx, certSerial)
	switch {
	case err == nil:
		// OK
	case fleet.IsNotFound(err):
		return nil, false, ctxerr.Wrap(ctx, fleet.NewAuthRequiredError("authentication error: invalid or missing certificate"))
	default:
		return nil, false, ctxerr.Wrap(ctx, err, "lookup certificate by serial")
	}

	// Verify certificate's device UUID matches the requested host UUID
	if certDeviceUUID != hostUUID {
		return nil, false, ctxerr.Wrap(ctx, fleet.NewAuthRequiredError("authentication error: certificate does not match host"))
	}

	// Look up the host by UUID
	host, err := svc.ds.HostByIdentifier(ctx, hostUUID)
	switch {
	case err == nil:
		// OK
	case fleet.IsNotFound(err):
		return nil, false, ctxerr.Wrap(ctx, fleet.NewAuthRequiredError("authentication error: host not found"))
	default:
		return nil, false, ctxerr.Wrap(ctx, err, "lookup host by UUID")
	}

	// Verify host platform is iOS or iPadOS
	if host.Platform != "ios" && host.Platform != "ipados" {
		return nil, false, ctxerr.Wrap(ctx, fleet.NewAuthRequiredError("authentication error: certificate authentication only supported for iOS and iPadOS devices"))
	}

	return host, svc.debugEnabledForHost(ctx, host.ID), nil
}

// AuthenticateIDeviceByURL returns the host identified by the URL UUID.
// This is used for iOS/iPadOS devices (iDevices) accessing endpoints via a unique URL parameter.
// Returns an error if the UUID doesn't exist or if the host is not iOS/iPadOS.
func (svc *Service) AuthenticateIDeviceByURL(ctx context.Context, urlUUID string) (*fleet.Host, bool, error) {
	// skipauth: Authorization is currently for user endpoints only.
	svc.authz.SkipAuthorization(ctx)

	if urlUUID == "" {
		return nil, false, ctxerr.Wrap(ctx, fleet.NewAuthRequiredError("authentication error: missing host UUID"))
	}

	// Look up the host by UUID
	host, err := svc.ds.HostByIdentifier(ctx, urlUUID)
	switch {
	case err == nil:
		// OK
	case fleet.IsNotFound(err):
		return nil, false, ctxerr.Wrap(ctx, fleet.NewAuthRequiredError("authentication error: host not found"))
	default:
		return nil, false, ctxerr.Wrap(ctx, err, "lookup host by UUID")
	}

	// Verify host platform is iOS or iPadOS
	if host.Platform != "ios" && host.Platform != "ipados" {
		return nil, false, ctxerr.Wrap(ctx, fleet.NewAuthRequiredError("authentication error: URL authentication only supported for iOS and iPadOS devices"))
	}

	return host, svc.debugEnabledForHost(ctx, host.ID), nil
}

// Refetch Current Device's Host
func refetchDeviceHostEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	host, ok := hostctx.FromContext(ctx)
	if !ok {
		err := ctxerr.Wrap(ctx, fleet.NewAuthRequiredError("internal error: missing host from request context"))
		return fleet.RefetchHostResponse{Err: err}, nil
	}

	err := svc.RefetchHost(ctx, host.ID)
	if err != nil {
		return fleet.RefetchHostResponse{Err: err}, nil
	}
	return fleet.RefetchHostResponse{}, nil
}

// List Current Device's Host Device Mappings
func listDeviceHostDeviceMappingEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	host, ok := hostctx.FromContext(ctx)
	if !ok {
		err := ctxerr.Wrap(ctx, fleet.NewAuthRequiredError("internal error: missing host from request context"))
		return fleet.ListHostDeviceMappingResponse{Err: err}, nil
	}

	dms, err := svc.ListHostDeviceMapping(ctx, host.ID)
	if err != nil {
		return fleet.ListHostDeviceMappingResponse{Err: err}, nil
	}
	return fleet.ListHostDeviceMappingResponse{HostID: host.ID, DeviceMapping: dms}, nil
}

// Get Current Device's Macadmins
func getDeviceMacadminsDataEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	host, ok := hostctx.FromContext(ctx)
	if !ok {
		err := ctxerr.Wrap(ctx, fleet.NewAuthRequiredError("internal error: missing host from request context"))
		return fleet.GetMacadminsDataResponse{Err: err}, nil
	}

	data, err := svc.MacadminsData(ctx, host.ID)
	if err != nil {
		return fleet.GetMacadminsDataResponse{Err: err}, nil
	}
	return fleet.GetMacadminsDataResponse{Macadmins: data}, nil
}

// List Current Device's Policies
func listDevicePoliciesEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	host, ok := hostctx.FromContext(ctx)
	if !ok {
		err := ctxerr.Wrap(ctx, fleet.NewAuthRequiredError("internal error: missing host from request context"))
		return fleet.ListDevicePoliciesResponse{Err: err}, nil
	}

	data, err := svc.ListDevicePolicies(ctx, host)
	if err != nil {
		return fleet.ListDevicePoliciesResponse{Err: err}, nil
	}

	return fleet.ListDevicePoliciesResponse{Policies: data}, nil
}

func (svc *Service) ListDevicePolicies(ctx context.Context, host *fleet.Host) ([]*fleet.HostPolicy, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}

// Bypass conditional access
func bypassConditionalAccessEndpoint(ctx context.Context, request any, svc fleet.Service) (fleet.Errorer, error) {
	host, ok := hostctx.FromContext(ctx)
	if !ok {
		err := ctxerr.Wrap(ctx, fleet.NewAuthRequiredError("internal error: missing host from request context"))
		return fleet.BypassConditionalAccessResponse{Err: err}, nil
	}

	if err := svc.BypassConditionalAccess(ctx, host); err != nil {
		return fleet.BypassConditionalAccessResponse{Err: err}, nil
	}

	return fleet.BypassConditionalAccessResponse{}, nil
}

func (svc *Service) BypassConditionalAccess(ctx context.Context, host *fleet.Host) error {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return fleet.ErrMissingLicense
}

// Resend configuration profile
func resendDeviceConfigurationProfileEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	host, ok := hostctx.FromContext(ctx)
	if !ok {
		err := ctxerr.Wrap(ctx, fleet.NewAuthRequiredError("internal error: missing host from request context"))
		return fleet.ResendDeviceConfigurationProfileResponse{Err: err}, nil
	}

	req := request.(*fleet.ResendDeviceConfigurationProfileRequest)
	err := svc.ResendDeviceHostMDMProfile(ctx, host, req.ProfileUUID)
	if err != nil {
		return fleet.ResendDeviceConfigurationProfileResponse{
			Err: err,
		}, nil
	}

	return fleet.ResendDeviceConfigurationProfileResponse{}, nil
}

// Get software MDM command results
func getDeviceMDMCommandResultsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	_, ok := hostctx.FromContext(ctx)
	if !ok {
		err := ctxerr.Wrap(ctx, fleet.NewAuthRequiredError("internal error: missing host from request context"))
		return fleet.GetMDMCommandResultsResponse{Err: err}, nil
	}

	req := request.(*fleet.GetDeviceMDMCommandResultsRequest)
	results, err := svc.GetMDMCommandResults(ctx, req.CommandUUID, "")
	if err != nil {
		return fleet.GetMDMCommandResultsResponse{
			Err: err,
		}, nil
	}

	return fleet.GetMDMCommandResultsResponse{
		Results: results,
	}, nil
}

// Transparency URL Redirect
func transparencyURL(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	transparencyURL, err := svc.GetTransparencyURL(ctx)

	return fleet.TransparencyURLResponse{RedirectURL: transparencyURL, Err: err}, nil
}

func (svc *Service) GetTransparencyURL(ctx context.Context) (string, error) {
	config, err := svc.AppConfigObfuscated(ctx)
	if err != nil {
		return "", err
	}

	license, err := svc.License(ctx)
	if err != nil {
		return "", err
	}

	transparencyURL := fleet.DefaultTransparencyURL
	// See #27309; overridden if on Fleet Premium and custom transparency URL is set
	if svc.config.Partnerships.EnableSecureframe {
		transparencyURL = fleet.SecureframeTransparencyURL
	}

	// Fleet Premium license is required for custom transparency URL
	if license.IsPremium() && config.FleetDesktop.TransparencyURL != "" {
		transparencyURL = config.FleetDesktop.TransparencyURL
	}

	return transparencyURL, nil
}

// ///////////////////////////////////////////////////////////////////////////////
// Software title icons
// ///////////////////////////////////////////////////////////////////////////////

func getDeviceSoftwareIconEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	host, ok := hostctx.FromContext(ctx)
	if !ok {
		err := ctxerr.Wrap(ctx, fleet.NewAuthRequiredError("internal error: missing host from request context"))
		return fleet.GetDeviceSoftwareIconResponse{Err: err}, nil
	}

	req := request.(*fleet.GetDeviceSoftwareIconRequest)
	var teamID uint
	if host.TeamID != nil {
		teamID = *host.TeamID
	}
	iconData, size, filename, err := svc.GetDeviceSoftwareIconsTitleIcon(ctx, teamID, req.SoftwareTitleID)
	if err != nil {
		var vppErr *fleet.VPPIconAvailable
		if errors.As(err, &vppErr) {
			// 302 redirect to vpp app IconURL
			return fleet.GetDeviceSoftwareIconRedirectResponse{RedirectURL: vppErr.IconURL}, nil
		}
		return fleet.GetDeviceSoftwareIconResponse{Err: err}, nil
	}

	return fleet.GetDeviceSoftwareIconResponse{
		ImageData:   iconData,
		ContentType: "image/png", // only type of icon we currently allow
		Filename:    filename,
		Size:        size,
	}, nil
}

func (svc *Service) GetDeviceSoftwareIconsTitleIcon(ctx context.Context, teamID uint, titleID uint) ([]byte, int64, string, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, 0, "", fleet.ErrMissingLicense
}

// Receive errors from the client
// Since we're directly storing what we get in Redis, limit the request size to
// 5MB, this combined with the rate limit of this endpoint should be enough to
// prevent a malicious actor.
// body limiting is done at the handler level

func fleetdError(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleet.FleetdErrorRequest)
	err := svc.LogFleetdError(ctx, req.FleetdError)
	if err != nil {
		return nil, err
	}
	return fleet.FleetdErrorResponse{}, nil
}

func (svc *Service) LogFleetdError(ctx context.Context, fleetdError fleet.FleetdError) error {
	// iOS/iPadOS devices don't have fleetd, so URL auth is not allowed here.
	if !svc.authz.IsAuthenticatedWith(ctx, authz.AuthnDeviceToken) &&
		!svc.authz.IsAuthenticatedWith(ctx, authz.AuthnDeviceCertificate) {
		return ctxerr.Wrap(ctx, fleet.NewPermissionError("forbidden: only device-authenticated hosts can access this endpoint"))
	}

	err := ctxerr.WrapWithData(ctx, fleetdError, "receive fleetd error", fleetdError.ToMap())
	level.Warn(svc.logger).Log(
		"msg",
		"fleetd error",
		"error",
		err,
	)
	// Send to Redis/telemetry (if enabled)
	ctxerr.Handle(ctx, err)

	return nil
}

// Get Current Device's MDM Apple Enrollment Profile
func getDeviceMDMManualEnrollProfileEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	// this call ensures that the authentication was done, no need to actually
	// use the host
	if _, ok := hostctx.FromContext(ctx); !ok {
		err := ctxerr.Wrap(ctx, fleet.NewAuthRequiredError("internal error: missing host from request context"))
		return fleet.GetDeviceMDMManualEnrollProfileResponse{Err: err}, nil
	}

	enrollURL, err := svc.GetDeviceMDMAppleEnrollmentProfile(ctx)
	if err != nil {
		return fleet.GetDeviceMDMManualEnrollProfileResponse{Err: err}, nil
	}
	return fleet.GetDeviceMDMManualEnrollProfileResponse{EnrollURL: enrollURL.String()}, nil
}

func (svc *Service) GetDeviceMDMAppleEnrollmentProfile(ctx context.Context) (*url.URL, error) {
	// must be device-authenticated, no additional authorization is required
	// iOS/iPadOS devices are enrolled via MDM profile or ABM, so URL auth is not allowed here.
	if !svc.authz.IsAuthenticatedWith(ctx, authz.AuthnDeviceToken) &&
		!svc.authz.IsAuthenticatedWith(ctx, authz.AuthnDeviceCertificate) {
		return nil, ctxerr.Wrap(ctx, fleet.NewPermissionError("forbidden: only device-authenticated hosts can access this endpoint"))
	}

	cfg, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "fetching app config")
	}

	host, ok := hostctx.FromContext(ctx)
	if !ok {
		return nil, ctxerr.Wrap(ctx, fleet.NewAuthRequiredError("internal error: missing host from request context"))
	}

	tmSecrets, err := svc.ds.GetEnrollSecrets(ctx, host.TeamID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, ctxerr.Wrap(ctx, err, "getting host team enroll secrets")
	}
	if len(tmSecrets) == 0 && host.TeamID != nil {
		tmSecrets, err = svc.ds.GetEnrollSecrets(ctx, nil)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, ctxerr.Wrap(ctx, err, "getting no team enroll secrets")
		}
	}
	if len(tmSecrets) == 0 {
		return nil, &fleet.BadRequestError{Message: "unable to find an enroll secret to generate enrollment profile"}
	}
	var enrollSecret fleet.EnrollSecret
	for _, s := range tmSecrets {
		if s.CreatedAt.After(enrollSecret.CreatedAt) {
			enrollSecret = *s
		}
	}
	url, err := url.Parse(cfg.MDMUrl())
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "parsing MDM URL from config")
	}
	url.Path = path.Join(url.Path, "enroll")
	q := url.Query()
	q.Set("enroll_secret", enrollSecret.Secret)
	url.RawQuery = q.Encode()

	return url, nil
}

// Signal start of mdm migration on a device
func migrateMDMDeviceEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	host, ok := hostctx.FromContext(ctx)
	if !ok {
		err := ctxerr.Wrap(ctx, fleet.NewAuthRequiredError("internal error: missing host from request context"))
		return fleet.DeviceMigrateMDMResponse{Err: err}, nil
	}

	if err := svc.TriggerMigrateMDMDevice(ctx, host); err != nil {
		return fleet.DeviceMigrateMDMResponse{Err: err}, nil
	}
	return fleet.DeviceMigrateMDMResponse{}, nil
}

func (svc *Service) TriggerMigrateMDMDevice(ctx context.Context, host *fleet.Host) error {
	return fleet.ErrMissingLicense
}

// Trigger linux key escrow
func triggerLinuxDiskEncryptionEscrowEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	host, ok := hostctx.FromContext(ctx)
	if !ok {
		err := ctxerr.Wrap(ctx, fleet.NewAuthRequiredError("internal error: missing host from request context"))
		return fleet.TriggerLinuxDiskEncryptionEscrowResponse{Err: err}, nil
	}

	if err := svc.TriggerLinuxDiskEncryptionEscrow(ctx, host); err != nil {
		return fleet.TriggerLinuxDiskEncryptionEscrowResponse{Err: err}, nil
	}
	return fleet.TriggerLinuxDiskEncryptionEscrowResponse{}, nil
}

func (svc *Service) TriggerLinuxDiskEncryptionEscrow(ctx context.Context, host *fleet.Host) error {
	return fleet.ErrMissingLicense
}

// Get Current Device's Software
func getDeviceSoftwareEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	host, ok := hostctx.FromContext(ctx)
	if !ok {
		err := ctxerr.Wrap(ctx, fleet.NewAuthRequiredError("internal error: missing host from request context"))
		return fleet.GetDeviceSoftwareResponse{Err: err}, nil
	}

	req := request.(*fleet.GetDeviceSoftwareRequest)
	res, meta, err := svc.ListHostSoftware(ctx, host.ID, req.HostSoftwareTitleListOptions)
	for _, s := range res {
		// mutate HostSoftwareWithInstaller records for my device page
		s.ForMyDevicePage(req.Token)
	}

	if err != nil {
		return fleet.GetDeviceSoftwareResponse{Err: err}, nil
	}
	if res == nil {
		res = []*fleet.HostSoftwareWithInstaller{}
	}
	var totalResults int
	if meta != nil {
		totalResults = int(meta.TotalResults) //nolint:gosec // dismiss G115
	}
	return fleet.GetDeviceSoftwareResponse{
		Software: res,
		Meta:     meta,
		Count:    totalResults,
	}, nil
}

// List Current Device's Certificates
func listDeviceCertificatesEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	host, ok := hostctx.FromContext(ctx)
	if !ok {
		err := ctxerr.Wrap(ctx, fleet.NewAuthRequiredError("internal error: missing host from request context"))
		return fleet.ListDevicePoliciesResponse{Err: err}, nil
	}

	req := request.(*fleet.ListDeviceCertificatesRequest)
	res, meta, err := svc.ListHostCertificates(ctx, host.ID, req.ListOptions)
	if err != nil {
		return fleet.ListDeviceCertificatesResponse{Err: err}, nil
	}
	if res == nil {
		res = []*fleet.HostCertificatePayload{}
	}
	return fleet.ListDeviceCertificatesResponse{Certificates: res, Meta: meta, Count: meta.TotalResults}, nil
}

// Get "Setup experience" status.
func getDeviceSetupExperienceStatusEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req, ok := request.(*fleet.GetDeviceSetupExperienceStatusRequest)
	if !ok {
		return nil, fmt.Errorf("internal error: invalid request type: %T", request)
	}
	results, err := svc.GetDeviceSetupExperienceStatus(ctx)
	if err != nil {
		return &fleet.GetDeviceSetupExperienceStatusResponse{Err: err}, nil
	}

	// only software can have custom icons, so no need to iterate over Scripts
	for _, r := range results.Software {
		// mutate SetupExperienceStatusResult records for my device page
		// (same approach used for HostSoftwareWithInstaller)
		r.ForMyDevicePage(req.Token)
	}

	return &fleet.GetDeviceSetupExperienceStatusResponse{Results: results}, nil
}

func (svc *Service) GetDeviceSetupExperienceStatus(ctx context.Context) (*fleet.DeviceSetupExperienceStatusPayload, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}
