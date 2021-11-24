package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogin(t *testing.T) {
	ds, users, server := setupAuthTest(t)
	var loginTests = []struct {
		email    string
		status   int
		password string
	}{
		{
			email:    "admin1@example.com",
			password: testUsers["admin1"].PlaintextPassword,
			status:   http.StatusOK,
		},
		{
			email:    "user1@example.com",
			password: testUsers["user1"].PlaintextPassword,
			status:   http.StatusOK,
		},
		{
			email:    "nosuchuser@example.com",
			password: "nosuchuser",
			status:   http.StatusUnauthorized,
		},
		{
			email:    "admin1@example.com",
			password: "badpassword",
			status:   http.StatusUnauthorized,
		},
	}

	for _, tt := range loginTests {
		// test sessions
		testUser := users[tt.email]

		params := loginRequest{
			Email:    tt.email,
			Password: tt.password,
		}
		j, err := json.Marshal(&params)
		assert.Nil(t, err)

		requestBody := io.NopCloser(bytes.NewBuffer(j))
		resp, err := http.Post(server.URL+"/api/v1/fleet/login", "application/json", requestBody)
		require.Nil(t, err)
		assert.Equal(t, tt.status, resp.StatusCode)

		var jsn = struct {
			User  *fleet.User         `json:"user"`
			Token string              `json:"token"`
			Err   []map[string]string `json:"errors,omitempty"`
		}{}
		err = json.NewDecoder(resp.Body).Decode(&jsn)
		require.Nil(t, err)

		if tt.status != http.StatusOK {
			assert.NotEqual(t, "", jsn.Err)
			continue // skip remaining tests
		}

		require.NotNil(t, jsn.User)
		assert.Equal(t, tt.email, jsn.User.Email)

		// ensure that a session was created for our test user and stored
		sessions, err := ds.ListSessionsForUser(context.Background(), testUser.ID)
		assert.Nil(t, err)
		assert.Len(t, sessions, 1)

		// ensure the session key is not blank
		assert.NotEqual(t, "", sessions[0].Key)

		// test logout
		req, _ := http.NewRequest("POST", server.URL+"/api/v1/fleet/logout", nil)
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", jsn.Token))
		client := fleethttp.NewClient()
		resp, err = client.Do(req)
		require.Nil(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode, strconv.Itoa(tt.status))

		_, err = ioutil.ReadAll(resp.Body)
		assert.Nil(t, err)

		// ensure that our user's session was deleted from the store
		sessions, err = ds.ListSessionsForUser(context.Background(), testUser.ID)
		assert.Nil(t, err)
		assert.Len(t, sessions, 0)
	}
}

func setupAuthTest(t *testing.T) (fleet.Datastore, map[string]fleet.User, *httptest.Server) {
	ds := new(mock.Store)
	var users []*fleet.User
	var admin *fleet.User
	sessions := make(map[string]*fleet.Session)
	ds.NewUserFunc = func(ctx context.Context, user *fleet.User) (*fleet.User, error) {
		if user.GlobalRole != nil && *user.GlobalRole == fleet.RoleAdmin {
			admin = user
		}
		users = append(users, user)
		return user, nil
	}
	ds.SessionByKeyFunc = func(ctx context.Context, key string) (*fleet.Session, error) {
		return sessions[key], nil
	}
	ds.MarkSessionAccessedFunc = func(ctx context.Context, session *fleet.Session) error {
		return nil
	}
	ds.UserByIDFunc = func(ctx context.Context, id uint) (*fleet.User, error) {
		return admin, nil
	}
	ds.ListUsersFunc = func(ctx context.Context, opt fleet.UserListOptions) ([]*fleet.User, error) {
		return users, nil
	}
	ds.ListSessionsForUserFunc = func(ctx context.Context, id uint) ([]*fleet.Session, error) {
		for _, session := range sessions {
			if session.UserID == id {
				return []*fleet.Session{session}, nil
			}
		}
		return nil, nil
	}
	ds.SessionByIDFunc = func(ctx context.Context, id uint) (*fleet.Session, error) {
		for _, session := range sessions {
			if session.ID == id {
				return session, nil
			}
		}
		return nil, nil
	}
	ds.DestroySessionFunc = func(ctx context.Context, session *fleet.Session) error {
		delete(sessions, session.Key)
		return nil
	}
	usersMap, server := RunServerForTestsWithDS(t, ds)
	ds.UserByEmailFunc = func(ctx context.Context, email string) (*fleet.User, error) {
		user := usersMap[email]
		return &user, nil
	}
	ds.NewSessionFunc = func(ctx context.Context, session *fleet.Session) (*fleet.Session, error) {
		sessions[session.Key] = session
		return session, nil
	}
	return ds, usersMap, server
}

func getTestAdminToken(t *testing.T, server *httptest.Server) string {
	testUser := testUsers["admin1"]

	params := loginRequest{
		Email:    testUser.Email,
		Password: testUser.PlaintextPassword,
	}
	j, err := json.Marshal(&params)
	assert.Nil(t, err)

	requestBody := io.NopCloser(bytes.NewBuffer(j))
	resp, err := http.Post(server.URL+"/api/v1/fleet/login", "application/json", requestBody)
	require.Nil(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var jsn = struct {
		User  *fleet.User         `json:"user"`
		Token string              `json:"token"`
		Err   []map[string]string `json:"errors,omitempty"`
	}{}
	err = json.NewDecoder(resp.Body).Decode(&jsn)
	require.Nil(t, err)

	return jsn.Token
}

func TestNoHeaderErrorsDifferently(t *testing.T) {
	_, _, server := setupAuthTest(t)

	req, _ := http.NewRequest("GET", server.URL+"/api/v1/fleet/users", nil)
	client := fleethttp.NewClient()
	resp, err := client.Do(req)
	require.Nil(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.Nil(t, err)
	assert.Equal(t, `{
  "message": "Authorization header required",
  "errors": [
    {
      "name": "base",
      "reason": "Authorization header required"
    }
  ]
}
`, string(bodyBytes))

	req, _ = http.NewRequest("GET", server.URL+"/api/v1/fleet/users", nil)
	req.Header.Add("Authorization", "Bearer AAAA")
	resp, err = client.Do(req)
	require.Nil(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	bodyBytes, err = ioutil.ReadAll(resp.Body)
	require.Nil(t, err)
	assert.Equal(t, `{
  "message": "Authentication required",
  "errors": [
    {
      "name": "base",
      "reason": "Authentication required"
    }
  ]
}
`, string(bodyBytes))
}
