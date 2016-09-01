package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/kolide/kolide-ose/kolide"
	"github.com/stretchr/testify/assert"
)

func TestLoginAndLogout(t *testing.T) {
	// create the test datastore and server
	ds := createTestUsers(t, createTestDatastore(t))
	server := createTestServer(ds)

	admin, err := ds.User("admin1")

	// ensure that there are no sessions in the database for our test user
	sessions, err := ds.FindAllSessionsForUser(admin.ID)
	assert.Nil(t, err)
	assert.Len(t, sessions, 0)

	////////////////////////////////////////////////////////////////////////////
	// Test logging in
	////////////////////////////////////////////////////////////////////////////

	// log in with test user created above
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
	cookie := response.Header().Get("Set-Cookie")
	assert.NotEmpty(t, cookie)

	// ensure that a session was created for our test user and stored
	sessions, err = ds.FindAllSessionsForUser(admin.ID)
	assert.Nil(t, err)
	assert.Len(t, sessions, 1)

	// ensure the session key is not blank
	assert.NotEqual(t, "", sessions[0].Key)

	////////////////////////////////////////////////////////////////////////////
	// Test logging out
	////////////////////////////////////////////////////////////////////////////

	// log out our test user
	response = makeRequest(
		t,
		server,
		"GET",
		"/api/v1/kolide/logout",
		nil,
		cookie,
	)
	assert.Equal(t, http.StatusOK, response.Code)

	// ensure that a cookie was actually set to erase the current cookie
	assert.Equal(t, "KolideSession=", response.Header().Get("Set-Cookie"))

	// ensure that our user's session was deleted from the store
	sessions, err = ds.FindAllSessionsForUser(admin.ID)
	assert.Nil(t, err)
	assert.Len(t, sessions, 0)
}

func TestNeedsPasswordReset(t *testing.T) {
	// create the test datastore and server
	ds := createTestUsers(t, createTestDatastore(t))
	server := createTestServer(ds)

	////////////////////////////////////////////////////////////////////////////
	// log-in with a user which needs a password reset
	////////////////////////////////////////////////////////////////////////////

	// log in with admin test user
	response := makeRequest(
		t,
		server,
		"POST",
		"/api/v1/kolide/login",
		CreateUserRequestBody{
			Username: "user2",
			Password: "foobar",
		},
		"",
	)
	assert.Equal(t, http.StatusOK, response.Code)

	// ensure that a non-empty cookie was in-fact set
	userCookie := response.Header().Get("Set-Cookie")
	assert.NotEmpty(t, userCookie)

	////////////////////////////////////////////////////////////////////////////
	// get the info of user1 from user2's account
	////////////////////////////////////////////////////////////////////////////

	user1, err := ds.User("user1")
	assert.Nil(t, err)

	response = makeRequest(
		t,
		server,
		"POST",
		"/api/v1/kolide/user",
		GetUserRequestBody{
			ID: user1.ID,
		},
		userCookie,
	)
	assert.Equal(t, http.StatusUnauthorized, response.Code)
}

func TestDeleteSession(t *testing.T) {
	// create the test datastore and server
	ds := createTestUsers(t, createTestDatastore(t))
	server := createTestServer(ds)

	admin, err := ds.User("admin1")

	// ensure that there are no sessions in the database for our test user
	sessions, err := ds.FindAllSessionsForUser(admin.ID)
	assert.Nil(t, err)
	assert.Len(t, sessions, 0)

	////////////////////////////////////////////////////////////////////////////
	// Login and create session
	////////////////////////////////////////////////////////////////////////////

	// log in with test user created above
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
	cookie1 := response.Header().Get("Set-Cookie")
	assert.NotEmpty(t, cookie1)

	// ensure that a session was created for our test user and stored
	sessions, err = ds.FindAllSessionsForUser(admin.ID)
	assert.Nil(t, err)
	assert.Len(t, sessions, 1)

	////////////////////////////////////////////////////////////////////////////
	// Login and create another session
	////////////////////////////////////////////////////////////////////////////

	// log in with test user created above
	response = makeRequest(
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
	cookie2 := response.Header().Get("Set-Cookie")
	assert.NotEmpty(t, cookie2)

	// ensure that a session was created for our test user and stored
	sessions, err = ds.FindAllSessionsForUser(admin.ID)
	assert.Nil(t, err)
	assert.Len(t, sessions, 2)

	////////////////////////////////////////////////////////////////////////////
	// Delete one session
	////////////////////////////////////////////////////////////////////////////

	// log out our test user
	response = makeRequest(
		t,
		server,
		"DELETE",
		"/api/v1/kolide/session",
		DeleteSessionRequestBody{
			SessionID: sessions[0].ID,
		},
		cookie1,
	)
	assert.Equal(t, http.StatusOK, response.Code)

	////////////////////////////////////////////////////////////////////////////
	// Verify there is only one session in the database
	////////////////////////////////////////////////////////////////////////////

	// ensure that our user's session was deleted from the store
	sessions, err = ds.FindAllSessionsForUser(admin.ID)
	assert.Nil(t, err)
	assert.Len(t, sessions, 1)
}

func TestDeleteSessionsForUser(t *testing.T) {
	// create the test datastore and server
	ds := createTestUsers(t, createTestDatastore(t))
	server := createTestServer(ds)

	admin, err := ds.User("admin1")

	// ensure that there are no sessions in the database for our test user
	sessions, err := ds.FindAllSessionsForUser(admin.ID)
	assert.Nil(t, err)
	assert.Len(t, sessions, 0)

	////////////////////////////////////////////////////////////////////////////
	// Login and create session
	////////////////////////////////////////////////////////////////////////////

	// log in with test user created above
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
	cookie1 := response.Header().Get("Set-Cookie")
	assert.NotEmpty(t, cookie1)

	// ensure that a session was created for our test user and stored
	sessions, err = ds.FindAllSessionsForUser(admin.ID)
	assert.Nil(t, err)
	assert.Len(t, sessions, 1)

	////////////////////////////////////////////////////////////////////////////
	// Login and create another session
	////////////////////////////////////////////////////////////////////////////

	// log in with test user created above
	response = makeRequest(
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
	cookie2 := response.Header().Get("Set-Cookie")
	assert.NotEmpty(t, cookie2)

	// ensure that a session was created for our test user and stored
	sessions, err = ds.FindAllSessionsForUser(admin.ID)
	assert.Nil(t, err)
	assert.Len(t, sessions, 2)

	////////////////////////////////////////////////////////////////////////////
	// Delete one session
	////////////////////////////////////////////////////////////////////////////

	// log out our test user
	response = makeRequest(
		t,
		server,
		"DELETE",
		"/api/v1/kolide/user/sessions",
		DeleteSessionsForUserRequestBody{
			ID: admin.ID,
		},
		cookie1,
	)
	assert.Equal(t, http.StatusOK, response.Code)

	////////////////////////////////////////////////////////////////////////////
	// Verify there are no sessions in the database
	////////////////////////////////////////////////////////////////////////////

	// ensure that our user's sessions were deleted from the store
	sessions, err = ds.FindAllSessionsForUser(admin.ID)
	assert.Nil(t, err)
	assert.Len(t, sessions, 0)
}

func TestGetInfoAboutSession(t *testing.T) {
	// create the test datastore and server
	ds := createTestUsers(t, createTestDatastore(t))
	server := createTestServer(ds)

	admin, err := ds.User("admin1")

	// ensure that there are no sessions in the database for our test user
	sessions, err := ds.FindAllSessionsForUser(admin.ID)
	assert.Nil(t, err)
	assert.Len(t, sessions, 0)

	////////////////////////////////////////////////////////////////////////////
	// Login and create session
	////////////////////////////////////////////////////////////////////////////

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
	cookie := response.Header().Get("Set-Cookie")
	assert.NotEmpty(t, cookie)

	// ensure that a session was created for our test user and stored
	sessions, err = ds.FindAllSessionsForUser(admin.ID)
	assert.Nil(t, err)
	assert.Len(t, sessions, 1)

	////////////////////////////////////////////////////////////////////////////
	// Get info about sessions for admin1
	////////////////////////////////////////////////////////////////////////////

	token, err := kolide.ParseJWT(strings.Split(cookie, "=")[1], "")
	assert.Nil(t, err)

	claims, ok := token.Claims.(jwt.MapClaims)
	assert.True(t, ok)

	key, ok := claims["session_key"]
	assert.True(t, ok)

	response = makeRequest(
		t,
		server,
		"POST",
		"/api/v1/kolide/session",
		GetInfoAboutSessionRequestBody{
			SessionKey: key.(string),
		},
		cookie,
	)
	assert.Equal(t, http.StatusOK, response.Code)

	var sessionInfo SessionInfoResponseBody
	err = json.NewDecoder(response.Body).Decode(&sessionInfo)
	assert.Nil(t, err)

	assert.Equal(t, sessionInfo.SessionID, sessions[0].ID)
}

func TestGetInfoAboutSessionsForUser(t *testing.T) {
	// create the test datastore and server
	ds := createTestUsers(t, createTestDatastore(t))
	server := createTestServer(ds)

	admin, err := ds.User("admin1")

	// ensure that there are no sessions in the database for our test user
	sessions, err := ds.FindAllSessionsForUser(admin.ID)
	assert.Nil(t, err)
	assert.Len(t, sessions, 0)

	////////////////////////////////////////////////////////////////////////////
	// Login and create session
	////////////////////////////////////////////////////////////////////////////

	// log in with test user
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
	cookie1 := response.Header().Get("Set-Cookie")
	assert.NotEmpty(t, cookie1)

	// ensure that a session was created for our test user and stored
	sessions, err = ds.FindAllSessionsForUser(admin.ID)
	assert.Nil(t, err)
	assert.Len(t, sessions, 1)

	////////////////////////////////////////////////////////////////////////////
	// Login and create another session
	////////////////////////////////////////////////////////////////////////////

	// log in with test user created above
	response = makeRequest(
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
	cookie2 := response.Header().Get("Set-Cookie")
	assert.NotEmpty(t, cookie2)

	// ensure that a session was created for our test user and stored
	sessions, err = ds.FindAllSessionsForUser(admin.ID)
	assert.Nil(t, err)
	assert.Len(t, sessions, 2)

	////////////////////////////////////////////////////////////////////////////
	// Get info about user's sessions
	////////////////////////////////////////////////////////////////////////////

	response = makeRequest(
		t,
		server,
		"POST",
		"/api/v1/kolide/user/sessions",
		GetInfoAboutSessionsForUserRequestBody{
			ID: admin.ID,
		},
		cookie1,
	)
	assert.Equal(t, http.StatusOK, response.Code)

	var sessionInfo GetInfoAboutSessionsForUserResponseBody
	err = json.NewDecoder(response.Body).Decode(&sessionInfo)
	assert.Nil(t, err)

	assert.Len(t, sessionInfo.Sessions, 2)
}
