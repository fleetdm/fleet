package android

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	shared_mdm "github.com/fleetdm/fleet/v4/pkg/mdm"
	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/fleetdm/fleet/v4/server/service/middleware/endpoint_utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/androidmanagement/v1"
)

func TestAndroid(t *testing.T) {
	s := SetUpSuite(t, "integrationtest.Android")

	cases := []struct {
		name string
		fn   func(t *testing.T, s *Suite)
	}{
		{"HappyPath", testHappyPath},
		{"CreateEnrollmentToken", testCreateEnrollmentToken},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer mysql.TruncateTables(t, s.DS)
			c.fn(t, s)
		})
	}
}

func testHappyPath(t *testing.T, s *Suite) {
	signupDetails := expectSignupDetails(t, s)
	var signupURL android.EnterpriseSignupResponse
	s.DoJSON(t, "GET", "/api/v1/fleet/android_enterprise/signup_url", nil, http.StatusOK, &signupURL)
	assert.Equal(t, signupURL.Url, signupDetails.Url)
}

type enrollmentTokenRequest struct {
	EnrollSecret string
	IdpUUID      string
}

func testCreateEnrollmentToken(t *testing.T, s *Suite) {
	appCfg := &fleet.AppConfig{
		MDM: fleet.MDM{
			AndroidEnabledAndConfigured: true,
		},
		ServerSettings: fleet.ServerSettings{
			ServerURL: "http://localhost",
		},
	}
	enableAndroidMDM := func() {
		_, err := s.DS.NewAppConfig(t.Context(), appCfg)
		require.NoError(t, err)
	}

	createTeamAndSecret := func(name, secret string, enableEndUserAuth bool) {
		team, err := s.DS.NewTeam(t.Context(), &fleet.Team{
			Name: name,
			Config: fleet.TeamConfig{
				MDM: fleet.TeamMDM{
					MacOSSetup: fleet.MacOSSetup{
						EnableEndUserAuthentication: enableEndUserAuth,
					},
				},
			},
		})
		require.NoError(t, err)
		err = s.DS.ApplyEnrollSecrets(t.Context(), &team.ID, []*fleet.EnrollSecret{
			{
				Secret: secret,
				TeamID: &team.ID,
			},
		})
		require.NoError(t, err)
	}

	setupAndroidEnterprise := func() {
		admin := s.Users["admin1"]
		enterpriseID, err := s.DS.CreateEnterprise(t.Context(), admin.ID)
		require.NoError(t, err)

		// signupToken is used to authenticate the signup callback URL -- to ensure that the callback came from our Android enterprise signup flow
		signupToken, err := server.GenerateRandomURLSafeText(32)
		require.NoError(t, err)

		callbackURL := fmt.Sprintf("%s/api/v1/fleet/android_enterprise/connect/%s", appCfg.ServerSettings.ServerURL, signupToken)
		signupDetails := android.SignupDetails{
			Name: "test",
			Url:  callbackURL,
		}

		err = s.DS.UpdateEnterprise(t.Context(), &android.EnterpriseDetails{
			Enterprise: android.Enterprise{
				ID:           enterpriseID,
				EnterpriseID: "test",
			},
			SignupName:  signupDetails.Name,
			SignupToken: signupToken,
		})
		require.NoError(t, err)
	}

	s.AndroidProxy.EnterprisesEnrollmentTokensCreateFunc = func(ctx context.Context, enterpriseName string, token *androidmanagement.EnrollmentToken) (*androidmanagement.EnrollmentToken, error) {
		// For ease of testing and validating, we base64 the json input as the output value

		jsonString, err := json.Marshal(token)
		require.NoError(t, err)
		base64Encoded := base64.StdEncoding.EncodeToString(jsonString)

		return &androidmanagement.EnrollmentToken{
			Value: base64Encoded,
		}, nil
	}

	t.Run("fails", func(t *testing.T) {
		t.Run("if enroll_secret query param is missing", func(t *testing.T) {
			s.Do(t, "GET", "/api/v1/fleet/android_enterprise/enrollment_token", nil, http.StatusBadRequest)
		})

		t.Run("if android MDM is not configured", func(t *testing.T) {
			s.Do(t, "GET", "/api/v1/fleet/android_enterprise/enrollment_token", nil, http.StatusConflict, "enroll_secret", "secret")
		})

		t.Run("if enroll secret is invalid", func(t *testing.T) {
			enableAndroidMDM()
			s.Do(t, "GET", "/api/v1/fleet/android_enterprise/enrollment_token", nil, http.StatusUnauthorized, "enroll_secret", "secret")
		})

		t.Run("if android enterprise is missing", func(t *testing.T) {
			enableAndroidMDM()
			secret := "global-enterprise-missing"
			createTeamAndSecret(secret, secret, false)
			resp := s.Do(t, "GET", "/api/v1/fleet/android_enterprise/enrollment_token", nil, http.StatusNotFound, "enroll_secret", secret)
			je := decodeJsonError(t, resp)

			require.Contains(t, "Android enterprise", je.Errors[0]["base"])
			mysql.TruncateTables(t, s.DS)
		})

		t.Run("if idp account does not exist", func(t *testing.T) {
			enableAndroidMDM()
			secret := "global-no-idp-account" // nolint: gosec
			createTeamAndSecret(secret, secret, false)
			resp := s.DoRawWithHeaders(t, "GET", "/api/v1/fleet/android_enterprise/enrollment_token", nil, http.StatusUnprocessableEntity, map[string]string{
				"Cookie": fmt.Sprintf("%s=%s", shared_mdm.BYODIdpCookieName, "test-uuid"),
			}, "enroll_secret", secret)
			je := decodeJsonError(t, resp)

			require.Contains(t, "validating idp account existence", je.Errors[0]["base"])
			mysql.TruncateTables(t, s.DS)
		})

		t.Run("if idp is required but not set", func(t *testing.T) {
			enableAndroidMDM()
			secret := "team"
			createTeamAndSecret("team", secret, true)
			s.DoRaw(t, "GET", "/api/v1/fleet/android_enterprise/enrollment_token", nil, http.StatusUnauthorized, "enroll_secret", secret)
		})

		t.Cleanup(func() {
			mysql.TruncateTables(t, s.DS)
		})
	})

	t.Run("succeeds", func(t *testing.T) {
		globalSecret := "global"

		t.Run("when enroll secret is passed", func(t *testing.T) {
			enableAndroidMDM()
			createTeamAndSecret(globalSecret, globalSecret, false)
			setupAndroidEnterprise()

			var resp android.EnrollmentTokenResponse
			s.DoJSON(t, "GET", "/api/v1/fleet/android_enterprise/enrollment_token", nil, http.StatusOK, &resp, "enroll_secret", globalSecret)

			decoded, err := base64.StdEncoding.DecodeString(resp.EnrollmentToken.EnrollmentToken)
			require.NoError(t, err)
			var et androidmanagement.EnrollmentToken
			err = json.Unmarshal(decoded, &et)
			require.NoError(t, err)

			var enrollmentRequest enrollmentTokenRequest
			err = json.Unmarshal([]byte(et.AdditionalData), &enrollmentRequest)
			require.NoError(t, err)

			require.Equal(t, globalSecret, enrollmentRequest.EnrollSecret)
			require.Equal(t, "", enrollmentRequest.IdpUUID)

			t.Cleanup(func() {
				mysql.TruncateTables(t, s.DS)
			})
		})

		t.Run("when enroll and idp uuid is set", func(t *testing.T) {
			enableAndroidMDM()
			createTeamAndSecret(globalSecret, globalSecret, true)
			setupAndroidEnterprise()
			idpEmail := "test@local.com"
			err := s.DS.InsertMDMIdPAccount(t.Context(), &fleet.MDMIdPAccount{
				Username: "test",
				Email:    idpEmail,
			})
			require.NoError(t, err)
			idpAccount, err := s.DS.GetMDMIdPAccountByEmail(t.Context(), idpEmail)
			require.NoError(t, err)

			resp := s.DoRawWithHeaders(t, "GET", "/api/v1/fleet/android_enterprise/enrollment_token", nil, http.StatusOK, map[string]string{
				"Cookie": fmt.Sprintf("%s=%s", shared_mdm.BYODIdpCookieName, idpAccount.UUID),
			}, "enroll_secret", globalSecret)

			bodyBytes, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			var etr android.EnrollmentTokenResponse
			err = json.Unmarshal(bodyBytes, &etr)
			require.NoError(t, err)

			decoded, err := base64.StdEncoding.DecodeString(etr.EnrollmentToken.EnrollmentToken)
			require.NoError(t, err)
			var et androidmanagement.EnrollmentToken
			err = json.Unmarshal(decoded, &et)
			require.NoError(t, err)

			var enrollmentRequest enrollmentTokenRequest
			err = json.Unmarshal([]byte(et.AdditionalData), &enrollmentRequest)
			require.NoError(t, err)

			require.Equal(t, globalSecret, enrollmentRequest.EnrollSecret)
			require.Equal(t, idpAccount.UUID, enrollmentRequest.IdpUUID)

			t.Cleanup(func() {
				mysql.TruncateTables(t, s.DS)
			})
		})
	})
}

func expectSignupDetails(t *testing.T, s *Suite) *android.SignupDetails {
	signupDetails := &android.SignupDetails{
		Url:  "URL",
		Name: "Name",
	}
	s.AndroidProxy.SignupURLsCreateFunc = func(_ context.Context, serverURL, callbackURL string) (*android.SignupDetails, error) {
		assert.Equal(t, s.Server.URL, serverURL)
		// We will need to extract the security token from the callbackURL for further testing
		assert.Contains(t, callbackURL, "/api/v1/fleet/android_enterprise/connect/")
		return signupDetails, nil
	}
	return signupDetails
}

func decodeJsonError(t *testing.T, response *http.Response) endpoint_utils.JsonError {
	bodyBytes, err := io.ReadAll(response.Body)
	require.NoError(t, err)

	var je endpoint_utils.JsonError
	err = json.Unmarshal(bodyBytes, &je)
	require.NoError(t, err)

	return je
}
