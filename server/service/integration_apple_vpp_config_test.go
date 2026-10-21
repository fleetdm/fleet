package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	eeservice "github.com/fleetdm/fleet/v4/ee/server/service"
	"github.com/fleetdm/fleet/v4/pkg/mdm/mdmtest"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxdb"
	"github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/dev_mode"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android/service/androidmgmt"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/apple_apps"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/vpp"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/androidmanagement/v1"
)

func (s *integrationMDMTestSuite) TestVPPAppleManagedAppConfiguration() {
	t := s.T()
	s.setSkipWorkerJobs(t)
	ctx := context.Background()

	// VPP setup: token + team association.
	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "vpp-apple-config-team"})
	require.NoError(t, err)

	orgName := "Fleet Device Management Inc."
	token := "applemcptoken"
	expDate := time.Now().Add(200 * time.Hour).UTC().Round(time.Second).Format(fleet.VPPTimeFormat)
	tokenJSON := fmt.Sprintf(`{"expDate":%q,"token":%q,"orgName":%q}`, expDate, token, orgName)
	dev_mode.SetOverride("FLEET_DEV_VPP_URL", s.appleVPPConfigSrv.URL, t)

	// Adam IDs "2" and "3" come pre-registered by the mock VPP server with iOS/iPadOS metadata.
	const iosAdamID = "2"
	const ipadOSAdamID = "3"

	var validToken uploadVPPTokenResponse
	s.uploadDataViaForm("/api/latest/fleet/vpp_tokens", "token", "token.vpptoken",
		[]byte(base64.StdEncoding.EncodeToString([]byte(tokenJSON))), http.StatusAccepted, "", &validToken)

	var getVPPTokenResp getVPPTokensResponse
	s.DoJSON("GET", "/api/latest/fleet/vpp_tokens", &getVPPTokensRequest{}, http.StatusOK, &getVPPTokenResp)

	var resPatchVPP patchVPPTokensTeamsResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/vpp_tokens/%d/teams", getVPPTokenResp.Tokens[0].ID),
		patchVPPTokensTeamsRequest{TeamIDs: []uint{team.ID}}, http.StatusOK, &resPatchVPP)

	const validPlist = `<dict><key>ServerURL</key><string>https://example.com</string></dict>`
	const validPlist2 = `<dict><key>ServerURL</key><string>https://other.example.com</string><key>HostUUID</key><string>$FLEET_VAR_HOST_UUID</string></dict>`

	// Helper: encode an XML string as a JSON string (the form clients send).
	asJSONString := func(s string) json.RawMessage {
		b, err := json.Marshal(s)
		require.NoError(t, err)
		return json.RawMessage(b)
	}

	// Helper: read the stored configuration directly from the datastore.
	readStoredConfig := func(adamID string, platform fleet.InstallableDevicePlatform) []byte {
		got, err := getFirstAddedVPPAppConfigurationForTest(t, s.ds, platform, adamID, team.ID)
		require.NoError(t, err)
		return got
	}

	// 1. Add iOS app with valid plist configuration → 200, stored, activity emitted.
	var addResp addAppStoreAppResponse
	s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{
		TeamID:        &team.ID,
		AppStoreID:    iosAdamID,
		Platform:      fleet.IOSPlatform,
		Configuration: asJSONString(validPlist),
	}, http.StatusOK, &addResp)
	require.NotZero(t, addResp.TitleID)

	require.Equal(t, []byte(validPlist), readStoredConfig(iosAdamID, fleet.IOSPlatform))

	// 2. Update iOS app with new configuration that includes an allowed Fleet variable.
	var updResp updateAppStoreAppResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/app_store_app", addResp.TitleID),
		&updateAppStoreAppRequest{
			TeamID:        &team.ID,
			Configuration: asJSONString(validPlist2),
		}, http.StatusOK, &updResp)
	require.Equal(t, []byte(validPlist2), readStoredConfig(iosAdamID, fleet.IOSPlatform))

	// GET title returns the iOS configuration as a JSON string of plist; unmarshal to recover the raw plist bytes.
	var titleResp getSoftwareTitleResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", addResp.TitleID),
		&getSoftwareTitleRequest{ID: addResp.TitleID, TeamID: &team.ID},
		http.StatusOK, &titleResp, "fleet_id", fmt.Sprint(team.ID))
	require.NotNil(t, titleResp.SoftwareTitle.AppStoreApp)
	var gotPlist string
	require.NoError(t, json.Unmarshal(titleResp.SoftwareTitle.AppStoreApp.Configuration, &gotPlist))
	require.Equal(t, validPlist2, gotPlist)

	// 3. Update iOS app omitting `configuration` field → no change.
	updResp = updateAppStoreAppResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/app_store_app", addResp.TitleID),
		&updateAppStoreAppRequest{TeamID: &team.ID, SelfService: new(true)}, http.StatusOK, &updResp)
	require.Equal(t, []byte(validPlist2), readStoredConfig(iosAdamID, fleet.IOSPlatform))

	// 3b. Update iOS app with `configuration: null` → row deleted (clear semantics
	// must match the batch path; previously the single-app PATCH stored empty bytes).
	updResp = updateAppStoreAppResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/app_store_app", addResp.TitleID),
		&updateAppStoreAppRequest{TeamID: &team.ID, Configuration: json.RawMessage(`null`)},
		http.StatusOK, &updResp)
	_, err = getFirstAddedVPPAppConfigurationForTest(t, s.ds, fleet.IOSPlatform, iosAdamID, team.ID)
	require.True(t, fleet.IsNotFound(err), "expected configuration row to be deleted on null PATCH, got %v", err)

	// Re-set the configuration so the rest of the test continues with state.
	updResp = updateAppStoreAppResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/app_store_app", addResp.TitleID),
		&updateAppStoreAppRequest{TeamID: &team.ID, Configuration: asJSONString(validPlist2)},
		http.StatusOK, &updResp)
	require.Equal(t, []byte(validPlist2), readStoredConfig(iosAdamID, fleet.IOSPlatform))

	// 4. Add iPadOS app with malformed XML → 422.
	res := s.Do("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{
		TeamID:        &team.ID,
		AppStoreID:    ipadOSAdamID,
		Platform:      fleet.IPadOSPlatform,
		Configuration: asJSONString(`not actually a plist`),
	}, http.StatusUnprocessableEntity)
	require.Contains(t, extractServerErrorText(res.Body), "invalid plist")

	// 5. Add iOS app with disallowed Fleet variable → 422.
	res = s.Do("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{
		TeamID:        &team.ID,
		AppStoreID:    ipadOSAdamID,
		Platform:      fleet.IPadOSPlatform,
		Configuration: asJSONString(`<dict><key>K</key><string>$FLEET_VAR_NDES_SCEP_CHALLENGE</string></dict>`),
	}, http.StatusUnprocessableEntity)
	require.Contains(t, extractServerErrorText(res.Body), "$FLEET_VAR_NDES_SCEP_CHALLENGE")

	// Update iOS app with malformed XML → 422.
	res = s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/app_store_app", addResp.TitleID),
		&updateAppStoreAppRequest{
			TeamID:        &team.ID,
			Configuration: asJSONString(`not actually a plist`),
		}, http.StatusUnprocessableEntity)
	require.Contains(t, extractServerErrorText(res.Body), "invalid plist")

	// Update iOS app with disallowed Fleet variable → 422.
	res = s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/app_store_app", addResp.TitleID),
		&updateAppStoreAppRequest{
			TeamID:        &team.ID,
			Configuration: asJSONString(`<dict><key>K</key><string>$FLEET_VAR_NDES_SCEP_CHALLENGE</string></dict>`),
		}, http.StatusUnprocessableEntity)
	require.Contains(t, extractServerErrorText(res.Body), "$FLEET_VAR_NDES_SCEP_CHALLENGE")

	// macOS adam ID — pre-registered as a macOS-only app in the mock VPP server.
	const macosAdamID = "1"

	requireNoStoredConfig := func(platform fleet.InstallableDevicePlatform, adamID string, teamID uint) {
		_, err := getFirstAddedVPPAppConfigurationForTest(t, s.ds, platform, adamID, teamID)
		require.True(t, fleet.IsNotFound(err), "expected not found, got %v", err)
	}

	// 6. Add macOS app with configuration → 200, configuration silently dropped.
	var addMacResp addAppStoreAppResponse
	s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{
		TeamID:        &team.ID,
		AppStoreID:    macosAdamID,
		Platform:      fleet.MacOSPlatform,
		Configuration: asJSONString(validPlist),
	}, http.StatusOK, &addMacResp)
	require.NotZero(t, addMacResp.TitleID)
	requireNoStoredConfig(fleet.MacOSPlatform, macosAdamID, team.ID)

	// 7. Update macOS app with configuration → 200, configuration still not stored.
	var updMacResp updateAppStoreAppResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/app_store_app", addMacResp.TitleID),
		&updateAppStoreAppRequest{
			TeamID:        &team.ID,
			Configuration: asJSONString(validPlist),
		}, http.StatusOK, &updMacResp)
	requireNoStoredConfig(fleet.MacOSPlatform, macosAdamID, team.ID)

	// 8. Add macOS app with malformed XML → 200 (silent drop must come before validation).
	const macosAdamIDInvalid = "2"
	var addMacInvalidResp addAppStoreAppResponse
	s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{
		TeamID:        &team.ID,
		AppStoreID:    macosAdamIDInvalid,
		Platform:      fleet.MacOSPlatform,
		Configuration: asJSONString(`not actually a plist`),
	}, http.StatusOK, &addMacInvalidResp)
	require.NotZero(t, addMacInvalidResp.TitleID)
	requireNoStoredConfig(fleet.MacOSPlatform, macosAdamIDInvalid, team.ID)

	// 9. Update macOS app with malformed XML → 200, still no row.
	var updMacInvalidResp updateAppStoreAppResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/app_store_app", addMacResp.TitleID),
		&updateAppStoreAppRequest{
			TeamID:        &team.ID,
			Configuration: asJSONString(`not actually a plist`),
		}, http.StatusOK, &updMacInvalidResp)
	requireNoStoredConfig(fleet.MacOSPlatform, macosAdamID, team.ID)

	t.Run("BatchAssociateVPPApps", func(t *testing.T) {
		// Use a fresh team so batch "replace all" doesn't clobber the prior state.
		batchTeam, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "vpp-apple-config-batch-team"})
		require.NoError(t, err)

		var resPatchVPPBatch patchVPPTokensTeamsResponse
		s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/vpp_tokens/%d/teams", getVPPTokenResp.Tokens[0].ID),
			patchVPPTokensTeamsRequest{TeamIDs: []uint{team.ID, batchTeam.ID}}, http.StatusOK, &resPatchVPPBatch)

		var batchResp batchAssociateAppStoreAppsResponse

		// iOS in the same batch proves the path actually ran, so a missing macOS row isn't a no-op.
		s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps/batch",
			batchAssociateAppStoreAppsRequest{
				Apps: []fleet.VPPBatchPayload{
					{AppStoreID: iosAdamID, Platform: fleet.IOSPlatform, Configuration: asJSONString(validPlist)},
					{AppStoreID: macosAdamID, Platform: fleet.MacOSPlatform, Configuration: asJSONString(validPlist)},
				},
			}, http.StatusOK, &batchResp, "fleet_name", batchTeam.Name)

		// iOS config IS stored — confirms the batch wrote configurations.
		iosCfg, err := getFirstAddedVPPAppConfigurationForTest(t, s.ds, fleet.IOSPlatform, iosAdamID, batchTeam.ID)
		require.NoError(t, err)
		require.Equal(t, []byte(validPlist), iosCfg)

		// macOS config silently dropped.
		requireNoStoredConfig(fleet.MacOSPlatform, macosAdamID, batchTeam.ID)

		// Sanity: macOS app IS associated — silent drop applies to config only, not the app association.
		macMeta, err := s.ds.GetVPPAppMetadataByAdamIDPlatformTeamID(ctx, macosAdamID, fleet.MacOSPlatform, &batchTeam.ID)
		require.NoError(t, err)
		require.Equal(t, macosAdamID, macMeta.AdamID)

		// Remove macOS via batch by omitting it from the payload.
		s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps/batch",
			batchAssociateAppStoreAppsRequest{
				Apps: []fleet.VPPBatchPayload{
					{AppStoreID: iosAdamID, Platform: fleet.IOSPlatform, Configuration: asJSONString(validPlist)},
				},
			}, http.StatusOK, &batchResp, "fleet_name", batchTeam.Name)

		_, err = s.ds.GetVPPAppMetadataByAdamIDPlatformTeamID(ctx, macosAdamID, fleet.MacOSPlatform, &batchTeam.ID)
		require.True(t, fleet.IsNotFound(err), "expected macOS app to be removed, got %v", err)

		// Re-add macOS via batch with malformed plist → locks the silent-drop ordering for the batch path.
		s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps/batch",
			batchAssociateAppStoreAppsRequest{
				Apps: []fleet.VPPBatchPayload{
					{AppStoreID: iosAdamID, Platform: fleet.IOSPlatform, Configuration: asJSONString(validPlist)},
					{AppStoreID: macosAdamID, Platform: fleet.MacOSPlatform, Configuration: asJSONString(`not actually a plist`)},
				},
			}, http.StatusOK, &batchResp, "fleet_name", batchTeam.Name)

		macMeta, err = s.ds.GetVPPAppMetadataByAdamIDPlatformTeamID(ctx, macosAdamID, fleet.MacOSPlatform, &batchTeam.ID)
		require.NoError(t, err)
		require.Equal(t, macosAdamID, macMeta.AdamID)
		requireNoStoredConfig(fleet.MacOSPlatform, macosAdamID, batchTeam.ID)
	})
}

// TestManagedAppConfigurationWireFormat asserts the on-the-wire shape of
// VPPAppStoreApp.Configuration using hand-built request bodies and raw
// response bytes (no Go struct marshalling on either side).
func (s *integrationMDMTestSuite) TestManagedAppConfigurationWireFormat() {
	t := s.T()
	s.setSkipWorkerJobs(t)
	ctx := context.Background()

	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "vpp-config-wire-team"})
	require.NoError(t, err)
	s.setVPPTokenForTeam(team.ID)

	// readBody compacts the pretty-printed JSON so assertions can match unindented wire bytes.
	readBody := func(resp *http.Response) string {
		t.Helper()
		raw, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.NoError(t, resp.Body.Close())
		var compact bytes.Buffer
		require.NoError(t, json.Compact(&compact, raw))
		return compact.String()
	}

	t.Run("vpp ios wire format", func(t *testing.T) {
		// Adam ID "2" is pre-registered as an iOS app by the mock VPP server.
		const iosAdamID = "2"
		reqBody := fmt.Appendf(nil,
			`{"fleet_id":%d,"app_store_id":%q,"platform":"ios","configuration":"<dict><key>K</key><string>v</string></dict>"}`,
			team.ID, iosAdamID)
		resp := s.DoRaw("POST", "/api/latest/fleet/software/app_store_apps", reqBody, http.StatusOK)
		var addResp addAppStoreAppResponse
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&addResp))
		require.NoError(t, resp.Body.Close())
		require.NotZero(t, addResp.TitleID)

		stored, err := s.ds.GetVPPAppConfiguration(ctxdb.RequirePrimary(ctx, true), addResp.VersionID)
		require.NoError(t, err)
		require.Equal(t, `<dict><key>K</key><string>v</string></dict>`, string(stored))

		resp = s.DoRaw("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", addResp.TitleID), nil, http.StatusOK, "fleet_id", fmt.Sprint(team.ID))
		body := readBody(resp)
		require.Contains(t, body, `"configuration":"<dict><key>K</key><string>v</string></dict>"`)
	})

	t.Run("android wire format", func(t *testing.T) {
		s.enableAndroidMDM(t)
		const androidAdamID = "com.test.wireformat"
		s.androidAPIClient.EnterprisesApplicationsFunc = func(_ context.Context, _, _ string) (*androidmanagement.Application, error) {
			return &androidmanagement.Application{IconUrl: "https://example.com/icon.png", Title: "WireApp"}, nil
		}
		s.androidAPIClient.EnterprisesPoliciesPatchFunc = func(_ context.Context, _ string, policy *androidmanagement.Policy, _ androidmgmt.PoliciesPatchOpts) (*androidmanagement.Policy, error) {
			return policy, nil
		}

		reqBody := fmt.Appendf(nil,
			`{"fleet_id":%d,"app_store_id":%q,"platform":"android","configuration":{"workProfileWidgets":"WORK_PROFILE_WIDGETS_ALLOWED"}}`,
			team.ID, androidAdamID)
		resp := s.DoRaw("POST", "/api/latest/fleet/software/app_store_apps", reqBody, http.StatusOK)
		var addResp addAppStoreAppResponse
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&addResp))
		require.NoError(t, resp.Body.Close())
		require.NotZero(t, addResp.TitleID)

		stored, err := s.ds.GetAndroidAppConfiguration(ctxdb.RequirePrimary(ctx, true), androidAdamID, team.ID)
		require.NoError(t, err)
		require.JSONEq(t, `{"workProfileWidgets":"WORK_PROFILE_WIDGETS_ALLOWED"}`, string(stored))

		resp = s.DoRaw("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", addResp.TitleID), nil, http.StatusOK, "fleet_id", fmt.Sprint(team.ID))
		body := readBody(resp)
		require.Contains(t, body, `"configuration":{"workProfileWidgets":"WORK_PROFILE_WIDGETS_ALLOWED"}`)
	})
}

// TestVPPManagedConfigurationOnInstallCommand asserts the InstallApplication
// MDM command bytes for the VPP managed-configuration paths.
func (s *integrationMDMTestSuite) TestVPPManagedConfigurationOnInstallCommand() {
	t := s.T()
	s.setSkipWorkerJobs(t)
	ctx := context.Background()

	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "vpp-managedcfg-install-team"})
	require.NoError(t, err)
	s.setVPPTokenForTeam(team.ID)
	s.registerResetVPPProxyData(t)

	asJSONString := func(s string) json.RawMessage {
		b, err := json.Marshal(s)
		require.NoError(t, err)
		return json.RawMessage(b)
	}

	titleIDFor := func(adamID string, platform fleet.InstallableDevicePlatform) uint {
		var id uint
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &id,
				`SELECT title_id FROM vpp_apps WHERE adam_id = ? AND platform = ?`, adamID, platform)
		})
		require.NotZero(t, id, "title_id for adam=%s platform=%s", adamID, platform)
		return id
	}

	// drainPendingCommands completes pending verification commands from prior installs, and the installs that
	// re-send the app after a configuration change.
	drainPendingCommands := func(t *testing.T, dev *mdmtest.TestAppleMDMClient, installed fleet.Software) {
		t.Helper()
		installed.Installed = true
		for {
			cmd, err := dev.Idle()
			require.NoError(t, err)
			if cmd == nil {
				break
			}
			if cmd.Command.RequestType == "InstallApplication" {
				_, err = dev.Acknowledge(cmd.CommandUUID)
				require.NoError(t, err)
				continue
			}
			require.Equal(t, "InstalledApplicationList", cmd.Command.RequestType,
				"unexpected pending command %q while draining verifications", cmd.Command.RequestType)
			_, err = dev.AcknowledgeInstalledApplicationList(dev.UUID, cmd.CommandUUID,
				[]fleet.Software{installed})
			require.NoError(t, err)
		}
	}

	// installAndCaptureCmd triggers an install and returns the InstallApplication bytes,
	// then completes the verification so the host is ready for another install.
	installAndCaptureCmd := func(t *testing.T, host *fleet.Host, dev *mdmtest.TestAppleMDMClient, titleID uint, installed fleet.Software) []byte {
		t.Helper()
		installed.Installed = true
		drainPendingCommands(t, dev, installed)

		var installResp installSoftwareResponse
		s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", host.ID, titleID),
			&installSoftwareRequest{}, http.StatusAccepted, &installResp)

		s.awaitRunAppleMDMWorkerSchedule()
		s.runWorker()
		cmd, err := dev.Idle()
		require.NoError(t, err)
		require.NotNil(t, cmd, "expected an MDM command after install trigger but device went idle")
		require.Equal(t, "InstallApplication", cmd.Command.RequestType)
		raw := append([]byte(nil), cmd.Raw...)
		_, err = dev.Acknowledge(cmd.CommandUUID)
		require.NoError(t, err)

		s.runWorker()
		cmd, err = dev.Idle()
		require.NoError(t, err)
		require.NotNil(t, cmd, "expected an InstalledApplicationList verification command")
		require.Equal(t, "InstalledApplicationList", cmd.Command.RequestType)
		installed.Installed = true
		_, err = dev.AcknowledgeInstalledApplicationList(dev.UUID, cmd.CommandUUID,
			[]fleet.Software{installed})
		require.NoError(t, err)
		return raw
	}

	// Adam ID "2" is registered for iOS + iPadOS + macOS in the mock proxy.
	const adamMulti = "2"
	// Adam ID "1" is macOS-only.
	const adamMac = "1"

	iosHost, iosDev := s.createAppleMobileHostThenEnrollMDM("ios")
	s.appleVPPConfigSrvConfig.SerialNumbers = append(s.appleVPPConfigSrvConfig.SerialNumbers, iosDev.SerialNumber)
	s.Do("POST", "/api/latest/fleet/hosts/transfer",
		&addHostsToTeamRequest{HostIDs: []uint{iosHost.ID}, TeamID: &team.ID}, http.StatusOK)

	// App-2 metadata as the mock proxy reports it for ios / ipados.
	app2Installed := fleet.Software{Name: "App 2", BundleIdentifier: "b-2", Version: "2.0.0"}

	t.Run("iOS install carries Configuration and resolves $FLEET_VAR_HOST_UUID", func(t *testing.T) {
		plistXML := `<dict><key>K</key><string>v</string><key>UUID</key><string>$FLEET_VAR_HOST_UUID</string></dict>`
		var addResp addAppStoreAppResponse
		s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{
			TeamID: &team.ID, AppStoreID: adamMulti, Platform: fleet.IOSPlatform,
			Configuration: asJSONString(plistXML),
		}, http.StatusOK, &addResp)

		raw := string(installAndCaptureCmd(t, iosHost, iosDev, titleIDFor(adamMulti, fleet.IOSPlatform), app2Installed))
		require.Contains(t, raw, "<key>Configuration</key>",
			"InstallApplication should include Configuration dict for iOS")
		require.Contains(t, raw, "<key>K</key>")
		require.Contains(t, raw, "<string>v</string>")
		require.NotContains(t, raw, "$FLEET_VAR_HOST_UUID",
			"Fleet variable must be resolved before sending to device")
		require.Contains(t, raw, fmt.Sprintf("<string>%s</string>", iosHost.UUID),
			"Resolved value should be the iOS host's own UUID")
	})

	t.Run("clearing configuration drops Configuration from the next install", func(t *testing.T) {
		titleID := titleIDFor(adamMulti, fleet.IOSPlatform)
		s.DoJSON("PATCH",
			fmt.Sprintf("/api/latest/fleet/software/titles/%d/app_store_app", titleID),
			&updateAppStoreAppRequest{TeamID: &team.ID, Configuration: json.RawMessage(`null`)},
			http.StatusOK, &updateAppStoreAppResponse{})

		s.lastActivityMatches(fleet.ActivityEditedAppStoreApp{}.ActivityName(), "", 0)

		_, err := getFirstAddedVPPAppConfigurationForTest(t, s.ds, fleet.IOSPlatform, adamMulti, team.ID)
		require.True(t, fleet.IsNotFound(err), "expected config row deleted")

		raw := string(installAndCaptureCmd(t, iosHost, iosDev, titleID, app2Installed))
		require.NotContains(t, raw, "<key>Configuration</key>",
			"InstallApplication should not include Configuration after clearing the stored config")
	})

	t.Run("second iOS host gets its own UUID substituted (per-host resolution)", func(t *testing.T) {
		titleID := titleIDFor(adamMulti, fleet.IOSPlatform)
		s.DoJSON("PATCH",
			fmt.Sprintf("/api/latest/fleet/software/titles/%d/app_store_app", titleID),
			&updateAppStoreAppRequest{
				TeamID:        &team.ID,
				Configuration: asJSONString(`<dict><key>UUID</key><string>$FLEET_VAR_HOST_UUID</string></dict>`),
			},
			http.StatusOK, &updateAppStoreAppResponse{})

		ios2Host, ios2Dev := s.createAppleMobileHostThenEnrollMDM("ios")
		s.appleVPPConfigSrvConfig.SerialNumbers = append(s.appleVPPConfigSrvConfig.SerialNumbers, ios2Dev.SerialNumber)
		s.Do("POST", "/api/latest/fleet/hosts/transfer",
			&addHostsToTeamRequest{HostIDs: []uint{ios2Host.ID}, TeamID: &team.ID}, http.StatusOK)
		require.NotEqual(t, iosHost.UUID, ios2Host.UUID, "second host needs a distinct UUID for this test to be meaningful")

		raw := string(installAndCaptureCmd(t, ios2Host, ios2Dev, titleID, app2Installed))
		require.Contains(t, raw, fmt.Sprintf("<string>%s</string>", ios2Host.UUID),
			"second host's command should carry its own UUID")
		require.NotContains(t, raw, fmt.Sprintf("<string>%s</string>", iosHost.UUID),
			"second host's command must not carry the first host's UUID")
	})

	t.Run("macOS install drops Configuration even when one was sent on add", func(t *testing.T) {
		plistXML := `<dict><key>K</key><string>v</string></dict>`
		var addResp addAppStoreAppResponse
		s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{
			TeamID: &team.ID, AppStoreID: adamMac, Platform: fleet.MacOSPlatform,
			Configuration: asJSONString(plistXML),
		}, http.StatusOK, &addResp)

		// macOS install path requires fleetd enrollment.
		macHost, macDev := createHostThenEnrollMDM(s.ds, s.server.URL, t)
		setOrbitEnrollment(t, macHost, s.ds)
		s.appleVPPConfigSrvConfig.SerialNumbers = append(s.appleVPPConfigSrvConfig.SerialNumbers, macHost.HardwareSerial)
		s.Do("POST", "/api/latest/fleet/hosts/transfer",
			&addHostsToTeamRequest{HostIDs: []uint{macHost.ID}, TeamID: &team.ID}, http.StatusOK)
		// Drain the InstallFleetd command queued by enrollment.
		s.awaitRunAppleMDMWorkerSchedule()
		s.runWorker()
		checkInstallFleetdCommandSent(t, macDev, true)

		raw := string(installAndCaptureCmd(t, macHost, macDev,
			titleIDFor(adamMac, fleet.MacOSPlatform),
			fleet.Software{Name: "App 1", BundleIdentifier: "a-1", Version: "1.0.0"}))
		require.NotContains(t, raw, "<key>Configuration</key>",
			"macOS InstallApplication must not carry Configuration even if one was POSTed")
	})

	t.Run("same adam_id on iOS and iPadOS keep configs isolated", func(t *testing.T) {
		const iosCfg = `<dict><key>side</key><string>ios</string></dict>`
		const ipadCfg = `<dict><key>side</key><string>ipados</string></dict>`

		iosTitle := titleIDFor(adamMulti, fleet.IOSPlatform)
		s.DoJSON("PATCH",
			fmt.Sprintf("/api/latest/fleet/software/titles/%d/app_store_app", iosTitle),
			&updateAppStoreAppRequest{TeamID: &team.ID, Configuration: asJSONString(iosCfg)},
			http.StatusOK, &updateAppStoreAppResponse{})

		var addResp addAppStoreAppResponse
		s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{
			TeamID: &team.ID, AppStoreID: adamMulti, Platform: fleet.IPadOSPlatform,
			Configuration: asJSONString(ipadCfg),
		}, http.StatusOK, &addResp)

		storedIOS, err := getFirstAddedVPPAppConfigurationForTest(t, s.ds, fleet.IOSPlatform, adamMulti, team.ID)
		require.NoError(t, err)
		require.Equal(t, iosCfg, string(storedIOS))
		storedIPad, err := getFirstAddedVPPAppConfigurationForTest(t, s.ds, fleet.IPadOSPlatform, adamMulti, team.ID)
		require.NoError(t, err)
		require.Equal(t, ipadCfg, string(storedIPad))
		require.NotEqual(t, string(storedIOS), string(storedIPad))

		ipadHost, ipadDev := s.createAppleMobileHostThenEnrollMDM("ipados")
		s.appleVPPConfigSrvConfig.SerialNumbers = append(s.appleVPPConfigSrvConfig.SerialNumbers, ipadDev.SerialNumber)
		s.Do("POST", "/api/latest/fleet/hosts/transfer",
			&addHostsToTeamRequest{HostIDs: []uint{ipadHost.ID}, TeamID: &team.ID}, http.StatusOK)

		raw := string(installAndCaptureCmd(t, ipadHost, ipadDev,
			titleIDFor(adamMulti, fleet.IPadOSPlatform), app2Installed))
		require.Contains(t, raw, "<string>ipados</string>",
			"iPadOS install must use the iPadOS-scoped config")
		require.NotContains(t, raw, "<string>ios</string>",
			"iPadOS install must not leak the iOS-scoped config bytes")
	})

	t.Run("updating configuration → next install carries the new bytes", func(t *testing.T) {
		// Every install path re-reads the stored config at enqueue time
		// (nanoEnqueueVPPInstall), so this also exercises the auto-update freshness path.
		titleID := titleIDFor(adamMulti, fleet.IOSPlatform)
		const updatedCfg = `<dict><key>K</key><string>updated</string></dict>`
		s.DoJSON("PATCH",
			fmt.Sprintf("/api/latest/fleet/software/titles/%d/app_store_app", titleID),
			&updateAppStoreAppRequest{TeamID: &team.ID, Configuration: asJSONString(updatedCfg)},
			http.StatusOK, &updateAppStoreAppResponse{})

		raw := string(installAndCaptureCmd(t, iosHost, iosDev, titleID, app2Installed))
		require.Contains(t, raw, "<key>Configuration</key>",
			"updated install should still carry a Configuration dict")
		require.Contains(t, raw, "<string>updated</string>",
			"updated install must carry the latest stored config bytes")

		// complete the install that re-sends the updated configuration, so the failed install below is the host's latest
		drainPendingCommands(t, iosDev, app2Installed)

		// Updating to a config that references an IDP variable the host hasn't
		// been linked to → the install is RECORDED AS FAILED (visible in the
		// activity feed + Install Details modal) instead of being rejected with
		// a 400. A pre-flight substitution check in InstallVPPAppPostValidation
		// catches this before reserving a license or enqueuing a command.
		// Regression for #45851 / #45854.
		s.DoJSON("PATCH",
			fmt.Sprintf("/api/latest/fleet/software/titles/%d/app_store_app", titleID),
			&updateAppStoreAppRequest{
				TeamID:        &team.ID,
				Configuration: asJSONString(`<dict><key>Email</key><string>$FLEET_VAR_HOST_END_USER_EMAIL_IDP</string></dict>`),
			},
			http.StatusOK, &updateAppStoreAppResponse{})

		var uaBefore int
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &uaBefore,
				`SELECT COUNT(*) FROM upcoming_activities WHERE host_id = ?`, iosHost.ID)
		})

		s.DoJSON("POST",
			fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", iosHost.ID, titleID),
			&installSoftwareRequest{}, http.StatusAccepted, &installSoftwareResponse{})

		// A failed install row is recorded (verification_failed_at set).
		var failedCount int
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &failedCount,
				`SELECT COUNT(*) FROM host_vpp_software_installs WHERE host_id = ? AND verification_failed_at IS NOT NULL`, iosHost.ID)
		})
		require.Equal(t, 1, failedCount, "a failed VPP install row must be recorded")

		// No zombie upcoming activity and no enqueued MDM command.
		var uaAfter, nanoCount int
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			if err := sqlx.GetContext(ctx, q, &uaAfter,
				`SELECT COUNT(*) FROM upcoming_activities WHERE host_id = ?`, iosHost.ID); err != nil {
				return err
			}
			return sqlx.GetContext(ctx, q, &nanoCount,
				`SELECT COUNT(*) FROM nano_commands nc
				 JOIN host_vpp_software_installs hvsi ON hvsi.command_uuid = nc.command_uuid
				 WHERE hvsi.host_id = ? AND hvsi.verification_failed_at IS NOT NULL`, iosHost.ID)
		})
		require.Equal(t, uaBefore, uaAfter, "failed install must not leave a zombie upcoming_activity row")
		require.Zero(t, nanoCount, "failed install must not enqueue an MDM command")

		// The failure is visible as a failed installed_app_store_app activity
		// carrying the unresolvable variable as the reason.
		var listActs listActivitiesResponse
		s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &listActs,
			"order_key", "id", "order_direction", "desc", "per_page", "10")
		var found bool
		for _, act := range listActs.Activities {
			if act.Type == (fleet.ActivityInstalledAppStoreApp{}).ActivityName() {
				require.NotNil(t, act.Details)
				require.Contains(t, string(*act.Details), string(fleet.SoftwareInstallFailed))
				require.Contains(t, string(*act.Details), "$FLEET_VAR_HOST_END_USER_EMAIL_IDP")
				found = true
				break
			}
		}
		require.True(t, found, "a failed installed_app_store_app activity must be in the feed")

		// Regression: the per-team/per-app status summary (vpp.go
		// GetSummaryHostVPPAppInstalls) and the host software list
		// status=failed_install filter (software_installers.go
		// vppHostSoftwareInstallJoin) must both count Fleet-side pre-flight
		// failures. Before the SQL fix, the WHERE `ncr.id IS NOT NULL …`
		// clause dropped these rows entirely (no nano_command_results is
		// written when no MDM command is enqueued), so the install row
		// existed in host_vpp_software_installs with verification_failed_at
		// set but was invisible to both surfaces.
		var titleSummary getSoftwareTitleResponse
		s.DoJSON("GET",
			fmt.Sprintf("/api/latest/fleet/software/titles/%d", titleID),
			&getSoftwareTitleRequest{ID: titleID, TeamID: &team.ID},
			http.StatusOK, &titleSummary, "fleet_id", fmt.Sprint(team.ID))
		require.NotNil(t, titleSummary.SoftwareTitle.AppStoreApp)
		require.NotNil(t, titleSummary.SoftwareTitle.AppStoreApp.Status)
		require.GreaterOrEqual(t, titleSummary.SoftwareTitle.AppStoreApp.Status.Failed, uint(1),
			"per-team/per-app failed-install summary must count the pre-flight failure")

		var hostSw getHostSoftwareResponse
		s.DoJSON("GET",
			fmt.Sprintf("/api/latest/fleet/hosts/%d/software", iosHost.ID),
			nil, http.StatusOK, &hostSw, "status", "failed_install")
		var foundInList bool
		for _, sw := range hostSw.Software {
			if sw.ID == titleID {
				foundInList = true
				break
			}
		}
		require.True(t, foundInList,
			"host software list filtered by status=failed_install must include the pre-flight-failed title")
	})

	t.Run("self-service install carries Configuration with $FLEET_VAR_HOST_UUID resolved", func(t *testing.T) {
		ssHost, ssDev := s.createAppleMobileHostThenEnrollMDM("ios")
		s.appleVPPConfigSrvConfig.SerialNumbers = append(s.appleVPPConfigSrvConfig.SerialNumbers, ssDev.SerialNumber)
		s.Do("POST", "/api/latest/fleet/hosts/transfer",
			&addHostsToTeamRequest{HostIDs: []uint{ssHost.ID}, TeamID: &team.ID}, http.StatusOK)

		titleID := titleIDFor(adamMulti, fleet.IOSPlatform)
		s.DoJSON("PATCH",
			fmt.Sprintf("/api/latest/fleet/software/titles/%d/app_store_app", titleID),
			&updateAppStoreAppRequest{
				TeamID: &team.ID, SelfService: new(true),
				Configuration: asJSONString(`<dict><key>UUID</key><string>$FLEET_VAR_HOST_UUID</string></dict>`),
			},
			http.StatusOK, &updateAppStoreAppResponse{})

		s.DoRawNoAuth("POST",
			fmt.Sprintf("/api/latest/fleet/device/%s/software/install/%d", ssHost.UUID, titleID),
			nil, http.StatusAccepted)

		s.awaitRunAppleMDMWorkerSchedule()
		s.runWorker()
		cmd, err := ssDev.Idle()
		require.NoError(t, err)
		require.NotNil(t, cmd)
		require.Equal(t, "InstallApplication", cmd.Command.RequestType)
		raw := string(cmd.Raw)
		require.Contains(t, raw, "<key>Configuration</key>",
			"self-service iOS install should still include Configuration")
		require.Contains(t, raw, fmt.Sprintf("<string>%s</string>", ssHost.UUID),
			"self-service install must resolve $FLEET_VAR_HOST_UUID to this host's UUID")
		_, err = ssDev.Acknowledge(cmd.CommandUUID)
		require.NoError(t, err)
	})

	t.Run("auto-update install carries the managed Configuration", func(t *testing.T) {
		// adam "3" is unused by prior subtests; override its proxy metadata
		// to declare iOS at v1.0.0 (default proxy data has it as iPadOS-only).
		const cfgAdamID = "3"
		s.appleVPPProxySrvData[cfgAdamID] = `{"id": "3", "attributes": {"name": "Config App", "platformAttributes": {"ios": {"bundleId": "app-cfg", "artwork": {"url": "https://example.com/images/3/{w}x{h}.{f}"}, "latestVersionInfo": {"versionDisplay": "1.0.0"}}}, "deviceFamilies": ["iphone", "ipad"]}}`

		cfgHost, cfgDev := s.createAppleMobileHostThenEnrollMDM("ios")
		s.appleVPPConfigSrvConfig.SerialNumbers = append(s.appleVPPConfigSrvConfig.SerialNumbers, cfgDev.SerialNumber)
		s.Do("POST", "/api/latest/fleet/hosts/transfer",
			&addHostsToTeamRequest{HostIDs: []uint{cfgHost.ID}, TeamID: &team.ID}, http.StatusOK)

		var addResp addAppStoreAppResponse
		s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{
			TeamID: &team.ID, AppStoreID: cfgAdamID, Platform: fleet.IOSPlatform,
			Configuration: json.RawMessage(`"<dict><key>auto</key><string>update</string></dict>"`),
		}, http.StatusOK, &addResp)
		appTitleID := addResp.TitleID
		require.NotZero(t, appTitleID)

		// First install.
		s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", cfgHost.ID, appTitleID),
			&installSoftwareRequest{}, http.StatusAccepted, &installSoftwareResponse{})
		s.awaitRunAppleMDMWorkerSchedule()
		s.runWorker()
		cmd, err := cfgDev.Idle()
		require.NoError(t, err)
		require.Equal(t, "InstallApplication", cmd.Command.RequestType)
		require.Contains(t, string(cmd.Raw), "<key>Configuration</key>")
		require.Contains(t, string(cmd.Raw), "<string>update</string>")
		_, err = cfgDev.Acknowledge(cmd.CommandUUID)
		require.NoError(t, err)
		s.runWorker()
		cmd, err = cfgDev.Idle()
		require.NoError(t, err)
		require.Equal(t, "InstalledApplicationList", cmd.Command.RequestType)
		_, err = cfgDev.AcknowledgeInstalledApplicationList(cfgDev.UUID, cmd.CommandUUID, []fleet.Software{
			{Name: "Config App", BundleIdentifier: "app-cfg", Version: "1.0.0", Installed: true},
		})
		require.NoError(t, err)

		// handleRefetch processes the three commands a refetch queues:
		// InstalledApplicationList, CertificateList, DeviceInformation.
		handleRefetch := func(software []fleet.Software) {
			s.runWorker()
			c, err := cfgDev.Idle()
			require.NoError(t, err)
			require.Equal(t, "InstalledApplicationList", c.Command.RequestType)
			_, err = cfgDev.AcknowledgeInstalledApplicationList(cfgDev.UUID, c.CommandUUID, software)
			require.NoError(t, err)
			c, err = cfgDev.Idle()
			require.NoError(t, err)
			require.Equal(t, "CertificateList", c.Command.RequestType)
			_, err = cfgDev.AcknowledgeCertificateList(cfgDev.UUID, c.CommandUUID, nil)
			require.NoError(t, err)
			c, err = cfgDev.Idle()
			require.NoError(t, err)
			require.Equal(t, "DeviceInformation", c.Command.RequestType)
			_, err = cfgDev.AcknowledgeDeviceInformation(cfgDev.UUID, c.CommandUUID, "iPhone 17", "iPhone", "America/Los_Angeles")
			require.NoError(t, err)
		}

		// First refetch populates the host's timezone (DeviceInformation is sent last).
		s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/refetch", cfgHost.ID), nil, http.StatusOK)
		handleRefetch([]fleet.Software{
			{Name: "Config App", BundleIdentifier: "app-cfg", Version: "1.0.0", Installed: true},
		})

		// Enable auto-update inside the current Los Angeles window.
		nowInLA, err := getCurrentLocalTimeInHostTimeZone(ctx, "America/Los_Angeles")
		require.NoError(t, err)
		startHHMM := nowInLA.Add(-30 * time.Minute).Format("15:04")
		endHHMM := nowInLA.Add(30 * time.Minute).Format("15:04")
		s.DoJSON("PATCH", fmt.Sprintf("/api/v1/fleet/software/titles/%d/app_store_app", appTitleID),
			updateAppStoreAppRequest{
				TeamID:              &team.ID,
				AutoUpdateEnabled:   new(true),
				AutoUpdateStartTime: &startHHMM,
				AutoUpdateEndTime:   &endHHMM,
			}, http.StatusOK, &updateAppStoreAppResponse{})

		// Bump latest version in proxy data, refresh, and bypass the 1-hour install filter.
		s.appleVPPProxySrvData[cfgAdamID] = `{"id": "3", "attributes": {"name": "Config App", "platformAttributes": {"ios": {"bundleId": "app-cfg", "artwork": {"url": "https://example.com/images/3/{w}x{h}.{f}"}, "latestVersionInfo": {"versionDisplay": "2.0.0"}}}, "deviceFamilies": ["iphone", "ipad"]}}`
		require.NoError(t, vpp.RefreshVersions(ctx, s.ds, apple_apps.StubbedConfig()))
		mysqltest.ExecAdhocSQL(t, s.ds, func(db sqlx.ExtContext) error {
			_, err := db.ExecContext(ctx,
				`UPDATE host_vpp_software_installs SET created_at = DATE_SUB(NOW(), INTERVAL 2 HOUR) WHERE host_id = ?`, cfgHost.ID)
			return err
		})

		// Refetch with the host still on v1.0.0 → auto-update kicks in.
		s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/refetch", cfgHost.ID), nil, http.StatusOK)
		handleRefetch([]fleet.Software{
			{Name: "Config App", BundleIdentifier: "app-cfg", Version: "1.0.0", Installed: true},
		})

		s.runWorker()
		cmd, err = cfgDev.Idle()
		require.NoError(t, err)
		require.NotNil(t, cmd, "expected an auto-update InstallApplication command")
		require.Equal(t, "InstallApplication", cmd.Command.RequestType)
		require.Contains(t, string(cmd.Raw), "<key>Configuration</key>",
			"scheduled auto-update InstallApplication must carry Configuration")
		require.Contains(t, string(cmd.Raw), "<string>update</string>",
			"scheduled auto-update must use the latest stored config bytes")

		isAuto, _, err := s.ds.GetVPPInstallAutomationReasons(ctx, cmd.CommandUUID)
		require.NoError(t, err)
		require.True(t, isAuto, "command must be recorded with from_auto_update=true")

		_, err = cfgDev.Acknowledge(cmd.CommandUUID)
		require.NoError(t, err)
	})

	t.Run("canceling a pending VPP install releases the reserved VPP license seat", func(t *testing.T) {
		// Regression for #45851 / #45854 (license-leak follow-up to Jonathan's
		// note about activated VPP install activities leaking licenses).
		//
		// A fresh install on a not-yet-licensed device reserves a seat via
		// AssociateAssets. If the admin then cancels the upcoming activity
		// before it reaches the device, Fleet must release the seat via
		// DisassociateAssets — otherwise the seat leaks.
		cancelHost, cancelDev := s.createAppleMobileHostThenEnrollMDM("ios")
		s.appleVPPConfigSrvConfig.SerialNumbers = append(s.appleVPPConfigSrvConfig.SerialNumbers, cancelDev.SerialNumber)
		s.Do("POST", "/api/latest/fleet/hosts/transfer",
			&addHostsToTeamRequest{HostIDs: []uint{cancelHost.ID}, TeamID: &team.ID}, http.StatusOK)

		titleID := titleIDFor(adamMulti, fleet.IOSPlatform)

		// Reset the managed config to a resolvable one so the install proceeds
		// past the pre-flight check and actually reserves a seat.
		s.DoJSON("PATCH",
			fmt.Sprintf("/api/latest/fleet/software/titles/%d/app_store_app", titleID),
			&updateAppStoreAppRequest{
				TeamID:        &team.ID,
				Configuration: asJSONString(`<dict><key>UUID</key><string>$FLEET_VAR_HOST_UUID</string></dict>`),
			},
			http.StatusOK, &updateAppStoreAppResponse{})

		// Snapshot the disassociate-call count so we can assert exactly one new
		// release happens on cancel.
		disassocBefore := len(s.appleVPPConfigSrvConfig.Disassociations)

		s.DoJSON("POST",
			fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", cancelHost.ID, titleID),
			&installSoftwareRequest{}, http.StatusAccepted, &installSoftwareResponse{})

		// Pull the activated install's command_uuid (== upcoming_activities.execution_id).
		var row struct {
			CommandUUID       string `db:"command_uuid"`
			AssociatedEventID string `db:"associated_event_id"`
		}
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &row,
				`SELECT command_uuid, COALESCE(associated_event_id, '') AS associated_event_id
				 FROM host_vpp_software_installs WHERE host_id = ? AND adam_id = ?`,
				cancelHost.ID, adamMulti)
		})
		// Sanity: this install was the one that reserved a seat.
		require.NotEmpty(t, row.AssociatedEventID, "test setup: this install should have reserved a license")
		commandUUID := row.CommandUUID

		// Cancel the upcoming activity (DELETE /hosts/{id}/activities/upcoming/{activity_id}).
		s.Do("DELETE",
			fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming/%s", cancelHost.ID, commandUUID),
			nil, http.StatusNoContent)

		// The mock VPP server must have received exactly one disassociate for
		// this adam_id + serial.
		require.Len(t, s.appleVPPConfigSrvConfig.Disassociations, disassocBefore+1,
			"cancel must trigger exactly one DisassociateAssets call")
		got := s.appleVPPConfigSrvConfig.Disassociations[disassocBefore]
		require.Len(t, got.Assets, 1)
		require.Equal(t, adamMulti, got.Assets[0].AdamID)
		require.Equal(t, []string{cancelDev.SerialNumber}, got.SerialNumbers)
		require.Empty(t, got.ClientUserIds, "device-enrolled host must disassociate by serial, not clientUserId")
	})

	t.Run("install with two versions carries the first-added version's Configuration", func(t *testing.T) {
		titleID := titleIDFor(adamMulti, fleet.IOSPlatform)
		const firstCfg = `<dict><key>K</key><string>first</string></dict>`
		const secondCfg = `<dict><key>K</key><string>second</string></dict>`

		// set the first version's configuration, then add a second version with another configuration
		s.DoJSON("PATCH",
			fmt.Sprintf("/api/latest/fleet/software/titles/%d/app_store_app", titleID),
			&updateAppStoreAppRequest{TeamID: &team.ID, Configuration: asJSONString(firstCfg)},
			http.StatusOK, &updateAppStoreAppResponse{})
		var addResp addAppStoreAppResponse
		s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{
			TeamID: &team.ID, AppStoreID: adamMulti, Platform: fleet.IOSPlatform, Name: "Second",
			Configuration: asJSONString(secondCfg),
		}, http.StatusOK, &addResp)

		// install on the iOS host, the command should carry the first version's configuration
		raw := string(installAndCaptureCmd(t, iosHost, iosDev, titleID, app2Installed))
		require.Contains(t, raw, "<string>first</string>")
		require.NotContains(t, raw, "<string>second</string>")

		// read the title, the install should be counted under the first version only
		var titleResp getSoftwareTitleResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", titleID), nil, http.StatusOK, &titleResp,
			"fleet_id", fmt.Sprint(team.ID))
		require.Len(t, titleResp.SoftwareTitle.AppStoreApps, 2)
		require.NotZero(t, titleResp.SoftwareTitle.AppStoreApps[0].Status.Installed)
		require.Equal(t, &fleet.VPPAppStatusSummary{}, titleResp.SoftwareTitle.AppStoreApps[1].Status)

		// delete the second version so the title has one version again
		s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/software/titles/%d/available_for_install", titleID), nil, http.StatusNoContent,
			"fleet_id", fmt.Sprint(team.ID), "version_id", fmt.Sprint(addResp.VersionID))
	})
}

func (s *integrationMDMTestSuite) TestAppStoreAppVersions() {
	t := s.T()
	s.setSkipWorkerJobs(t)
	ctx := context.Background()

	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "app-store-app-versions-team"})
	require.NoError(t, err)
	otherTeam, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "app-store-app-versions-other-team"})
	require.NoError(t, err)

	orgName := "Fleet Device Management Inc."
	token := "appstoreappversionstoken" //nolint:gosec // G101: test value, not a real credential
	expDate := time.Now().Add(200 * time.Hour).UTC().Round(time.Second).Format(fleet.VPPTimeFormat)
	tokenJSON := fmt.Sprintf(`{"expDate":%q,"token":%q,"orgName":%q}`, expDate, token, orgName)
	dev_mode.SetOverride("FLEET_DEV_VPP_URL", s.appleVPPConfigSrv.URL, t)

	// Adam ID "1" is macOS, "2" is iOS, "3" is iPadOS in the mock VPP server.
	const macOSAdamID = "1"
	const iosAdamID = "2"
	const ipadOSAdamID = "3"

	var validToken uploadVPPTokenResponse
	s.uploadDataViaForm("/api/latest/fleet/vpp_tokens", "token", "token.vpptoken",
		[]byte(base64.StdEncoding.EncodeToString([]byte(tokenJSON))), http.StatusAccepted, "", &validToken)

	// Assign the token to all fleets so the apps can be added to Unassigned too
	var resPatchVPP patchVPPTokensTeamsResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/vpp_tokens/%d/teams", validToken.Token.ID),
		patchVPPTokensTeamsRequest{TeamIDs: []uint{}}, http.StatusOK, &resPatchVPP)

	asJSONString := func(s string) json.RawMessage {
		b, err := json.Marshal(s)
		require.NoError(t, err)
		return json.RawMessage(b)
	}

	lastActivityVersionName := func(activityType string) string {
		var listActivities listActivitiesResponse
		s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &listActivities,
			"order_key", "id", "order_direction", "desc", "per_page", "1")
		require.Len(t, listActivities.Activities, 1)
		require.Equal(t, activityType, listActivities.Activities[0].Type)
		var details struct {
			VersionName string `json:"version_name"`
		}
		require.NoError(t, json.Unmarshal(*listActivities.Activities[0].Details, &details))
		return details.VersionName
	}

	const productionPlist = `<dict><key>ServerURL</key><string>https://production.example.com</string></dict>`
	const testPlist = `<dict><key>ServerURL</key><string>https://test.example.com</string></dict>`

	// add the iOS app without a name, the version should be named "Default version"
	var addDefaultResp addAppStoreAppResponse
	s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{
		TeamID:        &team.ID,
		AppStoreID:    iosAdamID,
		Platform:      fleet.IOSPlatform,
		Configuration: asJSONString(productionPlist),
	}, http.StatusOK, &addDefaultResp)
	require.NotZero(t, addDefaultResp.VersionID)
	require.Equal(t, fleet.DefaultAppStoreAppVersionName, lastActivityVersionName(fleet.ActivityAddedAppStoreApp{}.ActivityName()))
	titleID := addDefaultResp.TitleID

	// add the iOS app again with a new name, a second version should be added to the same title
	var addTestResp addAppStoreAppResponse
	s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{
		TeamID:        &team.ID,
		AppStoreID:    iosAdamID,
		Platform:      fleet.IOSPlatform,
		Name:          "Test",
		Configuration: asJSONString(testPlist),
	}, http.StatusOK, &addTestResp)
	require.Equal(t, titleID, addTestResp.TitleID)
	require.NotEqual(t, addDefaultResp.VersionID, addTestResp.VersionID)
	require.Equal(t, "Test", lastActivityVersionName(fleet.ActivityAddedAppStoreApp{}.ActivityName()))

	// add the iOS app with a name that already exists, the request should conflict
	s.Do("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{
		TeamID:     &team.ID,
		AppStoreID: iosAdamID,
		Platform:   fleet.IOSPlatform,
		Name:       "Test",
	}, http.StatusConflict)

	// add the iOS app with the existing name in a different case, the request should conflict
	s.Do("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{
		TeamID:     &team.ID,
		AppStoreID: iosAdamID,
		Platform:   fleet.IOSPlatform,
		Name:       "TEST",
	}, http.StatusConflict)

	// add the iOS app with the existing name written with an accent, the request should conflict instead of updating the existing version
	s.Do("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{
		TeamID:      &team.ID,
		AppStoreID:  iosAdamID,
		Platform:    fleet.IOSPlatform,
		Name:        "Tëst",
		SelfService: true,
	}, http.StatusConflict)

	// add the macOS app twice, the second version should conflict
	s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{
		TeamID:     &team.ID,
		AppStoreID: macOSAdamID,
		Platform:   fleet.MacOSPlatform,
	}, http.StatusOK, &addAppStoreAppResponse{})
	s.Do("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{
		TeamID:     &team.ID,
		AppStoreID: macOSAdamID,
		Platform:   fleet.MacOSPlatform,
		Name:       "Second",
	}, http.StatusConflict)

	// get the title, both versions should be listed and app_store_app should match the first-added version
	var titleResp getSoftwareTitleResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", titleID), nil, http.StatusOK, &titleResp,
		"fleet_id", fmt.Sprint(team.ID))
	require.Len(t, titleResp.SoftwareTitle.AppStoreApps, 2)
	require.Equal(t, addDefaultResp.VersionID, titleResp.SoftwareTitle.AppStoreApps[0].ID)
	require.Equal(t, fleet.DefaultAppStoreAppVersionName, titleResp.SoftwareTitle.AppStoreApps[0].Name)
	require.Equal(t, addTestResp.VersionID, titleResp.SoftwareTitle.AppStoreApps[1].ID)
	require.Equal(t, "Test", titleResp.SoftwareTitle.AppStoreApps[1].Name)
	var gotTestPlist string
	require.NoError(t, json.Unmarshal(titleResp.SoftwareTitle.AppStoreApps[1].Configuration, &gotTestPlist))
	require.Equal(t, testPlist, gotTestPlist)
	require.NotNil(t, titleResp.SoftwareTitle.AppStoreApp)
	require.Equal(t, iosAdamID, titleResp.SoftwareTitle.AppStoreApp.AdamID)
	var gotFirstPlist string
	require.NoError(t, json.Unmarshal(titleResp.SoftwareTitle.AppStoreApp.Configuration, &gotFirstPlist))
	require.Equal(t, productionPlist, gotFirstPlist)

	// edit the title without a version id, the request should be rejected since the title has two versions
	s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/app_store_app", titleID),
		&updateAppStoreAppRequest{TeamID: &team.ID, SelfService: new(true)}, http.StatusBadRequest)

	// edit the title with the version id of another title, the version should not be found
	var addIPadOSResp addAppStoreAppResponse
	s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{
		TeamID:     &team.ID,
		AppStoreID: ipadOSAdamID,
		Platform:   fleet.IPadOSPlatform,
	}, http.StatusOK, &addIPadOSResp)
	s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/app_store_app", titleID),
		&updateAppStoreAppRequest{TeamID: &team.ID, VersionID: &addIPadOSResp.VersionID, SelfService: new(true)}, http.StatusNotFound)

	// edit the title in another fleet with this fleet's version id, the version should not be found
	s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{
		TeamID:     &otherTeam.ID,
		AppStoreID: iosAdamID,
		Platform:   fleet.IOSPlatform,
	}, http.StatusOK, &addAppStoreAppResponse{})
	s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/app_store_app", titleID),
		&updateAppStoreAppRequest{TeamID: &otherTeam.ID, VersionID: &addTestResp.VersionID, SelfService: new(true)}, http.StatusNotFound)

	// rename the second version to the first version's name, the request should conflict
	s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/app_store_app", titleID),
		&updateAppStoreAppRequest{TeamID: &team.ID, VersionID: &addTestResp.VersionID, Name: new(fleet.DefaultAppStoreAppVersionName)}, http.StatusConflict)

	// rename the second version to the first version's name in a different case, the request should conflict
	s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/app_store_app", titleID),
		&updateAppStoreAppRequest{TeamID: &team.ID, VersionID: &addTestResp.VersionID, Name: new("DEFAULT VERSION")}, http.StatusConflict)

	// rename the second version to a blank name, the request should be rejected
	s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/app_store_app", titleID),
		&updateAppStoreAppRequest{TeamID: &team.ID, VersionID: &addTestResp.VersionID, Name: new("  ")}, http.StatusUnprocessableEntity)

	// add and rename a version to a 256 character name, both requests should be rejected with the length limit
	longName := strings.Repeat("a", fleet.MaxAppStoreAppVersionNameLength+1)
	res := s.Do("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{
		TeamID:     &team.ID,
		AppStoreID: iosAdamID,
		Platform:   fleet.IOSPlatform,
		Name:       longName,
	}, http.StatusUnprocessableEntity)
	require.Contains(t, extractServerErrorText(res.Body), "can't be longer than 255 characters")
	res = s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/app_store_app", titleID),
		&updateAppStoreAppRequest{TeamID: &team.ID, VersionID: &addTestResp.VersionID, Name: &longName}, http.StatusUnprocessableEntity)
	require.Contains(t, extractServerErrorText(res.Body), "can't be longer than 255 characters")

	// edit the second version's name, configuration, and auto updates, only the second version should change
	const editedTestPlist = `<dict><key>ServerURL</key><string>https://edited-test.example.com</string></dict>`
	var updResp updateAppStoreAppResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/app_store_app", titleID),
		&updateAppStoreAppRequest{
			TeamID:              &team.ID,
			VersionID:           &addTestResp.VersionID,
			Name:                new("Staging"),
			Configuration:       asJSONString(editedTestPlist),
			AutoUpdateEnabled:   new(true),
			AutoUpdateStartTime: new("01:00"),
			AutoUpdateEndTime:   new("03:00"),
		}, http.StatusOK, &updResp)
	require.NotNil(t, updResp.AppStoreApp)
	require.Equal(t, addTestResp.VersionID, updResp.AppStoreApp.ID)
	require.Equal(t, "Staging", updResp.AppStoreApp.Name)
	require.NotNil(t, updResp.AppStoreApp.AutoUpdateEnabled)
	require.True(t, *updResp.AppStoreApp.AutoUpdateEnabled)
	require.Equal(t, "Staging", lastActivityVersionName(fleet.ActivityEditedAppStoreApp{}.ActivityName()))

	titleResp = getSoftwareTitleResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", titleID), nil, http.StatusOK, &titleResp,
		"fleet_id", fmt.Sprint(team.ID))
	require.Len(t, titleResp.SoftwareTitle.AppStoreApps, 2)
	firstVersion := titleResp.SoftwareTitle.AppStoreApps[0]
	secondVersion := titleResp.SoftwareTitle.AppStoreApps[1]
	require.Equal(t, fleet.DefaultAppStoreAppVersionName, firstVersion.Name)
	require.Nil(t, firstVersion.AutoUpdateEnabled)
	require.NoError(t, json.Unmarshal(firstVersion.Configuration, &gotFirstPlist))
	require.Equal(t, productionPlist, gotFirstPlist)
	require.Equal(t, "Staging", secondVersion.Name)
	require.NotNil(t, secondVersion.AutoUpdateEnabled)
	require.True(t, *secondVersion.AutoUpdateEnabled)
	require.Equal(t, "01:00", *secondVersion.AutoUpdateStartTime)
	require.Equal(t, "03:00", *secondVersion.AutoUpdateEndTime)
	require.NoError(t, json.Unmarshal(secondVersion.Configuration, &gotTestPlist))
	require.Equal(t, editedTestPlist, gotTestPlist)

	// get the title without a fleet, no App Store app versions should be returned
	titleResp = getSoftwareTitleResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", titleID), nil, http.StatusOK, &titleResp)
	require.Nil(t, titleResp.SoftwareTitle.AppStoreApps)
	require.Nil(t, titleResp.SoftwareTitle.AppStoreApp)

	// rename the other fleet's only version, then apply GitOps with the app without versions, the renamed version should be replaced by a new Default version
	otherVersions, err := s.ds.GetAppStoreAppVersionsByTeamAndTitleID(ctx, otherTeam.ID, titleID)
	require.NoError(t, err)
	require.Len(t, otherVersions, 1)
	renamedVersionID := otherVersions[0].VPPAppsTeamsID
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/app_store_app", titleID),
		&updateAppStoreAppRequest{TeamID: &otherTeam.ID, Name: new("Renamed")}, http.StatusOK, &updateAppStoreAppResponse{})
	s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps/batch",
		batchAssociateAppStoreAppsRequest{Apps: []fleet.VPPBatchPayload{{AppStoreID: iosAdamID, Platform: fleet.IOSPlatform, SelfService: true}}},
		http.StatusOK, &batchAssociateAppStoreAppsResponse{}, "fleet_name", otherTeam.Name)
	otherVersions, err = s.ds.GetAppStoreAppVersionsByTeamAndTitleID(ctx, otherTeam.ID, titleID)
	require.NoError(t, err)
	require.Len(t, otherVersions, 1)
	require.Equal(t, fleet.DefaultAppStoreAppVersionName, otherVersions[0].VersionName)
	require.NotEqual(t, renamedVersionID, otherVersions[0].VPPAppsTeamsID)
	require.True(t, otherVersions[0].SelfService)

	// set a display name and a custom icon, then delete the second version, the first version should still show both
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/app_store_app", titleID),
		&updateAppStoreAppRequest{TeamID: &team.ID, VersionID: &addDefaultResp.VersionID, DisplayName: new("Renamed app")},
		http.StatusOK, &updateAppStoreAppResponse{})
	iconBytes, err := os.ReadFile("testdata/icons/valid-icon.png")
	require.NoError(t, err)
	iconBody, iconHeaders := generateMultipartRequest(t, "icon", "icon.png", iconBytes, s.token, nil)
	s.DoRawWithHeaders("PUT", fmt.Sprintf("/api/latest/fleet/software/titles/%d/icon?fleet_id=%d", titleID, team.ID),
		iconBody.Bytes(), http.StatusOK, iconHeaders)

	// delete the title with the version id of another title, the version should not be found
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/software/titles/%d/available_for_install", titleID), nil, http.StatusNotFound,
		"fleet_id", fmt.Sprint(team.ID), "version_id", fmt.Sprint(addIPadOSResp.VersionID))

	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/software/titles/%d/available_for_install", titleID), nil, http.StatusNoContent,
		"fleet_id", fmt.Sprint(team.ID), "version_id", fmt.Sprint(addTestResp.VersionID))
	require.Equal(t, "Staging", lastActivityVersionName(fleet.ActivityDeletedAppStoreApp{}.ActivityName()))

	titleResp = getSoftwareTitleResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", titleID), nil, http.StatusOK, &titleResp,
		"fleet_id", fmt.Sprint(team.ID))
	require.Len(t, titleResp.SoftwareTitle.AppStoreApps, 1)
	require.Equal(t, addDefaultResp.VersionID, titleResp.SoftwareTitle.AppStoreApps[0].ID)
	require.Equal(t, "Renamed app", titleResp.SoftwareTitle.AppStoreApps[0].DisplayName)
	_, err = s.ds.GetSoftwareTitleIcon(ctx, team.ID, titleID)
	require.NoError(t, err)

	// edit the title without a version id now that it has one version, the edit should apply to that version
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/app_store_app", titleID),
		&updateAppStoreAppRequest{TeamID: &team.ID, SelfService: new(true)}, http.StatusOK, &updateAppStoreAppResponse{})

	// add the iOS app to Unassigned under ten names, the eleventh version should be rejected
	for i := range fleet.MaxAppStoreAppVersions {
		s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{
			TeamID:     new(uint(0)),
			AppStoreID: iosAdamID,
			Platform:   fleet.IOSPlatform,
			Name:       fmt.Sprintf("Version %d", i),
		}, http.StatusOK, &addAppStoreAppResponse{})
	}
	s.Do("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{
		TeamID:     new(uint(0)),
		AppStoreID: iosAdamID,
		Platform:   fleet.IOSPlatform,
		Name:       "One too many",
	}, http.StatusBadRequest)

	// delete the title in Unassigned without a version id, every version should be deleted
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/software/titles/%d/available_for_install", titleID), nil, http.StatusNoContent,
		"fleet_id", "0")
	s.Do("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", titleID), nil, http.StatusNotFound, "fleet_id", "0")

	// delete the last version in the fleet, the display name and icon should be deleted with it
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/software/titles/%d/available_for_install", titleID), nil, http.StatusNoContent,
		"fleet_id", fmt.Sprint(team.ID), "version_id", fmt.Sprint(addDefaultResp.VersionID))
	_, err = s.ds.GetVPPAppMetadataByTeamAndTitleID(ctx, &team.ID, titleID)
	require.True(t, fleet.IsNotFound(err))
	var displayNameCount int
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &displayNameCount,
			`SELECT COUNT(*) FROM software_title_display_names WHERE team_id = ? AND software_title_id = ?`, team.ID, titleID)
	})
	require.Zero(t, displayNameCount)
	_, err = s.ds.GetSoftwareTitleIcon(ctx, team.ID, titleID)
	require.True(t, fleet.IsNotFound(err))

	// Auto-update coverage for the single-add endpoint (batch path already covered in TestBatchAppStoreAppVersions).
	addTeam, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "add-endpoint-auto-update"})
	require.NoError(t, err)

	// Valid 60+ min iOS window persists, and the activity carries the schedule.
	var addAutoResp addAppStoreAppResponse
	s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{
		TeamID:              &addTeam.ID,
		AppStoreID:          iosAdamID,
		Platform:            fleet.IOSPlatform,
		Name:                "Valid Window",
		AutoUpdateEnabled:   new(true),
		AutoUpdateStartTime: new("01:00"),
		AutoUpdateEndTime:   new("03:00"),
	}, http.StatusOK, &addAutoResp)
	addedVersions, err := s.ds.GetAppStoreAppVersionsByTeamAndTitleID(ctx, addTeam.ID, addAutoResp.TitleID)
	require.NoError(t, err)
	require.Len(t, addedVersions, 1)
	require.NotNil(t, addedVersions[0].AutoUpdateEnabled)
	require.True(t, *addedVersions[0].AutoUpdateEnabled)
	require.Equal(t, "01:00", *addedVersions[0].AutoUpdateStartTime)
	require.Equal(t, "03:00", *addedVersions[0].AutoUpdateEndTime)
	// Activity carries the window.
	var addActivities listActivitiesResponse
	s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &addActivities,
		"order_key", "id", "order_direction", "desc", "per_page", "1")
	require.Len(t, addActivities.Activities, 1)
	require.Equal(t, fleet.ActivityAddedAppStoreApp{}.ActivityName(), addActivities.Activities[0].Type)
	var addDetails struct {
		AutoUpdateEnabled   *bool   `json:"auto_update_enabled"`
		AutoUpdateStartTime *string `json:"auto_update_window_start"`
		AutoUpdateEndTime   *string `json:"auto_update_window_end"`
	}
	require.NoError(t, json.Unmarshal(*addActivities.Activities[0].Details, &addDetails))
	require.NotNil(t, addDetails.AutoUpdateEnabled)
	require.True(t, *addDetails.AutoUpdateEnabled)
	require.Equal(t, "01:00", *addDetails.AutoUpdateStartTime)
	require.Equal(t, "03:00", *addDetails.AutoUpdateEndTime)

	// Sub-hour window rejected at the service boundary.
	res = s.Do("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{
		TeamID:              &addTeam.ID,
		AppStoreID:          ipadOSAdamID,
		Platform:            fleet.IPadOSPlatform,
		Name:                "Short Window",
		AutoUpdateEnabled:   new(true),
		AutoUpdateStartTime: new("23:30"),
		AutoUpdateEndTime:   new("00:15"),
	}, http.StatusUnprocessableEntity)
	require.Contains(t, extractServerErrorText(res.Body), "at least one hour long")

	// Malformed times (not HH:MM) rejected at the service boundary.
	res = s.Do("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{
		TeamID:              &addTeam.ID,
		AppStoreID:          ipadOSAdamID,
		Platform:            fleet.IPadOSPlatform,
		Name:                "Malformed Window",
		AutoUpdateEnabled:   new(true),
		AutoUpdateStartTime: new("not-a-time"),
		AutoUpdateEndTime:   new("03:00"),
	}, http.StatusUnprocessableEntity)
	require.Contains(t, extractServerErrorText(res.Body), "HH:MM")

	// macOS is unsupported; auto-update fields are dropped silently.
	var addMacAutoResp addAppStoreAppResponse
	s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{
		TeamID:              &addTeam.ID,
		AppStoreID:          macOSAdamID,
		Platform:            fleet.MacOSPlatform,
		Name:                "macOS With Window",
		AutoUpdateEnabled:   new(true),
		AutoUpdateStartTime: new("01:00"),
		AutoUpdateEndTime:   new("03:00"),
	}, http.StatusOK, &addMacAutoResp)
	macVersions, err := s.ds.GetAppStoreAppVersionsByTeamAndTitleID(ctx, addTeam.ID, addMacAutoResp.TitleID)
	require.NoError(t, err)
	require.Len(t, macVersions, 1)
	require.Nil(t, macVersions[0].AutoUpdateEnabled, "macOS should not persist auto-update fields")
	require.Nil(t, macVersions[0].AutoUpdateStartTime)
	require.Nil(t, macVersions[0].AutoUpdateEndTime)
	// Activity for the macOS add should also omit the window.
	var macActivities listActivitiesResponse
	s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &macActivities,
		"order_key", "id", "order_direction", "desc", "per_page", "1")
	require.Len(t, macActivities.Activities, 1)
	require.Equal(t, fleet.ActivityAddedAppStoreApp{}.ActivityName(), macActivities.Activities[0].Type)
	var macDetails struct {
		AutoUpdateEnabled   *bool   `json:"auto_update_enabled"`
		AutoUpdateStartTime *string `json:"auto_update_window_start"`
		AutoUpdateEndTime   *string `json:"auto_update_window_end"`
	}
	require.NoError(t, json.Unmarshal(*macActivities.Activities[0].Details, &macDetails))
	require.Nil(t, macDetails.AutoUpdateEnabled)
	require.Nil(t, macDetails.AutoUpdateStartTime)
	require.Nil(t, macDetails.AutoUpdateEndTime)
}

func (s *integrationMDMTestSuite) TestBatchAppStoreAppVersions() {
	t := s.T()
	s.setSkipWorkerJobs(t)
	ctx := t.Context()

	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "batch-app-store-app-versions-team"})
	require.NoError(t, err)

	orgName := "Fleet Device Management Inc."
	token := "batchappstoreappversionstoken" //nolint:gosec // G101: test value, not a real credential
	expDate := time.Now().Add(200 * time.Hour).UTC().Round(time.Second).Format(fleet.VPPTimeFormat)
	tokenJSON := fmt.Sprintf(`{"expDate":%q,"token":%q,"orgName":%q}`, expDate, token, orgName)
	dev_mode.SetOverride("FLEET_DEV_VPP_URL", s.appleVPPConfigSrv.URL, t)

	// Adam ID "1" is macOS and "2" is iOS in the mock VPP server.
	const macOSAdamID = "1"
	const iosAdamID = "2"

	var validToken uploadVPPTokenResponse
	s.uploadDataViaForm("/api/latest/fleet/vpp_tokens", "token", "token.vpptoken",
		[]byte(base64.StdEncoding.EncodeToString([]byte(tokenJSON))), http.StatusAccepted, "", &validToken)

	// Assign the token to all fleets so the apps can be added to Unassigned too
	var resPatchVPP patchVPPTokensTeamsResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/vpp_tokens/%d/teams", validToken.Token.ID),
		patchVPPTokensTeamsRequest{TeamIDs: []uint{}}, http.StatusOK, &resPatchVPP)

	asJSONString := func(s string) json.RawMessage {
		b, err := json.Marshal(s)
		require.NoError(t, err)
		return json.RawMessage(b)
	}

	const productionPlist = `<dict><key>ServerURL</key><string>https://production.example.com</string></dict>`
	const testPlist = `<dict><key>ServerURL</key><string>https://test.example.com</string></dict>`
	productionVersion := fleet.VPPBatchPayload{
		AppStoreID: iosAdamID, Platform: fleet.IOSPlatform, VersionName: "Production", InstallDuringSetup: new(true),
		Configuration: asJSONString(productionPlist), AutoUpdateEnabled: new(true), AutoUpdateStartTime: new("01:00"), AutoUpdateEndTime: new("03:00"),
	}
	testVersion := fleet.VPPBatchPayload{
		AppStoreID: iosAdamID, Platform: fleet.IOSPlatform, VersionName: "Test", InstallDuringSetup: new(true),
		SelfService: true, Configuration: asJSONString(testPlist),
	}

	cases := []struct {
		name        string
		fleetID     uint
		queryParams []string
	}{
		{name: "no team", fleetID: 0, queryParams: nil},
		{name: "team", fleetID: team.ID, queryParams: []string{"fleet_name", team.Name}},
	}
	for _, c := range cases {
		t.Log(c.name)
		fleetID := c.fleetID

		// apply two versions of the iOS app, both should be added in list order and only the first should install during setup
		var batchResp batchAssociateAppStoreAppsResponse
		s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps/batch",
			batchAssociateAppStoreAppsRequest{Apps: []fleet.VPPBatchPayload{productionVersion, testVersion}},
			http.StatusOK, &batchResp, c.queryParams...)
		require.Len(t, batchResp.Apps, 2)
		titleID := *batchResp.Apps[0].TitleID
		versions, err := s.ds.GetAppStoreAppVersionsByTeamAndTitleID(ctx, fleetID, titleID)
		require.NoError(t, err)
		require.Len(t, versions, 2)
		productionID := versions[0].VPPAppsTeamsID
		testID := versions[1].VPPAppsTeamsID
		require.Equal(t, "Production", versions[0].VersionName)
		require.False(t, versions[0].SelfService)
		require.Equal(t, productionPlist, string(versions[0].Configuration))
		require.True(t, *versions[0].AutoUpdateEnabled)
		require.Equal(t, "01:00", *versions[0].AutoUpdateStartTime)
		require.Equal(t, "Test", versions[1].VersionName)
		require.True(t, versions[1].SelfService)
		require.Equal(t, testPlist, string(versions[1].Configuration))
		require.Nil(t, versions[1].AutoUpdateEnabled)
		forSetup, err := s.ds.GetVPPAppsToInstallDuringSetupExperience(ctx, &fleetID, string(fleet.IOSPlatform))
		require.NoError(t, err)
		require.Len(t, forSetup, 1)
		require.Equal(t, productionID, forSetup[0].AppTeamID)

		// apply the versions in reverse order, the versions should keep their ids and order
		s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps/batch",
			batchAssociateAppStoreAppsRequest{Apps: []fleet.VPPBatchPayload{testVersion, productionVersion}},
			http.StatusOK, &batchResp, c.queryParams...)
		versions, err = s.ds.GetAppStoreAppVersionsByTeamAndTitleID(ctx, fleetID, titleID)
		require.NoError(t, err)
		require.Len(t, versions, 2)
		require.Equal(t, productionID, versions[0].VPPAppsTeamsID)
		require.Equal(t, testID, versions[1].VPPAppsTeamsID)
		require.True(t, *versions[0].AutoUpdateEnabled)

		// dry run two versions with the same name in a different case, the request should fail
		caseVersion := testVersion
		caseVersion.VersionName = "test"
		res := s.Do("POST", "/api/latest/fleet/software/app_store_apps/batch",
			batchAssociateAppStoreAppsRequest{Apps: []fleet.VPPBatchPayload{testVersion, caseVersion}},
			http.StatusUnprocessableEntity, append([]string{"dry_run", "true"}, c.queryParams...)...)
		require.Contains(t, extractServerErrorText(res.Body), `More than one version is named "test"`)

		// dry run Test and Tëst, both names match the stored Test version in MySQL so the request should fail
		accentVersion := testVersion
		accentVersion.VersionName = "Tëst"
		res = s.Do("POST", "/api/latest/fleet/software/app_store_apps/batch",
			batchAssociateAppStoreAppsRequest{Apps: []fleet.VPPBatchPayload{productionVersion, testVersion, accentVersion}},
			http.StatusUnprocessableEntity, append([]string{"dry_run", "true"}, c.queryParams...)...)
		require.Contains(t, extractServerErrorText(res.Body), "More than one version is named")

		// dry run two new versions named Beta and Bêta, MySQL compares the names as equal so the request should fail
		betaVersion := fleet.VPPBatchPayload{AppStoreID: iosAdamID, Platform: fleet.IOSPlatform, VersionName: "Beta"}
		betaAccentVersion := fleet.VPPBatchPayload{AppStoreID: iosAdamID, Platform: fleet.IOSPlatform, VersionName: "Bêta"}
		res = s.Do("POST", "/api/latest/fleet/software/app_store_apps/batch",
			batchAssociateAppStoreAppsRequest{Apps: []fleet.VPPBatchPayload{productionVersion, testVersion, betaVersion, betaAccentVersion}},
			http.StatusUnprocessableEntity, append([]string{"dry_run", "true"}, c.queryParams...)...)
		require.Contains(t, extractServerErrorText(res.Body), `More than one version is named "Bêta"`)

		// apply the Test version renamed to tëst, the Test version should keep its id and take the new name
		renamedTestVersion := testVersion
		renamedTestVersion.VersionName = "tëst"
		s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps/batch",
			batchAssociateAppStoreAppsRequest{Apps: []fleet.VPPBatchPayload{productionVersion, renamedTestVersion}},
			http.StatusOK, &batchResp, c.queryParams...)
		versions, err = s.ds.GetAppStoreAppVersionsByTeamAndTitleID(ctx, fleetID, titleID)
		require.NoError(t, err)
		require.Len(t, versions, 2)
		require.Equal(t, testID, versions[1].VPPAppsTeamsID)
		require.Equal(t, "tëst", versions[1].VersionName)

		// apply only the Production version, the Test version should be removed
		s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps/batch",
			batchAssociateAppStoreAppsRequest{Apps: []fleet.VPPBatchPayload{productionVersion}},
			http.StatusOK, &batchResp, c.queryParams...)
		versions, err = s.ds.GetAppStoreAppVersionsByTeamAndTitleID(ctx, fleetID, titleID)
		require.NoError(t, err)
		require.Len(t, versions, 1)
		require.Equal(t, productionID, versions[0].VPPAppsTeamsID)

		// dry run the iOS app as an entry without versions and an entry with versions, the request should fail
		unnamedVersion := testVersion
		unnamedVersion.VersionName = ""
		res = s.Do("POST", "/api/latest/fleet/software/app_store_apps/batch",
			batchAssociateAppStoreAppsRequest{Apps: []fleet.VPPBatchPayload{unnamedVersion, testVersion}},
			http.StatusUnprocessableEntity, append([]string{"dry_run", "true"}, c.queryParams...)...)
		require.Contains(t, extractServerErrorText(res.Body), "The app has an entry without versions and an entry with versions")

		// apply eleven versions of the iOS app, the request should fail
		var elevenVersions []fleet.VPPBatchPayload
		for i := range 11 {
			elevenVersions = append(elevenVersions, fleet.VPPBatchPayload{AppStoreID: iosAdamID, Platform: fleet.IOSPlatform, VersionName: fmt.Sprintf("Version %d", i)})
		}
		res = s.Do("POST", "/api/latest/fleet/software/app_store_apps/batch",
			batchAssociateAppStoreAppsRequest{Apps: elevenVersions}, http.StatusUnprocessableEntity, c.queryParams...)
		require.Contains(t, extractServerErrorText(res.Body), "An app can have at most 10 versions per fleet")

		// apply two versions of the macOS app, the request should fail
		res = s.Do("POST", "/api/latest/fleet/software/app_store_apps/batch",
			batchAssociateAppStoreAppsRequest{Apps: []fleet.VPPBatchPayload{
				{AppStoreID: macOSAdamID, Platform: fleet.MacOSPlatform, VersionName: "Production"},
				{AppStoreID: macOSAdamID, Platform: fleet.MacOSPlatform, VersionName: "Test"},
			}}, http.StatusUnprocessableEntity, c.queryParams...)
		require.Contains(t, extractServerErrorText(res.Body), "macOS App Store apps can only have one version")

		// apply the iOS app without a version name, both versions should be replaced by a new Default version that installs during setup
		s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps/batch",
			batchAssociateAppStoreAppsRequest{Apps: []fleet.VPPBatchPayload{{AppStoreID: iosAdamID, Platform: fleet.IOSPlatform, InstallDuringSetup: new(true)}}},
			http.StatusOK, &batchResp, c.queryParams...)
		versions, err = s.ds.GetAppStoreAppVersionsByTeamAndTitleID(ctx, fleetID, titleID)
		require.NoError(t, err)
		require.Len(t, versions, 1)
		require.Equal(t, fleet.DefaultAppStoreAppVersionName, versions[0].VersionName)
		require.NotEqual(t, productionID, versions[0].VPPAppsTeamsID)
		require.NotEqual(t, testID, versions[0].VPPAppsTeamsID)
		forSetup, err = s.ds.GetVPPAppsToInstallDuringSetupExperience(ctx, &fleetID, string(fleet.IOSPlatform))
		require.NoError(t, err)
		require.Len(t, forSetup, 1)
		require.Equal(t, versions[0].VPPAppsTeamsID, forSetup[0].AppTeamID)

		// apply no apps, every version of the iOS app should be deleted
		s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps/batch",
			batchAssociateAppStoreAppsRequest{Apps: []fleet.VPPBatchPayload{}}, http.StatusOK, &batchResp, c.queryParams...)
		_, err = s.ds.GetAppStoreAppVersionsByTeamAndTitleID(ctx, fleetID, titleID)
		require.True(t, fleet.IsNotFound(err))
	}
}

func (s *integrationMDMTestSuite) TestAppStoreAppVersionHostPrecedence() {
	t := s.T()
	s.setSkipWorkerJobs(t)
	ctx := t.Context()

	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "app-store-app-version-precedence-team"})
	require.NoError(t, err)
	s.setVPPTokenForTeam(team.ID)
	s.registerResetVPPProxyData(t)

	labelA, err := s.ds.NewLabel(ctx, &fleet.Label{Name: "precedence-label-a", LabelMembershipType: fleet.LabelMembershipTypeManual})
	require.NoError(t, err)
	labelB, err := s.ds.NewLabel(ctx, &fleet.Label{Name: "precedence-label-b", LabelMembershipType: fleet.LabelMembershipTypeManual})
	require.NoError(t, err)

	// Adam ID "2" is registered for iOS in the mock proxy.
	const iosAdamID = "2"
	const configA = `<dict><key>version</key><string>configuration-a</string></dict>`
	const configB = `<dict><key>version</key><string>configuration-b</string></dict>`
	configAJSON, err := json.Marshal(configA)
	require.NoError(t, err)
	configBJSON, err := json.Marshal(configB)
	require.NoError(t, err)

	// add version A scoped to label A, then version B scoped to label B, both self-service
	var addAResp addAppStoreAppResponse
	s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{
		TeamID: &team.ID, AppStoreID: iosAdamID, Platform: fleet.IOSPlatform, SelfService: true,
		LabelsIncludeAny: []string{labelA.Name}, Configuration: configAJSON,
	}, http.StatusOK, &addAResp)
	var addBResp addAppStoreAppResponse
	s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{
		TeamID: &team.ID, AppStoreID: iosAdamID, Platform: fleet.IOSPlatform, SelfService: true, Name: "Version B",
		LabelsIncludeAny: []string{labelB.Name}, Configuration: configBJSON,
	}, http.StatusOK, &addBResp)
	require.Equal(t, addAResp.TitleID, addBResp.TitleID)
	titleID := addAResp.TitleID

	hostInA, devInA := s.createAppleMobileHostThenEnrollMDM("ios")
	hostInAAndB, devInAAndB := s.createAppleMobileHostThenEnrollMDM("ios")
	hostInB, devInB := s.createAppleMobileHostThenEnrollMDM("ios")
	hostInNeither, _ := s.createAppleMobileHostThenEnrollMDM("ios")
	s.appleVPPConfigSrvConfig.SerialNumbers = append(s.appleVPPConfigSrvConfig.SerialNumbers,
		devInA.SerialNumber, devInAAndB.SerialNumber, devInB.SerialNumber)
	s.Do("POST", "/api/latest/fleet/hosts/transfer", &addHostsToTeamRequest{
		HostIDs: []uint{hostInA.ID, hostInAAndB.ID, hostInB.ID, hostInNeither.ID}, TeamID: &team.ID,
	}, http.StatusOK)
	require.NoError(t, s.ds.AddLabelsToHost(ctx, hostInA.ID, []uint{labelA.ID}))
	require.NoError(t, s.ds.AddLabelsToHost(ctx, hostInAAndB.ID, []uint{labelA.ID, labelB.ID}))
	require.NoError(t, s.ds.AddLabelsToHost(ctx, hostInB.ID, []uint{labelB.ID}))

	// install the title on the host in scope for A and B, the command should carry version A's configuration
	s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", hostInAAndB.ID, titleID), &installSoftwareRequest{}, http.StatusAccepted)
	s.awaitRunAppleMDMWorkerSchedule()
	s.runWorker()
	cmd, err := devInAAndB.Idle()
	require.NoError(t, err)
	require.NotNil(t, cmd)
	require.Equal(t, "InstallApplication", cmd.Command.RequestType)
	require.Contains(t, string(cmd.Raw), "configuration-a")
	_, err = devInAAndB.Acknowledge(cmd.CommandUUID)
	require.NoError(t, err)

	// install the title on the host in scope for A only, the command should carry version A's configuration
	s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", hostInA.ID, titleID), &installSoftwareRequest{}, http.StatusAccepted)
	s.awaitRunAppleMDMWorkerSchedule()
	s.runWorker()
	cmd, err = devInA.Idle()
	require.NoError(t, err)
	require.NotNil(t, cmd)
	require.Equal(t, "InstallApplication", cmd.Command.RequestType)
	require.Contains(t, string(cmd.Raw), "configuration-a")
	_, err = devInA.Acknowledge(cmd.CommandUUID)
	require.NoError(t, err)

	// install the title through self-service on the host in scope for B only, the command should carry version B's configuration
	s.DoRawNoAuth("POST", fmt.Sprintf("/api/latest/fleet/device/%s/software/install/%d", hostInB.UUID, titleID), nil, http.StatusAccepted)
	s.awaitRunAppleMDMWorkerSchedule()
	s.runWorker()
	cmd, err = devInB.Idle()
	require.NoError(t, err)
	require.NotNil(t, cmd)
	require.Equal(t, "InstallApplication", cmd.Command.RequestType)
	require.Contains(t, string(cmd.Raw), "configuration-b")
	require.NotContains(t, string(cmd.Raw), "configuration-a")
	_, err = devInB.Acknowledge(cmd.CommandUUID)
	require.NoError(t, err)

	// list the software of the host in scope for B only, the title should be listed once with version B's name
	var hostSoftwareResp getHostSoftwareResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", hostInB.ID), nil, http.StatusOK, &hostSoftwareResp,
		"available_for_install", "true")
	require.Len(t, hostSoftwareResp.Software, 1)
	require.Equal(t, titleID, hostSoftwareResp.Software[0].ID)
	require.NotNil(t, hostSoftwareResp.Software[0].AppStoreApp)
	require.Equal(t, "Version B", hostSoftwareResp.Software[0].AppStoreApp.VersionName)
	require.Equal(t, addBResp.VersionID, hostSoftwareResp.Software[0].AppStoreApp.VersionID)

	// remove label A from the host that installed version A, the title should be listed with version B, the version the host is scoped to
	require.NoError(t, s.ds.RemoveLabelsFromHost(ctx, hostInAAndB.ID, []uint{labelA.ID}))
	hostSoftwareResp = getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", hostInAAndB.ID), nil, http.StatusOK, &hostSoftwareResp,
		"available_for_install", "true")
	require.Len(t, hostSoftwareResp.Software, 1)
	require.NotNil(t, hostSoftwareResp.Software[0].AppStoreApp)
	require.Equal(t, "Version B", hostSoftwareResp.Software[0].AppStoreApp.VersionName)
	require.Equal(t, addBResp.VersionID, hostSoftwareResp.Software[0].AppStoreApp.VersionID)

	// install the title on the host in scope for neither version, the request should be rejected
	res := s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", hostInNeither.ID, titleID), &installSoftwareRequest{}, http.StatusBadRequest)
	require.Contains(t, extractServerErrorText(res.Body), "isn't a member of the labels")

	// install the title through self-service on the host in scope for neither version, the request should be rejected
	res = s.DoRawNoAuth("POST", fmt.Sprintf("/api/latest/fleet/device/%s/software/install/%d", hostInNeither.UUID, titleID), nil, http.StatusBadRequest)
	require.Contains(t, extractServerErrorText(res.Body), "not available for this host")

	// get the title, version A should count two pending installs and version B one
	var titleResp getSoftwareTitleResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", titleID), nil, http.StatusOK, &titleResp,
		"fleet_id", fmt.Sprint(team.ID))
	require.Len(t, titleResp.SoftwareTitle.AppStoreApps, 2)
	require.Equal(t, addAResp.VersionID, titleResp.SoftwareTitle.AppStoreApps[0].ID)
	require.NotNil(t, titleResp.SoftwareTitle.AppStoreApps[0].Status)
	require.Equal(t, uint(2), titleResp.SoftwareTitle.AppStoreApps[0].Status.Pending)
	require.Equal(t, addBResp.VersionID, titleResp.SoftwareTitle.AppStoreApps[1].ID)
	require.NotNil(t, titleResp.SoftwareTitle.AppStoreApps[1].Status)
	require.Equal(t, uint(1), titleResp.SoftwareTitle.AppStoreApps[1].Status.Pending)
}

func (s *integrationMDMTestSuite) TestVPPAppConfigurationReconciliation() {
	t := s.T()
	s.setSkipWorkerJobs(t)
	ctx := t.Context()
	s.registerResetVPPProxyData(t)

	s.setVPPTokenForTeam(0)
	var tokensResp getVPPTokensResponse
	s.DoJSON("GET", "/api/latest/fleet/vpp_tokens", &getVPPTokensRequest{}, http.StatusOK, &tokensResp)
	vppTokenID := tokensResp.Tokens[0].ID
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/vpp_tokens/%d/teams", vppTokenID),
		patchVPPTokensTeamsRequest{TeamIDs: []uint{}}, http.StatusOK, &patchVPPTokensTeamsResponse{})

	// Adam ID "2" is registered for iOS in the mock proxy. Adam ID "3" is changed to iOS, it is installed without a
	// configuration to keep the host busy so an install of app "2" waits in the host's queue.
	const adamID = "2"
	const otherAdamID = "3"
	s.appleVPPProxySrvData = maps.Clone(s.appleVPPProxySrvData)
	s.appleVPPProxySrvData[otherAdamID] = `{"id": "3", "attributes": {"name": "App 3", "platformAttributes": {"ios": {"bundleId": "c-3", "artwork": {"url": "https://example.com/images/3/{w}x{h}.{f}"}, "latestVersionInfo": {"versionDisplay": "3.0.0"}}}, "deviceFamilies": ["iphone"]}}`
	appInInventory := fleet.Software{Name: "App 2", BundleIdentifier: "b-2", Version: "2.0.0", Source: "ios_apps", Installed: true}
	otherAppInInventory := fleet.Software{Name: "App 3", BundleIdentifier: "c-3", Version: "3.0.0", Source: "ios_apps", Installed: true}
	configurationValueRegexp := regexp.MustCompile(`<key>value</key><string>([^<]*)</string>`)
	appInstallRegexp := regexp.MustCompile(`<key>iTunesStoreID</key>\s*<integer>` + adamID + `</integer>`)

	configurationJSON := func(value string) json.RawMessage {
		b, err := json.Marshal(fmt.Sprintf(`<dict><key>value</key><string>%s</string></dict>`, value))
		require.NoError(t, err)
		return b
	}

	newFleet := func(t *testing.T) *fleet.Team {
		team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name()})
		require.NoError(t, err)
		return team
	}

	newManualLabel := func(t *testing.T, name string) *fleet.Label {
		label, err := s.ds.NewLabel(ctx, &fleet.Label{Name: t.Name() + " " + name, LabelMembershipType: fleet.LabelMembershipTypeManual})
		require.NoError(t, err)
		return label
	}

	newHostVitalsLabel := func(t *testing.T, criteria *fleet.HostVitalCriteria) *fleet.Label {
		var labelResp fleet.CreateLabelResponse
		s.DoJSON("POST", "/api/latest/fleet/labels", &fleet.CreateLabelRequest{
			Name: t.Name() + " host vitals", Criteria: criteria,
		}, http.StatusOK, &labelResp)
		label, _, err := s.ds.Label(ctx, labelResp.Label.ID, fleet.TeamFilter{User: test.UserAdmin})
		require.NoError(t, err)
		return &label.Label
	}

	runHostVitalsLabelCron := func(t *testing.T, label *fleet.Label) {
		storedLabel, _, err := s.ds.Label(ctx, label.ID, fleet.TeamFilter{User: test.UserAdmin})
		require.NoError(t, err)
		_, _, err = s.ds.UpdateLabelMembershipByHostCriteria(ctx, storedLabel)
		require.NoError(t, err)
	}

	addVersion := func(t *testing.T, teamID uint, name string, configurationValue string, labelsIncludeAny []string) addAppStoreAppResponse {
		var resp addAppStoreAppResponse
		s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{
			TeamID: &teamID, AppStoreID: adamID, Platform: fleet.IOSPlatform, Name: name,
			LabelsIncludeAny: labelsIncludeAny, Configuration: configurationJSON(configurationValue),
		}, http.StatusOK, &resp)
		return resp
	}

	// addVersionAInLabelAndVersionB adds version A scoped to the label with configuration "configuration-a", then version
	// B without labels with configuration "configuration-b", and returns version A
	addVersionAInLabelAndVersionB := func(t *testing.T, teamID uint, labelName string) addAppStoreAppResponse {
		versionA := addVersion(t, teamID, "A", "configuration-a", []string{labelName})
		addVersion(t, teamID, "B", "configuration-b", nil)
		return versionA
	}

	addVersionWithRawConfiguration := func(t *testing.T, teamID uint, configuration string) addAppStoreAppResponse {
		b, err := json.Marshal(configuration)
		require.NoError(t, err)
		var resp addAppStoreAppResponse
		s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{
			TeamID: &teamID, AppStoreID: adamID, Platform: fleet.IOSPlatform, Configuration: b,
		}, http.StatusOK, &resp)
		return resp
	}

	editConfiguration := func(t *testing.T, teamID uint, version addAppStoreAppResponse, configuration json.RawMessage) {
		s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/app_store_app", version.TitleID), &updateAppStoreAppRequest{
			TeamID: &teamID, VersionID: &version.VersionID, Configuration: configuration,
		}, http.StatusOK, &updateAppStoreAppResponse{})
	}

	// configurationsReceived acknowledges every command the device has, and returns the configuration value of each
	// InstallApplication command of app "2", an empty value for a command without a configuration
	configurationsReceived := func(t *testing.T, device *mdmtest.TestAppleMDMClient) []string {
		var configurations []string
		for {
			cmd, err := device.Idle()
			require.NoError(t, err)
			if cmd == nil {
				s.runWorkerUntilDone()
				cmd, err = device.Idle()
				require.NoError(t, err)
				if cmd == nil {
					return configurations
				}
			}
			switch cmd.Command.RequestType {
			case "InstallApplication":
				if appInstallRegexp.Match(cmd.Raw) {
					var value string
					match := configurationValueRegexp.FindStringSubmatch(string(cmd.Raw))
					if match != nil {
						value = match[1]
					}
					configurations = append(configurations, value)
				}
				_, err = device.Acknowledge(cmd.CommandUUID)
			case "InstalledApplicationList":
				_, err = device.AcknowledgeInstalledApplicationList(device.UUID, cmd.CommandUUID, []fleet.Software{appInInventory, otherAppInInventory})
			case "CertificateList":
				_, err = device.AcknowledgeCertificateList(device.UUID, cmd.CommandUUID, nil)
			case "DeviceInformation":
				_, err = device.AcknowledgeDeviceInformation(device.UUID, cmd.CommandUUID, "iPhone", "iPhone", "America/Los_Angeles")
			default:
				_, err = device.Acknowledge(cmd.CommandUUID)
			}
			require.NoError(t, err)
		}
	}

	newHost := func(t *testing.T, teamID *uint, labels ...*fleet.Label) (*fleet.Host, *mdmtest.TestAppleMDMClient) {
		host, device := s.createAppleMobileHostThenEnrollMDM("ios")
		s.appleVPPConfigSrvConfig.SerialNumbers = append(s.appleVPPConfigSrvConfig.SerialNumbers, device.SerialNumber)
		if teamID != nil {
			s.Do("POST", "/api/latest/fleet/hosts/transfer", &addHostsToTeamRequest{HostIDs: []uint{host.ID}, TeamID: teamID}, http.StatusOK)
		}
		s.awaitRunAppleMDMWorkerSchedule()
		configurationsReceived(t, device)
		for _, label := range labels {
			require.NoError(t, s.ds.AddLabelsToHost(ctx, host.ID, []uint{label.ID}))
		}
		return host, device
	}

	installApp := func(t *testing.T, host *fleet.Host, device *mdmtest.TestAppleMDMClient, titleID uint) []string {
		// Wait for the next second so the install isn't recorded in the same second as the configuration write, install
		// times have whole seconds and would compare as older than the configuration
		time.Sleep(time.Until(time.Now().Truncate(time.Second).Add(time.Second)))
		s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", host.ID, titleID), &installSoftwareRequest{}, http.StatusAccepted)
		s.awaitRunAppleMDMWorkerSchedule()
		configurations := configurationsReceived(t, device)
		_, err := s.ds.UpdateHostSoftware(ctx, host.ID, []fleet.Software{appInInventory})
		require.NoError(t, err)
		return configurations
	}

	// queueInstallBehindOtherApp queues an install of app "3" that the device doesn't fetch, then an install of app "2"
	// that waits behind it in the host's queue
	queueInstallBehindOtherApp := func(t *testing.T, teamID uint, host *fleet.Host, titleID uint) {
		var otherAppResp addAppStoreAppResponse
		s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{
			TeamID: &teamID, AppStoreID: otherAdamID, Platform: fleet.IOSPlatform,
		}, http.StatusOK, &otherAppResp)
		s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", host.ID, otherAppResp.TitleID), &installSoftwareRequest{}, http.StatusAccepted)
		s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", host.ID, titleID), &installSoftwareRequest{}, http.StatusAccepted)
	}

	refetch := func(t *testing.T, host *fleet.Host) {
		s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/refetch", host.ID), nil, http.StatusOK)
	}

	// reconcile runs the work Fleet does in the background to send configurations
	reconcile := func() {
		s.runWorkerUntilDone()
		premiumCtx := license.NewContext(ctx, &fleet.LicenseInfo{Tier: fleet.TierPremium})
		err := eeservice.ReconcileAppStoreAppConfigurations(premiumCtx, s.ds, s.fleetSvc.(fleet.AppleMDMVPPInstaller), s.logger)
		require.NoError(t, err)
		s.runWorkerUntilDone()
	}

	reconcileAndReceive := func(t *testing.T, device *mdmtest.TestAppleMDMClient) []string {
		reconcile()
		return configurationsReceived(t, device)
	}

	t.Run("editing the configuration of a host's version sends the new configuration to the host", func(t *testing.T) {
		team := newFleet(t)
		version := addVersion(t, team.ID, "Only", "configuration-before", nil)
		host, device := newHost(t, &team.ID)
		require.Equal(t, []string{"configuration-before"}, installApp(t, host, device, version.TitleID))

		// edit the version's configuration, the host should get the new configuration
		editConfiguration(t, team.ID, version, configurationJSON("configuration-after"))
		require.Equal(t, []string{"configuration-after"}, reconcileAndReceive(t, device))
	})

	t.Run("clearing the configuration of a host's version sends an install without a configuration to the host", func(t *testing.T) {
		team := newFleet(t)
		version := addVersion(t, team.ID, "Only", "configuration-before", nil)
		host, device := newHost(t, &team.ID)
		require.Equal(t, []string{"configuration-before"}, installApp(t, host, device, version.TitleID))

		// clear the version's configuration, the host should get an install without a configuration
		editConfiguration(t, team.ID, version, json.RawMessage(`null`))
		require.Equal(t, []string{""}, reconcileAndReceive(t, device))
	})

	t.Run("editing a version's labels so a host moves to that version sends the version's configuration to the host", func(t *testing.T) {
		team := newFleet(t)
		labelA := newManualLabel(t, "a")
		labelB := newManualLabel(t, "b")
		versionA := addVersionAInLabelAndVersionB(t, team.ID, labelA.Name)
		host, device := newHost(t, &team.ID, labelB)
		require.Equal(t, []string{"configuration-b"}, installApp(t, host, device, versionA.TitleID))

		// add label B to version A, the host should get version A's configuration
		s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/app_store_app", versionA.TitleID), &updateAppStoreAppRequest{
			TeamID: &team.ID, VersionID: &versionA.VersionID, LabelsIncludeAny: []string{labelA.Name, labelB.Name},
		}, http.StatusOK, &updateAppStoreAppResponse{})
		require.Equal(t, []string{"configuration-a"}, reconcileAndReceive(t, device))
	})

	t.Run("deleting a host's version sends the configuration of the host's next version", func(t *testing.T) {
		team := newFleet(t)
		labelA := newManualLabel(t, "a")
		versionA := addVersionAInLabelAndVersionB(t, team.ID, labelA.Name)
		host, device := newHost(t, &team.ID, labelA)
		require.Equal(t, []string{"configuration-a"}, installApp(t, host, device, versionA.TitleID))

		// delete version A, the host should get version B's configuration
		s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/software/titles/%d/available_for_install", versionA.TitleID), nil, http.StatusNoContent,
			"fleet_id", fmt.Sprint(team.ID), "version_id", fmt.Sprint(versionA.VersionID))
		require.Equal(t, []string{"configuration-b"}, reconcileAndReceive(t, device))
	})

	t.Run("moving a host to a version without a configuration sends an install without a configuration", func(t *testing.T) {
		team := newFleet(t)
		labelA := newManualLabel(t, "a")
		versionA := addVersion(t, team.ID, "A", "configuration-a", []string{labelA.Name})
		s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{
			TeamID: &team.ID, AppStoreID: adamID, Platform: fleet.IOSPlatform, Name: "B",
		}, http.StatusOK, &addAppStoreAppResponse{})
		host, device := newHost(t, &team.ID, labelA)
		require.Equal(t, []string{"configuration-a"}, installApp(t, host, device, versionA.TitleID))

		// remove label A from the host, the host should get version B without a configuration
		require.NoError(t, s.ds.RemoveLabelsFromHost(ctx, host.ID, []uint{labelA.ID}))
		require.Equal(t, []string{""}, reconcileAndReceive(t, device))
	})

	t.Run("moving a host between versions without configurations sends nothing", func(t *testing.T) {
		team := newFleet(t)
		labelA := newManualLabel(t, "a")
		var versionA addAppStoreAppResponse
		s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{
			TeamID: &team.ID, AppStoreID: adamID, Platform: fleet.IOSPlatform, Name: "A", LabelsIncludeAny: []string{labelA.Name},
		}, http.StatusOK, &versionA)
		s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{
			TeamID: &team.ID, AppStoreID: adamID, Platform: fleet.IOSPlatform, Name: "B",
		}, http.StatusOK, &addAppStoreAppResponse{})
		host, device := newHost(t, &team.ID, labelA)
		require.Equal(t, []string{""}, installApp(t, host, device, versionA.TitleID))

		// remove label A from the host, the host should get nothing since neither version has a configuration
		require.NoError(t, s.ds.RemoveLabelsFromHost(ctx, host.ID, []uint{labelA.ID}))
		require.Empty(t, reconcileAndReceive(t, device))
	})

	t.Run("replacing a manual label's hosts sends the configuration of the host's next version", func(t *testing.T) {
		team := newFleet(t)
		labelA := newManualLabel(t, "a")
		versionA := addVersionAInLabelAndVersionB(t, team.ID, labelA.Name)
		host, device := newHost(t, &team.ID, labelA)
		otherHost, _ := newHost(t, &team.ID)
		require.Equal(t, []string{"configuration-a"}, installApp(t, host, device, versionA.TitleID))

		// replace label A's hosts with another host, the host should get version B's configuration
		s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/labels/%d", labelA.ID), &fleet.ModifyLabelPayload{HostIDs: []uint{otherHost.ID}}, http.StatusOK, &fleet.ModifyLabelResponse{})
		require.Equal(t, []string{"configuration-b"}, reconcileAndReceive(t, device))
	})

	t.Run("clearing a manual label's hosts sends the configuration of the host's next version", func(t *testing.T) {
		team := newFleet(t)
		labelA := newManualLabel(t, "a")
		versionA := addVersionAInLabelAndVersionB(t, team.ID, labelA.Name)
		host, device := newHost(t, &team.ID, labelA)
		require.Equal(t, []string{"configuration-a"}, installApp(t, host, device, versionA.TitleID))

		// clear label A's hosts, the host should get version B's configuration
		s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/labels/%d", labelA.ID), &fleet.ModifyLabelPayload{Hosts: []string{}}, http.StatusOK, &fleet.ModifyLabelResponse{})
		require.Equal(t, []string{"configuration-b"}, reconcileAndReceive(t, device))
	})

	t.Run("adding a label to a host sends the configuration of the label's version", func(t *testing.T) {
		team := newFleet(t)
		labelA := newManualLabel(t, "a")
		versionA := addVersionAInLabelAndVersionB(t, team.ID, labelA.Name)
		host, device := newHost(t, &team.ID)
		require.Equal(t, []string{"configuration-b"}, installApp(t, host, device, versionA.TitleID))

		// add label A to the host, the host should get version A's configuration
		s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", host.ID), addLabelsToHostRequest{Labels: []string{labelA.Name}}, http.StatusOK, &addLabelsToHostResponse{})
		require.Equal(t, []string{"configuration-a"}, reconcileAndReceive(t, device))
	})

	t.Run("removing a label from a host sends the configuration of the host's next version", func(t *testing.T) {
		team := newFleet(t)
		labelA := newManualLabel(t, "a")
		versionA := addVersionAInLabelAndVersionB(t, team.ID, labelA.Name)
		host, device := newHost(t, &team.ID, labelA)
		require.Equal(t, []string{"configuration-a"}, installApp(t, host, device, versionA.TitleID))

		// remove label A from the host, the host should get version B's configuration
		s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", host.ID), &fleet.RemoveLabelsFromHostRequest{Labels: []string{labelA.Name}}, http.StatusOK)
		require.Equal(t, []string{"configuration-b"}, reconcileAndReceive(t, device))
	})

	t.Run("applying a label spec without the host sends the configuration of the host's next version", func(t *testing.T) {
		team := newFleet(t)
		labelA := newManualLabel(t, "a")
		versionA := addVersionAInLabelAndVersionB(t, team.ID, labelA.Name)
		host, device := newHost(t, &team.ID, labelA)
		otherHost, _ := newHost(t, &team.ID)
		require.Equal(t, []string{"configuration-a"}, installApp(t, host, device, versionA.TitleID))

		// apply label A's spec with only another host, the host should get version B's configuration
		s.DoJSON("POST", "/api/latest/fleet/spec/labels", fleet.ApplyLabelSpecsRequest{Specs: []*fleet.LabelSpec{{
			Name: labelA.Name, LabelMembershipType: fleet.LabelMembershipTypeManual, Hosts: []string{otherHost.HardwareSerial},
		}}}, http.StatusOK, &fleet.ApplyLabelSpecsResponse{})
		require.Equal(t, []string{"configuration-b"}, reconcileAndReceive(t, device))
	})

	t.Run("applying a label spec that changes the label's type and platform sends the configuration of the host's next version", func(t *testing.T) {
		team := newFleet(t)
		labelA := newManualLabel(t, "a")
		versionA := addVersionAInLabelAndVersionB(t, team.ID, labelA.Name)
		host, device := newHost(t, &team.ID, labelA)
		require.Equal(t, []string{"configuration-a"}, installApp(t, host, device, versionA.TitleID))

		// apply label A's spec as a dynamic label for another platform, the label's hosts are cleared and the host should get version B's configuration
		s.DoJSON("POST", "/api/latest/fleet/spec/labels", fleet.ApplyLabelSpecsRequest{Specs: []*fleet.LabelSpec{{
			Name: labelA.Name, LabelMembershipType: fleet.LabelMembershipTypeDynamic, Query: "SELECT 1", Platform: "darwin",
		}}}, http.StatusOK, &fleet.ApplyLabelSpecsResponse{})
		require.Equal(t, []string{"configuration-b"}, reconcileAndReceive(t, device))
	})

	t.Run("changing a host's IdP group sends the configuration of the host's next version after the host vitals label cron", func(t *testing.T) {
		team := newFleet(t)
		scimUserID, err := s.ds.CreateScimUser(ctx, &fleet.ScimUser{UserName: t.Name() + "@example.com"})
		require.NoError(t, err)
		groupID, err := s.ds.CreateScimGroup(ctx, &fleet.ScimGroup{DisplayName: t.Name() + " group", ScimUsers: []uint{scimUserID}})
		require.NoError(t, err)
		hostVitalsLabel := newHostVitalsLabel(t, &fleet.HostVitalCriteria{Vital: new("end_user_idp_group"), Value: new(t.Name() + " group")})
		versionA := addVersionAInLabelAndVersionB(t, team.ID, hostVitalsLabel.Name)
		host, device := newHost(t, &team.ID)
		_, err = s.ds.SetOrUpdateHostSCIMUserMapping(ctx, host.ID, scimUserID)
		require.NoError(t, err)
		runHostVitalsLabelCron(t, hostVitalsLabel)
		require.Equal(t, []string{"configuration-a"}, installApp(t, host, device, versionA.TitleID))

		// remove the host's IdP user from the group and run the host vitals label cron, the host should get version B's configuration
		err = s.ds.ReplaceScimGroup(ctx, &fleet.ScimGroup{ID: groupID, DisplayName: t.Name() + " group", ScimUsers: []uint{}})
		require.NoError(t, err)
		runHostVitalsLabelCron(t, hostVitalsLabel)
		require.Equal(t, []string{"configuration-b"}, reconcileAndReceive(t, device))
	})

	t.Run("changing a host's IdP department sends the configuration of the host's next version after the host vitals label cron", func(t *testing.T) {
		team := newFleet(t)
		scimUserID, err := s.ds.CreateScimUser(ctx, &fleet.ScimUser{UserName: t.Name() + "@example.com", Department: new("department-a")})
		require.NoError(t, err)
		hostVitalsLabel := newHostVitalsLabel(t, &fleet.HostVitalCriteria{Vital: new("end_user_idp_department"), Value: new("department-a")})
		versionA := addVersionAInLabelAndVersionB(t, team.ID, hostVitalsLabel.Name)
		host, device := newHost(t, &team.ID)
		_, err = s.ds.SetOrUpdateHostSCIMUserMapping(ctx, host.ID, scimUserID)
		require.NoError(t, err)
		runHostVitalsLabelCron(t, hostVitalsLabel)
		require.Equal(t, []string{"configuration-a"}, installApp(t, host, device, versionA.TitleID))

		// change the host's IdP department and run the host vitals label cron, the host should get version B's configuration
		_, err = s.ds.ReplaceScimUser(ctx, &fleet.ScimUser{ID: scimUserID, UserName: t.Name() + "@example.com", Department: new("department-b")})
		require.NoError(t, err)
		runHostVitalsLabelCron(t, hostVitalsLabel)
		require.Equal(t, []string{"configuration-b"}, reconcileAndReceive(t, device))
	})

	t.Run("linking a host to another IdP user sends the configuration of the host's next version after the host vitals label cron", func(t *testing.T) {
		team := newFleet(t)
		scimUserID, err := s.ds.CreateScimUser(ctx, &fleet.ScimUser{UserName: t.Name() + "-a@example.com"})
		require.NoError(t, err)
		otherScimUserID, err := s.ds.CreateScimUser(ctx, &fleet.ScimUser{UserName: t.Name() + "-b@example.com"})
		require.NoError(t, err)
		_, err = s.ds.CreateScimGroup(ctx, &fleet.ScimGroup{DisplayName: t.Name() + " group", ScimUsers: []uint{scimUserID}})
		require.NoError(t, err)
		hostVitalsLabel := newHostVitalsLabel(t, &fleet.HostVitalCriteria{Vital: new("end_user_idp_group"), Value: new(t.Name() + " group")})
		versionA := addVersionAInLabelAndVersionB(t, team.ID, hostVitalsLabel.Name)
		host, device := newHost(t, &team.ID)
		_, err = s.ds.SetOrUpdateHostSCIMUserMapping(ctx, host.ID, scimUserID)
		require.NoError(t, err)
		runHostVitalsLabelCron(t, hostVitalsLabel)
		require.Equal(t, []string{"configuration-a"}, installApp(t, host, device, versionA.TitleID))

		// link the host to an IdP user outside the group and run the host vitals label cron, the host should get version B's configuration
		_, err = s.ds.SetOrUpdateHostSCIMUserMapping(ctx, host.ID, otherScimUserID)
		require.NoError(t, err)
		runHostVitalsLabelCron(t, hostVitalsLabel)
		require.Equal(t, []string{"configuration-b"}, reconcileAndReceive(t, device))
	})

	t.Run("changing a host's custom host vital sends the configuration of the host's next version after the host vitals label cron", func(t *testing.T) {
		team := newFleet(t)
		customHostVital, err := s.ds.CreateCustomHostVital(ctx, t.Name())
		require.NoError(t, err)
		hostVitalsLabel := newHostVitalsLabel(t, &fleet.HostVitalCriteria{Vital: new("custom_host_vital"), Value: new("vital-a"), CustomHostVitalID: &customHostVital.ID})
		versionA := addVersionAInLabelAndVersionB(t, team.ID, hostVitalsLabel.Name)
		host, device := newHost(t, &team.ID)
		s.Do("PUT", fmt.Sprintf("/api/latest/fleet/hosts/%d/custom_host_vitals/%d", host.ID, customHostVital.ID), fleet.SetHostCustomHostVitalValueRequest{Value: "vital-a"}, http.StatusOK)
		runHostVitalsLabelCron(t, hostVitalsLabel)
		require.Equal(t, []string{"configuration-a"}, installApp(t, host, device, versionA.TitleID))

		// change the host's custom host vital and run the host vitals label cron, the host should get version B's configuration
		s.Do("PUT", fmt.Sprintf("/api/latest/fleet/hosts/%d/custom_host_vitals/%d", host.ID, customHostVital.ID), fleet.SetHostCustomHostVitalValueRequest{Value: "vital-b"}, http.StatusOK)
		runHostVitalsLabelCron(t, hostVitalsLabel)
		require.Equal(t, []string{"configuration-b"}, reconcileAndReceive(t, device))
	})

	t.Run("editing a host vitals label's criteria sends the configuration of the host's next version after the host vitals label cron", func(t *testing.T) {
		team := newFleet(t)
		scimUserID, err := s.ds.CreateScimUser(ctx, &fleet.ScimUser{UserName: t.Name() + "@example.com", Department: new("department-a")})
		require.NoError(t, err)
		hostVitalsLabel := newHostVitalsLabel(t, &fleet.HostVitalCriteria{Vital: new("end_user_idp_department"), Value: new("department-a")})
		versionA := addVersionAInLabelAndVersionB(t, team.ID, hostVitalsLabel.Name)
		host, device := newHost(t, &team.ID)
		_, err = s.ds.SetOrUpdateHostSCIMUserMapping(ctx, host.ID, scimUserID)
		require.NoError(t, err)
		runHostVitalsLabelCron(t, hostVitalsLabel)
		require.Equal(t, []string{"configuration-a"}, installApp(t, host, device, versionA.TitleID))

		// apply the label's spec with another department and run the host vitals label cron, the host should get version B's configuration
		criteria, err := json.Marshal(fleet.HostVitalCriteria{Vital: new("end_user_idp_department"), Value: new("department-b")})
		require.NoError(t, err)
		s.DoJSON("POST", "/api/latest/fleet/spec/labels", fleet.ApplyLabelSpecsRequest{Specs: []*fleet.LabelSpec{{
			Name: hostVitalsLabel.Name, LabelMembershipType: fleet.LabelMembershipTypeHostVitals, HostVitalsCriteria: new(json.RawMessage(criteria)),
		}}}, http.StatusOK, &fleet.ApplyLabelSpecsResponse{})
		runHostVitalsLabelCron(t, hostVitalsLabel)
		require.Equal(t, []string{"configuration-b"}, reconcileAndReceive(t, device))
	})

	t.Run("moving a host to another fleet sends the configuration of the new fleet's version", func(t *testing.T) {
		team := newFleet(t)
		otherTeam, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + " other"})
		require.NoError(t, err)
		version := addVersion(t, team.ID, "Old fleet", "configuration-old-fleet", nil)
		addVersion(t, otherTeam.ID, "New fleet", "configuration-new-fleet", nil)
		host, device := newHost(t, &team.ID)
		require.Equal(t, []string{"configuration-old-fleet"}, installApp(t, host, device, version.TitleID))

		// move the host to the other fleet, the host should get the other fleet's configuration
		s.Do("POST", "/api/latest/fleet/hosts/transfer", &addHostsToTeamRequest{HostIDs: []uint{host.ID}, TeamID: &otherTeam.ID}, http.StatusOK)
		require.Equal(t, []string{"configuration-new-fleet"}, reconcileAndReceive(t, device))
	})

	t.Run("deleting a host's fleet sends the configuration of the Unassigned version", func(t *testing.T) {
		team := newFleet(t)
		version := addVersion(t, team.ID, "Deleted fleet", "configuration-deleted-fleet", nil)
		unassignedVersion := addVersion(t, 0, "Unassigned", "configuration-unassigned", nil)
		t.Cleanup(func() {
			s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/software/titles/%d/available_for_install", unassignedVersion.TitleID), nil, http.StatusNoContent, "fleet_id", "0")
		})
		host, device := newHost(t, &team.ID)
		require.Equal(t, []string{"configuration-deleted-fleet"}, installApp(t, host, device, version.TitleID))

		// delete the host's fleet, the host should get the Unassigned version's configuration
		s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK)
		require.Equal(t, []string{"configuration-unassigned"}, reconcileAndReceive(t, device))
	})

	t.Run("re-enrolling a host through ADE sends the configuration of the host's next version", func(t *testing.T) {
		team := newFleet(t)
		labelA := newManualLabel(t, "a")
		versionA := addVersionAInLabelAndVersionB(t, team.ID, labelA.Name)
		host, device := newHost(t, &team.ID, labelA)
		require.Equal(t, []string{"configuration-a"}, installApp(t, host, device, versionA.TitleID))

		// re-enroll the host through ADE, the host's labels are cleared and the host should get version B's configuration
		require.NoError(t, device.Enroll())
		s.awaitRunAppleMDMWorkerSchedule()
		require.Equal(t, []string{"configuration-b"}, reconcileAndReceive(t, device))
	})

	t.Run("deleting a host that checks in again sends the configuration of the host's version", func(t *testing.T) {
		labelA := newManualLabel(t, "a")
		versionA := addVersionAInLabelAndVersionB(t, 0, labelA.Name)
		t.Cleanup(func() {
			s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/software/titles/%d/available_for_install", versionA.TitleID), nil, http.StatusNoContent, "fleet_id", "0")
		})
		host, device := newHost(t, nil, labelA)
		require.Equal(t, []string{"configuration-a"}, installApp(t, host, device, versionA.TitleID))

		// delete the host and let it check in again, the re-created host has no labels and should get version B's configuration
		s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK)
		err := apple_mdm.IOSiPadOSRevive(ctx, s.ds, s.mdmCommander, s.logger)
		require.NoError(t, err)
		cmd, err := device.Idle()
		require.NoError(t, err)
		require.Nil(t, cmd)
		recreatedHost, err := s.ds.HostByIdentifier(ctx, device.UUID)
		require.NoError(t, err)
		refetch(t, recreatedHost)
		configurationsReceived(t, device)
		require.Equal(t, []string{"configuration-b"}, reconcileAndReceive(t, device))
	})

	t.Run("refetching a host after a dynamic label in exclude any was created sends the configuration of the host's next version", func(t *testing.T) {
		team := newFleet(t)
		host, device := newHost(t, &team.ID)
		time.Sleep(time.Second)
		dynamicLabel, err := s.ds.NewLabel(ctx, &fleet.Label{Name: t.Name() + " dynamic", Query: "SELECT 1", LabelMembershipType: fleet.LabelMembershipTypeDynamic})
		require.NoError(t, err)
		var versionA addAppStoreAppResponse
		s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{
			TeamID: &team.ID, AppStoreID: adamID, Platform: fleet.IOSPlatform, Name: "A",
			LabelsExcludeAny: []string{dynamicLabel.Name}, Configuration: configurationJSON("configuration-a"),
		}, http.StatusOK, &versionA)
		addVersion(t, team.ID, "B", "configuration-b", nil)
		require.Equal(t, []string{"configuration-b"}, installApp(t, host, device, versionA.TitleID))

		// refetch the host, the dynamic label's result is known after the refetch and the host should get version A's configuration
		refetch(t, host)
		configurationsReceived(t, device)
		require.Equal(t, []string{"configuration-a"}, reconcileAndReceive(t, device))
	})

	t.Run("adding a version that a host is in scope for sends the new version's configuration", func(t *testing.T) {
		team := newFleet(t)
		labelA := newManualLabel(t, "a")
		versionA := addVersion(t, team.ID, "A", "configuration-a", []string{labelA.Name})
		host, device := newHost(t, &team.ID, labelA)
		require.Equal(t, []string{"configuration-a"}, installApp(t, host, device, versionA.TitleID))

		// remove label A from the host, the host is in scope for no version and should get nothing
		require.NoError(t, s.ds.RemoveLabelsFromHost(ctx, host.ID, []uint{labelA.ID}))
		require.Empty(t, reconcileAndReceive(t, device))

		// add version B without labels, the host should get version B's configuration
		addVersion(t, team.ID, "B", "configuration-b", nil)
		require.Equal(t, []string{"configuration-b"}, reconcileAndReceive(t, device))
	})

	t.Run("deleting every version sends nothing to the host", func(t *testing.T) {
		team := newFleet(t)
		version := addVersion(t, team.ID, "Only", "configuration-deleted", nil)
		host, device := newHost(t, &team.ID)
		require.Equal(t, []string{"configuration-deleted"}, installApp(t, host, device, version.TitleID))

		// delete every version, the host should get nothing
		s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/software/titles/%d/available_for_install", version.TitleID), nil, http.StatusNoContent,
			"fleet_id", fmt.Sprint(team.ID))
		require.Empty(t, reconcileAndReceive(t, device))
	})

	t.Run("removing the app from GitOps sends nothing to the host", func(t *testing.T) {
		team := newFleet(t)
		version := addVersion(t, team.ID, "Only", "configuration-removed", nil)
		host, device := newHost(t, &team.ID)
		require.Equal(t, []string{"configuration-removed"}, installApp(t, host, device, version.TitleID))

		// apply the fleet's App Store apps without the app, the host should get nothing
		s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps/batch", batchAssociateAppStoreAppsRequest{Apps: []fleet.VPPBatchPayload{}},
			http.StatusOK, &batchAssociateAppStoreAppsResponse{}, "fleet_name", team.Name)
		require.Empty(t, reconcileAndReceive(t, device))
	})

	t.Run("installing a host through the setup experience sends the first flagged version's configuration, then the configuration of the host's version", func(t *testing.T) {
		team := newFleet(t)
		labelA := newManualLabel(t, "a")
		versionA := addVersionAInLabelAndVersionB(t, team.ID, labelA.Name)
		s.DoJSON("PUT", "/api/v1/fleet/setup_experience/software", putSetupExperienceSoftwareRequest{
			Platform: string(fleet.IOSPlatform), TeamID: team.ID, TitleIDs: []uint{versionA.TitleID},
		}, http.StatusOK, &putSetupExperienceSoftwareResponse{})
		_, device := newHost(t, &team.ID)

		// enroll the host through ADE in the fleet, the setup experience ignores labels and the host should get version A's configuration
		require.NoError(t, device.Enroll())
		s.awaitRunAppleMDMWorkerSchedule()
		require.Equal(t, []string{"configuration-a"}, reconcileAndReceive(t, device))

		// reconcile again, the host isn't in label A and should get version B's configuration
		require.Equal(t, []string{"configuration-b"}, reconcileAndReceive(t, device))
	})

	t.Run("deleting a version with an install queued for it sends the configuration of the host's next version", func(t *testing.T) {
		team := newFleet(t)
		labelA := newManualLabel(t, "a")
		labelB := newManualLabel(t, "b")
		addVersion(t, team.ID, "A", "configuration-a", []string{labelA.Name})
		versionB := addVersion(t, team.ID, "B", "configuration-b", []string{labelB.Name})
		addVersion(t, team.ID, "C", "configuration-c", []string{labelB.Name})
		host, device := newHost(t, &team.ID, labelB)
		queueInstallBehindOtherApp(t, team.ID, host, versionB.TitleID)

		// delete version B before its install runs, the host should get version C's configuration
		s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/software/titles/%d/available_for_install", versionB.TitleID), nil, http.StatusNoContent,
			"fleet_id", fmt.Sprint(team.ID), "version_id", fmt.Sprint(versionB.VersionID))
		require.Equal(t, []string{"configuration-c"}, reconcileAndReceive(t, device))
	})

	t.Run("moving a host with a queued install to another fleet sends the configuration of the new fleet's version", func(t *testing.T) {
		team := newFleet(t)
		otherTeam, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + " other"})
		require.NoError(t, err)
		labelA := newManualLabel(t, "a")
		version := addVersion(t, team.ID, "Old fleet", "configuration-old-fleet", nil)
		addVersionAInLabelAndVersionB(t, otherTeam.ID, labelA.Name)
		host, device := newHost(t, &team.ID)
		queueInstallBehindOtherApp(t, team.ID, host, version.TitleID)

		// move the host to the other fleet before its install runs, the host isn't in label A and should get version B's configuration
		s.Do("POST", "/api/latest/fleet/hosts/transfer", &addHostsToTeamRequest{HostIDs: []uint{host.ID}, TeamID: &otherTeam.ID}, http.StatusOK)
		require.Equal(t, []string{"configuration-b"}, reconcileAndReceive(t, device))
	})

	t.Run("changing a host's IdP group sends the configuration with the new group", func(t *testing.T) {
		team := newFleet(t)
		scimUserID, err := s.ds.CreateScimUser(ctx, &fleet.ScimUser{UserName: t.Name() + "@example.com"})
		require.NoError(t, err)
		groupID, err := s.ds.CreateScimGroup(ctx, &fleet.ScimGroup{DisplayName: "group-before", ScimUsers: []uint{scimUserID}})
		require.NoError(t, err)
		version := addVersionWithRawConfiguration(t, team.ID, `<dict><key>value</key><string>$FLEET_VAR_HOST_END_USER_IDP_GROUPS</string></dict>`)
		host, device := newHost(t, &team.ID)
		_, err = s.ds.SetOrUpdateHostSCIMUserMapping(ctx, host.ID, scimUserID)
		require.NoError(t, err)
		require.Equal(t, []string{"group-before"}, installApp(t, host, device, version.TitleID))

		// rename the host's IdP group, the host should get the configuration with the new group
		err = s.ds.ReplaceScimGroup(ctx, &fleet.ScimGroup{ID: groupID, DisplayName: "group-after", ScimUsers: []uint{scimUserID}})
		require.NoError(t, err)
		require.Equal(t, []string{"group-after"}, reconcileAndReceive(t, device))
	})

	t.Run("linking a host to another IdP user sends the configuration with the new username", func(t *testing.T) {
		team := newFleet(t)
		scimUserID, err := s.ds.CreateScimUser(ctx, &fleet.ScimUser{UserName: "username-before"})
		require.NoError(t, err)
		otherScimUserID, err := s.ds.CreateScimUser(ctx, &fleet.ScimUser{UserName: "username-after"})
		require.NoError(t, err)
		version := addVersionWithRawConfiguration(t, team.ID, `<dict><key>value</key><string>$FLEET_VAR_HOST_END_USER_IDP_USERNAME</string></dict>`)
		host, device := newHost(t, &team.ID)
		_, err = s.ds.SetOrUpdateHostSCIMUserMapping(ctx, host.ID, scimUserID)
		require.NoError(t, err)
		require.Equal(t, []string{"username-before"}, installApp(t, host, device, version.TitleID))

		// link the host to another IdP user, the host should get the configuration with the new username
		_, err = s.ds.SetOrUpdateHostSCIMUserMapping(ctx, host.ID, otherScimUserID)
		require.NoError(t, err)
		require.Equal(t, []string{"username-after"}, reconcileAndReceive(t, device))
	})

	t.Run("changing a host's IdP email on enrollment sends the configuration with the new email", func(t *testing.T) {
		team := newFleet(t)
		version := addVersionWithRawConfiguration(t, team.ID, `<dict><key>value</key><string>$FLEET_VAR_HOST_END_USER_EMAIL_IDP</string></dict>`)
		host, device := newHost(t, &team.ID)
		accountUUIDBefore := uuid.NewString()
		err := s.ds.InsertMDMIdPAccount(ctx, &fleet.MDMIdPAccount{UUID: accountUUIDBefore, Username: "before", Email: "email-before@example.com"})
		require.NoError(t, err)
		_, err = s.ds.AssociateHostMDMIdPAccount(ctx, host.UUID, accountUUIDBefore)
		require.NoError(t, err)
		require.Equal(t, []string{"email-before@example.com"}, installApp(t, host, device, version.TitleID))

		// link the host to another IdP account, the host should get the configuration with the new email
		accountUUIDAfter := uuid.NewString()
		err = s.ds.InsertMDMIdPAccount(ctx, &fleet.MDMIdPAccount{UUID: accountUUIDAfter, Username: "after", Email: "email-after@example.com"})
		require.NoError(t, err)
		_, err = s.ds.AssociateHostMDMIdPAccount(ctx, host.UUID, accountUUIDAfter)
		require.NoError(t, err)
		require.Equal(t, []string{"email-after@example.com"}, reconcileAndReceive(t, device))
	})

	t.Run("changing the IdP username of a host without an IdP user sends the configuration with the new username", func(t *testing.T) {
		team := newFleet(t)
		version := addVersionWithRawConfiguration(t, team.ID, `<dict><key>value</key><string>$FLEET_VAR_HOST_END_USER_IDP_USERNAME</string></dict>`)
		host, device := newHost(t, &team.ID)
		s.DoJSON("PUT", fmt.Sprintf("/api/latest/fleet/hosts/%d/device_mapping", host.ID),
			putHostDeviceMappingRequest{Email: "mapped-before@example.com", Source: fleet.DeviceMappingIDP}, http.StatusOK, &putHostDeviceMappingResponse{})
		require.Equal(t, []string{"mapped-before@example.com"}, installApp(t, host, device, version.TitleID))

		// set another IdP username through device mapping, the host should get the configuration with the new username
		s.DoJSON("PUT", fmt.Sprintf("/api/latest/fleet/hosts/%d/device_mapping", host.ID),
			putHostDeviceMappingRequest{Email: "mapped-after@example.com", Source: fleet.DeviceMappingIDP}, http.StatusOK, &putHostDeviceMappingResponse{})
		require.Equal(t, []string{"mapped-after@example.com"}, reconcileAndReceive(t, device))
	})

	t.Run("changing a host's serial number sends the configuration with the new serial number", func(t *testing.T) {
		team := newFleet(t)
		version := addVersionWithRawConfiguration(t, team.ID, `<dict><key>value</key><string>$FLEET_VAR_HOST_HARDWARE_SERIAL</string></dict>`)
		host, device := newHost(t, &team.ID)
		require.Equal(t, []string{host.HardwareSerial}, installApp(t, host, device, version.TitleID))

		// change the host's serial number, the host should get the configuration with the new serial number
		newSerial := mdmtest.RandSerialNumber()
		s.appleVPPConfigSrvConfig.SerialNumbers = append(s.appleVPPConfigSrvConfig.SerialNumbers, newSerial)
		host, err := s.ds.Host(ctx, host.ID)
		require.NoError(t, err)
		host.HardwareSerial = newSerial
		require.NoError(t, s.ds.UpdateHost(ctx, host))
		require.Equal(t, []string{newSerial}, reconcileAndReceive(t, device))
	})

	t.Run("reinstalling an app the user removed sends later configuration edits to the host", func(t *testing.T) {
		team := newFleet(t)
		version := addVersion(t, team.ID, "Only", "configuration-before", nil)
		host, device := newHost(t, &team.ID)
		require.Equal(t, []string{"configuration-before"}, installApp(t, host, device, version.TitleID))

		// remove the app from the host's inventory, then add it back as if the user reinstalled it
		_, err := s.ds.UpdateHostSoftware(ctx, host.ID, []fleet.Software{})
		require.NoError(t, err)
		_, err = s.ds.UpdateHostSoftware(ctx, host.ID, []fleet.Software{appInInventory})
		require.NoError(t, err)

		// edit the version's configuration, the host should get the new configuration
		editConfiguration(t, team.ID, version, configurationJSON("configuration-after"))
		require.Equal(t, []string{"configuration-after"}, reconcileAndReceive(t, device))
	})

	t.Run("resolving a variable that failed a configuration send sends the configuration to the host", func(t *testing.T) {
		team := newFleet(t)
		version := addVersion(t, team.ID, "Only", "configuration-before", nil)
		host, device := newHost(t, &team.ID)
		require.Equal(t, []string{"configuration-before"}, installApp(t, host, device, version.TitleID))

		// edit the configuration to use the IdP email the host doesn't have, the host should get nothing
		b, err := json.Marshal(`<dict><key>value</key><string>$FLEET_VAR_HOST_END_USER_EMAIL_IDP</string></dict>`)
		require.NoError(t, err)
		editConfiguration(t, team.ID, version, b)
		require.Empty(t, reconcileAndReceive(t, device))

		// reconcile again, the failed install should not be retried within the hour
		require.Empty(t, reconcileAndReceive(t, device))
		var failedInstalls int
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &failedInstalls, `SELECT COUNT(*) FROM host_vpp_software_installs WHERE host_id = ? AND verification_failed_at IS NOT NULL`, host.ID)
		})
		require.Equal(t, 1, failedInstalls)

		// link the host to an IdP account, the host should get nothing until the failed install is an hour old
		accountUUID := uuid.NewString()
		err = s.ds.InsertMDMIdPAccount(ctx, &fleet.MDMIdPAccount{UUID: accountUUID, Username: "resolved", Email: "resolved@example.com"})
		require.NoError(t, err)
		_, err = s.ds.AssociateHostMDMIdPAccount(ctx, host.UUID, accountUUID)
		require.NoError(t, err)
		require.Empty(t, reconcileAndReceive(t, device))

		// move the failed install back two hours, the host should get the configuration with the email
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx, `UPDATE host_vpp_software_installs SET created_at = created_at - INTERVAL 2 HOUR WHERE host_id = ? AND verification_failed_at IS NOT NULL`, host.ID)
			return err
		})
		require.Equal(t, []string{"resolved@example.com"}, reconcileAndReceive(t, device))
	})

	t.Run("adding back an app removed from Apple Business sends the configuration skipped while it was removed", func(t *testing.T) {
		team := newFleet(t)
		version := addVersion(t, team.ID, "Only", "configuration-before", nil)
		host, device := newHost(t, &team.ID)
		require.Equal(t, []string{"configuration-before"}, installApp(t, host, device, version.TitleID))

		// remove the app from Apple Business and edit the version's configuration, the host should get nothing
		var assetsWithoutApp []vpp.Asset
		for _, asset := range defaultVPPAssetList {
			if asset.AdamID != adamID {
				assetsWithoutApp = append(assetsWithoutApp, asset)
			}
		}
		s.appleVPPConfigSrvConfig.Assets = assetsWithoutApp
		t.Cleanup(func() { s.appleVPPConfigSrvConfig.Assets = defaultVPPAssetList })
		editConfiguration(t, team.ID, version, configurationJSON("configuration-after"))
		require.Empty(t, reconcileAndReceive(t, device))

		// add the app back to Apple Business, the host should get the new configuration
		s.appleVPPConfigSrvConfig.Assets = defaultVPPAssetList
		require.Equal(t, []string{"configuration-after"}, reconcileAndReceive(t, device))
	})

	t.Run("turning MDM back on for a host sends the configuration skipped while MDM was off", func(t *testing.T) {
		team := newFleet(t)
		version := addVersion(t, team.ID, "Only", "configuration-before", nil)
		host, device := newHost(t, &team.ID)
		require.Equal(t, []string{"configuration-before"}, installApp(t, host, device, version.TitleID))

		// turn MDM off for the host and edit the version's configuration, nothing can be sent to the host
		s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/mdm", host.ID), nil, http.StatusNoContent)
		require.NoError(t, device.Checkout())
		editConfiguration(t, team.ID, version, configurationJSON("configuration-after"))
		reconcile()

		// enroll the host again, the host should get the new configuration
		require.NoError(t, device.Enroll())
		s.awaitRunAppleMDMWorkerSchedule()
		require.Equal(t, []string{"configuration-after"}, reconcileAndReceive(t, device))
	})

	t.Run("removing the VPP token from a host's fleet sends nothing to the host", func(t *testing.T) {
		team := newFleet(t)
		otherTeam, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + " other"})
		require.NoError(t, err)
		version := addVersion(t, team.ID, "Only", "configuration-before", nil)
		host, device := newHost(t, &team.ID)
		require.Equal(t, []string{"configuration-before"}, installApp(t, host, device, version.TitleID))
		t.Cleanup(func() {
			s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/vpp_tokens/%d/teams", vppTokenID),
				patchVPPTokensTeamsRequest{TeamIDs: []uint{}}, http.StatusOK, &patchVPPTokensTeamsResponse{})
		})

		// assign the VPP token to another fleet only, the host should get nothing
		s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/vpp_tokens/%d/teams", vppTokenID),
			patchVPPTokensTeamsRequest{TeamIDs: []uint{otherTeam.ID}}, http.StatusOK, &patchVPPTokensTeamsResponse{})
		require.Empty(t, reconcileAndReceive(t, device))
	})

	t.Run("deleting the VPP token sends nothing to the host", func(t *testing.T) {
		team := newFleet(t)
		version := addVersion(t, team.ID, "Only", "configuration-before", nil)
		host, device := newHost(t, &team.ID)
		require.Equal(t, []string{"configuration-before"}, installApp(t, host, device, version.TitleID))

		// delete the VPP token, the host should get nothing
		s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/vpp_tokens/%d", vppTokenID), &deleteVPPTokenRequest{}, http.StatusNoContent)
		require.Empty(t, reconcileAndReceive(t, device))
	})
}

func getFirstAddedVPPAppConfigurationForTest(t *testing.T, ds *mysql.Datastore, platform fleet.InstallableDevicePlatform, adamID string, teamID uint) ([]byte, error) {
	var vppAppTeamID uint
	mysqltest.ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(t.Context(), q, &vppAppTeamID,
			`SELECT COALESCE(MIN(id), 0) FROM vpp_apps_teams WHERE adam_id = ? AND platform = ? AND global_or_team_id = ?`,
			adamID, platform, teamID)
	})
	return ds.GetVPPAppConfiguration(ctxdb.RequirePrimary(t.Context(), true), vppAppTeamID)
}

func (s *integrationMDMTestSuite) TestAppStoreAppsRequireAppleMDM() {
	t := s.T()
	ctx := t.Context()

	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "vpp-requires-apple-mdm"})
	require.NoError(t, err)

	orgName := "Fleet Device Management Inc."
	token := "vpprequiresapplemdmtoken" //nolint:gosec // G101: test value, not a real credential
	expDate := time.Now().Add(200 * time.Hour).UTC().Round(time.Second).Format(fleet.VPPTimeFormat)
	tokenJSON := fmt.Sprintf(`{"expDate":%q,"token":%q,"orgName":%q}`, expDate, token, orgName)
	dev_mode.SetOverride("FLEET_DEV_VPP_URL", s.appleVPPConfigSrv.URL, t)

	var uploadResp uploadVPPTokenResponse
	s.uploadDataViaForm("/api/latest/fleet/vpp_tokens", "token", "token.vpptoken",
		[]byte(base64.StdEncoding.EncodeToString([]byte(tokenJSON))), http.StatusAccepted, "", &uploadResp)
	var patchResp patchVPPTokensTeamsResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/vpp_tokens/%d/teams", uploadResp.Token.ID),
		patchVPPTokensTeamsRequest{TeamIDs: []uint{team.ID}}, http.StatusOK, &patchResp)

	var appsResp getAppStoreAppsResponse
	s.DoJSON("GET", "/api/latest/fleet/software/app_store_apps", nil, http.StatusOK, &appsResp, "fleet_id", fmt.Sprint(team.ID))
	require.NotEmpty(t, appsResp.AppStoreApps)

	appCfg, err := s.ds.AppConfig(ctx)
	require.NoError(t, err)
	appCfg.MDM.EnabledAndConfigured = false
	require.NoError(t, s.ds.SaveAppConfig(ctx, appCfg))
	t.Cleanup(func() {
		appCfg.MDM.EnabledAndConfigured = true
		require.NoError(t, s.ds.SaveAppConfig(context.Background(), appCfg))
	})

	res := s.Do("GET", "/api/latest/fleet/software/app_store_apps", nil, http.StatusBadRequest, "fleet_id", fmt.Sprint(team.ID))
	require.Contains(t, extractServerErrorText(res.Body), fleet.MDMNotConfiguredMessage)

	res = s.Do("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{
		TeamID:     &team.ID,
		AppStoreID: "1",
		Platform:   fleet.MacOSPlatform,
	}, http.StatusBadRequest)
	require.Contains(t, extractServerErrorText(res.Body), fleet.MDMNotConfiguredMessage)

	res = s.Do("POST", "/api/latest/fleet/software/app_store_apps/batch", batchAssociateAppStoreAppsRequest{
		Apps: []fleet.VPPBatchPayload{{AppStoreID: "1", Platform: fleet.MacOSPlatform}},
	}, http.StatusBadRequest, "team_name", team.Name)
	require.Contains(t, extractServerErrorText(res.Body), fleet.MDMNotConfiguredMessage)
}
