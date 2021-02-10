package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testAdminUserSetAdmin(t *testing.T, r *testResource) {
	user, err := r.ds.User("user1")
	require.Nil(t, err)
	assert.False(t, user.Admin)
	inJson := `{"admin":true}`
	buff := bytes.NewBufferString(inJson)
	path := fmt.Sprintf("/api/v1/fleet/users/%d/admin", user.ID)
	req, err := http.NewRequest("POST", r.server.URL+path, buff)
	require.Nil(t, err)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", r.adminToken))
	client := &http.Client{}
	resp, err := client.Do(req)
	require.Nil(t, err)
	var actual adminUserResponse
	err = json.NewDecoder(resp.Body).Decode(&actual)
	require.Nil(t, err)
	assert.Nil(t, actual.Err)
	require.NotNil(t, actual.User)
	assert.True(t, actual.User.Admin)
	user, err = r.ds.User("user1")
	require.Nil(t, err)
	assert.True(t, user.Admin)
}

func testNonAdminUserSetAdmin(t *testing.T, r *testResource) {
	user, err := r.ds.User("user1")
	require.Nil(t, err)
	assert.False(t, user.Admin)

	inJson := `{"admin":true}`
	buff := bytes.NewBufferString(inJson)
	path := fmt.Sprintf("/api/v1/fleet/users/%d/admin", user.ID)
	req, err := http.NewRequest("POST", r.server.URL+path, buff)
	require.Nil(t, err)
	// user NOT admin
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", r.userToken))
	client := &http.Client{}
	resp, err := client.Do(req)
	require.Nil(t, err)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)

	user, err = r.ds.User("user1")
	require.Nil(t, err)
	assert.False(t, user.Admin)
}

func testAdminUserSetEnabled(t *testing.T, r *testResource) {
	user, err := r.ds.User("user1")
	require.Nil(t, err)
	assert.True(t, user.Enabled)
	inJson := `{"enabled":false}`
	buff := bytes.NewBufferString(inJson)
	path := fmt.Sprintf("/api/v1/fleet/users/%d/enable", user.ID)
	req, err := http.NewRequest("POST", r.server.URL+path, buff)
	require.Nil(t, err)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", r.adminToken))
	client := &http.Client{}
	resp, err := client.Do(req)
	require.Nil(t, err)
	var actual adminUserResponse
	err = json.NewDecoder(resp.Body).Decode(&actual)
	require.Nil(t, err)
	assert.Nil(t, actual.Err)
	require.NotNil(t, actual.User)
	assert.False(t, actual.User.Enabled)
	user, err = r.ds.User("user1")
	require.Nil(t, err)
	assert.False(t, user.Enabled)
}

func testNonAdminUserSetEnabled(t *testing.T, r *testResource) {
	user, err := r.ds.User("user1")
	require.Nil(t, err)
	assert.True(t, user.Enabled)

	inJson := `{"enabled":false}`
	buff := bytes.NewBufferString(inJson)
	path := fmt.Sprintf("/api/v1/fleet/users/%d/enable", user.ID)
	req, err := http.NewRequest("POST", r.server.URL+path, buff)
	require.Nil(t, err)
	// user NOT admin
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", r.userToken))
	client := &http.Client{}
	resp, err := client.Do(req)
	require.Nil(t, err)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)

	user, err = r.ds.User("user1")
	require.Nil(t, err)
	// shouldn't change
	assert.True(t, user.Enabled)
}
