package apple_mdm

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"net/url"
	"path"
	"strings"
	"text/template"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/logging"
	"github.com/getsentry/sentry-go"
	"github.com/go-kit/log/level"
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

	// FleetFileVaultPayloadIdentifier is the value for the PayloadIdentifier
	// used by Fleet to configure FileVault and FileVault Escrow.
	FleetFileVaultPayloadIdentifier = "com.fleetdm.fleet.mdm.filevault"

	// FleetdConfigPayloadIdentifier is the value for the PayloadIdentifier used
	// by fleetd to read configuration values from the system.
	FleetdConfigPayloadIdentifier = "com.fleetdm.fleetd.config"
)

// ProfilesManagedByFleet returns a list of profile identifiers
// that are handled and delivered by Fleet.
func ProfilesManagedByFleet() []string {
	return []string{
		FleetPayloadIdentifier,
		FleetFileVaultPayloadIdentifier,
		FleetdConfigPayloadIdentifier,
	}
}

func ResolveAppleMDMURL(serverURL string) (string, error) {
	return resolveURL(serverURL, MDMPath)
}

func ResolveAppleEnrollMDMURL(serverURL string) (string, error) {
	return resolveURL(serverURL, EnrollPath)
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

type DEPSyncer struct {
	depStorage nanodep_storage.AllStorage
	syncer     *depsync.Syncer
	logger     kitlog.Logger
}

func (d *DEPSyncer) Run(ctx context.Context) error {
	profileUUID, profileModTime, err := d.depStorage.RetrieveAssignerProfile(ctx, DEPName)
	if err != nil {
		return err
	}
	if profileUUID == "" {
		d.logger.Log("msg", "DEP profile not set, nothing to do")
		return nil
	}
	cursor, cursorModTime, err := d.depStorage.RetrieveCursor(ctx, DEPName)
	if err != nil {
		return err
	}
	// If the DEP Profile was changed since last sync then we clear
	// the cursor and perform a full sync of all devices and profile assigning.
	if cursor != "" && profileModTime.After(cursorModTime) {
		d.logger.Log("msg", "clearing device syncer cursor")
		if err := d.depStorage.StoreCursor(ctx, DEPName, ""); err != nil {
			return err
		}
	}
	return d.syncer.Run(ctx)
}

func NewDEPSyncer(
	ds fleet.Datastore,
	depStorage nanodep_storage.AllStorage,
	logger kitlog.Logger,
	loggingDebug bool,
) *DEPSyncer {
	depClient := NewDEPClient(depStorage, ds, logger)
	assignerOpts := []depsync.AssignerOption{
		depsync.WithAssignerLogger(logging.NewNanoDEPLogger(kitlog.With(logger, "component", "nanodep-assigner"))),
	}
	if loggingDebug {
		assignerOpts = append(assignerOpts, depsync.WithDebug())
	}
	assigner := depsync.NewAssigner(
		depClient,
		DEPName,
		depStorage,
		assignerOpts...,
	)

	syncer := depsync.NewSyncer(
		depClient,
		DEPName,
		depStorage,
		depsync.WithLogger(logging.NewNanoDEPLogger(kitlog.With(logger, "component", "nanodep-syncer"))),
		depsync.WithCallback(func(ctx context.Context, isFetch bool, resp *godep.DeviceResponse) error {
			n, err := ds.IngestMDMAppleDevicesFromDEPSync(ctx, resp.Devices)
			switch {
			case err != nil:
				level.Error(kitlog.With(logger)).Log("err", err)
				sentry.CaptureException(err)
			case n > 0:
				level.Info(kitlog.With(logger)).Log("msg", fmt.Sprintf("added %d new mdm device(s) to pending hosts", n))
			case n == 0:
				level.Info(kitlog.With(logger)).Log("msg", "no DEP hosts to add")
			}

			return assigner.ProcessDeviceResponse(ctx, resp)
		}),
	)

	return &DEPSyncer{
		syncer:     syncer,
		depStorage: depStorage,
		logger:     logger,
	}
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
	<string>{{ .Organization }} Enrollment</string>
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
