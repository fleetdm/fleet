package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type abstractIntegrationDSTestSuite struct {
	suite.Suite

	ds *mysql.Datastore
}

type integrationDSTestSuite struct {
	abstractIntegrationDSTestSuite
}

func TestIntegrationDSTestSuite(t *testing.T) {
	suite.Run(t, new(integrationDSTestSuite))
}

func (s *abstractIntegrationDSTestSuite) SetupSuite() {
	s.ds = mysql.CreateMySQLDS(s.T())
	test.AddAllHostsLabel(s.T(), s.ds)
}

func (s *abstractIntegrationDSTestSuite) TearDownSuite() {
	s.ds.Close()
}

type abstractIntegrationTestSuite struct {
	abstractIntegrationDSTestSuite

	server *httptest.Server
	users  map[string]fleet.User
	token  string
}

type integrationTestSuite struct {
	abstractIntegrationTestSuite
}

func (s *integrationTestSuite) SetupSuite() {
	t := s.T()

	// need to call in every setup suite because the name otherwise doesn't work
	s.ds = mysql.CreateMySQLDS(t)
	test.AddAllHostsLabel(t, s.ds)

	users, server := RunServerForTestsWithDS(s.T(), s.ds)
	s.server = server
	s.users = users
	s.token = getTestAdminToken(t, s.server)
}

func (s *integrationTestSuite) TearDownSuite() {
	s.abstractIntegrationDSTestSuite.TearDownSuite()
	s.server.Close()
}

func (s *abstractIntegrationTestSuite) Do(verb, path string, params interface{}, expectedStatusCode int) *http.Response {
	t := s.T()

	j, err := json.Marshal(params)
	require.NoError(t, err)

	resp := s.DoRaw(verb, path, j, expectedStatusCode)

	return resp
}

func (s *abstractIntegrationTestSuite) DoRaw(verb string, path string, rawBytes []byte, expectedStatusCode int) *http.Response {
	t := s.T()

	requestBody := &nopCloser{bytes.NewBuffer(rawBytes)}
	req, _ := http.NewRequest(verb, s.server.URL+path, requestBody)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", s.token))
	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err)
	require.Equal(t, expectedStatusCode, resp.StatusCode)
	return resp
}

func (s *abstractIntegrationTestSuite) DoJSON(verb, path string, params interface{}, expectedStatusCode int, v interface{}) {
	resp := s.Do(verb, path, params, expectedStatusCode)
	defer resp.Body.Close()
	err := json.NewDecoder(resp.Body).Decode(v)
	require.NoError(s.T(), err)
	if e, ok := v.(errorer); ok {
		require.NoError(s.T(), e.error())
	}
}

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

func (s *integrationTestSuite) applyConfig(spec []byte) {
	var appConfigSpec interface{}
	err := yaml.Unmarshal(spec, &appConfigSpec)
	require.NoError(s.T(), err)

	resp := s.Do("PATCH", "/api/v1/fleet/config", appConfigSpec, http.StatusOK)
	resp.Body.Close()
}

func (s *abstractIntegrationTestSuite) getConfig() *appConfigResponse {
	var responseBody *appConfigResponse
	s.DoJSON("GET", "/api/v1/fleet/config", nil, http.StatusOK, &responseBody)
	return responseBody
}

func TestIntegrations(t *testing.T) {
	suite.Run(t, new(integrationTestSuite))
}

func (s *integrationTestSuite) TestDoubleUserCreationErrors() {
	t := s.T()

	params := fleet.UserPayload{
		Name:       ptr.String("user1"),
		Email:      ptr.String("email@asd.com"),
		Password:   ptr.String("pass"),
		GlobalRole: ptr.String(fleet.RoleObserver),
	}

	respFirst := s.Do("POST", "/api/v1/fleet/users/admin", &params, http.StatusOK)
	defer respFirst.Body.Close()
	respSecond := s.Do("POST", "/api/v1/fleet/users/admin", &params, http.StatusConflict)
	defer respSecond.Body.Close()

	assertBodyContains(t, respSecond, `Error 1062: Duplicate entry 'email@asd.com'`)
}

func (s *integrationTestSuite) TestUserWithoutRoleErrors() {
	t := s.T()

	params := fleet.UserPayload{
		Name:     ptr.String("user1"),
		Email:    ptr.String("email@asd.com"),
		Password: ptr.String("pass"),
	}

	resp := s.Do("POST", "/api/v1/fleet/users/admin", &params, http.StatusUnprocessableEntity)
	defer resp.Body.Close()
	assertErrorCodeAndMessage(t, resp, fleet.ErrNoRoleNeeded, "either global role or team role needs to be defined")
}

func (s *integrationTestSuite) TestUserWithWrongRoleErrors() {
	t := s.T()

	params := fleet.UserPayload{
		Name:       ptr.String("user1"),
		Email:      ptr.String("email@asd.com"),
		Password:   ptr.String("pass"),
		GlobalRole: ptr.String("wrongrole"),
	}
	resp := s.Do("POST", "/api/v1/fleet/users/admin", &params, http.StatusUnprocessableEntity)
	defer resp.Body.Close()
	assertErrorCodeAndMessage(t, resp, fleet.ErrNoRoleNeeded, "GlobalRole role can only be admin, observer, or maintainer.")
}

func (s *integrationTestSuite) TestUserCreationWrongTeamErrors() {
	t := s.T()

	teams := []fleet.UserTeam{
		{
			Team: fleet.Team{
				ID: 9999,
			},
			Role: fleet.RoleObserver,
		},
	}

	params := fleet.UserPayload{
		Name:     ptr.String("user2"),
		Email:    ptr.String("email2@asd.com"),
		Password: ptr.String("pass"),
		Teams:    &teams,
	}
	resp := s.Do("POST", "/api/v1/fleet/users/admin", &params, http.StatusUnprocessableEntity)
	defer resp.Body.Close()
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
) (*http.Response, func()) {
	j, err := json.Marshal(&params)
	assert.Nil(t, err)

	requestBody := &nopCloser{bytes.NewBuffer(j)}
	req, _ := http.NewRequest(method, server.URL+path, requestBody)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	client := &http.Client{}
	resp, err := client.Do(req)
	require.Nil(t, err)
	assert.Equal(t, expectedStatusCode, resp.StatusCode)
	return resp, func() {
		thisResp := resp
		thisResp.Body.Close()
	}
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

func assertBodyContains(t *testing.T, resp *http.Response, expectedError string) {
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	bodyString := string(bodyBytes)
	assert.Contains(t, bodyString, expectedError)
}

func (s *integrationTestSuite) TestQueryCreationLogsActivity() {
	t := s.T()

	admin1 := s.users["admin1@example.com"]
	admin1.GravatarURL = "http://iii.com"
	err := s.ds.SaveUser(&admin1)
	require.NoError(t, err)

	params := fleet.QueryPayload{
		Name:  ptr.String("user1"),
		Query: ptr.String("select * from time;"),
	}
	resp := s.Do("POST", "/api/v1/fleet/queries", &params, http.StatusOK)
	defer resp.Body.Close()

	activities := listActivitiesResponse{}
	s.DoJSON("GET", "/api/v1/fleet/activities", nil, http.StatusOK, &activities)

	assert.Len(t, activities.Activities, 1)
	assert.Equal(t, "Test Name admin1@example.com", activities.Activities[0].ActorFullName)
	require.NotNil(t, activities.Activities[0].ActorGravatar)
	assert.Equal(t, "http://iii.com", *activities.Activities[0].ActorGravatar)
	assert.Equal(t, "created_saved_query", activities.Activities[0].Type)
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

func (s *integrationTestSuite) TestAppConfigAdditionalQueriesCanBeRemoved() {
	t := s.T()

	spec := []byte(`
  host_expiry_settings:
    host_expiry_enabled: true
    host_expiry_window: 0
  host_settings:
    additional_queries:
      time: SELECT * FROM time
    enable_host_users: true
`)
	s.applyConfig(spec)

	spec = []byte(`
  host_settings:
    enable_host_users: true
    additional_queries: null
`)
	s.applyConfig(spec)

	config := s.getConfig()
	assert.Nil(t, config.HostSettings.AdditionalQueries)
	assert.True(t, config.HostExpirySettings.HostExpiryEnabled)
}

func (s *integrationTestSuite) TestAppConfigDefaultValues() {
	config := s.getConfig()
	s.Run("Update interval", func() {
		require.Equal(s.T(), 1*time.Hour, config.UpdateInterval.OSQueryDetail)
	})

	s.Run("has logging", func() {
		require.NotNil(s.T(), config.Logging)
	})
}

func (s *integrationTestSuite) TestUserRolesSpec() {
	t := s.T()

	_, err := s.ds.NewTeam(&fleet.Team{
		ID:          42,
		Name:        "team1",
		Description: "desc team1",
	})
	require.NoError(t, err)

	email := t.Name() + "@asd.com"
	u := &fleet.User{
		Password:    []byte("asd"),
		Name:        t.Name(),
		Email:       email,
		GravatarURL: "http://asd.com",
		GlobalRole:  ptr.String(fleet.RoleObserver),
	}
	user, err := s.ds.NewUser(u)
	require.NoError(t, err)
	assert.Len(t, user.Teams, 0)

	spec := []byte(fmt.Sprintf(`
  roles:
    %s:
      global_role: null
      teams:
      - role: maintainer
        team: team1
`,
		email))

	var userRoleSpec applyUserRoleSpecsRequest
	err = yaml.Unmarshal(spec, &userRoleSpec.Spec)
	require.NoError(t, err)

	resp := s.Do("POST", "/api/v1/fleet/users/roles/spec", &userRoleSpec, http.StatusOK)
	defer resp.Body.Close()

	user, err = s.ds.UserByEmail(email)
	require.NoError(t, err)
	require.Len(t, user.Teams, 1)
	assert.Equal(t, fleet.RoleMaintainer, user.Teams[0].Role)
}

func (s *integrationTestSuite) TestGlobalSchedule() {
	t := s.T()

	gs := fleet.GlobalSchedulePayload{}
	s.DoJSON("GET", "/api/v1/fleet/global/schedule", nil, http.StatusOK, &gs)
	require.Len(t, gs.GlobalSchedule, 0)

	qr, err := s.ds.NewQuery(&fleet.Query{
		Name:           "TestQuery1",
		Description:    "Some description",
		Query:          "select * from osquery;",
		ObserverCanRun: true,
	})
	require.NoError(t, err)

	gsParams := fleet.ScheduledQueryPayload{QueryID: ptr.Uint(qr.ID), Interval: ptr.Uint(42)}
	r := globalScheduleQueryResponse{}
	s.DoJSON("POST", "/api/v1/fleet/global/schedule", gsParams, http.StatusOK, &r)

	gs = fleet.GlobalSchedulePayload{}
	s.DoJSON("GET", "/api/v1/fleet/global/schedule", nil, http.StatusOK, &gs)
	require.Len(t, gs.GlobalSchedule, 1)
	assert.Equal(t, uint(42), gs.GlobalSchedule[0].Interval)
	assert.Equal(t, "TestQuery1", gs.GlobalSchedule[0].Name)
	id := gs.GlobalSchedule[0].ID

	gs = fleet.GlobalSchedulePayload{}
	gsParams = fleet.ScheduledQueryPayload{Interval: ptr.Uint(55)}
	s.DoJSON("PATCH", fmt.Sprintf("/api/v1/fleet/global/schedule/%d", id), gsParams, http.StatusOK, &gs)

	gs = fleet.GlobalSchedulePayload{}
	s.DoJSON("GET", "/api/v1/fleet/global/schedule", nil, http.StatusOK, &gs)
	require.Len(t, gs.GlobalSchedule, 1)
	assert.Equal(t, uint(55), gs.GlobalSchedule[0].Interval)

	r = globalScheduleQueryResponse{}
	s.DoJSON("DELETE", fmt.Sprintf("/api/v1/fleet/global/schedule/%d", id), nil, http.StatusOK, &r)

	gs = fleet.GlobalSchedulePayload{}
	s.DoJSON("GET", "/api/v1/fleet/global/schedule", nil, http.StatusOK, &gs)
	require.Len(t, gs.GlobalSchedule, 0)
}

func (s *integrationTestSuite) TestTranslator() {
	t := s.T()

	payload := translatorResponse{}
	params := translatorRequest{List: []fleet.TranslatePayload{
		{
			Type:    fleet.TranslatorTypeUserEmail,
			Payload: fleet.StringIdentifierToIDPayload{Identifier: "admin1@example.com"},
		},
	}}
	s.DoJSON("POST", "/api/v1/fleet/translate", &params, http.StatusOK, &payload)
	require.Len(t, payload.List, 1)

	assert.Equal(t, s.users[payload.List[0].Payload.Identifier].ID, payload.List[0].Payload.ID)
}

func (s *integrationTestSuite) TestVulnerableSoftware() {
	t := s.T()

	host, err := s.ds.NewHost(&fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         t.Name() + "1",
		UUID:            t.Name() + "1",
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
	require.NoError(t, s.ds.SaveHostSoftware(host))
	require.NoError(t, s.ds.LoadHostSoftware(host))

	soft1 := host.Software[0]
	if soft1.Name != "bar" {
		soft1 = host.Software[1]
	}

	require.NoError(t, s.ds.AddCPEForSoftware(soft1, "somecpe"))
	require.NoError(t, s.ds.InsertCVEForCPE("cve-123-123-132", []string{"somecpe"}))

	resp := s.Do("GET", fmt.Sprintf("/api/v1/fleet/hosts/%d", host.ID), nil, http.StatusOK)
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

func (s *integrationTestSuite) TestGlobalPolicies() {
	t := s.T()

	for i := 0; i < 3; i++ {
		_, err := s.ds.NewHost(&fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			SeenTime:        time.Now().Add(-time.Duration(i) * time.Minute),
			OsqueryHostID:   strconv.Itoa(i),
			NodeKey:         fmt.Sprintf("%d", i),
			UUID:            fmt.Sprintf("%d", i),
			Hostname:        fmt.Sprintf("foo.local%d", i),
		})
		require.NoError(t, err)
	}

	qr, err := s.ds.NewQuery(&fleet.Query{
		Name:           "TestQuery3",
		Description:    "Some description",
		Query:          "select * from osquery;",
		ObserverCanRun: true,
	})
	require.NoError(t, err)

	gpParams := globalPolicyRequest{QueryID: qr.ID}
	gpResp := globalPolicyResponse{}
	s.DoJSON("POST", "/api/v1/fleet/global/policies", gpParams, http.StatusOK, &gpResp)
	require.NotNil(t, gpResp.Policy)
	assert.Equal(t, qr.ID, gpResp.Policy.QueryID)

	policiesResponse := listGlobalPoliciesResponse{}
	s.DoJSON("GET", "/api/v1/fleet/global/policies", nil, http.StatusOK, &policiesResponse)
	require.Len(t, policiesResponse.Policies, 1)
	assert.Equal(t, qr.ID, policiesResponse.Policies[0].QueryID)

	singlePolicyResponse := getPolicyByIDResponse{}
	singlePolicyURL := fmt.Sprintf("/api/v1/fleet/global/policies/%d", policiesResponse.Policies[0].ID)
	s.DoJSON("GET", singlePolicyURL, nil, http.StatusOK, &singlePolicyResponse)
	assert.Equal(t, qr.ID, singlePolicyResponse.Policy.QueryID)
	assert.Equal(t, qr.Name, singlePolicyResponse.Policy.QueryName)

	listHostsURL := fmt.Sprintf("/api/v1/fleet/hosts?policy_id=%d", policiesResponse.Policies[0].ID)
	listHostsResp := listHostsResponse{}
	s.DoJSON("GET", listHostsURL, nil, http.StatusOK, &listHostsResp)
	require.Len(t, listHostsResp.Hosts, 3)

	h1 := listHostsResp.Hosts[0]
	h2 := listHostsResp.Hosts[1]

	listHostsURL = fmt.Sprintf("/api/v1/fleet/hosts?policy_id=%d&policy_response=passing", policiesResponse.Policies[0].ID)
	listHostsResp = listHostsResponse{}
	s.DoJSON("GET", listHostsURL, nil, http.StatusOK, &listHostsResp)
	require.Len(t, listHostsResp.Hosts, 0)

	require.NoError(t, s.ds.RecordPolicyQueryExecutions(h1.Host, map[uint]*bool{policiesResponse.Policies[0].ID: ptr.Bool(true)}, time.Now()))
	require.NoError(t, s.ds.RecordPolicyQueryExecutions(h2.Host, map[uint]*bool{policiesResponse.Policies[0].ID: nil}, time.Now()))

	listHostsURL = fmt.Sprintf("/api/v1/fleet/hosts?policy_id=%d&policy_response=passing", policiesResponse.Policies[0].ID)
	listHostsResp = listHostsResponse{}
	s.DoJSON("GET", listHostsURL, nil, http.StatusOK, &listHostsResp)
	require.Len(t, listHostsResp.Hosts, 1)

	deletePolicyParams := deleteGlobalPoliciesRequest{IDs: []uint{policiesResponse.Policies[0].ID}}
	deletePolicyResp := deleteGlobalPoliciesResponse{}
	s.DoJSON("POST", "/api/v1/fleet/global/policies/delete", deletePolicyParams, http.StatusOK, &deletePolicyResp)

	policiesResponse = listGlobalPoliciesResponse{}
	s.DoJSON("GET", "/api/v1/fleet/global/policies", nil, http.StatusOK, &policiesResponse)
	require.Len(t, policiesResponse.Policies, 0)
}

//func (s *integrationDSTestSuite) TestLicenseExpiration() {
//	testCases := []struct {
//		name             string
//		tier             string
//		expiration       time.Time
//		shouldHaveHeader bool
//	}{
//		{"basic expired", fleet.TierBasic, time.Now().Add(-24 * time.Hour), true},
//		{"basic not expired", fleet.TierBasic, time.Now().Add(24 * time.Hour), false},
//		{"core expired", fleet.TierCore, time.Now().Add(-24 * time.Hour), false},
//		{"core not expired", fleet.TierCore, time.Now().Add(24 * time.Hour), false},
//	}
//
//	createTestUsers(s.T(), s.ds)
//	for _, tt := range testCases {
//		s.Run(tt.name, func() {
//			t := s.T()
//
//			license := &fleet.LicenseInfo{Tier: tt.tier, Expiration: tt.expiration}
//			_, server := RunServerForTestsWithDS(t, s.ds, TestServerOpts{License: license, SkipCreateTestUsers: true})
//			defer server.Close()
//
//			token := getTestAdminToken(t, server)
//
//			resp, closeFunc := doReq(t, nil, "GET", server, "/api/v1/fleet/config", token, http.StatusOK)
//			defer closeFunc()
//			if tt.shouldHaveHeader {
//				require.Equal(t, fleet.HeaderLicenseValueExpired, resp.Header.Get(fleet.HeaderLicenseKey))
//			} else {
//				require.Equal(t, "", resp.Header.Get(fleet.HeaderLicenseKey))
//			}
//		})
//	}
//}
