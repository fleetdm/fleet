package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/ghodss/yaml"
	"github.com/google/uuid"
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
	t := s.T()
	ctx := context.Background()

	u := s.users["admin1@example.com"]
	filter := fleet.TeamFilter{User: &u}
	hosts, err := s.ds.ListHosts(ctx, filter, fleet.HostListOptions{})
	require.NoError(t, err)
	var ids []uint
	for _, host := range hosts {
		ids = append(ids, host.ID)
		require.NoError(t, s.ds.UpdateHostSoftware(context.Background(), host.ID, nil))
	}
	if len(ids) > 0 {
		require.NoError(t, s.ds.DeleteHosts(ctx, ids))
	}

	// recalculate software counts will remove the software entries
	require.NoError(t, s.ds.CalculateHostsPerSoftware(context.Background(), time.Now()))

	lbls, err := s.ds.ListLabels(ctx, fleet.TeamFilter{}, fleet.ListOptions{})
	require.NoError(t, err)
	for _, lbl := range lbls {
		if lbl.LabelType != fleet.LabelTypeBuiltIn {
			err := s.ds.DeleteLabel(ctx, lbl.Name)
			require.NoError(t, err)
		}
	}

	users, err := s.ds.ListUsers(ctx, fleet.UserListOptions{})
	require.NoError(t, err)
	for _, u := range users {
		if _, ok := s.users[u.Email]; !ok {
			err := s.ds.DeleteUser(ctx, u.ID)
			require.NoError(t, err)
		}
	}

	globalPolicies, err := s.ds.ListGlobalPolicies(ctx)
	require.NoError(t, err)
	if len(globalPolicies) > 0 {
		var globalPolicyIDs []uint
		for _, gp := range globalPolicies {
			globalPolicyIDs = append(globalPolicyIDs, gp.ID)
		}
		_, err = s.ds.DeleteGlobalPolicies(ctx, globalPolicyIDs)
		require.NoError(t, err)
	}

	// CalculateHostsPerSoftware performs a cleanup.
	err = s.ds.CalculateHostsPerSoftware(ctx, time.Now())
	require.NoError(t, err)
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
	var createQueryResp createQueryResponse
	s.DoJSON("POST", "/api/v1/fleet/queries", &params, http.StatusOK, &createQueryResp)
	defer cleanupQuery(s, createQueryResp.Query.ID)

	activities := listActivitiesResponse{}
	s.DoJSON("GET", "/api/v1/fleet/activities", nil, http.StatusOK, &activities)

	assert.GreaterOrEqual(t, len(activities.Activities), 1)
	found := false
	for _, activity := range activities.Activities {
		if activity.Type == "created_saved_query" {
			found = true
			assert.Equal(t, "Test Name admin1@example.com", activity.ActorFullName)
			require.NotNil(t, activity.ActorGravatar)
			assert.Equal(t, "http://iii.com", *activity.ActorGravatar)
		}
	}
	require.True(t, found)
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

	// list the existing global schedules (none yet)
	gs := fleet.GlobalSchedulePayload{}
	s.DoJSON("GET", "/api/v1/fleet/global/schedule", nil, http.StatusOK, &gs)
	require.Len(t, gs.GlobalSchedule, 0)

	// create a query that can be scheduled
	qr, err := s.ds.NewQuery(context.Background(), &fleet.Query{
		Name:           "TestQuery1",
		Description:    "Some description",
		Query:          "select * from osquery;",
		ObserverCanRun: true,
	})
	require.NoError(t, err)

	// schedule that query
	gsParams := fleet.ScheduledQueryPayload{QueryID: ptr.Uint(qr.ID), Interval: ptr.Uint(42)}
	r := globalScheduleQueryResponse{}
	s.DoJSON("POST", "/api/v1/fleet/global/schedule", gsParams, http.StatusOK, &r)

	// list the scheduled queries, get the one just created
	gs = fleet.GlobalSchedulePayload{}
	s.DoJSON("GET", "/api/v1/fleet/global/schedule", nil, http.StatusOK, &gs)
	require.Len(t, gs.GlobalSchedule, 1)
	assert.Equal(t, uint(42), gs.GlobalSchedule[0].Interval)
	assert.Equal(t, "TestQuery1", gs.GlobalSchedule[0].Name)
	id := gs.GlobalSchedule[0].ID

	// list page 2, should be empty
	s.DoJSON("GET", "/api/v1/fleet/global/schedule", nil, http.StatusOK, &gs, "page", "2", "per_page", "4")
	require.Len(t, gs.GlobalSchedule, 0)

	// update the scheduled query
	gs = fleet.GlobalSchedulePayload{}
	gsParams = fleet.ScheduledQueryPayload{Interval: ptr.Uint(55)}
	s.DoJSON("PATCH", fmt.Sprintf("/api/v1/fleet/global/schedule/%d", id), gsParams, http.StatusOK, &gs)

	// update a non-existing schedule
	gsParams = fleet.ScheduledQueryPayload{Interval: ptr.Uint(66)}
	s.DoJSON("PATCH", fmt.Sprintf("/api/v1/fleet/global/schedule/%d", id+1), gsParams, http.StatusNotFound, &gs)

	// read back that updated scheduled query
	gs = fleet.GlobalSchedulePayload{}
	s.DoJSON("GET", "/api/v1/fleet/global/schedule", nil, http.StatusOK, &gs)
	require.Len(t, gs.GlobalSchedule, 1)
	assert.Equal(t, id, gs.GlobalSchedule[0].ID)
	assert.Equal(t, uint(55), gs.GlobalSchedule[0].Interval)

	// delete the scheduled query
	r = globalScheduleQueryResponse{}
	s.DoJSON("DELETE", fmt.Sprintf("/api/v1/fleet/global/schedule/%d", id), nil, http.StatusOK, &r)

	// delete a non-existing schedule
	s.DoJSON("DELETE", fmt.Sprintf("/api/v1/fleet/global/schedule/%d", id+1), nil, http.StatusNotFound, &r)

	// list the scheduled queries, back to none
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

	software := []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
		{Name: "bar", Version: "0.0.3", Source: "apps"},
		{Name: "baz", Version: "0.0.4", Source: "apps"},
	}
	require.NoError(t, s.ds.UpdateHostSoftware(context.Background(), host.ID, software))
	require.NoError(t, s.ds.LoadHostSoftware(context.Background(), host))

	soft1 := host.Software[0]
	if soft1.Name != "bar" {
		soft1 = host.Software[1]
	}

	require.NoError(t, s.ds.AddCPEForSoftware(context.Background(), soft1, "somecpe"))
	_, err = s.ds.InsertCVEForCPE(context.Background(), "cve-123-123-132", []string{"somecpe"})
	require.NoError(t, err)

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

	countReq := countSoftwareRequest{}
	countResp := countSoftwareResponse{}
	s.DoJSON("GET", "/api/v1/fleet/software/count", countReq, http.StatusOK, &countResp)
	assert.Equal(t, 3, countResp.Count)

	// no software host counts have been calculated yet, so this returns nothing
	var lsResp listSoftwareResponse
	resp = s.Do("GET", "/api/v1/fleet/software", nil, http.StatusOK, "vulnerable", "true", "order_key", "generated_cpe", "order_direction", "desc")
	bodyBytes, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(bodyBytes), `"counts_updated_at": null`)

	require.NoError(t, json.Unmarshal(bodyBytes, &lsResp))
	require.Len(t, lsResp.Software, 0)
	assert.Nil(t, lsResp.CountsUpdatedAt)

	// the software/count endpoint is different, it doesn't care about hosts counts
	countResp = countSoftwareResponse{}
	s.DoJSON("GET", "/api/v1/fleet/software/count", countReq, http.StatusOK, &countResp, "vulnerable", "true", "order_key", "generated_cpe", "order_direction", "desc")
	assert.Equal(t, 1, countResp.Count)

	// calculate hosts counts
	hostsCountTs := time.Now().UTC()
	require.NoError(t, s.ds.CalculateHostsPerSoftware(context.Background(), hostsCountTs))

	// now the list software endpoint returns the software
	lsResp = listSoftwareResponse{}
	s.DoJSON("GET", "/api/v1/fleet/software", nil, http.StatusOK, &lsResp, "vulnerable", "true", "order_key", "generated_cpe", "order_direction", "desc")
	require.Len(t, lsResp.Software, 1)
	assert.Equal(t, soft1.ID, lsResp.Software[0].ID)
	assert.Len(t, lsResp.Software[0].Vulnerabilities, 1)
	require.NotNil(t, lsResp.CountsUpdatedAt)
	assert.WithinDuration(t, hostsCountTs, *lsResp.CountsUpdatedAt, time.Second)

	// the count endpoint still returns 1
	countResp = countSoftwareResponse{}
	s.DoJSON("GET", "/api/v1/fleet/software/count", countReq, http.StatusOK, &countResp, "vulnerable", "true", "order_key", "generated_cpe", "order_direction", "desc")
	assert.Equal(t, 1, countResp.Count)

	// default sort, not only vulnerable
	lsResp = listSoftwareResponse{}
	s.DoJSON("GET", "/api/v1/fleet/software", nil, http.StatusOK, &lsResp)
	require.True(t, len(lsResp.Software) >= len(software))
	require.NotNil(t, lsResp.CountsUpdatedAt)
	assert.WithinDuration(t, hostsCountTs, *lsResp.CountsUpdatedAt, time.Second)

	// request with a per_page limit (see #4058)
	lsResp = listSoftwareResponse{}
	s.DoJSON("GET", "/api/v1/fleet/software", nil, http.StatusOK, &lsResp, "page", "0", "per_page", "2", "order_key", "hosts_count", "order_direction", "desc")
	require.Len(t, lsResp.Software, 2)
	require.NotNil(t, lsResp.CountsUpdatedAt)
	assert.WithinDuration(t, hostsCountTs, *lsResp.CountsUpdatedAt, time.Second)

	// request next page, with per_page limit
	lsResp = listSoftwareResponse{}
	s.DoJSON("GET", "/api/v1/fleet/software", nil, http.StatusOK, &lsResp, "per_page", "2", "page", "1", "order_key", "hosts_count", "order_direction", "desc")
	require.Len(t, lsResp.Software, 1)
	require.NotNil(t, lsResp.CountsUpdatedAt)
	assert.WithinDuration(t, hostsCountTs, *lsResp.CountsUpdatedAt, time.Second)

	// request one past the last page
	lsResp = listSoftwareResponse{}
	s.DoJSON("GET", "/api/v1/fleet/software", nil, http.StatusOK, &lsResp, "per_page", "2", "page", "2", "order_key", "hosts_count", "order_direction", "desc")
	require.Len(t, lsResp.Software, 0)
	require.Nil(t, lsResp.CountsUpdatedAt)
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

	gpParams := globalPolicyRequest{
		QueryID:    &qr.ID,
		Resolution: "some global resolution",
	}
	gpResp := globalPolicyResponse{}
	s.DoJSON("POST", "/api/v1/fleet/global/policies", gpParams, http.StatusOK, &gpResp)
	require.NotNil(t, gpResp.Policy)
	assert.Equal(t, qr.Name, gpResp.Policy.Name)
	assert.Equal(t, qr.Query, gpResp.Policy.Query)
	assert.Equal(t, qr.Description, gpResp.Policy.Description)
	require.NotNil(t, gpResp.Policy.Resolution)
	assert.Equal(t, "some global resolution", *gpResp.Policy.Resolution)

	policiesResponse := listGlobalPoliciesResponse{}
	s.DoJSON("GET", "/api/v1/fleet/global/policies", nil, http.StatusOK, &policiesResponse)
	require.Len(t, policiesResponse.Policies, 1)
	assert.Equal(t, qr.Name, policiesResponse.Policies[0].Name)
	assert.Equal(t, qr.Query, policiesResponse.Policies[0].Query)
	assert.Equal(t, qr.Description, policiesResponse.Policies[0].Description)

	// Get an unexistent policy
	s.Do("GET", fmt.Sprintf("/api/v1/fleet/global/policies/%d", 9999), nil, http.StatusNotFound)

	singlePolicyResponse := getPolicyByIDResponse{}
	singlePolicyURL := fmt.Sprintf("/api/v1/fleet/global/policies/%d", policiesResponse.Policies[0].ID)
	s.DoJSON("GET", singlePolicyURL, nil, http.StatusOK, &singlePolicyResponse)
	assert.Equal(t, qr.Name, singlePolicyResponse.Policy.Name)
	assert.Equal(t, qr.Query, singlePolicyResponse.Policy.Query)
	assert.Equal(t, qr.Description, singlePolicyResponse.Policy.Description)

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

	p, err := s.ds.NewPack(context.Background(), &fleet.Pack{
		Name: t.Name(),
		Hosts: []fleet.Target{
			{
				Type:     fleet.TargetHost,
				TargetID: hosts[0].ID,
			},
		},
	})
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

	_, err = s.ds.Host(context.Background(), hosts[0].ID, false)
	require.Error(t, err)
	_, err = s.ds.Host(context.Background(), hosts[1].ID, false)
	require.NoError(t, err)
	_, err = s.ds.Host(context.Background(), hosts[2].ID, false)
	require.NoError(t, err)

	err = s.ds.DeleteHosts(context.Background(), []uint{hosts[1].ID, hosts[2].ID})
	require.NoError(t, err)

	newP, err := s.ds.Pack(context.Background(), p.ID)
	require.NoError(t, err)
	require.Empty(t, newP.Hosts)
	require.NoError(t, s.ds.DeletePack(context.Background(), newP.Name))
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

	_, err = s.ds.Host(context.Background(), hosts[0].ID, false)
	require.NoError(t, err)
	_, err = s.ds.Host(context.Background(), hosts[1].ID, false)
	require.Error(t, err)
	_, err = s.ds.Host(context.Background(), hosts[2].ID, false)
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

	_, err := s.ds.Host(context.Background(), hosts[0].ID, false)
	require.Error(t, err)
	_, err = s.ds.Host(context.Background(), hosts[1].ID, false)
	require.Error(t, err)
	_, err = s.ds.Host(context.Background(), hosts[2].ID, false)
	require.NoError(t, err)

	err = s.ds.DeleteHosts(context.Background(), []uint{hosts[2].ID})
	require.NoError(t, err)
}

func (s *integrationTestSuite) createHosts(t *testing.T) []*fleet.Host {
	var hosts []*fleet.Host

	platforms := []string{"debian", "rhel", "linux"}
	for i := 0; i < 3; i++ {
		host, err := s.ds.NewHost(context.Background(), &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now().Add(-time.Duration(i) * time.Minute),
			OsqueryHostID:   fmt.Sprintf("%s%d", t.Name(), i),
			NodeKey:         fmt.Sprintf("%s%d", t.Name(), i),
			UUID:            uuid.New().String(),
			Hostname:        fmt.Sprintf("%sfoo.local%d", t.Name(), i),
			Platform:        platforms[i],
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

func (s *integrationTestSuite) TestPacks() {
	t := s.T()

	var packResp getPackResponse
	// get non-existing pack
	s.Do("GET", "/api/v1/fleet/packs/999", nil, http.StatusNotFound)

	// create some packs
	packs := make([]fleet.Pack, 3)
	for i := range packs {
		req := &createPackRequest{
			PackPayload: fleet.PackPayload{
				Name: ptr.String(fmt.Sprintf("%s_%d", strings.ReplaceAll(t.Name(), "/", "_"), i)),
			},
		}

		var createResp createPackResponse
		s.DoJSON("POST", "/api/v1/fleet/packs", req, http.StatusOK, &createResp)
		packs[i] = createResp.Pack.Pack
	}

	// get existing pack
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/packs/%d", packs[0].ID), nil, http.StatusOK, &packResp)
	require.Equal(t, packs[0].ID, packResp.Pack.ID)

	// list packs
	var listResp listPacksResponse
	s.DoJSON("GET", "/api/v1/fleet/packs", nil, http.StatusOK, &listResp, "per_page", "2", "order_key", "name")
	require.Len(t, listResp.Packs, 2)
	assert.Equal(t, packs[0].ID, listResp.Packs[0].ID)
	assert.Equal(t, packs[1].ID, listResp.Packs[1].ID)

	// get page 1
	s.DoJSON("GET", "/api/v1/fleet/packs", nil, http.StatusOK, &listResp, "page", "1", "per_page", "2", "order_key", "name")
	require.Len(t, listResp.Packs, 1)
	assert.Equal(t, packs[2].ID, listResp.Packs[0].ID)

	// get page 2, empty
	s.DoJSON("GET", "/api/v1/fleet/packs", nil, http.StatusOK, &listResp, "page", "2", "per_page", "2", "order_key", "name")
	require.Len(t, listResp.Packs, 0)

	var delResp deletePackResponse
	// delete non-existing pack by name
	s.DoJSON("DELETE", fmt.Sprintf("/api/v1/fleet/packs/%s", "zzz"), nil, http.StatusNotFound, &delResp)

	// delete existing pack by name
	s.DoJSON("DELETE", fmt.Sprintf("/api/v1/fleet/packs/%s", url.PathEscape(packs[0].Name)), nil, http.StatusOK, &delResp)

	// delete non-existing pack by id
	var delIDResp deletePackByIDResponse
	s.DoJSON("DELETE", fmt.Sprintf("/api/v1/fleet/packs/id/%d", packs[2].ID+1), nil, http.StatusNotFound, &delIDResp)

	// delete existing pack by id
	s.DoJSON("DELETE", fmt.Sprintf("/api/v1/fleet/packs/id/%d", packs[1].ID), nil, http.StatusOK, &delIDResp)

	var modResp modifyPackResponse
	// modify non-existing pack
	req := &fleet.PackPayload{Name: ptr.String("updated_" + packs[2].Name)}
	s.DoJSON("PATCH", fmt.Sprintf("/api/v1/fleet/packs/%d", packs[2].ID+1), req, http.StatusNotFound, &modResp)

	// modify existing pack
	req = &fleet.PackPayload{Name: ptr.String("updated_" + packs[2].Name)}
	s.DoJSON("PATCH", fmt.Sprintf("/api/v1/fleet/packs/%d", packs[2].ID), req, http.StatusOK, &modResp)
	require.Equal(t, packs[2].ID, modResp.Pack.ID)
	require.Contains(t, modResp.Pack.Name, "updated_")

	// list packs, only packs[2] remains
	s.DoJSON("GET", "/api/v1/fleet/packs", nil, http.StatusOK, &listResp, "per_page", "2", "order_key", "name")
	require.Len(t, listResp.Packs, 1)
	assert.Equal(t, packs[2].ID, listResp.Packs[0].ID)
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

	s.DoJSON("GET", "/api/v1/fleet/hosts", nil, http.StatusOK, &resp, "order_key", "h.id", "after", fmt.Sprint(hosts[1].ID))
	require.Len(t, resp.Hosts, len(hosts)-2)

	host := hosts[2]
	software := []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
	}
	require.NoError(t, s.ds.UpdateHostSoftware(context.Background(), host.ID, software))
	require.NoError(t, s.ds.LoadHostSoftware(context.Background(), host))

	s.DoJSON("GET", "/api/v1/fleet/hosts", nil, http.StatusOK, &resp, "software_id", fmt.Sprint(host.Software[0].ID))
	require.Len(t, resp.Hosts, 1)
	assert.Equal(t, host.ID, resp.Hosts[0].ID)
	assert.Equal(t, "foo", resp.Software.Name)

	user1 := test.NewUser(t, s.ds, "Alice", "alice@example.com", true)
	q := test.NewQuery(t, s.ds, "query1", "select 1", 0, true)
	defer cleanupQuery(s, q.ID)
	p, err := s.ds.NewGlobalPolicy(context.Background(), &user1.ID, fleet.PolicyPayload{
		QueryID: &q.ID,
	})
	require.NoError(t, err)

	require.NoError(t, s.ds.RecordPolicyQueryExecutions(context.Background(), host, map[uint]*bool{p.ID: ptr.Bool(false)}, time.Now(), false))

	s.DoJSON("GET", "/api/v1/fleet/hosts", nil, http.StatusOK, &resp, "software_id", fmt.Sprint(host.Software[0].ID))
	require.Len(t, resp.Hosts, 1)
	assert.Equal(t, 1, resp.Hosts[0].HostIssues.FailingPoliciesCount)
	assert.Equal(t, 1, resp.Hosts[0].HostIssues.TotalIssuesCount)

	s.DoJSON("GET", "/api/v1/fleet/hosts", nil, http.StatusOK, &resp, "software_id", fmt.Sprint(host.Software[0].ID), "disable_failing_policies", "true")
	require.Len(t, resp.Hosts, 1)
	assert.Equal(t, 0, resp.Hosts[0].HostIssues.FailingPoliciesCount)
	assert.Equal(t, 0, resp.Hosts[0].HostIssues.TotalIssuesCount)
}

func (s *integrationTestSuite) TestInvites() {
	t := s.T()

	team, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		Name:        t.Name() + "team1",
		Description: "desc team1",
	})
	require.NoError(t, err)

	// list invites, none yet
	var listResp listInvitesResponse
	s.DoJSON("GET", "/api/v1/fleet/invites", nil, http.StatusOK, &listResp)
	require.Len(t, listResp.Invites, 0)

	// create valid invite
	createInviteReq := createInviteRequest{InvitePayload: fleet.InvitePayload{
		Email:      ptr.String("some email"),
		Name:       ptr.String("some name"),
		GlobalRole: null.StringFrom(fleet.RoleAdmin),
	}}
	createInviteResp := createInviteResponse{}
	s.DoJSON("POST", "/api/v1/fleet/invites", createInviteReq, http.StatusOK, &createInviteResp)
	require.NotNil(t, createInviteResp.Invite)
	require.NotZero(t, createInviteResp.Invite.ID)
	validInvite := *createInviteResp.Invite

	// create invite without an email
	createInviteReq = createInviteRequest{InvitePayload: fleet.InvitePayload{
		Email:      nil,
		Name:       ptr.String("some other name"),
		GlobalRole: null.StringFrom(fleet.RoleObserver),
	}}
	createInviteResp = createInviteResponse{}
	s.DoJSON("POST", "/api/v1/fleet/invites", createInviteReq, http.StatusUnprocessableEntity, &createInviteResp)

	// create invite for an existing user
	existingEmail := "admin1@example.com"
	createInviteReq = createInviteRequest{InvitePayload: fleet.InvitePayload{
		Email:      ptr.String(existingEmail),
		Name:       ptr.String("some other name"),
		GlobalRole: null.StringFrom(fleet.RoleObserver),
	}}
	createInviteResp = createInviteResponse{}
	s.DoJSON("POST", "/api/v1/fleet/invites", createInviteReq, http.StatusUnprocessableEntity, &createInviteResp)

	// create invite for an existing user with email ALL CAPS
	createInviteReq = createInviteRequest{InvitePayload: fleet.InvitePayload{
		Email:      ptr.String(strings.ToUpper(existingEmail)),
		Name:       ptr.String("some other name"),
		GlobalRole: null.StringFrom(fleet.RoleObserver),
	}}
	createInviteResp = createInviteResponse{}
	s.DoJSON("POST", "/api/v1/fleet/invites", createInviteReq, http.StatusUnprocessableEntity, &createInviteResp)

	// list invites, we have one now
	listResp = listInvitesResponse{}
	s.DoJSON("GET", "/api/v1/fleet/invites", nil, http.StatusOK, &listResp)
	require.Len(t, listResp.Invites, 1)
	require.Equal(t, validInvite.ID, listResp.Invites[0].ID)

	// list invites, next page is empty
	listResp = listInvitesResponse{}
	s.DoJSON("GET", "/api/v1/fleet/invites", nil, http.StatusOK, &listResp, "page", "1", "per_page", "2")
	require.Len(t, listResp.Invites, 0)

	// update a non-existing invite
	updateInviteReq := updateInviteRequest{InvitePayload: fleet.InvitePayload{
		Teams: []fleet.UserTeam{
			{Team: fleet.Team{ID: team.ID}, Role: fleet.RoleObserver},
		}}}
	updateInviteResp := updateInviteResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/v1/fleet/invites/%d", validInvite.ID+1), updateInviteReq, http.StatusNotFound, &updateInviteResp)

	// update the valid invite created earlier, make it an observer of a team
	updateInviteResp = updateInviteResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/v1/fleet/invites/%d", validInvite.ID), updateInviteReq, http.StatusOK, &updateInviteResp)

	verify, err := s.ds.Invite(context.Background(), validInvite.ID)
	require.NoError(t, err)
	require.Equal(t, "", verify.GlobalRole.String)
	require.Len(t, verify.Teams, 1)
	assert.Equal(t, team.ID, verify.Teams[0].ID)

	// delete an existing invite
	var delResp deleteInviteResponse
	s.DoJSON("DELETE", fmt.Sprintf("/api/v1/fleet/invites/%d", validInvite.ID), nil, http.StatusOK, &delResp)

	// list invites, is now empty
	listResp = listInvitesResponse{}
	s.DoJSON("GET", "/api/v1/fleet/invites", nil, http.StatusOK, &listResp)
	require.Len(t, listResp.Invites, 0)

	// delete a now non-existing invite
	s.DoJSON("DELETE", fmt.Sprintf("/api/v1/fleet/invites/%d", validInvite.ID), nil, http.StatusNotFound, &delResp)
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
	require.Len(t, resp.Platforms, 3)
	gotPlatforms, wantPlatforms := make([]string, 3), []string{"linux", "debian", "rhel"}
	for i, p := range resp.Platforms {
		gotPlatforms[i] = p.Platform
		// each platform has a count of 1
		require.Equal(t, uint(1), p.HostsCount)
	}
	require.ElementsMatch(t, wantPlatforms, gotPlatforms)
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
	require.Equal(t, "debian", resp.Platforms[0].Platform)
	require.Equal(t, uint(1), resp.Platforms[0].HostsCount)
	require.Equal(t, team1.ID, *resp.TeamID)

	s.DoJSON("GET", "/api/v1/fleet/host_summary", nil, http.StatusOK, &resp, "team_id", fmt.Sprint(team1.ID), "platform", "linux")
	require.Equal(t, resp.TotalsHostsCount, uint(1))
	require.Equal(t, "debian", resp.Platforms[0].Platform)

	s.DoJSON("GET", "/api/v1/fleet/host_summary", nil, http.StatusOK, &resp, "platform", "rhel")
	require.Equal(t, resp.TotalsHostsCount, uint(1))
	require.Equal(t, "rhel", resp.Platforms[0].Platform)

	s.DoJSON("GET", "/api/v1/fleet/host_summary", nil, http.StatusOK, &resp, "platform", "linux")
	require.Equal(t, resp.TotalsHostsCount, uint(3))
	require.Len(t, resp.Platforms, 3)
	for i, p := range resp.Platforms {
		gotPlatforms[i] = p.Platform
		// each platform has a count of 1
		require.Equal(t, uint(1), p.HostsCount)
	}
	require.ElementsMatch(t, wantPlatforms, gotPlatforms)

	s.DoJSON("GET", "/api/v1/fleet/host_summary", nil, http.StatusOK, &resp, "platform", "darwin")
	require.Equal(t, resp.TotalsHostsCount, uint(0))
	require.Len(t, resp.Platforms, 0)
}

func (s *integrationTestSuite) TestGlobalPoliciesProprietary() {
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
			Platform:        "darwin",
		})
		require.NoError(t, err)
	}

	qr, err := s.ds.NewQuery(context.Background(), &fleet.Query{
		Name:           "TestQuery321",
		Description:    "Some description",
		Query:          "select * from osquery;",
		ObserverCanRun: true,
	})
	require.NoError(t, err)
	// Cannot set both QueryID and Query.
	gpParams0 := globalPolicyRequest{
		QueryID: &qr.ID,
		Query:   "select * from osquery;",
	}
	gpResp0 := globalPolicyResponse{}
	s.DoJSON("POST", "/api/v1/fleet/global/policies", gpParams0, http.StatusBadRequest, &gpResp0)
	require.Nil(t, gpResp0.Policy)

	gpParams := globalPolicyRequest{
		Name:        "TestQuery3",
		Query:       "select * from osquery;",
		Description: "Some description",
		Resolution:  "some global resolution",
		Platform:    "darwin",
	}
	gpResp := globalPolicyResponse{}
	s.DoJSON("POST", "/api/v1/fleet/global/policies", gpParams, http.StatusOK, &gpResp)
	require.NotNil(t, gpResp.Policy)
	require.NotEmpty(t, gpResp.Policy.ID)
	assert.Equal(t, "TestQuery3", gpResp.Policy.Name)
	assert.Equal(t, "select * from osquery;", gpResp.Policy.Query)
	assert.Equal(t, "Some description", gpResp.Policy.Description)
	require.NotNil(t, gpResp.Policy.Resolution)
	assert.Equal(t, "some global resolution", *gpResp.Policy.Resolution)
	assert.NotNil(t, gpResp.Policy.AuthorID)
	assert.Equal(t, "Test Name admin1@example.com", gpResp.Policy.AuthorName)
	assert.Equal(t, "admin1@example.com", gpResp.Policy.AuthorEmail)
	assert.Equal(t, "darwin", gpResp.Policy.Platform)

	mgpParams := modifyGlobalPolicyRequest{
		ModifyPolicyPayload: fleet.ModifyPolicyPayload{
			Name:        ptr.String("TestQuery4"),
			Query:       ptr.String("select * from osquery_info;"),
			Description: ptr.String("Some description updated"),
			Resolution:  ptr.String("some global resolution updated"),
		},
	}
	mgpResp := modifyGlobalPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/v1/fleet/global/policies/%d", gpResp.Policy.ID), mgpParams, http.StatusOK, &mgpResp)
	require.NotNil(t, gpResp.Policy)
	assert.Equal(t, "TestQuery4", mgpResp.Policy.Name)
	assert.Equal(t, "select * from osquery_info;", mgpResp.Policy.Query)
	assert.Equal(t, "Some description updated", mgpResp.Policy.Description)
	require.NotNil(t, mgpResp.Policy.Resolution)
	assert.Equal(t, "some global resolution updated", *mgpResp.Policy.Resolution)
	assert.Equal(t, "darwin", mgpResp.Policy.Platform)

	ggpResp := getPolicyByIDResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/global/policies/%d", gpResp.Policy.ID), getPolicyByIDRequest{}, http.StatusOK, &ggpResp)
	require.NotNil(t, ggpResp.Policy)
	assert.Equal(t, "TestQuery4", ggpResp.Policy.Name)
	assert.Equal(t, "select * from osquery_info;", ggpResp.Policy.Query)
	assert.Equal(t, "Some description updated", ggpResp.Policy.Description)
	require.NotNil(t, ggpResp.Policy.Resolution)
	assert.Equal(t, "some global resolution updated", *ggpResp.Policy.Resolution)
	assert.Equal(t, "darwin", mgpResp.Policy.Platform)

	policiesResponse := listGlobalPoliciesResponse{}
	s.DoJSON("GET", "/api/v1/fleet/global/policies", nil, http.StatusOK, &policiesResponse)
	require.Len(t, policiesResponse.Policies, 1)
	assert.Equal(t, "TestQuery4", policiesResponse.Policies[0].Name)
	assert.Equal(t, "select * from osquery_info;", policiesResponse.Policies[0].Query)
	assert.Equal(t, "Some description updated", policiesResponse.Policies[0].Description)
	require.NotNil(t, policiesResponse.Policies[0].Resolution)
	assert.Equal(t, "some global resolution updated", *policiesResponse.Policies[0].Resolution)
	assert.Equal(t, "darwin", policiesResponse.Policies[0].Platform)

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

func (s *integrationTestSuite) TestTeamPoliciesProprietary() {
	t := s.T()

	team1, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		ID:          42,
		Name:        "team1-policies",
		Description: "desc team1",
	})
	require.NoError(t, err)
	hosts := make([]uint, 2)
	for i := 0; i < 2; i++ {
		h, err := s.ds.NewHost(context.Background(), &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now().Add(-time.Duration(i) * time.Minute),
			OsqueryHostID:   fmt.Sprintf("%s%d", t.Name(), i),
			NodeKey:         fmt.Sprintf("%s%d", t.Name(), i),
			UUID:            fmt.Sprintf("%s%d", t.Name(), i),
			Hostname:        fmt.Sprintf("%sfoo.local%d", t.Name(), i),
			Platform:        "darwin",
		})
		require.NoError(t, err)
		hosts[i] = h.ID
	}
	err = s.ds.AddHostsToTeam(context.Background(), &team1.ID, hosts)
	require.NoError(t, err)

	tpParams := teamPolicyRequest{
		Name:        "TestQuery3",
		Query:       "select * from osquery;",
		Description: "Some description",
		Resolution:  "some team resolution",
		Platform:    "darwin",
	}
	tpResp := teamPolicyResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/v1/fleet/teams/%d/policies", team1.ID), tpParams, http.StatusOK, &tpResp)
	require.NotNil(t, tpResp.Policy)
	require.NotEmpty(t, tpResp.Policy.ID)
	assert.Equal(t, "TestQuery3", tpResp.Policy.Name)
	assert.Equal(t, "select * from osquery;", tpResp.Policy.Query)
	assert.Equal(t, "Some description", tpResp.Policy.Description)
	require.NotNil(t, tpResp.Policy.Resolution)
	assert.Equal(t, "some team resolution", *tpResp.Policy.Resolution)
	assert.NotNil(t, tpResp.Policy.AuthorID)
	assert.Equal(t, "Test Name admin1@example.com", tpResp.Policy.AuthorName)
	assert.Equal(t, "admin1@example.com", tpResp.Policy.AuthorEmail)

	mtpParams := modifyTeamPolicyRequest{
		ModifyPolicyPayload: fleet.ModifyPolicyPayload{
			Name:        ptr.String("TestQuery4"),
			Query:       ptr.String("select * from osquery_info;"),
			Description: ptr.String("Some description updated"),
			Resolution:  ptr.String("some team resolution updated"),
		},
	}
	mtpResp := modifyTeamPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/v1/fleet/teams/%d/policies/%d", team1.ID, tpResp.Policy.ID), mtpParams, http.StatusOK, &mtpResp)
	require.NotNil(t, mtpResp.Policy)
	assert.Equal(t, "TestQuery4", mtpResp.Policy.Name)
	assert.Equal(t, "select * from osquery_info;", mtpResp.Policy.Query)
	assert.Equal(t, "Some description updated", mtpResp.Policy.Description)
	require.NotNil(t, mtpResp.Policy.Resolution)
	assert.Equal(t, "some team resolution updated", *mtpResp.Policy.Resolution)
	assert.Equal(t, "darwin", mtpResp.Policy.Platform)

	gtpResp := getPolicyByIDResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/teams/%d/policies/%d", team1.ID, tpResp.Policy.ID), getPolicyByIDRequest{}, http.StatusOK, &gtpResp)
	require.NotNil(t, gtpResp.Policy)
	assert.Equal(t, "TestQuery4", gtpResp.Policy.Name)
	assert.Equal(t, "select * from osquery_info;", gtpResp.Policy.Query)
	assert.Equal(t, "Some description updated", gtpResp.Policy.Description)
	require.NotNil(t, gtpResp.Policy.Resolution)
	assert.Equal(t, "some team resolution updated", *gtpResp.Policy.Resolution)
	assert.Equal(t, "darwin", gtpResp.Policy.Platform)

	policiesResponse := listTeamPoliciesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/teams/%d/policies", team1.ID), nil, http.StatusOK, &policiesResponse)
	require.Len(t, policiesResponse.Policies, 1)
	assert.Equal(t, "TestQuery4", policiesResponse.Policies[0].Name)
	assert.Equal(t, "select * from osquery_info;", policiesResponse.Policies[0].Query)
	assert.Equal(t, "Some description updated", policiesResponse.Policies[0].Description)
	require.NotNil(t, policiesResponse.Policies[0].Resolution)
	assert.Equal(t, "some team resolution updated", *policiesResponse.Policies[0].Resolution)
	assert.Equal(t, "darwin", policiesResponse.Policies[0].Platform)

	listHostsURL := fmt.Sprintf("/api/v1/fleet/hosts?policy_id=%d", policiesResponse.Policies[0].ID)
	listHostsResp := listHostsResponse{}
	s.DoJSON("GET", listHostsURL, nil, http.StatusOK, &listHostsResp)
	require.Len(t, listHostsResp.Hosts, 2)
	h1 := listHostsResp.Hosts[0]
	h2 := listHostsResp.Hosts[1]

	listHostsURL = fmt.Sprintf("/api/v1/fleet/hosts?team_id=%d&policy_id=%d&policy_response=passing", team1.ID, policiesResponse.Policies[0].ID)
	listHostsResp = listHostsResponse{}
	s.DoJSON("GET", listHostsURL, nil, http.StatusOK, &listHostsResp)
	require.Len(t, listHostsResp.Hosts, 0)

	require.NoError(t, s.ds.RecordPolicyQueryExecutions(context.Background(), h1.Host, map[uint]*bool{policiesResponse.Policies[0].ID: ptr.Bool(true)}, time.Now(), false))
	require.NoError(t, s.ds.RecordPolicyQueryExecutions(context.Background(), h2.Host, map[uint]*bool{policiesResponse.Policies[0].ID: nil}, time.Now(), false))

	listHostsURL = fmt.Sprintf("/api/v1/fleet/hosts?team_id=%d&policy_id=%d&policy_response=passing", team1.ID, policiesResponse.Policies[0].ID)
	listHostsResp = listHostsResponse{}
	s.DoJSON("GET", listHostsURL, nil, http.StatusOK, &listHostsResp)
	require.Len(t, listHostsResp.Hosts, 1)

	deletePolicyParams := deleteTeamPoliciesRequest{IDs: []uint{policiesResponse.Policies[0].ID}}
	deletePolicyResp := deleteTeamPoliciesResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/v1/fleet/teams/%d/policies/delete", team1.ID), deletePolicyParams, http.StatusOK, &deletePolicyResp)

	policiesResponse = listTeamPoliciesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/teams/%d/policies", team1.ID), nil, http.StatusOK, &policiesResponse)
	require.Len(t, policiesResponse.Policies, 0)
}

func (s *integrationTestSuite) TestTeamPoliciesProprietaryInvalid() {
	t := s.T()

	team1, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		ID:          42,
		Name:        "team1-policies-2",
		Description: "desc team1",
	})
	require.NoError(t, err)

	tpParams := teamPolicyRequest{
		Name:        "TestQuery3-Team",
		Query:       "select * from osquery;",
		Description: "Some description",
		Resolution:  "some team resolution",
	}
	tpResp := teamPolicyResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/v1/fleet/teams/%d/policies", team1.ID), tpParams, http.StatusOK, &tpResp)
	require.NotNil(t, tpResp.Policy)
	teamPolicyID := tpResp.Policy.ID

	gpParams := globalPolicyRequest{
		Name:        "TestQuery3-Global",
		Query:       "select * from osquery;",
		Description: "Some description",
		Resolution:  "some global resolution",
	}
	gpResp := globalPolicyResponse{}
	s.DoJSON("POST", "/api/v1/fleet/global/policies", gpParams, http.StatusOK, &gpResp)
	require.NotNil(t, gpResp.Policy)
	require.NotEmpty(t, gpResp.Policy.ID)
	globalPolicyID := gpResp.Policy.ID

	for _, tc := range []struct {
		tname      string
		testUpdate bool
		queryID    *uint
		name       string
		query      string
		platforms  string
	}{
		{
			tname:      "set both QueryID and Query",
			testUpdate: false,
			queryID:    ptr.Uint(1),
			name:       "Some name",
			query:      "select * from osquery;",
		},
		{
			tname:      "empty query",
			testUpdate: true,
			name:       "Some name",
			query:      "",
		},
		{
			tname:      "empty name",
			testUpdate: true,
			name:       "",
			query:      "select 1;",
		},
		{
			tname:      "empty with space",
			testUpdate: true,
			name:       " ", // #3704
			query:      "select 1;",
		},
		{
			tname:      "Invalid query",
			testUpdate: true,
			name:       "Invalid query",
			query:      "ATTACH 'foo' AS bar;",
		},
	} {
		t.Run(tc.tname, func(t *testing.T) {
			tpReq := teamPolicyRequest{
				QueryID:  tc.queryID,
				Name:     tc.name,
				Query:    tc.query,
				Platform: tc.platforms,
			}
			tpResp := teamPolicyResponse{}
			s.DoJSON("POST", fmt.Sprintf("/api/v1/fleet/teams/%d/policies", team1.ID), tpReq, http.StatusBadRequest, &tpResp)
			require.Nil(t, tpResp.Policy)

			testUpdate := tc.queryID == nil

			if testUpdate {
				tpReq := modifyTeamPolicyRequest{
					ModifyPolicyPayload: fleet.ModifyPolicyPayload{
						Name:  ptr.String(tc.name),
						Query: ptr.String(tc.query),
					},
				}
				tpResp := modifyTeamPolicyResponse{}
				s.DoJSON("PATCH", fmt.Sprintf("/api/v1/fleet/teams/%d/policies/%d", team1.ID, teamPolicyID), tpReq, http.StatusBadRequest, &tpResp)
				require.Nil(t, tpResp.Policy)
			}

			gpReq := globalPolicyRequest{
				QueryID:  tc.queryID,
				Name:     tc.name,
				Query:    tc.query,
				Platform: tc.platforms,
			}
			gpResp := globalPolicyResponse{}
			s.DoJSON("POST", "/api/v1/fleet/global/policies", gpReq, http.StatusBadRequest, &gpResp)
			require.Nil(t, tpResp.Policy)

			if testUpdate {
				gpReq := modifyGlobalPolicyRequest{
					ModifyPolicyPayload: fleet.ModifyPolicyPayload{
						Name:  ptr.String(tc.name),
						Query: ptr.String(tc.query),
					},
				}
				gpResp := modifyGlobalPolicyResponse{}
				s.DoJSON("PATCH", fmt.Sprintf("/api/v1/fleet/global/policies/%d", globalPolicyID), gpReq, http.StatusBadRequest, &gpResp)
				require.Nil(t, tpResp.Policy)
			}
		})
	}
}

func (s *integrationTestSuite) TestHostDetailsPolicies() {
	t := s.T()

	hosts := s.createHosts(t)
	host1 := hosts[0]
	team1, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		ID:          42,
		Name:        "HostDetailsPolicies-Team",
		Description: "desc team1",
	})
	require.NoError(t, err)
	err = s.ds.AddHostsToTeam(context.Background(), &team1.ID, []uint{host1.ID})
	require.NoError(t, err)

	gpParams := globalPolicyRequest{
		Name:        "HostDetailsPolicies",
		Query:       "select * from osquery;",
		Description: "Some description",
		Resolution:  "some global resolution",
	}
	gpResp := globalPolicyResponse{}
	s.DoJSON("POST", "/api/v1/fleet/global/policies", gpParams, http.StatusOK, &gpResp)
	require.NotNil(t, gpResp.Policy)
	require.NotEmpty(t, gpResp.Policy.ID)

	tpParams := teamPolicyRequest{
		Name:        "HostDetailsPolicies-Team",
		Query:       "select * from osquery;",
		Description: "Some description",
		Resolution:  "some team resolution",
	}
	tpResp := teamPolicyResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/v1/fleet/teams/%d/policies", team1.ID), tpParams, http.StatusOK, &tpResp)
	require.NotNil(t, tpResp.Policy)
	require.NotEmpty(t, tpResp.Policy.ID)

	err = s.ds.RecordPolicyQueryExecutions(
		context.Background(),
		host1,
		map[uint]*bool{gpResp.Policy.ID: ptr.Bool(true)},
		time.Now(),
		false,
	)
	require.NoError(t, err)

	resp := s.Do("GET", fmt.Sprintf("/api/v1/fleet/hosts/%d", host1.ID), nil, http.StatusOK)
	b, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	var r struct {
		Host *HostDetailResponse `json:"host"`
		Err  error               `json:"error,omitempty"`
	}
	err = json.Unmarshal(b, &r)
	require.NoError(t, err)
	require.Nil(t, r.Err)
	hd := r.Host.HostDetail
	require.Len(t, hd.Policies, 2)
	require.True(t, reflect.DeepEqual(gpResp.Policy.PolicyData, hd.Policies[0].PolicyData))
	require.Equal(t, hd.Policies[0].Response, "pass")

	require.True(t, reflect.DeepEqual(tpResp.Policy.PolicyData, hd.Policies[1].PolicyData))
	require.Equal(t, hd.Policies[1].Response, "") // policy didn't "run"

	// Try to create a global policy with an existing name.
	s.DoJSON("POST", "/api/v1/fleet/global/policies", gpParams, http.StatusConflict, &gpResp)
	// Try to create a team policy with an existing name.
	s.DoJSON("POST", fmt.Sprintf("/api/v1/fleet/teams/%d/policies", team1.ID), tpParams, http.StatusConflict, &tpResp)
}

func (s *integrationTestSuite) TestListActivities() {
	t := s.T()

	ctx := context.Background()
	u := s.users["admin1@example.com"]
	details := make(map[string]interface{})

	err := s.ds.NewActivity(ctx, &u, fleet.ActivityTypeAppliedSpecPack, &details)
	require.NoError(t, err)

	err = s.ds.NewActivity(ctx, &u, fleet.ActivityTypeDeletedPack, &details)
	require.NoError(t, err)

	err = s.ds.NewActivity(ctx, &u, fleet.ActivityTypeEditedPack, &details)
	require.NoError(t, err)

	var listResp listActivitiesResponse
	s.DoJSON("GET", "/api/v1/fleet/activities", nil, http.StatusOK, &listResp, "per_page", "2", "order_key", "id")
	require.Len(t, listResp.Activities, 2)
	assert.Equal(t, fleet.ActivityTypeAppliedSpecPack, listResp.Activities[0].Type)
	assert.Equal(t, fleet.ActivityTypeDeletedPack, listResp.Activities[1].Type)

	s.DoJSON("GET", "/api/v1/fleet/activities", nil, http.StatusOK, &listResp, "per_page", "2", "order_key", "id", "page", "1")
	require.Len(t, listResp.Activities, 1)
	assert.Equal(t, fleet.ActivityTypeEditedPack, listResp.Activities[0].Type)

	s.DoJSON("GET", "/api/v1/fleet/activities", nil, http.StatusOK, &listResp, "per_page", "1", "order_key", "id", "order_direction", "desc")
	require.Len(t, listResp.Activities, 1)
	assert.Equal(t, fleet.ActivityTypeEditedPack, listResp.Activities[0].Type)
}

func (s *integrationTestSuite) TestListGetCarves() {
	t := s.T()

	ctx := context.Background()

	hosts := s.createHosts(t)
	c1, err := s.ds.NewCarve(ctx, &fleet.CarveMetadata{
		CreatedAt: time.Now(),
		HostId:    hosts[0].ID,
		Name:      t.Name() + "_1",
		SessionId: "ssn1",
	})
	require.NoError(t, err)
	c2, err := s.ds.NewCarve(ctx, &fleet.CarveMetadata{
		CreatedAt: time.Now(),
		HostId:    hosts[1].ID,
		Name:      t.Name() + "_2",
		SessionId: "ssn2",
	})
	require.NoError(t, err)
	c3, err := s.ds.NewCarve(ctx, &fleet.CarveMetadata{
		CreatedAt: time.Now(),
		HostId:    hosts[2].ID,
		Name:      t.Name() + "_3",
		SessionId: "ssn3",
	})
	require.NoError(t, err)

	// set c1 max block
	c1.MaxBlock = 3
	require.NoError(t, s.ds.UpdateCarve(ctx, c1))
	// make c2 expired, set max block
	c2.Expired = true
	c2.MaxBlock = 3
	require.NoError(t, s.ds.UpdateCarve(ctx, c2))

	var listResp listCarvesResponse
	s.DoJSON("GET", "/api/v1/fleet/carves", nil, http.StatusOK, &listResp, "per_page", "2", "order_key", "id")
	require.Len(t, listResp.Carves, 2)
	assert.Equal(t, c1.ID, listResp.Carves[0].ID)
	assert.Equal(t, c3.ID, listResp.Carves[1].ID)

	// include expired
	s.DoJSON("GET", "/api/v1/fleet/carves", nil, http.StatusOK, &listResp, "per_page", "2", "order_key", "id", "expired", "1")
	require.Len(t, listResp.Carves, 2)
	assert.Equal(t, c1.ID, listResp.Carves[0].ID)
	assert.Equal(t, c2.ID, listResp.Carves[1].ID)

	// empty page
	s.DoJSON("GET", "/api/v1/fleet/carves", nil, http.StatusOK, &listResp, "page", "3", "per_page", "2", "order_key", "id", "expired", "1")
	require.Len(t, listResp.Carves, 0)

	// get specific carve
	var getResp getCarveResponse
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/carves/%d", c2.ID), nil, http.StatusOK, &getResp)
	require.Equal(t, c2.ID, getResp.Carve.ID)
	require.True(t, getResp.Carve.Expired)

	// get non-existing carve
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/carves/%d", c3.ID+1), nil, http.StatusNotFound, &getResp)

	// get expired carve block
	var blkResp getCarveBlockResponse
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/carves/%d/block/%d", c2.ID, 1), nil, http.StatusInternalServerError, &blkResp)

	// get valid carve block, but block not inserted yet
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/carves/%d/block/%d", c1.ID, 1), nil, http.StatusNotFound, &blkResp)

	require.NoError(t, s.ds.NewBlock(ctx, c1, 1, []byte("block1")))
	require.NoError(t, s.ds.NewBlock(ctx, c1, 2, []byte("block2")))
	require.NoError(t, s.ds.NewBlock(ctx, c1, 3, []byte("block3")))

	// get valid carve block
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/carves/%d/block/%d", c1.ID, 1), nil, http.StatusOK, &blkResp)
	require.Equal(t, "block1", string(blkResp.Data))
}

func (s *integrationTestSuite) TestHostsAddToTeam() {
	t := s.T()

	ctx := context.Background()

	tm1, err := s.ds.NewTeam(ctx, &fleet.Team{
		Name: uuid.New().String(),
	})
	require.NoError(t, err)
	tm2, err := s.ds.NewTeam(ctx, &fleet.Team{
		Name: uuid.New().String(),
	})
	require.NoError(t, err)

	hosts := s.createHosts(t)
	var refetchResp refetchHostResponse
	// refetch existing
	s.DoJSON("POST", fmt.Sprintf("/api/v1/fleet/hosts/%d/refetch", hosts[0].ID), nil, http.StatusOK, &refetchResp)
	require.NoError(t, refetchResp.Err)

	// refetch unknown
	s.DoJSON("POST", fmt.Sprintf("/api/v1/fleet/hosts/%d/refetch", hosts[2].ID+1), nil, http.StatusNotFound, &refetchResp)

	// get by identifier unknown
	var getResp getHostResponse
	s.DoJSON("GET", "/api/v1/fleet/hosts/identifier/no-such-host", nil, http.StatusNotFound, &getResp)

	// get by identifier valid
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/hosts/identifier/%s", hosts[0].UUID), nil, http.StatusOK, &getResp)
	require.Equal(t, hosts[0].ID, getResp.Host.ID)
	require.Nil(t, getResp.Host.TeamID)

	// assign hosts to team 1
	var addResp addHostsToTeamResponse
	s.DoJSON("POST", "/api/v1/fleet/hosts/transfer", addHostsToTeamRequest{
		TeamID:  &tm1.ID,
		HostIDs: []uint{hosts[0].ID, hosts[1].ID},
	}, http.StatusOK, &addResp)

	// check that hosts are now part of that team
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/hosts/%d", hosts[0].ID), nil, http.StatusOK, &getResp)
	require.NotNil(t, getResp.Host.TeamID)
	require.Equal(t, tm1.ID, *getResp.Host.TeamID)
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/hosts/%d", hosts[1].ID), nil, http.StatusOK, &getResp)
	require.NotNil(t, getResp.Host.TeamID)
	require.Equal(t, tm1.ID, *getResp.Host.TeamID)
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/hosts/%d", hosts[2].ID), nil, http.StatusOK, &getResp)
	require.Nil(t, getResp.Host.TeamID)

	// assign host to team 2 with filter
	var addfResp addHostsToTeamByFilterResponse
	req := addHostsToTeamByFilterRequest{TeamID: &tm2.ID}
	req.Filters.MatchQuery = hosts[2].Hostname
	s.DoJSON("POST", "/api/v1/fleet/hosts/transfer/filter", req, http.StatusOK, &addfResp)

	// check that host 2 is now part of team 2
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/hosts/%d", hosts[2].ID), nil, http.StatusOK, &getResp)
	require.NotNil(t, getResp.Host.TeamID)
	require.Equal(t, tm2.ID, *getResp.Host.TeamID)

	// delete host 0
	var delResp deleteHostResponse
	s.DoJSON("DELETE", fmt.Sprintf("/api/v1/fleet/hosts/%d", hosts[0].ID), nil, http.StatusOK, &delResp)
	// delete non-existing host
	s.DoJSON("DELETE", fmt.Sprintf("/api/v1/fleet/hosts/%d", hosts[2].ID+1), nil, http.StatusNotFound, &delResp)

	// list the hosts
	var listResp listHostsResponse
	s.DoJSON("GET", "/api/v1/fleet/hosts", nil, http.StatusOK, &listResp, "per_page", "3")
	require.Len(t, listResp.Hosts, 2)
	ids := []uint{listResp.Hosts[0].ID, listResp.Hosts[1].ID}
	require.ElementsMatch(t, ids, []uint{hosts[1].ID, hosts[2].ID})
}

func (s *integrationTestSuite) TestScheduledQueries() {
	t := s.T()

	// create a pack
	var createPackResp createPackResponse
	reqPack := &createPackRequest{
		PackPayload: fleet.PackPayload{
			Name: ptr.String(strings.ReplaceAll(t.Name(), "/", "_")),
		},
	}
	s.DoJSON("POST", "/api/v1/fleet/packs", reqPack, http.StatusOK, &createPackResp)
	pack := createPackResp.Pack.Pack

	// try a non existent query
	s.Do("GET", fmt.Sprintf("/api/v1/fleet/queries/%d", 9999), nil, http.StatusNotFound)

	// list queries
	var listQryResp listQueriesResponse
	s.DoJSON("GET", "/api/v1/fleet/queries", nil, http.StatusOK, &listQryResp)
	assert.Len(t, listQryResp.Queries, 0)

	// create a query
	var createQueryResp createQueryResponse
	reqQuery := &fleet.QueryPayload{
		Name:  ptr.String(strings.ReplaceAll(t.Name(), "/", "_")),
		Query: ptr.String("select * from time;"),
	}
	s.DoJSON("POST", "/api/v1/fleet/queries", reqQuery, http.StatusOK, &createQueryResp)
	query := createQueryResp.Query

	// listing returns that query
	s.DoJSON("GET", "/api/v1/fleet/queries", nil, http.StatusOK, &listQryResp)
	require.Len(t, listQryResp.Queries, 1)
	assert.Equal(t, query.Name, listQryResp.Queries[0].Name)

	// next page returns nothing
	s.DoJSON("GET", "/api/v1/fleet/queries", nil, http.StatusOK, &listQryResp, "per_page", "2", "page", "1")
	require.Len(t, listQryResp.Queries, 0)

	// getting that query works
	var getQryResp getQueryResponse
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/queries/%d", query.ID), nil, http.StatusOK, &getQryResp)
	assert.Equal(t, query.ID, getQryResp.Query.ID)

	// list scheduled queries in pack, none yet
	var getInPackResp getScheduledQueriesInPackResponse
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/packs/%d/scheduled", pack.ID), nil, http.StatusOK, &getInPackResp)
	assert.Len(t, getInPackResp.Scheduled, 0)

	// list scheduled queries in non-existing pack
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/packs/%d/scheduled", pack.ID+1), nil, http.StatusOK, &getInPackResp)
	assert.Len(t, getInPackResp.Scheduled, 0)

	// create scheduled query
	var createResp scheduleQueryResponse
	reqSQ := &scheduleQueryRequest{
		PackID:   pack.ID,
		QueryID:  query.ID,
		Interval: 1,
	}
	s.DoJSON("POST", "/api/v1/fleet/schedule", reqSQ, http.StatusOK, &createResp)
	sq1 := createResp.Scheduled.ScheduledQuery
	assert.NotZero(t, sq1.ID)
	assert.Equal(t, uint(1), sq1.Interval)

	// create scheduled query with invalid pack
	reqSQ = &scheduleQueryRequest{
		PackID:   pack.ID + 1,
		QueryID:  query.ID,
		Interval: 2,
	}
	s.DoJSON("POST", "/api/v1/fleet/schedule", reqSQ, http.StatusUnprocessableEntity, &createResp)

	// create scheduled query with invalid query
	reqSQ = &scheduleQueryRequest{
		PackID:   pack.ID,
		QueryID:  query.ID + 1,
		Interval: 3,
	}
	s.DoJSON("POST", "/api/v1/fleet/schedule", reqSQ, http.StatusNotFound, &createResp)

	// list scheduled queries in pack
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/packs/%d/scheduled", pack.ID), nil, http.StatusOK, &getInPackResp)
	require.Len(t, getInPackResp.Scheduled, 1)
	assert.Equal(t, sq1.ID, getInPackResp.Scheduled[0].ID)

	// list scheduled queries in pack, next page
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/packs/%d/scheduled", pack.ID), nil, http.StatusOK, &getInPackResp, "page", "1", "per_page", "2")
	require.Len(t, getInPackResp.Scheduled, 0)

	// get non-existing scheduled query
	var getResp getScheduledQueryResponse
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/schedule/%d", sq1.ID+1), nil, http.StatusNotFound, &getResp)

	// get existing scheduled query
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/schedule/%d", sq1.ID), nil, http.StatusOK, &getResp)
	assert.Equal(t, sq1.ID, getResp.Scheduled.ID)
	assert.Equal(t, sq1.Interval, getResp.Scheduled.Interval)

	// modify scheduled query
	var modResp modifyScheduledQueryResponse
	reqMod := fleet.ScheduledQueryPayload{
		Interval: ptr.Uint(4),
	}
	s.DoJSON("PATCH", fmt.Sprintf("/api/v1/fleet/schedule/%d", sq1.ID), reqMod, http.StatusOK, &modResp)
	assert.Equal(t, sq1.ID, modResp.Scheduled.ID)
	assert.Equal(t, uint(4), modResp.Scheduled.Interval)

	// modify non-existing scheduled query
	reqMod = fleet.ScheduledQueryPayload{
		Interval: ptr.Uint(5),
	}
	s.DoJSON("PATCH", fmt.Sprintf("/api/v1/fleet/schedule/%d", sq1.ID+1), reqMod, http.StatusNotFound, &modResp)

	// delete non-existing scheduled query
	var delResp deleteScheduledQueryResponse
	s.DoJSON("DELETE", fmt.Sprintf("/api/v1/fleet/schedule/%d", sq1.ID+1), nil, http.StatusNotFound, &delResp)

	// delete existing scheduled query
	s.DoJSON("DELETE", fmt.Sprintf("/api/v1/fleet/schedule/%d", sq1.ID), nil, http.StatusOK, &delResp)

	// get the now-deleted scheduled query
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/schedule/%d", sq1.ID), nil, http.StatusNotFound, &getResp)

	// modify the query
	var modQryResp modifyQueryResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/v1/fleet/queries/%d", query.ID), fleet.QueryPayload{Description: ptr.String("updated")}, http.StatusOK, &modQryResp)
	assert.Equal(t, "updated", modQryResp.Query.Description)

	// modify a non-existing query
	s.DoJSON("PATCH", fmt.Sprintf("/api/v1/fleet/queries/%d", query.ID+1), fleet.QueryPayload{Description: ptr.String("updated")}, http.StatusNotFound, &modQryResp)

	// delete the query by name
	var delByNameResp deleteQueryResponse
	s.DoJSON("DELETE", fmt.Sprintf("/api/v1/fleet/queries/%s", query.Name), nil, http.StatusOK, &delByNameResp)

	// delete unknown query by name (i.e. the same, now deleted)
	s.DoJSON("DELETE", fmt.Sprintf("/api/v1/fleet/queries/%s", query.Name), nil, http.StatusNotFound, &delByNameResp)

	// create another query
	reqQuery = &fleet.QueryPayload{
		Name:  ptr.String(strings.ReplaceAll(t.Name(), "/", "_") + "_2"),
		Query: ptr.String("select 2"),
	}
	s.DoJSON("POST", "/api/v1/fleet/queries", reqQuery, http.StatusOK, &createQueryResp)
	query2 := createQueryResp.Query

	// delete it by id
	var delByIDResp deleteQueryByIDResponse
	s.DoJSON("DELETE", fmt.Sprintf("/api/v1/fleet/queries/id/%d", query2.ID), nil, http.StatusOK, &delByIDResp)

	// delete unknown query by id (same id just deleted)
	s.DoJSON("DELETE", fmt.Sprintf("/api/v1/fleet/queries/id/%d", query2.ID), nil, http.StatusNotFound, &delByIDResp)

	// create another query
	reqQuery = &fleet.QueryPayload{
		Name:  ptr.String(strings.ReplaceAll(t.Name(), "/", "_") + "_3"),
		Query: ptr.String("select 3"),
	}
	s.DoJSON("POST", "/api/v1/fleet/queries", reqQuery, http.StatusOK, &createQueryResp)
	query3 := createQueryResp.Query

	// batch-delete by id, 3 ids, only one exists
	var delBatchResp deleteQueriesResponse
	s.DoJSON("POST", "/api/v1/fleet/queries/delete", map[string]interface{}{
		"ids": []uint{query.ID, query2.ID, query3.ID},
	}, http.StatusOK, &delBatchResp)
	assert.Equal(t, uint(1), delBatchResp.Deleted)

	// batch-delete by id, none exist
	delBatchResp.Deleted = 0
	s.DoJSON("POST", "/api/v1/fleet/queries/delete", map[string]interface{}{
		"ids": []uint{query.ID, query2.ID, query3.ID},
	}, http.StatusNotFound, &delBatchResp)
	assert.Equal(t, uint(0), delBatchResp.Deleted)
}

func (s *integrationTestSuite) TestHostDeviceMapping() {
	t := s.T()
	ctx := context.Background()

	hosts := s.createHosts(t)

	// get host device mappings of invalid host
	var listResp listHostDeviceMappingResponse
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/hosts/%d/device_mapping", hosts[2].ID+1), nil, http.StatusNotFound, &listResp)

	// existing host but none yet
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/hosts/%d/device_mapping", hosts[0].ID), nil, http.StatusOK, &listResp)
	require.Len(t, listResp.DeviceMapping, 0)

	// create some mappings
	s.ds.ReplaceHostDeviceMapping(ctx, hosts[0].ID, []*fleet.HostDeviceMapping{
		{HostID: hosts[0].ID, Email: "a@b.c", Source: "google_chrome_profiles"},
		{HostID: hosts[0].ID, Email: "b@b.c", Source: "google_chrome_profiles"},
	})

	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/hosts/%d/device_mapping", hosts[0].ID), nil, http.StatusOK, &listResp)
	require.Len(t, listResp.DeviceMapping, 2)
	require.Equal(t, "a@b.c", listResp.DeviceMapping[0].Email)
	require.Equal(t, "google_chrome_profiles", listResp.DeviceMapping[0].Source)
	require.Zero(t, listResp.DeviceMapping[0].HostID)
	require.Equal(t, "b@b.c", listResp.DeviceMapping[1].Email)
	require.Equal(t, "google_chrome_profiles", listResp.DeviceMapping[1].Source)
	require.Zero(t, listResp.DeviceMapping[1].HostID)
	require.Equal(t, hosts[0].ID, listResp.HostID)

	// other host still has none
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/hosts/%d/device_mapping", hosts[1].ID), nil, http.StatusOK, &listResp)
	require.Len(t, listResp.DeviceMapping, 0)

	// search host by email address finds the corresponding host
	var listHosts listHostsResponse
	s.DoJSON("GET", "/api/v1/fleet/hosts", nil, http.StatusOK, &listHosts, "query", "a@b.c")
	require.Len(t, listHosts.Hosts, 1)
	assert.Equal(t, hosts[0].ID, listHosts.Hosts[0].ID)

	s.DoJSON("GET", "/api/v1/fleet/hosts", nil, http.StatusOK, &listHosts, "query", "c@b.c")
	require.Len(t, listHosts.Hosts, 0)
}

func (s *integrationTestSuite) TestGetMacadminsData() {
	t := s.T()

	ctx := context.Background()

	hostAll, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         t.Name() + "1",
		UUID:            t.Name() + "1",
		Hostname:        t.Name() + "foo.local",
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-58",
		OsqueryHostID:   "1",
	})
	require.NoError(t, err)
	require.NotNil(t, hostAll)

	hostNothing, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         t.Name() + "2",
		UUID:            t.Name() + "2",
		Hostname:        t.Name() + "foo.local2",
		PrimaryIP:       "192.168.1.2",
		PrimaryMac:      "30-65-EC-6F-C4-59",
		OsqueryHostID:   "2",
	})
	require.NoError(t, err)
	require.NotNil(t, hostNothing)

	hostOnlyMunki, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         t.Name() + "3",
		UUID:            t.Name() + "3",
		Hostname:        t.Name() + "foo.local3",
		PrimaryIP:       "192.168.1.3",
		PrimaryMac:      "30-65-EC-6F-C4-5F",
		OsqueryHostID:   "3",
	})
	require.NoError(t, err)
	require.NotNil(t, hostOnlyMunki)

	hostOnlyMDM, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         t.Name() + "4",
		UUID:            t.Name() + "4",
		Hostname:        t.Name() + "foo.local4",
		PrimaryIP:       "192.168.1.4",
		PrimaryMac:      "30-65-EC-6F-C4-5A",
		OsqueryHostID:   "4",
	})
	require.NoError(t, err)
	require.NotNil(t, hostOnlyMDM)

	require.NoError(t, s.ds.SetOrUpdateMDMData(ctx, hostAll.ID, true, "url", false))
	require.NoError(t, s.ds.SetOrUpdateMunkiVersion(ctx, hostAll.ID, "1.3.0"))

	macadminsData := getMacadminsDataResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/hosts/%d/macadmins", hostAll.ID), nil, http.StatusOK, &macadminsData)
	require.NotNil(t, macadminsData.Macadmins)
	assert.Equal(t, "url", macadminsData.Macadmins.MDM.ServerURL)
	assert.Equal(t, "Enrolled (manual)", macadminsData.Macadmins.MDM.EnrollmentStatus)
	assert.Equal(t, "1.3.0", macadminsData.Macadmins.Munki.Version)

	require.NoError(t, s.ds.SetOrUpdateMDMData(ctx, hostAll.ID, true, "url2", true))
	require.NoError(t, s.ds.SetOrUpdateMunkiVersion(ctx, hostAll.ID, "1.5.0"))

	macadminsData = getMacadminsDataResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/hosts/%d/macadmins", hostAll.ID), nil, http.StatusOK, &macadminsData)
	require.NotNil(t, macadminsData.Macadmins)
	assert.Equal(t, "url2", macadminsData.Macadmins.MDM.ServerURL)
	assert.Equal(t, "Enrolled (automated)", macadminsData.Macadmins.MDM.EnrollmentStatus)
	assert.Equal(t, "1.5.0", macadminsData.Macadmins.Munki.Version)

	require.NoError(t, s.ds.SetOrUpdateMDMData(ctx, hostAll.ID, false, "url2", false))

	macadminsData = getMacadminsDataResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/hosts/%d/macadmins", hostAll.ID), nil, http.StatusOK, &macadminsData)
	require.NotNil(t, macadminsData.Macadmins)
	assert.Equal(t, "Unenrolled", macadminsData.Macadmins.MDM.EnrollmentStatus)

	// nothing returns null
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/hosts/%d/macadmins", hostNothing.ID), nil, http.StatusOK, &macadminsData)
	require.Nil(t, macadminsData.Macadmins)

	// only munki info returns null on mdm
	require.NoError(t, s.ds.SetOrUpdateMunkiVersion(ctx, hostOnlyMunki.ID, "3.2.0"))
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/hosts/%d/macadmins", hostOnlyMunki.ID), nil, http.StatusOK, &macadminsData)
	require.NotNil(t, macadminsData.Macadmins)
	require.Nil(t, macadminsData.Macadmins.MDM)
	require.NotNil(t, macadminsData.Macadmins.Munki)
	assert.Equal(t, "3.2.0", macadminsData.Macadmins.Munki.Version)

	// only mdm returns null on munki info
	require.NoError(t, s.ds.SetOrUpdateMDMData(ctx, hostOnlyMDM.ID, true, "AAA", true))
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/hosts/%d/macadmins", hostOnlyMDM.ID), nil, http.StatusOK, &macadminsData)
	require.NotNil(t, macadminsData.Macadmins)
	require.NotNil(t, macadminsData.Macadmins.MDM)
	require.Nil(t, macadminsData.Macadmins.Munki)
	assert.Equal(t, "AAA", macadminsData.Macadmins.MDM.ServerURL)
	assert.Equal(t, "Enrolled (automated)", macadminsData.Macadmins.MDM.EnrollmentStatus)

	// generate aggregated data
	require.NoError(t, s.ds.GenerateAggregatedMunkiAndMDM(context.Background()))

	agg := getAggregatedMacadminsDataResponse{}
	s.DoJSON("GET", "/api/v1/fleet/macadmins", nil, http.StatusOK, &agg)
	require.NotNil(t, agg.Macadmins)
	assert.NotZero(t, agg.Macadmins.CountsUpdatedAt)
	assert.Len(t, agg.Macadmins.MunkiVersions, 2)
	assert.ElementsMatch(t, agg.Macadmins.MunkiVersions, []fleet.AggregatedMunkiVersion{
		{
			HostMunkiInfo: fleet.HostMunkiInfo{Version: "1.5.0"},
			HostsCount:    1,
		},
		{
			HostMunkiInfo: fleet.HostMunkiInfo{Version: "3.2.0"},
			HostsCount:    1,
		},
	})
	assert.Equal(t, agg.Macadmins.MDMStatus.EnrolledManualHostsCount, 0)
	assert.Equal(t, agg.Macadmins.MDMStatus.EnrolledAutomatedHostsCount, 1)
	assert.Equal(t, agg.Macadmins.MDMStatus.UnenrolledHostsCount, 1)
	assert.Equal(t, agg.Macadmins.MDMStatus.HostsCount, 2)

	team, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		Name:        "team1" + t.Name(),
		Description: "desc team1",
	})
	require.NoError(t, err)

	agg = getAggregatedMacadminsDataResponse{}
	s.DoJSON("GET", "/api/v1/fleet/macadmins", nil, http.StatusOK, &agg, "team_id", fmt.Sprint(team.ID))
	require.NotNil(t, agg.Macadmins)
	require.Empty(t, agg.Macadmins.MunkiVersions)
	require.Empty(t, agg.Macadmins.MDMStatus)

	agg = getAggregatedMacadminsDataResponse{}
	s.DoJSON("GET", "/api/v1/fleet/macadmins", nil, http.StatusNotFound, &agg, "team_id", "9999999")
}

func (s *integrationTestSuite) TestLabels() {
	t := s.T()

	// list labels, has the built-in ones
	var listResp listLabelsResponse
	s.DoJSON("GET", "/api/v1/fleet/labels", nil, http.StatusOK, &listResp)
	assert.True(t, len(listResp.Labels) > 0)
	for _, lbl := range listResp.Labels {
		assert.Equal(t, fleet.LabelTypeBuiltIn, lbl.LabelType)
	}
	builtInsCount := len(listResp.Labels)

	// create a label without name, an error
	var createResp createLabelResponse
	s.DoJSON("POST", "/api/v1/fleet/labels", &fleet.LabelPayload{Query: ptr.String("select 1")}, http.StatusUnprocessableEntity, &createResp)

	// create a valid label
	s.DoJSON("POST", "/api/v1/fleet/labels", &fleet.LabelPayload{Name: ptr.String(t.Name()), Query: ptr.String("select 1")}, http.StatusOK, &createResp)
	assert.NotZero(t, createResp.Label.ID)
	assert.Equal(t, t.Name(), createResp.Label.Name)
	lbl1 := createResp.Label.Label

	// get the label
	var getResp getLabelResponse
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/labels/%d", lbl1.ID), nil, http.StatusOK, &getResp)
	assert.Equal(t, lbl1.ID, getResp.Label.ID)

	// get a non-existing label
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/labels/%d", lbl1.ID+1), nil, http.StatusNotFound, &getResp)

	// modify that label
	var modResp modifyLabelResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/v1/fleet/labels/%d", lbl1.ID), &fleet.ModifyLabelPayload{Name: ptr.String(t.Name() + "zzz")}, http.StatusOK, &modResp)
	assert.Equal(t, lbl1.ID, modResp.Label.ID)
	assert.NotEqual(t, lbl1.Name, modResp.Label.Name)

	// modify a non-existing label
	s.DoJSON("PATCH", fmt.Sprintf("/api/v1/fleet/labels/%d", lbl1.ID+1), &fleet.ModifyLabelPayload{Name: ptr.String("zzz")}, http.StatusNotFound, &modResp)

	// list labels
	s.DoJSON("GET", "/api/v1/fleet/labels", nil, http.StatusOK, &listResp, "per_page", strconv.Itoa(builtInsCount+1))
	assert.Len(t, listResp.Labels, builtInsCount+1)

	// next page is empty
	s.DoJSON("GET", "/api/v1/fleet/labels", nil, http.StatusOK, &listResp, "per_page", "2", "page", "1", "query", t.Name())
	assert.Len(t, listResp.Labels, 0)

	// create another label
	s.DoJSON("POST", "/api/v1/fleet/labels", &fleet.LabelPayload{Name: ptr.String(strings.ReplaceAll(t.Name(), "/", "_")), Query: ptr.String("select 1")}, http.StatusOK, &createResp)
	assert.NotZero(t, createResp.Label.ID)
	lbl2 := createResp.Label.Label

	// create hosts and add them to that label
	hosts := s.createHosts(t)
	for _, h := range hosts {
		err := s.ds.RecordLabelQueryExecutions(context.Background(), h, map[uint]*bool{lbl2.ID: ptr.Bool(true)}, time.Now(), false)
		require.NoError(t, err)
	}

	// list hosts in label
	var listHostsResp listHostsResponse
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/labels/%d/hosts", lbl2.ID), nil, http.StatusOK, &listHostsResp)
	assert.Len(t, listHostsResp.Hosts, len(hosts))

	// lists hosts in label without hosts
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/labels/%d/hosts", lbl1.ID), nil, http.StatusOK, &listHostsResp)
	assert.Len(t, listHostsResp.Hosts, 0)

	// lists hosts in invalid label
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/labels/%d/hosts", lbl2.ID+1), nil, http.StatusOK, &listHostsResp)
	assert.Len(t, listHostsResp.Hosts, 0)

	// delete a label by id
	var delIDResp deleteLabelByIDResponse
	s.DoJSON("DELETE", fmt.Sprintf("/api/v1/fleet/labels/id/%d", lbl1.ID), nil, http.StatusOK, &delIDResp)

	// delete a non-existing label by id
	s.DoJSON("DELETE", fmt.Sprintf("/api/v1/fleet/labels/id/%d", lbl2.ID+1), nil, http.StatusNotFound, &delIDResp)

	// delete a label by name
	var delResp deleteLabelResponse
	s.DoJSON("DELETE", fmt.Sprintf("/api/v1/fleet/labels/%s", url.PathEscape(lbl2.Name)), nil, http.StatusOK, &delResp)

	// delete a non-existing label by name
	s.DoJSON("DELETE", fmt.Sprintf("/api/v1/fleet/labels/%s", url.PathEscape(lbl2.Name)), nil, http.StatusNotFound, &delResp)

	// list labels, only the built-ins remain
	s.DoJSON("GET", "/api/v1/fleet/labels", nil, http.StatusOK, &listResp, "per_page", strconv.Itoa(builtInsCount+1))
	assert.Len(t, listResp.Labels, builtInsCount)
	for _, lbl := range listResp.Labels {
		assert.Equal(t, fleet.LabelTypeBuiltIn, lbl.LabelType)
	}
}

func (s *integrationTestSuite) TestLabelSpecs() {
	t := s.T()

	// list label specs, only those of the built-ins
	var listResp getLabelSpecsResponse
	s.DoJSON("GET", "/api/v1/fleet/spec/labels", nil, http.StatusOK, &listResp)
	assert.True(t, len(listResp.Specs) > 0)
	for _, spec := range listResp.Specs {
		assert.Equal(t, fleet.LabelTypeBuiltIn, spec.LabelType)
	}
	builtInsCount := len(listResp.Specs)

	name := strings.ReplaceAll(t.Name(), "/", "_")
	// apply an invalid label spec - dynamic membership with host specified
	var applyResp applyLabelSpecsResponse
	s.DoJSON("POST", "/api/v1/fleet/spec/labels", applyLabelSpecsRequest{
		Specs: []*fleet.LabelSpec{
			{
				Name:                name,
				Query:               "select 1",
				Platform:            "linux",
				LabelMembershipType: fleet.LabelMembershipTypeDynamic,
				Hosts:               []string{"abc"},
			},
		},
	}, http.StatusInternalServerError, &applyResp)

	// apply an invalid label spec - manual membership without a host specified
	s.DoJSON("POST", "/api/v1/fleet/spec/labels", applyLabelSpecsRequest{
		Specs: []*fleet.LabelSpec{
			{
				Name:                name,
				Query:               "select 1",
				Platform:            "linux",
				LabelMembershipType: fleet.LabelMembershipTypeManual,
			},
		},
	}, http.StatusInternalServerError, &applyResp)

	// apply a valid label spec
	s.DoJSON("POST", "/api/v1/fleet/spec/labels", applyLabelSpecsRequest{
		Specs: []*fleet.LabelSpec{
			{
				Name:                name,
				Query:               "select 1",
				Platform:            "linux",
				LabelMembershipType: fleet.LabelMembershipTypeDynamic,
			},
		},
	}, http.StatusOK, &applyResp)

	// list label specs, has the newly created one
	s.DoJSON("GET", "/api/v1/fleet/spec/labels", nil, http.StatusOK, &listResp)
	assert.Len(t, listResp.Specs, builtInsCount+1)

	// get a specific label spec
	var getResp getLabelSpecResponse
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/spec/labels/%s", url.PathEscape(name)), nil, http.StatusOK, &getResp)
	assert.Equal(t, name, getResp.Spec.Name)

	// get a non-existing label spec
	s.DoJSON("GET", "/api/v1/fleet/spec/labels/zzz", nil, http.StatusNotFound, &getResp)
}

func (s *integrationTestSuite) TestUsers() {
	t := s.T()

	// list existing users
	var listResp listUsersResponse
	s.DoJSON("GET", "/api/v1/fleet/users", nil, http.StatusOK, &listResp)
	assert.Len(t, listResp.Users, len(s.users))

	// test available teams returned by `/me` endpoint for existing user
	var getMeResp getUserResponse
	ssn := createSession(t, 1, s.ds)
	resp := s.DoRawWithHeaders("GET", "/api/v1/fleet/me", []byte(""), http.StatusOK, map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", ssn.Key),
	})
	err := json.NewDecoder(resp.Body).Decode(&getMeResp)
	require.NoError(t, err)
	assert.Equal(t, uint(1), getMeResp.User.ID)
	assert.NotNil(t, getMeResp.User.GlobalRole)
	assert.Len(t, getMeResp.User.Teams, 0)
	assert.Len(t, getMeResp.AvailableTeams, 0)

	// create a new user
	var createResp createUserResponse
	params := fleet.UserPayload{
		Name:       ptr.String("extra"),
		Email:      ptr.String("extra@asd.com"),
		Password:   ptr.String("pass"),
		GlobalRole: ptr.String(fleet.RoleObserver),
	}
	s.DoJSON("POST", "/api/v1/fleet/users/admin", params, http.StatusOK, &createResp)
	assert.NotZero(t, createResp.User.ID)
	u := *createResp.User

	// login as that user and check that teams info is empty
	var loginResp loginResponse
	s.DoJSON("POST", "/api/v1/fleet/login", params, http.StatusOK, &loginResp)
	require.Equal(t, loginResp.User.ID, u.ID)
	assert.Len(t, loginResp.User.Teams, 0)
	assert.Len(t, loginResp.AvailableTeams, 0)

	// get that user from `/users` endpoint and check that teams info is empty
	var getResp getUserResponse
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/users/%d", u.ID), nil, http.StatusOK, &getResp)
	assert.Equal(t, u.ID, getResp.User.ID)
	assert.Len(t, getResp.User.Teams, 0)
	assert.Len(t, getResp.AvailableTeams, 0)

	// get non-existing user
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/users/%d", u.ID+1), nil, http.StatusNotFound, &getResp)

	// modify that user - simple name change
	var modResp modifyUserResponse
	params = fleet.UserPayload{
		Name: ptr.String("extraz"),
	}
	s.DoJSON("PATCH", fmt.Sprintf("/api/v1/fleet/users/%d", u.ID), params, http.StatusOK, &modResp)
	assert.Equal(t, u.ID, modResp.User.ID)
	assert.Equal(t, u.Name+"z", modResp.User.Name)

	// modify user - email change, password does not match
	params = fleet.UserPayload{
		Email:    ptr.String("extra2@asd.com"),
		Password: ptr.String("wrongpass"),
	}
	s.DoJSON("PATCH", fmt.Sprintf("/api/v1/fleet/users/%d", u.ID), params, http.StatusForbidden, &modResp)

	// modify user - email change, password ok
	params = fleet.UserPayload{
		Email:    ptr.String("extra2@asd.com"),
		Password: ptr.String("pass"),
	}
	s.DoJSON("PATCH", fmt.Sprintf("/api/v1/fleet/users/%d", u.ID), params, http.StatusOK, &modResp)
	assert.Equal(t, u.ID, modResp.User.ID)
	assert.NotEqual(t, u.ID, modResp.User.Email)

	// modify invalid user
	params = fleet.UserPayload{
		Name: ptr.String("nosuchuser"),
	}
	s.DoJSON("PATCH", fmt.Sprintf("/api/v1/fleet/users/%d", u.ID+1), params, http.StatusNotFound, &modResp)

	// require a password reset
	var reqResetResp requirePasswordResetResponse
	s.DoJSON("POST", fmt.Sprintf("/api/v1/fleet/users/%d/require_password_reset", u.ID), map[string]bool{"require": true}, http.StatusOK, &reqResetResp)
	assert.Equal(t, u.ID, reqResetResp.User.ID)
	assert.True(t, reqResetResp.User.AdminForcedPasswordReset)

	// require a password reset to invalid user
	s.DoJSON("POST", fmt.Sprintf("/api/v1/fleet/users/%d/require_password_reset", u.ID+1), map[string]bool{"require": true}, http.StatusNotFound, &reqResetResp)

	// delete user
	var delResp deleteUserResponse
	s.DoJSON("DELETE", fmt.Sprintf("/api/v1/fleet/users/%d", u.ID), nil, http.StatusOK, &delResp)

	// delete invalid user
	s.DoJSON("DELETE", fmt.Sprintf("/api/v1/fleet/users/%d", u.ID), nil, http.StatusNotFound, &delResp)
}

func (s *integrationTestSuite) TestGlobalPoliciesAutomationConfig() {
	t := s.T()

	gpParams := globalPolicyRequest{
		Name:  "policy1",
		Query: "select 41;",
	}
	gpResp := globalPolicyResponse{}
	s.DoJSON("POST", "/api/v1/fleet/global/policies", gpParams, http.StatusOK, &gpResp)
	require.NotNil(t, gpResp.Policy)

	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"webhook_settings": {
    		"failing_policies_webhook": {
     	 		"enable_failing_policies_webhook": true,
     	 		"destination_url": "http://some/url",
     			"policy_ids": [%d],
				"host_batch_size": 1000
    		},
    		"interval": "1h"
  		}
	}`, gpResp.Policy.ID)), http.StatusOK)

	config := s.getConfig()
	require.True(t, config.WebhookSettings.FailingPoliciesWebhook.Enable)
	require.Equal(t, "http://some/url", config.WebhookSettings.FailingPoliciesWebhook.DestinationURL)
	require.Equal(t, []uint{gpResp.Policy.ID}, config.WebhookSettings.FailingPoliciesWebhook.PolicyIDs)
	require.Equal(t, 1*time.Hour, config.WebhookSettings.Interval.Duration)
	require.Equal(t, 1000, config.WebhookSettings.FailingPoliciesWebhook.HostBatchSize)

	deletePolicyParams := deleteGlobalPoliciesRequest{IDs: []uint{gpResp.Policy.ID}}
	deletePolicyResp := deleteGlobalPoliciesResponse{}
	s.DoJSON("POST", "/api/v1/fleet/global/policies/delete", deletePolicyParams, http.StatusOK, &deletePolicyResp)

	config = s.getConfig()
	require.Empty(t, config.WebhookSettings.FailingPoliciesWebhook.PolicyIDs)
}

func (s *integrationTestSuite) TestVulnerabilitiesWebhookConfig() {
	t := s.T()

	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(`{
		"webhook_settings": {
    		"vulnerabilities_webhook": {
     	 		"enable_vulnerabilities_webhook": true,
     	 		"destination_url": "http://some/url",
     	 		"host_batch_size": 1234
    		},
    		"interval": "1h"
  		}
	}`), http.StatusOK)

	config := s.getConfig()
	require.True(t, config.WebhookSettings.VulnerabilitiesWebhook.Enable)
	require.Equal(t, "http://some/url", config.WebhookSettings.VulnerabilitiesWebhook.DestinationURL)
	require.Equal(t, 1234, config.WebhookSettings.VulnerabilitiesWebhook.HostBatchSize)
	require.Equal(t, 1*time.Hour, config.WebhookSettings.Interval.Duration)
}

func (s *integrationTestSuite) TestQueriesBadRequests() {
	t := s.T()

	reqQuery := &fleet.QueryPayload{
		Name:  ptr.String("existing query"),
		Query: ptr.String("select 42;"),
	}
	createQueryResp := createQueryResponse{}
	s.DoJSON("POST", "/api/v1/fleet/queries", reqQuery, http.StatusOK, &createQueryResp)
	require.NotNil(t, createQueryResp.Query)
	existingQueryID := createQueryResp.Query.ID
	defer cleanupQuery(s, existingQueryID)

	for _, tc := range []struct {
		tname string
		name  string
		query string
	}{
		{
			tname: "empty name",
			name:  " ", // #3704
			query: "select 42;",
		},
		{
			tname: "empty query",
			name:  "Some name",
			query: "",
		},
		{
			tname: "Invalid query",
			name:  "Invalid query",
			query: "ATTACH 'foo' AS bar;",
		},
	} {
		t.Run(tc.tname, func(t *testing.T) {
			reqQuery := &fleet.QueryPayload{
				Name:  ptr.String(tc.name),
				Query: ptr.String(tc.query),
			}
			createQueryResp := createQueryResponse{}
			s.DoJSON("POST", "/api/v1/fleet/queries", reqQuery, http.StatusBadRequest, &createQueryResp)
			require.Nil(t, createQueryResp.Query)

			payload := fleet.QueryPayload{
				Name:  ptr.String(tc.name),
				Query: ptr.String(tc.query),
			}
			mResp := modifyQueryResponse{}
			s.DoJSON("PATCH", fmt.Sprintf("/api/v1/fleet/queries/%d", existingQueryID), &payload, http.StatusBadRequest, &mResp)
			require.Nil(t, mResp.Query)
		})
	}
}

func (s *integrationTestSuite) TestPacksBadRequests() {
	t := s.T()

	reqPacks := &fleet.PackPayload{
		Name: ptr.String("existing pack"),
	}
	createPackResp := createPackResponse{}
	s.DoJSON("POST", "/api/v1/fleet/packs", reqPacks, http.StatusOK, &createPackResp)
	existingPackID := createPackResp.Pack.ID

	for _, tc := range []struct {
		tname string
		name  string
	}{
		{
			tname: "empty name",
			name:  " ", // #3704
		},
	} {
		t.Run(tc.tname, func(t *testing.T) {
			reqQuery := &fleet.PackPayload{
				Name: ptr.String(tc.name),
			}
			createPackResp := createQueryResponse{}
			s.DoJSON("POST", "/api/v1/fleet/packs", reqQuery, http.StatusBadRequest, &createPackResp)

			payload := fleet.PackPayload{
				Name: ptr.String(tc.name),
			}
			mResp := modifyPackResponse{}
			s.DoJSON("PATCH", fmt.Sprintf("/api/v1/fleet/packs/%d", existingPackID), &payload, http.StatusBadRequest, &mResp)
		})
	}
}

func (s *integrationTestSuite) TestTeamsEndpointsWithoutLicense() {
	t := s.T()

	// list teams, none
	var listResp listTeamsResponse
	s.DoJSON("GET", "/api/v1/fleet/teams", nil, http.StatusPaymentRequired, &listResp)
	assert.Len(t, listResp.Teams, 0)

	// get team
	var getResp getTeamResponse
	s.DoJSON("GET", "/api/v1/fleet/teams/123", nil, http.StatusPaymentRequired, &getResp)
	assert.Nil(t, getResp.Team)

	// create team
	var tmResp teamResponse
	s.DoJSON("POST", "/api/v1/fleet/teams", &createTeamRequest{}, http.StatusPaymentRequired, &tmResp)
	assert.Nil(t, tmResp.Team)

	// modify team
	s.DoJSON("PATCH", "/api/v1/fleet/teams/123", fleet.TeamPayload{}, http.StatusPaymentRequired, &tmResp)
	assert.Nil(t, tmResp.Team)

	// delete team
	var delResp deleteTeamResponse
	s.DoJSON("DELETE", "/api/v1/fleet/teams/123", nil, http.StatusPaymentRequired, &delResp)

	// apply team specs
	var specResp applyTeamSpecsResponse
	teamSpecs := applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{Name: "newteam", Secrets: []fleet.EnrollSecret{{Secret: "ABC"}}}}}
	s.DoJSON("POST", "/api/v1/fleet/spec/teams", teamSpecs, http.StatusPaymentRequired, &specResp)

	// modify team agent options
	s.DoJSON("POST", "/api/v1/fleet/teams/123/agent_options", nil, http.StatusPaymentRequired, &tmResp)
	assert.Nil(t, tmResp.Team)

	// list team users
	var usersResp listUsersResponse
	s.DoJSON("GET", "/api/v1/fleet/teams/123/users", nil, http.StatusPaymentRequired, &usersResp, "page", "1")
	assert.Len(t, usersResp.Users, 0)

	// add team users
	s.DoJSON("PATCH", "/api/v1/fleet/teams/123/users", modifyTeamUsersRequest{Users: []fleet.TeamUser{{User: fleet.User{ID: 1}}}}, http.StatusPaymentRequired, &tmResp)
	assert.Nil(t, tmResp.Team)

	// delete team users
	s.DoJSON("DELETE", "/api/v1/fleet/teams/123/users", modifyTeamUsersRequest{Users: []fleet.TeamUser{{User: fleet.User{ID: 1}}}}, http.StatusPaymentRequired, &tmResp)
	assert.Nil(t, tmResp.Team)

	// get team enroll secrets
	var secResp teamEnrollSecretsResponse
	s.DoJSON("GET", "/api/v1/fleet/teams/123/secrets", nil, http.StatusPaymentRequired, &secResp)
	assert.Len(t, secResp.Secrets, 0)

	// modify team enroll secrets
	s.DoJSON("PATCH", "/api/v1/fleet/teams/123/secrets", modifyTeamEnrollSecretsRequest{Secrets: []fleet.EnrollSecret{{Secret: "DEF"}}}, http.StatusPaymentRequired, &secResp)
	assert.Len(t, secResp.Secrets, 0)
}

// TestGlobalPoliciesBrowsing tests that team users can browse (read) global policies (see #3722).
func (s *integrationTestSuite) TestGlobalPoliciesBrowsing() {
	t := s.T()

	team, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		ID:          42,
		Name:        "team_for_global_policies_browsing",
		Description: "desc team1",
	})
	require.NoError(t, err)

	gpParams0 := globalPolicyRequest{
		Name:  "global policy",
		Query: "select * from osquery;",
	}
	gpResp0 := globalPolicyResponse{}
	s.DoJSON("POST", "/api/v1/fleet/global/policies", gpParams0, http.StatusOK, &gpResp0)
	require.NotNil(t, gpResp0.Policy)

	email := "team.observer@example.com"
	teamObserver := &fleet.User{
		Name:       "team observer user",
		Email:      email,
		GlobalRole: nil,
		Teams: []fleet.UserTeam{
			{
				Team: *team,
				Role: fleet.RoleObserver,
			},
		},
	}
	password := "p4ssw0rd."
	require.NoError(t, teamObserver.SetPassword(password, 10, 10))
	_, err = s.ds.NewUser(context.Background(), teamObserver)
	require.NoError(t, err)

	oldToken := s.token
	s.token = s.getTestToken(email, password)
	t.Cleanup(func() {
		s.token = oldToken
	})

	policiesResponse := listGlobalPoliciesResponse{}
	s.DoJSON("GET", "/api/v1/fleet/global/policies", nil, http.StatusOK, &policiesResponse)
	require.Len(t, policiesResponse.Policies, 1)
	assert.Equal(t, "global policy", policiesResponse.Policies[0].Name)
	assert.Equal(t, "select * from osquery;", policiesResponse.Policies[0].Query)
}

func (s *integrationTestSuite) TestTeamPoliciesTeamNotExists() {
	t := s.T()

	teamPoliciesResponse := listTeamPoliciesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/teams/%d/policies", 9999999), nil, http.StatusNotFound, &teamPoliciesResponse)
	require.Len(t, teamPoliciesResponse.Policies, 0)
}

func (s *integrationTestSuite) TestSessionInfo() {
	t := s.T()

	ssn := createSession(t, 1, s.ds)

	var meResp getUserResponse
	resp := s.DoRawWithHeaders("GET", "/api/v1/fleet/me", nil, http.StatusOK, map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", ssn.Key),
	})
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&meResp))
	assert.Equal(t, uint(1), meResp.User.ID)

	// get info about session
	var getResp getInfoAboutSessionResponse
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/sessions/%d", ssn.ID), nil, http.StatusOK, &getResp)
	assert.Equal(t, ssn.ID, getResp.SessionID)
	assert.Equal(t, uint(1), getResp.UserID)

	// get info about session - non-existing: appears to deliberately return 500 due to forbidden,
	// which takes precedence vs the not found returned by the datastore (it still shouldn't be a
	// 500 though).
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/sessions/%d", ssn.ID+1), nil, http.StatusInternalServerError, &getResp)

	// delete session
	var delResp deleteSessionResponse
	s.DoJSON("DELETE", fmt.Sprintf("/api/v1/fleet/sessions/%d", ssn.ID), nil, http.StatusOK, &delResp)

	// delete session - non-existing: again, 500 due to forbidden instead of 404.
	s.DoJSON("DELETE", fmt.Sprintf("/api/v1/fleet/sessions/%d", ssn.ID), nil, http.StatusInternalServerError, &delResp)
}

func (s *integrationTestSuite) TestAppConfig() {
	t := s.T()

	// get the app config
	var acResp appConfigResponse
	s.DoJSON("GET", "/api/v1/fleet/config", nil, http.StatusOK, &acResp)
	assert.Equal(t, "free", acResp.License.Tier)
	assert.Equal(t, "", acResp.OrgInfo.OrgName)

	// no server settings set for the URL, so not possible to test the
	// certificate endpoint
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/v1/fleet/config", json.RawMessage(`{
    "org_info": {
        "org_name": "test"
    }
  }`), http.StatusOK, &acResp)
	assert.Equal(t, "test", acResp.OrgInfo.OrgName)

	var verResp versionResponse
	s.DoJSON("GET", "/api/v1/fleet/version", nil, http.StatusOK, &verResp)
	assert.NotEmpty(t, verResp.Branch)

	// get enroll secrets, none yet
	var specResp getEnrollSecretSpecResponse
	s.DoJSON("GET", "/api/v1/fleet/spec/enroll_secret", nil, http.StatusOK, &specResp)
	assert.Empty(t, specResp.Spec.Secrets)

	// apply spec, one secret
	var applyResp applyEnrollSecretSpecResponse
	s.DoJSON("POST", "/api/v1/fleet/spec/enroll_secret", applyEnrollSecretSpecRequest{
		Spec: &fleet.EnrollSecretSpec{
			Secrets: []*fleet.EnrollSecret{{Secret: "XYZ"}},
		},
	}, http.StatusOK, &applyResp)

	// get enroll secrets, one
	s.DoJSON("GET", "/api/v1/fleet/spec/enroll_secret", nil, http.StatusOK, &specResp)
	require.Len(t, specResp.Spec.Secrets, 1)
	assert.Equal(t, "XYZ", specResp.Spec.Secrets[0].Secret)

	// remove secret just to prevent affecting other tests
	s.DoJSON("POST", "/api/v1/fleet/spec/enroll_secret", applyEnrollSecretSpecRequest{
		Spec: &fleet.EnrollSecretSpec{},
	}, http.StatusOK, &applyResp)

	s.DoJSON("GET", "/api/v1/fleet/spec/enroll_secret", nil, http.StatusOK, &specResp)
	require.Len(t, specResp.Spec.Secrets, 0)
}

func (s *integrationTestSuite) TestQuerySpecs() {
	t := s.T()

	// list specs, none yet
	var getSpecsResp getQuerySpecsResponse
	s.DoJSON("GET", "/api/v1/fleet/spec/queries", nil, http.StatusOK, &getSpecsResp)
	assert.Len(t, getSpecsResp.Specs, 0)

	// get unknown one
	var getSpecResp getQuerySpecResponse
	s.DoJSON("GET", "/api/v1/fleet/spec/queries/nonesuch", nil, http.StatusNotFound, &getSpecResp)

	// create some queries via apply specs
	q1 := strings.ReplaceAll(t.Name(), "/", "_")
	q2 := q1 + "_2"
	var applyResp applyQuerySpecsResponse
	s.DoJSON("POST", "/api/v1/fleet/spec/queries", applyQuerySpecsRequest{
		Specs: []*fleet.QuerySpec{
			{Name: q1, Query: "SELECT 1"},
			{Name: q2, Query: "SELECT 2"},
		},
	}, http.StatusOK, &applyResp)

	// get the queries back
	var listQryResp listQueriesResponse
	s.DoJSON("GET", "/api/v1/fleet/queries", nil, http.StatusOK, &listQryResp, "order_key", "name")
	require.Len(t, listQryResp.Queries, 2)
	assert.Equal(t, q1, listQryResp.Queries[0].Name)
	assert.Equal(t, q2, listQryResp.Queries[1].Name)
	q1ID, q2ID := listQryResp.Queries[0].ID, listQryResp.Queries[1].ID

	// list specs
	s.DoJSON("GET", "/api/v1/fleet/spec/queries", nil, http.StatusOK, &getSpecsResp)
	require.Len(t, getSpecsResp.Specs, 2)
	names := []string{getSpecsResp.Specs[0].Name, getSpecsResp.Specs[1].Name}
	assert.ElementsMatch(t, []string{q1, q2}, names)

	// get specific spec
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/spec/queries/%s", q1), nil, http.StatusOK, &getSpecResp)
	assert.Equal(t, getSpecResp.Spec.Query, "SELECT 1")

	// apply specs again - create q3 and update q2
	q3 := q1 + "_3"
	s.DoJSON("POST", "/api/v1/fleet/spec/queries", applyQuerySpecsRequest{
		Specs: []*fleet.QuerySpec{
			{Name: q2, Query: "SELECT -2"},
			{Name: q3, Query: "SELECT 3"},
		},
	}, http.StatusOK, &applyResp)

	// list specs - has 3, not 4 (one was an update)
	s.DoJSON("GET", "/api/v1/fleet/spec/queries", nil, http.StatusOK, &getSpecsResp)
	require.Len(t, getSpecsResp.Specs, 3)
	names = []string{getSpecsResp.Specs[0].Name, getSpecsResp.Specs[1].Name, getSpecsResp.Specs[2].Name}
	assert.ElementsMatch(t, []string{q1, q2, q3}, names)

	// get the queries back again
	s.DoJSON("GET", "/api/v1/fleet/queries", nil, http.StatusOK, &listQryResp, "order_key", "name")
	require.Len(t, listQryResp.Queries, 3)
	assert.Equal(t, q1ID, listQryResp.Queries[0].ID)
	assert.Equal(t, q2ID, listQryResp.Queries[1].ID)
	assert.Equal(t, "SELECT -2", listQryResp.Queries[1].Query)
	q3ID := listQryResp.Queries[2].ID

	// delete all queries created
	var delBatchResp deleteQueriesResponse
	s.DoJSON("POST", "/api/v1/fleet/queries/delete", map[string]interface{}{
		"ids": []uint{q1ID, q2ID, q3ID},
	}, http.StatusOK, &delBatchResp)
	assert.Equal(t, uint(3), delBatchResp.Deleted)
}

func (s *integrationTestSuite) TestPaginateListSoftware() {
	t := s.T()

	// create a few hosts specific to this test
	hosts := make([]*fleet.Host, 20)
	for i := range hosts {
		host, err := s.ds.NewHost(context.Background(), &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now(),
			NodeKey:         t.Name() + strconv.Itoa(i),
			OsqueryHostID:   t.Name() + strconv.Itoa(i),
			UUID:            t.Name() + strconv.Itoa(i),
			Hostname:        t.Name() + "foo" + strconv.Itoa(i) + ".local",
			PrimaryIP:       "192.168.1." + strconv.Itoa(i),
			PrimaryMac:      fmt.Sprintf("30-65-EC-6F-C4-%02d", i),
		})
		require.NoError(t, err)
		require.NotNil(t, host)
		hosts[i] = host
	}

	// create a bunch of software
	sws := make([]fleet.Software, 20)
	for i := range sws {
		sw := fleet.Software{Name: "sw" + strconv.Itoa(i), Version: "0.0." + strconv.Itoa(i), Source: "apps"}
		sws[i] = sw
	}

	// mark them as installed on the hosts, with host at index 0 having all 20,
	// at index 1 having 19, index 2 = 18, etc. until index 19 = 1. So software
	// sws[0] is only used by 1 host, while sws[19] is used by all.
	for i, h := range hosts {
		require.NoError(t, s.ds.UpdateHostSoftware(context.Background(), h.ID, sws[i:]))
		require.NoError(t, s.ds.LoadHostSoftware(context.Background(), h))

		if i == 0 {
			// this host has all software, refresh the list so we have the software.ID filled
			sws = h.Software
		}
	}

	for i, sw := range sws {
		cpe := "somecpe" + strconv.Itoa(i)
		require.NoError(t, s.ds.AddCPEForSoftware(context.Background(), sw, cpe))

		if i < 10 {
			// add CVEs for the first 10 software, which are the least used (lower hosts_count)
			_, err := s.ds.InsertCVEForCPE(context.Background(), fmt.Sprintf("cve-123-123-%03d", i), []string{cpe})
			require.NoError(t, err)
		}
	}

	assertResp := func(resp listSoftwareResponse, want []fleet.Software, ts time.Time, counts ...int) {
		require.Len(t, resp.Software, len(want))
		for i := range resp.Software {
			wantID, gotID := want[i].ID, resp.Software[i].ID
			assert.Equal(t, wantID, gotID)
			wantCount, gotCount := counts[i], resp.Software[i].HostsCount
			assert.Equal(t, wantCount, gotCount)
		}
		if ts.IsZero() {
			assert.Nil(t, resp.CountsUpdatedAt)
		} else {
			require.NotNil(t, resp.CountsUpdatedAt)
			assert.WithinDuration(t, ts, *resp.CountsUpdatedAt, time.Second)
		}
	}

	// no software host counts have been calculated yet, so this returns nothing
	var lsResp listSoftwareResponse
	s.DoJSON("GET", "/api/v1/fleet/software", nil, http.StatusOK, &lsResp, "order_key", "hosts_count", "order_direction", "desc")
	assertResp(lsResp, nil, time.Time{})

	// calculate hosts counts
	hostsCountTs := time.Now().UTC()
	require.NoError(t, s.ds.CalculateHostsPerSoftware(context.Background(), hostsCountTs))

	// now the list software endpoint returns the software, get the first page without vulns
	lsResp = listSoftwareResponse{}
	s.DoJSON("GET", "/api/v1/fleet/software", nil, http.StatusOK, &lsResp, "per_page", "5", "page", "0", "order_key", "hosts_count", "order_direction", "desc")
	assertResp(lsResp, []fleet.Software{sws[19], sws[18], sws[17], sws[16], sws[15]}, hostsCountTs, 20, 19, 18, 17, 16)

	// second page (page=1)
	lsResp = listSoftwareResponse{}
	s.DoJSON("GET", "/api/v1/fleet/software", nil, http.StatusOK, &lsResp, "per_page", "5", "page", "1", "order_key", "hosts_count", "order_direction", "desc")
	assertResp(lsResp, []fleet.Software{sws[14], sws[13], sws[12], sws[11], sws[10]}, hostsCountTs, 15, 14, 13, 12, 11)

	// third page (page=2)
	lsResp = listSoftwareResponse{}
	s.DoJSON("GET", "/api/v1/fleet/software", nil, http.StatusOK, &lsResp, "per_page", "5", "page", "2", "order_key", "hosts_count", "order_direction", "desc")
	assertResp(lsResp, []fleet.Software{sws[9], sws[8], sws[7], sws[6], sws[5]}, hostsCountTs, 10, 9, 8, 7, 6)

	// last page (page=3)
	lsResp = listSoftwareResponse{}
	s.DoJSON("GET", "/api/v1/fleet/software", nil, http.StatusOK, &lsResp, "per_page", "5", "page", "3", "order_key", "hosts_count", "order_direction", "desc")
	assertResp(lsResp, []fleet.Software{sws[4], sws[3], sws[2], sws[1], sws[0]}, hostsCountTs, 5, 4, 3, 2, 1)

	// past the end
	lsResp = listSoftwareResponse{}
	s.DoJSON("GET", "/api/v1/fleet/software", nil, http.StatusOK, &lsResp, "per_page", "5", "page", "4", "order_key", "hosts_count", "order_direction", "desc")
	assertResp(lsResp, nil, time.Time{})

	// no explicit sort order, defaults to hosts_count DESC
	lsResp = listSoftwareResponse{}
	s.DoJSON("GET", "/api/v1/fleet/software", nil, http.StatusOK, &lsResp, "per_page", "2", "page", "0")
	assertResp(lsResp, []fleet.Software{sws[19], sws[18]}, hostsCountTs, 20, 19)

	// hosts_count ascending
	lsResp = listSoftwareResponse{}
	s.DoJSON("GET", "/api/v1/fleet/software", nil, http.StatusOK, &lsResp, "per_page", "3", "page", "0", "order_key", "hosts_count", "order_direction", "asc")
	assertResp(lsResp, []fleet.Software{sws[0], sws[1], sws[2]}, hostsCountTs, 1, 2, 3)

	// vulnerable software only
	lsResp = listSoftwareResponse{}
	s.DoJSON("GET", "/api/v1/fleet/software", nil, http.StatusOK, &lsResp, "vulnerable", "true", "per_page", "5", "page", "0", "order_key", "hosts_count", "order_direction", "desc")
	assertResp(lsResp, []fleet.Software{sws[9], sws[8], sws[7], sws[6], sws[5]}, hostsCountTs, 10, 9, 8, 7, 6)

	// vulnerable software only, next page
	lsResp = listSoftwareResponse{}
	s.DoJSON("GET", "/api/v1/fleet/software", nil, http.StatusOK, &lsResp, "vulnerable", "true", "per_page", "5", "page", "1", "order_key", "hosts_count", "order_direction", "desc")
	assertResp(lsResp, []fleet.Software{sws[4], sws[3], sws[2], sws[1], sws[0]}, hostsCountTs, 5, 4, 3, 2, 1)

	// vulnerable software only, past last page
	lsResp = listSoftwareResponse{}
	s.DoJSON("GET", "/api/v1/fleet/software", nil, http.StatusOK, &lsResp, "vulnerable", "true", "per_page", "5", "page", "2", "order_key", "hosts_count", "order_direction", "desc")
	assertResp(lsResp, nil, time.Time{})
}

// creates a session and returns it, its key is to be passed as authorization header.
func createSession(t *testing.T, uid uint, ds fleet.Datastore) *fleet.Session {
	key := make([]byte, 64)
	_, err := rand.Read(key)
	require.NoError(t, err)

	sessionKey := base64.StdEncoding.EncodeToString(key)
	session := &fleet.Session{
		UserID:     uid,
		Key:        sessionKey,
		AccessedAt: time.Now().UTC(),
	}
	ssn, err := ds.NewSession(context.Background(), session)
	require.NoError(t, err)

	return ssn
}

func cleanupQuery(s *integrationTestSuite, queryID uint) {
	var delResp deleteQueryByIDResponse
	s.DoJSON("DELETE", fmt.Sprintf("/api/v1/fleet/queries/id/%d", queryID), nil, http.StatusOK, &delResp)
}
