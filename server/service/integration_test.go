package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/ghodss/yaml"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
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

func doRawReq(
	t *testing.T,
	body []byte,
	method string,
	server *httptest.Server,
	path string,
	token string,
	expectedStatusCode int,
) *http.Response {
	requestBody := &nopCloser{bytes.NewBuffer(body)}
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

	users, server := RunServerForTestsWithDS(t, ds)
	token := getTestAdminToken(t, server)

	admin1 := users["admin1@example.com"]
	admin1.GravatarURL = "http://iii.com"
	err := ds.SaveUser(&admin1)
	require.NoError(t, err)

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
	assert.Equal(t, "http://iii.com", activities.Activities[0]["actor_gravatar"])
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
    enable_host_users: true
`)
	applyConfig(t, spec, server, token)

	spec = []byte(`
  host_expiry_settings:
    host_expiry_enabled: false
    host_expiry_window: 0
  host_settings:
    enable_host_users: true
`)
	applyConfig(t, spec, server, token)

	config := getConfig(t, server, token)
	assert.Nil(t, config.HostSettings.AdditionalQueries)
}

func TestAppConfigHasLogging(t *testing.T) {
	ds := mysql.CreateMySQLDS(t)
	defer ds.Close()

	_, server := RunServerForTestsWithDS(t, ds)
	token := getTestAdminToken(t, server)

	config := getConfig(t, server, token)
	require.NotNil(t, config.Logging)
}

func applyConfig(t *testing.T, spec []byte, server *httptest.Server, token string) {
	var appConfigSpec fleet.AppConfigPayload
	err := yaml.Unmarshal(spec, &appConfigSpec)
	require.NoError(t, err)

	doReq(t, appConfigSpec, "PATCH", server, "/api/v1/fleet/config", token, http.StatusOK)
}

func getConfig(t *testing.T, server *httptest.Server, token string) *appConfigResponse {
	var responseBody *appConfigResponse
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
	r := globalScheduleQueryResponse{}
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

	r = globalScheduleQueryResponse{}
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
	require.JSONEq(t, string(agentOpts), string(*team.AgentOptions))

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

	defaultOpts := `{"config": {"options": {"logger_plugin": "tls", "pack_delimiter": "/", "logger_tls_period": 10, "distributed_plugin": "tls", "disable_distributed": false, "logger_tls_endpoint": "/api/v1/osquery/log", "distributed_interval": 10, "distributed_tls_max_attempts": 3}, "decorators": {"load": ["SELECT uuid AS host_uuid FROM system_info;", "SELECT hostname AS hostname FROM system_info;"]}}, "overrides": {}}`
	assert.Len(t, team.Secrets, 0)
	require.JSONEq(t, defaultOpts, string(*team.AgentOptions))

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

func TestTeamSchedule(t *testing.T) {
	ds := mysql.CreateMySQLDS(t)
	defer ds.Close()

	test.AddAllHostsLabel(t, ds)

	_, server := RunServerForTestsWithDS(t, ds)
	token := getTestAdminToken(t, server)

	team1, err := ds.NewTeam(&fleet.Team{
		ID:          42,
		Name:        "team1",
		Description: "desc team1",
	})
	require.NoError(t, err)

	ts := getTeamScheduleResponse{}
	doJSONReq(t, nil, "GET", server, fmt.Sprintf("/api/v1/fleet/team/%d/schedule", team1.ID), token, http.StatusOK, &ts)
	assert.Len(t, ts.Scheduled, 0)

	qr, err := ds.NewQuery(&fleet.Query{Name: "TestQuery", Description: "Some description", Query: "select * from osquery;", ObserverCanRun: true})
	require.NoError(t, err)

	gsParams := teamScheduleQueryRequest{ScheduledQueryPayload: fleet.ScheduledQueryPayload{QueryID: &qr.ID, Interval: ptr.Uint(42)}}
	r := teamScheduleQueryResponse{}
	doJSONReq(t, gsParams, "POST", server, fmt.Sprintf("/api/v1/fleet/team/%d/schedule", team1.ID), token, http.StatusOK, &r)
	require.Nil(t, r.Err)

	ts = getTeamScheduleResponse{}
	doJSONReq(t, nil, "GET", server, fmt.Sprintf("/api/v1/fleet/team/%d/schedule", team1.ID), token, http.StatusOK, &ts)
	require.Len(t, ts.Scheduled, 1)
	assert.Equal(t, uint(42), ts.Scheduled[0].Interval)
	assert.Equal(t, "TestQuery", ts.Scheduled[0].Name)
	assert.Equal(t, qr.ID, ts.Scheduled[0].QueryID)
	id := ts.Scheduled[0].ID

	modifyResp := modifyTeamScheduleResponse{}
	modifyParams := modifyTeamScheduleRequest{ScheduledQueryPayload: fleet.ScheduledQueryPayload{Interval: ptr.Uint(55)}}
	doJSONReq(
		t, modifyParams, "PATCH", server,
		fmt.Sprintf("/api/v1/fleet/team/%d/schedule/%d", team1.ID, id),
		token, http.StatusOK, &modifyResp,
	)

	// just to satisfy my paranoia, wanted to make sure the contents of the json would work
	doRawReq(t, []byte(`{"interval": 77}`), "PATCH", server,
		fmt.Sprintf("/api/v1/fleet/team/%d/schedule/%d", team1.ID, id),
		token, http.StatusOK)

	ts = getTeamScheduleResponse{}
	doJSONReq(t, nil, "GET", server, fmt.Sprintf("/api/v1/fleet/team/%d/schedule", team1.ID), token, http.StatusOK, &ts)
	assert.Len(t, ts.Scheduled, 1)
	assert.Equal(t, uint(77), ts.Scheduled[0].Interval)

	deleteResp := deleteTeamScheduleResponse{}
	doJSONReq(
		t, nil, "DELETE", server,
		fmt.Sprintf("/api/v1/fleet/team/%d/schedule/%d", team1.ID, id),
		token, http.StatusOK, &deleteResp,
	)
	require.Nil(t, r.Err)

	ts = getTeamScheduleResponse{}
	doJSONReq(t, nil, "GET", server, fmt.Sprintf("/api/v1/fleet/team/%d/schedule", team1.ID), token, http.StatusOK, &ts)
	assert.Len(t, ts.Scheduled, 0)
}

func TestLogger(t *testing.T) {
	buf := new(bytes.Buffer)
	logger := log.NewJSONLogger(buf)
	logger = level.NewFilter(logger, level.AllowDebug())

	ds := mysql.CreateMySQLDS(t)
	defer ds.Close()

	_, server := RunServerForTestsWithDS(t, ds, TestServerOpts{Logger: logger})
	token := getTestAdminToken(t, server)

	getConfig(t, server, token)
	params := fleet.QueryPayload{
		Name:        ptr.String("somequery"),
		Description: ptr.String("desc"),
		Query:       ptr.String("select 1 from osquery;"),
	}
	payload := createQueryRequest{}
	doJSONReq(t, params, "POST", server, "/api/v1/fleet/queries", token, http.StatusOK, &payload)

	logs := buf.String()
	parts := strings.Split(strings.TrimSpace(logs), "\n")
	assert.Len(t, parts, 3)
	for i, part := range parts {
		kv := make(map[string]string)
		err := json.Unmarshal([]byte(part), &kv)
		require.NoError(t, err)

		assert.NotEqual(t, "", kv["took"])

		switch i {
		case 0:
			assert.Equal(t, "info", kv["level"])
			assert.Equal(t, "POST", kv["method"])
			assert.Equal(t, "/api/v1/fleet/login", kv["uri"])
		case 1:
			assert.Equal(t, "debug", kv["level"])
			assert.Equal(t, "GET", kv["method"])
			assert.Equal(t, "/api/v1/fleet/config", kv["uri"])
			assert.Equal(t, "admin1@example.com", kv["user"])
		case 2:
			assert.Equal(t, "info", kv["level"])
			assert.Equal(t, "POST", kv["method"])
			assert.Equal(t, "/api/v1/fleet/queries", kv["uri"])
			assert.Equal(t, "admin1@example.com", kv["user"])
			assert.Equal(t, "somequery", kv["name"])
			assert.Equal(t, "select 1 from osquery;", kv["sql"])
		default:
			t.Fail()
		}
	}
}

func TestVulnerableSoftware(t *testing.T) {
	ds := mysql.CreateMySQLDS(t)
	defer ds.Close()

	_, server := RunServerForTestsWithDS(t, ds)
	token := getTestAdminToken(t, server)

	host, err := ds.NewHost(&fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         "1",
		UUID:            "1",
		Hostname:        "foo.local",
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-58",
	})
	require.NoError(t, err)
	require.NotNil(t, host)

	soft := fleet.HostSoftware{
		Modified: true,
		Software: []fleet.Software{
			{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
			{Name: "bar", Version: "0.0.3", Source: "apps"},
		},
	}
	host.HostSoftware = soft
	require.NoError(t, ds.SaveHostSoftware(host))
	require.NoError(t, ds.LoadHostSoftware(host))

	soft1 := host.Software[0]
	if soft1.Name != "bar" {
		soft1 = host.Software[1]
	}

	require.NoError(t, ds.AddCPEForSoftware(soft1, "somecpe"))
	require.NoError(t, ds.InsertCVEForCPE("cve-123-123-132", []string{"somecpe"}))

	path := fmt.Sprintf("/api/v1/fleet/hosts/%d", host.ID)
	resp := doReq(t, nil, "GET", server, path, token, http.StatusOK)
	defer resp.Body.Close()
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	expectedJSONSoft2 := `"name": "bar",
        "version": "0.0.3",
        "source": "apps",
        "generated_cpe": "somecpe",
        "vulnerabilities": [
          {
            "cve": "cve-123-123-132",
            "details_link": "https://nvd.nist.gov/vuln/detail/cve-123-123-132"
          }
        ]`
	expectedJSONSoft1 := `"name": "foo",
        "version": "0.0.1",
        "source": "chrome_extensions",
        "generated_cpe": "",
        "vulnerabilities": null`
	// We are doing Contains instead of equals to test the output for software in particular
	// ignoring other things like timestamps and things that are outside the cope of this ticket
	assert.Contains(t, string(bodyBytes), expectedJSONSoft2)
	assert.Contains(t, string(bodyBytes), expectedJSONSoft1)
}
