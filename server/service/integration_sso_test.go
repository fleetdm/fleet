package service

import (
	"bytes"
	"compress/flate"
	"context"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/datastore/redis/redistest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/sso"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type integrationSSOTestSuite struct {
	suite.Suite
	withServer
}

func (s *integrationSSOTestSuite) SetupSuite() {
	s.withDS.SetupSuite("integrationSSOTestSuite")

	pool := redistest.SetupRedis(s.T(), "zz", false, false, false)
	users, server := RunServerForTestsWithDS(s.T(), s.ds, &TestServerOpts{Pool: pool})
	s.server = server
	s.users = users
	s.token = s.getTestAdminToken()
}

func TestIntegrationsSSO(t *testing.T) {
	testingSuite := new(integrationSSOTestSuite)
	testingSuite.s = &testingSuite.Suite
	suite.Run(t, testingSuite)
}

func (s *integrationSSOTestSuite) TestGetSSOSettings() {
	t := s.T()

	acResp := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"sso_settings": {
			"enable_sso": true,
			"entity_id": "https://localhost:8080",
			"issuer_uri": "http://localhost:8080/simplesaml/saml2/idp/SSOService.php",
			"idp_name": "SimpleSAML",
			"metadata_url": "http://localhost:9080/simplesaml/saml2/idp/metadata.php",
			"enable_jit_provisioning": false
		}
	}`), http.StatusOK, &acResp)
	require.NotNil(t, acResp)

	// double-check the settings
	var resGet ssoSettingsResponse
	s.DoJSON("GET", "/api/v1/fleet/sso", nil, http.StatusOK, &resGet)
	require.True(t, resGet.Settings.SSOEnabled)

	// initiate an SSO auth
	var resIni initiateSSOResponse
	s.DoJSON("POST", "/api/v1/fleet/sso", map[string]string{}, http.StatusOK, &resIni)
	require.NotEmpty(t, resIni.URL)

	parsed, err := url.Parse(resIni.URL)
	require.NoError(t, err)
	q := parsed.Query()
	encoded := q.Get("SAMLRequest")
	assert.NotEmpty(t, encoded)
	authReq := inflate(t, encoded)
	assert.Equal(t, "https://localhost:8080", authReq.Issuer.Url)
	assert.Equal(t, "Fleet", authReq.ProviderName)
	assert.True(t, strings.HasPrefix(authReq.ID, "id"), authReq.ID)
}

func (s *integrationSSOTestSuite) TestSSOLogin() {
	t := s.T()

	acResp := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"sso_settings": {
			"enable_sso": true,
			"entity_id": "https://localhost:8080",
			"issuer_uri": "http://localhost:8080/simplesaml/saml2/idp/SSOService.php",
			"idp_name": "SimpleSAML",
			"metadata_url": "http://localhost:9080/simplesaml/saml2/idp/metadata.php"
		}
	}`), http.StatusOK, &acResp)
	require.NotNil(t, acResp)

	// Register current number of activities.
	activitiesResp := listActivitiesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &activitiesResp)
	require.NoError(t, activitiesResp.Err)
	oldActivitiesCount := len(activitiesResp.Activities)

	// users can't login if they don't have an account on free plans
	_, body := s.LoginSSOUser("sso_user", "user123#")
	require.Contains(t, body, "/login?status=account_invalid")

	newActivitiesCount := 1
	checkNewFailedLoginActivity := func() {
		activitiesResp = listActivitiesResponse{}
		s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &activitiesResp)
		require.NoError(t, activitiesResp.Err)
		require.Len(t, activitiesResp.Activities, oldActivitiesCount+newActivitiesCount)
		sort.Slice(activitiesResp.Activities, func(i, j int) bool {
			return activitiesResp.Activities[i].ID < activitiesResp.Activities[j].ID
		})
		activity := activitiesResp.Activities[len(activitiesResp.Activities)-1]
		require.Equal(t, activity.Type, fleet.ActivityTypeUserFailedLogin{}.ActivityName())
		require.NotNil(t, activity.Details)
		actDetails := fleet.ActivityTypeUserFailedLogin{}
		err := json.Unmarshal(*activity.Details, &actDetails)
		require.NoError(t, err)
		require.Equal(t, "sso_user@example.com", actDetails.Email)

		newActivitiesCount++
	}

	// A new activity item for the failed SSO login is created.
	checkNewFailedLoginActivity()

	// users can't login if they don't have an account on free plans
	// even if JIT provisioning is enabled
	ac, err := s.ds.AppConfig(context.Background())
	ac.SSOSettings.EnableJITProvisioning = true
	require.NoError(t, err)
	err = s.ds.SaveAppConfig(context.Background(), ac)
	require.NoError(t, err)
	_, body = s.LoginSSOUser("sso_user", "user123#")
	require.Contains(t, body, "/login?status=account_invalid")

	// A new activity item for the failed SSO login is created.
	checkNewFailedLoginActivity()

	// an user created by an admin without SSOEnabled can't log-in
	params := fleet.UserPayload{
		Name:       ptr.String("SSO User 1"),
		Email:      ptr.String("sso_user@example.com"),
		GlobalRole: ptr.String(fleet.RoleObserver),
		SSOEnabled: ptr.Bool(false),
	}
	s.Do("POST", "/api/latest/fleet/users/admin", &params, http.StatusUnprocessableEntity)
	_, body = s.LoginSSOUser("sso_user", "user123#")
	require.Contains(t, body, "/login?status=account_invalid")

	// A new activity item for the failed SSO login is created.
	checkNewFailedLoginActivity()

	// A user created by an admin with SSOEnabled is able to log-in
	params = fleet.UserPayload{
		Name:       ptr.String("SSO User 2"),
		Email:      ptr.String("sso_user2@example.com"),
		GlobalRole: ptr.String(fleet.RoleObserver),
		SSOEnabled: ptr.Bool(true),
	}
	s.Do("POST", "/api/latest/fleet/users/admin", &params, http.StatusOK)
	auth, body := s.LoginSSOUser("sso_user2", "user123#")
	assert.Equal(t, "sso_user2@example.com", auth.UserID())
	assert.Equal(t, "SSO User 2", auth.UserDisplayName())
	require.Contains(t, body, "Redirecting to Fleet at  ...")

	// a new activity item is created
	activitiesResp = listActivitiesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &activitiesResp)
	require.NoError(t, activitiesResp.Err)
	require.NotEmpty(t, activitiesResp.Activities)
	require.Condition(t, func() bool {
		for _, a := range activitiesResp.Activities {
			if (a.Type == fleet.ActivityTypeUserLoggedIn{}.ActivityName()) && *a.ActorEmail == auth.UserID() {
				return true
			}
		}
		return false
	})
}

func inflate(t *testing.T, s string) *sso.AuthnRequest {
	t.Helper()

	decoded, err := base64.StdEncoding.DecodeString(s)
	require.NoError(t, err)

	r := flate.NewReader(bytes.NewReader(decoded))
	defer r.Close()

	var req sso.AuthnRequest
	require.NoError(t, xml.NewDecoder(r).Decode(&req))
	return &req
}
