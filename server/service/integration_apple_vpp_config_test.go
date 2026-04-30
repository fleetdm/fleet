package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxdb"
	"github.com/fleetdm/fleet/v4/server/dev_mode"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
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

	// Adam IDs "2" and "3" come pre-registered by the mock VPP server with
	// iOS/iPadOS metadata; no asset list mutation needed.
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

	// Helper: encode a plist XML string as a JSON string (the wire format clients use).
	asJSONString := func(s string) json.RawMessage {
		b, err := json.Marshal(s)
		require.NoError(t, err)
		return json.RawMessage(b)
	}

	// Helper: read the stored configuration directly from the datastore so we
	// don't depend on response wire-format details.
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

	// 3. Update iOS app omitting `configuration` field → no change.
	updResp = updateAppStoreAppResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/app_store_app", addResp.TitleID),
		&updateAppStoreAppRequest{TeamID: &team.ID, SelfService: new(true)}, http.StatusOK, &updResp)
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
}

