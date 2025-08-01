package service

import (
	"bytes"
	"compress/flate"
	"context"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/crewjam/saml"
	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/datastore/redis/redistest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	kitlog "github.com/go-kit/log"
	"github.com/jmoiron/sqlx"
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
	opts := &TestServerOpts{Pool: pool}
	if os.Getenv("FLEET_INTEGRATION_TESTS_DISABLE_LOG") != "" {
		opts.Logger = kitlog.NewNopLogger()
	}
	users, server := RunServerForTestsWithDS(s.T(), s.ds, opts)
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

	// Javascript redirect URLs are forbidden
	var resIni initiateSSOResponse
	s.DoJSON("POST", "/api/v1/fleet/sso", map[string]string{"relay_url": "javascript:alert(1)"}, http.StatusBadRequest, &resIni)

	// initiate an SSO auth
	resIni = initiateSSOResponse{}
	s.DoJSON("POST", "/api/v1/fleet/sso", map[string]string{}, http.StatusOK, &resIni)
	require.NotEmpty(t, resIni.URL)

	parsed, err := url.Parse(resIni.URL)
	require.NoError(t, err)
	q := parsed.Query()
	encoded := q.Get("SAMLRequest")
	assert.NotEmpty(t, encoded)
	authReq := inflate(t, encoded)
	assert.Equal(t, "https://localhost:8080", authReq.Issuer.Value)
	assert.Equal(t, "Fleet", authReq.ProviderName)
	assert.True(t, strings.HasPrefix(authReq.ID, "id"), authReq.ID)
}

func (s *integrationSSOTestSuite) TestSSOInvalidMetadataURL() {
	t := s.T()

	badMetadataUrl := "https://www.fleetdm.com"
	acResp := appConfigResponse{}
	s.DoJSON(
		"PATCH", "/api/latest/fleet/config", json.RawMessage(
			`{
		"sso_settings": {
			"enable_sso": true,
			"entity_id": "https://localhost:8080",
			"idp_name": "SimpleSAML",
			"metadata_url": "`+badMetadataUrl+`",
			"enable_jit_provisioning": false
		}
	}`,
		), http.StatusOK, &acResp,
	)
	require.NotNil(t, acResp)

	var resIni initiateSSOResponse
	expectedStatus := http.StatusBadRequest
	t.Logf("Expecting 400 %v status when bad SSO metadata_url is set: %v", expectedStatus, badMetadataUrl)
	s.DoJSON("POST", "/api/v1/fleet/sso", map[string]string{}, expectedStatus, &resIni)
}

func (s *integrationSSOTestSuite) TestSSOInvalidMetadata() {
	t := s.T()

	badMetadata := "<EntityDescriptor>foo</EntityDescriptor>"
	acResp := appConfigResponse{}
	s.DoJSON(
		"PATCH", "/api/latest/fleet/config", json.RawMessage(
			`{
		"sso_settings": {
			"enable_sso": true,
			"entity_id": "https://localhost:8080",
			"idp_name": "SimpleSAML",
			"metadata": "`+badMetadata+`",
			"metadata_url": "",
			"enable_jit_provisioning": false
		}
	}`,
		), http.StatusOK, &acResp,
	)
	require.NotNil(t, acResp)

	var resIni initiateSSOResponse
	expectedStatus := http.StatusBadRequest
	t.Logf("Expecting %v status when bad SSO metadata is provided: %v", expectedStatus, badMetadata)
	s.DoJSON("POST", "/api/v1/fleet/sso", map[string]string{}, expectedStatus, &resIni)
}

func (s *integrationSSOTestSuite) TestSSOValidation() {
	acResp := appConfigResponse{}
	// Test we are validating metadata_url
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"sso_settings": {
			"enable_sso": true,
			"entity_id": "https://localhost:8080",
			"idp_name": "SimpleSAML",
			"metadata_url": "ssh://localhost:9080/simplesaml/saml2/idp/metadata.php"
		}
	}`), http.StatusUnprocessableEntity, &acResp)
}

func (s *integrationSSOTestSuite) TestSSOLogin() {
	t := s.T()

	acResp := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
        "server_settings": {
          "server_url": "https://localhost:8080"
        },
		"sso_settings": {
			"enable_sso": true,
			"entity_id": "https://localhost:8080",
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
	body := s.LoginSSOUser("sso_user", "user123#")
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
	require.NoError(t, err)
	ac.SSOSettings.EnableJITProvisioning = true
	err = s.ds.SaveAppConfig(context.Background(), ac)
	require.NoError(t, err)
	body = s.LoginSSOUser("sso_user", "user123#")
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
	body = s.LoginSSOUser("sso_user", "user123#")
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
	body = s.LoginSSOUser("sso_user2", "user123#")
	require.Contains(t, body, "Redirecting to Fleet at  ...")

	// a new activity item is created
	activitiesResp = listActivitiesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &activitiesResp)
	require.NoError(t, activitiesResp.Err)
	require.NotEmpty(t, activitiesResp.Activities)
	require.Condition(t, func() bool {
		for _, a := range activitiesResp.Activities {
			if (a.Type == fleet.ActivityTypeUserLoggedIn{}.ActivityName()) && *a.ActorEmail == "sso_user2@example.com" {
				return true
			}
		}
		return false
	})
}

func (s *integrationSSOTestSuite) TestPerformRequiredPasswordResetWithSSO() {
	// ensure that on exit, the admin token is used
	defer func() { s.token = s.getTestAdminToken() }()

	t := s.T()

	// create a non-SSO user
	var createResp createUserResponse
	userRawPwd := test.GoodPassword
	params := fleet.UserPayload{
		Name:       ptr.String("extra"),
		Email:      ptr.String("extra@asd.com"),
		Password:   ptr.String(userRawPwd),
		GlobalRole: ptr.String(fleet.RoleObserver),
	}
	s.DoJSON("POST", "/api/latest/fleet/users/admin", params, http.StatusOK, &createResp)
	assert.NotZero(t, createResp.User.ID)
	assert.True(t, createResp.User.AdminForcedPasswordReset)
	nonSSOUser := *createResp.User

	// enable SSO
	acResp := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"sso_settings": {
			"enable_sso": true,
			"entity_id": "https://localhost:8080",
			"idp_name": "SimpleSAML",
			"metadata_url": "http://localhost:9080/simplesaml/saml2/idp/metadata.php"
		}
	}`), http.StatusOK, &acResp)
	require.NotNil(t, acResp)

	// perform a required password change using the non-SSO user, works
	s.token = s.getTestToken(nonSSOUser.Email, userRawPwd)
	perfPwdResetResp := performRequiredPasswordResetResponse{}
	newRawPwd := "new_password2!"
	s.DoJSON("POST", "/api/latest/fleet/perform_required_password_reset", performRequiredPasswordResetRequest{
		Password: newRawPwd,
		ID:       nonSSOUser.ID,
	}, http.StatusOK, &perfPwdResetResp)
	require.False(t, perfPwdResetResp.User.AdminForcedPasswordReset)

	// trick the user into one with SSO enabled (we could create that user but it
	// won't have a password nor an API token to use for the request, so we mock
	// it in the DB)
	mysql.ExecAdhocSQL(t, s.ds, func(db sqlx.ExtContext) error {
		_, err := db.ExecContext(
			context.Background(),
			"UPDATE users SET sso_enabled = 1, admin_forced_password_reset = 1 WHERE id = ?",
			nonSSOUser.ID,
		)
		return err
	})

	// perform a required password change using the mocked SSO user, disallowed
	perfPwdResetResp = performRequiredPasswordResetResponse{}
	newRawPwd = "new_password2!"
	s.DoJSON("POST", "/api/latest/fleet/perform_required_password_reset", performRequiredPasswordResetRequest{
		Password: newRawPwd,
		ID:       nonSSOUser.ID,
	}, http.StatusForbidden, &perfPwdResetResp)
}

func inflate(t *testing.T, s string) *saml.AuthnRequest {
	t.Helper()

	decoded, err := base64.StdEncoding.DecodeString(s)
	require.NoError(t, err)

	r := flate.NewReader(bytes.NewReader(decoded))
	defer r.Close()

	var req saml.AuthnRequest
	require.NoError(t, xml.NewDecoder(r).Decode(&req))
	return &req
}

func (s *integrationSSOTestSuite) TestSSOLoginWithMetadata() {
	t := s.T()

	acResp := appConfigResponse{}
	metadata, err := json.Marshal([]byte(`<?xml version="1.0"?>
<md:EntityDescriptor xmlns:md="urn:oasis:names:tc:SAML:2.0:metadata" xmlns:ds="http://www.w3.org/2000/09/xmldsig#" entityID="http://localhost:9080/simplesaml/saml2/idp/metadata.php">
  <md:IDPSSODescriptor protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol">
    <md:KeyDescriptor use="signing">
      <ds:KeyInfo xmlns:ds="http://www.w3.org/2000/09/xmldsig#">
        <ds:X509Data>
          <ds:X509Certificate>MIIDXTCCAkWgAwIBAgIJALmVVuDWu4NYMA0GCSqGSIb3DQEBCwUAMEUxCzAJBgNVBAYTAkFVMRMwEQYDVQQIDApTb21lLVN0YXRlMSEwHwYDVQQKDBhJbnRlcm5ldCBXaWRnaXRzIFB0eSBMdGQwHhcNMTYxMjMxMTQzNDQ3WhcNNDgwNjI1MTQzNDQ3WjBFMQswCQYDVQQGEwJBVTETMBEGA1UECAwKU29tZS1TdGF0ZTEhMB8GA1UECgwYSW50ZXJuZXQgV2lkZ2l0cyBQdHkgTHRkMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAzUCFozgNb1h1M0jzNRSCjhOBnR+uVbVpaWfXYIR+AhWDdEe5ryY+CgavOg8bfLybyzFdehlYdDRgkedEB/GjG8aJw06l0qF4jDOAw0kEygWCu2mcH7XOxRt+YAH3TVHa/Hu1W3WjzkobqqqLQ8gkKWWM27fOgAZ6GieaJBN6VBSMMcPey3HWLBmc+TYJmv1dbaO2jHhKh8pfKw0W12VM8P1PIO8gv4Phu/uuJYieBWKixBEyy0lHjyixYFCR12xdh4CA47q958ZRGnnDUGFVE1QhgRacJCOZ9bd5t9mr8KLaVBYTCJo5ERE8jymab5dPqe5qKfJsCZiqWglbjUo9twIDAQABo1AwTjAdBgNVHQ4EFgQUxpuwcs/CYQOyui+r1G+3KxBNhxkwHwYDVR0jBBgwFoAUxpuwcs/CYQOyui+r1G+3KxBNhxkwDAYDVR0TBAUwAwEB/zANBgkqhkiG9w0BAQsFAAOCAQEAAiWUKs/2x/viNCKi3Y6blEuCtAGhzOOZ9EjrvJ8+COH3Rag3tVBWrcBZ3/uhhPq5gy9lqw4OkvEws99/5jFsX1FJ6MKBgqfuy7yh5s1YfM0ANHYczMmYpZeAcQf2CGAaVfwTTfSlzNLsF2lW/ly7yapFzlYSJLGoVE+OHEu8g5SlNACUEfkXw+5Eghh+KzlIN7R6Q7r2ixWNFBC/jWf7NKUfJyX8qIG5md1YUeT6GBW9Bm2/1/RiO24JTaYlfLdKK9TYb8sG5B+OLab2DImG99CJ25RkAcSobWNF5zD0O6lgOo3cEdB/ksCq3hmtlC/DlLZ/D8CJ+7VuZnS1rR2naQ==</ds:X509Certificate>
        </ds:X509Data>
      </ds:KeyInfo>
    </md:KeyDescriptor>
    <md:KeyDescriptor use="encryption">
      <ds:KeyInfo xmlns:ds="http://www.w3.org/2000/09/xmldsig#">
        <ds:X509Data>
          <ds:X509Certificate>MIIDXTCCAkWgAwIBAgIJALmVVuDWu4NYMA0GCSqGSIb3DQEBCwUAMEUxCzAJBgNVBAYTAkFVMRMwEQYDVQQIDApTb21lLVN0YXRlMSEwHwYDVQQKDBhJbnRlcm5ldCBXaWRnaXRzIFB0eSBMdGQwHhcNMTYxMjMxMTQzNDQ3WhcNNDgwNjI1MTQzNDQ3WjBFMQswCQYDVQQGEwJBVTETMBEGA1UECAwKU29tZS1TdGF0ZTEhMB8GA1UECgwYSW50ZXJuZXQgV2lkZ2l0cyBQdHkgTHRkMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAzUCFozgNb1h1M0jzNRSCjhOBnR+uVbVpaWfXYIR+AhWDdEe5ryY+CgavOg8bfLybyzFdehlYdDRgkedEB/GjG8aJw06l0qF4jDOAw0kEygWCu2mcH7XOxRt+YAH3TVHa/Hu1W3WjzkobqqqLQ8gkKWWM27fOgAZ6GieaJBN6VBSMMcPey3HWLBmc+TYJmv1dbaO2jHhKh8pfKw0W12VM8P1PIO8gv4Phu/uuJYieBWKixBEyy0lHjyixYFCR12xdh4CA47q958ZRGnnDUGFVE1QhgRacJCOZ9bd5t9mr8KLaVBYTCJo5ERE8jymab5dPqe5qKfJsCZiqWglbjUo9twIDAQABo1AwTjAdBgNVHQ4EFgQUxpuwcs/CYQOyui+r1G+3KxBNhxkwHwYDVR0jBBgwFoAUxpuwcs/CYQOyui+r1G+3KxBNhxkwDAYDVR0TBAUwAwEB/zANBgkqhkiG9w0BAQsFAAOCAQEAAiWUKs/2x/viNCKi3Y6blEuCtAGhzOOZ9EjrvJ8+COH3Rag3tVBWrcBZ3/uhhPq5gy9lqw4OkvEws99/5jFsX1FJ6MKBgqfuy7yh5s1YfM0ANHYczMmYpZeAcQf2CGAaVfwTTfSlzNLsF2lW/ly7yapFzlYSJLGoVE+OHEu8g5SlNACUEfkXw+5Eghh+KzlIN7R6Q7r2ixWNFBC/jWf7NKUfJyX8qIG5md1YUeT6GBW9Bm2/1/RiO24JTaYlfLdKK9TYb8sG5B+OLab2DImG99CJ25RkAcSobWNF5zD0O6lgOo3cEdB/ksCq3hmtlC/DlLZ/D8CJ+7VuZnS1rR2naQ==</ds:X509Certificate>
        </ds:X509Data>
      </ds:KeyInfo>
    </md:KeyDescriptor>
    <md:SingleLogoutService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect" Location="http://localhost:9080/simplesaml/saml2/idp/SingleLogoutService.php"/>
    <md:NameIDFormat>urn:oasis:names:tc:SAML:2.0:nameid-format:transient</md:NameIDFormat>
    <md:SingleSignOnService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect" Location="http://localhost:9080/simplesaml/saml2/idp/SSOService.php"/>
  </md:IDPSSODescriptor>
</md:EntityDescriptor>`))
	require.NoError(t, err)
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(fmt.Sprintf(`{
		"server_settings": {
			"server_url": "https://localhost:8080"
		},
		"sso_settings": {
			"enable_sso": true,
			"entity_id": "https://localhost:8080",
			"idp_name": "SimpleSAML",
			"metadata": %s
		}
	}`, metadata)), http.StatusOK, &acResp)
	require.NotNil(t, acResp)

	// Create sso_user2@example.com if it doesn't exist (because this is
	// a free instance and doesn't support enable_jit_provisioning).
	u := &fleet.User{
		Name:       "SSO User 2",
		Email:      "sso_user2@example.com",
		GlobalRole: ptr.String(fleet.RoleObserver),
		SSOEnabled: true,
	}
	password := test.GoodPassword
	require.NoError(t, u.SetPassword(password, 10, 10))
	_, _ = s.ds.NewUser(context.Background(), u)

	body := s.LoginSSOUser("sso_user2", "user123#")
	require.Contains(t, body, "Redirecting to Fleet at  ...")
}

// This test increases coverage on using server_url.hostname as
// entity_id when not set, and audience validation errors.
func (s *integrationSSOTestSuite) TestSSOLoginNoEntityID() {
	t := s.T()

	acResp := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
        "server_settings": {
          "server_url": "https://localhost:8080"
        },
		"sso_settings": {
			"enable_sso": true,
			"entity_id": "localhost",
			"idp_name": "SimpleSAML",
			"metadata_url": "http://localhost:9080/simplesaml/saml2/idp/metadata.php"
		}
	}`), http.StatusOK, &acResp)
	require.NotNil(t, acResp)

	ac, err := s.ds.AppConfig(context.Background())
	require.NoError(t, err)
	ac.SSOSettings.EntityID = ""
	err = s.ds.SaveAppConfig(context.Background(), ac)
	require.NoError(t, err)

	// Create sso_user2@example.com if it doesn't exist (because this is
	// a free instance and doesn't support enable_jit_provisioning).
	u := &fleet.User{
		Name:       "SSO User 2",
		Email:      "sso_user2@example.com",
		GlobalRole: ptr.String(fleet.RoleObserver),
		SSOEnabled: true,
	}
	password := test.GoodPassword
	require.NoError(t, u.SetPassword(password, 10, 10))
	_, _ = s.ds.NewUser(context.Background(), u)

	body := s.LoginSSOUser("sso_user2", "user123#")
	// Fails due to `audience restriction validation failed: wrong audience: [{Audience:{Value:localhost}}]`
	require.Contains(t, body, "/login?status=error")
}

// This test increases coverage to test failure on ParseXMLResponse.
func (s *integrationSSOTestSuite) TestSSOLoginSAMLResponseTampered() {
	t := s.T()

	if _, ok := os.LookupEnv("SAML_IDP_TEST"); !ok {
		t.Skip("SSO tests are disabled")
	}

	acResp := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
        "server_settings": {
          "server_url": "https://localhost:8080"
        },
		"sso_settings": {
			"enable_sso": true,
			"entity_id": "sso.test.com",
			"idp_name": "SimpleSAML",
			"metadata_url": "http://localhost:9080/simplesaml/saml2/idp/metadata.php"
		}
	}`), http.StatusOK, &acResp)
	require.NotNil(t, acResp)

	// Create sso_user2@example.com if it doesn't exist (because this is
	// a free instance and doesn't support enable_jit_provisioning).
	u := &fleet.User{
		Name:       "SSO User 2",
		Email:      "sso_user2@example.com",
		GlobalRole: ptr.String(fleet.RoleObserver),
		SSOEnabled: true,
	}
	password := test.GoodPassword
	require.NoError(t, u.SetPassword(password, 10, 10))
	_, _ = s.ds.NewUser(context.Background(), u)

	var (
		idpUsername = "sso_user2"
		idpPassword = "user123#"
	)
	var resIni initiateSSOResponse
	s.DoJSON("POST", "/api/v1/fleet/sso", map[string]string{}, http.StatusOK, &resIni)
	jar, err := cookiejar.New(nil)
	require.NoError(t, err)
	client := fleethttp.NewClient(
		fleethttp.WithFollowRedir(false),
		fleethttp.WithCookieJar(jar),
	)
	resp, err := client.Get(resIni.URL)
	require.NoError(t, err)
	// From the redirect Location header we can get the AuthState and the URL to
	// which we submit the credentials
	parsed, err := url.Parse(resp.Header.Get("Location"))
	require.NoError(t, err)
	data := url.Values{
		"username":  {idpUsername},
		"password":  {idpPassword},
		"AuthState": {parsed.Query().Get("AuthState")},
	}
	resp, err = client.PostForm(parsed.Scheme+"://"+parsed.Host+parsed.Path, data)
	require.NoError(t, err)
	// The response is an HTML form, we can extract the base64-encoded response
	// to submit to the Fleet server from here
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	re := regexp.MustCompile(`name="SAMLResponse" value="([^\s]*)" />`)
	matches := re.FindSubmatch(body)
	require.NotEmptyf(t, matches, "callback HTML doesn't contain a SAMLResponse value, got body: %s", body)
	samlResponse := string(matches[1])
	samlResponseDecoded, err := base64.RawStdEncoding.DecodeString(samlResponse)
	require.NoError(t, err)

	tamperedSAMLResponseDecoded := strings.ReplaceAll(string(samlResponseDecoded), idpUsername, "sso_us3r2")
	tampteredSAMLResponseEncoded := base64.RawStdEncoding.EncodeToString([]byte(tamperedSAMLResponseDecoded))

	ssoURL := fmt.Sprintf(
		"/api/v1/fleet/sso/callback?SAMLResponse=%s",
		url.QueryEscape(tampteredSAMLResponseEncoded),
	)
	resp = s.DoRawNoAuth("POST", ssoURL, nil, http.StatusOK)

	t.Cleanup(func() {
		resp.Body.Close()
	})
	body, err = io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Contains(t, string(body), "/login?status=error")
}

func (s *integrationSSOTestSuite) TestSSOURL() {
	t := s.T()

	// Use the test metadata instead of trying to fetch from localhost:9080
	testMetadata := `
<md:EntityDescriptor xmlns:md="urn:oasis:names:tc:SAML:2.0:metadata" xmlns:ds="http://www.w3.org/2000/09/xmldsig#" entityID="http://localhost:9080/simplesaml/saml2/idp/metadata.php">
  <md:IDPSSODescriptor protocolSupportEnumeration="urn:oasis:names:tc:SAML:2.0:protocol">
    <md:KeyDescriptor use="signing">
      <ds:KeyInfo xmlns:ds="http://www.w3.org/2000/09/xmldsig#">
        <ds:X509Data>
          <ds:X509Certificate>MIIDXTCCAkWgAwIBAgIJALmVVuDWu4NYMA0GCSqGSIb3DQEBCwUAMEUxCzAJBgNVBAYTAkFVMRMwEQYDVQQIDApTb21lLVN0YXRlMSEwHwYDVQQKDBhJbnRlcm5ldCBXaWRnaXRzIFB0eSBMdGQwHhcNMTYxMjMxMTQzNDQ3WhcNNDgwNjI1MTQzNDQ3WjBFMQswCQYDVQQGEwJBVTETMBEGA1UECAwKU29tZS1TdGF0ZTEhMB8GA1UECgwYSW50ZXJuZXQgV2lkZ2l0cyBQdHkgTHRkMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAzUCFozgNb1h1M0jzNRSCjhOBnR+uVbVpaWfXYIR+AhWDdEe5ryY+CgavOg8bfLybyzFdehlYdDRgkedEB/GjG8aJw06l0qF4jDOAw0kEygWCu2mcH7XOxRt+YAH3TVHa/Hu1W3WjzkobqqqLQ8gkKWWM27fOgAZ6GieaJBN6VBSMMcPey3HWLBmc+TYJmv1dbaO2jHhKh8pfKw0W12VM8P1PIO8gv4Phu/uuJYieBWKixBEyy0lHjyixYFCR12xdh4CA47q958ZRGnnDUGFVE1QhgRacJCOZ9bd5t9mr8KLaVBYTCJo5ERE8jymab5dPqe5qKfJsCZiqWglbjUo9twIDAQABo1AwTjAdBgNVHQ4EFgQUxpuwcs/CYQOyui+r1G+3KxBNhxkwHwYDVR0jBBgwFoAUxpuwcs/CYQOyui+r1G+3KxBNhxkwDAYDVR0TBAUwAwEB/zANBgkqhkiG9w0BAQsFAAOCAQEAAiWUKs/2x/viNCKi3Y6blEuCtAGhzOOZ9EjrvJ8+COH3Rag3tVBWrcBZ3/uhhPq5gy9lqw4OkvEws99/5jFsX1FJ6MKBgqfuy7yh5s1YfM0ANHYczMmYpZeAcQf2CGAaVfwTTfSlzNLsF2lW/ly7yapFzlYSJLGoVE+OHEu8g5SlNACUEfkXw+5Eghh+KzlIN7R6Q7r2ixWNFBC/jWf7NKUfJyX8qIG5md1YUeT6GBW9Bm2/1/RiO24JTaYlfLdKK9TYb8sG5B+OLab2DImG99CJ25RkAcSobWNF5zD0O6lgOo3cEdB/ksCq3hmtlC/DlLZ/D8CJ+7VuZnS1rR2naQ==</ds:X509Certificate>
        </ds:X509Data>
      </ds:KeyInfo>
    </md:KeyDescriptor>
    <md:SingleLogoutService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect" Location="http://localhost:9080/simplesaml/saml2/idp/SingleLogoutService.php"/>
    <md:NameIDFormat>urn:oasis:names:tc:SAML:2.0:nameid-format:transient</md:NameIDFormat>
    <md:SingleSignOnService Binding="urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect" Location="http://localhost:9080/simplesaml/saml2/idp/SSOService.php"/>
  </md:IDPSSODescriptor>
</md:EntityDescriptor>`

	// Configure SSO with a specific SSO URL and inline metadata
	acResp := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(fmt.Sprintf(`{
		"sso_settings": {
			"enable_sso": true,
			"entity_id": "https://localhost:8080",
			"idp_name": "SimpleSAML",
			"metadata": %q,
			"enable_jit_provisioning": false,
			"sso_url": "https://admin.localhost:8080"
		}
	}`, testMetadata)), http.StatusOK, &acResp)
	require.NotNil(t, acResp)

	// Verify the SSO URL is set
	require.NotNil(t, acResp.SSOSettings)
	require.Equal(t, "https://admin.localhost:8080", acResp.SSOSettings.SSOURL)

	// Initiate SSO
	var resIni initiateSSOResponse
	s.DoJSON("POST", "/api/v1/fleet/sso", map[string]string{}, http.StatusOK, &resIni)
	require.NotEmpty(t, resIni.URL)

	// Parse the auth request to verify it uses the SSO URL
	parsed, err := url.Parse(resIni.URL)
	require.NoError(t, err)
	q := parsed.Query()
	encoded := q.Get("SAMLRequest")
	assert.NotEmpty(t, encoded)
	authReq := inflate(t, encoded)

	// Check that the ACS URL in the auth request uses the SSO URL
	require.NotNil(t, authReq.AssertionConsumerServiceURL)
	assert.Equal(t, "https://admin.localhost:8080/api/v1/fleet/sso/callback", authReq.AssertionConsumerServiceURL)
}
