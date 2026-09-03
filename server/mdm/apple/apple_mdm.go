package apple_mdm

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"slices"
	"strings"
	"text/template"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	ctxabm "github.com/fleetdm/fleet/v4/server/contexts/apple_bm"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/logging"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/gdmf"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	"github.com/fleetdm/fleet/v4/server/mdm/internal/commonmdm"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/godep"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/cryptoutil"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"

	depclient "github.com/fleetdm/fleet/v4/server/mdm/nanodep/client"
	nanodep_storage "github.com/fleetdm/fleet/v4/server/mdm/nanodep/storage"
	depsync "github.com/fleetdm/fleet/v4/server/mdm/nanodep/sync"
)

const (
	// SCEPPath is Fleet's HTTP path for the SCEP service.
	SCEPPath = "/mdm/apple/scep"
	// MDMPath is Fleet's HTTP path for the core MDM service.
	MDMPath = "/mdm/apple/mdm"
	// MDMServiceDiscoveryPath is Fleet's base HTTP path for the MDM service discovery service. And is kept for backwards compatible reasons.
	//
	// Deprecated: Use ServiceDiscoveryTokenPath instead.
	ServiceDiscoveryPath      = "/mdm/apple/service_discovery"
	ServiceDiscoveryTokenPath = "/mdm/apple/service_discovery/{token}" // nolint:gosec // Not a secret

	// EnrollPath is the HTTP path that serves the mobile profile to devices when enrolling.
	EnrollPath = "/api/mdm/apple/enroll"
	// AccountDrivenEnrollPath is the HTTP path that serves the mobile profile to devices when enrolling.
	//
	// Deprecated: Use AccountDrivenEnrollTokenPath instead.
	AccountDrivenEnrollPath      = "/api/mdm/apple/account_driven_enroll"
	AccountDrivenEnrollTokenPath = "/api/mdm/apple/account_driven_enroll/{token}" // nolint:gosec // Not a secret
	// InstallerPath is the HTTP path that serves installers to Apple devices.
	InstallerPath = "/api/mdm/apple/installer"

	// FleetUISSOCallbackPath is the front-end route used to
	// redirect after the SSO flow is completed.
	FleetUISSOCallbackPath = "/mdm/sso/callback"

	// FleetUISSOCallbackError redirects to the callback route's generic error.
	FleetUISSOCallbackError = FleetUISSOCallbackPath + "?error=true"

	// FleetUISSOCallbackSessionExpired redirects to the callback route and asks
	// it for the timed-out message instead of the generic one. Signing in can
	// take a while on a device being set up for the first time, and the generic
	// error gives the end user nothing to act on.
	FleetUISSOCallbackSessionExpired = FleetUISSOCallbackError + "&reason=session_expired"

	// FleetUIDeviceSSOError is where the Fleet Desktop SSO flow lands when the
	// callback fails before loading the SSO session: the device-page counterpart
	// of FleetUISSOCallbackError.
	FleetUIDeviceSSOError = "/device/sso-error?reason=error"

	// FleetUIDeviceSSOErrorSessionExpired is the device-page counterpart of
	// FleetUISSOCallbackSessionExpired.
	FleetUIDeviceSSOErrorSessionExpired = "/device/sso-error?reason=session_expired"

	// FleetPayloadIdentifier is the value for the "<key>PayloadIdentifier</key>"
	// used by Fleet MDM on the enrollment profile.
	FleetPayloadIdentifier = "com.fleetdm.fleet.mdm.apple"

	// SCEPProxyPath is the HTTP path that serves the SCEP proxy service. The path is followed by identifier.
	SCEPProxyPath = "/mdm/scep/proxy/"

	// It's important we don't sync more than 1000 at a time,
	// as we also process DEP cooldowns and limit how many we process with this variable
	DEPSyncLimit = 200

	VPPLicenseNotFound = 9610

	DeclarationTypeSoftwareUpdate = "com.apple.configuration.softwareupdate.enforcement.specific"
)

// MDM AccessRights bitmask values per Apple Device Management documentation.
// https://developer.apple.com/documentation/devicemanagement/mdm#properties
//
// MDMAccessRightAll is the full set Fleet has always delivered historically.
// Callers that renew an existing host's profile must compute the new value as
// (stored_rights AND current_max_rights) to honour the monotonic-narrowing rule:
// Apple rejects an enrollment-profile replacement that grants MORE rights than
// the previously-installed profile.
const (
	MDMAccessRightAll         = 8191 // all 13 bits (2^13 - 1)
	MDMAccessRightDeviceLock  = 4    // bit 2: Device Lock & Passcode Removal
	MDMAccessRightDeviceErase = 8    // bit 3: Device Erase (wipe)
)

// AppleEnrollmentAccessRights returns the AccessRights bitmask to embed in a
// manual (SCEP/ACME) enrollment profile. For personal (BYOD) devices the lock
// and erase bits are stripped so IT admins cannot lock the device out or wipe
// personal data; company-owned devices receive full rights.
func AppleEnrollmentAccessRights(personal bool) int {
	if !personal {
		return MDMAccessRightAll
	}
	return MDMAccessRightAll &^ (MDMAccessRightDeviceLock | MDMAccessRightDeviceErase)
}

// FleetPersonalEnrollmentKey is the URL query-parameter key added to the MDM
// ServerURL in enrollment profiles for personal (BYOD) devices. nanomdm surfaces
// URL query parameters as request.Params so the Authenticate checkin handler can
// read it and set host_mdm.is_personal_enrollment accordingly. The "byod" key
// matches the param the OTA endpoint and the /enroll page already use.
const FleetPersonalEnrollmentKey = "byod"

// FleetEnrollmentSubjectOU is the Organizational Unit Fleet embeds in the SCEP
// certificate Subject of NEW-enrollment profiles (never renewals). The SCEP signer
// copies the CSR Subject verbatim into the issued identity certificate, so this
// marker survives into the cert the device presents at MDM Authenticate. The checkin
// handler reads it to tell a fresh enrollment apart from a stale pending SCEP renewal
// (see server/service/apple_mdm.go Authenticate). It must NOT be set on renewal
// profiles, or renewals would be misclassified as fresh enrollments.
const FleetEnrollmentSubjectOU = "Fleet Device Enrollment"

// AddPersonalEnrollmentToFleetURL appends the FleetPersonalEnrollmentKey query
// param to fleetURL when personal is true. If personal is false the URL is
// returned unchanged so company-owned profiles carry no extra parameters.
func AddPersonalEnrollmentToFleetURL(fleetURL string, personal bool) (string, error) {
	if !personal {
		return fleetURL, nil
	}
	u, err := url.Parse(fleetURL)
	if err != nil {
		return "", fmt.Errorf("parsing configured server URL: %w", err)
	}
	q := u.Query()
	q.Set(FleetPersonalEnrollmentKey, "1")
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func ResolveAppleMDMURL(serverURL string) (string, error) {
	return commonmdm.ResolveURL(serverURL, MDMPath, false)
}

func ResolveAppleEnrollMDMURL(serverURL string) (string, error) {
	return commonmdm.ResolveURL(serverURL, EnrollPath, false)
}

func ResolveAppleSCEPURL(serverURL string) (string, error) {
	// Apple's SCEP client appends a query string to the SCEP URL in the
	// enrollment profile, without checking if the URL already has a query
	// string. Eg: if the URL is `/test/example?foo=bar` it'll make a
	// request to `/test/example?foo=bar?SCEPOperation=..`
	//
	// As a consequence we ensure that the query is always clean for the SCEP URL.
	return commonmdm.ResolveURL(serverURL, SCEPPath, true)
}

func ResolveAppleACMEDirectoryURL(serverURL string, acmeIdent string) (string, error) {
	return commonmdm.ResolveURL(serverURL, fmt.Sprintf("/api/mdm/acme/%s/directory", acmeIdent), true)
}

// DEPService is used to encapsulate tasks related to DEP enrollment.
//
// This service doesn't perform any authentication checks, so its suitable for
// internal usage within Fleet. If you need to expose any of the functionality
// to users, please make sure the caller is enforcing the right authorization
// checks.
type DEPService struct {
	ds         fleet.Datastore
	depStorage nanodep_storage.AllDEPStorage
	depClient  *godep.Client
	logger     *slog.Logger
}

// GetDefaultProfile returns a godep.Profile with default values set.
func (d *DEPService) GetDefaultProfile() *godep.Profile {
	// If this definition change, make sure to update the fleetctl new template file
	return &godep.Profile{
		ProfileName:    "Fleet default enrollment profile",
		IsSupervised:   true,
		IsMDMRemovable: false,
	}
}

// createDefaultAutomaticProfile creates the default automatic (DEP) enrollment
// profile in mdm_apple_enrollment_profiles but does not register it with
// Apple. It also creates the authentication token to get enrollment profiles.
func (d *DEPService) createDefaultAutomaticProfile(ctx context.Context) error {
	depProfile := d.GetDefaultProfile()
	token := uuid.New().String()
	rawDEPProfile, err := json.Marshal(depProfile)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "marshaling default profile")
	}

	payload := fleet.MDMAppleEnrollmentProfilePayload{
		Token:      token,
		Type:       fleet.MDMAppleEnrollmentTypeAutomatic,
		DEPProfile: ptr.RawMessage(rawDEPProfile),
	}
	if _, err := d.ds.NewMDMAppleEnrollmentProfile(ctx, payload); err != nil {
		return ctxerr.Wrap(ctx, err, "saving enrollment profile in DB")
	}
	return nil
}

// CreateDefaultAutomaticProfile creates the default automatic enrollment profile in the DB.
func (d *DEPService) CreateDefaultAutomaticProfile(ctx context.Context) error {
	return d.createDefaultAutomaticProfile(ctx)
}

func (d *DEPService) buildJSONProfile(ctx context.Context, setupAsstJSON json.RawMessage, appCfg *fleet.AppConfig, team *fleet.Team, enrollURL string) (*godep.Profile, error) {
	var jsonProf godep.Profile
	jsonProf.IsMDMRemovable = true // the default value defined by Apple is true
	if err := json.Unmarshal(setupAsstJSON, &jsonProf); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "unmarshalling DEP profile")
	}

	// if configuration_web_url is set, this setting is completely managed by the
	// IT admin.
	if jsonProf.ConfigurationWebURL == "" {
		// If SSO is configured, use the `/mdm/sso` page which starts the SSO
		// flow, otherwise leave it blank.
		endUserAuthEnabled := appCfg.MDM.MacOSSetup.EnableEndUserAuthentication
		if team != nil {
			endUserAuthEnabled = team.Config.MDM.MacOSSetup.EnableEndUserAuthentication
		}
		if endUserAuthEnabled {
			mdmSSOURL, err := commonmdm.ResolveURL(appCfg.MDMUrl(), "/mdm/sso", false)
			if err != nil {
				return nil, fmt.Errorf("resolve MDM SSO URL: %w", err)
			}
			jsonProf.ConfigurationWebURL = mdmSSOURL
		}
	}

	if jsonProf.ConfigurationWebURL != "" {
		// ensure `url` is the same as `configuration_web_url`, to not leak the URL
		// to get a token without SSO enabled
		jsonProf.URL = jsonProf.ConfigurationWebURL
	} else {
		// Without configuration_web_url set, the host will send a POST request
		// to `url` location to get the MDM profile.
		// 2025-05-20 unofficial docs: https://github.com/4d-for-ios-sdk/Mobile-Device-Management-Protocol-Reference/blob/master/markdown/4-Profile_Management/4-Profile_Management.md#request-to-a-profile-url
		jsonProf.URL = enrollURL
	}

	// always set await_device_configured to true - it will be released either
	// automatically by Fleet or manually by the user if
	// enable_release_device_manually is true.
	jsonProf.AwaitDeviceConfigured = true

	return &jsonProf, nil
}

// RegisterProfileWithAppleDEPServer registers the enrollment profile in
// Apple's servers via the DEP API, so it can be used for assignment. If
// setupAsst is nil, the default profile is registered. It assigns the
// up-to-date dynamic settings such as the server URL and MDM SSO URL if
// end-user authentication is enabled for that team/no-team.
//
// It does that registration for all tokens associated in any way with that
// team - that is, if DEP hosts are part of that team then the token used to
// discover those hosts will be used to register the profile, and if a token
// has that team as default team for a platform, it will also be used to
// register the profile.
//
// Note that this means that a team must either have DEP hosts associated with
// it with corresponding host_dep_assignment records or be the default team for a
// class of devices(see GetABMTokenOrgNamesAssociatedWithTeam)
//
// On success, it returns the profile uuid and timestamp for the specific token
// of interest to the caller (identified by its organization name).
func (d *DEPService) RegisterProfileWithAppleDEPServer(ctx context.Context, team *fleet.Team, setupAsst *fleet.MDMAppleSetupAssistant, abmTokenOrgName string) (string, time.Time, error) {
	appCfg, err := d.ds.AppConfig(ctx)
	if err != nil {
		return "", time.Time{}, ctxerr.Wrap(ctx, err, "fetching app config")
	}

	// must always get the default profile, because the authentication token is
	// defined on that profile.
	defaultProf, err := d.ds.GetMDMAppleEnrollmentProfileByType(ctx, fleet.MDMAppleEnrollmentTypeAutomatic)
	if err != nil {
		return "", time.Time{}, ctxerr.Wrap(ctx, err, "fetching default profile")
	}

	enrollURL, err := EnrollURL(defaultProf.Token, appCfg)
	if err != nil {
		return "", time.Time{}, ctxerr.Wrap(ctx, err, "generating enroll URL")
	}

	var rawJSON json.RawMessage
	var requestedTokenModTime time.Time
	if defaultProf.DEPProfile != nil {
		rawJSON = *defaultProf.DEPProfile
		requestedTokenModTime = defaultProf.UpdatedAt
	}
	if setupAsst != nil {
		rawJSON = setupAsst.Profile
		requestedTokenModTime = setupAsst.UploadedAt
	}

	jsonProf, err := d.buildJSONProfile(ctx, rawJSON, appCfg, team, enrollURL)
	if err != nil {
		return "", time.Time{}, ctxerr.Wrap(ctx, err, "building json profile")
	}

	depClient := NewDEPClient(d.depStorage, d.ds, d.logger)
	// Get all relevant org names
	var tmID *uint
	if team != nil {
		tmID = &team.ID
	}

	orgNames, err := d.ds.GetABMTokenOrgNamesAssociatedWithTeam(ctx, tmID)
	if err != nil {
		return "", time.Time{}, ctxerr.Wrap(ctx, err, "getting org names for team to register profile")
	}

	if len(orgNames) == 0 {
		d.logger.InfoContext(ctx, "skipping defining profile for team with no relevant ABM token")
		return "", time.Time{}, nil
	}

	var requestedTokenProfileUUID string
	for _, orgName := range orgNames {
		res, err := depClient.DefineProfile(ctx, orgName, jsonProf)
		if err != nil {
			return "", time.Time{}, ctxerr.Wrap(ctx, err, "apple POST /profile request failed")
		}

		if setupAsst != nil {
			if err := d.ds.SetMDMAppleSetupAssistantProfileUUID(ctx, setupAsst.TeamID, res.ProfileUUID, orgName); err != nil {
				return "", time.Time{}, ctxerr.Wrap(ctx, err, "save setup assistant profile UUID")
			}
		} else {
			if err := d.ds.SetMDMAppleDefaultSetupAssistantProfileUUID(ctx, tmID, res.ProfileUUID, orgName); err != nil {
				return "", time.Time{}, ctxerr.Wrap(ctx, err, "save default setup assistant profile UUID")
			}
		}
		if orgName == abmTokenOrgName {
			requestedTokenProfileUUID = res.ProfileUUID
		}
	}
	return requestedTokenProfileUUID, requestedTokenModTime, nil
}

// ValidateSetupAssistant validates the setup assistant by sending the profile to the DefineProfile
// Apple API.
func (d *DEPService) ValidateSetupAssistant(ctx context.Context, team *fleet.Team, setupAsst *fleet.MDMAppleSetupAssistant, abmTokenOrgName string) error {
	appCfg, err := d.ds.AppConfig(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "fetching app config")
	}

	// must always get the default profile, because the authentication token is
	// defined on that profile.
	defaultProf, err := d.ds.GetMDMAppleEnrollmentProfileByType(ctx, fleet.MDMAppleEnrollmentTypeAutomatic)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "fetching default profile")
	}

	enrollURL, err := EnrollURL(defaultProf.Token, appCfg)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "generating enroll URL")
	}

	rawJSON := setupAsst.Profile

	jsonProf, err := d.buildJSONProfile(ctx, rawJSON, appCfg, team, enrollURL)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "building json profile")
	}

	depClient := NewDEPClient(d.depStorage, d.ds, d.logger)
	// Get all relevant org names
	var tmID *uint
	if team != nil {
		tmID = &team.ID
	}

	orgNames, err := d.ds.GetABMTokenOrgNamesAssociatedWithTeam(ctx, tmID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting org names for team to register profile")
	}

	if len(orgNames) == 0 {
		// Then check to see if there are any tokens at all. If there is only 1, we assume we can
		// use it (the vast majority of deployments will only have a single token).
		toks, err := d.ds.ListABMTokens(ctx)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "listing ABM tokens")
		}

		if len(toks) != 1 {
			return ctxerr.New(ctx, "No relevant ABM tokens found. Please set this team as a default team for an ABM token.")
		}

		orgNames = append(orgNames, toks[0].OrganizationName)
	}

	for _, orgName := range orgNames {
		_, err := depClient.DefineProfile(ctx, orgName, jsonProf)
		if err != nil {
			var httpErr *godep.HTTPError
			if errors.As(err, &httpErr) {
				// We can count on this working because of how the godep.HTTPerror Error() method
				// formats its output.
				return ctxerr.Errorf(ctx, "Couldn't add. %s", string(httpErr.Body))
			}

			return ctxerr.Wrap(ctx, err, "sending profile to Apple failed")
		}
	}

	return nil
}

// EnsureDefaultSetupAssistant ensures that the default Setup Assistant profile
// is created and registered with Apple for the provided team/no-team (if team
// is nil) using the specified ABM token, and returns its profile UUID. It does
// not re-define the profile if it already exists and registered for that
// token.
func (d *DEPService) EnsureDefaultSetupAssistant(ctx context.Context, team *fleet.Team, abmTokenOrgName string) (string, time.Time, error) {
	// the first step is to ensure that the default profile entry exists in the
	// mdm_apple_enrollment_profiles table. When we create it there we also
	// create the authentication token to retrieve enrollment profiles, and
	// that's the place the token is stored.
	defProf, err := d.ds.GetMDMAppleEnrollmentProfileByType(ctx, fleet.MDMAppleEnrollmentTypeAutomatic)
	if err != nil && !fleet.IsNotFound(err) {
		return "", time.Time{}, ctxerr.Wrap(ctx, err, "get default automatic profile")
	}
	if defProf == nil || defProf.Token == "" {
		if err := d.createDefaultAutomaticProfile(ctx); err != nil {
			return "", time.Time{}, ctxerr.Wrap(ctx, err, "create default automatic profile")
		}
	}

	// now that the default automatic profile is created and a token generated,
	// check if the default profile was registered with Apple for the ABM token.
	var tmID *uint
	if team != nil {
		tmID = &team.ID
	}
	profUUID, modTime, err := d.ds.GetMDMAppleDefaultSetupAssistant(ctx, tmID, abmTokenOrgName)
	if err != nil && !fleet.IsNotFound(err) {
		return "", time.Time{}, ctxerr.Wrap(ctx, err, "get default setup assistant profile uuid")
	}
	if profUUID == "" {
		d.logger.InfoContext(ctx, "default DEP profile not set, registering")
		profUUID, modTime, err = d.RegisterProfileWithAppleDEPServer(ctx, team, nil, abmTokenOrgName)
		if err != nil {
			return "", time.Time{}, ctxerr.Wrap(ctx, err, "register default setup assistant with Apple")
		}
	}
	return profUUID, modTime, nil
}

// EnsureCustomSetupAssistantIfExists ensures that the custom Setup Assistant
// profile associated with the provided team (or no team) is registered with
// Apple for the specified ABM token, and returns its profile UUID. It does not
// re-define the profile if it is already registered for that token. If no
// custom setup assistant exists, it returns an empty string and timestamp and
// no error.
func (d *DEPService) EnsureCustomSetupAssistantIfExists(ctx context.Context, team *fleet.Team, abmTokenOrgName string) (string, time.Time, error) {
	var tmID *uint
	if team != nil {
		tmID = &team.ID
	}

	asst, err := d.ds.GetMDMAppleSetupAssistant(ctx, tmID)
	if err != nil {
		if fleet.IsNotFound(err) {
			// no error, no custom setup assistant for that team
			return "", time.Time{}, nil
		}
		return "", time.Time{}, err
	}

	// if we get here, there IS a custom setup assistant, so get its profile UUID
	profileUUID, modTime, err := d.ds.GetMDMAppleSetupAssistantProfileForABMToken(ctx, tmID, abmTokenOrgName)
	if err != nil && !fleet.IsNotFound(err) {
		return "", time.Time{}, err
	}

	if profileUUID == "" {
		// registers the profile for all tokens associated with the team
		profileUUID, modTime, err = d.RegisterProfileWithAppleDEPServer(ctx, team, asst, abmTokenOrgName)
		if err != nil {
			return "", time.Time{}, err
		}
	}
	return profileUUID, modTime, nil
}

func (d *DEPService) RunAssigner(ctx context.Context) error {
	syncerLogger := logging.NewNanoDEPLogger(ctx, d.logger.With("component", "nanodep-syncer"))
	teams, err := d.ds.ListTeams(
		ctx, fleet.TeamFilter{
			User: &fleet.User{
				GlobalRole: ptr.String(fleet.RoleAdmin),
			},
		}, fleet.ListOptions{},
	)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "listing teams")
	}

	teamsByID := make(map[uint]*fleet.Team, len(teams))
	for _, tm := range teams {
		teamsByID[tm.ID] = tm
	}

	tokens, err := d.ds.ListABMTokens(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "listing ABM tokens")
	}

	var result error
	for _, token := range tokens {
		var macOSTeam, iosTeam, ipadTeam *fleet.Team

		if token.MacOSDefaultTeamID != nil {
			macOSTeam = teamsByID[*token.MacOSDefaultTeamID]
		}

		if token.IOSDefaultTeamID != nil {
			iosTeam = teamsByID[*token.IOSDefaultTeamID]
		}

		if token.IPadOSDefaultTeamID != nil {
			ipadTeam = teamsByID[*token.IPadOSDefaultTeamID]
		}

		teams := []*fleet.Team{macOSTeam, iosTeam, ipadTeam}
		for _, team := range teams {
			// ensure the default (fallback) setup assistant profile exists, registered
			// with Apple DEP.
			_, defModTime, err := d.EnsureDefaultSetupAssistant(ctx, team, token.OrganizationName)
			if err != nil {
				result = multierror.Append(result, err)
				continue
			}

			// if the team/no-team has a custom setup assistant, ensure it is registered
			// with Apple DEP.
			customUUID, customModTime, err := d.EnsureCustomSetupAssistantIfExists(ctx, team, token.OrganizationName)
			if err != nil {
				result = multierror.Append(result, err)
				continue
			}

			// get the modification timestamp of the effective profile (custom or default)
			effectiveProfModTime := defModTime
			if customUUID != "" {
				effectiveProfModTime = customModTime
			}

			cursor, cursorModTime, err := d.depStorage.RetrieveCursor(ctx, token.OrganizationName)
			if err != nil {
				result = multierror.Append(result, err)
				continue
			}

			if cursor != "" && effectiveProfModTime.After(cursorModTime) {
				d.logger.InfoContext(ctx, "clearing device syncer cursor", "org_name", token.OrganizationName)
				if err := d.depStorage.StoreCursor(ctx, token.OrganizationName, ""); err != nil {
					result = multierror.Append(result, err)
					continue
				}
			}

		}

		syncer := depsync.NewSyncer(
			d.depClient,
			token.OrganizationName,
			d.depStorage,
			depsync.WithLogger(syncerLogger),
			depsync.WithCallback(func(ctx context.Context, isFetch bool, resp *godep.DeviceResponse) error {
				// the nanodep syncer just logs the error of the callback, so in order to
				// capture it we need to do this here.
				err := d.processDeviceResponse(ctx, resp, token.ID, token.OrganizationName, macOSTeam, iosTeam, ipadTeam)
				if err != nil {
					ctxerr.Handle(ctx, err)
				}
				return err
			}),
			depsync.WithLimit(DEPSyncLimit),
		)

		if err := syncer.Run(ctx); err != nil {
			result = multierror.Append(result, err)
			continue
		}
	}

	return result
}

func (d *DEPService) GetMDMAppleServiceDiscoveryDetails(ctx context.Context, tokenOrgName string) (*godep.AccountDrivenEnrollmentProfileResponse, error) {
	// TODO: In some of the other DEPService methods (e.g., RegisterProfileWithAppleDEPServiceE)
	// we always create a new depClient specifically for that method. Why? Should we do the same
	// here or should we update those other methods to use the d.depClient instance like we are here?
	if d.depClient == nil {
		d.depClient = NewDEPClient(d.depStorage, d.ds, d.logger)
	}

	return d.depClient.FetchAccountDrivenEnrollmentServiceDiscovery(ctx, tokenOrgName)
}

func (d *DEPService) AssignMDMAppleServiceDiscoveryURL(ctx context.Context, tokenOrgName string, url string) error {
	// TODO: In some of the other DEPService methods (e.g., RegisterProfileWithAppleDEPServiceE)
	// we always create a new depClient specifically for that method. Why? Should we do the same
	// here or should we update those other methods to use the d.depClient instance like we are here?
	if d.depClient == nil {
		d.depClient = NewDEPClient(d.depStorage, d.ds, d.logger)
	}

	return d.depClient.AssignAccountDrivenEnrollmentServiceDiscovery(ctx, tokenOrgName, url)
}

func NewDEPService(
	ds fleet.Datastore,
	depStorage nanodep_storage.AllDEPStorage,
	logger *slog.Logger,
) *DEPService {
	depSvc := &DEPService{
		depStorage: depStorage,
		logger:     logger,
		ds:         ds,
		depClient:  NewDEPClient(depStorage, ds, logger),
	}

	return depSvc
}

// processDeviceResponse processes the device response from the device sync
// DEP API endpoints and assigns the profile UUID associated with the DEP
// client DEP name.
func (d *DEPService) processDeviceResponse(
	ctx context.Context,
	resp *godep.DeviceResponse,
	abmTokenID uint,
	abmOrganizationName string,
	macOSTeam *fleet.Team,
	iosTeam *fleet.Team,
	ipadTeam *fleet.Team,
) error {
	if len(resp.Devices) < 1 {
		// no devices means we can't assign anything
		return nil
	}

	var addedDevicesSlice []godep.Device
	var addedSerials []string
	var deletedSerials []string
	var modifiedSerials []string
	addedDevices := map[string]godep.Device{}
	modifiedDevices := map[string]godep.Device{}
	deletedDevices := map[string]godep.Device{}

	// This service may return the same device more than once. You must resolve duplicates by matching on the device
	// serial number and the op_type and op_date fields. The record with the latest op_date indicates the last known
	// state of the device in DEP.
	// Reference: https://developer.apple.com/documentation/devicemanagement/sync_the_list_of_devices#discussion
	keepRecent := func(device godep.Device, existing map[string]godep.Device) {
		existingDevice, ok := existing[device.SerialNumber]
		if !ok || device.OpDate.After(existingDevice.OpDate) {
			existing[device.SerialNumber] = device
		}
	}

	for _, device := range resp.Devices {
		deadline := "nil"
		if device.MDMMigrationDeadline != nil {
			deadline = device.MDMMigrationDeadline.String()
		}
		// FIXME: Move this log back to debug level after we've added/improved functionality for accessing DEP status.
		d.logger.InfoContext(ctx, "process device response",
			"serial_number", device.SerialNumber,
			"device_assigned_by", device.DeviceAssignedBy,
			"device_assigned_date", device.DeviceAssignedDate,
			"op_date", device.OpDate,
			"op_type", device.OpType,
			"profile_status", device.ProfileStatus,
			"profile_assign_time", device.ProfileAssignTime,
			"push_push_time", device.ProfilePushTime,
			"profile_uuid", device.ProfileUUID,
			"mdm_migration_deadline", deadline,
		)

		switch strings.ToLower(device.OpType) {
		// The op_type field is only applicable with the SyncDevices API call,
		// Empty op_type come from the first call to FetchDevices without a cursor,
		// and we do want to assign profiles to them.
		case "added", "":
			keepRecent(device, addedDevices)
		case "modified":
			keepRecent(device, modifiedDevices)
		case "deleted":
			keepRecent(device, deletedDevices)
		default:
			d.logger.WarnContext(ctx, "unrecognized op_type",
				"op_type", device.OpType,
				"serial_number", device.SerialNumber,
			)
		}
	}

	// Remove added/modified devices if they have been subsequently deleted
	// Remove deleted devices if they have been subsequently added (or re-added)
	for _, deletedDevice := range deletedDevices {
		// FIXME: Shouldn't the logic for modified devices follow the if/else pattern used for added
		// devices? It seems like it should, but it doesn't seem to be making a difference in
		// practice. Presumably, we're catching this sommewhere else, but it isn't obvious where.
		modifiedDevice, ok := modifiedDevices[deletedDevice.SerialNumber]
		if ok && deletedDevice.OpDate.After(modifiedDevice.OpDate) {
			delete(modifiedDevices, deletedDevice.SerialNumber)
		}
		addedDevice, ok := addedDevices[deletedDevice.SerialNumber]
		if ok {
			if deletedDevice.OpDate.After(addedDevice.OpDate) {
				delete(addedDevices, deletedDevice.SerialNumber)
			} else {
				delete(deletedDevices, deletedDevice.SerialNumber)
			}
		}
	}

	// Devices just added to an MDM server must have their profile updated.
	// In our testing, added devices with a profile_uuid (which were removed and then re-added, for example)
	// may not be able to download the profile and enroll in MDM.
	needProfileAssign := make(map[string]struct{})
	for _, addedDevice := range addedDevices {
		addedDevicesSlice = append(addedDevicesSlice, addedDevice)
		needProfileAssign[addedDevice.SerialNumber] = struct{}{}
	}
	for _, modifiedDevice := range modifiedDevices {
		// FIXME: Are we properly determining whether a modified device needs a profile assigned?
		modifiedSerials = append(modifiedSerials, modifiedDevice.SerialNumber)
	}
	for _, deletedDevice := range deletedDevices {
		deletedSerials = append(deletedSerials, deletedDevice.SerialNumber)
	}

	// find out if we already have entries in the `hosts` table with
	// matching serial numbers for any devices with op_type = "modified"
	existingSerials, err := d.ds.GetMatchingHostSerials(ctx, modifiedSerials)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get matching host serials")
	}

	// treat devices with op_type = "modified" that doesn't exist in the
	// `hosts` table, as an "added" device.
	//
	// we need to do this because _sometimes_, ABM sends op_type = "modified"
	// if the IT admin changes the MDM server assignment in the ABM UI. In
	// these cases, the device is new ("added") to us, but it comes with
	// the wrong op_type.
	for _, md := range modifiedDevices {
		if _, ok := existingSerials[md.SerialNumber]; !ok {
			d.logger.InfoContext(ctx, "treating device with op_type modified as added device", "serial_number", md.SerialNumber)
			addedDevicesSlice = append(addedDevicesSlice, md)
		}
		// FIXME: addedDevicesSlice is used in part to determine if a profile assignment is needed.
		// Should be be checking if the modified device has the right profile UUID and current timestamp?
	}

	// Check if added devices belong to another ABM server. If so, we must delete them before adding them.
	for _, device := range addedDevicesSlice {
		addedSerials = append(addedSerials, device.SerialNumber)
	}

	// Check if any of the "added" or "modified" hosts are hosts that we've recently removed from
	// Fleet in ABM. A host in this state will have a row in `host_dep_assignments` where the
	// `deleted_at ` col is NOT NULL. Down below we skip assigning the profile to devices that we
	// think are still enrolled; doing this check here allows us to avoid skipping devices that
	// _seem_ like they're still enrolled but were actually removed and should get the profile.
	// See https://github.com/fleetdm/fleet/issues/23200 for more context.
	existingDeletedSerials, err := d.ds.GetMatchingHostSerialsMarkedDeleted(ctx, addedSerials)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get matching deleted host serials")
	}

	err = d.ds.DeleteHostDEPAssignmentsFromAnotherABM(ctx, abmTokenID, addedSerials)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "deleting dep assignments from another abm")
	}

	err = d.ds.DeleteHostDEPAssignments(ctx, abmTokenID, deletedSerials)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "deleting DEP assignments")
	}

	n, err := d.ds.IngestMDMAppleDevicesFromDEPSync(ctx, addedDevicesSlice, abmTokenID, macOSTeam, iosTeam, ipadTeam)
	switch {
	case err != nil:
		d.logger.ErrorContext(ctx, "error ingesting DEP devices", "err", err)
		ctxerr.Handle(ctx, err)
	case n > 0:
		d.logger.InfoContext(ctx, fmt.Sprintf("added %d new mdm device(s) to pending hosts", n))
	case n == 0:
		d.logger.DebugContext(ctx, "no DEP hosts to add")
	}

	d.logger.InfoContext(ctx, "devices to assign DEP profiles",
		"to_add", strings.Join(addedSerials, ", "),
		"to_remove", strings.Join(deletedSerials, ", "),
		"to_modify", strings.Join(modifiedSerials, ", "),
	)

	// at this point, the hosts rows are created for the devices, with the
	// correct team_id, so we know what team-specific profile needs to be applied.
	//
	// collect a map of all the profiles => serials we need to assign.
	profileToDevices := map[string][]godep.Device{}
	var iosTeamID, macOSTeamID, ipadTeamID *uint
	if iosTeam != nil {
		iosTeamID = &iosTeam.ID
	}
	if macOSTeam != nil {
		macOSTeamID = &macOSTeam.ID
	}
	if ipadTeam != nil {
		ipadTeamID = &ipadTeam.ID
	}

	// each new device should be assigned the DEP profile of the default
	// ABM team as configured by the IT admin.
	devicesByTeam := map[*uint][]godep.Device{}
	for _, newDevice := range addedDevicesSlice {
		var teamID *uint
		switch newDevice.DeviceFamily {
		case "iPhone", "iPod":
			teamID = iosTeamID
		case "iPad":
			teamID = ipadTeamID
		default:
			teamID = macOSTeamID
		}
		devicesByTeam[teamID] = append(devicesByTeam[teamID], newDevice)
	}

	// for all other hosts we received, find out the right DEP profile to
	// assign, based on the team.
	existingHosts := []fleet.Host{}
	existingHostMigrationDeadlines := make(map[uint]time.Time)
	for _, existingHost := range existingSerials {
		d.logger.InfoContext(ctx, "preparing to upsert DEP assignment for existing host", "serial", existingHost.HardwareSerial, "host_id", existingHost.ID)
		md, ok := modifiedDevices[existingHost.HardwareSerial]
		if !ok {
			d.logger.ErrorContext(ctx,
				"serial coming from ABM is in the database, but it's not in the list of modified devices", "serial",
				existingHost.HardwareSerial)
			continue
		}
		if md.MDMMigrationDeadline != nil {
			existingHostMigrationDeadlines[existingHost.ID] = *md.MDMMigrationDeadline
		}
		existingHosts = append(existingHosts, *existingHost)
		devicesByTeam[existingHost.TeamID] = append(devicesByTeam[existingHost.TeamID], md)
	}

	// Upsert the host DEP assignment records now so that the team is properly linked to the ABM
	// token if this is the first device DEP host for this token assigned to the team.
	if len(existingHosts) > 0 {
		if err := d.ds.UpsertMDMAppleHostDEPAssignments(ctx, existingHosts, abmTokenID, existingHostMigrationDeadlines); err != nil {
			return ctxerr.Wrap(ctx, err, "upserting dep assignment for existing devices")
		}
	}

	// assign the profile to each device
	for team, devices := range devicesByTeam {
		// FIXME: Do we have replication issues or races? There seem to be alot of calls going on inside this function.
		profUUID, err := d.getProfileUUIDForTeam(ctx, team, abmOrganizationName)
		if err != nil {
			return ctxerr.Wrapf(ctx, err, "getting profile for team with id: %v", team)
		}

		profileToDevices[profUUID] = append(profileToDevices[profUUID], devices...)
	}

	// keep track of the serials we're going to skip for all profiles in
	// order to log them later.
	var skippedSerials []string
	for profUUID, devices := range profileToDevices {
		var serials []string
		for _, device := range devices {
			_, deleted := existingDeletedSerials[device.SerialNumber]
			_, needsProfile := needProfileAssign[device.SerialNumber]
			if device.ProfileUUID == profUUID && device.ProfileStatus != "removed" && !deleted && !needsProfile {
				skippedSerials = append(skippedSerials, device.SerialNumber)
				continue
			}
			serials = append(serials, device.SerialNumber)
		}

		if len(serials) == 0 {
			continue
		}

		logger := d.logger.With("profile_uuid", profUUID)

		skipSerials, assignSerials, err := d.ds.ScreenDEPAssignProfileSerialsForCooldown(ctx, serials)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "process device response")
		}

		seen := make(map[string]struct{}, len(serials))
		for _, ss := range skipSerials {
			for _, s := range ss {
				seen[s] = struct{}{}
			}
		}
		for _, ss := range assignSerials {
			for _, s := range ss {
				seen[s] = struct{}{}
			}
		}
		var missing []string
		for _, s := range serials {
			if _, ok := seen[s]; !ok {
				missing = append(missing, s)
			}
		}

		if len(missing) > 0 {
			// We did not get all serials passed in, back assigned to a bucket, could be due to potential replica lag.
			// We force the remaining serials into the assign bucket, to ensure they get processed.
			logger.InfoContext(ctx, "found missing serials after cooldown screening, adding them to assign profile operation", "serials", missing, "count", len(missing), "org_name", abmOrganizationName)
			assignSerials[abmOrganizationName] = append(assignSerials[abmOrganizationName], missing...)
		}

		if len(skipSerials) > 0 {
			// NOTE: the `dep_cooldown` job of the `integrations`` cron picks up the assignments
			// after the cooldown period is over
			logger.InfoContext(ctx, "process device response: skipping assign profile for devices on cooldown", "serials", fmt.Sprintf("%s",
				skipSerials))
		}
		if len(assignSerials) == 0 {
			logger.InfoContext(ctx, "process device response: no devices to assign profile")
			continue
		}

		for orgName, serials := range assignSerials {
			apiResp, err := d.depClient.AssignProfile(ctx, orgName, profUUID, serials...)
			if err != nil {
				// only log the error so the failure can be recorded
				// below in UpdateHostDEPAssignProfileResponses and
				// the proper cooldowns are applied
				logger.ErrorContext(ctx, "assign profile",
					"devices", len(serials),
					"err", err,
				)
			}
			// Verify that all serials assigned get some sort of terminal status. Otherwise an error
			// that returns no devices at all(i.e. a network error) could result in a serial being
			// dropped on the floor. Failed may or may not be the right status here since it will
			// cause cooldowns to be applied however it ensures we retry these assignments
			implicitlyFailedAssignments := 0
			if apiResp.Devices == nil {
				apiResp.Devices = make(map[string]string)
			}
			for _, serial := range serials {
				if _, ok := apiResp.Devices[serial]; !ok {
					apiResp.Devices[serial] = string(fleet.DEPAssignProfileResponseFailed)
					implicitlyFailedAssignments++
				}
			}
			// We don't expect to see this but log here just in case
			if err != nil && implicitlyFailedAssignments > 0 {
				logger.ErrorContext(ctx,
					"assign profile: no error was returned but some devices were not assigned a status in the response",
					"devices", implicitlyFailedAssignments,
				)
			}

			attrs := []any{
				"devices", len(serials),
			}
			attrs = append(attrs, logCountsForResults(apiResp.Devices)...)
			logger.InfoContext(ctx, "profile assigned", attrs...)

			if err := d.ds.UpdateHostDEPAssignProfileResponses(ctx, apiResp, abmTokenID); err != nil {
				return ctxerr.Wrap(ctx, err, "update host dep assign profile responses")
			}
		}
	}

	if len(skippedSerials) > 0 {
		d.logger.InfoContext(ctx, "found devices that already have the right profile, skipping assignment", "serials",
			fmt.Sprintf("%s", skippedSerials))
	}

	return nil
}

func (d *DEPService) getProfileUUIDForTeam(ctx context.Context, tmID *uint, abmTokenOrgName string) (string, error) {
	var appleBMTeam *fleet.Team
	if tmID != nil {
		tm, err := d.ds.TeamWithExtras(ctx, *tmID) // TODO see if we can convert to TeamLite
		if err != nil && !fleet.IsNotFound(err) {
			return "", ctxerr.Wrap(ctx, err, "get team")
		}
		appleBMTeam = tm
	}

	// get profile uuid of team or default
	profUUID, _, err := d.EnsureCustomSetupAssistantIfExists(ctx, appleBMTeam, abmTokenOrgName)
	if err != nil {
		return "", fmt.Errorf("ensure setup assistant for team: %w", err)
	}
	if profUUID == "" {
		profUUID, _, err = d.EnsureDefaultSetupAssistant(ctx, appleBMTeam, abmTokenOrgName)
		if err != nil {
			return "", fmt.Errorf("ensure default setup assistant: %w", err)
		}
	}

	return profUUID, nil
}

// logCountsForResults tries to aggregate the result types and log the counts.
func logCountsForResults(deviceResults map[string]string) (out []interface{}) {
	results := map[string]int{"success": 0, "not_accessible": 0, "failed": 0, "throttled": 0, "other": 0}
	for _, result := range deviceResults {
		l := strings.ToLower(result)
		if _, ok := results[l]; !ok {
			l = "other"
		}
		results[l] += 1
	}
	for k, v := range results {
		if v > 0 {
			out = append(out, k, v)
		}
	}
	return
}

// NewDEPClient creates an Apple DEP API HTTP client based on the provided
// storage that will flag the ABM token's terms expired field and the
// AppConfig's AppleBMTermsExpired field whenever the status of the terms
// changes, and flag the ABM token's token_invalid field whenever Apple
// rejects the token or reports its signature as invalid.
func NewDEPClient(storage godep.ClientStorage, updater fleet.ABMTermsUpdater, logger *slog.Logger) *godep.Client {
	return godep.NewClient(storage, fleethttp.NewClient(), godep.WithAfterHook(func(ctx context.Context, reqErr error) error {
		// to check for ABM terms expired, we must have an ABM token organization
		// name and NOT a raw ABM token in the context (as the presence of a raw
		// ABM token means that the token is new, hasn't been saved in the DB yet
		// so no point checking for the terms expired as we don't have a row in
		// abm_tokens to save that flag).
		orgName := depclient.GetName(ctx)
		if _, rawTokenPresent := ctxabm.FromContext(ctx); rawTokenPresent || orgName == "" {
			return reqErr
		}

		// if the request failed due to the token being rejected or its
		// signature being invalid, flag the ABM token's token_invalid. If it
		// succeeded, or failed with a different *definitive* signal that the
		// token itself was accepted (e.g. terms not signed -- Apple only
		// evaluates terms after authenticating the token), clear the flag.
		// Any other failure (e.g. a transient network or server error) is
		// inconclusive and leaves the flag untouched. This must happen before
		// the terms-expired handling below, as that block can return early
		// from this after-hook once its own bookkeeping is done.
		tokenInvalid := reqErr != nil && (godep.IsTokenRejected(reqErr) || godep.IsSignatureInvalid(reqErr))
		tokenAccepted := reqErr == nil || godep.IsTermsNotSigned(reqErr)
		if tokenAccepted || tokenInvalid {
			// Check the current value via a read replica first, so the common
			// case (the flag already has the desired value, which is most DEP
			// API calls) doesn't hit the writer. If the read fails, fall back to
			// always writing, since that's no worse than before this check
			// existed.
			needsUpdate := true
			if currentlyInvalid, err := updater.IsABMTokenInvalidForOrgName(ctx, orgName); err != nil {
				logger.ErrorContext(ctx, "Apple DEP client: failed to get token invalid status of ABM token", "err", err)
			} else {
				needsUpdate = currentlyInvalid != tokenInvalid
			}
			if needsUpdate {
				if _, err := updater.SetABMTokenInvalidForOrgName(ctx, orgName, tokenInvalid); err != nil {
					logger.ErrorContext(ctx, "Apple DEP client: failed to update token invalid status of ABM token", "err", err)
				}
			}
		}

		// if the request failed due to terms not signed, or if it succeeded,
		// update the ABM token's (and possibly the app config's) flag accordingly.
		// If it failed for any other reason, do not update the flag.
		termsExpired := reqErr != nil && godep.IsTermsNotSigned(reqErr)
		if reqErr == nil || termsExpired {
			// get the count of tokens with the flag still set
			count, err := updater.CountABMTokensWithTermsExpired(ctx)
			if err != nil {
				logger.ErrorContext(ctx, "Apple DEP client: failed to get count of tokens with terms expired", "err", err)
				return reqErr
			}

			// get the appconfig for the global flag
			appCfg, err := updater.AppConfig(ctx)
			if err != nil {
				logger.ErrorContext(ctx, "Apple DEP client: failed to get app config", "err", err)
				return reqErr
			}

			// on API call success, if the global terms expired flag is not set and
			// the count is 0, no need to do anything else (it means this ABM token
			// already had the flag cleared).
			if reqErr == nil && count == 0 && !appCfg.MDM.AppleBMTermsExpired {
				return reqErr
			}

			// otherwise, update the specific ABM token's flag
			wasSet, err := updater.SetABMTokenTermsExpiredForOrgName(ctx, orgName, termsExpired)
			if err != nil {
				logger.ErrorContext(ctx, "Apple DEP client: failed to update terms expired of ABM token", "err", err)
				return reqErr
			}

			// update the count of ABM tokens with the flag set accordingly
			stillSetCount := count
			if wasSet && !termsExpired {
				stillSetCount--
			} else if !wasSet && termsExpired {
				stillSetCount++
			}

			var mustSaveAppCfg bool
			if stillSetCount > 0 && !appCfg.MDM.AppleBMTermsExpired {
				// flag the AppConfig that the terms have changed and must be accepted
				// for at least one token
				appCfg.MDM.AppleBMTermsExpired = true
				mustSaveAppCfg = true
			} else if stillSetCount == 0 && appCfg.MDM.AppleBMTermsExpired {
				// flag the AppConfig that the terms have been accepted for all tokens
				appCfg.MDM.AppleBMTermsExpired = false
				mustSaveAppCfg = true
			}

			if mustSaveAppCfg {
				if err := updater.SaveAppConfig(ctx, appCfg); err != nil {
					logger.ErrorContext(ctx, "Apple DEP client: failed to save app config", "err", err)
				}
				logger.InfoContext(ctx, "Apple DEP client: updated app config Terms Expired flag",
					"apple_bm_terms_expired", appCfg.MDM.AppleBMTermsExpired)
			}
		}

		return reqErr
	}))
}

// ClassifyDEPDeviceError classifies an error returned by a DEP device-details
// style call (e.g. godep.Client.GetDeviceDetails) into a DEPDeviceErrorType
// for API consumers, or "" if err is nil.
func ClassifyDEPDeviceError(err error) fleet.DEPDeviceErrorType {
	switch {
	case err == nil:
		return ""
	case godep.IsTokenRejected(err) || godep.IsSignatureInvalid(err):
		return fleet.DEPDeviceErrorTokenInvalid
	case godep.IsTermsNotSigned(err):
		return fleet.DEPDeviceErrorTermsExpired
	case godep.IsServerError(err):
		return fleet.DEPDeviceErrorServerError
	default:
		return fleet.DEPDeviceErrorUnavailable
	}
}

var funcMap = map[string]any{
	"xml": mobileconfig.XMLEscapeString,
}

var OTASCEPTemplate = template.Must(template.New("").Funcs(funcMap).Parse(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple Inc//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
  <dict>
    <key>PayloadVersion</key>
    <integer>1</integer>
    <key>PayloadType</key>
    <string>Configuration</string>
    <key>PayloadIdentifier</key>
    <string>Ignored</string>
    <key>PayloadUUID</key>
    <string>Ignored</string>
    <key>PayloadContent</key>
    <array>
      <dict>
        <key>PayloadContent</key>
        <dict>
          <key>Key Type</key>
          <string>RSA</string>
          <key>Challenge</key>
          <string>{{ .SCEPChallenge | xml }}</string>
          <key>Key Usage</key>
          <integer>5</integer>
          <key>Keysize</key>
          <integer>2048</integer>
          <key>URL</key>
          <string>{{ .SCEPURL }}</string>
          <key>Subject</key>
          <array>
            <array>
              <array>
                <string>O</string>
                <string>Fleet</string>
              </array>
            </array>
            <array>
              <array>
                <string>CN</string>
                <string>Fleet Identity</string>
              </array>
            </array>
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
    </array>
  </dict>
</plist>`))

// enrollmentProfileMobileconfigTemplate is the template Fleet uses to assemble a .mobileconfig enrollment profile to serve to devices.
//
// During a profile replacement, the system updates payloads with the same PayloadIdentifier and
// PayloadUUID in the old and new profiles.
var enrollmentProfileMobileconfigTemplate = template.Must(template.New("").Funcs(funcMap).Parse(`
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
				<string>{{ .SCEPChallenge | xml }}</string>
				<key>Key Usage</key>
				<integer>5</integer>
				<key>Keysize</key>
				<integer>2048</integer>
				<key>URL</key>
				<string>{{ .SCEPURL }}</string>
				<key>Subject</key>
				<array>
					<array><array><string>O</string><string>Fleet</string></array></array>
					{{ if .NewEnrollmentSubjectOU }}<array><array><string>OU</string><string>{{ .NewEnrollmentSubjectOU | xml }}</string></array></array>
					{{ end }}<array><array><string>CN</string><string>Fleet Identity</string></array></array>
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
			<integer>{{ .AccessRights }}</integer>
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
				<string>com.apple.mdm.bootstraptoken</string>
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
	<string>{{ .Organization | xml }} enrollment</string>
	<key>PayloadIdentifier</key>
	<string>` + FleetPayloadIdentifier + `</string>
	<key>PayloadOrganization</key>
	<string>{{ .Organization | xml }}</string>
	<key>PayloadType</key>
	<string>Configuration</string>
	<key>PayloadUUID</key>
	<string>5ACABE91-CE30-4C05-93E3-B235C152404E</string>
	<key>PayloadVersion</key>
	<integer>1</integer>
</dict>
</plist>`))

var accountDrivenUserEnrollmentProfileMobileconfigTemplate = template.Must(template.New("").Funcs(funcMap).Parse(`
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
				<string>{{ .SCEPChallenge | xml }}</string>
				<key>Key Usage</key>
				<integer>5</integer>
				<key>Keysize</key>
				<integer>2048</integer>
				<key>URL</key>
				<string>{{ .SCEPURL }}</string>
				<key>Subject</key>
				<array>
					<array><array><string>O</string><string>Fleet</string></array></array>
					{{ if .NewEnrollmentSubjectOU }}<array><array><string>OU</string><string>{{ .NewEnrollmentSubjectOU | xml }}</string></array></array>
					{{ end }}<array><array><string>CN</string><string>Fleet Identity</string></array></array>
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
			<key>AssignedManagedAppleID</key>
			<string>{{ .AssignedManagedAppleID | xml }}</string>
			<key>EnrollmentMode</key>
			<string>BYOD</string>
			<key>ServerCapabilities</key>
			<array>
				<string>UserEnrollment</string>
				<string>com.apple.mdm.per-user-connections</string>
				<string>com.apple.mdm.bootstraptoken</string>
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
	<string>{{ .Organization | xml }} enrollment</string>
	<key>PayloadIdentifier</key>
	<string>` + FleetPayloadIdentifier + `</string>
	<key>PayloadOrganization</key>
	<string>{{ .Organization | xml }}</string>
	<key>PayloadScope</key>
	<string>User</string>
	<key>PayloadType</key>
	<string>Configuration</string>
	<key>PayloadUUID</key>
	<string>5ACABE91-CE30-4C05-93E3-B235C152404E</string>
	<key>PayloadVersion</key>
	<integer>1</integer>
</dict>
</plist>`))

var acmeEnrollmentProfileMobileconfigTemplate = template.Must(template.New("").Funcs(funcMap).Parse(`
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>PayloadContent</key>
	<array>
		<dict>
			<key>Attest</key>
			<true/>
			<key>ClientIdentifier</key>
			<string>{{ .ClientIdentifier | xml }}</string>
			<key>DirectoryURL</key>
			<string>{{ .DirectoryURL | xml }}</string>
			<key>HardwareBound</key>
			<true/>
			<key>KeySize</key>
			<integer>384</integer>
			<key>KeyType</key>
			<string>ECSECPrimeRandom</string>
			<key>PayloadDisplayName</key>
			<string>Fleet Identity ACME</string>
			<key>PayloadIdentifier</key>
			<string>BCA53F9D-5DD2-494D-98D3-0D0F20FF6BA1</string>
			<key>PayloadType</key>
			<string>com.apple.security.acme</string>
			<key>PayloadUUID</key>
			<string>BCA53F9D-5DD2-494D-98D3-0D0F20FF6BA1</string>
			<key>PayloadVersion</key>
			<integer>1</integer>
			<key>Subject</key>
			<array>
				{{ if .NewEnrollmentSubjectOU }}<array><array><string>OU</string><string>{{ .NewEnrollmentSubjectOU | xml }}</string></array></array>
				{{ end }}<array>
					<array>
						<string>CN</string>
						<string>{{ .ClientIdentifier | xml }}</string>
					</array>
				</array>
			</array>
		</dict>
		<dict>
			<key>AccessRights</key>
			<integer>{{ .AccessRights }}</integer>
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
				<string>com.apple.mdm.bootstraptoken</string>
			</array>
			<key>ServerURL</key>
			<string>{{ .ServerURL | xml }}</string>
			<key>SignMessage</key>
			<true/>
			<key>Topic</key>
			<string>{{ .Topic | xml }}</string>
		</dict>
	</array>
	<key>PayloadDisplayName</key>
	<string>{{ .Organization | xml }} enrollment</string>
	<key>PayloadIdentifier</key>
	<string>` + FleetPayloadIdentifier + `</string>
	<key>PayloadOrganization</key>
	<string>{{ .Organization | xml }}</string>
	<key>PayloadType</key>
	<string>Configuration</string>
	<key>PayloadUUID</key>
	<string>5ACABE91-CE30-4C05-93E3-B235C152404E</string>
	<key>PayloadVersion</key>
	<integer>1</integer>
</dict>
</plist>`))

// GenerateEnrollmentProfileMobileconfig builds a standard SCEP enrollment profile. Set newEnrollment
// to true for profiles served from an enroll endpoint and false for SCEP renewal profiles; it controls
// whether the SCEP Subject carries FleetEnrollmentSubjectOU, the marker the checkin handler uses to
// distinguish a fresh enrollment from a renewal.
func GenerateEnrollmentProfileMobileconfig(orgName, fleetURL, scepChallenge, topic string, accessRights int, newEnrollment bool) ([]byte, error) {
	scepURL, err := ResolveAppleSCEPURL(fleetURL)
	if err != nil {
		return nil, fmt.Errorf("resolve Apple SCEP url: %w", err)
	}
	serverURL, err := ResolveAppleMDMURL(fleetURL)
	if err != nil {
		return nil, fmt.Errorf("resolve Apple MDM url: %w", err)
	}

	var buf bytes.Buffer
	if err := enrollmentProfileMobileconfigTemplate.Funcs(funcMap).Execute(&buf, struct {
		Organization           string
		SCEPURL                string
		SCEPChallenge          string
		Topic                  string
		ServerURL              string
		AccessRights           int
		NewEnrollmentSubjectOU string
	}{
		Organization:           orgName,
		SCEPURL:                scepURL,
		SCEPChallenge:          scepChallenge,
		Topic:                  topic,
		ServerURL:              serverURL,
		AccessRights:           accessRights,
		NewEnrollmentSubjectOU: newEnrollmentSubjectOU(newEnrollment),
	}); err != nil {
		return nil, fmt.Errorf("execute template: %w", err)
	}
	return buf.Bytes(), nil
}

// GenerateAccountDrivenEnrollmentProfileMobileconfig builds an account-driven (BYOD) SCEP enrollment
// profile. See GenerateEnrollmentProfileMobileconfig for newEnrollment.
func GenerateAccountDrivenEnrollmentProfileMobileconfig(orgName, fleetURL, scepChallenge, topic, assignedManagedAppleID string, newEnrollment bool) ([]byte, error) {
	scepURL, err := ResolveAppleSCEPURL(fleetURL)
	if err != nil {
		return nil, fmt.Errorf("resolve Apple SCEP url: %w", err)
	}
	serverURL, err := ResolveAppleMDMURL(fleetURL)
	if err != nil {
		return nil, fmt.Errorf("resolve Apple MDM url: %w", err)
	}

	var buf bytes.Buffer
	if err := accountDrivenUserEnrollmentProfileMobileconfigTemplate.Funcs(funcMap).Execute(&buf, struct {
		Organization           string
		SCEPURL                string
		SCEPChallenge          string
		Topic                  string
		ServerURL              string
		AssignedManagedAppleID string
		NewEnrollmentSubjectOU string
	}{
		Organization:           orgName,
		SCEPURL:                scepURL,
		SCEPChallenge:          scepChallenge,
		Topic:                  topic,
		ServerURL:              serverURL,
		AssignedManagedAppleID: assignedManagedAppleID,
		NewEnrollmentSubjectOU: newEnrollmentSubjectOU(newEnrollment),
	}); err != nil {
		return nil, fmt.Errorf("execute template: %w", err)
	}
	return buf.Bytes(), nil
}

// newEnrollmentSubjectOU returns the SCEP Subject OU marker for a fresh enrollment, or "" for a
// renewal (which omits the OU so renewals aren't misclassified as fresh enrollments).
func newEnrollmentSubjectOU(newEnrollment bool) string {
	if newEnrollment {
		return FleetEnrollmentSubjectOU
	}
	return ""
}

func AddEnrollmentRefToFleetURL(fleetURL, reference string) (string, error) {
	if reference == "" {
		return fleetURL, nil
	}

	u, err := url.Parse(fleetURL)
	if err != nil {
		return "", fmt.Errorf("parsing configured server URL: %w", err)
	}
	q := u.Query()
	q.Add(mobileconfig.FleetEnrollReferenceKey, reference)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// GenerateACMEEnrollmentProfileMobileconfig builds an ACME (hardware-attested) enrollment profile. See
// GenerateEnrollmentProfileMobileconfig for newEnrollment; the OU marker survives because Fleet's ACME
// signer reuses the SCEP depot signer, which copies the CSR Subject verbatim.
// deviceSerial fills both ClientIdentifier and the Subject CN. We observed that on iOS renewals,
// the device does not substitute %SerialNumber% with its serial number, so we fill it in.
func GenerateACMEEnrollmentProfileMobileconfig(orgName, mdmURL, acmeIdent, deviceSerial, topic string, accessRights int, newEnrollment bool) ([]byte, error) {
	serverURL, err := ResolveAppleMDMURL(mdmURL)
	if err != nil {
		return nil, fmt.Errorf("resolve Apple MDM url: %w", err)
	}

	acmeURL, err := ResolveAppleACMEDirectoryURL(mdmURL, acmeIdent)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := acmeEnrollmentProfileMobileconfigTemplate.Funcs(funcMap).Execute(&buf, struct {
		Organization           string
		DirectoryURL           string
		Topic                  string
		ServerURL              string
		ClientIdentifier       string
		AccessRights           int
		NewEnrollmentSubjectOU string
	}{
		Organization:           orgName,
		DirectoryURL:           acmeURL,
		Topic:                  topic,
		ServerURL:              serverURL,
		ClientIdentifier:       deviceSerial,
		AccessRights:           accessRights,
		NewEnrollmentSubjectOU: newEnrollmentSubjectOU(newEnrollment),
	}); err != nil {
		return nil, fmt.Errorf("execute template: %w", err)
	}

	// TODO: In the PoC, the generated profile unexpectedly escaped the left angle bracket in the opening
	// `<?xml` tag. If we see that again, the replacement below can be used as a workaround, but
	// ideally we should figure out why that is happening in the first place.
	// return bytes.Replace(buf.Bytes(), []byte("&lt;"), []byte("<"), 1), nil

	return buf.Bytes(), nil
}

// ProfileBimap implements bidirectional mapping for profiles, and utility
// functions to generate those mappings based on frequently used operations.
type ProfileBimap struct {
	wantedState  map[*fleet.MDMAppleProfilePayload]*fleet.MDMAppleProfilePayload
	currentState map[*fleet.MDMAppleProfilePayload]*fleet.MDMAppleProfilePayload
}

// NewProfileBimap retuns a new ProfileBimap
func NewProfileBimap() *ProfileBimap {
	return &ProfileBimap{
		map[*fleet.MDMAppleProfilePayload]*fleet.MDMAppleProfilePayload{},
		map[*fleet.MDMAppleProfilePayload]*fleet.MDMAppleProfilePayload{},
	}
}

// GetMatchingProfileInDesiredState returns the addition key that matches the given removal
func (pb *ProfileBimap) GetMatchingProfileInDesiredState(removal *fleet.MDMAppleProfilePayload) (*fleet.MDMAppleProfilePayload, bool) {
	value, ok := pb.currentState[removal]
	return value, ok
}

// GetMatchingProfileInCurrentState returns the removal key that matches the given addition
func (pb *ProfileBimap) GetMatchingProfileInCurrentState(addition *fleet.MDMAppleProfilePayload) (*fleet.MDMAppleProfilePayload, bool) {
	key, ok := pb.wantedState[addition]
	return key, ok
}

// IntersectByIdentifierAndHostUUID populates the bimap matching the profiles by Identifier and HostUUID
func (pb *ProfileBimap) IntersectByIdentifierAndHostUUID(wantedProfiles, currentProfiles []*fleet.MDMAppleProfilePayload) {
	key := func(p *fleet.MDMAppleProfilePayload) string {
		return fmt.Sprintf("%s-%s", p.ProfileIdentifier, p.HostUUID)
	}

	removeProfs := map[string]*fleet.MDMAppleProfilePayload{}
	for _, p := range currentProfiles {
		removeProfs[key(p)] = p
	}

	for _, p := range wantedProfiles {
		if pp, ok := removeProfs[key(p)]; ok {
			pb.add(p, pp)
		}
	}
}

func (pb *ProfileBimap) add(wantedProfile, currentProfile *fleet.MDMAppleProfilePayload) {
	pb.wantedState[wantedProfile] = currentProfile
	pb.currentState[currentProfile] = wantedProfile
}

// NewActivityFunc is an alias for fleet.NewActivityFunc.
type NewActivityFunc = fleet.NewActivityFunc

func IOSiPadOSRefetch(ctx context.Context, ds fleet.Datastore, commander *MDMAppleCommander, logger *slog.Logger,
	newActivityFn NewActivityFunc,
) error {
	appCfg, err := ds.AppConfig(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "fetching app config")
	}

	if !appCfg.MDM.EnabledAndConfigured {
		logger.DebugContext(ctx, "apple mdm is not configured, skipping run")
		return nil
	}

	start := time.Now()
	devices, err := ds.ListIOSAndIPadOSToRefetch(ctx, 1*time.Hour)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "list ios and ipados devices to refetch")
	}
	if len(devices) == 0 {
		return nil
	}
	logger.InfoContext(ctx, "sending commands to refetch", "count", len(devices), "lookup-duration", time.Since(start))

	type deviceGroup struct {
		hostIDs []uint
		uuids   []string
	}

	// trackAndSend records the tracking rows BEFORE enqueueing so that a device
	// acknowledging the command right after the APNs push can't race the
	// insert and leave behind a row that no result will ever clear. The nano
	// enqueue is a single transaction, so on enqueue failure nothing was
	// queued and the rows are removed again; if only the notification failed
	// the command is durably queued and the rows must stay.
	trackAndSend := func(commandType string, group deviceGroup, wrapMsg string, enqueue func() error) error {
		rows := make([]fleet.HostMDMCommand, 0, len(group.hostIDs))
		for _, hostID := range group.hostIDs {
			rows = append(rows, fleet.HostMDMCommand{
				HostID:      hostID,
				CommandType: commandType,
			})
		}
		if err := ds.AddHostMDMCommands(ctx, rows); err != nil {
			return ctxerr.Wrap(ctx, err, "add host mdm commands")
		}
		err := enqueue()
		if err != nil {
			if _, isNotifErr := errors.AsType[*NotificationFailedError](err); !isNotifErr {
				if rmErr := ds.RemoveHostMDMCommands(ctx, group.hostIDs, commandType); rmErr != nil {
					logger.ErrorContext(ctx, "untrack host mdm commands after enqueue failure",
						"err", rmErr, "command_type", commandType)
				}
			}
			turnedOff, turnedOffError := turnOffMDMIfAPNSFailed(ctx, ds, err, logger, newActivityFn)
			if turnedOffError != nil {
				return turnedOffError
			}
			if !turnedOff {
				return ctxerr.Wrap(ctx, err, wrapMsg)
			}
		}
		return nil
	}

	// groupToSend returns the devices that were not already sent a commandType command.
	groupToSend := func(commandType string) deviceGroup {
		var group deviceGroup
		for _, device := range devices {
			if slices.Contains(device.CommandsAlreadySent, commandType) {
				continue
			}
			group.hostIDs = append(group.hostIDs, device.HostID)
			group.uuids = append(group.uuids, device.UUID)
		}
		return group
	}

	// groupsByFlag groups the devices that were not already sent a commandType
	// command by the value of the command's per-batch flag (e.g. managedOnly),
	// so each group can be enqueued with the flag value its devices require.
	groupsByFlag := func(commandType string, flag func(device fleet.AppleDevicesToRefetch) bool) map[bool]deviceGroup {
		groups := map[bool]deviceGroup{}
		for _, device := range devices {
			if slices.Contains(device.CommandsAlreadySent, commandType) {
				continue
			}
			flagValue := flag(device)
			group := groups[flagValue]
			group.hostIDs = append(group.hostIDs, device.HostID)
			group.uuids = append(group.uuids, device.UUID)
			groups[flagValue] = group
		}
		return groups
	}

	appGroups := groupsByFlag(fleet.RefetchAppsCommandUUIDPrefix, func(device fleet.AppleDevicesToRefetch) bool {
		isBYODDevice := !device.InstalledFromDEP
		return isBYODDevice // BYOD devices are only queried for managed apps
	})
	for _, managedOnly := range []bool{true, false} { // fixed order keeps enqueue order deterministic
		group, ok := appGroups[managedOnly]
		if !ok {
			continue
		}

		commandUUID := uuid.NewString()
		err := trackAndSend(fleet.RefetchAppsCommandUUIDPrefix, group,
			"send InstalledApplicationList commands to ios and ipados devices", func() error {
				return commander.InstalledApplicationList(ctx, group.uuids, fleet.RefetchAppsCommandUUIDPrefix+commandUUID, managedOnly)
			})
		if err != nil {
			return err
		}
	}

	certs := groupToSend(fleet.RefetchCertsCommandUUIDPrefix)
	if len(certs.uuids) > 0 {
		commandUUID := uuid.NewString()
		err := trackAndSend(fleet.RefetchCertsCommandUUIDPrefix, certs,
			"send CertificateList commands to ios and ipados devices", func() error {
				return commander.CertificateList(ctx, certs.uuids, fleet.RefetchCertsCommandUUIDPrefix+commandUUID)
			})
		if err != nil {
			return err
		}
	}

	// DeviceInformation is last because the refetch response clears the refetch_requested flag
	deviceInfoGroups := groupsByFlag(fleet.RefetchDeviceCommandUUIDPrefix, func(device fleet.AppleDevicesToRefetch) bool {
		return device.IsPersonalEnrollment
	})
	for _, isPersonalEnrollment := range []bool{true, false} {
		group, ok := deviceInfoGroups[isPersonalEnrollment]
		if !ok {
			continue
		}

		commandUUID := uuid.NewString()
		err := trackAndSend(fleet.RefetchDeviceCommandUUIDPrefix, group,
			"send DeviceInformation commands to ios and ipados devices", func() error {
				return commander.DeviceInformation(ctx, group.uuids, fleet.RefetchDeviceCommandUUIDPrefix+commandUUID, isPersonalEnrollment)
			})
		if err != nil {
			return err
		}
	}

	return nil
}

// turnOffMDMIfAPNSFailed turns off MDM for any device whose push was rejected
// by APNs with an Unregistered reason (dead token). It returns true whenever
// err is an APNSDeliveryError — even when no device was turned off — signaling
// the caller that only push delivery failed (commands remain durably queued)
// so processing can continue.
func turnOffMDMIfAPNSFailed(ctx context.Context, ds fleet.Datastore, err error, logger *slog.Logger, newActivityFn NewActivityFunc) (bool,
	error,
) {
	var e *APNSDeliveryError
	if !errors.As(err, &e) {
		return false, nil
	}

	for uuid, err := range e.errorsByUUID {
		// nanopush surfaces APNs' structured rejection reason; buford rendered
		// this same condition as "device token is inactive". If the push
		// provider ever changes, this classification must change with it.
		if APNSReason(err) == APNSReasonUnregistered {
			logger.InfoContext(ctx, "turning off MDM for device with inactive device token", "uuid", uuid)
			users, activities, err := ds.MDMTurnOff(ctx, uuid)
			if err != nil {
				return false, ctxerr.Wrap(ctx, err, "turn off mdm for failed device")
			}

			if len(users) != len(activities) {
				return false, ctxerr.New(ctx, "number of users and activities must match, this is a Fleet development bug")
			}

			for i, act := range activities {
				user := users[i]
				if err := newActivityFn(ctx, user, act); err != nil {
					return false, ctxerr.Wrap(ctx, err, "create activity")
				}
			}
		}
	}
	return true, nil
}

func GenerateOTAEnrollmentProfileMobileconfig(orgName, fleetURL, enrollSecret, idpUUID string, personal bool) ([]byte, error) {
	path, err := url.JoinPath(fleetURL, "/api/v1/fleet/ota_enrollment")
	if err != nil {
		return nil, fmt.Errorf("creating path for ota enrollment url: %w", err)
	}

	enrollURL, err := url.Parse(path)
	if err != nil {
		return nil, fmt.Errorf("parsing ota enrollment url: %w", err)
	}

	q := enrollURL.Query()
	q.Set("enroll_secret", enrollSecret)
	if idpUUID != "" {
		q.Set("idp_uuid", idpUUID)
	}
	if personal {
		q.Set("byod", "true")
	}
	enrollURL.RawQuery = q.Encode()

	var profileBuf bytes.Buffer
	tmplArgs := struct {
		Organization string
		URL          string
		EnrollSecret string
	}{
		Organization: orgName,
		URL:          enrollURL.String(),
	}

	err = mobileconfig.OTAMobileConfigTemplate.Execute(&profileBuf, tmplArgs)
	if err != nil {
		return nil, fmt.Errorf("executing ota profile template: %w", err)
	}

	return profileBuf.Bytes(), nil
}

func IOSiPadOSRevive(ctx context.Context, ds fleet.Datastore, commander *MDMAppleCommander, logger *slog.Logger) error {
	appCfg, err := ds.AppConfig(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "fetching app config")
	}

	if !appCfg.MDM.EnabledAndConfigured {
		logger.DebugContext(ctx, "apple mdm is not configured, skipping run")
		return nil
	}

	ids, err := ds.ListMDMAppleEnrolledIPhoneIpadDeletedFromFleet(ctx, 500)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "list ios and ipados devices to revive")
	}
	if len(ids) == 0 {
		return nil
	}

	if err := commander.SendNotifications(ctx, ids); err != nil {
		var apnsErr *APNSDeliveryError
		if errors.As(err, &apnsErr) {
			logger.InfoContext(ctx, "failed to send APNs notification to some hosts", "error", apnsErr.Error())
			return nil
		}
		return ctxerr.Wrap(ctx, err, "sending push notifications")
	}
	return nil
}

func ValidateMDMSettingsAppleSupportedOSVersion[T fleet.MDM | fleet.TeamMDM](settings T, excludeNonPublicAssetSets bool) (map[string]string, error) {
	var macOSUpdates, iOSUpdates, iPadOSUpdates fleet.AppleOSUpdateSettings
	if m, ok := any(settings).(fleet.MDM); ok {
		macOSUpdates = m.MacOSUpdates
		iOSUpdates = m.IOSUpdates
		iPadOSUpdates = m.IPadOSUpdates
	} else if t, ok := any(settings).(fleet.TeamMDM); ok {
		macOSUpdates = t.MacOSUpdates
		iOSUpdates = t.IOSUpdates
		iPadOSUpdates = t.IPadOSUpdates
	} else {
		return nil, errors.New("invalid settings type")
	}

	// "latest" is a sentinel, not a version: the concrete target is resolved per
	// host from Apple's published versions later on, so there is nothing to look
	// up here.
	needsVersionCheck := func(s fleet.AppleOSUpdateSettings) bool {
		return s.MinimumVersion.Value != "" && !s.EnforcesLatestVersion()
	}

	if !needsVersionCheck(macOSUpdates) && !needsVersionCheck(iOSUpdates) && !needsVersionCheck(iPadOSUpdates) {
		// nothing to validate, so don't pay for the round trip to Apple.
		return nil, nil
	}

	am, err := gdmf.GetAssetMetadata()
	if err != nil {
		return nil, fmt.Errorf("fetching Apple asset metadata: %w", err)
	} else if am == nil {
		// this should never happen, but just in case, return an error indicating that the metadata is not available instead of panicking with a nil pointer dereference
		return nil, errors.New("Apple asset metadata is not available")
	}

	invalid := make(map[string]string, 3)
	if needsVersionCheck(macOSUpdates) {
		if ok := am.IsSupportedMacOSVersion(macOSUpdates.MinimumVersion.Value, excludeNonPublicAssetSets); !ok {
			invalid["macos"] = fleet.AppleOSVersionUnsupportedMessage
		}
	}
	if needsVersionCheck(iOSUpdates) {
		// NOTE: iPod generally falls in the category of iOS in Fleet, but we're only validating against iPhone here
		// because we assume Apple will eventually remove iPod versions from the Apple Software Lookup Service
		// and we want to avoid breaking workflows for users in that event
		if ok := am.IsSupportedIOSVersion(iOSUpdates.MinimumVersion.Value, "iphone", excludeNonPublicAssetSets); !ok {
			invalid["ios"] = fleet.AppleOSVersionUnsupportedMessage
		}
	}
	if needsVersionCheck(iPadOSUpdates) {
		if ok := am.IsSupportedIOSVersion(iPadOSUpdates.MinimumVersion.Value, "ipad", excludeNonPublicAssetSets); !ok {
			invalid["ipados"] = fleet.AppleOSVersionUnsupportedMessage
		}
	}

	return invalid, nil
}

// RecoveryLockCommander defines the interface for sending recovery lock commands.
// This interface is implemented by MDMAppleCommander and allows for testing.
type RecoveryLockCommander interface {
	SetRecoveryLock(ctx context.Context, hostUUIDs []string, cmdUUID string) error
	ClearRecoveryLock(ctx context.Context, hostUUIDs []string, cmdUUID string) error
	RotateRecoveryLock(ctx context.Context, hostUUIDs []string, cmdUUID string) error
}

// SendRecoveryLockCommands is the cron job function that sends SetRecoveryLock MDM commands
// to hosts that need a recovery lock password.
//
// Note: SetRecoveryLock command results are handled in the MDM results handler
// (server/service/apple_mdm.go), which sends VerifyRecoveryLock immediately upon acknowledgment.
func SendRecoveryLockCommands(
	ctx context.Context,
	ds fleet.Datastore,
	commander *MDMAppleCommander,
	logger *slog.Logger,
	newActivityFn fleet.NewActivityFunc,
) error {
	return sendRecoveryLockCommandsWithCommander(ctx, ds, commander, logger, newActivityFn)
}

func sendRecoveryLockCommandsWithCommander(
	ctx context.Context,
	ds fleet.Datastore,
	commander RecoveryLockCommander,
	logger *slog.Logger,
	newActivityFn fleet.NewActivityFunc,
) error {
	var result *multierror.Error

	// Restore hosts that were in "pending remove" state but feature was re-enabled.
	// This transitions them back to "verified install" to preserve the existing password.
	restored, err := ds.RestoreRecoveryLockForReenabledHosts(ctx)
	if err != nil {
		result = multierror.Append(result, ctxerr.Wrap(ctx, err, "restore recovery lock for re-enabled hosts"))
	} else if restored > 0 {
		logger.InfoContext(ctx, "restored recovery lock for re-enabled hosts", "count", restored)
	}

	// Soft-delete any live rows whose host is no longer MDM enrolled. Catches hosts
	// where MDM was disabled without firing MDMTurnOff or MDMResetEnrollment — typically
	// when the device user removed the MDM profile manually and only osquery refetch
	// eventually reported host_mdm.enrolled=0. One bounded UPDATE per cron tick.
	swept, err := ds.SoftDeleteRecoveryLockPasswordsForUnenrolledHosts(ctx)
	if err != nil {
		result = multierror.Append(result, ctxerr.Wrap(ctx, err, "soft-delete recovery lock passwords for unenrolled hosts"))
	} else if swept > 0 {
		logger.InfoContext(ctx, "soft-deleted recovery lock passwords for unenrolled hosts", "count", swept)
	}

	// Handle SET password operations (hosts that need a recovery lock password)
	if err := sendSetRecoveryLockCommands(ctx, ds, commander, logger); err != nil {
		result = multierror.Append(result, err)
	}

	// Handle CLEAR password operations (hosts that need their recovery lock cleared)
	if err := sendClearRecoveryLockCommands(ctx, ds, commander, logger); err != nil {
		result = multierror.Append(result, err)
	}

	// Handle AUTO-ROTATION for viewed passwords (password viewed 1+ hour ago)
	if err := sendAutoRotationCommands(ctx, ds, commander, logger, newActivityFn); err != nil {
		result = multierror.Append(result, err)
	}

	return result.ErrorOrNil()
}

func sendSetRecoveryLockCommands(
	ctx context.Context,
	ds fleet.Datastore,
	commander RecoveryLockCommander,
	logger *slog.Logger,
) error {
	hosts, err := ds.GetHostsForRecoveryLockAction(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get hosts for recovery lock action")
	}

	if len(hosts) == 0 {
		logger.DebugContext(ctx, "no hosts need SetRecoveryLock")
		return nil
	}

	logger.InfoContext(ctx, "sending SetRecoveryLock commands", "count", len(hosts))

	// Hosts that already hold a password need the rotate variant of the command, which
	// also carries CurrentPassword. That is a different payload, so it is a separate
	// command with its own UUID.
	setCmdUUID, rotateCmdUUID := uuid.NewString(), uuid.NewString()

	// Generate passwords for all hosts upfront.
	// Passwords must be stored BEFORE enqueuing commands because they are injected
	// at delivery time by ExpandHostSecrets (which looks up by host UUID).
	passwords := make([]fleet.HostRecoveryLockPasswordPayload, 0, len(hosts))
	freshSetHostUUIDs := make([]string, 0, len(hosts))
	rotateHostUUIDs := make([]string, 0, len(hosts))
	for hostUUID, hasPassword := range hosts {
		cmdUUID := setCmdUUID
		if hasPassword {
			cmdUUID = rotateCmdUUID
			rotateHostUUIDs = append(rotateHostUUIDs, hostUUID)
		} else {
			freshSetHostUUIDs = append(freshSetHostUUIDs, hostUUID)
		}

		passwords = append(passwords, fleet.HostRecoveryLockPasswordPayload{
			HostUUID:              hostUUID,
			Password:              GenerateRecoveryLockPassword(),
			PendingSetCommandUUID: cmdUUID,
		})
	}

	// Store passwords with status='pending' atomically. This prevents the host from
	// being picked up again by the next cron run while we're enqueuing the command.
	// If enqueue fails, we reset the status to NULL so the host can be retried.
	if err := ds.SetHostsRecoveryLockPasswords(ctx, passwords); err != nil {
		return ctxerr.Wrap(ctx, err, "bulk set recovery lock passwords")
	}

	// Enqueue one command per batch. Each host gets their own queue entry pointing to
	// that command, and ExpandHostSecrets injects the per-host password at delivery time.
	enqueue := func(send func(context.Context, []string, string) error, hostUUIDs []string, cmdUUID string) error {
		if len(hostUUIDs) == 0 {
			return nil
		}

		err := send(ctx, hostUUIDs, cmdUUID)
		if err == nil {
			logger.InfoContext(ctx, "sent SetRecoveryLock commands",
				"host_count", len(hostUUIDs),
				"command_uuid", cmdUUID,
			)
			return nil
		}

		// Check if this is an APNs delivery error (command was persisted but push failed).
		// In this case, the command is already queued and will be delivered when the device
		// checks in, so we should NOT clear the pending status (which would cause duplicates).
		if apnsErr, ok := errors.AsType[*APNSDeliveryError](err); ok {
			logger.WarnContext(ctx, "SetRecoveryLock commands enqueued but APNs push failed",
				"host_count", len(hostUUIDs),
				"command_uuid", cmdUUID,
				"error", apnsErr,
			)
			return nil
		}

		// Persistence failed - reset status to NULL so hosts will be picked up again on next cron run.
		// The password is already stored, but a new one will be generated on retry (overwrites old).
		logger.ErrorContext(ctx, "failed to enqueue SetRecoveryLock commands",
			"host_count", len(hostUUIDs),
			"error", err,
		)
		if clearErr := ds.ClearRecoveryLockPendingStatus(ctx, hostUUIDs); clearErr != nil {
			logger.ErrorContext(ctx, "failed to clear recovery lock pending status after enqueue failure",
				"host_count", len(hostUUIDs),
				"error", clearErr,
			)
			err = multierror.Append(err, clearErr)
		}
		return ctxerr.Wrap(ctx, err, "enqueue SetRecoveryLock commands")
	}

	var result *multierror.Error
	result = multierror.Append(result,
		enqueue(commander.SetRecoveryLock, freshSetHostUUIDs, setCmdUUID),
		enqueue(commander.RotateRecoveryLock, rotateHostUUIDs, rotateCmdUUID),
	)

	return result.ErrorOrNil()
}

func sendClearRecoveryLockCommands(
	ctx context.Context,
	ds fleet.Datastore,
	commander RecoveryLockCommander,
	logger *slog.Logger,
) error {
	// The command UUID is recorded on the claimed rows so the result handler can match the
	// result back to the in-flight clear, so it has to be minted before claiming.
	cmdUUID := uuid.NewString()
	hosts, err := ds.ClaimHostsForRecoveryLockClear(ctx, cmdUUID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get hosts for recovery lock clear action")
	}

	if len(hosts) == 0 {
		logger.DebugContext(ctx, "no hosts need ClearRecoveryLock")
		return nil
	}

	logger.InfoContext(ctx, "sending ClearRecoveryLock commands", "count", len(hosts))

	// Enqueue clear command. The CurrentPassword placeholder will be expanded at
	// delivery time by ExpandHostSecrets (which looks up by host UUID).
	if err := commander.ClearRecoveryLock(ctx, hosts, cmdUUID); err != nil {
		var apnsErr *APNSDeliveryError
		if errors.As(err, &apnsErr) {
			// Command was persisted but push notification failed - log warning but don't fail.
			logger.WarnContext(ctx, "ClearRecoveryLock commands enqueued but APNs push failed",
				"host_count", len(hosts),
				"command_uuid", cmdUUID,
				"error", err,
			)
			return nil
		}

		// Persistence failed - reset status to NULL so hosts will be picked up again.
		logger.ErrorContext(ctx, "failed to enqueue ClearRecoveryLock commands",
			"host_count", len(hosts),
			"error", err,
		)
		if clearErr := ds.ClearRecoveryLockPendingStatus(ctx, hosts); clearErr != nil {
			logger.ErrorContext(ctx, "failed to clear recovery lock pending status after enqueue failure",
				"host_count", len(hosts),
				"error", clearErr,
			)
			err = multierror.Append(err, clearErr)
		}
		return ctxerr.Wrap(ctx, err, "enqueue ClearRecoveryLock commands")
	}

	logger.InfoContext(ctx, "sent ClearRecoveryLock commands",
		"host_count", len(hosts),
		"command_uuid", cmdUUID,
	)

	return nil
}

func sendAutoRotationCommands(
	ctx context.Context,
	ds fleet.Datastore,
	commander RecoveryLockCommander,
	logger *slog.Logger,
	newActivityFn fleet.NewActivityFunc,
) error {
	hosts, err := ds.GetHostsForAutoRotation(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get hosts for auto rotation")
	}

	if len(hosts) == 0 {
		logger.DebugContext(ctx, "no hosts need auto-rotation")
		return nil
	}

	logger.InfoContext(ctx, "performing auto-rotation for viewed passwords", "count", len(hosts))

	var result *multierror.Error
	for _, host := range hosts {
		newPassword := GenerateRecoveryLockPassword()
		setCmdUUID := uuid.NewString()

		// Initiate rotation - stores pending password and validates eligibility
		if err := ds.InitiateRecoveryLockRotation(ctx, host.HostUUID, setCmdUUID, newPassword); err != nil {
			// Check for benign race conditions where host state changed between
			// GetHostsForAutoRotation and now (e.g., manual rotation started,
			// password removed, host deleted, etc.)
			if fleet.IsNotFound(err) ||
				errors.Is(err, fleet.ErrRecoveryLockRotationPending) ||
				errors.Is(err, fleet.ErrRecoveryLockNotEligible) {
				logger.DebugContext(ctx, "host lost eligibility for auto-rotation",
					"host_uuid", host.HostUUID,
					"error", err,
				)
				continue
			}

			logger.ErrorContext(ctx, "failed to initiate auto-rotation",
				"host_uuid", host.HostUUID,
				"error", err,
			)
			result = multierror.Append(result, err)
			continue
		}

		// Enqueue RotateRecoveryLock command
		if err := commander.RotateRecoveryLock(ctx, []string{host.HostUUID}, setCmdUUID); err != nil {
			if apnsErr, ok := errors.AsType[*APNSDeliveryError](err); ok {
				// Command was persisted but push notification failed - log activity and continue.
				// The command will be retried when the device checks in.
				logAutoRotationActivity(ctx, logger, newActivityFn, host)
				logger.WarnContext(ctx, "auto-rotation command enqueued but APNs push failed",
					"host_uuid", host.HostUUID,
					"command_uuid", setCmdUUID,
					"error", apnsErr,
				)
				continue
			}

			// Persistence failed - clear pending rotation so host can be retried
			logger.ErrorContext(ctx, "failed to enqueue auto-rotation command",
				"host_uuid", host.HostUUID,
				"error", err,
			)
			if clearErr := ds.ClearRecoveryLockRotation(ctx, host.HostUUID); clearErr != nil {
				logger.ErrorContext(ctx, "failed to clear pending rotation after enqueue failure",
					"host_uuid", host.HostUUID,
					"error", clearErr,
				)
				result = multierror.Append(result, clearErr)
			}
			result = multierror.Append(result, err)
			continue
		}

		// Log activity for auto-rotation (Fleet-initiated)
		logAutoRotationActivity(ctx, logger, newActivityFn, host)

		logger.DebugContext(ctx, "sent auto-rotation command",
			"host_uuid", host.HostUUID,
			"command_uuid", setCmdUUID,
		)
	}

	return result.ErrorOrNil()
}

// logAutoRotationActivity logs the rotation activity for auto-rotations.
// It uses the same activity type as manual rotations but marks it as Fleet-initiated.
func logAutoRotationActivity(
	ctx context.Context,
	logger *slog.Logger,
	newActivityFn fleet.NewActivityFunc,
	host fleet.HostAutoRotationInfo,
) {
	if newActivityFn == nil {
		return
	}

	if err := newActivityFn(ctx, nil, fleet.ActivityTypeRotatedHostRecoveryLockPassword{
		HostID:          host.HostID,
		HostDisplayName: host.DisplayName,
		FleetInitiated:  true,
	}); err != nil {
		logger.WarnContext(ctx, "auto-rotation: failed to create activity",
			"host_uuid", host.HostUUID,
			"err", err,
		)
	}
}

// ManagedLocalAccountRotationCommander is the narrow interface SendManagedLocalAccountRotationCommands
// needs from the commander; lets tests inject a stand-in.
type ManagedLocalAccountRotationCommander interface {
	SetAutoAdminPassword(ctx context.Context, hostUUID, guid string, passwordHashPlist []byte, cmdUUID string) error
}

// ManagedLocalAccountRotationStore is the narrow datastore surface
// EnqueueManagedLocalAccountRotation needs.
type ManagedLocalAccountRotationStore interface {
	InitiateManagedLocalAccountRotation(ctx context.Context, hostUUID, pendingPlaintextPassword, cmdUUID string) error
	ClearManagedLocalAccountRotation(ctx context.Context, hostUUID string) error
}

// EnqueueManagedLocalAccountRotation generates a new password, persists pending
// rotation state, and enqueues SetAutoAdminPassword for hostUUID. It is the shared
// core of the user-triggered (EE service) and cron-driven rotation paths — outer
// concerns like authz, error → user-message mapping, multierror accumulation, and
// activity logging are left to callers.
//
// Outcomes:
//   - err == nil: command persisted in nano AND APNs push succeeded.
//   - errors.As(err, &APNSDeliveryError{}): command persisted, APNs push failed.
//     Pending state is preserved; caller should still log the rotation activity.
//   - any other err: pending state has been (or never was) unwound so the caller
//     can retry cleanly. If the rollback itself failed, rollbackErr is non-nil.
//
// cmdUUID is set whenever Initiate succeeded — useful for downstream logging.
func EnqueueManagedLocalAccountRotation(
	ctx context.Context,
	ds ManagedLocalAccountRotationStore,
	commander ManagedLocalAccountRotationCommander,
	hostUUID, accountUUID string,
) (cmdUUID string, err error, rollbackErr error) {
	newPassword := fleet.GenerateManagedLocalAccountPassword(false)
	hashPlist, hashErr := GenerateSaltedSHA512PBKDF2Hash(newPassword)
	if hashErr != nil {
		return "", hashErr, nil
	}

	cmdUUID = uuid.NewString()
	if initErr := ds.InitiateManagedLocalAccountRotation(ctx, hostUUID, newPassword, cmdUUID); initErr != nil {
		return "", initErr, nil
	}

	if sendErr := commander.SetAutoAdminPassword(ctx, hostUUID, accountUUID, hashPlist, cmdUUID); sendErr != nil {
		var apnsErr *APNSDeliveryError
		if errors.As(sendErr, &apnsErr) {
			return cmdUUID, sendErr, nil
		}
		if clearErr := ds.ClearManagedLocalAccountRotation(ctx, hostUUID); clearErr != nil {
			return cmdUUID, sendErr, clearErr
		}
		return cmdUUID, sendErr, nil
	}

	return cmdUUID, nil, nil
}

// SendManagedLocalAccountRotationCommands rotates the macOS managed local admin
// (`_fleetadmin`) password for hosts whose auto_rotate_at has elapsed. It mirrors
// SendRecoveryLockCommands: each row is generated/initiated/enqueued individually,
// and a benign race (host gone, manual rotation snuck in, etc.) is debug-logged
// and skipped rather than failing the cron iteration.
//
// Activity logging:
//   - rows with initiated_by_fleet=1 (view-driven) → log with FleetInitiated=true
//   - rows with initiated_by_fleet=0 (deferred manual rotation) → SKIP logging;
//     the manual click already logged the activity at click time
func SendManagedLocalAccountRotationCommands(
	ctx context.Context,
	ds fleet.Datastore,
	commander *MDMAppleCommander,
	logger *slog.Logger,
	newActivityFn fleet.NewActivityFunc,
) error {
	return sendManagedLocalAccountRotationCommandsWithCommander(ctx, ds, commander, logger, newActivityFn)
}

func sendManagedLocalAccountRotationCommandsWithCommander(
	ctx context.Context,
	ds fleet.Datastore,
	commander ManagedLocalAccountRotationCommander,
	logger *slog.Logger,
	newActivityFn fleet.NewActivityFunc,
) error {
	hosts, err := ds.GetManagedLocalAccountsForAutoRotation(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get managed local accounts for auto rotation")
	}
	if len(hosts) == 0 {
		logger.DebugContext(ctx, "no managed local accounts due for rotation")
		return nil
	}
	logger.InfoContext(ctx, "rotating managed local account passwords", "count", len(hosts))

	var result *multierror.Error
	for _, host := range hosts {
		cmdUUID, sendErr, rollbackErr := EnqueueManagedLocalAccountRotation(ctx, ds, commander, host.HostUUID, host.AccountUUID)
		if sendErr != nil {
			// Benign races: the row's eligibility flipped between the SELECT and
			// Initiate (manual rotation, host wipe, etc.). Skip silently.
			if fleet.IsNotFound(sendErr) ||
				errors.Is(sendErr, fleet.ErrManagedLocalAccountRotationPending) ||
				errors.Is(sendErr, fleet.ErrManagedLocalAccountNotEligible) {
				logger.DebugContext(ctx, "managed local account no longer eligible for rotation",
					"host_uuid", host.HostUUID, "err", sendErr)
				continue
			}
			var apnsErr *APNSDeliveryError
			if errors.As(sendErr, &apnsErr) {
				// Command persisted but push failed — pending state is preserved
				// so the device picks it up on next checkin. Still log activity
				// if the row was view-driven.
				logManagedLocalAccountRotationActivity(ctx, logger, newActivityFn, host)
				logger.WarnContext(ctx, "managed local account rotation enqueued but APNs push failed",
					"host_uuid", host.HostUUID, "command_uuid", cmdUUID, "err", sendErr)
				continue
			}
			if rollbackErr != nil {
				logger.ErrorContext(ctx, "clear managed local account pending rotation after enqueue failure",
					"host_uuid", host.HostUUID, "err", rollbackErr)
				result = multierror.Append(result, rollbackErr)
			}
			logger.ErrorContext(ctx, "managed local account rotation",
				"host_uuid", host.HostUUID, "err", sendErr)
			result = multierror.Append(result, sendErr)
			continue
		}

		logManagedLocalAccountRotationActivity(ctx, logger, newActivityFn, host)
		logger.DebugContext(ctx, "sent managed local account rotation command",
			"host_uuid", host.HostUUID, "command_uuid", cmdUUID)
	}

	return result.ErrorOrNil()
}

// logManagedLocalAccountRotationActivity logs the rotation activity ONLY when the
// row was view-driven (initiated_by_fleet=1). Rows with initiated_by_fleet=0 are
// deferred-manual rotations whose activity was logged by the EE service at click
// time — re-logging here would double-count.
func logManagedLocalAccountRotationActivity(
	ctx context.Context,
	logger *slog.Logger,
	newActivityFn fleet.NewActivityFunc,
	host fleet.HostManagedLocalAccountAutoRotationInfo,
) {
	if !host.InitiatedByFleet {
		return
	}
	if newActivityFn == nil {
		return
	}
	if err := newActivityFn(ctx, nil, fleet.ActivityTypeRotatedManagedLocalAccountPassword{
		HostID:          host.HostID,
		HostDisplayName: host.DisplayName,
		FleetInitiated:  true,
	}); err != nil {
		logger.WarnContext(ctx, "managed local account rotation: failed to create activity",
			"host_uuid", host.HostUUID, "err", err)
	}
}

// RecoveryLockPasswordCharset excludes confusing characters (0/O, 1/I/l)
const RecoveryLockPasswordCharset = "23456789ABCDEFGHJKLMNPQRSTUVWXYZ"

// GenerateRecoveryLockPassword generates a password in format: 5ADZ-HTZ8-LJJ4-B2F8-JWH3-YPBT
// (6 groups of 4 alphanumeric characters separated by dashes)
func GenerateRecoveryLockPassword() string {
	const (
		groupCount = 6
		groupLen   = 4
	)

	groups := make([]string, groupCount)
	charsetLen := len(RecoveryLockPasswordCharset)

	for i := range groupCount {
		randBytes := make([]byte, groupLen)
		_, _ = rand.Read(randBytes) // rand.Read never returns an error; it panics on failure

		group := make([]byte, groupLen)
		for j := range groupLen {
			group[j] = RecoveryLockPasswordCharset[int(randBytes[j])%charsetLen]
		}
		groups[i] = string(group)
	}

	return strings.Join(groups, "-")
}

func MDMPushCertTopic(ctx context.Context, ds fleet.MDMAssetRetriever) (string, error) {
	assets, err := ds.GetAllMDMConfigAssetsByName(ctx, []fleet.MDMAssetName{
		fleet.MDMAssetAPNSCert,
	}, nil)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "loading SCEP keypair from the database")
	}

	block, _ := pem.Decode(assets[fleet.MDMAssetAPNSCert].Value)
	if block == nil || block.Type != "CERTIFICATE" {
		return "", ctxerr.New(ctx, "decoding APNs certificate PEM data")
	}

	apnsCert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "parsing APNs certificate")
	}

	mdmPushCertTopic, err := cryptoutil.TopicFromCert(apnsCert)
	if err != nil {
		return "", ctxerr.Wrap(ctx, err, "extracting topic from APNs certificate")
	}

	return mdmPushCertTopic, nil
}

func HandleAppleMDMOSUpdates(ctx context.Context, ds fleet.Datastore, logger *slog.Logger) error {
	lastUpdatedAt, err := ds.GetLastAppleOSUpdatesUpdate(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get last apple os updates update")
	}

	if lastUpdatedAt == nil || time.Since(*lastUpdatedAt) > 24*time.Hour {
		logger.InfoContext(ctx, "pulling fresh apple os updates from gdmf")

		assetMetadata, err := gdmf.GetAssetMetadata()
		if err != nil {
			logger.ErrorContext(ctx, "error getting asset metadata from GDMF", "error", err)
			goto computeOSTargets
		}

		updates := map[string][]fleet.OSUpdateAsset{
			"macos": assetMetadata.PublicAssetSets.MacOS,
			"ios":   assetMetadata.PublicAssetSets.IOS,
		}

		err = ds.UpsertAppleOSUpdates(ctx, updates)
		if err != nil {
			logger.ErrorContext(ctx, "error upserting apple os updates", "error", err)
			goto computeOSTargets
		}

		// Apple stops reporting versions once they expire, so drop the cached assets that are no
		// longer in the set we just fetched. This only runs when the fetch succeeded, otherwise we
		// would delete assets based on an incomplete view of what Apple currently publishes.
		for class, assets := range updates {
			if len(assets) == 0 {
				logger.WarnContext(ctx, "gdmf returned no os updates for class, keeping cached assets", "class", class)
			}
		}
		deleted, err := ds.DeleteStaleAppleOSUpdates(ctx, updates)
		if err != nil {
			logger.ErrorContext(ctx, "error deleting stale apple os updates", "error", err)
			goto computeOSTargets
		}
		if deleted > 0 {
			logger.InfoContext(ctx, "deleted apple os updates no longer reported by gdmf", "count", deleted)
		}
	} else {
		logger.InfoContext(ctx, "apple os updates are less than 24 hours old, not pulling new os updates", "last_updated_at", *lastUpdatedAt)
	}

computeOSTargets:
	appCfg, err := ds.AppConfig(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "fetching app config")
	}

	// Get a list of all teams that has latest configured for macOS, iOS, or iPadOS.
	// We use admin global role to have access to all teams.
	teams, err := ds.ListTeams(ctx, fleet.TeamFilter{User: &fleet.User{
		GlobalRole: new("admin"),
	}}, fleet.ListOptions{})
	if err != nil {
		return ctxerr.Wrap(ctx, err, "listing teams")
	}

	// platform -> map of team_id -> deadline_days
	teamsWithLatest := map[string]map[uint]int{}
	teamsWithLatest["darwin"] = map[uint]int{}
	teamsWithLatest["ios"] = map[uint]int{}
	teamsWithLatest["ipados"] = map[uint]int{}

	for _, team := range teams {
		if team.Config.MDM.MacOSUpdates.MinimumVersion.Value == fleet.AppleOSUpdateLatestVersion {
			teamsWithLatest["darwin"][team.ID] = team.Config.MDM.MacOSUpdates.DeadlineDays.Value
		}
		if team.Config.MDM.IOSUpdates.MinimumVersion.Value == fleet.AppleOSUpdateLatestVersion {
			teamsWithLatest["ios"][team.ID] = team.Config.MDM.IOSUpdates.DeadlineDays.Value
		}
		if team.Config.MDM.IPadOSUpdates.MinimumVersion.Value == fleet.AppleOSUpdateLatestVersion {
			teamsWithLatest["ipados"][team.ID] = team.Config.MDM.IPadOSUpdates.DeadlineDays.Value
		}
	}

	// We will replace 0 with NULL check when looking up hosts to reconcile and checking the team_id column
	if appCfg.MDM.MacOSUpdates.MinimumVersion.Value == fleet.AppleOSUpdateLatestVersion {
		teamsWithLatest["darwin"][0] = appCfg.MDM.MacOSUpdates.DeadlineDays.Value
	}
	if appCfg.MDM.IOSUpdates.MinimumVersion.Value == fleet.AppleOSUpdateLatestVersion {
		teamsWithLatest["ios"][0] = appCfg.MDM.IOSUpdates.DeadlineDays.Value
	}
	if appCfg.MDM.IPadOSUpdates.MinimumVersion.Value == fleet.AppleOSUpdateLatestVersion {
		teamsWithLatest["ipados"][0] = appCfg.MDM.IPadOSUpdates.DeadlineDays.Value
	}

	updateAssets, err := ds.ListAppleOSUpdateAssets(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "listing apple os update assets")
	}
	const batchSize = 1000
	var cursor string
	for {
		hostBatch, err := ds.ListAppleOSUpdateHostsForReconcile(ctx, cursor, batchSize, teamsWithLatest)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "listing apple os update hosts for reconcile")
		}
		if len(hostBatch) == 0 {
			break
		}
		logger.InfoContext(ctx, "recomputing target os version and deadline for hosts", "cursor", cursor, "end_cursor", hostBatch[len(hostBatch)-1].HostUUID, "count", len(hostBatch))

		targets := computeOSUpdatesTarget(ctx, logger, hostBatch, updateAssets, teamsWithLatest)
		logger.InfoContext(ctx, "updating target os version and deadline for hosts", "count", len(targets))
		if err := ds.SetAppleOSUpdateTargetsAndResend(ctx, targets); err != nil {
			return ctxerr.Wrap(ctx, err, "setting apple os update targets and resending profiles")
		}

		cursor = hostBatch[len(hostBatch)-1].HostUUID
	}

	return nil
}

func computeOSUpdatesTarget(ctx context.Context, logger *slog.Logger, hosts []*fleet.AppleSoftwareUpdateHost, updateAssets map[string][]fleet.AppleSoftwareUpdateAsset, teamsWithLatest map[string]map[uint]int) []*fleet.ComputedAppleSoftwareUpdateHost {
	var computedHosts []*fleet.ComputedAppleSoftwareUpdateHost

	// Deduping hosts to avoid gnarly bugs
	seen := make(map[string]struct{}, len(hosts))
	var duplicateHosts int
	for _, host := range hosts {
		if _, ok := seen[host.HostUUID]; ok {
			duplicateHosts++
			continue
		}
		seen[host.HostUUID] = struct{}{}

		var updateAssetsPlatform string
		if host.Platform == "darwin" {
			updateAssetsPlatform = "macos"
		} else {
			updateAssetsPlatform = "ios" // covers iPhone, iPad, iPod
		}

		// teamsWithLatest is configured per platform, one for macos, ios, ipados.
		teamsWithLatestForPlatform, ok := teamsWithLatest[host.Platform]
		if !ok {
			logger.DebugContext(ctx, "unsupported os update platform", "platform", host.Platform, "host_uuid", host.HostUUID)
			continue
		}

		deadlineDays, ok := teamsWithLatestForPlatform[host.TeamID]
		if !ok {
			// Host no longer has a team with latest set. Clear the target version and deadline, do NOT mark for resend as the reconciler will handle complete removal of profile.
			host.TargetOSVersion = ""
			host.TargetDeadline = nil
			host.ResolvedAt = nil
			computedHosts = append(computedHosts, &fleet.ComputedAppleSoftwareUpdateHost{
				AppleSoftwareUpdateHost: *host,
				Resend:                  false,
			})
			continue
		}

		// Host has a team with latest set. Compute the target OS version and deadline.

		// Look up latest OS version from updateAssets
		assets, ok := updateAssets[updateAssetsPlatform]
		if !ok || len(assets) == 0 {
			logger.DebugContext(ctx, "no update assets found for platform", "platform", updateAssetsPlatform, "host_uuid", host.HostUUID)
			continue
		}

		var latestAsset *fleet.AppleSoftwareUpdateAsset
		for i := range assets {
			asset := &assets[i]
			if !slices.Contains(asset.SupportedDevices, host.SoftwareUpdateDeviceID) {
				continue
			}

			if latestAsset == nil {
				latestAsset = asset
				continue
			}
			// Current latest is less than this asset, so update latestAsset to this one
			if less, _ := IsLessThanVersion(latestAsset.ProductVersion, asset.ProductVersion); less {
				latestAsset = asset
			}

		}

		if latestAsset == nil {
			logger.DebugContext(ctx, "no update asset found for host's device id", "host_uuid", host.HostUUID, "device_id", host.SoftwareUpdateDeviceID)
			continue
		}

		startDate := latestAsset.PostingDate
		if latestAsset.FirstSeenAt.After(startDate) {
			startDate = latestAsset.FirstSeenAt
		}
		targetDeadline := startDate.Add(time.Duration(deadlineDays) * 24 * time.Hour)

		if latestAsset.ProductVersion == host.TargetOSVersion && (host.TargetDeadline != nil && targetDeadline.Equal(*host.TargetDeadline)) {
			logger.DebugContext(ctx, "host target version and deadline unchanged", "host_uuid", host.HostUUID, "target_version", host.TargetOSVersion, "target_deadline", host.TargetDeadline)
			continue
		}

		logger.DebugContext(ctx, "host target version and/or deadline changed", "host_uuid", host.HostUUID, "old_target_version", host.TargetOSVersion, "new_target_version", latestAsset.ProductVersion, "old_target_deadline", host.TargetDeadline, "new_target_deadline", targetDeadline)

		host.TargetOSVersion = latestAsset.ProductVersion
		host.TargetDeadline = &targetDeadline
		host.ResolvedAt = new(time.Now().UTC())

		computedHosts = append(computedHosts, &fleet.ComputedAppleSoftwareUpdateHost{
			AppleSoftwareUpdateHost: *host,
			Resend:                  true,
		})
	}

	if duplicateHosts > 0 {
		logger.WarnContext(ctx, "os updates reconcile: skipped rows for host UUIDs already seen in this batch; likely duplicate host rows sharing a UUID",
			"skipped", duplicateHosts)
	}

	return computedHosts
}
