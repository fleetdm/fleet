package apple_mdm

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"net/url"
	"path"
	"strings"
	"text/template"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/logging"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/getsentry/sentry-go"
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

	// FleetPayloadIdentifier is the value for the "<key>PayloadIdentifier</key>"
	// used by Fleet MDM on the enrollment profile.
	FleetPayloadIdentifier = "com.fleetdm.fleet.mdm.apple"

	// FleetdPublicManifestURL contains a valid manifest that can be used
	// by InstallEnterpriseApplication to install `fleetd` in a host.
	FleetdPublicManifestURL = "https://download.fleetdm.com/fleetd-base-manifest.plist"
)

func ResolveAppleMDMURL(serverURL string) (string, error) {
	return resolveURL(serverURL, MDMPath)
}

func ResolveAppleSCEPURL(serverURL string) (string, error) {
	return resolveURL(serverURL, SCEPPath)
}

func resolveURL(serverURL, relPath string) (string, error) {
	u, err := url.Parse(serverURL)
	if err != nil {
		return "", err
	}
	u.Path = path.Join(u.Path, relPath)
	return u.String(), nil
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

// CreateDefaultProfile creates a new DEP enrollment profile with default
// values in the database and registers it in Apple's servers.
func (d *DEPService) CreateDefaultProfile(ctx context.Context) error {
	if err := d.createProfile(ctx, d.getDefaultProfile()); err != nil {
		return ctxerr.Wrap(ctx, err, "creating profile")
	}
	return nil
}

// createProfile creates a new DEP enrollment profile with the provided values
// in the database and registers it in Apple's servers.
//
// All valid values are declared in the godep.Profile type and are specified in
// https://developer.apple.com/documentation/devicemanagement/profile
func (d *DEPService) createProfile(ctx context.Context, depProfile *godep.Profile) error {
	token := uuid.New().String()
	rawDEPProfile, err := json.Marshal(depProfile)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "marshaling provided profile")
	}

	payload := fleet.MDMAppleEnrollmentProfilePayload{
		Token:      token,
		Type:       fleet.MDMAppleEnrollmentTypeAutomatic,
		DEPProfile: ptr.RawMessage(rawDEPProfile),
	}
	if _, err := d.ds.NewMDMAppleEnrollmentProfile(ctx, payload); err != nil {
		return ctxerr.Wrap(ctx, err, "saving enrollment profile in DB")
	}

	if err := d.RegisterProfileWithAppleDEPServer(ctx, nil); err != nil {
		return ctxerr.Wrap(ctx, err, "registering profile in Apple servers")
	}

	return nil
}

// RegisterProfileWithAppleDEPServer registers the enrollment profile in
// Apple's servers via the DEP API, so it can be used for assignment. If
// setupAsst is nil, the default profile is registered. It assigns the
// up-to-date dynamic settings such as the server URL and MDM SSO URL.
func (d *DEPService) RegisterProfileWithAppleDEPServer(ctx context.Context, setupAsst *fleet.MDMAppleSetupAssistant) error {
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

	var jsonProf *godep.Profile
	if err := json.Unmarshal(rawJSON, &jsonProf); err != nil {
		return ctxerr.Wrap(ctx, err, "unmarshalling DEP profile")
	}

	jsonProf.URL = enrollURL

	// If SSO is configured, use the `/mdm/sso` page which starts the SSO
	// flow, otherwise use Fleet's enroll URL.
	//
	// Even though the DEP profile supports an `url` attribute, we should
	// always still set configuration_web_url, otherwise the request method
	// coming from Apple changes from GET to POST, and we want to preserve
	// backwards compatibility.
	jsonProf.ConfigurationWebURL = enrollURL
	if !appCfg.MDM.EndUserAuthentication.SSOProviderSettings.IsEmpty() {
		// TODO: modify method signatures for this (and callers as applicable)
		// to include a team config pointer and check enable_end_user_authenthication
		// in the team config if not nil otherwise check enable_end_user_authenthication
		// in the app config.
		jsonProf.ConfigurationWebURL = appCfg.ServerSettings.ServerURL + "/mdm/sso"
	}

	depClient := NewDEPClient(d.depStorage, d.ds, d.logger)
	res, err := depClient.DefineProfile(ctx, DEPName, jsonProf)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "apple POST /profile request failed")
	}

	if setupAsst != nil {
		setupAsst.ProfileUUID = res.ProfileUUID
		if err := d.ds.SetMDMAppleSetupAssistantProfileUUID(ctx, setupAsst.TeamID, res.ProfileUUID); err != nil {
			return ctxerr.Wrap(ctx, err, "save setup assistant profile UUID")
		}
	} else {
		// for backwards compatibility, we store the profile UUID of the default
		// profile in the nanomdm storage.
		if err := d.depStorage.StoreAssignerProfile(ctx, DEPName, res.ProfileUUID); err != nil {
			return ctxerr.Wrap(ctx, err, "save default profile UUID")
		}
	}

	return nil
}

// EnsureDefaultSetupAssistant ensures that the default Setup Assistant profile
// is created and registered with Apple, and returns its profile UUID. It does
// not re-define the profile if it already exists.
func (d *DEPService) EnsureDefaultSetupAssistant(ctx context.Context) (string, time.Time, error) {
	profileUUID, profileModTime, err := d.depStorage.RetrieveAssignerProfile(ctx, DEPName)
	if err != nil {
		return "", time.Time{}, err
	}
	if profileUUID == "" {
		d.logger.Log("msg", "default DEP profile not set, creating")
		if err := d.CreateDefaultProfile(ctx); err != nil {
			return "", time.Time{}, err
		}
		profileUUID, profileModTime, err = d.depStorage.RetrieveAssignerProfile(ctx, DEPName)
		if err != nil {
			return "", time.Time{}, err
		}
	}
	return profileUUID, profileModTime, nil
}

// EnsureCustomSetupAssistantIfExists ensures that the custom Setup Assistant
// profile associated with the provided team (or no team) is registered with
// Apple, and returns its profile UUID. It does not re-define the profile if it
// is already registered. If no custom setup assistant exists, it returns an
// empty string and timestamp and no error.
func (d *DEPService) EnsureCustomSetupAssistantIfExists(ctx context.Context, tmID *uint) (string, time.Time, error) {
	asst, err := d.ds.GetMDMAppleSetupAssistant(ctx, tmID)
	if err != nil {
		if fleet.IsNotFound(err) {
			// no error, no custom setup assistant for that team
			return "", time.Time{}, nil
		}
		return "", time.Time{}, err
	}

	if asst.ProfileUUID == "" {
		if err := d.RegisterProfileWithAppleDEPServer(ctx, asst); err != nil {
			return "", time.Time{}, err
		}
	}
	return asst.ProfileUUID, asst.UploadedAt, nil
}

func (d *DEPService) RunAssigner(ctx context.Context) error {
	// ensure the default (fallback) setup assistant profile exists, registered
	// with Apple DEP.
	_, defModTime, err := d.EnsureDefaultSetupAssistant(ctx)
	if err != nil {
		return err
	}

	// get the Apple BM default team and if it has a custom setup assistant,
	// ensure it is registered with Apple DEP.
	appCfg, err := d.ds.AppConfig(ctx)
	if err != nil {
		return err
	}
	var customTeamID *uint
	if appCfg.MDM.AppleBMDefaultTeam != "" {
		tm, err := d.ds.TeamByName(ctx, appCfg.MDM.AppleBMDefaultTeam)
		// NOTE: TeamByName does NOT return a not found error if it does not exist
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if tm != nil {
			customTeamID = &tm.ID
		}
	}
	customUUID, customModTime, err := d.EnsureCustomSetupAssistantIfExists(ctx, customTeamID)
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
	loggingDebug bool,
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
			n, teamID, err := ds.IngestMDMAppleDevicesFromDEPSync(ctx, resp.Devices)
			switch {
			case err != nil:
				level.Error(kitlog.With(logger)).Log("err", err)
				sentry.CaptureException(err)
			case n > 0:
				level.Info(kitlog.With(logger)).Log("msg", fmt.Sprintf("added %d new mdm device(s) to pending hosts", n))
			case n == 0:
				level.Info(kitlog.With(logger)).Log("msg", "no DEP hosts to add")
			}

			// at this point, the hosts rows are created for the devices, with the
			// correct team_id, so we know what team-specific profile needs to be applied.
			return depSvc.processDeviceResponse(ctx, depClient, resp, teamID)
		}),
	)

	return depSvc
}

// processDeviceResponse processes the device response from the device sync
// DEP API endpoints and assigns the profile UUID associated with the DEP
// client DEP name.
func (d *DEPService) processDeviceResponse(ctx context.Context, depClient *godep.Client, resp *godep.DeviceResponse, tmID *uint) error {
	if len(resp.Devices) < 1 {
		// no devices means we can't assign anything
		return nil
	}

	// get profile uuid of tmID or default
	profUUID, _, err := d.EnsureCustomSetupAssistantIfExists(ctx, tmID)
	if err != nil {
		return fmt.Errorf("ensure setup assistant for team %v: %w", tmID, err)
	}
	if profUUID == "" {
		profUUID, _, err = d.EnsureDefaultSetupAssistant(ctx)
		if err != nil {
			return fmt.Errorf("ensure default setup assistant: %w", err)
		}
	}

	if profUUID == "" {
		level.Debug(d.logger).Log("msg", "empty assigner profile UUID")
		return nil
	}

	var serials []string
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
		// We currently only listen for an op_type of "added", the other
		// op_types are ambiguous and it would be needless to assign the
		// profile UUID every single time we get an update.
		if strings.ToLower(device.OpType) == "added" ||
			// The op_type field is only applicable with the SyncDevices API call,
			// Empty op_type come from the first call to FetchDevices without a cursor,
			// and we do want to assign profiles to them.
			strings.ToLower(device.OpType) == "" {
			serials = append(serials, device.SerialNumber)
		}
	}

	logger := kitlog.With(d.logger, "profile_uuid", profUUID)

	if len(serials) < 1 {
		level.Debug(logger).Log(
			"msg", "no serials to assign",
			"devices", len(resp.Devices),
		)
		return nil
	}

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

	return nil
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
				level.Debug(logger).Log("msg", "Apple DEP client: updated app config Terms Expired flag",
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
