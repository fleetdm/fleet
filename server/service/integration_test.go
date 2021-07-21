package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDoubleUserCreationErrors(t *testing.T) {
	ds := mysql.CreateMySQLDS(t)
	defer ds.Close()

	_, server := RunServerForTestsWithDS(t, ds)
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

func TestUserWithoutRoleErrors(t *testing.T) {
	ds := mysql.CreateMySQLDS(t)
	defer ds.Close()

	_, server := RunServerForTestsWithDS(t, ds)
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

func TestUserWithWrongRoleErrors(t *testing.T) {
	ds := mysql.CreateMySQLDS(t)
	defer ds.Close()

	_, server := RunServerForTestsWithDS(t, ds)
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

func TestUserCreationWrongTeamErrors(t *testing.T) {
	ds := mysql.CreateMySQLDS(t)
	defer ds.Close()

	_, server := RunServerForTestsWithDS(t, ds)
	token := getTestAdminToken(t, server)

	teams := []fleet.UserTeam{
		{
			Team: fleet.Team{
				ID: 9999,
			},
			Role: fleet.RoleObserver,
		},
	}

	params := fleet.UserPayload{
		Name:     ptr.String("user1"),
		Email:    ptr.String("email@asd.com"),
		Password: ptr.String("pass"),
		Teams:    &teams,
	}
	method := "POST"
	path := "/api/v1/fleet/users/admin"
	expectedStatusCode := http.StatusUnprocessableEntity

	resp := doReq(t, params, method, server, path, token, expectedStatusCode)
	assertBodyContains(t, resp, `Error 1452: Cannot add or update a child row: a foreign key constraint fails`)
}

func doReq(
	t *testing.T,
	params interface{},
	method string,
	server *httptest.Server,
	path string,
	token string,
	expectedStatusCode int,
) *http.Response {
	j, err := json.Marshal(&params)
	assert.Nil(t, err)

	requestBody := &nopCloser{bytes.NewBuffer(j)}
	req, _ := http.NewRequest(method, server.URL+path, requestBody)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	client := &http.Client{}
	resp, err := client.Do(req)
	require.Nil(t, err)
	assert.Equal(t, expectedStatusCode, resp.StatusCode)
	return resp
}

func doJSONReq(
	t *testing.T,
	params interface{},
	method string,
	server *httptest.Server,
	path string,
	token string,
	expectedStatusCode int,
	v interface{},
) {
	resp := doReq(t, params, method, server, path, token, expectedStatusCode)
	err := json.NewDecoder(resp.Body).Decode(v)
	require.Nil(t, err)
}

func assertBodyContains(t *testing.T, resp *http.Response, expectedError string) {
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.Nil(t, err)
	bodyString := string(bodyBytes)
	assert.Contains(t, bodyString, expectedError)
}

func TestQueryCreationLogsActivity(t *testing.T) {
	ds := mysql.CreateMySQLDS(t)
	defer ds.Close()

	_, server := RunServerForTestsWithDS(t, ds)
	token := getTestAdminToken(t, server)

	params := fleet.QueryPayload{
		Name:  ptr.String("user1"),
		Query: ptr.String("select * from time;"),
	}
	doReq(t, params, "POST", server, "/api/v1/fleet/queries", token, http.StatusOK)
	type activitiesRespose struct {
		Activities []map[string]interface{} `json:"activities"`
	}
	activities := activitiesRespose{}
	doJSONReq(t, nil, "GET", server, "/api/v1/fleet/activities", token, http.StatusOK, &activities)

	assert.Len(t, activities.Activities, 1)
	assert.Equal(t, "Test Name admin1@example.com", activities.Activities[0]["actor_full_name"])
	assert.Equal(t, "created_saved_query", activities.Activities[0]["type"])
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

func TestAppConfigAdditionalQueriesCanBeRemoved(t *testing.T) {
	ds := mysql.CreateMySQLDS(t)
	defer ds.Close()

	_, server := RunServerForTestsWithDS(t, ds)
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

	doReq(t, appConfigSpec, "PATCH", server, "/api/v1/fleet/config", token, http.StatusOK)
}

func getConfig(t *testing.T, server *httptest.Server, token string) *fleet.AppConfigPayload {
	var responseBody *fleet.AppConfigPayload
	doJSONReq(t, nil, "GET", server, "/api/v1/fleet/config", token, http.StatusOK, &responseBody)
	return responseBody
}

func TestUserRolesSpec(t *testing.T) {
	ds := mysql.CreateMySQLDS(t)
	defer ds.Close()

	_, server := RunServerForTestsWithDS(t, ds)
	_, err := ds.NewTeam(&fleet.Team{
		ID:          42,
		Name:        "team1",
		Description: "desc team1",
	})
	require.NoError(t, err)
	token := getTestAdminToken(t, server)

	user, err := ds.UserByEmail("user1@example.com")
	require.NoError(t, err)
	assert.Len(t, user.Teams, 0)

	spec := []byte(`
  roles:
    user1@example.com:
      global_role: null
      teams:
      - role: maintainer
        team: team1
`)

	var userRoleSpec applyUserRoleSpecsRequest
	err = yaml.Unmarshal(spec, &userRoleSpec.Spec)
	require.NoError(t, err)

	doReq(t, userRoleSpec, "POST", server, "/api/v1/fleet/users/roles/spec", token, http.StatusOK)

	user, err = ds.UserByEmail("user1@example.com")
	require.NoError(t, err)
	require.Len(t, user.Teams, 1)
	assert.Equal(t, fleet.RoleMaintainer, user.Teams[0].Role)

	// But users are not deleted
	users, err := ds.ListUsers(fleet.UserListOptions{})
	require.NoError(t, err)
	assert.Len(t, users, 3)
}

func TestGlobalSchedule(t *testing.T) {
	ds := mysql.CreateMySQLDS(t)
	defer ds.Close()

	test.AddAllHostsLabel(t, ds)

	_, server := RunServerForTestsWithDS(t, ds)
	token := getTestAdminToken(t, server)

	gs := fleet.GlobalSchedulePayload{}
	doJSONReq(t, nil, "GET", server, "/api/v1/fleet/global/schedule", token, http.StatusOK, &gs)
	assert.Len(t, gs.GlobalSchedule, 0)

	qr, err := ds.NewQuery(&fleet.Query{
		Name:           "TestQuery",
		Description:    "Some description",
		Query:          "select * from osquery;",
		ObserverCanRun: true,
	})
	require.NoError(t, err)

	gsParams := fleet.ScheduledQueryPayload{QueryID: ptr.Uint(qr.ID), Interval: ptr.Uint(42)}
	type responseType struct {
		Scheduled *fleet.ScheduledQuery `json:"scheduled,omitempty"`
		Err       error                 `json:"error,omitempty"`
	}
	r := responseType{}
	doJSONReq(t, gsParams, "POST", server, "/api/v1/fleet/global/schedule", token, http.StatusOK, &r)
	require.Nil(t, r.Err)

	gs = fleet.GlobalSchedulePayload{}
	doJSONReq(t, nil, "GET", server, "/api/v1/fleet/global/schedule", token, http.StatusOK, &gs)
	require.Len(t, gs.GlobalSchedule, 1)
	assert.Equal(t, uint(42), gs.GlobalSchedule[0].Interval)
	assert.Equal(t, "TestQuery", gs.GlobalSchedule[0].Name)
	id := gs.GlobalSchedule[0].ID

	gs = fleet.GlobalSchedulePayload{}
	gsParams = fleet.ScheduledQueryPayload{Interval: ptr.Uint(55)}
	doJSONReq(
		t, gsParams, "PATCH", server,
		fmt.Sprintf("/api/v1/fleet/global/schedule/%d", id),
		token, http.StatusOK, &gs,
	)

	gs = fleet.GlobalSchedulePayload{}
	doJSONReq(t, nil, "GET", server, "/api/v1/fleet/global/schedule", token, http.StatusOK, &gs)
	require.Len(t, gs.GlobalSchedule, 1)
	assert.Equal(t, uint(55), gs.GlobalSchedule[0].Interval)

	r = responseType{}
	doJSONReq(
		t, nil, "DELETE", server,
		fmt.Sprintf("/api/v1/fleet/global/schedule/%d", id),
		token, http.StatusOK, &r,
	)
	require.Nil(t, r.Err)

	gs = fleet.GlobalSchedulePayload{}
	doJSONReq(t, nil, "GET", server, "/api/v1/fleet/global/schedule", token, http.StatusOK, &gs)
	require.Len(t, gs.GlobalSchedule, 0)
}

func TestTeamSpecs(t *testing.T) {
	ds := mysql.CreateMySQLDS(t)
	defer ds.Close()

	_, server := RunServerForTestsWithDS(t, ds)
	_, err := ds.NewTeam(&fleet.Team{
		ID:          42,
		Name:        "team1",
		Description: "desc team1",
	})
	require.NoError(t, err)
	token := getTestAdminToken(t, server)

	// updates a team
	agentOpts := json.RawMessage(`{"config": {"foo": "bar"}, "overrides": {"platforms": {"darwin": {"foo": "override"}}}}`)
	teamSpecs := applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{Name: "team1", AgentOptions: &agentOpts}}}
	doReq(t, teamSpecs, "POST", server, "/api/v1/fleet/spec/teams", token, http.StatusOK)

	team, err := ds.TeamByName("team1")
	require.NoError(t, err)

	assert.Len(t, team.Secrets, 0)
	assert.Equal(t, &agentOpts, team.AgentOptions)

	// creates a team with default agent options
	user, err := ds.UserByEmail("admin1@example.com")
	require.NoError(t, err)

	teams, err := ds.ListTeams(fleet.TeamFilter{User: user}, fleet.ListOptions{})
	require.NoError(t, err)
	assert.Len(t, teams, 1)

	teamSpecs = applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{Name: "team2"}}}
	doReq(t, teamSpecs, "POST", server, "/api/v1/fleet/spec/teams", token, http.StatusOK)

	teams, err = ds.ListTeams(fleet.TeamFilter{User: user}, fleet.ListOptions{})
	require.NoError(t, err)
	assert.Len(t, teams, 2)

	team, err = ds.TeamByName("team2")
	require.NoError(t, err)

	defaultOpts := json.RawMessage("{\"config\": {\"options\": {\"logger_plugin\": \"tls\", \"pack_delimiter\": \"/\", \"logger_tls_period\": 10, \"distributed_plugin\": \"tls\", \"disable_distributed\": false, \"logger_tls_endpoint\": \"/api/v1/osquery/log\", \"distributed_interval\": 10, \"distributed_tls_max_attempts\": 3}, \"decorators\": {\"load\": [\"SELECT uuid AS host_uuid FROM system_info;\", \"SELECT hostname AS hostname FROM system_info;\"]}}, \"overrides\": {}}")
	assert.Len(t, team.Secrets, 0)
	assert.Equal(t, &defaultOpts, team.AgentOptions)

	// updates secrets
	teamSpecs = applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{Name: "team2", Secrets: []fleet.EnrollSecret{{Secret: "ABC"}}}}}
	doReq(t, teamSpecs, "POST", server, "/api/v1/fleet/spec/teams", token, http.StatusOK)

	team, err = ds.TeamByName("team2")
	require.NoError(t, err)

	require.Len(t, team.Secrets, 1)
	assert.Equal(t, "ABC", team.Secrets[0].Secret)
}

func TestTranslator(t *testing.T) {
	ds := mysql.CreateMySQLDS(t)
	defer ds.Close()

	users, server := RunServerForTestsWithDS(t, ds)
	token := getTestAdminToken(t, server)

	payload := translatorResponse{}
	params := translatorRequest{List: []fleet.TranslatePayload{
		{
			Type:    fleet.TranslatorTypeUserEmail,
			Payload: fleet.StringIdentifierToIDPayload{Identifier: "admin1@example.com"},
		},
	}}
	doJSONReq(t, &params, "POST", server, "/api/v1/fleet/translate", token, http.StatusOK, &payload)

	require.Nil(t, payload.Err)
	assert.Len(t, payload.List, 1)

	assert.Equal(t, users[payload.List[0].Payload.Identifier].ID, payload.List[0].Payload.ID)
}
