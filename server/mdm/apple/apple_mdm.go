package apple_mdm

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/url"
	"strings"
	"text/template"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/logging"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	"github.com/fleetdm/fleet/v4/server/mdm/internal/commonmdm"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/go-kit/log/level"
	"github.com/google/uuid"
	"github.com/micromdm/nanodep/godep"

	kitlog "github.com/go-kit/kit/log"
	nanodep_storage "github.com/micromdm/nanodep/storage"
	depsync "github.com/micromdm/nanodep/sync"
)

// DEPName is the identifier/name used in nanodep MySQL storage which
// holds the DEP configuration.
//
// Fleet uses only one DEP configuration set for the whole deployment.
const DEPName = "fleet"

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

	// FleetdPublicManifestURL contains a valid manifest that can be used
	// by InstallEnterpriseApplication to install `fleetd` in a host.
	FleetdPublicManifestURL = "https://download.fleetdm.com/fleetd-base-manifest.plist"
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
	depStorage nanodep_storage.AllStorage
	syncer     *depsync.Syncer
	logger     kitlog.Logger
}

// getDefaultProfile returns a godep.Profile with default values set.
func (d *DEPService) getDefaultProfile() *godep.Profile {
	return &godep.Profile{
		ProfileName:           "FleetDM default enrollment profile",
		AllowPairing:          true,
		AutoAdvanceSetup:      false,
		AwaitDeviceConfigured: false,
		IsSupervised:          false,
		IsMultiUser:           false,
		IsMandatory:           false,
		IsMDMRemovable:        true,
		Language:              "en",
		OrgMagic:              "1",
		Region:                "US",
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

// RegisterProfileWithAppleDEPServer registers the enrollment profile in
// Apple's servers via the DEP API, so it can be used for assignment. If
// setupAsst is nil, the default profile is registered. It assigns the
// up-to-date dynamic settings such as the server URL and MDM SSO URL if
// end-user authentication is enabled for that team/no-team.
func (d *DEPService) RegisterProfileWithAppleDEPServer(ctx context.Context, team *fleet.Team, setupAsst *fleet.MDMAppleSetupAssistant) error {
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

	var rawJSON json.RawMessage
	if defaultProf.DEPProfile != nil {
		rawJSON = *defaultProf.DEPProfile
	}
	if setupAsst != nil {
		rawJSON = setupAsst.Profile
	}

	var jsonProf godep.Profile
	jsonProf.IsMDMRemovable = true // the default value defined by Apple is true
	if err := json.Unmarshal(rawJSON, &jsonProf); err != nil {
		return ctxerr.Wrap(ctx, err, "unmarshalling DEP profile")
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
			jsonProf.ConfigurationWebURL = appCfg.ServerSettings.ServerURL + "/mdm/sso"
		}
	}

	// ensure `url` is the same as `configuration_web_url`, to not leak the URL
	// to get a token without SSO enabled
	jsonProf.URL = jsonProf.ConfigurationWebURL

	depClient := NewDEPClient(d.depStorage, d.ds, d.logger)
	res, err := depClient.DefineProfile(ctx, DEPName, &jsonProf)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "apple POST /profile request failed")
	}

	if setupAsst != nil {
		setupAsst.ProfileUUID = res.ProfileUUID
		if err := d.ds.SetMDMAppleSetupAssistantProfileUUID(ctx, setupAsst.TeamID, res.ProfileUUID); err != nil {
			return ctxerr.Wrap(ctx, err, "save setup assistant profile UUID")
		}
	} else {
		var tmID *uint
		if team != nil {
			tmID = &team.ID
		}
		if err := d.ds.SetMDMAppleDefaultSetupAssistantProfileUUID(ctx, tmID, res.ProfileUUID); err != nil {
			return ctxerr.Wrap(ctx, err, "save default setup assistant profile UUID")
		}
	}
	return nil
}

// EnsureDefaultSetupAssistant ensures that the default Setup Assistant profile
// is created and registered with Apple for the provided team/no-team (if team
// is nil), and returns its profile UUID. It does not re-define the profile if
// it already exists and registered.
func (d *DEPService) EnsureDefaultSetupAssistant(ctx context.Context, team *fleet.Team) (string, time.Time, error) {
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
	// check if the default profile was registered with Apple for that team.
	var tmID *uint
	if team != nil {
		tmID = &team.ID
	}
	profUUID, modTime, err := d.ds.GetMDMAppleDefaultSetupAssistant(ctx, tmID)
	if err != nil && !fleet.IsNotFound(err) {
		return "", time.Time{}, ctxerr.Wrap(ctx, err, "get default setup assistant profile uuid")
	}
	if profUUID == "" {
		d.logger.Log("msg", "default DEP profile not set, registering")
		if err := d.RegisterProfileWithAppleDEPServer(ctx, team, nil); err != nil {
			return "", time.Time{}, ctxerr.Wrap(ctx, err, "register default setup assistant with Apple")
		}
		profUUID, modTime, err = d.ds.GetMDMAppleDefaultSetupAssistant(ctx, tmID)
		if err != nil {
			return "", time.Time{}, ctxerr.Wrap(ctx, err, "get default setup assistant profile uuid after registering")
		}
	}
	return profUUID, modTime, nil
}

// EnsureCustomSetupAssistantIfExists ensures that the custom Setup Assistant
// profile associated with the provided team (or no team) is registered with
// Apple, and returns its profile UUID. It does not re-define the profile if it
// is already registered. If no custom setup assistant exists, it returns an
// empty string and timestamp and no error.
func (d *DEPService) EnsureCustomSetupAssistantIfExists(ctx context.Context, team *fleet.Team) (string, time.Time, error) {
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

	if asst.ProfileUUID == "" {
		if err := d.RegisterProfileWithAppleDEPServer(ctx, team, asst); err != nil {
			return "", time.Time{}, err
		}
	}
	return asst.ProfileUUID, asst.UploadedAt, nil
}

func (d *DEPService) RunAssigner(ctx context.Context) error {
	// get the Apple BM default team
	appCfg, err := d.ds.AppConfig(ctx)
	if err != nil {
		return err
	}
	var appleBMTeam *fleet.Team
	if appCfg.MDM.AppleBMDefaultTeam != "" {
		tm, err := d.ds.TeamByName(ctx, appCfg.MDM.AppleBMDefaultTeam)
		if err != nil && !fleet.IsNotFound(err) {
			return err
		}
		appleBMTeam = tm
	}

	// ensure the default (fallback) setup assistant profile exists, registered
	// with Apple DEP.
	_, defModTime, err := d.EnsureDefaultSetupAssistant(ctx, appleBMTeam)
	if err != nil {
		return err
	}

	// if the team/no-team has a custom setup assistant, ensure it is registered
	// with Apple DEP.
	customUUID, customModTime, err := d.EnsureCustomSetupAssistantIfExists(ctx, appleBMTeam)
	if err != nil {
		return err
	}

	// get the modification timestamp of the effective profile (custom or default)
	effectiveProfModTime := defModTime
	if customUUID != "" {
		effectiveProfModTime = customModTime
	}

	cursor, cursorModTime, err := d.depStorage.RetrieveCursor(ctx, DEPName)
	if err != nil {
		return err
	}

	// If the effective profile was changed since last sync then we clear
	// the cursor and perform a full sync of all devices and profile assigning.
	if cursor != "" && effectiveProfModTime.After(cursorModTime) {
		d.logger.Log("msg", "clearing device syncer cursor")
		if err := d.depStorage.StoreCursor(ctx, DEPName, ""); err != nil {
			return err
		}
	}
	return d.syncer.Run(ctx)
}

func NewDEPService(
	ds fleet.Datastore,
	depStorage nanodep_storage.AllStorage,
	logger kitlog.Logger,
) *DEPService {
	depClient := NewDEPClient(depStorage, ds, logger)
	depSvc := &DEPService{
		depStorage: depStorage,
		logger:     logger,
		ds:         ds,
	}

	depSvc.syncer = depsync.NewSyncer(
		depClient,
		DEPName,
		depStorage,
		depsync.WithLogger(logging.NewNanoDEPLogger(kitlog.With(logger, "component", "nanodep-syncer"))),
		depsync.WithCallback(func(ctx context.Context, isFetch bool, resp *godep.DeviceResponse) error {
			// the nanodep syncer just logs the error of the callback, so in order to
			// capture it we need to do this here.
			err := depSvc.processDeviceResponse(ctx, depClient, resp)
			if err != nil {
				ctxerr.Handle(ctx, err)
			}
			return err
		}),
	)

	return depSvc
}

// processDeviceResponse processes the device response from the device sync
// DEP API endpoints and assigns the profile UUID associated with the DEP
// client DEP name.
func (d *DEPService) processDeviceResponse(ctx context.Context, depClient *godep.Client, resp *godep.DeviceResponse) error {
	if len(resp.Devices) < 1 {
		// no devices means we can't assign anything
		return nil
	}

	var addedDevices []godep.Device
	var deletedSerials []string
	var modifiedDevices []godep.Device
	var modifiedSerials []string
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
			addedDevices = append(addedDevices, device)
		case "modified":
			modifiedDevices = append(modifiedDevices, device)
			modifiedSerials = append(modifiedSerials, device.SerialNumber)
		case "deleted":
			deletedSerials = append(deletedSerials, device.SerialNumber)
		default:
			level.Warn(d.logger).Log(
				"msg", "unrecognized op_type",
				"op_type", device.OpType,
				"serial_number", device.SerialNumber,
			)
		}
	}

	existingSerials, err := d.ds.GetMatchingHostSerials(ctx, modifiedSerials)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "get matching host serials")
	}

	// treat device that's coming as "modified" but doesn't exist in the
	// `hosts` table, as an "added" device.
	for _, d := range modifiedDevices {
		if _, ok := existingSerials[d.SerialNumber]; !ok {
			addedDevices = append(addedDevices, d)
		}
	}

	err = d.ds.DeleteHostDEPAssignments(ctx, deletedSerials)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "deleting DEP assignments")
	}

	n, defaultABMTeamID, err := d.ds.IngestMDMAppleDevicesFromDEPSync(ctx, addedDevices)
	switch {
	case err != nil:
		level.Error(kitlog.With(d.logger)).Log("err", err)
		ctxerr.Handle(ctx, err)
	case n > 0:
		level.Info(kitlog.With(d.logger)).Log("msg", fmt.Sprintf("added %d new mdm device(s) to pending hosts", n))
	case n == 0:
		level.Info(kitlog.With(d.logger)).Log("msg", "no DEP hosts to add")
	}

	// at this point, the hosts rows are created for the devices, with the
	// correct team_id, so we know what team-specific profile needs to be applied.
	//
	// collect a map of all the profiles => serials we need to assign.
	profileToSerials := map[string][]string{}

	// each new device should be assigned the DEP profile of the default
	// ABM team as configured by the IT admin.
	if len(addedDevices) > 0 {
		level.Info(kitlog.With(d.logger)).Log("msg", "gathering added serials to assign devices", "len", len(addedDevices))
		profUUID, err := d.getProfileUUIDForTeam(ctx, defaultABMTeamID)
		if err != nil {
			return ctxerr.Wrapf(ctx, err, "getting profile for default team with id: %v", defaultABMTeamID)
		}

		var addedSerials []string
		for _, d := range addedDevices {
			addedSerials = append(addedSerials, d.SerialNumber)
		}
		profileToSerials[profUUID] = addedSerials
	} else {
		level.Info(kitlog.With(d.logger)).Log("msg", "no added devices to assign DEP profiles")
	}

	// for all other hosts we received, find out the right DEP profile to assign, based on the team.
	if len(existingSerials) > 0 {
		level.Info(kitlog.With(d.logger)).Log("msg", "gathering existing serials to assign devices", "len", len(existingSerials))
		serialsByTeam := map[*uint][]string{}
		hosts := []fleet.Host{}
		for _, host := range existingSerials {
			if serialsByTeam[host.TeamID] == nil {
				serialsByTeam[host.TeamID] = []string{}
			}
			serialsByTeam[host.TeamID] = append(serialsByTeam[host.TeamID], host.HardwareSerial)
			hosts = append(hosts, *host)
		}
		for team, serials := range serialsByTeam {
			profUUID, err := d.getProfileUUIDForTeam(ctx, team)
			if err != nil {
				return ctxerr.Wrapf(ctx, err, "getting profile for team with id: %v", team)
			}
			if profileToSerials[profUUID] == nil {
				profileToSerials[profUUID] = []string{}
			}
			profileToSerials[profUUID] = append(profileToSerials[profUUID], serials...)

		}

		if err := d.ds.UpsertMDMAppleHostDEPAssignments(ctx, hosts); err != nil {
			return ctxerr.Wrap(ctx, err, "upserting dep assignment for existing device")
		}

	} else {
		level.Info(kitlog.With(d.logger)).Log("msg", "no existing devices to assign DEP profiles")
	}

	for profUUID, serials := range profileToSerials {
		logger := kitlog.With(d.logger, "profile_uuid", profUUID)
		level.Info(logger).Log("msg", "calling DEP client to assign profile", "profile_uuid", profUUID)
		apiResp, err := depClient.AssignProfile(ctx, DEPName, profUUID, serials...)
		if err != nil {
			level.Info(logger).Log(
				"msg", "assign profile",
				"devices", len(serials),
				"err", err,
			)
			return fmt.Errorf("assign profile: %w", err)
		}

		logs := []interface{}{
			"msg", "profile assigned",
			"devices", len(serials),
		}
		logs = append(logs, logCountsForResults(apiResp.Devices)...)
		level.Info(logger).Log(logs...)

		debugLogs := []interface{}{"msg", "assign profile responses by device"}
		for k, v := range apiResp.Devices {
			debugLogs = append(debugLogs, k, v)
		}
		level.Debug(logger).Log(debugLogs...)
	}

	return nil
}

func (d *DEPService) getProfileUUIDForTeam(ctx context.Context, tmID *uint) (string, error) {
	var appleBMTeam *fleet.Team
	if tmID != nil {
		tm, err := d.ds.Team(ctx, *tmID)
		if err != nil && !fleet.IsNotFound(err) {
			return "", ctxerr.Wrap(ctx, err, "get team")
		}
		appleBMTeam = tm
	}

	// get profile uuid of team or default
	profUUID, _, err := d.EnsureCustomSetupAssistantIfExists(ctx, appleBMTeam)
	if err != nil {
		return "", fmt.Errorf("ensure setup assistant for team %v: %w", tmID, err)
	}
	if profUUID == "" {
		profUUID, _, err = d.EnsureDefaultSetupAssistant(ctx, appleBMTeam)
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
// storage that will flag the AppConfig's AppleBMTermsExpired field
// whenever the status of the terms changes.
func NewDEPClient(storage godep.ClientStorage, appCfgUpdater fleet.AppConfigUpdater, logger kitlog.Logger) *godep.Client {
	return godep.NewClient(storage, fleethttp.NewClient(), godep.WithAfterHook(func(ctx context.Context, reqErr error) error {
		// if the request failed due to terms not signed, or if it succeeded,
		// update the app config flag accordingly. If it failed for any other
		// reason, do not update the flag.
		termsExpired := reqErr != nil && godep.IsTermsNotSigned(reqErr)
		if reqErr == nil || termsExpired {
			appCfg, err := appCfgUpdater.AppConfig(ctx)
			if err != nil {
				level.Error(logger).Log("msg", "Apple DEP client: failed to get app config", "err", err)
				return reqErr
			}

			var mustSaveAppCfg bool
			if termsExpired && !appCfg.MDM.AppleBMTermsExpired {
				// flag the AppConfig that the terms have changed and must be accepted
				appCfg.MDM.AppleBMTermsExpired = true
				mustSaveAppCfg = true
			} else if reqErr == nil && appCfg.MDM.AppleBMTermsExpired {
				// flag the AppConfig that the terms have been accepted
				appCfg.MDM.AppleBMTermsExpired = false
				mustSaveAppCfg = true
			}

			if mustSaveAppCfg {
				if err := appCfgUpdater.SaveAppConfig(ctx, appCfg); err != nil {
					level.Error(logger).Log("msg", "Apple DEP client: failed to save app config", "err", err)
				}
				level.Info(logger).Log("msg", "Apple DEP client: updated app config Terms Expired flag",
					"apple_bm_terms_expired", appCfg.MDM.AppleBMTermsExpired)
			}
		}
		return reqErr
	}))
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
	<string>{{ .Organization }} enrollment</string>
	<key>PayloadIdentifier</key>
	<string>` + FleetPayloadIdentifier + `</string>
	<key>PayloadOrganization</key>
	<string>{{ .Organization }}</string>
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

	var escaped strings.Builder
	if err := xml.EscapeText(&escaped, []byte(scepChallenge)); err != nil {
		return nil, fmt.Errorf("escape SCEP challenge for XML: %w", err)
	}

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
		SCEPChallenge: escaped.String(),
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
