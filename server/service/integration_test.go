package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

func getTestAdminToken(t *testing.T, server *httptest.Server) string {
	testUser := testUsers["admin1"]

	params := loginRequest{
		Email:    testUser.Email,
		Password: testUser.PlaintextPassword,
	}
	j, err := json.Marshal(&params)
	assert.Nil(t, err)

	requestBody := &nopCloser{bytes.NewBuffer(j)}
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

func testDoubleUserCreationErrors(t *testing.T, ds fleet.Datastore) {
	_, server := runServerForTestsWithDS(t, ds)
	token := getTestAdminToken(t, server)

	params := fleet.UserPayload{
		Name:       ptr.String("user1"),
		Email:      ptr.String("email@asd.com"),
		Password:   ptr.String("pass"),
		GlobalRole: ptr.String(fleet.RoleObserver),
	}
	j, err := json.Marshal(&params)
	assert.Nil(t, err)

	requestBody := &nopCloser{bytes.NewBuffer(j)}
	req, _ := http.NewRequest("POST", server.URL+"/api/v1/fleet/users/admin", requestBody)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	client := &http.Client{}
	resp, err := client.Do(req)
	require.Nil(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	requestBody = &nopCloser{bytes.NewBuffer(j)}
	req, _ = http.NewRequest("POST", server.URL+"/api/v1/fleet/users/admin", requestBody)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	resp, err = client.Do(req)
	require.Nil(t, err)
	assert.Equal(t, http.StatusConflict, resp.StatusCode)
	assertBodyContains(t, resp, `Error 1062: Duplicate entry 'email@asd.com'`)
}

func testUserWithoutRoleErrors(t *testing.T, ds fleet.Datastore) {
	_, server := runServerForTestsWithDS(t, ds)
	token := getTestAdminToken(t, server)

	params := fleet.UserPayload{
		Name:     ptr.String("user1"),
		Email:    ptr.String("email@asd.com"),
		Password: ptr.String("pass"),
	}
	j, err := json.Marshal(&params)
	assert.Nil(t, err)

	requestBody := &nopCloser{bytes.NewBuffer(j)}
	req, _ := http.NewRequest("POST", server.URL+"/api/v1/fleet/users/admin", requestBody)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	client := &http.Client{}
	resp, err := client.Do(req)
	require.Nil(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
	assertErrorCodeAndMessage(t, resp, fleet.ErrNoRoleNeeded, "either global role or team role needs to be defined")
}

func testUserWithWrongRoleErrors(t *testing.T, ds fleet.Datastore) {
	_, server := runServerForTestsWithDS(t, ds)
	token := getTestAdminToken(t, server)

	params := fleet.UserPayload{
		Name:       ptr.String("user1"),
		Email:      ptr.String("email@asd.com"),
		Password:   ptr.String("pass"),
		GlobalRole: ptr.String("wrongrole"),
	}
	j, err := json.Marshal(&params)
	assert.Nil(t, err)

	requestBody := &nopCloser{bytes.NewBuffer(j)}
	req, _ := http.NewRequest("POST", server.URL+"/api/v1/fleet/users/admin", requestBody)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	client := &http.Client{}
	resp, err := client.Do(req)
	require.Nil(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
	assertErrorCodeAndMessage(t, resp, fleet.ErrNoRoleNeeded, "GlobalRole role can only be admin, observer, or maintainer.")
}

func testUserCreationWrongTeamErrors(t *testing.T, ds fleet.Datastore) {
	_, server := runServerForTestsWithDS(t, ds)
	token := getTestAdminToken(t, server)

	teams := []fleet.UserTeam{
		{
			Team: fleet.Team{
				ID: 9999,
			},
		},
	}

	params := fleet.UserPayload{
		Name:       ptr.String("user1"),
		Email:      ptr.String("email@asd.com"),
		Password:   ptr.String("pass"),
		GlobalRole: ptr.String(fleet.RoleObserver),
		Teams:      &teams,
	}
	j, err := json.Marshal(&params)
	assert.Nil(t, err)

	requestBody := &nopCloser{bytes.NewBuffer(j)}
	req, _ := http.NewRequest("POST", server.URL+"/api/v1/fleet/users/admin", requestBody)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	client := &http.Client{}
	resp, err := client.Do(req)
	require.Nil(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
	assertBodyContains(t, resp, `Error 1452: Cannot add or update a child row: a foreign key constraint fails`)
}

func assertBodyContains(t *testing.T, resp *http.Response, expectedError string) {
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.Nil(t, err)
	bodyString := string(bodyBytes)
	assert.Contains(t, bodyString, expectedError)
}

func getJSON(r *http.Response, target interface{}) error {
	return json.NewDecoder(r.Body).Decode(target)
}

func assertErrorCodeAndMessage(t *testing.T, resp *http.Response, code int, message string) {
	err := &fleet.Error{}
	require.Nil(t, getJSON(resp, err))
	assert.Equal(t, code, err.Code)
	assert.Equal(t, message, err.Message)
}

func testAppConfigAdditionalQueriesCanBeRemoved(t *testing.T, ds fleet.Datastore) {
	_, server := runServerForTestsWithDS(t, ds)
	token := getTestAdminToken(t, server)

	spec := []byte(`
  host_expiry_settings:
    host_expiry_enabled: false
    host_expiry_window: 0
  host_settings:
    additional_queries:
      time: SELECT * FROM time
`)
	applyConfig(t, spec, server, token)

	spec = []byte(`
  host_expiry_settings:
    host_expiry_enabled: false
    host_expiry_window: 0
  host_settings:
`)
	applyConfig(t, spec, server, token)

	config := getConfig(t, server, token)
	assert.Nil(t, config.HostSettings)
}

func applyConfig(t *testing.T, spec []byte, server *httptest.Server, token string) {
	var appConfigSpec fleet.AppConfigPayload
	err := yaml.Unmarshal(spec, &appConfigSpec)
	require.NoError(t, err)
	j, err := json.Marshal(&appConfigSpec)
	assert.Nil(t, err)

	requestBody := &nopCloser{bytes.NewBuffer(j)}
	req, _ := http.NewRequest("PATCH", server.URL+"/api/v1/fleet/config", requestBody)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	client := &http.Client{}
	resp, err := client.Do(req)
	require.Nil(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func getConfig(t *testing.T, server *httptest.Server, token string) *fleet.AppConfigPayload {
	req, _ := http.NewRequest("GET", server.URL+"/api/v1/fleet/config", nil)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	client := &http.Client{}
	resp, err := client.Do(req)
	require.Nil(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var responseBody *fleet.AppConfigPayload
	err = json.NewDecoder(resp.Body).Decode(&responseBody)
	require.Nil(t, err)
	return responseBody
}

func TestSQLErrorsAreProperlyHandled(t *testing.T) {
	mysql.RunTestsAgainstMySQL(t, []func(t *testing.T, ds fleet.Datastore){
		testDoubleUserCreationErrors,
		testUserCreationWrongTeamErrors,
		testUserWithoutRoleErrors,
		testUserWithWrongRoleErrors,
		testAppConfigAdditionalQueriesCanBeRemoved,
	})
}
