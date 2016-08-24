package server

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/kolide/kolide-ose/kolide"
	"github.com/stretchr/testify/assert"
)

func TestGetUser(t *testing.T) {
	// create the test datastore and server
	ds := createTestUsers(t, createTestDatastore(t))
	server := createTestServer(ds)

	////////////////////////////////////////////////////////////////////////////
	// log-in with a user
	////////////////////////////////////////////////////////////////////////////

	// log in with admin test user
	response := makeRequest(
		t,
		server,
		"POST",
		"/api/v1/kolide/login",
		CreateUserRequestBody{
			Username: "user1",
			Password: "foobar",
		},
		"",
	)
	assert.Equal(t, http.StatusOK, response.Code)

	// ensure that a non-empty cookie was in-fact set
	userCookie := response.Header().Get("Set-Cookie")
	assert.NotEmpty(t, userCookie)

	////////////////////////////////////////////////////////////////////////////
	// get the info of user2 from user1's account
	////////////////////////////////////////////////////////////////////////////

	user2, err := ds.User("user2")
	assert.Nil(t, err)

	response = makeRequest(
		t,
		server,
		"POST",
		"/api/v1/kolide/user",
		GetUserRequestBody{
			ID: user2.ID,
		},
		userCookie,
	)
	assert.Equal(t, http.StatusOK, response.Code)

	var user2Info GetUserResponseBody
	err = json.NewDecoder(response.Body).Decode(&user2Info)
	assert.Nil(t, err)

	assert.True(t, user2Info.NeedsPasswordReset)
}

func TestCreateUser(t *testing.T) {
	// create the test datastore and server
	ds := createTestUsers(t, createTestDatastore(t))
	server := createTestServer(ds)

	////////////////////////////////////////////////////////////////////////////
	// log-in with an admin
	////////////////////////////////////////////////////////////////////////////

	// log in with admin test user
	response := makeRequest(
		t,
		server,
		"POST",
		"/api/v1/kolide/login",
		CreateUserRequestBody{
			Username: "admin1",
			Password: "foobar",
		},
		"",
	)
	assert.Equal(t, http.StatusOK, response.Code)

	// ensure that a non-empty cookie was in-fact set
	adminCookie := response.Header().Get("Set-Cookie")
	assert.NotEmpty(t, adminCookie)

	////////////////////////////////////////////////////////////////////////////
	// create test user account
	////////////////////////////////////////////////////////////////////////////

	// make the request to create the new user and verify that it succeeded
	response = makeRequest(
		t,
		server,
		"PUT",
		"/api/v1/kolide/user",
		CreateUserRequestBody{
			Username:           "tester",
			Password:           "temp",
			Email:              "tester@kolide.co",
			Admin:              false,
			NeedsPasswordReset: true,
		},
		adminCookie,
	)
	assert.Equal(t, http.StatusOK, response.Code)

	// ensure that the new user was created in the database
	_, err := ds.User("tester")
	assert.Nil(t, err)
}

func TestModifyUser(t *testing.T) {
	// create the test datastore and server
	ds := createTestUsers(t, createTestDatastore(t))
	server := createTestServer(ds)

	////////////////////////////////////////////////////////////////////////////
	// log-in with an admin
	////////////////////////////////////////////////////////////////////////////

	// log in with admin test user
	response := makeRequest(
		t,
		server,
		"POST",
		"/api/v1/kolide/login",
		CreateUserRequestBody{
			Username: "admin1",
			Password: "foobar",
		},
		"",
	)
	assert.Equal(t, http.StatusOK, response.Code)

	// ensure that a non-empty cookie was in-fact set
	adminCookie := response.Header().Get("Set-Cookie")
	assert.NotEmpty(t, adminCookie)

	////////////////////////////////////////////////////////////////////////////
	// modify user1 to have a different name
	////////////////////////////////////////////////////////////////////////////

	user1, err := ds.User("user1")
	assert.Nil(t, err)

	response = makeRequest(
		t,
		server,
		"PATCH",
		"/api/v1/kolide/user",
		ModifyUserRequestBody{
			ID:   user1.ID,
			Name: "User McSwiggins",
		},
		adminCookie,
	)
	assert.Equal(t, http.StatusOK, response.Code)

	var user1Info GetUserResponseBody
	err = json.NewDecoder(response.Body).Decode(&user1Info)
	assert.Nil(t, err)

	assert.Equal(t, user1Info.Name, "User McSwiggins")

	////////////////////////////////////////////////////////////////////////////
	// verify name change in the database
	////////////////////////////////////////////////////////////////////////////

	user1, err = ds.User("user1")
	assert.Nil(t, err)
	assert.Equal(t, user1.Name, "User McSwiggins")
}

func TestSetUserAdminState(t *testing.T) {
	// create the test datastore and server
	ds := createTestUsers(t, createTestDatastore(t))
	server := createTestServer(ds)

	////////////////////////////////////////////////////////////////////////////
	// log-in with an admin
	////////////////////////////////////////////////////////////////////////////

	// log in with admin test user
	response := makeRequest(
		t,
		server,
		"POST",
		"/api/v1/kolide/login",
		CreateUserRequestBody{
			Username: "admin1",
			Password: "foobar",
		},
		"",
	)
	assert.Equal(t, http.StatusOK, response.Code)

	// ensure that a non-empty cookie was in-fact set
	adminCookie := response.Header().Get("Set-Cookie")
	assert.NotEmpty(t, adminCookie)

	////////////////////////////////////////////////////////////////////////////
	// promote user1 to admin
	////////////////////////////////////////////////////////////////////////////

	user1, err := ds.User("user1")
	assert.Nil(t, err)

	response = makeRequest(
		t,
		server,
		"PATCH",
		"/api/v1/kolide/user/admin",
		SetUserAdminStateRequestBody{
			ID:    user1.ID,
			Admin: true,
		},
		adminCookie,
	)
	assert.Equal(t, http.StatusOK, response.Code)

	var user1Info GetUserResponseBody
	err = json.NewDecoder(response.Body).Decode(&user1Info)
	assert.Nil(t, err)

	assert.True(t, user1Info.Admin)

	////////////////////////////////////////////////////////////////////////////
	// verify change in the database
	////////////////////////////////////////////////////////////////////////////

	user1, err = ds.User("user1")
	assert.Nil(t, err)
	assert.True(t, user1.Admin)
}

func TestSetUserEnabledState(t *testing.T) {
	// create the test datastore and server
	ds := createTestUsers(t, createTestDatastore(t))
	server := createTestServer(ds)

	////////////////////////////////////////////////////////////////////////////
	// log-in with an admin
	////////////////////////////////////////////////////////////////////////////

	// log in with admin test user
	response := makeRequest(
		t,
		server,
		"POST",
		"/api/v1/kolide/login",
		CreateUserRequestBody{
			Username: "admin1",
			Password: "foobar",
		},
		"",
	)
	assert.Equal(t, http.StatusOK, response.Code)

	// ensure that a non-empty cookie was in-fact set
	adminCookie := response.Header().Get("Set-Cookie")
	assert.NotEmpty(t, adminCookie)

	////////////////////////////////////////////////////////////////////////////
	// disable user1
	////////////////////////////////////////////////////////////////////////////

	user1, err := ds.User("user1")
	assert.Nil(t, err)

	response = makeRequest(
		t,
		server,
		"PATCH",
		"/api/v1/kolide/user/enabled",
		SetUserEnabledStateRequestBody{
			ID:      user1.ID,
			Enabled: false,
		},
		adminCookie,
	)
	assert.Equal(t, http.StatusOK, response.Code)

	var user1Info GetUserResponseBody
	err = json.NewDecoder(response.Body).Decode(&user1Info)
	assert.Nil(t, err)

	assert.False(t, user1Info.Enabled)

	////////////////////////////////////////////////////////////////////////////
	// verify change in the database
	////////////////////////////////////////////////////////////////////////////

	user1, err = ds.User("user1")
	assert.Nil(t, err)
	assert.False(t, user1.Enabled)
}

func TestUserChangeTheirOwnPassword(t *testing.T) {
	// create the test datastore and server
	ds := createTestUsers(t, createTestDatastore(t))
	server := createTestServer(ds)

	////////////////////////////////////////////////////////////////////////////
	// log-in with a user
	////////////////////////////////////////////////////////////////////////////

	// log in with admin test user
	response := makeRequest(
		t,
		server,
		"POST",
		"/api/v1/kolide/login",
		CreateUserRequestBody{
			Username: "user1",
			Password: "foobar",
		},
		"",
	)
	assert.Equal(t, http.StatusOK, response.Code)

	// ensure that a non-empty cookie was in-fact set
	userCookie := response.Header().Get("Set-Cookie")
	assert.NotEmpty(t, userCookie)

	////////////////////////////////////////////////////////////////////////////
	// user changes their own password
	////////////////////////////////////////////////////////////////////////////

	user1, err := ds.User("user1")
	assert.Nil(t, err)

	response = makeRequest(
		t,
		server,
		"PATCH",
		"/api/v1/kolide/user/password",
		ChangePasswordRequestBody{
			ID:                user1.ID,
			CurrentPassword:   "foobar",
			NewPassword:       "baz",
			NewPasswordConfim: "baz",
		},
		userCookie,
	)
	assert.Equal(t, http.StatusOK, response.Code)

	////////////////////////////////////////////////////////////////////////////
	// verify change in the database
	////////////////////////////////////////////////////////////////////////////

	user1, err = ds.User("user1")
	assert.Nil(t, err)
	assert.Nil(t, user1.ValidatePassword("baz"))
}

func TestUserChangeOtherUsersPassword(t *testing.T) {
	// create the test datastore and server
	ds := createTestUsers(t, createTestDatastore(t))
	server := createTestServer(ds)

	////////////////////////////////////////////////////////////////////////////
	// log-in with a user
	////////////////////////////////////////////////////////////////////////////

	// log in with test user
	response := makeRequest(
		t,
		server,
		"POST",
		"/api/v1/kolide/login",
		CreateUserRequestBody{
			Username: "user1",
			Password: "foobar",
		},
		"",
	)
	assert.Equal(t, http.StatusOK, response.Code)

	// ensure that a non-empty cookie was in-fact set
	userCookie := response.Header().Get("Set-Cookie")
	assert.NotEmpty(t, userCookie)

	////////////////////////////////////////////////////////////////////////////
	// user tries to change other user's password
	////////////////////////////////////////////////////////////////////////////

	user2, err := ds.User("user2")
	assert.Nil(t, err)

	response = makeRequest(
		t,
		server,
		"PATCH",
		"/api/v1/kolide/user/password",
		ChangePasswordRequestBody{
			ID:                user2.ID,
			CurrentPassword:   "foobar",
			NewPassword:       "baz",
			NewPasswordConfim: "baz",
		},
		userCookie,
	)
	assert.Equal(t, http.StatusUnauthorized, response.Code)

	////////////////////////////////////////////////////////////////////////////
	// verify nothing changed in the database
	////////////////////////////////////////////////////////////////////////////

	user2, err = ds.User("user2")
	assert.Nil(t, err)
	assert.Nil(t, user2.ValidatePassword("foobar"))
}

func TestAdminChangeUserPassword(t *testing.T) {
	// create the test datastore and server
	ds := createTestUsers(t, createTestDatastore(t))
	server := createTestServer(ds)

	////////////////////////////////////////////////////////////////////////////
	// log-in with an admin
	////////////////////////////////////////////////////////////////////////////

	// log in with admin test user
	response := makeRequest(
		t,
		server,
		"POST",
		"/api/v1/kolide/login",
		CreateUserRequestBody{
			Username: "admin1",
			Password: "foobar",
		},
		"",
	)
	assert.Equal(t, http.StatusOK, response.Code)

	// ensure that a non-empty cookie was in-fact set
	adminCookie := response.Header().Get("Set-Cookie")
	assert.NotEmpty(t, adminCookie)

	////////////////////////////////////////////////////////////////////////////
	// admin changes other user's password
	////////////////////////////////////////////////////////////////////////////

	user2, err := ds.User("user2")
	assert.Nil(t, err)

	response = makeRequest(
		t,
		server,
		"PATCH",
		"/api/v1/kolide/user/password",
		ChangePasswordRequestBody{
			ID:                user2.ID,
			NewPassword:       "baz",
			NewPasswordConfim: "baz",
		},
		adminCookie,
	)
	assert.Equal(t, http.StatusOK, response.Code)

	////////////////////////////////////////////////////////////////////////////
	// verify change in the database
	////////////////////////////////////////////////////////////////////////////

	user2, err = ds.User("user2")
	assert.Nil(t, err)
	assert.Nil(t, user2.ValidatePassword("baz"))
}

func TestChangePasswordEnforcesSamePassword(t *testing.T) {
	// create the test datastore and server
	ds := createTestUsers(t, createTestDatastore(t))
	server := createTestServer(ds)

	////////////////////////////////////////////////////////////////////////////
	// log-in with an admin
	////////////////////////////////////////////////////////////////////////////

	// log in with admin test user
	response := makeRequest(
		t,
		server,
		"POST",
		"/api/v1/kolide/login",
		CreateUserRequestBody{
			Username: "admin1",
			Password: "foobar",
		},
		"",
	)
	assert.Equal(t, http.StatusOK, response.Code)

	// ensure that a non-empty cookie was in-fact set
	adminCookie := response.Header().Get("Set-Cookie")
	assert.NotEmpty(t, adminCookie)

	////////////////////////////////////////////////////////////////////////////
	// admin tries to change other user's password but mismatches the password
	// confirmation
	////////////////////////////////////////////////////////////////////////////

	user2, err := ds.User("user2")
	assert.Nil(t, err)

	response = makeRequest(
		t,
		server,
		"PATCH",
		"/api/v1/kolide/user/password",
		ChangePasswordRequestBody{
			ID:                user2.ID,
			NewPassword:       "foo",
			NewPasswordConfim: "bar",
		},
		adminCookie,
	)
	assert.Equal(t, http.StatusBadRequest, response.Code)

	////////////////////////////////////////////////////////////////////////////
	// verify that nothing changed in the database
	////////////////////////////////////////////////////////////////////////////

	user2, err = ds.User("user2")
	assert.Nil(t, err)
	assert.Nil(t, user2.ValidatePassword("foobar"))
}

func TestResetUserPassword(t *testing.T) {
	pool := kolide.NewMockSMTPConnectionPool()
	ds := createTestUsers(t, createTestDatastore(t))
	server := createTestServerWithSMTP(ds, pool)

	////////////////////////////////////////////////////////////////////////////
	// Trigger a password reset email for a user
	////////////////////////////////////////////////////////////////////////////

	response := makeRequest(
		t,
		server,
		"POST",
		"/api/v1/kolide/user/password/reset",
		ResetPasswordRequestBody{
			Username: "user1",
			Email:    "user1@kolide.co",
		},
		"",
	)
	assert.Equal(t, http.StatusOK, response.Code)
	assert.Len(t, pool.Emails, 1)
}

func TestVerifyPasswordRequest(t *testing.T) {
	ds := createTestUsers(t, createTestDatastore(t))
	server := createTestServer(ds)

	////////////////////////////////////////////////////////////////////////////
	// Trigger a password reset email for a user
	////////////////////////////////////////////////////////////////////////////

	response := makeRequest(
		t,
		server,
		"POST",
		"/api/v1/kolide/user/password/reset",
		ResetPasswordRequestBody{
			Username: "user1",
			Email:    "user1@kolide.co",
		},
		"",
	)
	assert.Equal(t, http.StatusOK, response.Code)

	////////////////////////////////////////////////////////////////////////////
	// Verify the token
	////////////////////////////////////////////////////////////////////////////
	user, err := ds.User("user1")
	assert.Nil(t, err)

	resets, err := ds.FindPassswordResetsByUserID(user.ID)
	assert.Nil(t, err)
	assert.Len(t, resets, 1)
	assert.Equal(t, user.ID, resets[0].UserID)

	response = makeRequest(
		t,
		server,
		"POST",
		"/api/v1/kolide/user/password/reset/verify",
		VerifyPasswordResetRequestRequestBody{
			UserID: user.ID,
			Token:  resets[0].Token,
		},
		"",
	)
	assert.Equal(t, http.StatusOK, response.Code)

	var verify VerifyPasswordResetRequestResponseBody
	err = json.NewDecoder(response.Body).Decode(&verify)
	assert.Nil(t, err)
	assert.True(t, verify.Valid)
}

func TestDeletePasswordResetRequest(t *testing.T) {
	ds := createTestUsers(t, createTestDatastore(t))
	server := createTestServer(ds)

	////////////////////////////////////////////////////////////////////////////
	// Trigger a password reset email for a user
	////////////////////////////////////////////////////////////////////////////

	response := makeRequest(
		t,
		server,
		"POST",
		"/api/v1/kolide/user/password/reset",
		ResetPasswordRequestBody{
			Username: "user1",
			Email:    "user1@kolide.co",
		},
		"",
	)
	assert.Equal(t, http.StatusOK, response.Code)

	////////////////////////////////////////////////////////////////////////////
	// Delete the token
	////////////////////////////////////////////////////////////////////////////
	user, err := ds.User("user1")
	assert.Nil(t, err)

	resets, err := ds.FindPassswordResetsByUserID(user.ID)
	assert.Nil(t, err)
	assert.Len(t, resets, 1)
	assert.Equal(t, user.ID, resets[0].UserID)

	// log in with test user
	response = makeRequest(
		t,
		server,
		"POST",
		"/api/v1/kolide/login",
		CreateUserRequestBody{
			Username: "user1",
			Password: "foobar",
		},
		"",
	)
	assert.Equal(t, http.StatusOK, response.Code)
	userCookie := response.Header().Get("Set-Cookie")
	assert.NotEmpty(t, userCookie)

	response = makeRequest(
		t,
		server,
		"DELETE",
		"/api/v1/kolide/user/password/reset",
		DeletePasswordResetRequestRequestBody{
			ID: resets[0].ID,
		},
		userCookie,
	)
	assert.Equal(t, http.StatusNoContent, response.Code)

	resets, err = ds.FindPassswordResetsByUserID(user.ID)
	assert.NotNil(t, err)
	assert.Len(t, resets, 0)
}
