package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server"
	authz_ctx "github.com/fleetdm/fleet/v4/server/contexts/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	hostctx "github.com/fleetdm/fleet/v4/server/contexts/host"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/sso"
)

func (svc *Service) ListDevicePolicies(ctx context.Context, host *fleet.Host) ([]*fleet.DevicePolicy, error) {
	policies, err := svc.ds.ListPoliciesForHost(ctx, host)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list policies for host")
	}
	// return the device-safe representation of the policies, which excludes
	// the policy author's identity and the raw SQL query.
	return fleet.HostPoliciesToDevicePolicies(policies), nil
}

// TriggerMigrateMDMDevice triggers the webhook associated with the MDM
// migration to Fleet configuration. It is located in the ee package instead of
// the server/webhooks one because it is a Fleet Premium only feature and for
// licensing reasons this needs to live under this package.
func (svc *Service) TriggerMigrateMDMDevice(ctx context.Context, host *fleet.Host) error {
	svc.logger.DebugContext(ctx, "trigger migration webhook", "host_id", host.ID,
		"refetch_critical_queries_until", host.RefetchCriticalQueriesUntil)

	ac, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return err
	}
	if !ac.MDM.EnabledAndConfigured {
		return fleet.ErrMDMNotConfigured
	}

	if host.RefetchCriticalQueriesUntil != nil && host.RefetchCriticalQueriesUntil.After(svc.clock.Now()) {
		// the webhook has already been triggered successfully recently (within the
		// refetch critical queries delay), so return as if it did send it successfully
		// but do not re-send.
		svc.logger.DebugContext(ctx, "waiting for critical queries refetch, skip sending webhook",
			"host_id", host.ID)
		return nil
	}

	connected, err := svc.ds.IsHostConnectedToFleetMDM(ctx, host)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "checking if host is connected to Fleet")
	}

	var bre fleet.BadRequestError
	switch {
	case !ac.MDM.MacOSMigration.Enable:
		bre.InternalErr = ctxerr.New(ctx, "macOS migration not enabled")
	case ac.MDM.MacOSMigration.WebhookURL == "":
		bre.InternalErr = ctxerr.New(ctx, "macOS migration webhook URL not configured")
	}

	mdmInfo, err := svc.ds.GetHostMDM(ctx, host.ID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "fetching host mdm info")
	}

	manualMigrationEligible, err := fleet.IsEligibleForManualMigration(host, mdmInfo, connected)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "checking manual migration eligibility")
	}

	if !fleet.IsEligibleForDEPMigration(host, mdmInfo, connected) && !manualMigrationEligible {
		bre.InternalErr = ctxerr.New(ctx, "host not eligible for macOS migration")
	}

	if bre.InternalErr != nil {
		return &bre
	}

	p := fleet.MigrateMDMDeviceWebhookPayload{}
	p.Timestamp = time.Now().UTC()
	p.Host.ID = host.ID
	p.Host.UUID = host.UUID
	p.Host.HardwareSerial = host.HardwareSerial

	if err := server.PostJSONWithTimeout(ctx, ac.MDM.MacOSMigration.WebhookURL, p, svc.logger); err != nil {
		return ctxerr.Wrap(ctx, err, "posting macOS migration webhook")
	}

	// if the webhook was successfully triggered, we update the host to
	// constantly run the query to check if it has been unenrolled from its
	// existing third-party MDM.
	refetchUntil := svc.clock.Now().Add(fleet.RefetchMDMUnenrollCriticalQueryDuration)
	host.RefetchCriticalQueriesUntil = &refetchUntil
	if err := svc.ds.UpdateHostRefetchCriticalQueriesUntil(ctx, host.ID, &refetchUntil); err != nil {
		return ctxerr.Wrap(ctx, err, "save host with refetch critical queries timestamp")
	}

	return nil
}

func (svc *Service) BypassConditionalAccess(ctx context.Context, host *fleet.Host) error {
	// this is not a user-authenticated endpoint
	svc.authz.SkipAuthorization(ctx)

	// iOS/iPadOS devices authenticate by UUID in the URL and don't participate in
	// conditional access policies, so they can't bypass it.
	if svc.authz.IsAuthenticatedWith(ctx, authz_ctx.AuthnDeviceURL) {
		return fleet.NewUserMessageError(errors.New("conditional access bypass is not supported on this device"), http.StatusForbidden)
	}

	ac, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting device config")
	}

	if ac.ConditionalAccess != nil && !ac.ConditionalAccess.BypassEnabled() {
		return fleet.NewUserMessageError(errors.New("conditional access bypass disabled"), http.StatusForbidden)
	}

	if err := svc.ds.ConditionalAccessBypassDevice(ctx, host.ID); err != nil {
		return ctxerr.Wrap(ctx, err, "setting conditional access bypass")
	}

	idpFullName, err := fleet.GetEndUserIdpFullName(ctx, svc.ds, host.ID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting end users for bypass activity")
	}

	if idpFullName == "" {
		idpFullName = "An end user"
	}

	if err := svc.NewActivity(ctx, nil, fleet.ActivityTypeHostBypassedConditionalAccess{
		HostID:          host.ID,
		HostDisplayName: host.DisplayName(),
		IdPFullName:     idpFullName,
	}); err != nil {
		return ctxerr.Wrap(ctx, err, "creating host bypass activity")
	}

	return nil
}

func (svc *Service) GetFleetDesktopSummary(ctx context.Context) (fleet.DesktopSummary, error) {
	// this is not a user-authenticated endpoint
	svc.authz.SkipAuthorization(ctx)

	var sum fleet.DesktopSummary

	host, ok := hostctx.FromContext(ctx)

	if !ok {
		err := ctxerr.Wrap(ctx, fleet.NewAuthRequiredError("internal error: missing host from request context"))
		return sum, err
	}

	hasSelfService, err := svc.ds.HasSelfServiceSoftwareInstallers(ctx, host.Platform, host.TeamID)
	if err != nil {
		return sum, ctxerr.Wrap(ctx, err, "retrieving self service software installers")
	}
	sum.SelfService = &hasSelfService

	r, err := svc.ds.FailingPoliciesCount(ctx, host)
	if err != nil {
		return sum, ctxerr.Wrap(ctx, err, "retrieving failing policies")
	}
	sum.FailingPolicies = &r

	appCfg, err := svc.AppConfigObfuscated(ctx)
	if err != nil {
		return sum, ctxerr.Wrap(ctx, err, "retrieving app config")
	}

	if appCfg.MDM.EnabledAndConfigured && appCfg.MDM.MacOSMigration.Enable {
		connected, err := svc.ds.IsHostConnectedToFleetMDM(ctx, host)
		if err != nil {
			return sum, ctxerr.Wrap(ctx, err, "checking if host is connected to Fleet")
		}

		mdmInfo, err := svc.ds.GetHostMDM(ctx, host.ID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return sum, ctxerr.Wrap(ctx, err, "could not retrieve mdm info")
		}

		needsDEPEnrollment := mdmInfo != nil && !mdmInfo.Enrolled && host.IsDEPAssignedToFleet()

		if needsDEPEnrollment {
			sum.Notifications.RenewEnrollmentProfile = true
		}

		manualMigrationEligible, err := fleet.IsEligibleForManualMigration(host, mdmInfo, connected)
		if err != nil {
			return sum, ctxerr.Wrap(ctx, err, "checking manual migration eligibility")
		}

		if fleet.IsEligibleForDEPMigration(host, mdmInfo, connected) || manualMigrationEligible {
			sum.Notifications.NeedsMDMMigration = true
		}

	}

	// organization information
	sum.Config.OrgInfo.OrgName = appCfg.OrgInfo.OrgName
	sum.Config.OrgInfo.OrgLogoURL = appCfg.OrgInfo.OrgLogoURL
	sum.Config.OrgInfo.OrgLogoURLLightBackground = appCfg.OrgInfo.OrgLogoURLLightBackground
	sum.Config.OrgInfo.OrgLogoURLDarkMode = appCfg.OrgInfo.OrgLogoURLDarkMode
	sum.Config.OrgInfo.OrgLogoURLLightMode = appCfg.OrgInfo.OrgLogoURLLightMode
	sum.Config.OrgInfo.ContactURL = appCfg.OrgInfo.ContactURL

	// mdm information
	sum.Config.MDM.MacOSMigration.Mode = appCfg.MDM.MacOSMigration.Mode

	sum.AlternativeBrowserHost = appCfg.FleetDesktop.AlternativeBrowserHost

	return sum, nil
}

func (svc *Service) TriggerLinuxDiskEncryptionEscrow(ctx context.Context, host *fleet.Host) error {
	if svc.ds.IsHostPendingEscrow(ctx, host.ID) {
		return nil
	}

	if err := svc.ds.AssertHasNoEncryptionKeyStored(ctx, host.ID); err != nil {
		return err
	}

	if err := svc.validateReadyForLinuxEscrow(ctx, host); err != nil {
		_ = svc.ds.ReportEscrowError(ctx, host.ID, err.Error())
		return err
	}

	return svc.ds.QueueEscrow(ctx, host.ID)
}

func (svc *Service) validateReadyForLinuxEscrow(ctx context.Context, host *fleet.Host) error {
	if !host.IsLUKSSupported() {
		return &fleet.BadRequestError{Message: "Fleet does not yet support creating LUKS disk encryption keys on this platform."}
	}

	ac, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return err
	}

	if host.TeamID == nil {
		if !ac.MDM.LinuxSettings.EnableEscrowDiskEncryptionKey.Value {
			return &fleet.BadRequestError{Message: "Disk encryption is not enabled for hosts not assigned to a fleet."}
		}
	} else {
		tc, err := svc.ds.TeamMDMConfig(ctx, *host.TeamID)
		if err != nil {
			return err
		}
		if !tc.DiskEncryptionConfig().LinuxEscrowEnabled {
			return &fleet.BadRequestError{Message: "Disk encryption is not enabled for this host's fleet."}
		}
	}

	if host.DiskEncryptionEnabled == nil || !*host.DiskEncryptionEnabled {
		return &fleet.BadRequestError{Message: "Host's disk is not encrypted. Please encrypt your disk first."}
	}

	// We have to pull Orbit info because the auth context doesn't fill in host.OrbitVersion
	orbitInfo, err := svc.ds.GetHostOrbitInfo(ctx, host.ID)
	if err != nil {
		return err
	}

	if orbitInfo == nil || !fleet.IsAtLeastVersion(orbitInfo.Version, fleet.MinOrbitLUKSVersion) {
		return &fleet.BadRequestError{Message: "Your version of fleetd does not support creating disk encryption keys on Linux. Please upgrade fleetd, then click Refetch, then try again."}
	}

	return nil
}

func (svc *Service) GetDeviceSoftwareIconsTitleIcon(ctx context.Context, teamID uint, titleID uint) ([]byte, int64, string, error) {
	// can't call the already made GetSoftwareTitleIcon(ctx, teamID, titleID) method
	// because svc is the concrete open source service implementation despite it being in the ee/directory
	var err error

	icon, err := svc.ds.GetSoftwareTitleIcon(ctx, teamID, titleID)
	if err != nil && !fleet.IsNotFound(err) {
		return nil, 0, "", ctxerr.Wrap(ctx, err, "getting software title icon")
	}
	if icon == nil {
		vppApp, err := svc.ds.GetVPPAppMetadataByTeamAndTitleID(ctx, &teamID, titleID)
		if vppApp != nil && vppApp.IconURL != nil {
			return nil, 0, "", &fleet.VPPIconAvailable{IconURL: *vppApp.IconURL}
		}

		return nil, 0, "", ctxerr.Wrap(ctx, err, "getting software title icon")
	}

	iconData, size, err := svc.softwareTitleIconStore.Get(ctx, icon.StorageID)
	if err != nil {
		return nil, 0, "", ctxerr.Wrap(ctx, err, "getting software title icon data")
	}
	defer iconData.Close()
	imageBytes, err := io.ReadAll(iconData)
	if err != nil {
		return nil, 0, "", ctxerr.Wrap(ctx, err, "reading icon data")
	}

	return imageBytes, size, icon.Filename, nil
}

func (svc *Service) GetDeviceSetupExperienceStatus(ctx context.Context) (*fleet.DeviceSetupExperienceStatusPayload, error) {
	// This is a device endpoint, not a user-authenticated endpoint.
	svc.authz.SkipAuthorization(ctx)

	host, ok := hostctx.FromContext(ctx)
	if !ok {
		return nil, ctxerr.New(ctx, "internal error: missing host from request context")
	}

	return svc.getHostSetupExperienceStatus(ctx, host)
}

func (svc *Service) getHostSetupExperienceStatus(ctx context.Context, host *fleet.Host) (*fleet.DeviceSetupExperienceStatusPayload, error) {
	hostUUID, err := fleet.HostUUIDForSetupExperience(host)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "failed to get host's UUID for the setup experience")
	}

	// Get current status of the setup experience.
	results, err := svc.ds.ListSetupExperienceResultsByHostUUID(ctx, hostUUID, ptr.ValOrZero(host.TeamID))
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "listing setup experience results")
	}

	// Add activities for canceled installs + setup experience run
	err = svc.recordCanceledSetupExperienceSoftwareActivities(ctx, host.ID, hostUUID, host.DisplayName(), results)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "recording cancelled setup experience installs")
	}

	var software []*fleet.SetupExperienceStatusResult
	var scripts []*fleet.SetupExperienceStatusResult
	for _, result := range results {
		if result.IsForSoftware() {
			software = append(software, result)
		}
		if result.IsForScript() {
			scripts = append(scripts, result)
		}
	}

	// Continue with next step in setup experience.
	if _, err = svc.SetupExperienceNextStep(ctx, host); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting next step for host setup experience")
	}

	return &fleet.DeviceSetupExperienceStatusPayload{
		Software: software,
		Scripts:  scripts,
	}, nil
}

/////////////////////////////////////////////////////////////////////////////////
// My Device SSO Flow
/////////////////////////////////////////////////////////////////////////////////

const deviceSSOSessionKeyPrefix = "device_sso_session:"
const deviceSSOSessionIDLength = 24

// createDeviceSSOSession mints a new device SSO session for host.
func (svc *Service) createDeviceSSOSession(ctx context.Context, host *fleet.Host, idpAccountUUID string) (sessionID string, ttl time.Duration, err error) {
	sessionID, err = server.GenerateRandomURLSafeText(deviceSSOSessionIDLength)
	if err != nil {
		return "", 0, ctxerr.Wrap(ctx, err, "generate device sso session id")
	}

	ttl = svc.config.Session.Duration
	session := fleet.DeviceSSOSession{
		HostID:         host.ID,
		IdPAccountUUID: idpAccountUUID,
		ExpiresAt:      svc.clock.Now().Add(ttl),
	}
	b, err := json.Marshal(session)
	if err != nil {
		return "", 0, ctxerr.Wrap(ctx, err, "marshal device sso session")
	}

	if err := svc.keyValueStore.Set(ctx, deviceSSOSessionKeyPrefix+sessionID, string(b), ttl); err != nil {
		return "", 0, ctxerr.Wrap(ctx, err, "store device sso session")
	}

	return sessionID, ttl, nil
}

func (svc *Service) validateDeviceSSOSession(ctx context.Context, sessionID string) (*fleet.DeviceSSOSession, error) {
	if sessionID == "" {
		return nil, ctxerr.Wrap(ctx, fleet.NewAuthRequiredError("device sso session not found"))
	}

	val, err := svc.keyValueStore.Get(ctx, deviceSSOSessionKeyPrefix+sessionID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get device sso session")
	}
	if val == nil {
		return nil, ctxerr.Wrap(ctx, fleet.NewAuthRequiredError("device sso session not found"))
	}

	var session fleet.DeviceSSOSession
	if err := json.Unmarshal([]byte(*val), &session); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "unmarshal device sso session")
	}

	if !svc.clock.Now().Before(session.ExpiresAt) {
		return nil, ctxerr.Wrap(ctx, fleet.NewAuthRequiredError("device sso session expired"))
	}

	return &session, nil
}

func (svc *Service) RequireDeviceSSOSession(ctx context.Context, host *fleet.Host, sessionID string) error {
	appConfig, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting app config")
	}
	if !appConfig.FleetDesktop.SSOEnabled {
		return nil
	}

	// Setup Experience opens the device page in a web view while Setup Assistant
	// is still running, where the user won't be able to initiate the SSO flow.
	inSetupExperience, err := fleet.HostIsInSetupExperience(ctx, svc.ds, host)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "checking if host is in setup experience")
	}
	if inSetupExperience {
		return nil
	}

	session, err := svc.validateDeviceSSOSession(ctx, sessionID)
	var noSession *fleet.AuthRequiredError
	switch {
	case errors.As(err, &noSession):
		svc.logger.DebugContext(ctx, "no device sso session for request", "host_id", host.ID, "err", err)
	case err != nil:
		// Anything else is the session store failing, not the end user being
		// signed out.
		return ctxerr.Wrap(ctx, err, "validating device sso session")
	case session.HostID != host.ID:
		// The session is bound to the host its device token minted it for, so one
		// device's cookie cannot unlock another device's page in the same browser.
		svc.logger.WarnContext(ctx, "device sso session belongs to another host",
			"host_id", host.ID, "session_host_id", session.HostID)
	default:
		return nil
	}

	return ctxerr.Wrap(ctx, fleet.NewDeviceSSORequiredError("no device sso session"), "require device sso session")
}

func (svc *Service) InitiateDeviceSSO(ctx context.Context, deviceURL string) (*fleet.DeviceSSOInitiation, error) {
	// the device middleware already authenticated the host via its device token.
	svc.authz.SkipAuthorization(ctx)

	host, ok := hostctx.FromContext(ctx)
	if !ok {
		return nil, ctxerr.Wrap(ctx, fleet.NewAuthRequiredError("internal error: missing host from request context"))
	}

	appConfig, err := svc.ds.AppConfig(ctx)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting app config")
	}

	if !appConfig.FleetDesktop.SSOEnabled {
		err := &fleet.BadRequestError{Message: "Single sign-on for Fleet Desktop is not enabled."}
		return nil, ctxerr.Wrap(ctx, err, "initiate device sso")
	}

	mdmSSOSettings := appConfig.MDM.EndUserAuthentication.SSOProviderSettings
	if mdmSSOSettings.IsEmpty() {
		err := &fleet.BadRequestError{Message: "Couldn't initiate single sign-on for Fleet Desktop because no IdP is configured for end user authentication."}
		return nil, ctxerr.Wrap(ctx, err, "initiate device sso")
	}

	// The ACS base must match what the IdP app has registered,
	// will be ServerURL if none currently set.
	acsBase, err := url.Parse(appConfig.MDMUrl())
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "invalid MDM URL")
	}

	browserBase, err := appConfig.FleetDesktopBrowserUrl()
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "invalid server URL")
	}

	// SSO cookies are __Host- prefixed, so each is pinned to one host name.
	// The handshake cookie is set on the browser's host but read on the ACS, and
	// the session cookie the other way around: if those hosts differ, neither
	// arrives and the end user loops.
	if !strings.EqualFold(browserBase.Hostname(), acsBase.Hostname()) {
		err := &fleet.BadRequestError{
			Message: "Fleet Desktop single sign-on requires the device page and the SAML callback on the same host.",
		}
		return nil, ctxerr.Wrap(ctx, err, "initiate device sso")
	}

	acsURL := sso.CallbackURL(acsBase, svc.config.Server.URLPrefix, "/api/v1/fleet/mdm/sso/callback").String()

	samlProvider, err := sso.SAMLProviderFromConfiguredMetadata(ctx, mdmSSOSettings.EntityID, acsURL, &mdmSSOSettings)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "failed to create provider from metadata")
	}

	sessionDuration := svc.config.Auth.SsoSessionValidityPeriod
	sessionID, idpURL, err := sso.CreateAuthorizationRequest(ctx,
		samlProvider,
		svc.ssoSessionStore,
		sso.URLWithPrefix(browserBase, svc.config.Server.URLPrefix, deviceURL).String(),
		uint(sessionDuration.Seconds()), //nolint:gosec // dismiss G115
		fleet.SSORelayState(fleet.SSOInitiatorFleetDesktop),
		sso.SSORequestData{
			HostUUID:  host.UUID,
			Initiator: fleet.SSOInitiatorFleetDesktop,
		},
	)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "InitiateDeviceSSO creating authorization")
	}

	return &fleet.DeviceSSOInitiation{
		IdPURL:          idpURL,
		SessionID:       sessionID,
		SessionDuration: sessionDuration,
	}, nil
}
