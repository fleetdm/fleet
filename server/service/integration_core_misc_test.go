package service

// Cross-cutting tests for the core (no-license) suite that fit no single topic.
//
// Belongs here: the status, ping and version endpoints; license gating of
// premium-only endpoints on an unlicensed deployment; responses when MDM is not
// configured; API versioning; the translator; debug endpoints; and cross-origin
// request handling.
//
// This is the bucket of last resort. If a test has a real home in one of the other
// integration_core_* files, put it there instead.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/guregu/null.v3"
)

// TestMDMAnyMiddlewareAccess performs an end-to-end check through the HTTP
// handler to confirm the new middleware respects each platform toggle.
func (s *integrationTestSuite) TestMDMAnyMiddlewareAccess() {
	t := s.T()
	ctx := context.Background()
	appCfg, err := s.ds.AppConfig(ctx)
	require.NoError(t, err)

	ensureAppleMDMAssets := func() {
		assets := []fleet.MDMConfigAsset{
			{Name: fleet.MDMAssetCACert, Value: []byte("test-ca-cert")},
			{Name: fleet.MDMAssetCAKey, Value: []byte("test-ca-key")},
			{Name: fleet.MDMAssetAPNSCert, Value: []byte("test-apns-cert")},
			{Name: fleet.MDMAssetAPNSKey, Value: []byte("test-apns-key")},
		}
		if err := s.ds.InsertMDMConfigAssets(ctx, assets, nil); err != nil && !mysql.IsDuplicate(err) {
			require.NoError(t, err)
		}
	}
	ensureAppleMDMAssets()

	origMDM := appCfg.MDM
	defer func(orig fleet.MDM) {
		appCfg.MDM = orig
		require.NoError(t, s.ds.SaveAppConfig(ctx, appCfg))
		require.NoError(t, s.ds.SetAndroidEnabledAndConfigured(ctx, orig.AndroidEnabledAndConfigured))
	}(origMDM)

	const endpoint = "/api/latest/fleet/configuration_profiles"

	requestProfiles := func() *http.Response {
		req, err := http.NewRequest("GET", s.server.URL+endpoint, nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+s.token)

		resp, err := fleethttp.NewClient().Do(req)
		require.NoError(t, err)
		return resp
	}

	setConfig := func(apple, windows, android bool) {
		appCfg.MDM.EnabledAndConfigured = apple
		appCfg.MDM.WindowsEnabledAndConfigured = windows
		require.NoError(t, s.ds.SaveAppConfig(ctx, appCfg))

		require.NoError(t, s.ds.SetAndroidEnabledAndConfigured(ctx, android))

		appCfg, err = s.ds.AppConfig(ctx)
		require.NoError(t, err)
	}

	setConfig(false, false, false)
	res := s.Do("GET", endpoint, nil, http.StatusBadRequest)
	errMsg := extractServerErrorText(res.Body)
	require.Contains(t, errMsg, fleet.ErrMDMNotConfigured.Error())
	require.NoError(t, res.Body.Close())

	assertNotMDMNotConfigured := func() {
		resp := requestProfiles()
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.NoError(t, resp.Body.Close())
		require.NotEqual(t, http.StatusBadRequest, resp.StatusCode)
		require.NotContains(t, string(body), fleet.ErrMDMNotConfigured.Error())
	}

	setConfig(true, false, false)
	assertNotMDMNotConfigured()

	setConfig(false, true, false)
	assertNotMDMNotConfigured()

	setConfig(false, false, true)
	assertNotMDMNotConfigured()
}

func (s *integrationTestSuite) TestTranslator() {
	t := s.T()

	payload := translatorResponse{}
	params := translatorRequest{List: []fleet.TranslatePayload{
		{
			Type:    fleet.TranslatorTypeUserEmail,
			Payload: fleet.StringIdentifierToIDPayload{Identifier: "admin1@example.com"},
		},
	}}
	s.DoJSON("POST", "/api/latest/fleet/translate", &params, http.StatusOK, &payload)
	require.Len(t, payload.List, 1)

	assert.Equal(t, s.users[payload.List[0].Payload.Identifier].ID, payload.List[0].Payload.ID)

	// empty body
	s.DoJSON("POST", "/api/latest/fleet/translate", &translatorRequest{}, http.StatusBadRequest, &payload)

	s.DoJSON("POST", "/api/latest/fleet/translate", &translatorRequest{List: []fleet.TranslatePayload{{Type: "notavalidtype", Payload: fleet.StringIdentifierToIDPayload{}}}}, http.StatusBadRequest, &payload)
}

func (s *integrationTestSuite) TestCrossOriginJSONSecurity() {
	t := s.T()

	// valid request with no Origin or Referer headers
	createInviteReq := createInviteRequest{InvitePayload: fleet.InvitePayload{
		Email:      new("some email"),
		Name:       new("some name"),
		GlobalRole: null.StringFrom(fleet.RoleAdmin),
	}}
	createInviteResp := createInviteResponse{}
	s.DoJSON("POST", "/api/latest/fleet/invites", createInviteReq, http.StatusOK, &createInviteResp)
	require.NotNil(t, createInviteResp.Invite)
	require.NotZero(t, createInviteResp.Invite.ID)

	createInviteReq.Email = new("other@email.com")
	createInviteReq.Name = new("other name")
	req, err := json.Marshal(createInviteReq)
	require.NoError(t, err)

	// cross origin request with Origin header and no Content-Type
	resp := s.DoRawWithHeaders("POST", "/api/latest/fleet/invites", req, http.StatusUnsupportedMediaType, map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", s.withServer.token),
		"Origin":        "example.com",
	})
	resp.Body.Close()

	// cross origin request with Referer header and no Content-Type
	resp = s.DoRawWithHeaders("POST", "/api/latest/fleet/invites", req, http.StatusUnsupportedMediaType, map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", s.withServer.token),
		"Referer":       "example.com",
	})
	resp.Body.Close()

	// cross origin request with valid Content-Type
	resp = s.DoRawWithHeaders("POST", "/api/latest/fleet/invites", req, http.StatusOK, map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", s.withServer.token),
		"Origin":        "example.com",
		"Referer":       "example.com",
		"Content-Type":  "application/json",
	})
	resp.Body.Close()
}

func (s *integrationTestSuite) TestPremiumEndpointsWithoutLicense() {
	t := s.T()

	// list teams, none
	var listResp listTeamsResponse
	s.DoJSON("GET", "/api/latest/fleet/teams", nil, http.StatusPaymentRequired, &listResp)
	assert.Empty(t, listResp.Teams)

	// get team
	var getResp getTeamResponse
	s.DoJSON("GET", "/api/latest/fleet/teams/123", nil, http.StatusPaymentRequired, &getResp)
	assert.Nil(t, getResp.Team)

	// create team
	var tmResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", &createTeamRequest{}, http.StatusPaymentRequired, &tmResp)
	assert.Nil(t, tmResp.Team)

	// modify team
	s.DoJSON("PATCH", "/api/latest/fleet/teams/123", fleet.TeamPayload{}, http.StatusPaymentRequired, &tmResp)
	assert.Nil(t, tmResp.Team)

	// delete team
	var delResp deleteTeamResponse
	s.DoJSON("DELETE", "/api/latest/fleet/teams/123", nil, http.StatusPaymentRequired, &delResp)

	// apply team specs
	var specResp applyTeamSpecsResponse
	teamSpecs := applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{Name: "newteam", Secrets: &[]fleet.EnrollSecret{{Secret: "ABC"}}}}}
	s.DoJSON("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusPaymentRequired, &specResp)

	// modify team agent options
	s.DoJSON("POST", "/api/latest/fleet/teams/123/agent_options", nil, http.StatusPaymentRequired, &tmResp)
	assert.Nil(t, tmResp.Team)

	// list team users
	var usersResp listUsersResponse
	s.DoJSON("GET", "/api/latest/fleet/teams/123/users", nil, http.StatusPaymentRequired, &usersResp, "page", "1")
	assert.Empty(t, usersResp.Users)

	// add team users
	s.DoJSON("PATCH", "/api/latest/fleet/teams/123/users", modifyTeamUsersRequest{Users: []fleet.TeamUser{{User: fleet.User{ID: 1}}}}, http.StatusPaymentRequired, &tmResp)
	assert.Nil(t, tmResp.Team)

	// delete team users
	s.DoJSON("DELETE", "/api/latest/fleet/teams/123/users", modifyTeamUsersRequest{Users: []fleet.TeamUser{{User: fleet.User{ID: 1}}}}, http.StatusPaymentRequired, &tmResp)
	assert.Nil(t, tmResp.Team)

	// get team enroll secrets
	var secResp teamEnrollSecretsResponse
	s.DoJSON("GET", "/api/latest/fleet/teams/123/secrets", nil, http.StatusPaymentRequired, &secResp)
	assert.Empty(t, secResp.Secrets)

	// modify team enroll secrets
	s.DoJSON("PATCH", "/api/latest/fleet/teams/123/secrets", modifyTeamEnrollSecretsRequest{Secrets: []fleet.EnrollSecret{{Secret: "DEF"}}}, http.StatusPaymentRequired, &secResp)
	assert.Empty(t, secResp.Secrets)

	// get apple BM configuration
	var appleBMResp getAppleBMResponse
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple_bm", nil, http.StatusPaymentRequired, &appleBMResp)
	assert.Nil(t, appleBMResp.AppleBM)

	// batch-apply an empty set of MDM profiles succeeds even though MDM is not
	// enabled, because it wouldn't change anything (and it needs to support the
	// case where `fleetctl get config`'s output is used as input to `fleetctl
	// apply`).
	s.Do("POST", "/api/latest/fleet/mdm/apple/profiles/batch", nil, http.StatusNoContent)

	// batch-apply a non-empty set of MDM profiles fails
	res := s.Do("POST", "/api/latest/fleet/mdm/apple/profiles/batch",
		map[string]any{"profiles": [][]byte{[]byte(`xyz`)}}, http.StatusUnprocessableEntity)
	errMsg := extractServerErrorText(res.Body)
	require.Contains(t, errMsg, fleet.ErrMDMNotConfigured.Error())

	// update MDM disk encryption
	_ = s.Do("POST", "/api/latest/fleet/disk_encryption", fleet.MDMAppleSettingsPayload{}, http.StatusPaymentRequired)

	// update MDM host name template
	_ = s.Do("POST", "/api/latest/fleet/host_name_template", updateHostNameTemplateRequest{}, http.StatusPaymentRequired)

	// Turn on MDM.
	ctx := t.Context()
	appCfg, err := s.ds.AppConfig(ctx)
	require.NoError(t, err)
	origEnabledAndConfigured := appCfg.MDM.EnabledAndConfigured
	appCfg.MDM.EnabledAndConfigured = true
	err = s.ds.SaveAppConfig(ctx, appCfg)
	require.NoError(t, err)
	defer func() {
		appCfg.MDM.EnabledAndConfigured = origEnabledAndConfigured
		err = s.ds.SaveAppConfig(ctx, appCfg)
		require.NoError(t, err)
	}()

	// device migrate mdm endpoint returns an error if not premium
	createHostAndDeviceToken(t, s.ds, "some-token")
	s.Do("POST", fmt.Sprintf("/api/v1/fleet/device/%s/migrate_mdm", "some-token"), nil, http.StatusPaymentRequired)

	// uploading a DDM declaration with a Fleet variable returns a license error
	// (single profile upload endpoint)
	ddmWithFleetVar := []byte(`{
		"Type": "com.apple.configuration.management.test",
		"Identifier": "com.example.fleetvar-test",
		"Payload": {"Value": "$FLEET_VAR_HOST_HARDWARE_SERIAL"}
	}`)
	body, headers := generateNewProfileMultipartRequest(t, "fleetvar-test.json", ddmWithFleetVar, s.token, nil)
	res = s.DoRawWithHeaders("POST", "/api/latest/fleet/configuration_profiles", body.Bytes(), http.StatusPaymentRequired, headers)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Requires Fleet Premium license")

	// uploading a DDM declaration with a Fleet variable returns a license error
	// (batch profiles endpoint)
	res = s.Do("POST", "/api/latest/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
		{Name: "N1", Contents: ddmWithFleetVar},
	}}, http.StatusPaymentRequired)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Requires Fleet Premium license")

	// software titles
	// a normal request works fine
	var resp listSoftwareTitlesResponse
	s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{}, http.StatusOK, &resp)
	// TODO: there's a race condition that makes this number change from
	// 0-3, commenting for now since it's not really relevant for this
	// test (we only care about the response status)
	// require.NotEmpty(t, 0, resp.Count)
	// require.Nil(t, resp.SoftwareTitles)

	// a request with a team_id parameter returns a license error
	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{}, http.StatusPaymentRequired, &resp,
		"team_id", "1",
	)

	// a request with a premium vulnerability filter returns a license error
	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{fleet.SoftwareTitleListOptions{VulnerableOnly: true, MinimumCVSS: 7.5}}, http.StatusPaymentRequired, &resp,
	)
	verResp := listSoftwareVersionsResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/versions",
		listSoftwareRequest{fleet.SoftwareListOptions{VulnerableOnly: true, MinimumCVSS: 7.5}}, http.StatusPaymentRequired, &verResp,
	)
	countResp := countSoftwareResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/count",
		listSoftwareRequest{fleet.SoftwareListOptions{VulnerableOnly: true, MinimumCVSS: 7.5}}, http.StatusPaymentRequired, &countResp,
	)

	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{fleet.SoftwareTitleListOptions{VulnerableOnly: true, MaximumCVSS: 7.5}}, http.StatusPaymentRequired, &resp,
	)
	verResp = listSoftwareVersionsResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/versions",
		listSoftwareRequest{fleet.SoftwareListOptions{VulnerableOnly: true, MaximumCVSS: 7.5}}, http.StatusPaymentRequired, &verResp,
	)
	countResp = countSoftwareResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/count",
		listSoftwareRequest{fleet.SoftwareListOptions{VulnerableOnly: true, MaximumCVSS: 7.5}}, http.StatusPaymentRequired, &countResp,
	)

	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{fleet.SoftwareTitleListOptions{VulnerableOnly: true, KnownExploit: true}}, http.StatusPaymentRequired, &resp,
	)
	verResp = listSoftwareVersionsResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/versions",
		listSoftwareRequest{fleet.SoftwareListOptions{VulnerableOnly: true, KnownExploit: true}}, http.StatusPaymentRequired, &verResp,
	)
	countResp = countSoftwareResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/count",
		listSoftwareRequest{fleet.SoftwareListOptions{VulnerableOnly: true, KnownExploit: true}}, http.StatusPaymentRequired, &countResp,
	)

	// lock/unlock/wipe a host. Wipe is Premium-only only for non-Android platforms (Android COBO wipe is available on Fleet Free).
	wipeHost := s.createHosts(t, "darwin")[0]
	s.Do("POST", "/api/v1/fleet/hosts/123/lock", nil, http.StatusPaymentRequired)
	s.Do("POST", "/api/v1/fleet/hosts/123/unlock", nil, http.StatusPaymentRequired)
	s.Do("POST", fmt.Sprintf("/api/v1/fleet/hosts/%d/wipe", wipeHost.ID), nil, http.StatusPaymentRequired)

	// try to update the enable_release_device_manually setting, requires premium.
	s.Do("PATCH", "/api/v1/fleet/setup_experience", fleet.MDMAppleSetupPayload{EnableReleaseDeviceManually: new(true)}, http.StatusPaymentRequired)

	res = s.Do("PATCH", "/api/v1/fleet/config", json.RawMessage(`{
		"mdm": { "macos_setup": { "enable_release_device_manually": true } }
	}`), http.StatusUnprocessableEntity)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "missing or invalid license")

	res = s.Do("PATCH", "/api/v1/fleet/config", json.RawMessage(`{
		"mdm": { "apple_require_hardware_attestation": true }
	}`), http.StatusUnprocessableEntity)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "missing or invalid license")

	res = s.Do("PATCH", "/api/v1/fleet/config", json.RawMessage(`{
		"mdm": { "windows_migration_enabled": true }
	}`), http.StatusUnprocessableEntity)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "missing or invalid license")

	// list API endpoints requires premium license
	var listAPIEndpointsResp listAPIEndpointsResponse
	s.DoJSON("GET", "/api/latest/fleet/rest_api", nil, http.StatusPaymentRequired, &listAPIEndpointsResp)
}

func (s *integrationTestSuite) TestScriptsEndpointsWithoutLicense() {
	t := s.T()

	// this is just checking that the endpoints do not fail with "no license", the actual tests
	// for scripts endpoints are in the enterprise integrations tests.

	// run a script
	var runResp fleet.RunScriptResponse
	s.DoJSON("POST", "/api/latest/fleet/scripts/run", fleet.HostScriptRequestPayload{HostID: 1, ScriptContents: "echo foo"}, http.StatusNotFound, &runResp)

	// run a script sync
	s.DoJSON("POST", "/api/latest/fleet/scripts/run/sync", fleet.HostScriptRequestPayload{HostID: 1, ScriptContents: "echo foo"}, http.StatusNotFound, &runResp)

	// get script result
	var scriptResultResp fleet.GetScriptResultResponse
	s.DoJSON("GET", "/api/latest/fleet/scripts/results/test-id", nil, http.StatusNotFound, &scriptResultResp)

	// create a saved script
	body, headers := generateNewScriptMultipartRequest(t,
		"myscript.sh", []byte(`echo "hello"`), s.token, nil)
	s.DoRawWithHeaders("POST", "/api/latest/fleet/scripts", body.Bytes(), http.StatusOK, headers)

	// run a saved script by name without team id (should fail host not found)
	res := s.Do("POST", "/api/latest/fleet/scripts/run/sync", fleet.RunScriptSyncRequest{ScriptName: "myscript.sh"}, http.StatusNotFound)
	errMsg := extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Host was not found in the datastore")

	// run a saved script by name with team id (should fail with license error)
	res = s.Do("POST", "/api/latest/fleet/scripts/run/sync", fleet.RunScriptSyncRequest{ScriptName: "myscript.sh", TeamID: 1}, http.StatusPaymentRequired)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Requires Fleet Premium license")

	// scripts containing Fleet variables require a premium license
	res = s.Do("POST", "/api/latest/fleet/scripts/run", fleet.HostScriptRequestPayload{HostID: 1, ScriptContents: "echo $FLEET_VAR_HOST_UUID"}, http.StatusPaymentRequired)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Requires Fleet Premium license")

	body, headers = generateNewScriptMultipartRequest(t,
		"varscript.sh", []byte("echo $FLEET_VAR_HOST_UUID"), s.token, nil)
	res = s.DoRawWithHeaders("POST", "/api/latest/fleet/scripts", body.Bytes(), http.StatusPaymentRequired, headers)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Requires Fleet Premium license")

	res = s.Do("POST", "/api/v1/fleet/scripts/batch", fleet.BatchSetScriptsRequest{Scripts: []fleet.ScriptPayload{
		{Name: "vars.sh", ScriptContents: []byte("echo $FLEET_VAR_HOST_UUID")},
	}}, http.StatusPaymentRequired)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Requires Fleet Premium license")

	// delete a saved script
	var delScriptResp fleet.DeleteScriptResponse
	s.DoJSON("DELETE", "/api/latest/fleet/scripts/123", nil, http.StatusNotFound, &delScriptResp)

	// list saved scripts
	var listScriptsResp fleet.ListScriptsResponse
	s.DoJSON("GET", "/api/latest/fleet/scripts", nil, http.StatusOK, &listScriptsResp, "per_page", "10")

	// get a saved script
	var getScriptResp fleet.GetScriptResponse
	s.DoJSON("GET", "/api/latest/fleet/scripts/123", nil, http.StatusNotFound, &getScriptResp)

	// get host script details
	var getHostScriptDetailsResp fleet.GetHostScriptDetailsResponse
	s.DoJSON("GET", "/api/latest/fleet/hosts/123/scripts", nil, http.StatusNotFound, &getHostScriptDetailsResp)

	// batch set scripts
	s.Do("POST", "/api/v1/fleet/scripts/batch", fleet.BatchSetScriptsRequest{Scripts: nil}, http.StatusOK)
}

func (s *integrationTestSuite) TestStatus() {
	var statusResp statusResponse
	s.DoJSON("GET", "/api/latest/fleet/status/result_store", nil, http.StatusOK, &statusResp)
	s.DoJSON("GET", "/api/latest/fleet/status/live_query", nil, http.StatusOK, &statusResp)
}

func (s *integrationTestSuite) TestPingEndpoints() {
	t := s.T()

	s.DoRaw("HEAD", "/api/fleet/orbit/ping", nil, http.StatusOK)
	// unauthenticated works too
	s.DoRawNoAuth("HEAD", "/api/fleet/orbit/ping", nil, http.StatusOK)

	s.DoRaw("HEAD", "/api/fleet/device/ping", nil, http.StatusOK)
	// unauthenticated works too
	s.DoRawNoAuth("HEAD", "/api/fleet/device/ping", nil, http.StatusOK)

	// device authenticated ping
	createHostAndDeviceToken(t, s.ds, "ping-token")
	s.DoRaw("HEAD", fmt.Sprintf("/api/v1/fleet/device/%s/ping", "ping-token"), nil, http.StatusOK)
	s.DoRawNoAuth("HEAD", fmt.Sprintf("/api/v1/fleet/device/%s/ping", "ping-token"), nil, http.StatusOK)
	s.DoRaw("HEAD", fmt.Sprintf("/api/v1/fleet/device/%s/ping", "bozo-token"), nil, http.StatusUnauthorized)
	s.DoRawNoAuth("HEAD", fmt.Sprintf("/api/v1/fleet/device/%s/ping", "bozo-token"), nil, http.StatusUnauthorized)
}

func (s *integrationTestSuite) TestInitiateDeviceSSOFreeTier() {
	t := s.T()

	createHostAndDeviceToken(t, s.ds, "device-sso-token")

	// free tier: no InitiateDeviceSSO implementation beyond the license error,
	// same shape as every other premium-only device endpoint.
	res := s.DoRawNoAuth("POST", "/api/latest/fleet/device/device-sso-token/sso", nil, http.StatusPaymentRequired)
	errMsg := extractServerErrorText(res.Body)
	assert.Contains(t, errMsg, fleet.ErrMissingLicense.Error())

	// invalid device token behaves like every other device-authenticated route
	s.DoRawNoAuth("POST", "/api/latest/fleet/device/bozo-token/sso", nil, http.StatusUnauthorized)
}

func (s *integrationTestSuite) TestMDMNotConfiguredEndpoints() {
	t := s.T()

	// create a host with device token to test device authenticated routes
	tkn := "D3V1C370K3N"
	h := createHostAndDeviceToken(t, s.ds, tkn)
	orbitKey := setOrbitEnrollment(t, h, s.ds)
	h.OrbitNodeKey = &orbitKey

	windowsOnly := windowsMDMConfigurationRequiredEndpoints()
	androidOnly := androidMDMConfigurationRequiredEndpoints()

	for _, route := range mdmConfigurationRequiredEndpoints() {
		var expectedErr fleet.ErrWithStatusCode = fleet.ErrMDMNotConfigured
		path := route.path
		if slices.Contains(windowsOnly, path) {
			expectedErr = fleet.ErrWindowsMDMNotConfigured
		} else if slices.Contains(androidOnly, path) {
			expectedErr = fleet.ErrAndroidMDMNotConfigured
		}

		if route.deviceAuthenticated {
			path = fmt.Sprintf(path, tkn)
		}

		// build the body of the request
		var params any
		var multipartBody *bytes.Buffer
		var headers map[string]string
		switch {
		case route.method == "POST" && route.path == "/api/fleet/orbit/setup_experience/status":
			params = fleet.GetOrbitSetupExperienceStatusRequest{
				OrbitNodeKey: *h.OrbitNodeKey,
			}

		case route.method == "POST" && route.path == "/api/latest/fleet/software/web_apps":
			multipartBody, headers = generateMultipartRequest(t, "", "", nil, s.token, map[string][]string{
				"title": {"Test App"},
				"url":   {"https://example.com"},
			})

		case route.method == "PATCH" && (route.path == "/api/latest/fleet/setup_experience" || route.path == "/api/latest/fleet/mdm/apple/setup"):
			// These routes don't require MDM because they can be used to change end-user auth, but they do require a license.
			expectedErr = fleet.ErrMissingLicense
		}

		var res *http.Response
		if multipartBody != nil {
			res = s.DoRawWithHeaders(route.method, path, multipartBody.Bytes(), expectedErr.StatusCode(), headers)
		} else {
			res = s.Do(route.method, path, params, expectedErr.StatusCode())
		}
		errMsg := extractServerErrorText(res.Body)
		assert.Contains(t, errMsg, expectedErr.Error(), "%s %s", route.method, path)
	}

	fleetdmSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Setenv("TEST_FLEETDM_API_URL", fleetdmSrv.URL)
	t.Cleanup(fleetdmSrv.Close)

	// Always accessible
	var reqCSRResp requestMDMAppleCSRResponse
	s.DoJSON("POST", "/api/latest/fleet/mdm/apple/request_csr", requestMDMAppleCSRRequest{EmailAddress: "a@b.c", Organization: "test"}, http.StatusOK, &reqCSRResp)
	s.Do("POST", "/api/latest/fleet/mdm/apple/dep/key_pair", nil, http.StatusOK)
}

// this test can be deleted once the "v1" version is removed.
func (s *integrationTestSuite) TestAPIVersion_v1_2022_04() {
	t := s.T()

	// create a query that can be scheduled
	qr, err := s.ds.NewQuery(context.Background(), &fleet.Query{
		Name:           "TestQuery2",
		Query:          "select * from osquery;",
		ObserverCanRun: true,
		Saved:          true,
		Logging:        fleet.LoggingSnapshot,
	})
	require.NoError(t, err)

	// try to schedule that query on the endpoint that is deprecated
	// in that version
	gsParams := fleet.ScheduledQueryPayload{QueryID: &qr.ID, Interval: new(uint(42))}
	res := s.DoRaw("POST", "/api/2022-04/fleet/global/schedule", jsonMustMarshal(t, gsParams), http.StatusNotFound)
	res.Body.Close()
	// use the correct version for that deprecated API
	createResp := globalScheduleQueryResponse{}
	s.DoJSON("POST", "/api/v1/fleet/global/schedule", gsParams, http.StatusOK, &createResp)
	require.NotZero(t, createResp.Scheduled.ID)

	// list the scheduled queries with the new endpoint, but the old version
	res = s.DoRaw("GET", "/api/v1/fleet/schedule", nil, http.StatusMethodNotAllowed)
	res.Body.Close()

	// list again, this time with the correct version
	gs := fleet.GlobalSchedulePayload{}
	s.DoJSON("GET", "/api/2022-04/fleet/schedule", nil, http.StatusOK, &gs)
	require.Len(t, gs.GlobalSchedule, 1)

	// delete using the old endpoint but on the wrong new version
	res = s.DoRaw("DELETE", fmt.Sprintf("/api/2022-04/fleet/global/schedule/%d", createResp.Scheduled.ID), nil, http.StatusNotFound)
	res.Body.Close()
	// properly delete with old endpoint and old version
	var delResp deleteGlobalScheduleResponse
	s.DoJSON("DELETE", fmt.Sprintf("/api/v1/fleet/global/schedule/%d", createResp.Scheduled.ID), nil, http.StatusOK, &delResp)
}

func (s *integrationTestSuite) TestDebugDB() {
	t := s.T()
	var response map[string]string
	s.DoJSON("GET", "/debug/db/locks", nil, http.StatusOK, &response)
	assert.Empty(t, response)

	var responseString string
	s.DoJSON("GET", "/debug/db/innodb-status", nil, http.StatusOK, &responseString)
	assert.Contains(t, responseString, "INNODB MONITOR OUTPUT")
}

func (s *integrationTestSuite) TestConditionalAccessRequiresPremium() {
	// Microsoft compliance partner APIs should fail on Fleet Free (this suite
	// runs without a premium license).
	var r conditionalAccessMicrosoftCreateResponse
	s.DoJSON("POST", "/api/latest/fleet/conditional-access/microsoft", conditionalAccessMicrosoftCreateRequest{
		MicrosoftTenantID: "foobar",
	}, http.StatusPaymentRequired, &r)
	var c conditionalAccessMicrosoftConfirmResponse
	s.DoJSON("POST", "/api/latest/fleet/conditional-access/microsoft/confirm", conditionalAccessMicrosoftConfirmRequest{},
		http.StatusPaymentRequired, &c)
	var d conditionalAccessMicrosoftDeleteResponse
	s.DoJSON("DELETE", "/api/latest/fleet/conditional-access/microsoft", nil,
		http.StatusPaymentRequired, &d)
}
