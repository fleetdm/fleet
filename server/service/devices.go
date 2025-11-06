package service

import (
	"context"
	"crypto/x509"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
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

/////////////////////////////////////////////////////////////////////////////////
// Ping device endpoint
/////////////////////////////////////////////////////////////////////////////////

type devicePingRequest struct{}

type deviceAuthPingRequest struct {
	Token string `url:"token"`
}

func (r *deviceAuthPingRequest) deviceAuthToken() string {
	return r.Token
}

type devicePingResponse struct{}

func (r devicePingResponse) Error() error { return nil }

func (r devicePingResponse) HijackRender(ctx context.Context, w http.ResponseWriter) {
	writeCapabilitiesHeader(w, fleet.GetServerDeviceCapabilities())
}

// NOTE: we're intentionally not reading the capabilities header in this
// endpoint as is unauthenticated and we don't want to trust whatever comes in
// there.
func devicePingEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	svc.DisableAuthForPing(ctx)
	return devicePingResponse{}, nil
}

func (svc *Service) DisableAuthForPing(ctx context.Context) {
	// skipauth: this endpoint is intentionally public to allow devices to ping
	// the server and among other things, get the fleet.Capabilities header to
	// determine which capabilities are enabled in the server.
	svc.authz.SkipAuthorization(ctx)
}

/////////////////////////////////////////////////////////////////////////////////
// Fleet Desktop endpoints
/////////////////////////////////////////////////////////////////////////////////

type fleetDesktopResponse struct {
	Err error `json:"error,omitempty"`
	fleet.DesktopSummary
}

func (r fleetDesktopResponse) Error() error { return r.Err }

type getFleetDesktopRequest struct {
	Token string `url:"token"`
}

func (r *getFleetDesktopRequest) deviceAuthToken() string {
	return r.Token
}

// getFleetDesktopEndpoint is meant to be the only API endpoint used by Fleet Desktop. This
// endpoint should not include any kind of identifying information about the host.
func getFleetDesktopEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	sum, err := svc.GetFleetDesktopSummary(ctx)
	if err != nil {
		return fleetDesktopResponse{Err: err}, nil
	}
	return fleetDesktopResponse{DesktopSummary: sum}, nil
}

func (svc *Service) GetFleetDesktopSummary(ctx context.Context) (fleet.DesktopSummary, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return fleet.DesktopSummary{}, fleet.ErrMissingLicense
}

/////////////////////////////////////////////////////////////////////////////////
// Get Current Device's Host
/////////////////////////////////////////////////////////////////////////////////

type getDeviceHostRequest struct {
	Token           string `url:"token"`
	ExcludeSoftware bool   `query:"exclude_software,optional"`
}

func (r *getDeviceHostRequest) deviceAuthToken() string {
	return r.Token
}

type getDeviceHostResponse struct {
	Host                      *HostDetailResponse      `json:"host"`
	SelfService               bool                     `json:"self_service"`
	OrgLogoURL                string                   `json:"org_logo_url"`
	OrgLogoURLLightBackground string                   `json:"org_logo_url_light_background"`
	OrgContactURL             string                   `json:"org_contact_url"`
	Err                       error                    `json:"error,omitempty"`
	License                   fleet.LicenseInfo        `json:"license"`
	GlobalConfig              fleet.DeviceGlobalConfig `json:"global_config"`
}

func (r getDeviceHostResponse) Error() error { return r.Err }

func getDeviceHostEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*getDeviceHostRequest)
	host, ok := hostctx.FromContext(ctx)
	if !ok {
		err := ctxerr.Wrap(ctx, fleet.NewAuthRequiredError("internal error: missing host from request context"))
		return getDeviceHostResponse{Err: err}, nil
	}

	// must still load the full host details, as it returns more information
	opts := fleet.HostDetailOptions{
		IncludeCVEScores: false,
		IncludePolicies:  false,
		ExcludeSoftware:  req.ExcludeSoftware,
	}
	hostDetails, err := svc.GetHost(ctx, host.ID, opts)
	if err != nil {
		return getDeviceHostResponse{Err: err}, nil
	}

	resp, err := hostDetailResponseForHost(ctx, svc, hostDetails)
	if err != nil {
		return getDeviceHostResponse{Err: err}, nil
	}

	// the org logo URL config is required by the frontend to render the page;
	// we need to be careful with what we return from AppConfig in the response
	// as this is a weakly authenticated endpoint (with the device auth token).
	ac, err := svc.AppConfigObfuscated(ctx)
	if err != nil {
		return getDeviceHostResponse{Err: err}, nil
	}

	license, err := svc.License(ctx)
	if err != nil {
		return getDeviceHostResponse{Err: err}, nil
	}

	resp.DEPAssignedToFleet = ptr.Bool(false)
	if ac.MDM.EnabledAndConfigured && license.IsPremium() {
		hdep, err := svc.GetHostDEPAssignment(ctx, host)
		if err != nil && !fleet.IsNotFound(err) {
			return getDeviceHostResponse{Err: err}, nil
		}
		resp.DEPAssignedToFleet = ptr.Bool(hdep.IsDEPAssignedToFleet())
	}

	softwareInventoryEnabled := ac.Features.EnableSoftwareInventory
	requireAllSoftware := ac.MDM.MacOSSetup.RequireAllSoftware
	if resp.TeamID != nil {
		// load the team to get the device's team's software inventory config.
		tm, err := svc.GetTeam(ctx, *resp.TeamID)
		if err != nil && !fleet.IsNotFound(err) {
			return getDeviceHostResponse{Err: err}, nil
		}
		if tm != nil {
			softwareInventoryEnabled = tm.Config.Features.EnableSoftwareInventory // TODO: We should look for opportunities to fix the confusing name of the `global_config` object in the API response. Also, how can we better clarify/document the expected order of precedence for team and global feature flags?
			requireAllSoftware = tm.Config.MDM.MacOSSetup.RequireAllSoftware
		}
	}

	hasSelfService := false
	if softwareInventoryEnabled {
		hasSelfService, err = svc.HasSelfServiceSoftwareInstallers(ctx, host)
		if err != nil {
			return getDeviceHostResponse{Err: err}, nil
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
			EnableSoftwareInventory: softwareInventoryEnabled,
		},
	}

	return getDeviceHostResponse{
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
	alreadyAuthd := svc.authz.IsAuthenticatedWith(ctx, authz.AuthnDeviceToken)
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

	return host, svc.debugEnabledForHost(ctx, host.ID), nil
}

/////////////////////////////////////////////////////////////////////////////////
// Refetch Current Device's Host
/////////////////////////////////////////////////////////////////////////////////

type refetchDeviceHostRequest struct {
	Token string `url:"token"`
}

func (r *refetchDeviceHostRequest) deviceAuthToken() string {
	return r.Token
}

func refetchDeviceHostEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	host, ok := hostctx.FromContext(ctx)
	if !ok {
		err := ctxerr.Wrap(ctx, fleet.NewAuthRequiredError("internal error: missing host from request context"))
		return refetchHostResponse{Err: err}, nil
	}

	err := svc.RefetchHost(ctx, host.ID)
	if err != nil {
		return refetchHostResponse{Err: err}, nil
	}
	return refetchHostResponse{}, nil
}

////////////////////////////////////////////////////////////////////////////////
// List Current Device's Host Device Mappings
////////////////////////////////////////////////////////////////////////////////

type listDeviceHostDeviceMappingRequest struct {
	Token string `url:"token"`
}

func (r *listDeviceHostDeviceMappingRequest) deviceAuthToken() string {
	return r.Token
}

func listDeviceHostDeviceMappingEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	host, ok := hostctx.FromContext(ctx)
	if !ok {
		err := ctxerr.Wrap(ctx, fleet.NewAuthRequiredError("internal error: missing host from request context"))
		return listHostDeviceMappingResponse{Err: err}, nil
	}

	dms, err := svc.ListHostDeviceMapping(ctx, host.ID)
	if err != nil {
		return listHostDeviceMappingResponse{Err: err}, nil
	}
	return listHostDeviceMappingResponse{HostID: host.ID, DeviceMapping: dms}, nil
}

////////////////////////////////////////////////////////////////////////////////
// Get Current Device's Macadmins
////////////////////////////////////////////////////////////////////////////////

type getDeviceMacadminsDataRequest struct {
	Token string `url:"token"`
}

func (r *getDeviceMacadminsDataRequest) deviceAuthToken() string {
	return r.Token
}

func getDeviceMacadminsDataEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	host, ok := hostctx.FromContext(ctx)
	if !ok {
		err := ctxerr.Wrap(ctx, fleet.NewAuthRequiredError("internal error: missing host from request context"))
		return getMacadminsDataResponse{Err: err}, nil
	}

	data, err := svc.MacadminsData(ctx, host.ID)
	if err != nil {
		return getMacadminsDataResponse{Err: err}, nil
	}
	return getMacadminsDataResponse{Macadmins: data}, nil
}

////////////////////////////////////////////////////////////////////////////////
// List Current Device's Policies
////////////////////////////////////////////////////////////////////////////////

type listDevicePoliciesRequest struct {
	Token string `url:"token"`
}

func (r *listDevicePoliciesRequest) deviceAuthToken() string {
	return r.Token
}

type listDevicePoliciesResponse struct {
	Err      error               `json:"error,omitempty"`
	Policies []*fleet.HostPolicy `json:"policies"`
}

func (r listDevicePoliciesResponse) Error() error { return r.Err }

func listDevicePoliciesEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	host, ok := hostctx.FromContext(ctx)
	if !ok {
		err := ctxerr.Wrap(ctx, fleet.NewAuthRequiredError("internal error: missing host from request context"))
		return listDevicePoliciesResponse{Err: err}, nil
	}

	data, err := svc.ListDevicePolicies(ctx, host)
	if err != nil {
		return listDevicePoliciesResponse{Err: err}, nil
	}

	return listDevicePoliciesResponse{Policies: data}, nil
}

func (svc *Service) ListDevicePolicies(ctx context.Context, host *fleet.Host) ([]*fleet.HostPolicy, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}

////////////////////////////////////////////////////////////////////////////////
// Resend configuration profile
////////////////////////////////////////////////////////////////////////////////

type resendDeviceConfigurationProfileRequest struct {
	Token       string `url:"token"`
	ProfileUUID string `url:"profile_uuid"`
}

func (r *resendDeviceConfigurationProfileRequest) deviceAuthToken() string {
	return r.Token
}

type resendDeviceConfigurationProfileResponse struct {
	Err error `json:"error,omitempty"`
}

func (r resendDeviceConfigurationProfileResponse) Error() error { return r.Err }

func (r resendDeviceConfigurationProfileResponse) Status() int { return http.StatusAccepted }

func resendDeviceConfigurationProfileEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	host, ok := hostctx.FromContext(ctx)
	if !ok {
		err := ctxerr.Wrap(ctx, fleet.NewAuthRequiredError("internal error: missing host from request context"))
		return resendDeviceConfigurationProfileResponse{Err: err}, nil
	}

	req := request.(*resendDeviceConfigurationProfileRequest)
	err := svc.ResendDeviceHostMDMProfile(ctx, host, req.ProfileUUID)
	if err != nil {
		return resendDeviceConfigurationProfileResponse{
			Err: err,
		}, nil
	}

	return resendDeviceConfigurationProfileResponse{}, nil
}

////////////////////////////////////////////////////////////////////////////////
// Get software MDM command results
////////////////////////////////////////////////////////////////////////////////

type getDeviceMDMCommandResultsRequest struct {
	Token       string `url:"token"`
	CommandUUID string `url:"command_uuid"`
}

func (r *getDeviceMDMCommandResultsRequest) deviceAuthToken() string {
	return r.Token
}

func getDeviceMDMCommandResultsEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	_, ok := hostctx.FromContext(ctx)
	if !ok {
		err := ctxerr.Wrap(ctx, fleet.NewAuthRequiredError("internal error: missing host from request context"))
		return getMDMCommandResultsResponse{Err: err}, nil
	}

	req := request.(*getDeviceMDMCommandResultsRequest)
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

////////////////////////////////////////////////////////////////////////////////
// Transparency URL Redirect
////////////////////////////////////////////////////////////////////////////////

type transparencyURLRequest struct {
	Token string `url:"token"`
}

func (r *transparencyURLRequest) deviceAuthToken() string {
	return r.Token
}

type transparencyURLResponse struct {
	RedirectURL string `json:"-"` // used to control the redirect, see HijackRender method
	Err         error  `json:"error,omitempty"`
}

func (r transparencyURLResponse) HijackRender(ctx context.Context, w http.ResponseWriter) {
	w.Header().Set("Location", r.RedirectURL)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

func (r transparencyURLResponse) Error() error { return r.Err }

func transparencyURL(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	transparencyURL, err := svc.GetTransparencyURL(ctx)

	return transparencyURLResponse{RedirectURL: transparencyURL, Err: err}, nil
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
type getDeviceSoftwareIconRequest struct {
	Token           string `url:"token"`
	SoftwareTitleID uint   `url:"software_title_id"`
}

func (r *getDeviceSoftwareIconRequest) deviceAuthToken() string {
	return r.Token
}

type getDeviceSoftwareIconResponse struct {
	Err         error  `json:"error,omitempty"`
	ImageData   []byte `json:"-"`
	ContentType string `json:"-"`
	Filename    string `json:"-"`
	Size        int64  `json:"-"`
}

func (r getDeviceSoftwareIconResponse) Error() error { return r.Err }

type getDeviceSoftwareIconRedirectResponse struct {
	Err         error  `json:"error,omitempty"`
	RedirectURL string `json:"-"`
}

func (r getDeviceSoftwareIconRedirectResponse) Error() error { return r.Err }

func (r getDeviceSoftwareIconRedirectResponse) HijackRender(ctx context.Context, w http.ResponseWriter) {
	if r.Err != nil {
		return
	}

	w.Header().Set("Location", r.RedirectURL)
	w.WriteHeader(http.StatusFound)
}

func (r getDeviceSoftwareIconResponse) HijackRender(ctx context.Context, w http.ResponseWriter) {
	w.Header().Set("Content-Type", r.ContentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, r.Filename))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", r.Size))

	_, _ = w.Write(r.ImageData)
}

func getDeviceSoftwareIconEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	host, ok := hostctx.FromContext(ctx)
	if !ok {
		err := ctxerr.Wrap(ctx, fleet.NewAuthRequiredError("internal error: missing host from request context"))
		return getDeviceSoftwareIconResponse{Err: err}, nil
	}

	req := request.(*getDeviceSoftwareIconRequest)
	var teamID uint
	if host.TeamID != nil {
		teamID = *host.TeamID
	}
	iconData, size, filename, err := svc.GetDeviceSoftwareIconsTitleIcon(ctx, teamID, req.SoftwareTitleID)
	if err != nil {
		var vppErr *fleet.VPPIconAvailable
		if errors.As(err, &vppErr) {
			// 302 redirect to vpp app IconURL
			return getDeviceSoftwareIconRedirectResponse{RedirectURL: vppErr.IconURL}, nil
		}
		return getDeviceSoftwareIconResponse{Err: err}, nil
	}

	return getDeviceSoftwareIconResponse{
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

////////////////////////////////////////////////////////////////////////////////
// Receive errors from the client
////////////////////////////////////////////////////////////////////////////////

type fleetdErrorRequest struct {
	Token string `url:"token"`
	fleet.FleetdError
}

func (f *fleetdErrorRequest) deviceAuthToken() string {
	return f.Token
}

// Since we're directly storing what we get in Redis, limit the request size to
// 5MB, this combined with the rate limit of this endpoint should be enough to
// prevent a malicious actor.
const maxFleetdErrorReportSize int64 = 5 * 1024 * 1024

func (f *fleetdErrorRequest) DecodeBody(ctx context.Context, r io.Reader, u url.Values, c []*x509.Certificate) error {
	limitedReader := io.LimitReader(r, maxFleetdErrorReportSize+1)
	decoder := json.NewDecoder(limitedReader)

	for {
		if err := decoder.Decode(&f.FleetdError); err == io.EOF {
			break
		} else if err == io.ErrUnexpectedEOF {
			return &fleet.BadRequestError{Message: "payload exceeds maximum accepted size"}
		} else if err != nil {
			return &fleet.BadRequestError{Message: "invalid payload"}
		}
	}

	return nil
}

type fleetdErrorResponse struct{}

func (r fleetdErrorResponse) Error() error { return nil }

func fleetdError(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	req := request.(*fleetdErrorRequest)
	err := svc.LogFleetdError(ctx, req.FleetdError)
	if err != nil {
		return nil, err
	}
	return fleetdErrorResponse{}, nil
}

func (svc *Service) LogFleetdError(ctx context.Context, fleetdError fleet.FleetdError) error {
	if !svc.authz.IsAuthenticatedWith(ctx, authz.AuthnDeviceToken) {
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

////////////////////////////////////////////////////////////////////////////////
// Get Current Device's MDM Apple Enrollment Profile
////////////////////////////////////////////////////////////////////////////////

type getDeviceMDMManualEnrollProfileRequest struct {
	Token string `url:"token"`
}

func (r *getDeviceMDMManualEnrollProfileRequest) deviceAuthToken() string {
	return r.Token
}

type getDeviceMDMManualEnrollProfileResponse struct {
	// EnrollURL field is used in HijackRender for the response.
	EnrollURL string `json:"enroll_url,omitempty"`

	Err error `json:"error,omitempty"`
}

func (r getDeviceMDMManualEnrollProfileResponse) Error() error { return r.Err }

func getDeviceMDMManualEnrollProfileEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	// this call ensures that the authentication was done, no need to actually
	// use the host
	if _, ok := hostctx.FromContext(ctx); !ok {
		err := ctxerr.Wrap(ctx, fleet.NewAuthRequiredError("internal error: missing host from request context"))
		return getDeviceMDMManualEnrollProfileResponse{Err: err}, nil
	}

	enrollURL, err := svc.GetDeviceMDMAppleEnrollmentProfile(ctx)
	if err != nil {
		return getDeviceMDMManualEnrollProfileResponse{Err: err}, nil
	}
	return getDeviceMDMManualEnrollProfileResponse{EnrollURL: enrollURL.String()}, nil
}

func (svc *Service) GetDeviceMDMAppleEnrollmentProfile(ctx context.Context) (*url.URL, error) {
	// must be device-authenticated, no additional authorization is required
	if !svc.authz.IsAuthenticatedWith(ctx, authz.AuthnDeviceToken) {
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

////////////////////////////////////////////////////////////////////////////////
// Signal start of mdm migration on a device
////////////////////////////////////////////////////////////////////////////////

type deviceMigrateMDMRequest struct {
	Token string `url:"token"`
}

func (r *deviceMigrateMDMRequest) deviceAuthToken() string {
	return r.Token
}

type deviceMigrateMDMResponse struct {
	Err error `json:"error,omitempty"`
}

func (r deviceMigrateMDMResponse) Error() error { return r.Err }

func (r deviceMigrateMDMResponse) Status() int { return http.StatusNoContent }

func migrateMDMDeviceEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	host, ok := hostctx.FromContext(ctx)
	if !ok {
		err := ctxerr.Wrap(ctx, fleet.NewAuthRequiredError("internal error: missing host from request context"))
		return deviceMigrateMDMResponse{Err: err}, nil
	}

	if err := svc.TriggerMigrateMDMDevice(ctx, host); err != nil {
		return deviceMigrateMDMResponse{Err: err}, nil
	}
	return deviceMigrateMDMResponse{}, nil
}

func (svc *Service) TriggerMigrateMDMDevice(ctx context.Context, host *fleet.Host) error {
	return fleet.ErrMissingLicense
}

////////////////////////////////////////////////////////////////////////////////
// Trigger linux key escrow
////////////////////////////////////////////////////////////////////////////////

type triggerLinuxDiskEncryptionEscrowRequest struct {
	Token string `url:"token"`
}

func (r *triggerLinuxDiskEncryptionEscrowRequest) deviceAuthToken() string {
	return r.Token
}

type triggerLinuxDiskEncryptionEscrowResponse struct {
	Err error `json:"error,omitempty"`
}

func (r triggerLinuxDiskEncryptionEscrowResponse) Error() error { return r.Err }

func (r triggerLinuxDiskEncryptionEscrowResponse) Status() int { return http.StatusNoContent }

func triggerLinuxDiskEncryptionEscrowEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	host, ok := hostctx.FromContext(ctx)
	if !ok {
		err := ctxerr.Wrap(ctx, fleet.NewAuthRequiredError("internal error: missing host from request context"))
		return triggerLinuxDiskEncryptionEscrowResponse{Err: err}, nil
	}

	if err := svc.TriggerLinuxDiskEncryptionEscrow(ctx, host); err != nil {
		return triggerLinuxDiskEncryptionEscrowResponse{Err: err}, nil
	}
	return triggerLinuxDiskEncryptionEscrowResponse{}, nil
}

func (svc *Service) TriggerLinuxDiskEncryptionEscrow(ctx context.Context, host *fleet.Host) error {
	return fleet.ErrMissingLicense
}

////////////////////////////////////////////////////////////////////////////////
// Get Current Device's Software
////////////////////////////////////////////////////////////////////////////////

type getDeviceSoftwareRequest struct {
	Token string `url:"token"`
	fleet.HostSoftwareTitleListOptions
}

func (r *getDeviceSoftwareRequest) deviceAuthToken() string {
	return r.Token
}

type getDeviceSoftwareResponse struct {
	Software []*fleet.HostSoftwareWithInstaller `json:"software"`
	Count    int                                `json:"count"`
	Meta     *fleet.PaginationMetadata          `json:"meta,omitempty"`
	Err      error                              `json:"error,omitempty"`
}

func (r getDeviceSoftwareResponse) Error() error { return r.Err }

func getDeviceSoftwareEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	host, ok := hostctx.FromContext(ctx)
	if !ok {
		err := ctxerr.Wrap(ctx, fleet.NewAuthRequiredError("internal error: missing host from request context"))
		return getDeviceSoftwareResponse{Err: err}, nil
	}

	req := request.(*getDeviceSoftwareRequest)
	res, meta, err := svc.ListHostSoftware(ctx, host.ID, req.HostSoftwareTitleListOptions)
	for _, s := range res {
		// mutate HostSoftwareWithInstaller records for my device page
		s.ForMyDevicePage(req.Token)
	}

	if err != nil {
		return getDeviceSoftwareResponse{Err: err}, nil
	}
	if res == nil {
		res = []*fleet.HostSoftwareWithInstaller{}
	}
	return getDeviceSoftwareResponse{Software: res, Meta: meta, Count: int(meta.TotalResults)}, nil //nolint:gosec // dismiss G115
}

////////////////////////////////////////////////////////////////////////////////
// List Current Device's Certificates
////////////////////////////////////////////////////////////////////////////////

type listDeviceCertificatesRequest struct {
	Token string `url:"token"`
	fleet.ListOptions
}

func (r *listDeviceCertificatesRequest) ValidateRequest() error {
	if r.ListOptions.OrderKey != "" && !listHostCertificatesSortCols[r.ListOptions.OrderKey] {
		return badRequest("invalid order key")
	}
	return nil
}

func (r *listDeviceCertificatesRequest) deviceAuthToken() string {
	return r.Token
}

type listDeviceCertificatesResponse struct {
	Certificates []*fleet.HostCertificatePayload `json:"certificates"`
	Meta         *fleet.PaginationMetadata       `json:"meta,omitempty"`
	Count        uint                            `json:"count"`
	Err          error                           `json:"error,omitempty"`
}

func (r listDeviceCertificatesResponse) Error() error { return r.Err }

func listDeviceCertificatesEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	host, ok := hostctx.FromContext(ctx)
	if !ok {
		err := ctxerr.Wrap(ctx, fleet.NewAuthRequiredError("internal error: missing host from request context"))
		return listDevicePoliciesResponse{Err: err}, nil
	}

	req := request.(*listDeviceCertificatesRequest)
	res, meta, err := svc.ListHostCertificates(ctx, host.ID, req.ListOptions)
	if err != nil {
		return listDeviceCertificatesResponse{Err: err}, nil
	}
	if res == nil {
		res = []*fleet.HostCertificatePayload{}
	}
	return listDeviceCertificatesResponse{Certificates: res, Meta: meta, Count: meta.TotalResults}, nil
}

/////////////////////////////////////////////////////////////////////////////////
// Get "Setup experience" status.
/////////////////////////////////////////////////////////////////////////////////

type getDeviceSetupExperienceStatusRequest struct {
	Token string `url:"token"`
}

func (r *getDeviceSetupExperienceStatusRequest) deviceAuthToken() string {
	return r.Token
}

type getDeviceSetupExperienceStatusResponse struct {
	Results *fleet.DeviceSetupExperienceStatusPayload `json:"setup_experience_results,omitempty"`
	Err     error                                     `json:"error,omitempty"`
}

func (r getDeviceSetupExperienceStatusResponse) Error() error { return r.Err }

func getDeviceSetupExperienceStatusEndpoint(ctx context.Context, request interface{}, svc fleet.Service) (fleet.Errorer, error) {
	if _, ok := request.(*getDeviceSetupExperienceStatusRequest); !ok {
		return nil, fmt.Errorf("internal error: invalid request type: %T", request)
	}
	results, err := svc.GetDeviceSetupExperienceStatus(ctx)
	if err != nil {
		return &getDeviceSetupExperienceStatusResponse{Err: err}, nil
	}
	return &getDeviceSetupExperienceStatusResponse{Results: results}, nil
}

func (svc *Service) GetDeviceSetupExperienceStatus(ctx context.Context) (*fleet.DeviceSetupExperienceStatusPayload, error) {
	// skipauth: No authorization check needed due to implementation returning
	// only license error.
	svc.authz.SkipAuthorization(ctx)

	return nil, fleet.ErrMissingLicense
}
