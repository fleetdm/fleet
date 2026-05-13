package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/mdm/mdmtest"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxdb"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/dev_mode"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android/service/androidmgmt"
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
		got, err := s.ds.GetVPPAppConfiguration(ctxdb.RequirePrimary(ctx, true), platform, adamID, team.ID)
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
	_, err = s.ds.GetVPPAppConfiguration(ctxdb.RequirePrimary(ctx, true), fleet.IOSPlatform, iosAdamID, team.ID)
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
		_, err := s.ds.GetVPPAppConfiguration(ctxdb.RequirePrimary(ctx, true), platform, adamID, teamID)
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
		iosCfg, err := s.ds.GetVPPAppConfiguration(ctxdb.RequirePrimary(ctx, true), fleet.IOSPlatform, iosAdamID, batchTeam.ID)
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

		stored, err := s.ds.GetVPPAppConfiguration(ctxdb.RequirePrimary(ctx, true), fleet.IOSPlatform, iosAdamID, team.ID)
		require.NoError(t, err)
		require.Equal(t, `<dict><key>K</key><string>v</string></dict>`, string(stored))

		resp = s.DoRaw("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", addResp.TitleID), nil, http.StatusOK, "fleet_id", fmt.Sprint(team.ID))
		body := readBody(resp)
		require.Contains(t, body, `"configuration":"<dict><key>K</key><string>v</string></dict>"`)
		require.NotContains(t, body, base64.StdEncoding.EncodeToString([]byte(`"<dict><key>K</key><string>v</string></dict>"`)))
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
		require.NotContains(t, body, base64.StdEncoding.EncodeToString([]byte(`{"workProfileWidgets":"WORK_PROFILE_WIDGETS_ALLOWED"}`)))
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

	asJSONString := func(s string) json.RawMessage {
		b, err := json.Marshal(s)
		require.NoError(t, err)
		return json.RawMessage(b)
	}

	titleIDFor := func(adamID string, platform fleet.InstallableDevicePlatform) uint {
		var id uint
		mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &id,
				`SELECT title_id FROM vpp_apps WHERE adam_id = ? AND platform = ?`, adamID, platform)
		})
		require.NotZero(t, id, "title_id for adam=%s platform=%s", adamID, platform)
		return id
	}

	// drainVerifyCmds acks any pending InstalledApplicationList commands as "installed".
	drainVerifyCmds := func(t *testing.T, dev *mdmtest.TestAppleMDMClient, installed fleet.Software) {
		t.Helper()
		installed.Installed = true
		for {
			cmd, err := dev.Idle()
			require.NoError(t, err)
			if cmd == nil {
				return
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
		drainVerifyCmds(t, dev, installed)

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

		_, err := s.ds.GetVPPAppConfiguration(ctxdb.RequirePrimary(ctx, true), fleet.IOSPlatform, adamMulti, team.ID)
		require.True(t, fleet.IsNotFound(err), "expected config row deleted")

		raw := string(installAndCaptureCmd(t, iosHost, iosDev, titleID, app2Installed))
		require.NotContains(t, raw, "<key>Configuration</key>",
			"InstallApplication should not include Configuration after clearing the stored config")
	})

	t.Run("second iOS host gets its own UUID substituted (per-host resolution)", func(t *testing.T) {
		titleID := titleIDFor(adamMulti, fleet.IOSPlatform)
		s.DoJSON("PATCH",
			fmt.Sprintf("/api/latest/fleet/software/titles/%d/app_store_app", titleID),
			&updateAppStoreAppRequest{TeamID: &team.ID,
				Configuration: asJSONString(`<dict><key>UUID</key><string>$FLEET_VAR_HOST_UUID</string></dict>`)},
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

		storedIOS, err := s.ds.GetVPPAppConfiguration(ctxdb.RequirePrimary(ctx, true),
			fleet.IOSPlatform, adamMulti, team.ID)
		require.NoError(t, err)
		require.Equal(t, iosCfg, string(storedIOS))
		storedIPad, err := s.ds.GetVPPAppConfiguration(ctxdb.RequirePrimary(ctx, true),
			fleet.IPadOSPlatform, adamMulti, team.ID)
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
	})

	t.Run("self-service install carries Configuration with $FLEET_VAR_HOST_UUID resolved", func(t *testing.T) {
		ssHost, ssDev := s.createAppleMobileHostThenEnrollMDM("ios")
		s.appleVPPConfigSrvConfig.SerialNumbers = append(s.appleVPPConfigSrvConfig.SerialNumbers, ssDev.SerialNumber)
		s.Do("POST", "/api/latest/fleet/hosts/transfer",
			&addHostsToTeamRequest{HostIDs: []uint{ssHost.ID}, TeamID: &team.ID}, http.StatusOK)

		// Self-service device endpoint requires cert auth.
		const certSerial = uint64(987654321)
		s.addHostIdentityCertificate(ssHost.UUID, certSerial)
		headers := map[string]string{
			"X-Client-Cert-Serial": fmt.Sprintf("%d", certSerial),
		}

		drainVerifyCmds(t, ssDev, app2Installed)

		titleID := titleIDFor(adamMulti, fleet.IOSPlatform)
		s.DoJSON("PATCH",
			fmt.Sprintf("/api/latest/fleet/software/titles/%d/app_store_app", titleID),
			&updateAppStoreAppRequest{TeamID: &team.ID, SelfService: new(true),
				Configuration: asJSONString(`<dict><key>UUID</key><string>$FLEET_VAR_HOST_UUID</string></dict>`)},
			http.StatusOK, &updateAppStoreAppResponse{})

		s.DoRawWithHeaders("POST",
			fmt.Sprintf("/api/latest/fleet/device/%s/software/install/%d", ssHost.UUID, titleID),
			nil, http.StatusAccepted, headers)

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
}
