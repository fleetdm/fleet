package service

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"gopkg.in/guregu/null.v3"
)

type integrationTestSuite struct {
	suite.Suite

	withServer
}

func (s *integrationTestSuite) SetupSuite() {
	s.withServer.SetupSuite("integrationTestSuite")
}

func (s *integrationTestSuite) TearDownTest() {
	u := s.users["admin1@example.com"]
	filter := fleet.TeamFilter{User: &u}
	hosts, _ := s.ds.ListHosts(context.Background(), filter, fleet.HostListOptions{})
	var ids []uint
	for _, host := range hosts {
		ids = append(ids, host.ID)
	}
	s.ds.DeleteHosts(context.Background(), ids)
}

func TestIntegrations(t *testing.T) {
	testingSuite := new(integrationTestSuite)
	testingSuite.s = &testingSuite.Suite
	suite.Run(t, testingSuite)
}

func (s *integrationTestSuite) TestDoubleUserCreationErrors() {
	t := s.T()

	params := fleet.UserPayload{
		Name:       ptr.String("user1"),
		Email:      ptr.String("email@asd.com"),
		Password:   ptr.String("pass"),
		GlobalRole: ptr.String(fleet.RoleObserver),
	}

	s.Do("POST", "/api/v1/fleet/users/admin", &params, http.StatusOK)
	respSecond := s.Do("POST", "/api/v1/fleet/users/admin", &params, http.StatusConflict)

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
	assertBodyContains(t, resp, `Error 1452: Cannot add or update a child row: a foreign key constraint fails`)
}

func (s *integrationTestSuite) TestQueryCreationLogsActivity() {
	t := s.T()

	admin1 := s.users["admin1@example.com"]
	admin1.GravatarURL = "http://iii.com"
	err := s.ds.SaveUser(context.Background(), &admin1)
	require.NoError(t, err)

	params := fleet.QueryPayload{
		Name:  ptr.String("user1"),
		Query: ptr.String("select * from time;"),
	}
	s.Do("POST", "/api/v1/fleet/queries", &params, http.StatusOK)

	activities := listActivitiesResponse{}
	s.DoJSON("GET", "/api/v1/fleet/activities", nil, http.StatusOK, &activities)

	assert.Len(t, activities.Activities, 1)
	assert.Equal(t, "Test Name admin1@example.com", activities.Activities[0].ActorFullName)
	require.NotNil(t, activities.Activities[0].ActorGravatar)
	assert.Equal(t, "http://iii.com", *activities.Activities[0].ActorGravatar)
	assert.Equal(t, "created_saved_query", activities.Activities[0].Type)
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

	_, err := s.ds.NewTeam(context.Background(), &fleet.Team{
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
	user, err := s.ds.NewUser(context.Background(), u)
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

	s.Do("POST", "/api/v1/fleet/users/roles/spec", &userRoleSpec, http.StatusOK)

	user, err = s.ds.UserByEmail(context.Background(), email)
	require.NoError(t, err)
	require.Len(t, user.Teams, 1)
	assert.Equal(t, fleet.RoleMaintainer, user.Teams[0].Role)
}

func (s *integrationTestSuite) TestGlobalSchedule() {
	t := s.T()

	gs := fleet.GlobalSchedulePayload{}
	s.DoJSON("GET", "/api/v1/fleet/global/schedule", nil, http.StatusOK, &gs)
	require.Len(t, gs.GlobalSchedule, 0)

	qr, err := s.ds.NewQuery(context.Background(), &fleet.Query{
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

	host, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         t.Name() + "1",
		UUID:            t.Name() + "1",
		Hostname:        t.Name() + "foo.local",
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
	require.NoError(t, s.ds.SaveHostSoftware(context.Background(), host))
	require.NoError(t, s.ds.LoadHostSoftware(context.Background(), host))

	soft1 := host.Software[0]
	if soft1.Name != "bar" {
		soft1 = host.Software[1]
	}

	require.NoError(t, s.ds.AddCPEForSoftware(context.Background(), soft1, "somecpe"))
	require.NoError(t, s.ds.InsertCVEForCPE(context.Background(), "cve-123-123-132", []string{"somecpe"}))

	resp := s.Do("GET", fmt.Sprintf("/api/v1/fleet/hosts/%d", host.ID), nil, http.StatusOK)
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

	lsReq := listSoftwareRequest{}
	lsResp := listSoftwareResponse{}
	s.DoJSON("GET", "/api/v1/fleet/software", lsReq, http.StatusOK, &lsResp, "vulnerable", "true", "order_key", "generated_cpe", "order_direction", "desc")
	assert.Len(t, lsResp.Software, 1)
	assert.Equal(t, soft1.ID, lsResp.Software[0].ID)
	assert.Len(t, lsResp.Software[0].Vulnerabilities, 1)
	assert.Equal(t, 1, lsResp.Software[0].HostCount)

	s.DoJSON("GET", "/api/v1/fleet/software", lsReq, http.StatusOK, &lsResp, "vulnerable", "true", "order_key", "host_count", "order_direction", "desc")
	assert.Len(t, lsResp.Software, 1)
	assert.Equal(t, soft1.ID, lsResp.Software[0].ID)
	assert.Len(t, lsResp.Software[0].Vulnerabilities, 1)
	assert.Equal(t, 1, lsResp.Software[0].HostCount)
}

func (s *integrationTestSuite) TestGlobalPolicies() {
	t := s.T()

	for i := 0; i < 3; i++ {
		_, err := s.ds.NewHost(context.Background(), &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now().Add(-time.Duration(i) * time.Minute),
			OsqueryHostID:   fmt.Sprintf("%s%d", t.Name(), i),
			NodeKey:         fmt.Sprintf("%s%d", t.Name(), i),
			UUID:            fmt.Sprintf("%s%d", t.Name(), i),
			Hostname:        fmt.Sprintf("%sfoo.local%d", t.Name(), i),
		})
		require.NoError(t, err)
	}

	qr, err := s.ds.NewQuery(context.Background(), &fleet.Query{
		Name:           "TestQuery3",
		Description:    "Some description",
		Query:          "select * from osquery;",
		ObserverCanRun: true,
	})
	require.NoError(t, err)

	gpParams := globalPolicyRequest{QueryID: qr.ID, Resolution: "some global resolution"}
	gpResp := globalPolicyResponse{}
	s.DoJSON("POST", "/api/v1/fleet/global/policies", gpParams, http.StatusOK, &gpResp)
	require.NotNil(t, gpResp.Policy)
	assert.Equal(t, qr.ID, gpResp.Policy.QueryID)
	require.NotNil(t, gpResp.Policy.Resolution)
	assert.Equal(t, "some global resolution", *gpResp.Policy.Resolution)

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

	require.NoError(t, s.ds.RecordPolicyQueryExecutions(context.Background(), h1.Host, map[uint]*bool{policiesResponse.Policies[0].ID: ptr.Bool(true)}, time.Now(), false))
	require.NoError(t, s.ds.RecordPolicyQueryExecutions(context.Background(), h2.Host, map[uint]*bool{policiesResponse.Policies[0].ID: nil}, time.Now(), false))

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

func (s *integrationTestSuite) TestBulkDeleteHostsFromTeam() {
	t := s.T()

	hosts := s.createHosts(t)

	team1, err := s.ds.NewTeam(context.Background(), &fleet.Team{Name: t.Name() + "team1"})
	require.NoError(t, err)

	require.NoError(t, s.ds.AddHostsToTeam(context.Background(), &team1.ID, []uint{hosts[0].ID}))

	req := deleteHostsRequest{
		Filters: struct {
			MatchQuery string           `json:"query"`
			Status     fleet.HostStatus `json:"status"`
			LabelID    *uint            `json:"label_id"`
			TeamID     *uint            `json:"team_id"`
		}{TeamID: ptr.Uint(team1.ID)},
	}
	resp := deleteHostsResponse{}
	s.DoJSON("POST", "/api/v1/fleet/hosts/delete", req, http.StatusOK, &resp)

	_, err = s.ds.Host(context.Background(), hosts[0].ID)
	require.Error(t, err)
	_, err = s.ds.Host(context.Background(), hosts[1].ID)
	require.NoError(t, err)
	_, err = s.ds.Host(context.Background(), hosts[2].ID)
	require.NoError(t, err)

	err = s.ds.DeleteHosts(context.Background(), []uint{hosts[1].ID, hosts[2].ID})
	require.NoError(t, err)
}

func (s *integrationTestSuite) TestBulkDeleteHostsInLabel() {
	t := s.T()

	hosts := s.createHosts(t)

	label := &fleet.Label{
		Name:  "foo",
		Query: "select * from foo;",
	}
	label, err := s.ds.NewLabel(context.Background(), label)
	require.NoError(t, err)

	require.NoError(t, s.ds.RecordLabelQueryExecutions(context.Background(), hosts[1], map[uint]*bool{label.ID: ptr.Bool(true)}, time.Now(), false))
	require.NoError(t, s.ds.RecordLabelQueryExecutions(context.Background(), hosts[2], map[uint]*bool{label.ID: ptr.Bool(true)}, time.Now(), false))

	req := deleteHostsRequest{
		Filters: struct {
			MatchQuery string           `json:"query"`
			Status     fleet.HostStatus `json:"status"`
			LabelID    *uint            `json:"label_id"`
			TeamID     *uint            `json:"team_id"`
		}{LabelID: ptr.Uint(label.ID)},
	}
	resp := deleteHostsResponse{}
	s.DoJSON("POST", "/api/v1/fleet/hosts/delete", req, http.StatusOK, &resp)

	_, err = s.ds.Host(context.Background(), hosts[0].ID)
	require.NoError(t, err)
	_, err = s.ds.Host(context.Background(), hosts[1].ID)
	require.Error(t, err)
	_, err = s.ds.Host(context.Background(), hosts[2].ID)
	require.Error(t, err)

	err = s.ds.DeleteHosts(context.Background(), []uint{hosts[0].ID})
	require.NoError(t, err)
}

func (s *integrationTestSuite) TestBulkDeleteHostByIDs() {
	t := s.T()

	hosts := s.createHosts(t)

	req := deleteHostsRequest{
		IDs: []uint{hosts[0].ID, hosts[1].ID},
	}
	resp := deleteHostsResponse{}
	s.DoJSON("POST", "/api/v1/fleet/hosts/delete", req, http.StatusOK, &resp)

	_, err := s.ds.Host(context.Background(), hosts[0].ID)
	require.Error(t, err)
	_, err = s.ds.Host(context.Background(), hosts[1].ID)
	require.Error(t, err)
	_, err = s.ds.Host(context.Background(), hosts[2].ID)
	require.NoError(t, err)

	err = s.ds.DeleteHosts(context.Background(), []uint{hosts[2].ID})
	require.NoError(t, err)
}

func (s *integrationTestSuite) createHosts(t *testing.T) []*fleet.Host {
	var hosts []*fleet.Host

	for i := 0; i < 3; i++ {
		host, err := s.ds.NewHost(context.Background(), &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now().Add(-time.Duration(i) * time.Minute),
			OsqueryHostID:   fmt.Sprintf("%s%d", t.Name(), i),
			NodeKey:         fmt.Sprintf("%s%d", t.Name(), i),
			UUID:            fmt.Sprintf("%s%d", t.Name(), i),
			Hostname:        fmt.Sprintf("%sfoo.local%d", t.Name(), i),
			Platform:        "linux",
		})
		require.NoError(t, err)
		hosts = append(hosts, host)
	}
	return hosts
}

func (s *integrationTestSuite) TestBulkDeleteHostsErrors() {
	t := s.T()

	hosts := s.createHosts(t)

	req := deleteHostsRequest{
		IDs: []uint{hosts[0].ID, hosts[1].ID},
		Filters: struct {
			MatchQuery string           `json:"query"`
			Status     fleet.HostStatus `json:"status"`
			LabelID    *uint            `json:"label_id"`
			TeamID     *uint            `json:"team_id"`
		}{LabelID: ptr.Uint(1)},
	}
	resp := deleteHostsResponse{}
	s.DoJSON("POST", "/api/v1/fleet/hosts/delete", req, http.StatusBadRequest, &resp)
}

func (s *integrationTestSuite) TestCountSoftware() {
	t := s.T()

	hosts := s.createHosts(t)

	label := &fleet.Label{
		Name:  t.Name() + "foo",
		Query: "select * from foo;",
	}
	label, err := s.ds.NewLabel(context.Background(), label)
	require.NoError(t, err)

	require.NoError(t, s.ds.RecordLabelQueryExecutions(context.Background(), hosts[0], map[uint]*bool{label.ID: ptr.Bool(true)}, time.Now(), false))

	req := countHostsRequest{}
	resp := countHostsResponse{}
	s.DoJSON(
		"GET", "/api/v1/fleet/hosts/count", req, http.StatusOK, &resp,
		"additional_info_filters", "*",
	)
	assert.Equal(t, 3, resp.Count)

	req = countHostsRequest{}
	resp = countHostsResponse{}
	s.DoJSON(
		"GET", "/api/v1/fleet/hosts/count", req, http.StatusOK, &resp,
		"additional_info_filters", "*",
		"label_id", fmt.Sprint(label.ID),
	)
	assert.Equal(t, 1, resp.Count)
}

func (s *integrationTestSuite) TestGetPack() {
	t := s.T()

	pack := &fleet.Pack{
		Name: t.Name(),
	}
	pack, err := s.ds.NewPack(context.Background(), pack)
	require.NoError(t, err)

	var packResp getPackResponse
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/packs/%d", pack.ID), nil, http.StatusOK, &packResp)
	require.Equal(t, packResp.Pack.ID, pack.ID)

	s.Do("GET", fmt.Sprintf("/api/v1/fleet/packs/%d", pack.ID+1), nil, http.StatusNotFound)
}

func (s *integrationTestSuite) TestListHosts() {
	t := s.T()

	hosts := s.createHosts(t)

	var resp listHostsResponse
	s.DoJSON("GET", "/api/v1/fleet/hosts", nil, http.StatusOK, &resp)
	require.Len(t, resp.Hosts, len(hosts))

	s.DoJSON("GET", "/api/v1/fleet/hosts", nil, http.StatusOK, &resp, "per_page", "1")
	require.Len(t, resp.Hosts, 1)
	assert.Nil(t, resp.Software)

	host := hosts[2]
	host.HostSoftware = fleet.HostSoftware{
		Modified: true,
		Software: []fleet.Software{
			{Name: "foo", Version: "0.0.2", Source: "chrome_extensions"},
		},
	}
	require.NoError(t, s.ds.SaveHostSoftware(context.Background(), host))
	require.NoError(t, s.ds.LoadHostSoftware(context.Background(), host))

	s.DoJSON("GET", "/api/v1/fleet/hosts", nil, http.StatusOK, &resp, "software_id", fmt.Sprint(host.Software[0].ID))
	require.Len(t, resp.Hosts, 1)
	assert.Equal(t, host.ID, resp.Hosts[0].ID)
	assert.Equal(t, "foo", resp.Software.Name)

	q := test.NewQuery(t, s.ds, "query1", "select 1", 0, true)
	p, err := s.ds.NewGlobalPolicy(context.Background(), q.ID, "")
	require.NoError(t, err)

	require.NoError(t, s.ds.RecordPolicyQueryExecutions(context.Background(), host, map[uint]*bool{p.ID: ptr.Bool(false)}, time.Now(), false))

	s.DoJSON("GET", "/api/v1/fleet/hosts", nil, http.StatusOK, &resp, "software_id", fmt.Sprint(host.Software[0].ID))
	require.Len(t, resp.Hosts, 1)
	assert.Equal(t, 1, resp.Hosts[0].HostIssues.FailingPoliciesCount)
	assert.Equal(t, 1, resp.Hosts[0].HostIssues.TotalIssuesCount)
}

func (s *integrationTestSuite) TestInvites() {
	t := s.T()

	team, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		Name:        t.Name() + "team1",
		Description: "desc team1",
	})
	require.NoError(t, err)

	createInviteReq := createInviteRequest{
		payload: fleet.InvitePayload{
			Email:      ptr.String("some email"),
			Name:       ptr.String("some name"),
			Position:   nil,
			SSOEnabled: nil,
			GlobalRole: null.StringFrom(fleet.RoleAdmin),
			Teams:      nil,
		},
	}
	createInviteResp := createInviteResponse{}
	s.DoJSON("POST", "/api/v1/fleet/invites", createInviteReq.payload, http.StatusOK, &createInviteResp)
	require.NotNil(t, createInviteResp.Invite)
	require.NotZero(t, createInviteResp.Invite.ID)

	updateInviteReq := updateInviteRequest{
		InvitePayload: fleet.InvitePayload{
			Teams: []fleet.UserTeam{
				{
					Team: fleet.Team{ID: team.ID},
					Role: fleet.RoleObserver,
				},
			},
		},
	}
	updateInviteResp := updateInviteResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/v1/fleet/invites/%d", createInviteResp.Invite.ID), updateInviteReq, http.StatusOK, &updateInviteResp)

	verify, err := s.ds.Invite(context.Background(), createInviteResp.Invite.ID)
	require.NoError(t, err)
	require.Equal(t, "", verify.GlobalRole.String)
	require.Len(t, verify.Teams, 1)
	assert.Equal(t, team.ID, verify.Teams[0].ID)
}

func (s *integrationTestSuite) TestGetHostSummary() {
	t := s.T()

	hosts := s.createHosts(t)

	team1, err := s.ds.NewTeam(context.Background(), &fleet.Team{Name: t.Name() + "team1"})
	require.NoError(t, err)
	team2, err := s.ds.NewTeam(context.Background(), &fleet.Team{Name: t.Name() + "team2"})
	require.NoError(t, err)

	require.NoError(t, s.ds.AddHostsToTeam(context.Background(), &team1.ID, []uint{hosts[0].ID}))

	var resp getHostSummaryResponse

	// no team filter
	s.DoJSON("GET", "/api/v1/fleet/host_summary", nil, http.StatusOK, &resp)
	require.Equal(t, resp.TotalsHostsCount, uint(len(hosts)))
	require.Len(t, resp.Platforms, 1)
	require.Equal(t, "linux", resp.Platforms[0].Platform)
	require.Equal(t, uint(len(hosts)), resp.Platforms[0].HostsCount)
	require.Nil(t, resp.TeamID)

	// team filter, no host
	s.DoJSON("GET", "/api/v1/fleet/host_summary", nil, http.StatusOK, &resp, "team_id", fmt.Sprint(team2.ID))
	require.Equal(t, resp.TotalsHostsCount, uint(0))
	require.Len(t, resp.Platforms, 0)
	require.Equal(t, team2.ID, *resp.TeamID)

	// team filter, one host
	s.DoJSON("GET", "/api/v1/fleet/host_summary", nil, http.StatusOK, &resp, "team_id", fmt.Sprint(team1.ID))
	require.Equal(t, resp.TotalsHostsCount, uint(1))
	require.Len(t, resp.Platforms, 1)
	require.Equal(t, "linux", resp.Platforms[0].Platform)
	require.Equal(t, uint(1), resp.Platforms[0].HostsCount)
	require.Equal(t, team1.ID, *resp.TeamID)
}
