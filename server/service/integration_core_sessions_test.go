package service

// Session and credential tests for the core (no-license) suite.
//
// Belongs here: session info and deletion, login attempt logging, changing and
// resetting a password, and login behaviour when SSO is disabled.
//
// Does not belong here: creating or modifying the user record itself
// (integration_core_users_test.go).

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"testing"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *integrationTestSuite) TestSessionInfo() {
	t := s.T()

	ssn := createSession(t, 1, s.ds)

	var meResp getUserResponse
	resp := s.DoRawWithHeaders("GET", "/api/latest/fleet/me", nil, http.StatusOK, map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", ssn.Key),
	})
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&meResp))
	assert.Equal(t, uint(1), meResp.User.ID)

	// get info about session
	var getResp getInfoAboutSessionResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/sessions/%d", ssn.ID), nil, http.StatusOK, &getResp)
	assert.Equal(t, ssn.ID, getResp.SessionID)
	assert.Equal(t, uint(1), getResp.UserID)

	// get info about session
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/sessions/%d", ssn.ID+1), nil, http.StatusNotFound, &getResp)

	// delete session
	var delResp deleteSessionResponse
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/sessions/%d", ssn.ID), nil, http.StatusOK, &delResp)

	// delete session - non-existing
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/sessions/%d", ssn.ID), nil, http.StatusNotFound, &delResp) // nolint:nilaway // createSession fails the test via require internally on error, cannot be nil here
}

func (s *integrationTestSuite) TestLogLoginAttempts() {
	t := s.T()

	// create a new user
	var createResp createUserResponse
	params := fleet.UserPayload{
		Name:       new("foobar"),
		Email:      new("foobar@example.com"),
		Password:   new(test.GoodPassword),
		GlobalRole: new(fleet.RoleObserver),
	}
	s.DoJSON("POST", "/api/latest/fleet/users/admin", params, http.StatusOK, &createResp)
	require.NotZero(t, createResp.User.ID)
	u := *createResp.User

	// Register current number of activities.
	activitiesResp := listActivitiesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &activitiesResp)
	require.NoError(t, activitiesResp.Err)
	oldActivitiesCount := len(activitiesResp.Activities)

	// Login with invalid passwordm, should fail.
	res := s.DoRawNoAuth(
		"POST", "/api/latest/fleet/login",
		jsonMustMarshal(t, fleet.LoginRequest{Email: u.Email, Password: test.GoodPassword2}),
		http.StatusUnauthorized,
	)
	res.Body.Close()

	// A new activity item for the failed login attempt is created.
	activitiesResp = listActivitiesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &activitiesResp)
	require.NoError(t, activitiesResp.Err)
	require.Len(t, activitiesResp.Activities, oldActivitiesCount+1)
	sort.Slice(activitiesResp.Activities, func(i, j int) bool {
		return activitiesResp.Activities[i].ID < activitiesResp.Activities[j].ID
	})
	activity := activitiesResp.Activities[len(activitiesResp.Activities)-1]
	require.Equal(t, activity.Type, fleet.ActivityTypeUserFailedLogin{}.ActivityName())
	require.NotNil(t, activity.Details)
	actDetails := fleet.ActivityTypeUserFailedLogin{}
	err := json.Unmarshal(*activity.Details, &actDetails)
	require.NoError(t, err)
	require.Equal(t, "foobar@example.com", actDetails.Email)

	// login with good password, should succeed
	res = s.DoRawNoAuth(
		"POST", "/api/latest/fleet/login",
		jsonMustMarshal(t, fleet.LoginRequest{
			Email:    u.Email,
			Password: test.GoodPassword,
		}), http.StatusOK,
	)
	res.Body.Close()

	// A new activity item for the successful login is created.
	activitiesResp = listActivitiesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &activitiesResp)
	require.NoError(t, activitiesResp.Err)
	require.Len(t, activitiesResp.Activities, oldActivitiesCount+2)
	sort.Slice(activitiesResp.Activities, func(i, j int) bool {
		return activitiesResp.Activities[i].ID < activitiesResp.Activities[j].ID
	})
	activity = activitiesResp.Activities[len(activitiesResp.Activities)-1]
	require.Equal(t, activity.Type, fleet.ActivityTypeUserLoggedIn{}.ActivityName())
	require.NotNil(t, activity.Details)
	err = json.Unmarshal(*activity.Details, &fleet.ActivityTypeUserLoggedIn{})
	require.NoError(t, err)
}

func (s *integrationTestSuite) TestChangePassword() {
	t := s.T()

	endpoint := "/api/latest/fleet/change_password"
	// also the default password for the default logged in admin user
	startPwd := test.GoodPassword

	testCases := []struct {
		oldPw          string
		newPw          string
		expectedStatus int
	}{
		// valid changes – 12-48 characters, with at least 1 number (e.g. 0 - 9) and 1 symbol (e.g. &*#).
		{startPwd, "password123$", http.StatusOK},
		{"password123$", "Password$321", http.StatusOK},

		// invalid changes
		// empty old
		{"", "PassworD$321", http.StatusUnprocessableEntity},
		// empty new
		{"password123$", "", http.StatusUnprocessableEntity},
		// too short
		{"password123$", "Password$21", http.StatusUnprocessableEntity},
		// too long
		{"password123$", "Password$321Password$321Password$321Password$321Password$321", http.StatusUnprocessableEntity},
		// no numbers
		{"password123$", "Password$!@#", http.StatusUnprocessableEntity},
		// no symbols
		{"password123$", "Password4321", http.StatusUnprocessableEntity},
		// new pw is same as old
		{"password123$", "password123$", http.StatusUnprocessableEntity},
		// wrong old pw
		{"passgord123$", "Password$321", http.StatusUnprocessableEntity},

		// change back to original password for cleanup
		{"Password$321", startPwd, http.StatusOK},
	}

	runTestCases := func(name, userEmail string) {
		currentPwd := startPwd
		for _, tc := range testCases {
			t.Run(name, func(t *testing.T) {
				var changePwResp changePasswordResponse
				s.DoJSON("POST", endpoint, changePasswordRequest{OldPassword: tc.oldPw, NewPassword: tc.newPw}, tc.expectedStatus, &changePwResp)
				// After a successful password change, the session is invalidated, so we need to re-authenticate
				if tc.expectedStatus == http.StatusOK {
					currentPwd = tc.newPw
					s.token = s.getTestToken(userEmail, currentPwd)
				}
			})
		}
	}

	adminEmail := "admin1@example.com"
	// Clear the cached admin token so it will be refreshed after password changes
	s.cachedAdminToken = ""
	runTestCases("test change passwords as admin", adminEmail)

	// create a new user
	testUserEmail := "changepwd@example.com"
	var createResp createUserResponse
	params := fleet.UserPayload{
		Name:                     new("Test Change Password"),
		Email:                    new(testUserEmail),
		Password:                 new(startPwd),
		GlobalRole:               new(fleet.RoleObserver),
		AdminForcedPasswordReset: new(false),
	}
	s.DoJSON("POST", "/api/latest/fleet/users/admin", params, http.StatusOK, &createResp)
	require.NotZero(t, createResp.User.ID)

	// Clear the cached admin token again and schedule cleanup to restore it
	s.cachedAdminToken = ""
	t.Cleanup(func() { s.token = s.getTestAdminToken() })

	// login and run the change password tests as the user
	s.token = s.getTestToken(testUserEmail, startPwd)
	runTestCases("test change passwords as user", testUserEmail)
}

func (s *integrationTestSuite) TestPasswordReset() {
	t := s.T()

	// Clear any previous usage of forgot_password in the test suite to start from scatch.
	clearKeys := func() {
		clearRedisKey(t, s.redisPool, "ratelimit::forgot_password")
	}
	clearKeys()
	s.T().Cleanup(func() {
		clearKeys()
	})

	// create a new user
	var createResp createUserResponse
	userRawPwd := test.GoodPassword
	params := fleet.UserPayload{
		Name:       new("forgotpwd"),
		Email:      new("forgotpwd@example.com"),
		Password:   new(userRawPwd),
		GlobalRole: new(fleet.RoleObserver),
	}
	s.DoJSON("POST", "/api/latest/fleet/users/admin", params, http.StatusOK, &createResp)
	require.NotZero(t, createResp.User.ID)
	u := *createResp.User

	// Request password reset when SMTP/SES is not configured
	res := s.DoRawNoAuth("POST", "/api/latest/fleet/forgot_password", jsonMustMarshal(t, forgotPasswordRequest{Email: "invalid@asd.com"}), http.StatusInternalServerError)
	res.Body.Close()

	// Configure SMTP
	var configResp appConfigResponse
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage("{\"smtp_settings\":{\"enable_smtp\":true,\"sender_address\":\"user@example.com\",\"server\":\"127.0.0.1\",\"port\":1025,\"authentication_type\":\"authtype_none\"}}"), http.StatusOK, &configResp)

	// request forgot password, invalid email
	res = s.DoRawNoAuth("POST", "/api/latest/fleet/forgot_password", jsonMustMarshal(t, forgotPasswordRequest{Email: "invalid@asd.com"}), http.StatusAccepted)
	res.Body.Close()

	// TODO: tested manually (adds too much time to the test), works but hitting the rate
	// limit returns 500 instead of 429, see #4406. We get the authz check missing error instead.
	// // trigger the rate limit with a batch of requests in a short burst
	// for i := 0; i < 20; i++ {
	//	s.DoJSON("POST", "/api/latest/fleet/forgot_password", forgotPasswordRequest{Email: "invalid@asd.com"}, http.StatusAccepted, &forgotResp)
	// }

	// request forgot password, valid email
	res = s.DoRawNoAuth("POST", "/api/latest/fleet/forgot_password", jsonMustMarshal(t, forgotPasswordRequest{Email: u.Email}), http.StatusAccepted)
	res.Body.Close()

	var token string
	mysqltest.ExecAdhocSQL(t, s.ds, func(db sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), db, &token, "SELECT token FROM password_reset_requests WHERE user_id = ?", u.ID)
	})

	// proceed with reset password
	userNewPwd := test.GoodPassword2
	res = s.DoRawNoAuth("POST", "/api/latest/fleet/reset_password", jsonMustMarshal(t, resetPasswordRequest{PasswordResetToken: token, NewPassword: userNewPwd}), http.StatusOK)
	res.Body.Close()

	// attempt it again with already-used token
	userUnusedPwd := "unusedpassw0rd!"
	res = s.DoRawNoAuth("POST", "/api/latest/fleet/reset_password", jsonMustMarshal(t, resetPasswordRequest{PasswordResetToken: token, NewPassword: userUnusedPwd}), http.StatusUnauthorized)
	res.Body.Close()

	// login with the old password, should not succeed
	res = s.DoRawNoAuth("POST", "/api/latest/fleet/login", jsonMustMarshal(t, fleet.LoginRequest{Email: u.Email, Password: userRawPwd}),
		http.StatusUnauthorized)
	res.Body.Close()

	// login with the new password, should succeed
	res = s.DoRawNoAuth("POST", "/api/latest/fleet/login", jsonMustMarshal(t, fleet.LoginRequest{Email: u.Email, Password: userNewPwd}),
		http.StatusOK)
	res.Body.Close()
}

func (s *integrationTestSuite) TestSSODisabled() {
	t := s.T()

	s.DoRawNoAuth("POST", "/api/v1/fleet/sso", nil, http.StatusBadRequest)

	// callback without SAML response
	s.DoRawNoAuth("POST", "/api/v1/fleet/sso/callback", nil, http.StatusBadRequest)

	// callback with invalid SAML response
	s.DoRawNoAuth("POST", "/api/v1/fleet/sso/callback?SAMLResponse=zz", nil, http.StatusBadRequest)

	// callback with valid SAML response
	validResponse := `<?xml version="1.0" encoding="UTF-8"?>
<samlp:Response xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol" Destination="https://localhost:8080/api/v1/fleet/sso/callback" ID="_52f2515c5319f2adf3f072d9d9f2b6881493305396746" InResponseTo="4982b430-73e1-4ad2-885a-4a775a91f820" IssueInstant="2017-04-27T15:03:16.747Z" Version="2.0">
</samlp:Response>`
	samlResponse := base64.StdEncoding.EncodeToString([]byte(validResponse))
	res := s.DoRawNoAuth("POST", "/api/v1/fleet/sso/callback?SAMLResponse="+url.QueryEscape(samlResponse), nil, http.StatusOK)
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	require.Contains(t, string(body), "/login?status=org_disabled") // html contains a script that redirects to this path
}
