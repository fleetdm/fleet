// Package svctest provides test helpers (test server, test service, login
// helper) for the server/service package. It imports the testing package and
// must therefore only ever be imported from test code; importing it from
// production code would pull "testing" into the resulting binary.
package svctest

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sort"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/fleetdm/fleet/v4/server/service/contract"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testUsers is the set of users seeded into the database when a test server
// is created. Internal service tests have their own copy with the same
// content; both must stay in sync.
var testUsers = map[string]struct {
	Email             string
	PlaintextPassword string
	GlobalRole        *string
}{
	"admin1": {
		PlaintextPassword: test.GoodPassword,
		Email:             service.TestAdminUserEmail,
		GlobalRole:        ptr.String(fleet.RoleAdmin),
	},
	"user1": {
		PlaintextPassword: test.GoodPassword,
		Email:             service.TestMaintainerUserEmail,
		GlobalRole:        ptr.String(fleet.RoleMaintainer),
	},
	"user2": {
		PlaintextPassword: test.GoodPassword,
		Email:             service.TestObserverUserEmail,
		GlobalRole:        ptr.String(fleet.RoleObserver),
	},
}

// createTestUsers seeds the standard set of admin/maintainer/observer users
// into ds and returns them keyed by email.
func createTestUsers(t *testing.T, ds fleet.Datastore) map[string]fleet.User {
	users := make(map[string]fleet.User)
	// Map iteration is random so we sort and iterate using the testUsers keys.
	var keys []string
	for key := range testUsers {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	userID := uint(1)
	for _, key := range keys {
		u := testUsers[key]
		user := &fleet.User{
			ID:         userID,
			Name:       "Test Name " + u.Email,
			Email:      u.Email,
			GlobalRole: u.GlobalRole,
		}
		err := user.SetPassword(u.PlaintextPassword, 10, 10)
		require.NoError(t, err)
		user, err = ds.NewUser(context.Background(), user)
		require.NoError(t, err)
		users[user.Email] = *user
		userID++
	}
	return users
}

// GetToken posts to /api/latest/fleet/login and returns the auth token. It
// fails the test on any error.
func GetToken(t *testing.T, email string, password string, serverURL string) string {
	params := contract.LoginRequest{
		Email:    email,
		Password: password,
	}
	j, err := json.Marshal(&params) //nolint:gosec // dismiss G117
	require.NoError(t, err)

	requestBody := io.NopCloser(bytes.NewBuffer(j))
	resp, err := http.Post(serverURL+"/api/latest/fleet/login", "application/json", requestBody)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	jsn := struct {
		User  *fleet.User         `json:"user"`
		Token string              `json:"token"`
		Err   []map[string]string `json:"errors,omitempty"`
	}{}
	err = json.NewDecoder(resp.Body).Decode(&jsn)
	require.NoError(t, err)
	require.Empty(t, jsn.Err)

	return jsn.Token
}
