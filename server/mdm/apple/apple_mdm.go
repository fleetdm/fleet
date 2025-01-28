package apple_mdm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	"github.com/fleetdm/fleet/v4/server/mdm/internal/commonmdm"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/godep"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/go-kit/log/level"
	"github.com/google/uuid"
	"github.com/hashicorp/go-multierror"

	depclient "github.com/fleetdm/fleet/v4/server/mdm/nanodep/client"
	nanodep_storage "github.com/fleetdm/fleet/v4/server/mdm/nanodep/storage"
	depsync "github.com/fleetdm/fleet/v4/server/mdm/nanodep/sync"
	kitlog "github.com/go-kit/log"
)

const (
	// SCEPPath is Fleet's HTTP path for the SCEP service.
	SCEPPath = "/mdm/apple/scep"
	// MDMPath is Fleet's HTTP path for the core MDM service.
	MDMPath = "/mdm/apple/mdm"

	// EnrollPath is the HTTP path that serves the mobile profile to devices when enrolling.
	EnrollPath = "/api/mdm/apple/enroll"
	// InstallerPath is the HTTP path that serves installers to Apple devices.
	InstallerPath = "/api/mdm/apple/installer"

	// FleetUISSOCallbackPath is the front-end route used to
	// redirect after the SSO flow is completed.
	FleetUISSOCallbackPath = "/mdm/sso/callback"

	// FleetPayloadIdentifier is the value for the "<key>PayloadIdentifier</key>"
	// used by Fleet MDM on the enrollment profile.
	FleetPayloadIdentifier = "com.fleetdm.fleet.mdm.apple"

	// SCEPProxyPath is the HTTP path that serves the SCEP proxy service. The path is followed by identifier.
	SCEPProxyPath = "/mdm/scep/proxy/"
)

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
	logger     kitlog.Logger
}

// getDefaultProfile returns a godep.Profile with default values set.
func (d *DEPService) getDefaultProfile() *godep.Profile {
	return &godep.Profile{
		ProfileName:      "Fleet default enrollment profile",
		AllowPairing:     true,
		AutoAdvanceSetup: false,
		IsSupervised:     false,
		IsMultiUser:      false,
		IsMandatory:      false,
		IsMDMRemovable:   true,
		Language:         "en",
		OrgMagic:         "1",
		Region:           "US",
		SkipSetupItems: []string{
			"Accessibility",
			"Appearance",
			"AppleID",
			"AppStore",
			"Biometric",
			"Diagnostics",
			"FileVault",
			"iCloudDiagnostics",
			"iCloudStorage",
			"Location",
			"Payment",
			"Privacy",
			"Restore",
			"ScreenTime",
			"Siri",
			"TermsOfAddress",
			"TOS",
			"UnlockWithWatch",
		},
	}
}

// createDefaultAutomaticProfile creates the default automatic (DEP) enrollment
// profile in mdm_apple_enrollment_profiles but does not register it with
// Apple. It also creates the authentication token to get enrollment profiles.
func (d *DEPService) createDefaultAutomaticProfile(ctx context.Context) error {
	depProfile := d.getDefaultProfile()
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
		// flow, otherwise use Fleet's enroll URL.
		//
		// Even though the DEP profile supports an `url` attribute, we should
		// always still set configuration_web_url, otherwise the request method
		// coming from Apple changes from GET to POST, and we want to preserve
		// backwards compatibility.
		jsonProf.ConfigurationWebURL = enrollURL
		endUserAuthEnabled := appCfg.MDM.MacOSSetup.EnableEndUserAuthentication
		if team != nil {
			endUserAuthEnabled = team.Config.MDM.MacOSSetup.EnableEndUserAuthentication
		}
		if endUserAuthEnabled {
			jsonProf.ConfigurationWebURL = appCfg.MDMUrl() + "/mdm/sso"
		}
	}

	// ensure `url` is the same as `configuration_web_url`, to not leak the URL
	// to get a token without SSO enabled
	jsonProf.URL = jsonProf.ConfigurationWebURL
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
		d.logger.Log("msg", "skipping defining profile for team with no relevant ABM token")
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
				return ctxerr.Errorf(ctx, "Couldn't upload. %s", string(httpErr.Body))
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
		d.logger.Log("msg", "default DEP profile not set, registering")
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
	syncerLogger := logging.NewNanoDEPLogger(kitlog.With(d.logger, "component", "nanodep-syncer"))
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
				d.logger.Log("msg", "clearing device syncer cursor", "org_name", token.OrganizationName)
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
		)

		if err := syncer.Run(ctx); err != nil {
			result = multierror.Append(result, err)
			continue
		}
	}

	return result
}

func NewDEPService(
	ds fleet.Datastore,
	depStorage nanodep_storage.AllDEPStorage,
	logger kitlog.Logger,
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
		level.Debug(d.logger).Log(
			"msg", "device",
			"serial_number", device.SerialNumber,
			"device_assigned_by", device.DeviceAssignedBy,
			"device_assigned_date", device.DeviceAssignedDate,
			"op_date", device.OpDate,
			"op_type", device.OpType,
			"profile_assign_time", device.ProfileAssignTime,
			"push_push_time", device.ProfilePushTime,
			"profile_uuid", device.ProfileUUID,
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
			level.Warn(d.logger).Log(
				"msg", "unrecognized op_type",
				"op_type", device.OpType,
				"serial_number", device.SerialNumber,
			)
		}
	}

	// Remove added/modified devices if they have been subsequently deleted
	// Remove deleted devices if they have been subsequently added (or re-added)
	for _, deletedDevice := range deletedDevices {
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

	for _, addedDevice := range addedDevices {
		addedDevicesSlice = append(addedDevicesSlice, addedDevice)
	}
	for _, modifiedDevice := range modifiedDevices {
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
	for _, d := range modifiedDevices {
		if _, ok := existingSerials[d.SerialNumber]; !ok {
			addedDevicesSlice = append(addedDevicesSlice, d)
		}
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
		level.Error(kitlog.With(d.logger)).Log("err", err)
		ctxerr.Handle(ctx, err)
	case n > 0:
		level.Info(kitlog.With(d.logger)).Log("msg", fmt.Sprintf("added %d new mdm device(s) to pending hosts", n))
	case n == 0:
		level.Debug(kitlog.With(d.logger)).Log("msg", "no DEP hosts to add")
	}

	level.Debug(kitlog.With(d.logger)).Log("msg", "devices to assign DEP profiles", "to_add", len(addedDevicesSlice), "to_remove",
		strings.Join(deletedSerials, ", "), "to_modify", strings.Join(modifiedSerials, ", "))

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
		case "iPhone":
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
	for _, existingHost := range existingSerials {
		dd, ok := modifiedDevices[existingHost.HardwareSerial]
		if !ok {
			level.Error(kitlog.With(d.logger)).Log("msg",
				"serial coming from ABM is in the database, but it's not in the list of modified devices", "serial",
				existingHost.HardwareSerial)
			continue
		}
		existingHosts = append(existingHosts, *existingHost)
		devicesByTeam[existingHost.TeamID] = append(devicesByTeam[existingHost.TeamID], dd)
	}

	// assign the profile to each device
	for team, devices := range devicesByTeam {
		profUUID, err := d.getProfileUUIDForTeam(ctx, team, abmOrganizationName)
		if err != nil {
			return ctxerr.Wrapf(ctx, err, "getting profile for team with id: %v", team)
		}

		profileToDevices[profUUID] = append(profileToDevices[profUUID], devices...)
	}

	if len(existingHosts) > 0 {
		if err := d.ds.UpsertMDMAppleHostDEPAssignments(ctx, existingHosts, abmTokenID); err != nil {
			return ctxerr.Wrap(ctx, err, "upserting dep assignment for existing devices")
		}
	}

	// keep track of the serials we're going to skip for all profiles in
	// order to log them later.
	var skippedSerials []string
	for profUUID, devices := range profileToDevices {
		var serials []string
		for _, device := range devices {
			_, ok := existingDeletedSerials[device.SerialNumber]
			if device.ProfileUUID == profUUID && !ok {
				skippedSerials = append(skippedSerials, device.SerialNumber)
				continue
			}
			serials = append(serials, device.SerialNumber)
		}

		if len(serials) == 0 {
			continue
		}

		logger := kitlog.With(d.logger, "profile_uuid", profUUID)

		skipSerials, assignSerials, err := d.ds.ScreenDEPAssignProfileSerialsForCooldown(ctx, serials)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "process device response")
		}
		if len(skipSerials) > 0 {
			// NOTE: the `dep_cooldown` job of the `integrations`` cron picks up the assignments
			// after the cooldown period is over
			level.Debug(logger).Log("msg", "process device response: skipping assign profile for devices on cooldown", "serials", fmt.Sprintf("%s", skipSerials))
		}
		if len(assignSerials) == 0 {
			level.Debug(logger).Log("msg", "process device response: no devices to assign profile")
			continue
		}

		for orgName, serials := range assignSerials {
			apiResp, err := d.depClient.AssignProfile(ctx, orgName, profUUID, serials...)
			if err != nil {
				// only log the error so the failure can be recorded
				// below in UpdateHostDEPAssignProfileResponses and
				// the proper cooldowns are applied
				level.Error(logger).Log(
					"msg", "assign profile",
					"devices", len(assignSerials),
					"err", err,
				)
			}

			logs := []interface{}{
				"msg", "profile assigned",
				"devices", len(assignSerials),
			}
			logs = append(logs, logCountsForResults(apiResp.Devices)...)
			level.Info(logger).Log(logs...)

			if err := d.ds.UpdateHostDEPAssignProfileResponses(ctx, apiResp, abmTokenID); err != nil {
				return ctxerr.Wrap(ctx, err, "update host dep assign profile responses")
			}
		}
	}

	if len(skippedSerials) > 0 {
		level.Debug(kitlog.With(d.logger)).Log("msg", "found devices that already have the right profile, skipping assignment", "serials", fmt.Sprintf("%s", skippedSerials))
	}

	return nil
}

func (d *DEPService) getProfileUUIDForTeam(ctx context.Context, tmID *uint, abmTokenOrgName string) (string, error) {
	var appleBMTeam *fleet.Team
	if tmID != nil {
		tm, err := d.ds.Team(ctx, *tmID)
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
	results := map[string]int{"success": 0, "not_accessible": 0, "failed": 0, "other": 0}
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
// changes.
func NewDEPClient(storage godep.ClientStorage, updater fleet.ABMTermsUpdater, logger kitlog.Logger) *godep.Client {
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

		// if the request failed due to terms not signed, or if it succeeded,
		// update the ABM token's (and possibly the app config's) flag accordingly.
		// If it failed for any other reason, do not update the flag.
		termsExpired := reqErr != nil && godep.IsTermsNotSigned(reqErr)
		if reqErr == nil || termsExpired {
			// get the count of tokens with the flag still set
			count, err := updater.CountABMTokensWithTermsExpired(ctx)
			if err != nil {
				level.Error(logger).Log("msg", "Apple DEP client: failed to get count of tokens with terms expired", "err", err)
				return reqErr
			}

			// get the appconfig for the global flag
			appCfg, err := updater.AppConfig(ctx)
			if err != nil {
				level.Error(logger).Log("msg", "Apple DEP client: failed to get app config", "err", err)
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
				level.Error(logger).Log("msg", "Apple DEP client: failed to update terms expired of ABM token", "err", err)
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
					level.Error(logger).Log("msg", "Apple DEP client: failed to save app config", "err", err)
				}
				level.Info(logger).Log("msg", "Apple DEP client: updated app config Terms Expired flag",
					"apple_bm_terms_expired", appCfg.MDM.AppleBMTermsExpired)
			}
		}
		return reqErr
	}))
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
					<array><array><string>CN</string><string>Fleet Identity</string></array></array>
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
	<string>System</string>
	<key>PayloadType</key>
	<string>Configuration</string>
	<key>PayloadUUID</key>
	<string>5ACABE91-CE30-4C05-93E3-B235C152404E</string>
	<key>PayloadVersion</key>
	<integer>1</integer>
</dict>
</plist>`))

func GenerateEnrollmentProfileMobileconfig(orgName, fleetURL, scepChallenge, topic string) ([]byte, error) {
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

func IOSiPadOSRefetch(ctx context.Context, ds fleet.Datastore, commander *MDMAppleCommander, logger kitlog.Logger) error {
	appCfg, err := ds.AppConfig(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "fetching app config")
	}

	if !appCfg.MDM.EnabledAndConfigured {
		level.Debug(logger).Log("msg", "apple mdm is not configured, skipping run")
		return nil
	}

	start := time.Now()
	devices, err := ds.ListIOSAndIPadOSToRefetch(ctx, 1*time.Hour)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "list ios and ipad devices to refetch")
	}
	if len(devices) == 0 {
		return nil
	}
	logger.Log("msg", "sending commands to refetch", "count", len(devices), "lookup-duration", time.Since(start))
	commandUUID := uuid.NewString()

	hostMDMCommands := make([]fleet.HostMDMCommand, 0, 2*len(devices))
	installedAppsUUIDs := make([]string, 0, len(devices))
	for _, device := range devices {
		if !slices.Contains(device.CommandsAlreadySent, fleet.RefetchAppsCommandUUIDPrefix) {
			installedAppsUUIDs = append(installedAppsUUIDs, device.UUID)
			hostMDMCommands = append(hostMDMCommands, fleet.HostMDMCommand{
				HostID:      device.HostID,
				CommandType: fleet.RefetchAppsCommandUUIDPrefix,
			})
		}
	}
	if len(installedAppsUUIDs) > 0 {
		err = commander.InstalledApplicationList(ctx, installedAppsUUIDs, fleet.RefetchAppsCommandUUIDPrefix+commandUUID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "send InstalledApplicationList commands to ios and ipados devices")
		}
	}

	// DeviceInformation is last because the refetch response clears the refetch_requested flag
	deviceInfoUUIDs := make([]string, 0, len(devices))
	for _, device := range devices {
		if !slices.Contains(device.CommandsAlreadySent, fleet.RefetchDeviceCommandUUIDPrefix) {
			deviceInfoUUIDs = append(deviceInfoUUIDs, device.UUID)
			hostMDMCommands = append(hostMDMCommands, fleet.HostMDMCommand{
				HostID:      device.HostID,
				CommandType: fleet.RefetchDeviceCommandUUIDPrefix,
			})
		}
	}
	if len(deviceInfoUUIDs) > 0 {
		if err := commander.DeviceInformation(ctx, deviceInfoUUIDs, fleet.RefetchDeviceCommandUUIDPrefix+commandUUID); err != nil {
			return ctxerr.Wrap(ctx, err, "send DeviceInformation commands to ios and ipados devices")
		}
	}

	// Add commands to the database to track the commands sent
	err = ds.AddHostMDMCommands(ctx, hostMDMCommands)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "add host mdm commands")
	}
	return nil
}

func GenerateOTAEnrollmentProfileMobileconfig(orgName, fleetURL, enrollSecret string) ([]byte, error) {
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
