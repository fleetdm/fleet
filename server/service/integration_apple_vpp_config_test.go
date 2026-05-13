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

	"github.com/fleetdm/fleet/v4/server/contexts/ctxdb"
	"github.com/fleetdm/fleet/v4/server/dev_mode"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android/service/androidmgmt"
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

// TestManagedAppConfigurationWireFormat builds request bodies by hand and
// inspects raw HTTP response bytes — no Go struct marshalling — to lock in the
// on-the-wire shape of VPPAppStoreApp.Configuration after the switch from
// []byte (base64-encoded by encoding/json) to json.RawMessage (passed through
// as raw JSON). iOS / iPadOS configurations are sent and returned as a
// JSON-encoded string of XML; Android configurations are sent and returned as
// a raw JSON object.
func (s *integrationMDMTestSuite) TestManagedAppConfigurationWireFormat() {
	t := s.T()
	s.setSkipWorkerJobs(t)
	ctx := context.Background()

	// VPP setup.
	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "vpp-android-wire-team"})
	require.NoError(t, err)

	orgName := "Fleet Device Management Inc."
	token := "applewiretoken"
	expDate := time.Now().Add(200 * time.Hour).UTC().Round(time.Second).Format(fleet.VPPTimeFormat)
	tokenJSON := fmt.Sprintf(`{"expDate":%q,"token":%q,"orgName":%q}`, expDate, token, orgName)
	dev_mode.SetOverride("FLEET_DEV_VPP_URL", s.appleVPPConfigSrv.URL, t)

	var validToken uploadVPPTokenResponse
	s.uploadDataViaForm("/api/latest/fleet/vpp_tokens", "token", "token.vpptoken",
		[]byte(base64.StdEncoding.EncodeToString([]byte(tokenJSON))), http.StatusAccepted, "", &validToken)

	var getVPPTokenResp getVPPTokensResponse
	s.DoJSON("GET", "/api/latest/fleet/vpp_tokens", &getVPPTokensRequest{}, http.StatusOK, &getVPPTokenResp)

	var resPatchVPP patchVPPTokensTeamsResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/vpp_tokens/%d/teams", getVPPTokenResp.Tokens[0].ID),
		patchVPPTokensTeamsRequest{TeamIDs: []uint{team.ID}}, http.StatusOK, &resPatchVPP)

	// readBody reads the response body and compacts it (collapses the
	// pretty-printing the server applies) so assertions can be written against
	// the unindented wire bytes.
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
		// Hand-built JSON body. The configuration value is a JSON-encoded
		// string whose content is XML.
		reqBody := fmt.Appendf(nil,
			`{"fleet_id":%d,"app_store_id":%q,"platform":"ios","configuration":"<dict><key>K</key><string>v</string></dict>"}`,
			team.ID, iosAdamID)
		resp := s.DoRaw("POST", "/api/latest/fleet/software/app_store_apps", reqBody, http.StatusOK)
		var addResp addAppStoreAppResponse
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&addResp))
		require.NoError(t, resp.Body.Close())
		require.NotZero(t, addResp.TitleID)

		// Stored config is the raw XML (the JSON-string was unwrapped server-side).
		stored, err := s.ds.GetVPPAppConfiguration(ctxdb.RequirePrimary(ctx, true), fleet.IOSPlatform, iosAdamID, team.ID)
		require.NoError(t, err)
		require.Equal(t, `<dict><key>K</key><string>v</string></dict>`, string(stored))

		// GET response — wire shape is a JSON-encoded string of XML (no base64).
		resp = s.DoRaw("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", addResp.TitleID), nil, http.StatusOK, "fleet_id", fmt.Sprint(team.ID))
		body := readBody(resp)
		require.Contains(t, body, `"configuration":"<dict><key>K</key><string>v</string></dict>"`)
		// Confirm the value is not base64-encoded (which is what a []byte field would do).
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

		// Configuration is sent as a raw JSON object — not a JSON-encoded string.
		reqBody := fmt.Appendf(nil,
			`{"fleet_id":%d,"app_store_id":%q,"platform":"android","configuration":{"workProfileWidgets":"WORK_PROFILE_WIDGETS_ALLOWED"}}`,
			team.ID, androidAdamID)
		resp := s.DoRaw("POST", "/api/latest/fleet/software/app_store_apps", reqBody, http.StatusOK)
		var addResp addAppStoreAppResponse
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&addResp))
		require.NoError(t, resp.Body.Close())
		require.NotZero(t, addResp.TitleID)

		// Stored config is the raw JSON object.
		stored, err := s.ds.GetAndroidAppConfiguration(ctxdb.RequirePrimary(ctx, true), androidAdamID, team.ID)
		require.NoError(t, err)
		require.JSONEq(t, `{"workProfileWidgets":"WORK_PROFILE_WIDGETS_ALLOWED"}`, string(stored))

		// GET response — Android emits the configuration as a raw JSON object,
		// passed through unchanged from storage (no base64, no JSON-string wrap).
		resp = s.DoRaw("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", addResp.TitleID), nil, http.StatusOK, "fleet_id", fmt.Sprint(team.ID))
		body := readBody(resp)
		require.Contains(t, body, `"configuration":{"workProfileWidgets":"WORK_PROFILE_WIDGETS_ALLOWED"}`)
		require.NotContains(t, body, base64.StdEncoding.EncodeToString([]byte(`{"workProfileWidgets":"WORK_PROFILE_WIDGETS_ALLOWED"}`)))
	})
}
