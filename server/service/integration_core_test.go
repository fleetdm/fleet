package service

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/live_query/live_query_mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/ghodss/yaml"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
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
	s.withServer.lq = live_query_mock.New(s.T())
	s.withServer.SetupSuite("integrationTestSuite")
}

func (s *integrationTestSuite) TearDownTest() {
	s.withServer.commonTearDownTest(s.T())
}

func TestIntegrations(t *testing.T) {
	testingSuite := new(integrationTestSuite)
	testingSuite.s = &testingSuite.Suite
	suite.Run(t, testingSuite)
}

type slowReader struct{}

func (s *slowReader) Read(p []byte) (n int, err error) {
	time.Sleep(3 * time.Second)
	return 0, nil
}

func (s *integrationTestSuite) TestSlowOsqueryHost() {
	t := s.T()
	_, server := RunServerForTestsWithDS(
		t,
		s.ds,
		&TestServerOpts{
			SkipCreateTestUsers: true,
			//nolint:gosec // G112: server is just run for testing this explicit config.
			HTTPServerConfig: &http.Server{ReadTimeout: 2 * time.Second},
		},
	)
	defer func() {
		server.Close()
	}()

	req, err := http.NewRequest("POST", server.URL+"/api/v1/osquery/distributed/write", &slowReader{})
	require.NoError(t, err)

	client := fleethttp.NewClient()

	resp, err := client.Do(req)
	require.NoError(t, err)

	assert.Equal(t, http.StatusRequestTimeout, resp.StatusCode)
}

func (s *integrationTestSuite) TestDistributedReadWithChangedQueries() {
	t := s.T()

	spec := []byte(`
  features:
    enable_software_inventory: true
    enable_host_users: true
    detail_query_overrides:
      users: null
      software_macos: "SELECT * FROM foo;"
      unknown_query: "SELECT * FROM bar;"
`)
	s.applyConfig(spec)

	host, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   ptr.String(t.Name()),
		NodeKey:         ptr.String(t.Name()),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%sfoo.local", t.Name()),
		Platform:        "darwin",
	})
	require.NoError(t, err)

	s.lq.On("QueriesForHost", host.ID).Return(map[string]string{fmt.Sprintf("%d", host.ID): "SELECT 1 FROM osquery;"}, nil)

	// Ensure we can read distributed queries for the host.
	err = s.ds.UpdateHostRefetchRequested(context.Background(), host.ID, true)
	require.NoError(t, err)

	// Get distributed queries for the host.
	req := getDistributedQueriesRequest{NodeKey: *host.NodeKey}
	var dqResp getDistributedQueriesResponse
	s.DoJSON("POST", "/api/osquery/distributed/read", req, http.StatusOK, &dqResp)
	require.NotContains(t, dqResp.Queries, "fleet_detail_query_users")
	require.Contains(t, dqResp.Queries, "fleet_detail_query_software_macos")
	require.Equal(t, "SELECT * FROM foo;", dqResp.Queries["fleet_detail_query_software_macos"])

	err = s.ds.UpdateHostRefetchRequested(context.Background(), host.ID, true)
	require.NoError(t, err)

	spec = []byte(`
  features:
    enable_software_inventory: true
    enable_host_users: true
    detail_query_overrides:
`)
	s.applyConfig(spec)

	// Get distributed queries for the host.
	req = getDistributedQueriesRequest{NodeKey: *host.NodeKey}
	s.DoJSON("POST", "/api/osquery/distributed/read", req, http.StatusOK, &dqResp)
	require.Contains(t, dqResp.Queries, "fleet_detail_query_users")
	require.Contains(t, dqResp.Queries, "fleet_detail_query_software_macos")
	require.Contains(t, dqResp.Queries["fleet_detail_query_software_macos"], "FROM apps")
	require.Contains(t, dqResp.Queries["fleet_detail_query_users"], "FROM users")
}

func (s *integrationTestSuite) TestDoubleUserCreationErrors() {
	t := s.T()

	params := fleet.UserPayload{
		Name:       ptr.String("user1"),
		Email:      ptr.String("email@asd.com"),
		Password:   &test.GoodPassword,
		GlobalRole: ptr.String(fleet.RoleObserver),
	}

	s.Do("POST", "/api/latest/fleet/users/admin", &params, http.StatusOK)
	respSecond := s.Do("POST", "/api/latest/fleet/users/admin", &params, http.StatusConflict)

	assertBodyContains(t, respSecond, `Error 1062: Duplicate entry 'email@asd.com'`)
}

func (s *integrationTestSuite) TestUserWithoutRoleErrors() {
	t := s.T()

	params := fleet.UserPayload{
		Name:     ptr.String("user1"),
		Email:    ptr.String("email@asd.com"),
		Password: ptr.String(test.GoodPassword),
	}

	resp := s.Do("POST", "/api/latest/fleet/users/admin", &params, http.StatusUnprocessableEntity)
	assertErrorCodeAndMessage(t, resp, fleet.ErrNoRoleNeeded, "either global role or team role needs to be defined")
}

func (s *integrationTestSuite) TestUserEmailValidation() {
	params := fleet.UserPayload{
		Name:       ptr.String("user_invalid_email"),
		Email:      ptr.String("invalid"),
		Password:   &test.GoodPassword,
		GlobalRole: ptr.String(fleet.RoleObserver),
	}

	s.Do("POST", "/api/latest/fleet/users/admin", &params, http.StatusUnprocessableEntity)

	params.Email = ptr.String("user_valid_mail@example.com")
	s.Do("POST", "/api/latest/fleet/users/admin", &params, http.StatusOK)
}

func (s *integrationTestSuite) TestUserWithWrongRoleErrors() {
	t := s.T()

	params := fleet.UserPayload{
		Name:       ptr.String("user1"),
		Email:      ptr.String("email@asd.com"),
		Password:   ptr.String(test.GoodPassword),
		GlobalRole: ptr.String("wrongrole"),
	}
	resp := s.Do("POST", "/api/latest/fleet/users/admin", &params, http.StatusUnprocessableEntity)
	assertErrorCodeAndMessage(t, resp, fleet.ErrNoRoleNeeded, "invalid global role: wrongrole")
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
		Password: ptr.String(test.GoodPassword),
		Teams:    &teams,
	}
	resp := s.Do("POST", "/api/latest/fleet/users/admin", &params, http.StatusUnprocessableEntity)
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
	s.DoJSON("POST", "/api/latest/fleet/queries", &params, http.StatusOK, &createQueryResp)
	defer cleanupQuery(s, createQueryResp.Query.ID)

	activities := listActivitiesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &activities)

	assert.GreaterOrEqual(t, len(activities.Activities), 1)
	found := false
	for _, activity := range activities.Activities {
		if activity.Type == "created_saved_query" {
			found = true
			assert.Equal(t, "Test Name admin1@example.com", *activity.ActorFullName)
			require.NotNil(t, activity.ActorGravatar)
			assert.Equal(t, "http://iii.com", *activity.ActorGravatar)
		}
	}
	require.True(t, found)
}

func (s *integrationTestSuite) TestPolicyDeletionLogsActivity() {
	t := s.T()

	admin1 := s.users["admin1@example.com"]
	admin1.GravatarURL = "http://iii.com"
	err := s.ds.SaveUser(context.Background(), &admin1)
	require.NoError(t, err)

	testPolicies := []fleet.PolicyPayload{{
		Name:  "policy1",
		Query: "select * from time;",
	}, {
		Name:  "policy2",
		Query: "select * from osquery_info;",
	}}

	var policyIDs []uint
	for _, policy := range testPolicies {
		var resp globalPolicyResponse
		s.DoJSON("POST", "/api/latest/fleet/policies", policy, http.StatusOK, &resp)
		policyIDs = append(policyIDs, resp.Policy.PolicyData.ID)
	}

	// critical is premium only.
	s.DoJSON("POST", "/api/latest/fleet/policies", fleet.PolicyPayload{
		Name:     "policy3",
		Query:    "select * from time;",
		Critical: true,
	}, http.StatusBadRequest, new(struct{}))

	prevActivities := listActivitiesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &prevActivities)
	require.GreaterOrEqual(t, len(prevActivities.Activities), 2)

	var deletePoliciesResp deleteGlobalPoliciesResponse
	s.DoJSON("POST", "/api/latest/fleet/policies/delete", deleteGlobalPoliciesRequest{policyIDs}, http.StatusOK, &deletePoliciesResp)
	require.Equal(t, len(policyIDs), len(deletePoliciesResp.Deleted))

	newActivities := listActivitiesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &newActivities)
	require.Equal(t, len(newActivities.Activities), (len(prevActivities.Activities) + 2))

	var prevDeletes []*fleet.Activity
	for _, a := range prevActivities.Activities {
		if a.Type == "deleted_policy" {
			prevDeletes = append(prevDeletes, a)
		}
	}
	var newDeletes []*fleet.Activity
	for _, a := range newActivities.Activities {
		if a.Type == "deleted_policy" {
			newDeletes = append(newDeletes, a)
		}
	}
	require.Equal(t, len(newDeletes), (len(prevDeletes) + 2))

	type policyDetails struct {
		PolicyID   uint   `json:"policy_id"`
		PolicyName string `json:"policy_name"`
	}
	for _, id := range policyIDs {
		found := false
		for _, d := range newDeletes {
			var details policyDetails
			err := json.Unmarshal([]byte(*d.Details), &details)
			require.NoError(t, err)
			require.NotNil(t, details.PolicyID)
			if id == details.PolicyID {
				found = true
			}

		}
		require.True(t, found)
	}
	for _, p := range testPolicies {
		found := false
		for _, d := range newDeletes {
			var details policyDetails
			err := json.Unmarshal([]byte(*d.Details), &details)
			require.NoError(t, err)
			require.NotNil(t, details.PolicyName)
			if p.Name == details.PolicyName {
				found = true
			}

		}
		require.True(t, found)
	}
}

func (s *integrationTestSuite) TestAppConfigAdditionalQueriesCanBeRemoved() {
	t := s.T()

	spec := []byte(`
  host_expiry_settings:
    host_expiry_enabled: true
    host_expiry_window: 0
  features:
    additional_queries:
      time: SELECT * FROM time
    enable_host_users: true
`)
	s.applyConfig(spec)

	spec = []byte(`
  features:
    enable_host_users: true
    additional_queries: null
`)
	s.applyConfig(spec)

	config := s.getConfig()
	assert.Nil(t, config.Features.AdditionalQueries)
	assert.True(t, config.HostExpirySettings.HostExpiryEnabled)
}

func (s *integrationTestSuite) TestAppConfigDetailQueriesOverrides() {
	t := s.T()

	spec := []byte(`
  features:
    additional_queries:
      time: SELECT * FROM time
    enable_host_users: true
    detail_query_overrides:
      users: null
      software_linux: "select * from blah;"
`)
	s.applyConfig(spec)

	config := s.getConfig()
	require.NotNil(t, config.Features.DetailQueryOverrides)
	require.Nil(t, config.Features.DetailQueryOverrides["users"])
	require.NotNil(t, config.Features.DetailQueryOverrides["software_linux"])
	require.Equal(t, "select * from blah;", *config.Features.DetailQueryOverrides["software_linux"])
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

func (s *integrationTestSuite) TestAppConfigDeprecatedFields() {
	t := s.T()

	spec := []byte(`
  host_settings:
    additional_queries:
      time: SELECT * FROM time
    enable_host_users: true
    enable_software_inventory: true
`)
	s.applyConfig(spec)
	config := s.getConfig()
	require.NotNil(t, config.Features.AdditionalQueries)
	require.True(t, config.Features.EnableHostUsers)
	require.True(t, config.Features.EnableSoftwareInventory)

	spec = []byte(`
  host_settings:
    additional_queries: null
    enable_host_users: false
    enable_software_inventory: false
`)
	s.applyConfig(spec)
	config = s.getConfig()
	require.Nil(t, config.Features.AdditionalQueries)
	require.False(t, config.Features.EnableHostUsers)
	require.False(t, config.Features.EnableSoftwareInventory)

	// Test raw API interactions
	appConfigSpec := map[string]map[string]bool{
		"host_settings":   {"enable_software_inventory": true},
		"server_settings": {"enable_analytics": false},
	}
	s.Do("PATCH", "/api/latest/fleet/config", appConfigSpec, http.StatusOK)
	config = s.getConfig()
	require.True(t, config.Features.EnableSoftwareInventory)

	// Skip our serialization mechanism, to make sure an old config stored in the DB is still valid
	var previousRawConfig string
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		err := sqlx.GetContext(context.Background(), q, &previousRawConfig, "SELECT json_value FROM app_config_json")
		if err != nil {
			return err
		}
		insertAppConfigQuery := `INSERT INTO app_config_json(json_value) VALUES(?) ON DUPLICATE KEY UPDATE json_value = VALUES(json_value)`
		_, err = q.ExecContext(context.Background(), insertAppConfigQuery, `
    {
      "host_settings": {
        "enable_host_users": false,
        "enable_software_inventory": true,
        "additional_queries": { "foo": "bar" }
      }
    }`)
		return err
	})

	var resp appConfigResponse
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &resp)
	require.False(t, resp.Features.EnableHostUsers)
	require.True(t, resp.Features.EnableSoftwareInventory)
	require.NotNil(t, resp.Features.AdditionalQueries)

	// restore the previous appconfig so that other tests are not impacted
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		insertAppConfigQuery := `INSERT INTO app_config_json(json_value) VALUES(?) ON DUPLICATE KEY UPDATE json_value = VALUES(json_value)`
		_, err := q.ExecContext(context.Background(), insertAppConfigQuery, previousRawConfig)
		return err
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

	s.Do("POST", "/api/latest/fleet/users/roles/spec", &userRoleSpec, http.StatusOK)

	user, err = s.ds.UserByEmail(context.Background(), email)
	require.NoError(t, err)
	require.Len(t, user.Teams, 1)
	assert.Equal(t, fleet.RoleMaintainer, user.Teams[0].Role)

	spec = []byte(fmt.Sprintf(`
  roles:
    %s:
      global_role: null
      teams:
      - role: maintainer
        team: non-existent
`,
		email))
	userRoleSpec = applyUserRoleSpecsRequest{}
	err = yaml.Unmarshal(spec, &userRoleSpec.Spec)
	require.NoError(t, err)
	s.Do("POST", "/api/latest/fleet/users/roles/spec", &userRoleSpec, http.StatusBadRequest)
}

func (s *integrationTestSuite) TestGlobalSchedule() {
	t := s.T()

	// list the existing global schedules (none yet)
	gs := fleet.GlobalSchedulePayload{}
	s.DoJSON("GET", "/api/latest/fleet/schedule", nil, http.StatusOK, &gs)
	require.Len(t, gs.GlobalSchedule, 0)

	// create a query that can be scheduled
	qr, err := s.ds.NewQuery(context.Background(), &fleet.Query{
		Name:           "TestQuery1",
		Description:    "Some description",
		Query:          "select * from osquery;",
		ObserverCanRun: true,
		Saved:          true,
	})
	require.NoError(t, err)

	// schedule that query
	gsParams := fleet.ScheduledQueryPayload{QueryID: ptr.Uint(qr.ID), Interval: ptr.Uint(42)}
	r := globalScheduleQueryResponse{}
	s.DoJSON("POST", "/api/latest/fleet/schedule", gsParams, http.StatusOK, &r)

	// list the scheduled queries, get the one just created
	gs = fleet.GlobalSchedulePayload{}
	s.DoJSON("GET", "/api/latest/fleet/schedule", nil, http.StatusOK, &gs)
	require.Len(t, gs.GlobalSchedule, 1)
	assert.Equal(t, uint(42), gs.GlobalSchedule[0].Interval)
	assert.Contains(t, gs.GlobalSchedule[0].Name, "Copy of TestQuery1 (")
	id := gs.GlobalSchedule[0].ID

	// list page 2, should be empty
	s.DoJSON("GET", "/api/latest/fleet/schedule", nil, http.StatusOK, &gs, "page", "2", "per_page", "4")
	require.Len(t, gs.GlobalSchedule, 0)

	// update the scheduled query
	gs = fleet.GlobalSchedulePayload{}
	gsParams = fleet.ScheduledQueryPayload{Interval: ptr.Uint(55)}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/schedule/%d", id), gsParams, http.StatusOK, &gs)

	// update a non-existing schedule
	gsParams = fleet.ScheduledQueryPayload{Interval: ptr.Uint(66)}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/schedule/%d", id+1), gsParams, http.StatusNotFound, &gs)

	// read back that updated scheduled query
	gs = fleet.GlobalSchedulePayload{}
	s.DoJSON("GET", "/api/latest/fleet/schedule", nil, http.StatusOK, &gs)
	require.Len(t, gs.GlobalSchedule, 1)
	assert.Equal(t, id, gs.GlobalSchedule[0].ID)
	assert.Equal(t, uint(55), gs.GlobalSchedule[0].Interval)

	// delete the scheduled query
	r = globalScheduleQueryResponse{}
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/schedule/%d", id), nil, http.StatusOK, &r)

	// delete a non-existing schedule
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/schedule/%d", id+1), nil, http.StatusNotFound, &r)

	// list the scheduled queries, back to none
	gs = fleet.GlobalSchedulePayload{}
	s.DoJSON("GET", "/api/latest/fleet/schedule", nil, http.StatusOK, &gs)
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
	s.DoJSON("POST", "/api/latest/fleet/translate", &params, http.StatusOK, &payload)
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
		NodeKey:         ptr.String(t.Name() + "1"),
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
	_, err = s.ds.UpdateHostSoftware(context.Background(), host.ID, software)
	require.NoError(t, err)
	require.NoError(t, s.ds.LoadHostSoftware(context.Background(), host, false))

	soft1 := host.Software[0]
	if soft1.Name != "bar" {
		soft1 = host.Software[1]
	}

	cpes := []fleet.SoftwareCPE{{SoftwareID: soft1.ID, CPE: "somecpe"}}
	_, err = s.ds.UpsertSoftwareCPEs(context.Background(), cpes)
	require.NoError(t, err)

	// Reload software so that 'GeneratedCPEID is set.
	require.NoError(t, s.ds.LoadHostSoftware(context.Background(), host, false))
	soft1 = host.Software[0]
	if soft1.Name != "bar" {
		soft1 = host.Software[1]
	}

	inserted, err := s.ds.InsertSoftwareVulnerability(
		context.Background(), fleet.SoftwareVulnerability{
			SoftwareID: soft1.ID,
			CVE:        "cve-123-123-132",
		}, fleet.NVDSource,
	)
	require.NoError(t, err)
	require.True(t, inserted)

	resp := s.Do("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK)
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

	// no software host counts have been calculated yet, so this returns nothing
	var lsResp listSoftwareResponse
	resp = s.Do("GET", "/api/latest/fleet/software", nil, http.StatusOK, "vulnerable", "true", "order_key", "generated_cpe", "order_direction", "desc")
	bodyBytes, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(bodyBytes), `"counts_updated_at": null`)

	require.NoError(t, json.Unmarshal(bodyBytes, &lsResp))
	require.Len(t, lsResp.Software, 0)
	assert.Nil(t, lsResp.CountsUpdatedAt)

	// calculate hosts counts
	hostsCountTs := time.Now().UTC()
	require.NoError(t, s.ds.SyncHostsSoftware(context.Background(), hostsCountTs))

	countReq := countSoftwareRequest{}
	countResp := countSoftwareResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/count", countReq, http.StatusOK, &countResp)
	assert.Equal(t, 3, countResp.Count)

	// the software/count endpoint is different, it doesn't care about hosts counts
	countResp = countSoftwareResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/count", countReq, http.StatusOK, &countResp, "vulnerable", "true", "order_key", "generated_cpe", "order_direction", "desc")
	assert.Equal(t, 1, countResp.Count)

	// now the list software endpoint returns the software
	lsResp = listSoftwareResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software", nil, http.StatusOK, &lsResp, "vulnerable", "true", "order_key", "generated_cpe", "order_direction", "desc")
	require.Len(t, lsResp.Software, 1)
	assert.Equal(t, soft1.ID, lsResp.Software[0].ID)
	assert.Len(t, lsResp.Software[0].Vulnerabilities, 1)
	require.NotNil(t, lsResp.CountsUpdatedAt)
	assert.WithinDuration(t, hostsCountTs, *lsResp.CountsUpdatedAt, time.Second)

	// the count endpoint still returns 1
	countResp = countSoftwareResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/count", countReq, http.StatusOK, &countResp, "vulnerable", "true", "order_key", "generated_cpe", "order_direction", "desc")
	assert.Equal(t, 1, countResp.Count)

	// default sort, not only vulnerable
	lsResp = listSoftwareResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software", nil, http.StatusOK, &lsResp)
	require.True(t, len(lsResp.Software) >= len(software))
	require.NotNil(t, lsResp.CountsUpdatedAt)
	assert.WithinDuration(t, hostsCountTs, *lsResp.CountsUpdatedAt, time.Second)

	// request with a per_page limit (see #4058)
	lsResp = listSoftwareResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software", nil, http.StatusOK, &lsResp, "page", "0", "per_page", "2", "order_key", "hosts_count", "order_direction", "desc")
	require.Len(t, lsResp.Software, 2)
	require.NotNil(t, lsResp.CountsUpdatedAt)
	assert.WithinDuration(t, hostsCountTs, *lsResp.CountsUpdatedAt, time.Second)

	// request next page, with per_page limit
	lsResp = listSoftwareResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software", nil, http.StatusOK, &lsResp, "per_page", "2", "page", "1", "order_key", "hosts_count", "order_direction", "desc")
	require.Len(t, lsResp.Software, 1)
	require.NotNil(t, lsResp.CountsUpdatedAt)
	assert.WithinDuration(t, hostsCountTs, *lsResp.CountsUpdatedAt, time.Second)

	// request one past the last page
	lsResp = listSoftwareResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software", nil, http.StatusOK, &lsResp, "per_page", "2", "page", "2", "order_key", "hosts_count", "order_direction", "desc")
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
			OsqueryHostID:   ptr.String(fmt.Sprintf("%s%d", t.Name(), i)),
			NodeKey:         ptr.String(fmt.Sprintf("%s%d", t.Name(), i)),
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
	s.DoJSON("POST", "/api/latest/fleet/policies", gpParams, http.StatusOK, &gpResp)
	require.NotNil(t, gpResp.Policy)
	assert.Equal(t, qr.Name, gpResp.Policy.Name)
	assert.Equal(t, qr.Query, gpResp.Policy.Query)
	assert.Equal(t, qr.Description, gpResp.Policy.Description)
	require.NotNil(t, gpResp.Policy.Resolution)
	assert.Equal(t, "some global resolution", *gpResp.Policy.Resolution)

	policiesResponse := listGlobalPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/policies", nil, http.StatusOK, &policiesResponse)
	require.Len(t, policiesResponse.Policies, 1)
	assert.Equal(t, qr.Name, policiesResponse.Policies[0].Name)
	assert.Equal(t, qr.Query, policiesResponse.Policies[0].Query)
	assert.Equal(t, qr.Description, policiesResponse.Policies[0].Description)

	// Get an unexistent policy
	s.Do("GET", fmt.Sprintf("/api/latest/fleet/policies/%d", 9999), nil, http.StatusNotFound)

	singlePolicyResponse := getPolicyByIDResponse{}
	singlePolicyURL := fmt.Sprintf("/api/latest/fleet/policies/%d", policiesResponse.Policies[0].ID)
	s.DoJSON("GET", singlePolicyURL, nil, http.StatusOK, &singlePolicyResponse)
	assert.Equal(t, qr.Name, singlePolicyResponse.Policy.Name)
	assert.Equal(t, qr.Query, singlePolicyResponse.Policy.Query)
	assert.Equal(t, qr.Description, singlePolicyResponse.Policy.Description)

	listHostsURL := fmt.Sprintf("/api/latest/fleet/hosts?policy_id=%d", policiesResponse.Policies[0].ID)
	listHostsResp := listHostsResponse{}
	s.DoJSON("GET", listHostsURL, nil, http.StatusOK, &listHostsResp)
	require.Len(t, listHostsResp.Hosts, 3)

	h1 := listHostsResp.Hosts[0]
	h2 := listHostsResp.Hosts[1]

	listHostsURL = fmt.Sprintf("/api/latest/fleet/hosts?policy_id=%d&policy_response=passing", policiesResponse.Policies[0].ID)
	listHostsResp = listHostsResponse{}
	s.DoJSON("GET", listHostsURL, nil, http.StatusOK, &listHostsResp)
	require.Len(t, listHostsResp.Hosts, 0)

	require.NoError(t, s.ds.RecordPolicyQueryExecutions(context.Background(), h1.Host, map[uint]*bool{policiesResponse.Policies[0].ID: ptr.Bool(true)}, time.Now(), false))
	require.NoError(t, s.ds.RecordPolicyQueryExecutions(context.Background(), h2.Host, map[uint]*bool{policiesResponse.Policies[0].ID: nil}, time.Now(), false))

	listHostsURL = fmt.Sprintf("/api/latest/fleet/hosts?policy_id=%d&policy_response=passing", policiesResponse.Policies[0].ID)
	listHostsResp = listHostsResponse{}
	s.DoJSON("GET", listHostsURL, nil, http.StatusOK, &listHostsResp)
	require.Len(t, listHostsResp.Hosts, 1)

	deletePolicyParams := deleteGlobalPoliciesRequest{IDs: []uint{policiesResponse.Policies[0].ID}}
	deletePolicyResp := deleteGlobalPoliciesResponse{}
	s.DoJSON("POST", "/api/latest/fleet/policies/delete", deletePolicyParams, http.StatusOK, &deletePolicyResp)

	policiesResponse = listGlobalPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/policies", nil, http.StatusOK, &policiesResponse)
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
	s.DoJSON("POST", "/api/latest/fleet/hosts/delete", req, http.StatusOK, &resp)

	_, err = s.ds.Host(context.Background(), hosts[0].ID)
	require.Error(t, err)
	_, err = s.ds.Host(context.Background(), hosts[1].ID)
	require.NoError(t, err)
	_, err = s.ds.Host(context.Background(), hosts[2].ID)
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
	s.DoJSON("POST", "/api/latest/fleet/hosts/delete", req, http.StatusOK, &resp)

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
	s.DoJSON("POST", "/api/latest/fleet/hosts/delete", req, http.StatusOK, &resp)

	_, err := s.ds.Host(context.Background(), hosts[0].ID)
	require.Error(t, err)
	_, err = s.ds.Host(context.Background(), hosts[1].ID)
	require.Error(t, err)
	_, err = s.ds.Host(context.Background(), hosts[2].ID)
	require.NoError(t, err)

	err = s.ds.DeleteHosts(context.Background(), []uint{hosts[2].ID})
	require.NoError(t, err)
}

func (s *integrationTestSuite) createHosts(t *testing.T, platforms ...string) []*fleet.Host {
	var hosts []*fleet.Host
	if len(platforms) == 0 {
		platforms = []string{"debian", "rhel", "linux"}
	}
	for i, platform := range platforms {
		host, err := s.ds.NewHost(context.Background(), &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now().Add(-time.Duration(i) * time.Minute),
			OsqueryHostID:   ptr.String(fmt.Sprintf("%s%d", t.Name(), i)),
			NodeKey:         ptr.String(fmt.Sprintf("%s%d", t.Name(), i)),
			UUID:            uuid.New().String(),
			Hostname:        fmt.Sprintf("%sfoo.local%d", t.Name(), i),
			Platform:        platform,
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
	s.DoJSON("POST", "/api/latest/fleet/hosts/delete", req, http.StatusBadRequest, &resp)
}

func (s *integrationTestSuite) TestHostsCount() {
	t := s.T()

	hosts := s.createHosts(t, "darwin", "darwin", "darwin")

	// set disk space information for some hosts
	require.NoError(t, s.ds.SetOrUpdateHostDisksSpace(context.Background(), hosts[0].ID, 10.0, 2.0)) // low disk
	require.NoError(t, s.ds.SetOrUpdateHostDisksSpace(context.Background(), hosts[1].ID, 40.0, 4.0)) // not low disk

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
		"GET", "/api/latest/fleet/hosts/count", req, http.StatusOK, &resp,
		"additional_info_filters", "*",
	)
	assert.Equal(t, 3, resp.Count)

	req = countHostsRequest{}
	resp = countHostsResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/hosts/count", req, http.StatusOK, &resp,
		"additional_info_filters", "*",
		"label_id", fmt.Sprint(label.ID),
	)
	assert.Equal(t, 1, resp.Count)

	// filter by low_disk_space criteria is ignored (premium-only filter)
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &resp, "low_disk_space", "32")
	require.Equal(t, len(hosts), resp.Count)
	// but it is still validated for a correct value when provided (as that happens in a middleware before the handler)
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusInternalServerError, &resp, "low_disk_space", "123456") // TODO: status code to be fixed with #4406

	// filter by MDM criteria without any host having such information
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &resp, "mdm_id", fmt.Sprint(999))
	require.Equal(t, 0, resp.Count)
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &resp, "mdm_enrollment_status", "manual")
	require.Equal(t, 0, resp.Count)

	// set MDM information on a host
	require.NoError(t, s.ds.SetOrUpdateMDMData(context.Background(), hosts[1].ID, false, true, "https://simplemdm.com", false, fleet.WellKnownMDMSimpleMDM))
	// also create server with MDM information, which is ignored.
	require.NoError(t, s.ds.SetOrUpdateMDMData(context.Background(), hosts[2].ID, true, true, "https://simplemdm.com", false, fleet.WellKnownMDMSimpleMDM))
	var mdmID uint
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), q, &mdmID,
			`SELECT id FROM mobile_device_management_solutions WHERE name = ? AND server_url = ?`, fleet.WellKnownMDMSimpleMDM, "https://simplemdm.com")
	})

	// set MDM information for another host installed from DEP and pending enrollment to Fleet MDM
	pendingMDMHost, err := s.ds.NewHost(context.Background(), &fleet.Host{
		Platform:       "darwin",
		HardwareSerial: "532141num832",
		HardwareModel:  "MacBook Pro",
	})
	require.NoError(t, err)
	require.NoError(t, s.ds.SetOrUpdateMDMData(context.Background(), pendingMDMHost.ID, false, false, "https://fleetdm.com", true, fleet.WellKnownMDMFleet))

	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &resp, "mdm_id", fmt.Sprint(mdmID))
	require.Equal(t, 1, resp.Count)
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &resp, "mdm_enrollment_status", "manual")
	require.Equal(t, 1, resp.Count)
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &resp, "mdm_enrollment_status", "automatic")
	require.Equal(t, 0, resp.Count)
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &resp, "mdm_enrollment_status", "unenrolled")
	require.Equal(t, 0, resp.Count)
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &resp, "mdm_enrollment_status", "manual", "mdm_id", fmt.Sprint(mdmID))
	require.Equal(t, 1, resp.Count)
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &resp, "mdm_enrollment_status", "pending")
	require.Equal(t, 1, resp.Count)

	// get the host's MDM info
	var hostResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", pendingMDMHost.ID), nil, http.StatusOK, &hostResp)
	require.Equal(t, pendingMDMHost.ID, hostResp.Host.ID)
	require.Equal(t, "Pending", *hostResp.Host.MDM.EnrollmentStatus)
	require.Equal(t, "https://fleetdm.com", *hostResp.Host.MDM.ServerURL)

	// no macos_settings is returned when MDM is not configured
	require.Nil(t, hostResp.Host.MDM.MacOSSettings)
}

func (s *integrationTestSuite) TestPacks() {
	t := s.T()

	var packResp getPackResponse
	// get non-existing pack
	s.Do("GET", "/api/latest/fleet/packs/999", nil, http.StatusNotFound)

	// create some packs
	packs := make([]fleet.Pack, 3)
	for i := range packs {
		req := &createPackRequest{
			PackPayload: fleet.PackPayload{
				Name: ptr.String(fmt.Sprintf("%s_%d", strings.ReplaceAll(t.Name(), "/", "_"), i)),
			},
		}

		var createResp createPackResponse
		s.DoJSON("POST", "/api/latest/fleet/packs", req, http.StatusOK, &createResp)
		packs[i] = createResp.Pack.Pack
	}

	// get existing pack
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/packs/%d", packs[0].ID), nil, http.StatusOK, &packResp)
	require.Equal(t, packs[0].ID, packResp.Pack.ID)

	// list packs
	var listResp listPacksResponse
	s.DoJSON("GET", "/api/latest/fleet/packs", nil, http.StatusOK, &listResp, "per_page", "2", "order_key", "name")
	require.Len(t, listResp.Packs, 2)
	assert.Equal(t, packs[0].ID, listResp.Packs[0].ID)
	assert.Equal(t, packs[1].ID, listResp.Packs[1].ID)

	// get page 1
	s.DoJSON("GET", "/api/latest/fleet/packs", nil, http.StatusOK, &listResp, "page", "1", "per_page", "2", "order_key", "name")
	require.Len(t, listResp.Packs, 1)
	assert.Equal(t, packs[2].ID, listResp.Packs[0].ID)

	// get page 2, empty
	s.DoJSON("GET", "/api/latest/fleet/packs", nil, http.StatusOK, &listResp, "page", "2", "per_page", "2", "order_key", "name")
	require.Len(t, listResp.Packs, 0)

	var delResp deletePackResponse
	// delete non-existing pack by name
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/packs/%s", "zzz"), nil, http.StatusNotFound, &delResp)

	// delete existing pack by name
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/packs/%s", url.PathEscape(packs[0].Name)), nil, http.StatusOK, &delResp)

	// delete non-existing pack by id
	var delIDResp deletePackByIDResponse
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/packs/id/%d", packs[2].ID+1), nil, http.StatusNotFound, &delIDResp)

	// delete existing pack by id
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/packs/id/%d", packs[1].ID), nil, http.StatusOK, &delIDResp)

	var modResp modifyPackResponse
	// modify non-existing pack
	req := &fleet.PackPayload{Name: ptr.String("updated_" + packs[2].Name)}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/packs/%d", packs[2].ID+1), req, http.StatusNotFound, &modResp)

	// modify existing pack
	req = &fleet.PackPayload{Name: ptr.String("updated_" + packs[2].Name)}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/packs/%d", packs[2].ID), req, http.StatusOK, &modResp)
	require.Equal(t, packs[2].ID, modResp.Pack.ID)
	require.Contains(t, modResp.Pack.Name, "updated_")

	// list packs, only packs[2] remains
	s.DoJSON("GET", "/api/latest/fleet/packs", nil, http.StatusOK, &listResp, "per_page", "2", "order_key", "name")
	require.Len(t, listResp.Packs, 1)
	assert.Equal(t, packs[2].ID, listResp.Packs[0].ID)
}

func (s *integrationTestSuite) TestListHosts() {
	t := s.T()

	hosts := s.createHosts(t, "darwin", "darwin", "darwin")

	// set disk space information for some hosts
	require.NoError(t, s.ds.SetOrUpdateHostDisksSpace(context.Background(), hosts[0].ID, 10.0, 2.0)) // low disk
	require.NoError(t, s.ds.SetOrUpdateHostDisksSpace(context.Background(), hosts[1].ID, 40.0, 4.0)) // not low disk

	var resp listHostsResponse
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp)
	require.Len(t, resp.Hosts, len(hosts))
	for _, h := range resp.Hosts {
		switch h.ID {
		case hosts[0].ID:
			assert.Equal(t, 10.0, h.GigsDiskSpaceAvailable)
			assert.Equal(t, 2.0, h.PercentDiskSpaceAvailable)
		case hosts[1].ID:
			assert.Equal(t, 40.0, h.GigsDiskSpaceAvailable)
			assert.Equal(t, 4.0, h.PercentDiskSpaceAvailable)
		}
		assert.Equal(t, h.SoftwareUpdatedAt, h.CreatedAt)
	}

	// setting the low_disk_space criteria is ignored (premium-only)
	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "low_disk_space", "32")
	require.Len(t, resp.Hosts, len(hosts))

	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "per_page", "1")
	require.Len(t, resp.Hosts, 1)
	assert.Nil(t, resp.Software)
	assert.Nil(t, resp.MDMSolution)
	assert.Nil(t, resp.MunkiIssue)

	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "order_key", "h.id", "after", fmt.Sprint(hosts[1].ID))
	require.Len(t, resp.Hosts, len(hosts)-2)

	time.Sleep(1 * time.Second)
	host := hosts[2]
	software := []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
	}
	_, err := s.ds.UpdateHostSoftware(context.Background(), host.ID, software)
	require.NoError(t, err)
	require.NoError(t, s.ds.LoadHostSoftware(context.Background(), host, false))

	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "software_id", fmt.Sprint(host.Software[0].ID))
	require.Len(t, resp.Hosts, 1)
	assert.Equal(t, host.ID, resp.Hosts[0].ID)
	assert.Equal(t, "foo", resp.Software.Name)
	assert.Greater(t, resp.Hosts[0].SoftwareUpdatedAt, resp.Hosts[0].CreatedAt)

	user1 := test.NewUser(t, s.ds, "Alice", "alice@example.com", true)
	q := test.NewQuery(t, s.ds, nil, "query1", "select 1", 0, true)
	defer cleanupQuery(s, q.ID)
	p, err := s.ds.NewGlobalPolicy(context.Background(), &user1.ID, fleet.PolicyPayload{
		QueryID: &q.ID,
	})
	require.NoError(t, err)

	require.NoError(t, s.ds.RecordPolicyQueryExecutions(context.Background(), host, map[uint]*bool{p.ID: ptr.Bool(false)}, time.Now(), false))

	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "software_id", fmt.Sprint(host.Software[0].ID))
	require.Len(t, resp.Hosts, 1)
	assert.Equal(t, 1, resp.Hosts[0].HostIssues.FailingPoliciesCount)
	assert.Equal(t, 1, resp.Hosts[0].HostIssues.TotalIssuesCount)

	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "software_id", fmt.Sprint(host.Software[0].ID), "disable_failing_policies", "true")
	require.Len(t, resp.Hosts, 1)
	assert.Equal(t, 0, resp.Hosts[0].HostIssues.FailingPoliciesCount)
	assert.Equal(t, 0, resp.Hosts[0].HostIssues.TotalIssuesCount)

	// filter by MDM criteria without any host having such information
	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "mdm_id", fmt.Sprint(999))
	require.Len(t, resp.Hosts, 0)
	assert.Nil(t, resp.Software)
	assert.Nil(t, resp.MDMSolution)
	assert.Nil(t, resp.MunkiIssue)
	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "mdm_enrollment_status", "manual")
	require.Len(t, resp.Hosts, 0)
	assert.Nil(t, resp.Software)
	assert.Nil(t, resp.MDMSolution)
	assert.Nil(t, resp.MunkiIssue)
	// and same by munki issue id
	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "munki_issue_id", fmt.Sprint(999))
	require.Len(t, resp.Hosts, 0)
	assert.Nil(t, resp.Software)
	assert.Nil(t, resp.MDMSolution)
	assert.Nil(t, resp.MunkiIssue)

	// set MDM information on a host
	require.NoError(t, s.ds.SetOrUpdateMDMData(context.Background(), host.ID, false, true, "https://simplemdm.com", false, fleet.WellKnownMDMSimpleMDM))
	var mdmID uint
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), q, &mdmID,
			`SELECT id FROM mobile_device_management_solutions WHERE name = ? AND server_url = ?`, fleet.WellKnownMDMSimpleMDM, "https://simplemdm.com")
	})

	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp)

	// set MDM information for another host installed from DEP and pending enrollment to Fleet MDM
	pendingMDMHost, err := s.ds.NewHost(context.Background(), &fleet.Host{
		Platform:       "darwin",
		HardwareSerial: "532141num832",
		HardwareModel:  "MacBook Pro",
	})
	require.NoError(t, err)
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(context.Background(), "INSERT INTO mobile_device_management_solutions (name, server_url) VALUES ('https://fleetdm.com', 'Fleet')")
		require.NoError(t, err)
		return err
	})
	require.NoError(t, s.ds.SetOrUpdateMDMData(context.Background(), pendingMDMHost.ID, false, false, "https://fleetdm.com", true, fleet.WellKnownMDMFleet))

	// generate aggregated stats
	require.NoError(t, s.ds.GenerateAggregatedMunkiAndMDM(context.Background()))

	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "mdm_enrollment_status", "pending")
	require.Len(t, resp.Hosts, 1)
	require.Equal(t, "532141num832", resp.Hosts[0].HardwareSerial)
	assert.Nil(t, resp.Software)
	assert.Nil(t, resp.MunkiIssue)
	require.Nil(t, resp.MDMSolution) // MDM solution is included only if `mdm_id` query param is specified`

	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "mdm_enrollment_status", "manual")
	require.Len(t, resp.Hosts, 1)
	assert.Nil(t, resp.Software)
	assert.Nil(t, resp.MDMSolution)
	assert.Nil(t, resp.MunkiIssue)

	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "mdm_enrollment_status", "automatic")
	require.Len(t, resp.Hosts, 0)
	assert.Nil(t, resp.Software)
	assert.Nil(t, resp.MDMSolution)
	assert.Nil(t, resp.MunkiIssue)

	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "mdm_enrollment_status", "unenrolled")
	require.Len(t, resp.Hosts, 0)
	assert.Nil(t, resp.Software)
	assert.Nil(t, resp.MDMSolution)
	assert.Nil(t, resp.MunkiIssue)

	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "mdm_id", fmt.Sprint(mdmID))
	require.Len(t, resp.Hosts, 1)
	assert.Nil(t, resp.Software)
	assert.Nil(t, resp.MunkiIssue)
	require.NotNil(t, resp.MDMSolution)
	assert.Equal(t, mdmID, resp.MDMSolution.ID)
	assert.Equal(t, fleet.WellKnownMDMSimpleMDM, resp.MDMSolution.Name)
	assert.Equal(t, "https://simplemdm.com", resp.MDMSolution.ServerURL)

	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "mdm_id", fmt.Sprint(mdmID), "mdm_enrollment_status", "manual")
	require.Len(t, resp.Hosts, 1)
	assert.Nil(t, resp.Software)
	assert.Nil(t, resp.MunkiIssue)
	assert.NotNil(t, resp.MDMSolution)
	assert.Equal(t, mdmID, resp.MDMSolution.ID)

	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusBadRequest, &resp, "mdm_enrollment_status", "invalid-status")

	// Filter by inexistent software.
	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusNotFound, &resp, "software_id", fmt.Sprint(9999))

	// set munki information on a host
	require.NoError(t, s.ds.SetOrUpdateMunkiInfo(context.Background(), host.ID, "1.2.3", []string{"err"}, []string{"warn"}))
	var errMunkiID uint
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), q, &errMunkiID,
			`SELECT id FROM munki_issues WHERE name = 'err' AND issue_type = 'error'`)
	})
	// generate aggregated stats
	require.NoError(t, s.ds.GenerateAggregatedMunkiAndMDM(context.Background()))

	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "munki_issue_id", fmt.Sprint(errMunkiID))
	require.Len(t, resp.Hosts, 1)
	assert.Nil(t, resp.Software)
	assert.Nil(t, resp.MDMSolution)
	require.NotNil(t, resp.MunkiIssue)
	assert.Equal(t, fleet.MunkiIssue{
		ID:        errMunkiID,
		Name:      "err",
		IssueType: "error",
	}, *resp.MunkiIssue)

	// filters can be combined, no problem
	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "munki_issue_id", fmt.Sprint(errMunkiID), "mdm_id", fmt.Sprint(mdmID))
	require.Len(t, resp.Hosts, 1)
	assert.Nil(t, resp.Software)
	assert.NotNil(t, resp.MDMSolution)
	assert.NotNil(t, resp.MunkiIssue)

	// set operating system information on a host
	testOS := fleet.OperatingSystem{Name: "fooOS", Version: "4.2", Arch: "64bit", KernelVersion: "13.37", Platform: "bar"}
	require.NoError(t, s.ds.UpdateHostOperatingSystem(context.Background(), host.ID, testOS))
	var osID uint
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), q, &osID,
			`SELECT id FROM operating_systems WHERE name = ? AND version = ?`, "fooOS", "4.2")
	})
	require.Greater(t, osID, uint(0))

	// generate aggregated stats
	require.NoError(t, s.ds.UpdateOSVersions(context.Background()))

	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "os_name", testOS.Name, "os_version", testOS.Version)
	require.Len(t, resp.Hosts, 1)

	expected := resp.Hosts[0]
	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "os_id", fmt.Sprintf("%d", osID))
	require.Len(t, resp.Hosts, 1)
	require.Equal(t, expected, resp.Hosts[0])

	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "os_name", "unknownOS", "os_version", "4.2")
	require.Len(t, resp.Hosts, 0)

	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "os_id", fmt.Sprintf("%d", osID+1337))
	require.Len(t, resp.Hosts, 0)
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
	s.DoJSON("GET", "/api/latest/fleet/invites", nil, http.StatusOK, &listResp)
	require.Len(t, listResp.Invites, 0)

	// create valid invite
	createInviteReq := createInviteRequest{InvitePayload: fleet.InvitePayload{
		Email:      ptr.String("some email"),
		Name:       ptr.String("some name"),
		GlobalRole: null.StringFrom(fleet.RoleAdmin),
	}}
	createInviteResp := createInviteResponse{}
	s.DoJSON("POST", "/api/latest/fleet/invites", createInviteReq, http.StatusOK, &createInviteResp)
	require.NotNil(t, createInviteResp.Invite)
	require.NotZero(t, createInviteResp.Invite.ID)
	validInvite := *createInviteResp.Invite

	// create user from valid invite - the token was not returned via the
	// response's json, must get it from the db
	inv, err := s.ds.Invite(context.Background(), validInvite.ID)
	require.NoError(t, err)
	validInviteToken := inv.Token

	// verify the token with valid invite
	var verifyInvResp verifyInviteResponse
	s.DoJSON("GET", "/api/latest/fleet/invites/"+validInviteToken, nil, http.StatusOK, &verifyInvResp)
	require.Equal(t, validInvite.ID, verifyInvResp.Invite.ID)

	// verify the token with an invalid invite
	s.DoJSON("GET", "/api/latest/fleet/invites/invalid", nil, http.StatusNotFound, &verifyInvResp)

	// create invite without an email
	createInviteReq = createInviteRequest{InvitePayload: fleet.InvitePayload{
		Email:      nil,
		Name:       ptr.String("some other name"),
		GlobalRole: null.StringFrom(fleet.RoleObserver),
	}}
	createInviteResp = createInviteResponse{}
	s.DoJSON("POST", "/api/latest/fleet/invites", createInviteReq, http.StatusUnprocessableEntity, &createInviteResp)

	// create invite for an existing user
	existingEmail := "admin1@example.com"
	createInviteReq = createInviteRequest{InvitePayload: fleet.InvitePayload{
		Email:      ptr.String(existingEmail),
		Name:       ptr.String("some other name"),
		GlobalRole: null.StringFrom(fleet.RoleObserver),
	}}
	createInviteResp = createInviteResponse{}
	s.DoJSON("POST", "/api/latest/fleet/invites", createInviteReq, http.StatusUnprocessableEntity, &createInviteResp)

	// create invite for an existing user with email ALL CAPS
	createInviteReq = createInviteRequest{InvitePayload: fleet.InvitePayload{
		Email:      ptr.String(strings.ToUpper(existingEmail)),
		Name:       ptr.String("some other name"),
		GlobalRole: null.StringFrom(fleet.RoleObserver),
	}}
	createInviteResp = createInviteResponse{}
	s.DoJSON("POST", "/api/latest/fleet/invites", createInviteReq, http.StatusUnprocessableEntity, &createInviteResp)

	// list invites, we have one now
	listResp = listInvitesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/invites", nil, http.StatusOK, &listResp)
	require.Len(t, listResp.Invites, 1)
	require.Equal(t, validInvite.ID, listResp.Invites[0].ID)

	// list invites, next page is empty
	listResp = listInvitesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/invites", nil, http.StatusOK, &listResp, "page", "1", "per_page", "2")
	require.Len(t, listResp.Invites, 0)

	// update a non-existing invite
	updateInviteReq := updateInviteRequest{InvitePayload: fleet.InvitePayload{
		Teams: []fleet.UserTeam{
			{Team: fleet.Team{ID: team.ID}, Role: fleet.RoleObserver},
		},
	}}
	updateInviteResp := updateInviteResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/invites/%d", validInvite.ID+1), updateInviteReq, http.StatusNotFound, &updateInviteResp)

	// update the valid invite created earlier, make it an observer of a team
	updateInviteResp = updateInviteResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/invites/%d", validInvite.ID), updateInviteReq, http.StatusOK, &updateInviteResp)

	// update the valid invite: set an email that already exists for a user
	updateInviteReq = updateInviteRequest{
		InvitePayload: fleet.InvitePayload{
			Email: ptr.String(s.users["admin1@example.com"].Email),
			Teams: []fleet.UserTeam{
				{Team: fleet.Team{ID: team.ID}, Role: fleet.RoleObserver},
			},
		},
	}
	updateInviteResp = updateInviteResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/invites/%d", validInvite.ID), updateInviteReq, http.StatusConflict, &updateInviteResp)

	// update the valid invite: set an email that already exists for another invite
	createInviteReq = createInviteRequest{InvitePayload: fleet.InvitePayload{
		Email:      ptr.String("some@other.email"),
		Name:       ptr.String("some name"),
		GlobalRole: null.StringFrom(fleet.RoleAdmin),
	}}
	createInviteResp = createInviteResponse{}
	s.DoJSON("POST", "/api/latest/fleet/invites", createInviteReq, http.StatusOK, &createInviteResp)
	updateInviteReq = updateInviteRequest{
		InvitePayload: fleet.InvitePayload{
			Email: createInviteReq.Email,
			Teams: []fleet.UserTeam{
				{Team: fleet.Team{ID: team.ID}, Role: fleet.RoleObserver},
			},
		},
	}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/invites/%d", validInvite.ID), updateInviteReq, http.StatusConflict, &updateInviteResp)

	// update the valid invite to an email that is ok
	updateInviteReq = updateInviteRequest{
		InvitePayload: fleet.InvitePayload{
			Email: ptr.String("something@nonexistent.yet123"),
			Teams: []fleet.UserTeam{
				{Team: fleet.Team{ID: team.ID}, Role: fleet.RoleObserver},
			},
		},
	}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/invites/%d", validInvite.ID), updateInviteReq, http.StatusOK, &updateInviteResp)

	verify, err := s.ds.Invite(context.Background(), validInvite.ID)
	require.NoError(t, err)
	require.Equal(t, "", verify.GlobalRole.String)
	require.Len(t, verify.Teams, 1)
	assert.Equal(t, team.ID, verify.Teams[0].ID)

	var createFromInviteResp createUserResponse
	s.DoJSON("POST", "/api/latest/fleet/users", fleet.UserPayload{
		Name:        ptr.String("Full Name"),
		Password:    ptr.String(test.GoodPassword),
		Email:       ptr.String("a@b.c"),
		InviteToken: ptr.String(validInviteToken),
	}, http.StatusOK, &createFromInviteResp)

	// keep the invite token from the other valid invite (before deleting it)
	inv, err = s.ds.Invite(context.Background(), createInviteResp.Invite.ID)
	require.NoError(t, err)
	deletedInviteToken := inv.Token

	// delete an existing invite
	var delResp deleteInviteResponse
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/invites/%d", createInviteResp.Invite.ID), nil, http.StatusOK, &delResp)

	// list invites, is now empty
	listResp = listInvitesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/invites", nil, http.StatusOK, &listResp)
	require.Len(t, listResp.Invites, 0)

	// delete a now non-existing invite
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/invites/%d", validInvite.ID), nil, http.StatusNotFound, &delResp)

	// create user from never used but deleted invite
	s.DoJSON("POST", "/api/latest/fleet/users", fleet.UserPayload{
		Name:        ptr.String("Full Name"),
		Password:    ptr.String(test.GoodPassword),
		Email:       ptr.String("a@b.c"),
		InviteToken: ptr.String(deletedInviteToken),
	}, http.StatusNotFound, &createFromInviteResp)
}

func (s *integrationTestSuite) TestCreateUserFromInviteErrors() {
	t := s.T()

	// create a valid invite
	createInviteReq := createInviteRequest{InvitePayload: fleet.InvitePayload{
		Email:      ptr.String("a@b.c"),
		Name:       ptr.String("A"),
		GlobalRole: null.StringFrom(fleet.RoleObserver),
	}}
	createInviteResp := createInviteResponse{}
	s.DoJSON("POST", "/api/latest/fleet/invites", createInviteReq, http.StatusOK, &createInviteResp)

	// make sure to delete it on exit
	defer func() {
		var delResp deleteInviteResponse
		s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/invites/%d", createInviteResp.Invite.ID), nil, http.StatusOK, &delResp)
	}()

	// the token is not returned via the response's json, must get it from the db
	invite, err := s.ds.Invite(context.Background(), createInviteResp.Invite.ID)
	require.NoError(t, err)

	cases := []struct {
		desc string
		pld  fleet.UserPayload
		want int
	}{
		{
			"empty name",
			fleet.UserPayload{
				Name:        ptr.String(""),
				Password:    &test.GoodPassword,
				Email:       ptr.String("a@b.c"),
				InviteToken: ptr.String(invite.Token),
			},
			http.StatusUnprocessableEntity,
		},
		{
			"empty email",
			fleet.UserPayload{
				Name:        ptr.String("Name"),
				Password:    &test.GoodPassword,
				Email:       ptr.String(""),
				InviteToken: ptr.String(invite.Token),
			},
			http.StatusUnprocessableEntity,
		},
		{
			"empty password",
			fleet.UserPayload{
				Name:        ptr.String("Name"),
				Password:    ptr.String(""),
				Email:       ptr.String("a@b.c"),
				InviteToken: ptr.String(invite.Token),
			},
			http.StatusUnprocessableEntity,
		},
		{
			"empty token",
			fleet.UserPayload{
				Name:        ptr.String("Name"),
				Password:    &test.GoodPassword,
				Email:       ptr.String("a@b.c"),
				InviteToken: ptr.String(""),
			},
			http.StatusUnprocessableEntity,
		},
		{
			"invalid token",
			fleet.UserPayload{
				Name:        ptr.String("Name"),
				Password:    &test.GoodPassword,
				Email:       ptr.String("a@b.c"),
				InviteToken: ptr.String("invalid"),
			},
			http.StatusNotFound,
		},
		{
			"invalid password",
			fleet.UserPayload{
				Name:        ptr.String("Name"),
				Password:    ptr.String("password"), // no number or symbol
				Email:       ptr.String("a@b.c"),
				InviteToken: ptr.String(invite.Token),
			},
			http.StatusUnprocessableEntity,
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			var resp createUserResponse
			s.DoJSON("POST", "/api/latest/fleet/users", c.pld, c.want, &resp)
		})
	}
}

func (s *integrationTestSuite) TestGetHostSummary() {
	t := s.T()
	ctx := context.Background()

	hosts := s.createHosts(t)

	team1, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "team1"})
	require.NoError(t, err)
	team2, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "team2"})
	require.NoError(t, err)

	require.NoError(t, s.ds.AddHostsToTeam(ctx, &team1.ID, []uint{hosts[0].ID}))

	// set disk space information for hosts [0] and [1]
	require.NoError(t, s.ds.SetOrUpdateHostDisksSpace(ctx, hosts[0].ID, 1.0, 2.0))
	require.NoError(t, s.ds.SetOrUpdateHostDisksSpace(ctx, hosts[1].ID, 3.0, 4.0))

	var getHostResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", hosts[0].ID), nil, http.StatusOK, &getHostResp)
	assert.Equal(t, 1.0, getHostResp.Host.GigsDiskSpaceAvailable)
	assert.Equal(t, 2.0, getHostResp.Host.PercentDiskSpaceAvailable)

	var resp getHostSummaryResponse

	// no team filter
	s.DoJSON("GET", "/api/latest/fleet/host_summary", nil, http.StatusOK, &resp)
	require.Equal(t, resp.TotalsHostsCount, uint(len(hosts)))
	require.Nil(t, resp.LowDiskSpaceCount)
	require.Len(t, resp.Platforms, 3)
	gotPlatforms, wantPlatforms := make([]string, 3), []string{"linux", "debian", "rhel"}
	for i, p := range resp.Platforms {
		gotPlatforms[i] = p.Platform
		// each platform has a count of 1
		require.Equal(t, uint(1), p.HostsCount)
	}
	require.ElementsMatch(t, wantPlatforms, gotPlatforms)
	require.Nil(t, resp.TeamID)
	require.Equal(t, uint(3), resp.AllLinuxCount)
	assert.True(t, len(resp.BuiltinLabels) > 0)
	for _, lbl := range resp.BuiltinLabels {
		assert.Equal(t, fleet.LabelTypeBuiltIn, lbl.LabelType)
	}
	builtinsCount := len(resp.BuiltinLabels)

	// host summary builtin labels match list labels response
	var listResp listLabelsResponse
	s.DoJSON("GET", "/api/latest/fleet/labels", nil, http.StatusOK, &listResp)
	assert.True(t, len(listResp.Labels) > 0)
	for _, lbl := range listResp.Labels {
		assert.Equal(t, fleet.LabelTypeBuiltIn, lbl.LabelType)
	}
	assert.Equal(t, len(listResp.Labels), builtinsCount)

	// team filter, no host
	s.DoJSON("GET", "/api/latest/fleet/host_summary", nil, http.StatusOK, &resp, "team_id", fmt.Sprint(team2.ID))
	require.Equal(t, resp.TotalsHostsCount, uint(0))
	require.Len(t, resp.Platforms, 0)
	require.Equal(t, uint(0), resp.AllLinuxCount)
	require.Equal(t, team2.ID, *resp.TeamID)

	// team filter, one host, low_disk_count is ignored as not premium
	s.DoJSON("GET", "/api/latest/fleet/host_summary", nil, http.StatusOK, &resp, "team_id", fmt.Sprint(team1.ID), "low_disk_space", "2")
	require.Equal(t, resp.TotalsHostsCount, uint(1))
	require.Nil(t, resp.LowDiskSpaceCount)
	require.Len(t, resp.Platforms, 1)
	require.Equal(t, "debian", resp.Platforms[0].Platform)
	require.Equal(t, uint(1), resp.Platforms[0].HostsCount)
	require.Equal(t, uint(1), resp.AllLinuxCount)
	require.Equal(t, team1.ID, *resp.TeamID)

	s.DoJSON("GET", "/api/latest/fleet/host_summary", nil, http.StatusOK, &resp, "team_id", fmt.Sprint(team1.ID), "platform", "linux")
	require.Equal(t, resp.TotalsHostsCount, uint(1))
	require.Equal(t, "debian", resp.Platforms[0].Platform)
	require.Equal(t, uint(1), resp.AllLinuxCount)

	s.DoJSON("GET", "/api/latest/fleet/host_summary", nil, http.StatusOK, &resp, "platform", "rhel")
	require.Equal(t, resp.TotalsHostsCount, uint(1))
	require.Equal(t, "rhel", resp.Platforms[0].Platform)
	require.Equal(t, uint(1), resp.AllLinuxCount)

	s.DoJSON("GET", "/api/latest/fleet/host_summary", nil, http.StatusOK, &resp, "platform", "linux")
	require.Equal(t, resp.TotalsHostsCount, uint(3))
	require.Equal(t, uint(3), resp.AllLinuxCount)
	require.Len(t, resp.Platforms, 3)
	for i, p := range resp.Platforms {
		gotPlatforms[i] = p.Platform
		// each platform has a count of 1
		require.Equal(t, uint(1), p.HostsCount)
	}
	require.ElementsMatch(t, wantPlatforms, gotPlatforms)

	s.DoJSON("GET", "/api/latest/fleet/host_summary", nil, http.StatusOK, &resp, "platform", "darwin")
	require.Equal(t, resp.TotalsHostsCount, uint(0))
	require.Equal(t, resp.AllLinuxCount, uint(0))
	require.Len(t, resp.Platforms, 0)

	// invalid low_disk_space value is still validated and results in error
	s.DoJSON("GET", "/api/latest/fleet/host_summary", nil, http.StatusInternalServerError, &resp, "low_disk_space", "1234") // TODO: should be 400, see #4406
}

func (s *integrationTestSuite) TestGlobalPoliciesProprietary() {
	t := s.T()

	for i := 0; i < 3; i++ {
		_, err := s.ds.NewHost(context.Background(), &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now().Add(-time.Duration(i) * time.Minute),
			OsqueryHostID:   ptr.String(fmt.Sprintf("%s%d", t.Name(), i)),
			NodeKey:         ptr.String(fmt.Sprintf("%s%d", t.Name(), i)),
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
	s.DoJSON("POST", "/api/latest/fleet/policies", gpParams0, http.StatusBadRequest, &gpResp0)
	require.Nil(t, gpResp0.Policy)

	gpParams := globalPolicyRequest{
		Name:        "TestQuery3",
		Query:       "select * from osquery;",
		Description: "Some description",
		Resolution:  "some global resolution",
		Platform:    "darwin",
	}
	gpResp := globalPolicyResponse{}
	s.DoJSON("POST", "/api/latest/fleet/policies", gpParams, http.StatusOK, &gpResp)
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
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/policies/%d", gpResp.Policy.ID), mgpParams, http.StatusOK, &mgpResp)
	require.NotNil(t, gpResp.Policy)
	assert.Equal(t, "TestQuery4", mgpResp.Policy.Name)
	assert.Equal(t, "select * from osquery_info;", mgpResp.Policy.Query)
	assert.Equal(t, "Some description updated", mgpResp.Policy.Description)
	require.NotNil(t, mgpResp.Policy.Resolution)
	assert.Equal(t, "some global resolution updated", *mgpResp.Policy.Resolution)
	assert.Equal(t, "darwin", mgpResp.Policy.Platform)

	ggpResp := getPolicyByIDResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/policies/%d", gpResp.Policy.ID), getPolicyByIDRequest{}, http.StatusOK, &ggpResp)
	require.NotNil(t, ggpResp.Policy)
	assert.Equal(t, "TestQuery4", ggpResp.Policy.Name)
	assert.Equal(t, "select * from osquery_info;", ggpResp.Policy.Query)
	assert.Equal(t, "Some description updated", ggpResp.Policy.Description)
	require.NotNil(t, ggpResp.Policy.Resolution)
	assert.Equal(t, "some global resolution updated", *ggpResp.Policy.Resolution)
	assert.Equal(t, "darwin", mgpResp.Policy.Platform)

	policiesResponse := listGlobalPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/policies", nil, http.StatusOK, &policiesResponse)
	require.Len(t, policiesResponse.Policies, 1)
	assert.Equal(t, "TestQuery4", policiesResponse.Policies[0].Name)
	assert.Equal(t, "select * from osquery_info;", policiesResponse.Policies[0].Query)
	assert.Equal(t, "Some description updated", policiesResponse.Policies[0].Description)
	require.NotNil(t, policiesResponse.Policies[0].Resolution)
	assert.Equal(t, "some global resolution updated", *policiesResponse.Policies[0].Resolution)
	assert.Equal(t, "darwin", policiesResponse.Policies[0].Platform)

	listHostsURL := fmt.Sprintf("/api/latest/fleet/hosts?policy_id=%d", policiesResponse.Policies[0].ID)
	listHostsResp := listHostsResponse{}
	s.DoJSON("GET", listHostsURL, nil, http.StatusOK, &listHostsResp)
	require.Len(t, listHostsResp.Hosts, 3)
	h1 := listHostsResp.Hosts[0]
	h2 := listHostsResp.Hosts[1]

	listHostsURL = fmt.Sprintf("/api/latest/fleet/hosts?policy_id=%d&policy_response=passing", policiesResponse.Policies[0].ID)
	listHostsResp = listHostsResponse{}
	s.DoJSON("GET", listHostsURL, nil, http.StatusOK, &listHostsResp)
	require.Len(t, listHostsResp.Hosts, 0)

	require.NoError(t, s.ds.RecordPolicyQueryExecutions(context.Background(), h1.Host, map[uint]*bool{policiesResponse.Policies[0].ID: ptr.Bool(true)}, time.Now(), false))
	require.NoError(t, s.ds.RecordPolicyQueryExecutions(context.Background(), h2.Host, map[uint]*bool{policiesResponse.Policies[0].ID: nil}, time.Now(), false))

	listHostsURL = fmt.Sprintf("/api/latest/fleet/hosts?policy_id=%d&policy_response=passing", policiesResponse.Policies[0].ID)
	listHostsResp = listHostsResponse{}
	s.DoJSON("GET", listHostsURL, nil, http.StatusOK, &listHostsResp)
	require.Len(t, listHostsResp.Hosts, 1)

	deletePolicyParams := deleteGlobalPoliciesRequest{IDs: []uint{policiesResponse.Policies[0].ID}}
	deletePolicyResp := deleteGlobalPoliciesResponse{}
	s.DoJSON("POST", "/api/latest/fleet/policies/delete", deletePolicyParams, http.StatusOK, &deletePolicyResp)

	policiesResponse = listGlobalPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/policies", nil, http.StatusOK, &policiesResponse)
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
			OsqueryHostID:   ptr.String(fmt.Sprintf("%s%d", t.Name(), i)),
			NodeKey:         ptr.String(fmt.Sprintf("%s%d", t.Name(), i)),
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
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team1.ID), tpParams, http.StatusOK, &tpResp)
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
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team1.ID, tpResp.Policy.ID), mtpParams, http.StatusOK, &mtpResp)
	require.NotNil(t, mtpResp.Policy)
	assert.Equal(t, "TestQuery4", mtpResp.Policy.Name)
	assert.Equal(t, "select * from osquery_info;", mtpResp.Policy.Query)
	assert.Equal(t, "Some description updated", mtpResp.Policy.Description)
	require.NotNil(t, mtpResp.Policy.Resolution)
	assert.Equal(t, "some team resolution updated", *mtpResp.Policy.Resolution)
	assert.Equal(t, "darwin", mtpResp.Policy.Platform)

	gtpResp := getPolicyByIDResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team1.ID, tpResp.Policy.ID), getPolicyByIDRequest{}, http.StatusOK, &gtpResp)
	require.NotNil(t, gtpResp.Policy)
	assert.Equal(t, "TestQuery4", gtpResp.Policy.Name)
	assert.Equal(t, "select * from osquery_info;", gtpResp.Policy.Query)
	assert.Equal(t, "Some description updated", gtpResp.Policy.Description)
	require.NotNil(t, gtpResp.Policy.Resolution)
	assert.Equal(t, "some team resolution updated", *gtpResp.Policy.Resolution)
	assert.Equal(t, "darwin", gtpResp.Policy.Platform)

	policiesResponse := listTeamPoliciesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team1.ID), nil, http.StatusOK, &policiesResponse)
	require.Len(t, policiesResponse.Policies, 1)
	assert.Equal(t, "TestQuery4", policiesResponse.Policies[0].Name)
	assert.Equal(t, "select * from osquery_info;", policiesResponse.Policies[0].Query)
	assert.Equal(t, "Some description updated", policiesResponse.Policies[0].Description)
	require.NotNil(t, policiesResponse.Policies[0].Resolution)
	assert.Equal(t, "some team resolution updated", *policiesResponse.Policies[0].Resolution)
	assert.Equal(t, "darwin", policiesResponse.Policies[0].Platform)
	require.Len(t, policiesResponse.InheritedPolicies, 0)

	listHostsURL := fmt.Sprintf("/api/latest/fleet/hosts?policy_id=%d", policiesResponse.Policies[0].ID)
	listHostsResp := listHostsResponse{}
	s.DoJSON("GET", listHostsURL, nil, http.StatusOK, &listHostsResp)
	require.Len(t, listHostsResp.Hosts, 2)
	h1 := listHostsResp.Hosts[0]
	h2 := listHostsResp.Hosts[1]

	listHostsURL = fmt.Sprintf("/api/latest/fleet/hosts?team_id=%d&policy_id=%d&policy_response=passing", team1.ID, policiesResponse.Policies[0].ID)
	listHostsResp = listHostsResponse{}
	s.DoJSON("GET", listHostsURL, nil, http.StatusOK, &listHostsResp)
	require.Len(t, listHostsResp.Hosts, 0)

	require.NoError(t, s.ds.RecordPolicyQueryExecutions(context.Background(), h1.Host, map[uint]*bool{policiesResponse.Policies[0].ID: ptr.Bool(true)}, time.Now(), false))
	require.NoError(t, s.ds.RecordPolicyQueryExecutions(context.Background(), h2.Host, map[uint]*bool{policiesResponse.Policies[0].ID: nil}, time.Now(), false))

	listHostsURL = fmt.Sprintf("/api/latest/fleet/hosts?team_id=%d&policy_id=%d&policy_response=passing", team1.ID, policiesResponse.Policies[0].ID)
	listHostsResp = listHostsResponse{}
	s.DoJSON("GET", listHostsURL, nil, http.StatusOK, &listHostsResp)
	require.Len(t, listHostsResp.Hosts, 1)

	deletePolicyParams := deleteTeamPoliciesRequest{IDs: []uint{policiesResponse.Policies[0].ID}}
	deletePolicyResp := deleteTeamPoliciesResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/delete", team1.ID), deletePolicyParams, http.StatusOK, &deletePolicyResp)

	policiesResponse = listTeamPoliciesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team1.ID), nil, http.StatusOK, &policiesResponse)
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
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team1.ID), tpParams, http.StatusOK, &tpResp)
	require.NotNil(t, tpResp.Policy)
	teamPolicyID := tpResp.Policy.ID

	gpParams := globalPolicyRequest{
		Name:        "TestQuery3-Global",
		Query:       "select * from osquery;",
		Description: "Some description",
		Resolution:  "some global resolution",
	}
	gpResp := globalPolicyResponse{}
	s.DoJSON("POST", "/api/latest/fleet/policies", gpParams, http.StatusOK, &gpResp)
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
			query:      "",
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
			s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team1.ID), tpReq, http.StatusBadRequest, &tpResp)
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
				s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team1.ID, teamPolicyID), tpReq, http.StatusBadRequest, &tpResp)
				require.Nil(t, tpResp.Policy)
			}

			gpReq := globalPolicyRequest{
				QueryID:  tc.queryID,
				Name:     tc.name,
				Query:    tc.query,
				Platform: tc.platforms,
			}
			gpResp := globalPolicyResponse{}
			s.DoJSON("POST", "/api/latest/fleet/policies", gpReq, http.StatusBadRequest, &gpResp)
			require.Nil(t, tpResp.Policy)

			if testUpdate {
				gpReq := modifyGlobalPolicyRequest{
					ModifyPolicyPayload: fleet.ModifyPolicyPayload{
						Name:  ptr.String(tc.name),
						Query: ptr.String(tc.query),
					},
				}
				gpResp := modifyGlobalPolicyResponse{}
				s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/policies/%d", globalPolicyID), gpReq, http.StatusBadRequest, &gpResp)
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
	s.DoJSON("POST", "/api/latest/fleet/policies", gpParams, http.StatusOK, &gpResp)
	require.NotNil(t, gpResp.Policy)
	require.NotEmpty(t, gpResp.Policy.ID)

	tpParams := teamPolicyRequest{
		Name:        "HostDetailsPolicies-Team",
		Query:       "select * from osquery;",
		Description: "Some description",
		Resolution:  "some team resolution",
	}
	tpResp := teamPolicyResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team1.ID), tpParams, http.StatusOK, &tpResp)
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

	resp := s.Do("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host1.ID), nil, http.StatusOK)
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
	policies := *hd.Policies
	require.Len(t, policies, 2)
	require.True(t, reflect.DeepEqual(gpResp.Policy.PolicyData, policies[0].PolicyData))
	require.Equal(t, policies[0].Response, "pass")

	require.True(t, reflect.DeepEqual(tpResp.Policy.PolicyData, policies[1].PolicyData))
	require.Equal(t, policies[1].Response, "") // policy didn't "run"

	// Try to create a global policy with an existing name.
	s.DoJSON("POST", "/api/latest/fleet/policies", gpParams, http.StatusConflict, &gpResp)
	// Try to create a team policy with an existing name.
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team1.ID), tpParams, http.StatusConflict, &tpResp)
}

func (s *integrationTestSuite) TestListActivities() {
	t := s.T()

	ctx := context.Background()
	u := s.users["admin1@example.com"]

	prevActivities, _, err := s.ds.ListActivities(ctx, fleet.ListActivitiesOptions{})
	require.NoError(t, err)

	err = s.ds.NewActivity(ctx, &u, fleet.ActivityTypeAppliedSpecPack{})
	require.NoError(t, err)

	err = s.ds.NewActivity(ctx, &u, fleet.ActivityTypeDeletedPack{})
	require.NoError(t, err)

	err = s.ds.NewActivity(ctx, &u, fleet.ActivityTypeEditedPack{})
	require.NoError(t, err)

	lenPage := len(prevActivities) + 2

	var listResp listActivitiesResponse
	s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &listResp, "per_page", strconv.Itoa(lenPage), "order_key", "id")
	require.Len(t, listResp.Activities, lenPage)
	require.NotNil(t, listResp.Meta)
	assert.Equal(t, fleet.ActivityTypeAppliedSpecPack{}.ActivityName(), listResp.Activities[lenPage-2].Type)
	assert.Equal(t, fleet.ActivityTypeDeletedPack{}.ActivityName(), listResp.Activities[lenPage-1].Type)

	s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &listResp, "per_page", strconv.Itoa(lenPage), "order_key", "id", "page", "1")
	require.Len(t, listResp.Activities, 1)
	assert.Equal(t, fleet.ActivityTypeEditedPack{}.ActivityName(), listResp.Activities[0].Type)

	s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &listResp, "per_page", "1", "order_key", "id", "order_direction", "desc")
	require.Len(t, listResp.Activities, 1)
	assert.Equal(t, fleet.ActivityTypeEditedPack{}.ActivityName(), listResp.Activities[0].Type)

	listResp = listActivitiesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &listResp, "per_page", "1", "order_key", "a.id", "after", "0")
	require.Len(t, listResp.Activities, 1)
	require.Nil(t, listResp.Meta)
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
	s.DoJSON("GET", "/api/latest/fleet/carves", nil, http.StatusOK, &listResp, "per_page", "2", "order_key", "id")
	require.Len(t, listResp.Carves, 2)
	assert.Equal(t, c1.ID, listResp.Carves[0].ID)
	assert.Equal(t, c3.ID, listResp.Carves[1].ID)

	// include expired
	s.DoJSON("GET", "/api/latest/fleet/carves", nil, http.StatusOK, &listResp, "per_page", "2", "order_key", "id", "expired", "1")
	require.Len(t, listResp.Carves, 2)
	assert.Equal(t, c1.ID, listResp.Carves[0].ID)
	assert.Equal(t, c2.ID, listResp.Carves[1].ID)

	// empty page
	s.DoJSON("GET", "/api/latest/fleet/carves", nil, http.StatusOK, &listResp, "page", "3", "per_page", "2", "order_key", "id", "expired", "1")
	require.Len(t, listResp.Carves, 0)

	// get specific carve
	var getResp getCarveResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/carves/%d", c2.ID), nil, http.StatusOK, &getResp)
	require.Equal(t, c2.ID, getResp.Carve.ID)
	require.True(t, getResp.Carve.Expired)

	// get non-existing carve
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/carves/%d", c3.ID+1), nil, http.StatusNotFound, &getResp)

	// get expired carve block
	var blkResp getCarveBlockResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/carves/%d/block/%d", c2.ID, 1), nil, http.StatusInternalServerError, &blkResp)

	// get valid carve block, but block not inserted yet
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/carves/%d/block/%d", c1.ID, 1), nil, http.StatusNotFound, &blkResp)

	require.NoError(t, s.ds.NewBlock(ctx, c1, 1, []byte("block1")))
	require.NoError(t, s.ds.NewBlock(ctx, c1, 2, []byte("block2")))
	require.NoError(t, s.ds.NewBlock(ctx, c1, 3, []byte("block3")))

	// get valid carve block
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/carves/%d/block/%d", c1.ID, 1), nil, http.StatusOK, &blkResp)
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
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/refetch", hosts[0].ID), nil, http.StatusOK, &refetchResp)
	require.NoError(t, refetchResp.Err)

	// refetch unknown
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/refetch", hosts[2].ID+1), nil, http.StatusNotFound, &refetchResp)

	// get by identifier unknown
	var getResp getHostResponse
	s.DoJSON("GET", "/api/latest/fleet/hosts/identifier/no-such-host", nil, http.StatusNotFound, &getResp)

	// get by identifier valid
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/identifier/%s", hosts[0].UUID), nil, http.StatusOK, &getResp)
	require.Equal(t, hosts[0].ID, getResp.Host.ID)
	require.Nil(t, getResp.Host.TeamID)

	// assign hosts to team 1
	var addResp addHostsToTeamResponse
	s.DoJSON("POST", "/api/latest/fleet/hosts/transfer", addHostsToTeamRequest{
		TeamID:  &tm1.ID,
		HostIDs: []uint{hosts[0].ID, hosts[1].ID},
	}, http.StatusOK, &addResp)
	s.lastActivityOfTypeMatches(
		fleet.ActivityTypeTransferredHostsToTeam{}.ActivityName(),
		fmt.Sprintf(`{"team_id": %d, "team_name": %q, "host_ids": [%d, %d], "host_display_names": [%q, %q]}`,
			tm1.ID, tm1.Name, hosts[0].ID, hosts[1].ID, hosts[0].DisplayName(), hosts[1].DisplayName()),
		0,
	)

	// check that hosts are now part of that team
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", hosts[0].ID), nil, http.StatusOK, &getResp)
	require.NotNil(t, getResp.Host.TeamID)
	require.Equal(t, tm1.ID, *getResp.Host.TeamID)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", hosts[1].ID), nil, http.StatusOK, &getResp)
	require.NotNil(t, getResp.Host.TeamID)
	require.Equal(t, tm1.ID, *getResp.Host.TeamID)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", hosts[2].ID), nil, http.StatusOK, &getResp)
	require.Nil(t, getResp.Host.TeamID)

	// assign host to team 2 with filter
	var addfResp addHostsToTeamByFilterResponse
	req := addHostsToTeamByFilterRequest{TeamID: &tm2.ID}
	req.Filters.MatchQuery = hosts[2].Hostname
	s.DoJSON("POST", "/api/latest/fleet/hosts/transfer/filter", req, http.StatusOK, &addfResp)
	s.lastActivityOfTypeMatches(
		fleet.ActivityTypeTransferredHostsToTeam{}.ActivityName(),
		fmt.Sprintf(`{"team_id": %d, "team_name": %q, "host_ids": [%d], "host_display_names": [%q]}`,
			tm2.ID, tm2.Name, hosts[2].ID, hosts[2].DisplayName()),
		0,
	)

	// check that host 2 is now part of team 2
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", hosts[2].ID), nil, http.StatusOK, &getResp)
	require.NotNil(t, getResp.Host.TeamID)
	require.Equal(t, tm2.ID, *getResp.Host.TeamID)

	// delete host 0
	var delResp deleteHostResponse
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d", hosts[0].ID), nil, http.StatusOK, &delResp)
	// delete non-existing host
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d", hosts[2].ID+1), nil, http.StatusNotFound, &delResp)

	// assign host 1 to no team
	s.DoJSON("POST", "/api/latest/fleet/hosts/transfer", addHostsToTeamRequest{
		TeamID:  nil,
		HostIDs: []uint{hosts[1].ID},
	}, http.StatusOK, &addResp)
	s.lastActivityOfTypeMatches(
		fleet.ActivityTypeTransferredHostsToTeam{}.ActivityName(),
		fmt.Sprintf(`{"team_id": null, "team_name": null, "host_ids": [%d], "host_display_names": [%q]}`,
			hosts[1].ID, hosts[1].DisplayName()),
		0,
	)

	// list the hosts
	var listResp listHostsResponse
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listResp, "per_page", "3")
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
	s.DoJSON("POST", "/api/latest/fleet/packs", reqPack, http.StatusOK, &createPackResp)
	pack := createPackResp.Pack.Pack

	// try a non existent query
	s.Do("GET", fmt.Sprintf("/api/latest/fleet/queries/%d", 9999), nil, http.StatusNotFound)

	// list queries
	var listQryResp listQueriesResponse
	s.DoJSON("GET", "/api/latest/fleet/queries", nil, http.StatusOK, &listQryResp)
	assert.Len(t, listQryResp.Queries, 0)

	// create a query
	var createQueryResp createQueryResponse
	reqQuery := &fleet.QueryPayload{
		Name:  ptr.String(strings.ReplaceAll(t.Name(), "/", "_")),
		Query: ptr.String("select * from time;"),
	}
	s.DoJSON("POST", "/api/latest/fleet/queries", reqQuery, http.StatusOK, &createQueryResp)
	query := createQueryResp.Query

	// listing returns that query
	s.DoJSON("GET", "/api/latest/fleet/queries", nil, http.StatusOK, &listQryResp)
	require.Len(t, listQryResp.Queries, 1)
	assert.Equal(t, query.Name, listQryResp.Queries[0].Name)

	// next page returns nothing
	s.DoJSON("GET", "/api/latest/fleet/queries", nil, http.StatusOK, &listQryResp, "per_page", "2", "page", "1")
	require.Len(t, listQryResp.Queries, 0)

	// getting that query works
	var getQryResp getQueryResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d", query.ID), nil, http.StatusOK, &getQryResp)
	assert.Equal(t, query.ID, getQryResp.Query.ID)

	// list scheduled queries in pack, none yet
	var getInPackResp getScheduledQueriesInPackResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/packs/%d/scheduled", pack.ID), nil, http.StatusOK, &getInPackResp)
	assert.Len(t, getInPackResp.Scheduled, 0)

	// list scheduled queries in non-existing pack
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/packs/%d/scheduled", pack.ID+1), nil, http.StatusOK, &getInPackResp)
	assert.Len(t, getInPackResp.Scheduled, 0)

	// create scheduled query
	var createResp scheduleQueryResponse
	reqSQ := &scheduleQueryRequest{
		PackID:   pack.ID,
		QueryID:  query.ID,
		Interval: 1,
	}
	s.DoJSON("POST", "/api/latest/fleet/packs/schedule", reqSQ, http.StatusOK, &createResp)
	sq1 := createResp.Scheduled.ScheduledQuery
	assert.NotZero(t, sq1.ID)
	assert.Equal(t, uint(1), sq1.Interval)

	// create scheduled query with invalid pack
	reqSQ = &scheduleQueryRequest{
		PackID:   pack.ID + 1,
		QueryID:  query.ID,
		Interval: 2,
	}
	s.DoJSON("POST", "/api/latest/fleet/packs/schedule", reqSQ, http.StatusUnprocessableEntity, &createResp)

	// create scheduled query with invalid query
	reqSQ = &scheduleQueryRequest{
		PackID:   pack.ID,
		QueryID:  query.ID + 1,
		Interval: 3,
	}
	s.DoJSON("POST", "/api/latest/fleet/packs/schedule", reqSQ, http.StatusNotFound, &createResp)

	// list scheduled queries in pack
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/packs/%d/scheduled", pack.ID), nil, http.StatusOK, &getInPackResp)
	require.Len(t, getInPackResp.Scheduled, 1)
	assert.Equal(t, sq1.ID, getInPackResp.Scheduled[0].ID)

	// list scheduled queries in pack, next page
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/packs/%d/scheduled", pack.ID), nil, http.StatusOK, &getInPackResp, "page", "1", "per_page", "2")
	require.Len(t, getInPackResp.Scheduled, 0)

	// get non-existing scheduled query
	var getResp getScheduledQueryResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/schedule/%d", sq1.ID+1), nil, http.StatusNotFound, &getResp)

	// get existing scheduled query
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/schedule/%d", sq1.ID), nil, http.StatusOK, &getResp)
	assert.Equal(t, sq1.ID, getResp.Scheduled.ID)
	assert.Equal(t, sq1.Interval, getResp.Scheduled.Interval)

	// modify scheduled query
	var modResp modifyScheduledQueryResponse
	reqMod := fleet.ScheduledQueryPayload{
		Interval: ptr.Uint(4),
	}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/packs/schedule/%d", sq1.ID), reqMod, http.StatusOK, &modResp)
	assert.Equal(t, sq1.ID, modResp.Scheduled.ID)
	assert.Equal(t, uint(4), modResp.Scheduled.Interval)

	// modify non-existing scheduled query
	reqMod = fleet.ScheduledQueryPayload{
		Interval: ptr.Uint(5),
	}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/packs/schedule/%d", sq1.ID+1), reqMod, http.StatusNotFound, &modResp)

	// delete non-existing scheduled query
	var delResp deleteScheduledQueryResponse
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/packs/schedule/%d", sq1.ID+1), nil, http.StatusNotFound, &delResp)

	// delete existing scheduled query
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/packs/schedule/%d", sq1.ID), nil, http.StatusOK, &delResp)

	// get the now-deleted scheduled query
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/schedule/%d", sq1.ID), nil, http.StatusNotFound, &getResp)

	// modify the query
	var modQryResp modifyQueryResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/queries/%d", query.ID), fleet.QueryPayload{Description: ptr.String("updated")}, http.StatusOK, &modQryResp)
	assert.Equal(t, "updated", modQryResp.Query.Description)

	// modify a non-existing query
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/queries/%d", query.ID+1), fleet.QueryPayload{Description: ptr.String("updated")}, http.StatusNotFound, &modQryResp)

	// delete the query by name
	var delByNameResp deleteQueryResponse
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/queries/%s", query.Name), nil, http.StatusOK, &delByNameResp)

	// delete unknown query by name (i.e. the same, now deleted)
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/queries/%s", query.Name), nil, http.StatusNotFound, &delByNameResp)

	// create another query
	reqQuery = &fleet.QueryPayload{
		Name:  ptr.String(strings.ReplaceAll(t.Name(), "/", "_") + "_2"),
		Query: ptr.String("select 2"),
	}
	s.DoJSON("POST", "/api/latest/fleet/queries", reqQuery, http.StatusOK, &createQueryResp)
	query2 := createQueryResp.Query

	// delete it by id
	var delByIDResp deleteQueryByIDResponse
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/queries/id/%d", query2.ID), nil, http.StatusOK, &delByIDResp)

	// delete unknown query by id (same id just deleted)
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/queries/id/%d", query2.ID), nil, http.StatusNotFound, &delByIDResp)

	// create another query
	reqQuery = &fleet.QueryPayload{
		Name:  ptr.String(strings.ReplaceAll(t.Name(), "/", "_") + "_3"),
		Query: ptr.String("select 3"),
	}
	s.DoJSON("POST", "/api/latest/fleet/queries", reqQuery, http.StatusOK, &createQueryResp)
	query3 := createQueryResp.Query

	// batch-delete by id, 3 ids, only one exists
	var delBatchResp deleteQueriesResponse
	s.DoJSON("POST", "/api/latest/fleet/queries/delete", map[string]interface{}{
		"ids": []uint{query.ID, query2.ID, query3.ID},
	}, http.StatusOK, &delBatchResp)
	assert.Equal(t, uint(1), delBatchResp.Deleted)

	// batch-delete by id, none exist
	delBatchResp.Deleted = 0
	s.DoJSON("POST", "/api/latest/fleet/queries/delete", map[string]interface{}{
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
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/device_mapping", hosts[2].ID+1), nil, http.StatusNotFound, &listResp)

	// existing host but none yet
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/device_mapping", hosts[0].ID), nil, http.StatusOK, &listResp)
	require.Len(t, listResp.DeviceMapping, 0)

	// create some mappings
	require.NoError(t, s.ds.ReplaceHostDeviceMapping(ctx, hosts[0].ID, []*fleet.HostDeviceMapping{
		{HostID: hosts[0].ID, Email: "a@b.c", Source: "google_chrome_profiles"},
		{HostID: hosts[0].ID, Email: "b@b.c", Source: "google_chrome_profiles"},
	}))

	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/device_mapping", hosts[0].ID), nil, http.StatusOK, &listResp)
	require.Len(t, listResp.DeviceMapping, 2)
	require.Equal(t, "a@b.c", listResp.DeviceMapping[0].Email)
	require.Equal(t, "google_chrome_profiles", listResp.DeviceMapping[0].Source)
	require.Zero(t, listResp.DeviceMapping[0].HostID)
	require.Equal(t, "b@b.c", listResp.DeviceMapping[1].Email)
	require.Equal(t, "google_chrome_profiles", listResp.DeviceMapping[1].Source)
	require.Zero(t, listResp.DeviceMapping[1].HostID)
	require.Equal(t, hosts[0].ID, listResp.HostID)

	// other host still has none
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/device_mapping", hosts[1].ID), nil, http.StatusOK, &listResp)
	require.Len(t, listResp.DeviceMapping, 0)

	var listHosts listHostsResponse
	// list hosts response includes device mappings
	s.DoJSON("GET", "/api/latest/fleet/hosts?device_mapping=true", nil, http.StatusOK, &listHosts)
	require.Len(t, listHosts.Hosts, 3)
	hostsByID := make(map[uint]fleet.HostResponse)
	for _, h := range listHosts.Hosts {
		hostsByID[h.ID] = h
	}
	var dm []*fleet.HostDeviceMapping

	// device mapping for host 1
	host1 := hosts[0]
	require.NotNil(t, *hostsByID[host1.ID].DeviceMapping)

	err := json.Unmarshal(*hostsByID[host1.ID].DeviceMapping, &dm)
	require.NoError(t, err)
	assert.Len(t, dm, 2)

	var emails []string
	for _, e := range dm {
		emails = append(emails, e.Email)
	}
	assert.Contains(t, emails, "a@b.c")
	assert.Contains(t, emails, "b@b.c")
	assert.Equal(t, "google_chrome_profiles", dm[0].Source)
	assert.Equal(t, "google_chrome_profiles", dm[1].Source)

	// no device mapping for other hosts
	assert.Nil(t, hostsByID[hosts[1].ID].DeviceMapping)
	assert.Nil(t, hostsByID[hosts[2].ID].DeviceMapping)

	// search host by email address finds the corresponding host
	s.DoJSON("GET", "/api/latest/fleet/hosts?device_mapping=true", nil, http.StatusOK, &listHosts, "query", "a@b.c")
	require.Len(t, listHosts.Hosts, 1)
	require.Equal(t, host1.ID, listHosts.Hosts[0].ID)
	require.NotNil(t, listHosts.Hosts[0].DeviceMapping)

	err = json.Unmarshal(*listHosts.Hosts[0].DeviceMapping, &dm)
	require.NoError(t, err)
	assert.Len(t, dm, 2)

	for _, e := range dm {
		emails = append(emails, e.Email)
	}
	assert.Contains(t, emails, "a@b.c")
	assert.Contains(t, emails, "b@b.c")
	assert.Equal(t, "google_chrome_profiles", dm[0].Source)
	assert.Equal(t, "google_chrome_profiles", dm[1].Source)

	s.DoJSON("GET", "/api/latest/fleet/hosts?device_mapping=true", nil, http.StatusOK, &listHosts, "query", "c@b.c")
	require.Len(t, listHosts.Hosts, 0)
}

func (s *integrationTestSuite) TestListHostsDeviceMappingSize() {
	t := s.T()
	ctx := context.Background()
	hosts := s.createHosts(t)

	testSize := 50
	var mappings []*fleet.HostDeviceMapping
	for i := 0; i < testSize; i++ {
		testEmail, _ := server.GenerateRandomText(14)
		mappings = append(mappings, &fleet.HostDeviceMapping{HostID: hosts[0].ID, Email: testEmail, Source: "google_chrome_profiles"})
	}

	require.NoError(t, s.ds.ReplaceHostDeviceMapping(ctx, hosts[0].ID, mappings))

	var listHosts listHostsResponse
	s.DoJSON("GET", "/api/latest/fleet/hosts?device_mapping=true", nil, http.StatusOK, &listHosts)

	hostsByID := make(map[uint]fleet.HostResponse)
	for _, h := range listHosts.Hosts {
		hostsByID[h.ID] = h
	}
	require.NotNil(t, *hostsByID[hosts[0].ID].DeviceMapping)

	var dm []*fleet.HostDeviceMapping
	err := json.Unmarshal(*hostsByID[hosts[0].ID].DeviceMapping, &dm)
	require.NoError(t, err)
	require.Len(t, dm, testSize)
}

type macadminsDataResponse struct {
	Macadmins *struct {
		Munki       *fleet.HostMunkiInfo    `json:"munki"`
		MunkiIssues []*fleet.HostMunkiIssue `json:"munki_issues"`
		MDM         *struct {
			EnrollmentStatus string  `json:"enrollment_status"`
			ServerURL        string  `json:"server_url"`
			Name             *string `json:"name"`
			ID               *uint   `json:"id"`
		} `json:"mobile_device_management"`
	} `json:"macadmins"`
}

func (s *integrationTestSuite) TestGetMacadminsData() {
	t := s.T()

	ctx := context.Background()

	hostAll, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String(t.Name() + "1"),
		UUID:            t.Name() + "1",
		Hostname:        t.Name() + "foo.local",
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-58",
		OsqueryHostID:   ptr.String("1"),
		Platform:        "darwin",
	})
	require.NoError(t, err)
	require.NotNil(t, hostAll)

	hostNothing, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String(t.Name() + "2"),
		UUID:            t.Name() + "2",
		Hostname:        t.Name() + "foo.local2",
		PrimaryIP:       "192.168.1.2",
		PrimaryMac:      "30-65-EC-6F-C4-59",
		OsqueryHostID:   ptr.String("2"),
		Platform:        "darwin",
	})
	require.NoError(t, err)
	require.NotNil(t, hostNothing)

	hostOnlyMunki, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String(t.Name() + "3"),
		UUID:            t.Name() + "3",
		Hostname:        t.Name() + "foo.local3",
		PrimaryIP:       "192.168.1.3",
		PrimaryMac:      "30-65-EC-6F-C4-5F",
		OsqueryHostID:   ptr.String("3"),
		Platform:        "darwin",
	})
	require.NoError(t, err)
	require.NotNil(t, hostOnlyMunki)

	hostOnlyMDM, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String(t.Name() + "4"),
		UUID:            t.Name() + "4",
		Hostname:        t.Name() + "foo.local4",
		PrimaryIP:       "192.168.1.4",
		PrimaryMac:      "30-65-EC-6F-C4-5A",
		OsqueryHostID:   ptr.String("4"),
		Platform:        "darwin",
	})
	require.NoError(t, err)
	require.NotNil(t, hostOnlyMDM)

	hostMDMNoID, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String(t.Name() + "5"),
		UUID:            t.Name() + "5",
		Hostname:        t.Name() + "foo.local5",
		PrimaryIP:       "192.168.1.5",
		PrimaryMac:      "30-65-EC-6F-D5-5A",
		OsqueryHostID:   ptr.String("5"),
		Platform:        "darwin",
	})
	require.NoError(t, err)
	require.NotNil(t, hostMDMNoID)

	// insert a host_mdm row for hostMDMNoID without any mdm_id
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx,
			`INSERT INTO host_mdm (host_id, enrolled, server_url, installed_from_dep, is_server) VALUES (?, ?, ?, ?, ?)`,
			hostMDMNoID.ID, true, "https://simplemdm.com", true, false)
		return err
	})

	require.NoError(t, s.ds.SetOrUpdateMDMData(ctx, hostAll.ID, false, true, "url", false, ""))
	require.NoError(t, s.ds.SetOrUpdateMunkiInfo(ctx, hostAll.ID, "1.3.0", []string{"error1"}, []string{"warning1"}))

	macadminsData := macadminsDataResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/macadmins", hostAll.ID), nil, http.StatusOK, &macadminsData)
	require.NotNil(t, macadminsData.Macadmins)
	assert.Equal(t, "url", macadminsData.Macadmins.MDM.ServerURL)
	assert.Equal(t, "On (manual)", macadminsData.Macadmins.MDM.EnrollmentStatus)
	assert.Nil(t, macadminsData.Macadmins.MDM.Name)
	require.NotNil(t, macadminsData.Macadmins.MDM.ID)
	assert.NotZero(t, *macadminsData.Macadmins.MDM.ID)
	assert.Equal(t, "1.3.0", macadminsData.Macadmins.Munki.Version)

	require.Len(t, macadminsData.Macadmins.MunkiIssues, 2)
	sort.Slice(macadminsData.Macadmins.MunkiIssues, func(i, j int) bool {
		l, r := macadminsData.Macadmins.MunkiIssues[i], macadminsData.Macadmins.MunkiIssues[j]
		return l.Name < r.Name
	})
	assert.NotZero(t, macadminsData.Macadmins.MunkiIssues[0].MunkiIssueID)
	assert.False(t, macadminsData.Macadmins.MunkiIssues[0].HostIssueCreatedAt.IsZero())
	assert.Equal(t, "error1", macadminsData.Macadmins.MunkiIssues[0].Name)
	assert.Equal(t, "error", macadminsData.Macadmins.MunkiIssues[0].IssueType)
	assert.Equal(t, "warning1", macadminsData.Macadmins.MunkiIssues[1].Name)
	assert.NotZero(t, macadminsData.Macadmins.MunkiIssues[1].MunkiIssueID)
	assert.False(t, macadminsData.Macadmins.MunkiIssues[1].HostIssueCreatedAt.IsZero())
	assert.Equal(t, "warning", macadminsData.Macadmins.MunkiIssues[1].IssueType)

	require.NoError(t, s.ds.SetOrUpdateMDMData(ctx, hostAll.ID, false, true, "https://simplemdm.com", true, fleet.WellKnownMDMSimpleMDM))
	require.NoError(t, s.ds.SetOrUpdateMunkiInfo(ctx, hostAll.ID, "1.5.0", []string{"error1"}, nil))

	macadminsData = macadminsDataResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/macadmins", hostAll.ID), nil, http.StatusOK, &macadminsData)
	require.NotNil(t, macadminsData.Macadmins)
	assert.Equal(t, "https://simplemdm.com", macadminsData.Macadmins.MDM.ServerURL)
	assert.Equal(t, "On (automatic)", macadminsData.Macadmins.MDM.EnrollmentStatus)
	require.NotNil(t, macadminsData.Macadmins.MDM.Name)
	assert.Equal(t, fleet.WellKnownMDMSimpleMDM, *macadminsData.Macadmins.MDM.Name)
	require.NotNil(t, macadminsData.Macadmins.MDM.ID)
	assert.NotZero(t, *macadminsData.Macadmins.MDM.ID)
	assert.Equal(t, "1.5.0", macadminsData.Macadmins.Munki.Version)
	require.Len(t, macadminsData.Macadmins.MunkiIssues, 1)
	assert.Equal(t, "error1", macadminsData.Macadmins.MunkiIssues[0].Name)

	require.NoError(t, s.ds.SetOrUpdateMDMData(ctx, hostAll.ID, false, false, "url2", false, ""))

	macadminsData = macadminsDataResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/macadmins", hostAll.ID), nil, http.StatusOK, &macadminsData)
	require.NotNil(t, macadminsData.Macadmins)
	assert.Equal(t, "Off", macadminsData.Macadmins.MDM.EnrollmentStatus)
	assert.Nil(t, macadminsData.Macadmins.MDM.Name)
	require.NotNil(t, macadminsData.Macadmins.MDM.ID)
	assert.NotZero(t, *macadminsData.Macadmins.MDM.ID)
	assert.Len(t, macadminsData.Macadmins.MunkiIssues, 1)

	// nothing returns null
	macadminsData = macadminsDataResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/macadmins", hostNothing.ID), nil, http.StatusOK, &macadminsData)
	require.Nil(t, macadminsData.Macadmins)

	// only munki info returns null on mdm
	require.NoError(t, s.ds.SetOrUpdateMunkiInfo(ctx, hostOnlyMunki.ID, "3.2.0", nil, []string{"warning1"}))
	macadminsData = macadminsDataResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/macadmins", hostOnlyMunki.ID), nil, http.StatusOK, &macadminsData)
	require.NotNil(t, macadminsData.Macadmins)
	require.Nil(t, macadminsData.Macadmins.MDM)
	require.NotNil(t, macadminsData.Macadmins.Munki)
	assert.Equal(t, "3.2.0", macadminsData.Macadmins.Munki.Version)
	require.Len(t, macadminsData.Macadmins.MunkiIssues, 1)
	assert.Equal(t, "warning1", macadminsData.Macadmins.MunkiIssues[0].Name)

	// only mdm returns null on munki info
	require.NoError(t, s.ds.SetOrUpdateMDMData(ctx, hostOnlyMDM.ID, false, true, "https://kandji.io", true, fleet.WellKnownMDMKandji))
	macadminsData = macadminsDataResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/macadmins", hostOnlyMDM.ID), nil, http.StatusOK, &macadminsData)
	require.NotNil(t, macadminsData.Macadmins)
	require.NotNil(t, macadminsData.Macadmins.MDM)
	require.NotNil(t, macadminsData.Macadmins.MDM.Name)
	assert.Equal(t, fleet.WellKnownMDMKandji, *macadminsData.Macadmins.MDM.Name)
	require.NotNil(t, macadminsData.Macadmins.MDM.ID)
	assert.NotZero(t, *macadminsData.Macadmins.MDM.ID)
	require.Nil(t, macadminsData.Macadmins.Munki)
	require.Len(t, macadminsData.Macadmins.MunkiIssues, 0)
	assert.Equal(t, "https://kandji.io", macadminsData.Macadmins.MDM.ServerURL)
	assert.Equal(t, "On (automatic)", macadminsData.Macadmins.MDM.EnrollmentStatus)

	// host without mdm_id still works, returns nil id and unknown name
	macadminsData = macadminsDataResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/macadmins", hostMDMNoID.ID), nil, http.StatusOK, &macadminsData)
	require.NotNil(t, macadminsData.Macadmins)
	require.NotNil(t, macadminsData.Macadmins.MDM)
	assert.Nil(t, macadminsData.Macadmins.MDM.Name)
	assert.Nil(t, macadminsData.Macadmins.MDM.ID)
	require.Nil(t, macadminsData.Macadmins.Munki)
	assert.Equal(t, "On (automatic)", macadminsData.Macadmins.MDM.EnrollmentStatus)

	// generate aggregated data
	require.NoError(t, s.ds.GenerateAggregatedMunkiAndMDM(context.Background()))

	agg := getAggregatedMacadminsDataResponse{}
	s.DoJSON("GET", "/api/latest/fleet/macadmins", nil, http.StatusOK, &agg)
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
	require.Len(t, agg.Macadmins.MunkiIssues, 2)
	// ignore ids
	agg.Macadmins.MunkiIssues[0].ID = 0
	agg.Macadmins.MunkiIssues[1].ID = 0
	assert.ElementsMatch(t, agg.Macadmins.MunkiIssues, []fleet.AggregatedMunkiIssue{
		{
			MunkiIssue: fleet.MunkiIssue{
				Name:      "error1",
				IssueType: "error",
			},
			HostsCount: 1,
		},
		{
			MunkiIssue: fleet.MunkiIssue{
				Name:      "warning1",
				IssueType: "warning",
			},
			HostsCount: 1,
		},
	})
	assert.Equal(t, agg.Macadmins.MDMStatus.EnrolledManualHostsCount, 0)
	assert.Equal(t, agg.Macadmins.MDMStatus.EnrolledAutomatedHostsCount, 2)
	assert.Equal(t, agg.Macadmins.MDMStatus.UnenrolledHostsCount, 1)
	assert.Equal(t, agg.Macadmins.MDMStatus.HostsCount, 3)
	require.Len(t, agg.Macadmins.MDMSolutions, 2)
	for _, sol := range agg.Macadmins.MDMSolutions {
		switch sol.ServerURL {
		case "url2":
			assert.Equal(t, fleet.UnknownMDMName, sol.Name)
			assert.Equal(t, 1, sol.HostsCount)
		case "https://kandji.io":
			assert.Equal(t, fleet.WellKnownMDMKandji, sol.Name)
			assert.Equal(t, 1, sol.HostsCount)
		default:
			require.Fail(t, "unknown MDM server URL: %s", sol.ServerURL)
		}
	}

	// TODO: ideally we'd pull this out into its own function that specifically tests
	// the mdm summary endpoint. We can add additional tests for testing the platform
	// and team_id query params for this endpoint.
	mdmAgg := getHostMDMSummaryResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts/summary/mdm", nil, http.StatusOK, &mdmAgg)
	assert.NotZero(t, mdmAgg.AggregatedMDMData.CountsUpdatedAt)

	team, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		Name:        "team1" + t.Name(),
		Description: "desc team1",
	})
	require.NoError(t, err)

	agg = getAggregatedMacadminsDataResponse{}
	s.DoJSON("GET", "/api/latest/fleet/macadmins", nil, http.StatusOK, &agg, "team_id", fmt.Sprint(team.ID))
	require.NotNil(t, agg.Macadmins)
	require.Empty(t, agg.Macadmins.MunkiVersions)
	require.Empty(t, agg.Macadmins.MunkiIssues)
	require.Empty(t, agg.Macadmins.MDMStatus)
	require.Empty(t, agg.Macadmins.MDMSolutions)

	agg = getAggregatedMacadminsDataResponse{}
	s.DoJSON("GET", "/api/latest/fleet/macadmins", nil, http.StatusNotFound, &agg, "team_id", "9999999")

	// Hardcode response type because we are using a custom json marshaling so
	// using getHostMDMResponse fails with "JSON unmarshaling is not supported for HostMDM".
	type jsonMDM struct {
		EnrollmentStatus string `json:"enrollment_status"`
		ServerURL        string `json:"server_url"`
		Name             string `json:"name,omitempty"`
		ID               *uint  `json:"id,omitempty"`
	}
	type getHostMDMResponseTest struct {
		HostMDM *jsonMDM
		Err     error `json:"error,omitempty"`
	}
	ghr := getHostMDMResponseTest{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/mdm", hostNothing.ID), nil, http.StatusOK, &ghr)
	require.Nil(t, ghr.HostMDM)

	ghr = getHostMDMResponseTest{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/mdm", 999999), nil, http.StatusNotFound, &ghr)
	require.Nil(t, ghr.HostMDM)
}

func (s *integrationTestSuite) TestLabels() {
	t := s.T()

	// list labels, has the built-in ones
	var listResp listLabelsResponse
	s.DoJSON("GET", "/api/latest/fleet/labels", nil, http.StatusOK, &listResp)
	assert.True(t, len(listResp.Labels) > 0)
	for _, lbl := range listResp.Labels {
		assert.Equal(t, fleet.LabelTypeBuiltIn, lbl.LabelType)
	}
	builtInsCount := len(listResp.Labels)

	// labels summary has the built-in ones
	var summaryResp getLabelsSummaryResponse
	s.DoJSON("GET", "/api/latest/fleet/labels/summary", nil, http.StatusOK, &summaryResp)
	assert.Len(t, summaryResp.Labels, builtInsCount)
	for _, lbl := range summaryResp.Labels {
		assert.Equal(t, fleet.LabelTypeBuiltIn, lbl.LabelType)
	}

	// create a label without name, an error
	var createResp createLabelResponse
	s.DoJSON("POST", "/api/latest/fleet/labels", &fleet.LabelPayload{Query: ptr.String("select 1")}, http.StatusUnprocessableEntity, &createResp)

	// create a valid label
	s.DoJSON("POST", "/api/latest/fleet/labels", &fleet.LabelPayload{Name: ptr.String(t.Name()), Query: ptr.String("select 1")}, http.StatusOK, &createResp)
	assert.NotZero(t, createResp.Label.ID)
	assert.Equal(t, t.Name(), createResp.Label.Name)
	lbl1 := createResp.Label.Label

	// get the label
	var getResp getLabelResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/labels/%d", lbl1.ID), nil, http.StatusOK, &getResp)
	assert.Equal(t, lbl1.ID, getResp.Label.ID)

	// get a non-existing label
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/labels/%d", lbl1.ID+1), nil, http.StatusNotFound, &getResp)

	// modify that label
	var modResp modifyLabelResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/labels/%d", lbl1.ID), &fleet.ModifyLabelPayload{Name: ptr.String(t.Name() + "zzz")}, http.StatusOK, &modResp)
	assert.Equal(t, lbl1.ID, modResp.Label.ID)
	assert.NotEqual(t, lbl1.Name, modResp.Label.Name)

	// modify a non-existing label
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/labels/%d", lbl1.ID+1), &fleet.ModifyLabelPayload{Name: ptr.String("zzz")}, http.StatusNotFound, &modResp)

	// list labels
	s.DoJSON("GET", "/api/latest/fleet/labels", nil, http.StatusOK, &listResp, "per_page", strconv.Itoa(builtInsCount+1))
	assert.Len(t, listResp.Labels, builtInsCount+1)

	// labels summary
	s.DoJSON("GET", "/api/latest/fleet/labels/summary", nil, http.StatusOK, &summaryResp)
	assert.Len(t, summaryResp.Labels, builtInsCount+1)

	// next page is empty
	s.DoJSON("GET", "/api/latest/fleet/labels", nil, http.StatusOK, &listResp, "per_page", "2", "page", "1", "query", t.Name())
	assert.Len(t, listResp.Labels, 0)

	// create another label
	s.DoJSON("POST", "/api/latest/fleet/labels", &fleet.LabelPayload{Name: ptr.String(strings.ReplaceAll(t.Name(), "/", "_")), Query: ptr.String("select 1")}, http.StatusOK, &createResp)
	assert.NotZero(t, createResp.Label.ID)
	lbl2 := createResp.Label.Label

	// create hosts and add them to that label
	hosts := s.createHosts(t, "darwin", "darwin", "darwin")
	for _, h := range hosts {
		err := s.ds.RecordLabelQueryExecutions(context.Background(), h, map[uint]*bool{lbl2.ID: ptr.Bool(true)}, time.Now(), false)
		require.NoError(t, err)
	}

	// list hosts in label
	var listHostsResp listHostsResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/labels/%d/hosts", lbl2.ID), nil, http.StatusOK, &listHostsResp)
	assert.Len(t, listHostsResp.Hosts, len(hosts))

	// list hosts in label searching by display_name
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/labels/%d/hosts", lbl2.ID), nil, http.StatusOK, &listHostsResp, "order_key", "display_name", "order_direction", "desc")
	assert.Len(t, listHostsResp.Hosts, len(hosts))
	// first in the list is the last one, as the names are ordered with the index
	// of creation, and vice-versa
	assert.Equal(t, hosts[len(hosts)-1].ID, listHostsResp.Hosts[0].ID)
	assert.Equal(t, hosts[0].ID, listHostsResp.Hosts[len(hosts)-1].ID)

	// count hosts in label order by display_name
	var countResp countHostsResponse
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp, "label_id", fmt.Sprint(lbl2.ID), "order_key", "display_name", "order_direction", "desc")
	assert.Equal(t, len(hosts), countResp.Count)

	// lists hosts in label without hosts
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/labels/%d/hosts", lbl1.ID), nil, http.StatusOK, &listHostsResp)
	assert.Len(t, listHostsResp.Hosts, 0)

	// count hosts in label
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp, "label_id", fmt.Sprint(lbl1.ID))
	assert.Equal(t, 0, countResp.Count)

	// lists hosts in invalid label
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/labels/%d/hosts", lbl2.ID+1), nil, http.StatusOK, &listHostsResp)
	assert.Len(t, listHostsResp.Hosts, 0)

	// set MDM information on a host
	require.NoError(t, s.ds.SetOrUpdateMDMData(context.Background(), hosts[0].ID, false, true, "https://simplemdm.com", false, fleet.WellKnownMDMSimpleMDM))
	var mdmID uint
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), q, &mdmID,
			`SELECT id FROM mobile_device_management_solutions WHERE name = ? AND server_url = ?`, fleet.WellKnownMDMSimpleMDM, "https://simplemdm.com")
	})
	// generate aggregated stats
	require.NoError(t, s.ds.GenerateAggregatedMunkiAndMDM(context.Background()))

	// list host in label by mdm_id
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/labels/%d/hosts", lbl2.ID), nil, http.StatusOK, &listHostsResp, "mdm_id", fmt.Sprint(mdmID))
	require.Len(t, listHostsResp.Hosts, 1)
	assert.Nil(t, listHostsResp.Software)
	assert.Nil(t, listHostsResp.MunkiIssue)
	require.NotNil(t, listHostsResp.MDMSolution)
	assert.Equal(t, mdmID, listHostsResp.MDMSolution.ID)
	assert.Equal(t, fleet.WellKnownMDMSimpleMDM, listHostsResp.MDMSolution.Name)
	assert.Equal(t, "https://simplemdm.com", listHostsResp.MDMSolution.ServerURL)

	// delete a label by id
	var delIDResp deleteLabelByIDResponse
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/labels/id/%d", lbl1.ID), nil, http.StatusOK, &delIDResp)

	// delete a non-existing label by id
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/labels/id/%d", lbl2.ID+1), nil, http.StatusNotFound, &delIDResp)

	// delete a label by name
	var delResp deleteLabelResponse
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/labels/%s", url.PathEscape(lbl2.Name)), nil, http.StatusOK, &delResp)

	// delete a non-existing label by name
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/labels/%s", url.PathEscape(lbl2.Name)), nil, http.StatusNotFound, &delResp)

	// list labels, only the built-ins remain
	s.DoJSON("GET", "/api/latest/fleet/labels", nil, http.StatusOK, &listResp, "per_page", strconv.Itoa(builtInsCount+1))
	assert.Len(t, listResp.Labels, builtInsCount)
	for _, lbl := range listResp.Labels {
		assert.Equal(t, fleet.LabelTypeBuiltIn, lbl.LabelType)
	}

	// labels summary, only the built-ins remains
	s.DoJSON("GET", "/api/latest/fleet/labels/summary", nil, http.StatusOK, &summaryResp)
	assert.Len(t, summaryResp.Labels, builtInsCount)
	for _, lbl := range summaryResp.Labels {
		assert.Equal(t, fleet.LabelTypeBuiltIn, lbl.LabelType)
	}

	// host summary matches built-ins count
	var hostSummaryResp getHostSummaryResponse
	s.DoJSON("GET", "/api/latest/fleet/host_summary", nil, http.StatusOK, &hostSummaryResp)
	assert.Len(t, hostSummaryResp.BuiltinLabels, builtInsCount)
	for _, lbl := range hostSummaryResp.BuiltinLabels {
		assert.Equal(t, fleet.LabelTypeBuiltIn, lbl.LabelType)
	}
}

func (s *integrationTestSuite) TestLabelSpecs() {
	t := s.T()

	// list label specs, only those of the built-ins
	var listResp getLabelSpecsResponse
	s.DoJSON("GET", "/api/latest/fleet/spec/labels", nil, http.StatusOK, &listResp)
	assert.True(t, len(listResp.Specs) > 0)
	for _, spec := range listResp.Specs {
		assert.Equal(t, fleet.LabelTypeBuiltIn, spec.LabelType)
	}
	builtInsCount := len(listResp.Specs)

	name := strings.ReplaceAll(t.Name(), "/", "_")
	// apply an invalid label spec - dynamic membership with host specified
	var applyResp applyLabelSpecsResponse
	s.DoJSON("POST", "/api/latest/fleet/spec/labels", applyLabelSpecsRequest{
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
	s.DoJSON("POST", "/api/latest/fleet/spec/labels", applyLabelSpecsRequest{
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
	s.DoJSON("POST", "/api/latest/fleet/spec/labels", applyLabelSpecsRequest{
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
	s.DoJSON("GET", "/api/latest/fleet/spec/labels", nil, http.StatusOK, &listResp)
	assert.Len(t, listResp.Specs, builtInsCount+1)

	// get a specific label spec
	var getResp getLabelSpecResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/spec/labels/%s", url.PathEscape(name)), nil, http.StatusOK, &getResp)
	assert.Equal(t, name, getResp.Spec.Name)
	assert.NotEqual(t, 0, getResp.Spec.ID)

	// get a non-existing label spec
	s.DoJSON("GET", "/api/latest/fleet/spec/labels/zzz", nil, http.StatusNotFound, &getResp)
}

func (s *integrationTestSuite) TestUsers() {
	// ensure that on exit, the admin token is used
	defer func() { s.token = s.getTestAdminToken() }()

	t := s.T()

	// list existing users
	var listResp listUsersResponse
	s.DoJSON("GET", "/api/latest/fleet/users", nil, http.StatusOK, &listResp)
	assert.Len(t, listResp.Users, len(s.users))

	// test available teams returned by `/me` endpoint for existing user
	var getMeResp getUserResponse
	ssn := createSession(t, 1, s.ds)
	resp := s.DoRawWithHeaders("GET", "/api/latest/fleet/me", []byte(""), http.StatusOK, map[string]string{
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
	userRawPwd := test.GoodPassword
	params := fleet.UserPayload{
		Name:       ptr.String("extra"),
		Email:      ptr.String("extra@asd.com"),
		Password:   ptr.String(userRawPwd),
		GlobalRole: ptr.String(fleet.RoleObserver),
	}
	s.DoJSON("POST", "/api/latest/fleet/users/admin", params, http.StatusOK, &createResp)
	assert.NotZero(t, createResp.User.ID)
	assert.True(t, createResp.User.AdminForcedPasswordReset)
	u := *createResp.User

	// login as that user and check that teams info is empty
	var loginResp loginResponse
	s.DoJSON("POST", "/api/latest/fleet/login", params, http.StatusOK, &loginResp)
	require.Equal(t, loginResp.User.ID, u.ID)
	assert.Len(t, loginResp.User.Teams, 0)
	assert.Len(t, loginResp.AvailableTeams, 0)

	// get that user from `/users` endpoint and check that teams info is empty
	var getResp getUserResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/users/%d", u.ID), nil, http.StatusOK, &getResp)
	assert.Equal(t, u.ID, getResp.User.ID)
	assert.Len(t, getResp.User.Teams, 0)
	assert.Len(t, getResp.AvailableTeams, 0)

	// get non-existing user
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/users/%d", u.ID+1), nil, http.StatusNotFound, &getResp)

	// modify that user - simple name change
	var modResp modifyUserResponse
	params = fleet.UserPayload{
		Name: ptr.String("extraz"),
	}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/users/%d", u.ID), params, http.StatusOK, &modResp)
	assert.Equal(t, u.ID, modResp.User.ID)
	assert.Equal(t, u.Name+"z", modResp.User.Name)

	// modify that user - set an existing email
	params = fleet.UserPayload{
		Email: &getMeResp.User.Email,
	}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/users/%d", u.ID), params, http.StatusConflict, &modResp)

	// modify that user - set an email that has an invite for it
	createInviteReq := createInviteRequest{InvitePayload: fleet.InvitePayload{
		Email:      ptr.String("colliding@email.com"),
		Name:       ptr.String("some name"),
		GlobalRole: null.StringFrom(fleet.RoleAdmin),
	}}
	createInviteResp := createInviteResponse{}
	s.DoJSON("POST", "/api/latest/fleet/invites", createInviteReq, http.StatusOK, &createInviteResp)
	params = fleet.UserPayload{
		Email: ptr.String("colliding@email.com"),
	}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/users/%d", u.ID), params, http.StatusConflict, &modResp)

	// modify that user - set a non existent email
	params = fleet.UserPayload{
		Email: ptr.String("someemail@qowieuowh.com"),
	}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/users/%d", u.ID), params, http.StatusOK, &modResp)

	// modify user - email change, password does not match
	params = fleet.UserPayload{
		Email:    ptr.String("extra2@asd.com"),
		Password: ptr.String("wrongpass"),
	}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/users/%d", u.ID), params, http.StatusForbidden, &modResp)

	// modify user - email change, password ok
	params = fleet.UserPayload{
		Email:    ptr.String("extra2@asd.com"),
		Password: ptr.String(test.GoodPassword),
	}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/users/%d", u.ID), params, http.StatusOK, &modResp)
	assert.Equal(t, u.ID, modResp.User.ID)
	assert.NotEqual(t, u.ID, modResp.User.Email)

	// modify invalid user
	params = fleet.UserPayload{
		Name: ptr.String("nosuchuser"),
	}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/users/%d", u.ID+1), params, http.StatusNotFound, &modResp)

	// perform a required password change as the user themselves
	s.token = s.getTestToken(u.Email, userRawPwd)
	var perfPwdResetResp performRequiredPasswordResetResponse
	newRawPwd := test.GoodPassword2
	s.DoJSON("POST", "/api/latest/fleet/perform_required_password_reset", performRequiredPasswordResetRequest{
		Password: newRawPwd,
		ID:       u.ID,
	}, http.StatusOK, &perfPwdResetResp)
	assert.False(t, perfPwdResetResp.User.AdminForcedPasswordReset)
	oldUserRawPwd := userRawPwd
	userRawPwd = newRawPwd

	// perform a required password change again, this time it fails as there is no request pending
	perfPwdResetResp = performRequiredPasswordResetResponse{}
	newRawPwd = "new_password2!"
	s.DoJSON("POST", "/api/latest/fleet/perform_required_password_reset", performRequiredPasswordResetRequest{
		Password: newRawPwd,
		ID:       u.ID,
	}, http.StatusInternalServerError, &perfPwdResetResp) // TODO: should be 40?, see #4406
	s.token = s.getTestAdminToken()

	// login as that user to verify that the new password is active (userRawPwd was updated to the new pwd)
	loginResp = loginResponse{}
	s.DoJSON("POST", "/api/latest/fleet/login", loginRequest{Email: u.Email, Password: userRawPwd}, http.StatusOK, &loginResp)
	require.Equal(t, loginResp.User.ID, u.ID)

	// logout for that user
	s.token = loginResp.Token
	var logoutResp logoutResponse
	s.DoJSON("POST", "/api/latest/fleet/logout", nil, http.StatusOK, &logoutResp)

	// logout again, even though not logged in
	s.DoJSON("POST", "/api/latest/fleet/logout", nil, http.StatusUnauthorized, &logoutResp)

	s.token = s.getTestAdminToken()

	// login as that user with previous pwd fails
	loginResp = loginResponse{}
	s.DoJSON("POST", "/api/latest/fleet/login", loginRequest{Email: u.Email, Password: oldUserRawPwd}, http.StatusUnauthorized, &loginResp)

	// require a password reset
	var reqResetResp requirePasswordResetResponse
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/users/%d/require_password_reset", u.ID), map[string]bool{"require": true}, http.StatusOK, &reqResetResp)
	assert.Equal(t, u.ID, reqResetResp.User.ID)
	assert.True(t, reqResetResp.User.AdminForcedPasswordReset)

	// require a password reset to invalid user
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/users/%d/require_password_reset", u.ID+1), map[string]bool{"require": true}, http.StatusNotFound, &reqResetResp)

	// delete user
	var delResp deleteUserResponse
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/users/%d", u.ID), nil, http.StatusOK, &delResp)

	// delete invalid user
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/users/%d", u.ID), nil, http.StatusNotFound, &delResp)
}

func (s *integrationTestSuite) TestGlobalPoliciesAutomationConfig() {
	t := s.T()

	gpParams := globalPolicyRequest{
		Name:  "policy1",
		Query: "select 41;",
	}
	gpResp := globalPolicyResponse{}
	s.DoJSON("POST", "/api/latest/fleet/policies", gpParams, http.StatusOK, &gpResp)
	require.NotNil(t, gpResp.Policy)

	s.DoRaw("PATCH", "/api/latest/fleet/config", []byte(fmt.Sprintf(`{
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
	s.DoJSON("POST", "/api/latest/fleet/policies/delete", deletePolicyParams, http.StatusOK, &deletePolicyResp)

	config = s.getConfig()
	require.Empty(t, config.WebhookSettings.FailingPoliciesWebhook.PolicyIDs)
}

func (s *integrationTestSuite) TestHostStatusWebhookConfig() {
	t := s.T()

	// enable with valid config
	s.DoRaw("PATCH", "/api/latest/fleet/config", []byte(`{
		"webhook_settings": {
    		"host_status_webhook": {
     	 		"enable_host_status_webhook": true,
     	 		"destination_url": "http://some/url",
				  "host_percentage": 2,
					"days_count": 1
    		},
    		"interval": "1h"
  		}
	}`), http.StatusOK)

	config := s.getConfig()
	require.True(t, config.WebhookSettings.HostStatusWebhook.Enable)
	require.Equal(t, "http://some/url", config.WebhookSettings.HostStatusWebhook.DestinationURL)
	require.Equal(t, 2.0, config.WebhookSettings.HostStatusWebhook.HostPercentage)
	require.Equal(t, 1, config.WebhookSettings.HostStatusWebhook.DaysCount)

	// update without a destination url
	s.DoRaw("PATCH", "/api/latest/fleet/config", []byte(`{
		"webhook_settings": {
    		"host_status_webhook": {
     	 		"enable_host_status_webhook": true,
     	 		"destination_url": "",
				  "host_percentage": 2,
					"days_count": 1
    		},
    		"interval": "1h"
  		}
	}`), http.StatusUnprocessableEntity)

	// update without a negative days count
	s.DoRaw("PATCH", "/api/latest/fleet/config", []byte(`{
		"webhook_settings": {
    		"host_status_webhook": {
     	 		"enable_host_status_webhook": true,
					"destination_url": "http://other/url",
				  "host_percentage": 2,
					"days_count": -123
    		},
    		"interval": "1h"
  		}
	}`), http.StatusUnprocessableEntity)

	// update with 0%
	s.DoRaw("PATCH", "/api/latest/fleet/config", []byte(`{
		"webhook_settings": {
    		"host_status_webhook": {
     	 		"enable_host_status_webhook": true,
					"destination_url": "http://other/url",
				  "host_percentage": 0,
					"days_count": 12
    		},
    		"interval": "1h"
  		}
	}`), http.StatusUnprocessableEntity)

	// config left unmodified since last successful call
	config = s.getConfig()
	require.True(t, config.WebhookSettings.HostStatusWebhook.Enable)
	require.Equal(t, "http://some/url", config.WebhookSettings.HostStatusWebhook.DestinationURL)
	require.Equal(t, 2.0, config.WebhookSettings.HostStatusWebhook.HostPercentage)
	require.Equal(t, 1, config.WebhookSettings.HostStatusWebhook.DaysCount)

	// disabling ignores the invalid parameters
	s.DoRaw("PATCH", "/api/latest/fleet/config", []byte(`{
		"webhook_settings": {
    		"host_status_webhook": {
     	 		"enable_host_status_webhook": false,
     	 		"destination_url": "",
				  "host_percentage": 0
    		},
    		"interval": "1h"
  		}
	}`), http.StatusOK)

	config = s.getConfig()
	require.False(t, config.WebhookSettings.HostStatusWebhook.Enable)
}

func (s *integrationTestSuite) TestVulnerabilitiesWebhookConfig() {
	t := s.T()

	s.DoRaw("PATCH", "/api/latest/fleet/config", []byte(`{
		"integrations": {"jira": [], "zendesk": []},
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

func (s *integrationTestSuite) TestExternalIntegrationsConfig() {
	t := s.T()

	// create a test http server to act as the Jira and Zendesk server
	srvURL := startExternalServiceWebServer(t)

	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"jira": [{
				"url": %q,
				"username": "ok",
				"api_token": "bar",
				"project_key": "qux",
				"enable_software_vulnerabilities": true
			}]
		}
	}`, srvURL)), http.StatusOK)

	config := s.getConfig()
	require.Len(t, config.Integrations.Jira, 1)
	require.Equal(t, srvURL, config.Integrations.Jira[0].URL)
	require.Equal(t, "ok", config.Integrations.Jira[0].Username)
	require.Equal(t, fleet.MaskedPassword, config.Integrations.Jira[0].APIToken)
	require.Equal(t, "qux", config.Integrations.Jira[0].ProjectKey)
	require.True(t, config.Integrations.Jira[0].EnableSoftwareVulnerabilities)

	// add a second, disabled Jira integration
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"jira": [
				{
					"url": %q,
					"username": "ok",
					"api_token": "bar",
					"project_key": "qux",
					"enable_software_vulnerabilities": true
				},
				{
					"url": %[1]q,
					"username": "ok",
					"api_token": "bar",
					"project_key": "qux2",
					"enable_software_vulnerabilities": false
				}
			]
		}
	}`, srvURL)), http.StatusOK)

	config = s.getConfig()
	require.Len(t, config.Integrations.Jira, 2)

	// first integration
	require.Equal(t, srvURL, config.Integrations.Jira[0].URL)
	require.Equal(t, "ok", config.Integrations.Jira[0].Username)
	require.Equal(t, fleet.MaskedPassword, config.Integrations.Jira[0].APIToken)
	require.Equal(t, "qux", config.Integrations.Jira[0].ProjectKey)
	require.True(t, config.Integrations.Jira[0].EnableSoftwareVulnerabilities)

	// second integration
	require.Equal(t, srvURL, config.Integrations.Jira[1].URL)
	require.Equal(t, "ok", config.Integrations.Jira[1].Username)
	require.Equal(t, fleet.MaskedPassword, config.Integrations.Jira[1].APIToken)
	require.Equal(t, "qux2", config.Integrations.Jira[1].ProjectKey)
	require.False(t, config.Integrations.Jira[1].EnableSoftwareVulnerabilities)

	// delete first Jira integration
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"jira": [
				{
					"url": %q,
					"username": "ok",
					"project_key": "qux2",
					"enable_software_vulnerabilities": false
				}
			]
		}
	}`, srvURL)), http.StatusOK)

	config = s.getConfig()
	require.Len(t, config.Integrations.Jira, 1)
	require.Equal(t, "qux2", config.Integrations.Jira[0].ProjectKey)

	// replace Jira integration
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"jira": [
				{
					"url": %q,
					"username": "ok",
					"api_token": "ok",
					"project_key": "qux",
					"enable_software_vulnerabilities": false
				}
			]
		}
	}`, srvURL)), http.StatusOK)

	config = s.getConfig()
	require.Len(t, config.Integrations.Jira, 1)
	require.Equal(t, "qux", config.Integrations.Jira[0].ProjectKey)
	require.False(t, config.Integrations.Jira[0].EnableSoftwareVulnerabilities)

	// try adding Jira integration without sending API token
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"jira": [
				{
					"url": %q,
					"username": "ok",
					"api_token": "ok",
					"project_key": "qux",
					"enable_software_vulnerabilities": true
				},
				{
					"url": %[1]q,
					"username": "ok",
					"project_key": "qux2",
					"enable_software_vulnerabilities": false
				}
			]
		}
	}`, srvURL)), http.StatusBadRequest)

	// try adding Jira integration with masked API token
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"jira": [
				{
					"url": %q,
					"username": "ok",
					"api_token": "ok",
					"project_key": "qux",
					"enable_software_vulnerabilities": true
				},
				{
					"url": %[1]q,
					"username": "ok",
					"api_token": %q,
					"project_key": "qux2",
					"enable_software_vulnerabilities": false
				}
			]
		}
	}`, srvURL, fleet.MaskedPassword)), http.StatusBadRequest)

	// edit Jira integration without sending API token
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"jira": [
				{
					"url": %q,
					"username": "ok",
					"project_key": "qux",
					"enable_software_vulnerabilities": true
				}
			]
		}
	}`, srvURL)), http.StatusOK)

	config = s.getConfig()
	require.Len(t, config.Integrations.Jira, 1)
	require.Equal(t, "qux", config.Integrations.Jira[0].ProjectKey)
	require.True(t, config.Integrations.Jira[0].EnableSoftwareVulnerabilities)

	// edit Jira integration with masked API token
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"jira": [
				{
					"url": %q,
					"username": "ok",
					"api_token": %q,
					"project_key": "qux",
					"enable_software_vulnerabilities": false
				}
			]
		}
	}`, srvURL, fleet.MaskedPassword)), http.StatusOK)

	config = s.getConfig()
	require.Len(t, config.Integrations.Jira, 1)
	require.Equal(t, "qux", config.Integrations.Jira[0].ProjectKey)
	require.False(t, config.Integrations.Jira[0].EnableSoftwareVulnerabilities)

	// edit Jira integration sending explicit "" as API token
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"jira": [
				{
					"url": %q,
					"username": "ok",
					"api_token": "",
					"project_key": "qux",
					"enable_software_vulnerabilities": true
				}
			]
		}
	}`, srvURL)), http.StatusOK)

	config = s.getConfig()
	require.Len(t, config.Integrations.Jira, 1)
	require.Equal(t, "qux", config.Integrations.Jira[0].ProjectKey)
	require.True(t, config.Integrations.Jira[0].EnableSoftwareVulnerabilities)

	// unknown fields fails as bad request
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"jira": [{
				"url": %q,
				"UNKNOWN_FIELD": "foo"
			}]
		}
	}`, srvURL)), http.StatusBadRequest)

	// unknown project key fails as bad request
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"jira": [
				{
					"url": %q,
					"username": "ok",
					"api_token": %q,
					"project_key": "qux3",
					"enable_software_vulnerabilities": true
				}
			]
		}
	}`, srvURL, fleet.MaskedPassword)), http.StatusBadRequest)

	// cannot have two integrations enabled at the same time
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"jira": [
				{
					"url": %q,
					"username": "ok",
					"api_token": "bar",
					"project_key": "qux",
					"enable_software_vulnerabilities": true
				},
				{
					"url": %[1]q,
					"username": "ok",
					"api_token": "bar2",
					"project_key": "qux2",
					"enable_software_vulnerabilities": true
				}
			]
		}
	}`, srvURL)), http.StatusUnprocessableEntity)

	// cannot have two jira integrations with the same project key
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"jira": [
				{
					"url": %q,
					"username": "ok",
					"api_token": "bar",
					"project_key": "qux",
					"enable_software_vulnerabilities": true
				},
				{
					"url": %[1]q,
					"username": "ok",
					"api_token": "bar2",
					"project_key": "qux",
					"enable_software_vulnerabilities": false
				}
			]
		}
	}`, srvURL)), http.StatusUnprocessableEntity)

	// even disabled integrations are tested for Jira connection and credentials,
	// so this fails because the 2nd one uses the "fail" username.
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"jira": [
				{
					"url": %q,
					"username": "ok",
					"api_token": "bar",
					"project_key": "qux",
					"enable_software_vulnerabilities": true
				},
				{
					"url": %[1]q,
					"username": "fail",
					"api_token": "bar2",
					"project_key": "qux2",
					"enable_software_vulnerabilities": false
				}
			]
		}
	}`, srvURL)), http.StatusBadRequest)

	// cannot enable webhook with a jira integration already enabled
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(`{
		"webhook_settings": {
			"vulnerabilities_webhook": {
				"enable_vulnerabilities_webhook": true,
				"destination_url": "http://some/url",
				"host_batch_size": 1234
			},
			"interval": "1h"
		}
	}`), http.StatusUnprocessableEntity)

	// disable jira, now we can enable webhook
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
		"jira": [{
			"url": %q,
			"username": "ok",
			"api_token": "bar",
			"project_key": "qux",
			"enable_software_vulnerabilities": false
		}]
		},
		"webhook_settings": {
			"vulnerabilities_webhook": {
				"enable_vulnerabilities_webhook": true,
				"destination_url": "http://some/url",
				"host_batch_size": 1234
			},
			"interval": "1h"
		}
	}`, srvURL)), http.StatusOK)

	// cannot enable jira with webhook already enabled
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"jira": [{
				"url": %q,
				"username": "ok",
				"api_token": "bar",
				"project_key": "qux",
				"enable_software_vulnerabilities": true
			}]
		}
	}`, srvURL)), http.StatusUnprocessableEntity)

	// disable webhook, enable jira with wrong credentials
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"jira": [{
				"url": %q,
				"username": "fail",
				"api_token": "bar",
				"project_key": "qux",
				"enable_software_vulnerabilities": true
			}]
		},
		"webhook_settings": {
			"vulnerabilities_webhook": {
				"enable_vulnerabilities_webhook": false,
				"destination_url": "http://some/url",
				"host_batch_size": 1234
			},
			"interval": "1h"
		}
	}`, srvURL)), http.StatusBadRequest)

	// update jira config to correct credentials (need to disable webhook too as
	// last request failed)
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"jira": [{
				"url": %q,
				"username": "ok",
				"api_token": "bar",
				"project_key": "qux",
				"enable_software_vulnerabilities": true
			}]
		},
		"webhook_settings": {
			"vulnerabilities_webhook": {
				"enable_vulnerabilities_webhook": false,
				"destination_url": "http://some/url",
				"host_batch_size": 1234
			},
			"interval": "1h"
		}
	}`, srvURL)), http.StatusOK)

	// remove all integrations
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(`{
		"integrations": {
			"jira": [],
			"zendesk": []
		}
	}`), http.StatusOK)

	// set environmental varible to use Zendesk test client
	t.Setenv("TEST_ZENDESK_CLIENT", "true")
	// create zendesk integration
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"zendesk": [{
				"url": %q,
				"email": "ok@example.com",
				"api_token": "ok",
				"group_id": 122,
				"enable_software_vulnerabilities": true
			}]
		}
	}`, srvURL)), http.StatusOK)

	config = s.getConfig()
	require.Len(t, config.Integrations.Jira, 0)
	require.Len(t, config.Integrations.Zendesk, 1)
	require.Equal(t, srvURL, config.Integrations.Zendesk[0].URL)
	require.Equal(t, "ok@example.com", config.Integrations.Zendesk[0].Email)
	require.Equal(t, fleet.MaskedPassword, config.Integrations.Zendesk[0].APIToken)
	require.Equal(t, int64(122), config.Integrations.Zendesk[0].GroupID)
	require.True(t, config.Integrations.Zendesk[0].EnableSoftwareVulnerabilities)

	// add a second, disabled Zendesk integration
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"zendesk": [
				{
					"url": %q,
					"email": "ok@example.com",
					"api_token": "ok",
					"group_id": 122,
					"enable_software_vulnerabilities": true
				},
				{
					"url": %[1]q,
					"email": "test123@example.com",
					"api_token": "ok",
					"group_id": 123,
					"enable_software_vulnerabilities": false
				}
			]
		}
	}`, srvURL)), http.StatusOK)

	config = s.getConfig()
	require.Len(t, config.Integrations.Jira, 0)
	require.Len(t, config.Integrations.Zendesk, 2)

	// first integration
	require.Equal(t, srvURL, config.Integrations.Zendesk[0].URL)
	require.Equal(t, "ok@example.com", config.Integrations.Zendesk[0].Email)
	require.Equal(t, fleet.MaskedPassword, config.Integrations.Zendesk[0].APIToken)
	require.Equal(t, int64(122), config.Integrations.Zendesk[0].GroupID)
	require.True(t, config.Integrations.Zendesk[0].EnableSoftwareVulnerabilities)

	// second integration
	require.Equal(t, srvURL, config.Integrations.Zendesk[1].URL)
	require.Equal(t, "test123@example.com", config.Integrations.Zendesk[1].Email)
	require.Equal(t, fleet.MaskedPassword, config.Integrations.Zendesk[1].APIToken)
	require.Equal(t, int64(123), config.Integrations.Zendesk[1].GroupID)
	require.False(t, config.Integrations.Zendesk[1].EnableSoftwareVulnerabilities)

	// delete first Zendesk integration
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"zendesk": [
				{
					"url": %q,
					"email": "test123@example.com",
					"group_id": 123,
					"enable_software_vulnerabilities": false
				}
			]
		}
	}`, srvURL)), http.StatusOK)

	config = s.getConfig()
	require.Len(t, config.Integrations.Jira, 0)
	require.Len(t, config.Integrations.Zendesk, 1)
	require.Equal(t, int64(123), config.Integrations.Zendesk[0].GroupID)

	// replace Zendesk integration
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"zendesk": [
				{
					"url": %q,
					"email": "ok@example.com",
					"api_token": "ok",
					"group_id": 122,
					"enable_software_vulnerabilities": false
				}
			]
		}
	}`, srvURL)), http.StatusOK)

	config = s.getConfig()
	require.Len(t, config.Integrations.Jira, 0)
	require.Len(t, config.Integrations.Zendesk, 1)
	require.Equal(t, int64(122), config.Integrations.Zendesk[0].GroupID)
	require.False(t, config.Integrations.Zendesk[0].EnableSoftwareVulnerabilities)

	// try adding Zendesk integration without sending API token
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"zendesk": [
				{
					"url": %q,
					"email": "ok@example.com",
					"api_token": "ok",
					"group_id": 122,
					"enable_software_vulnerabilities": true
				},
				{
					"url": %[1]q,
					"email": "test123@example.com",
					"group_id": 123,
					"enable_software_vulnerabilities": false
				}
			]
		}
	}`, srvURL)), http.StatusBadRequest)

	// try adding Zendesk integration with masked API token
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"zendesk": [
				{
					"url": %q,
					"email": "ok@example.com",
					"api_token": "ok",
					"group_id": 122,
					"enable_software_vulnerabilities": true
				},
				{
					"url": %[1]q,
					"email": "test123@example.com",
					"api_token": %q,
					"group_id": 123,
					"enable_software_vulnerabilities": false
				}
			]
		}
	}`, srvURL, fleet.MaskedPassword)), http.StatusBadRequest)

	// edit Zendesk integration without sending API token
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"zendesk": [
				{
					"url": %q,
					"email": "ok@example.com",
					"group_id": 122,
					"enable_software_vulnerabilities": true
				}
			]
		}
	}`, srvURL)), http.StatusOK)

	config = s.getConfig()
	require.Len(t, config.Integrations.Jira, 0)
	require.Len(t, config.Integrations.Zendesk, 1)
	require.Equal(t, int64(122), config.Integrations.Zendesk[0].GroupID)
	require.True(t, config.Integrations.Zendesk[0].EnableSoftwareVulnerabilities)

	// edit Zendesk integration with masked API token
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"zendesk": [
				{
					"url": %q,
					"email": "ok@example.com",
					"api_token": %q,
					"group_id": 122,
					"enable_software_vulnerabilities": false
				}
			]
		}
	}`, srvURL, fleet.MaskedPassword)), http.StatusOK)

	config = s.getConfig()
	require.Len(t, config.Integrations.Jira, 0)
	require.Len(t, config.Integrations.Zendesk, 1)
	require.Equal(t, int64(122), config.Integrations.Zendesk[0].GroupID)
	require.False(t, config.Integrations.Zendesk[0].EnableSoftwareVulnerabilities)

	// edit Zendesk integration with explicit "" API token
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"zendesk": [
				{
					"url": %q,
					"email": "ok@example.com",
					"api_token": "",
					"group_id": 122,
					"enable_software_vulnerabilities": true
				}
			]
		}
	}`, srvURL)), http.StatusOK)

	config = s.getConfig()
	require.Len(t, config.Integrations.Jira, 0)
	require.Len(t, config.Integrations.Zendesk, 1)
	require.Equal(t, int64(122), config.Integrations.Zendesk[0].GroupID)
	require.True(t, config.Integrations.Zendesk[0].EnableSoftwareVulnerabilities)

	// unknown fields fails as bad request
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"zendesk": [{
				"url": %q,
				"UNKNOWN_FIELD": "foo"
			}]
		}
	}`, srvURL)), http.StatusBadRequest)

	// unknown group id fails as bad request
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"zendesk": [{
				"url": %q,
				"email": "ok@example.com",
				"api_token": "ok",
				"group_id": 999,
				"enable_software_vulnerabilities": true
			}]
		}
	}`, srvURL)), http.StatusBadRequest)

	// cannot have two zendesk integrations enabled at the same time
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"zendesk": [
				{
					"url": %q,
					"email": "ok@example.com",
					"api_token": "ok",
					"group_id": 122,
					"enable_software_vulnerabilities": true
				},
				{
					"url": %[1]q,
					"email": "not.ok@example.com",
					"api_token": "ok",
					"group_id": 123,
					"enable_software_vulnerabilities": true
				}
			]
		}
	}`, srvURL)), http.StatusUnprocessableEntity)

	// cannot have two zendesk integrations with the same group id
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"zendesk": [
				{
					"url": %q,
					"email": "ok@example.com",
					"api_token": "ok",
					"group_id": 122,
					"enable_software_vulnerabilities": true
				},
				{
					"url": %[1]q,
					"email": "not.ok@example.com",
					"api_token": "ok",
					"group_id": 122,
					"enable_software_vulnerabilities": false
				}
			]
		}
	}`, srvURL)), http.StatusUnprocessableEntity)

	// even disabled integrations are tested for Zendesk connection and credentials,
	// so this fails because the 2nd one uses the "fail" token.
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"zendesk": [
				{
					"url": %q,
					"email": "ok@example.com",
					"api_token": "ok",
					"group_id": 122,
					"enable_software_vulnerabilities": true
				},
				{
					"url": %[1]q,
					"email": "not.ok@example.com",
					"api_token": "fail",
					"group_id": 123,
					"enable_software_vulnerabilities": false
				}
			]
		}
	}`, srvURL)), http.StatusBadRequest)

	// cannot enable webhook with a zendesk integration already enabled
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(`{
		"webhook_settings": {
			"vulnerabilities_webhook": {
				"enable_vulnerabilities_webhook": true,
				"destination_url": "http://some/url",
				"host_batch_size": 1234
			},
			"interval": "1h"
		}
	}`), http.StatusUnprocessableEntity)

	// disable zendesk, now we can enable webhook
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"zendesk": [{
				"url": %q,
				"email": "ok@example.com",
				"api_token": "ok",
				"group_id": 122,
				"enable_software_vulnerabilities": false
			}]
		},
		"webhook_settings": {
			"vulnerabilities_webhook": {
				"enable_vulnerabilities_webhook": true,
				"destination_url": "http://some/url",
				"host_batch_size": 1234
			},
			"interval": "1h"
		}
	}`, srvURL)), http.StatusOK)

	// cannot enable zendesk with webhook already enabled
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"zendesk": [{
				"url": %q,
				"email": "ok@example.com",
				"api_token": "ok",
				"group_id": 122,
				"enable_software_vulnerabilities": true
			}]
		}
	}`, srvURL)), http.StatusUnprocessableEntity)

	// disable webhook, enable zendesk with wrong credentials
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"zendesk": [{
				"url": %q,
				"email": "not.ok@example.com",
				"api_token": "fail",
				"group_id": 122,
				"enable_software_vulnerabilities": true
			}]
		},
		"webhook_settings": {
			"vulnerabilities_webhook": {
				"enable_vulnerabilities_webhook": false,
				"destination_url": "http://some/url",
				"host_batch_size": 1234
			},
			"interval": "1h"
		}
	}`, srvURL)), http.StatusBadRequest)

	// update zendesk config to correct credentials (need to disable webhook too as
	// last request failed)
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"zendesk": [{
				"url": %q,
				"email": "ok@example.com",
				"api_token": "ok",
				"group_id": 122,
				"enable_software_vulnerabilities": true
			}]
		},
		"webhook_settings": {
			"vulnerabilities_webhook": {
				"enable_vulnerabilities_webhook": false,
				"destination_url": "http://some/url",
				"host_batch_size": 1234
			},
			"interval": "1h"
		}
	}`, srvURL)), http.StatusOK)

	// can have jira enabled and zendesk disabled
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"jira": [{
				"url": %q,
				"username": "ok",
				"api_token": "bar",
				"project_key": "qux",
				"enable_software_vulnerabilities": true
			}],
			"zendesk": [{
				"url": %[1]q,
				"email": "ok@example.com",
				"api_token": "ok",
				"group_id": 122,
				"enable_software_vulnerabilities": false
			}]
		}
	}`, srvURL)), http.StatusOK)
	config = s.getConfig()
	require.Len(t, config.Integrations.Jira, 1)
	require.True(t, config.Integrations.Jira[0].EnableSoftwareVulnerabilities)
	require.Len(t, config.Integrations.Zendesk, 1)
	require.False(t, config.Integrations.Zendesk[0].EnableSoftwareVulnerabilities)

	// can have jira disabled and zendesk enabled
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"jira": [{
				"url": %q,
				"username": "ok",
				"api_token": "bar",
				"project_key": "qux",
				"enable_software_vulnerabilities": false
			}],
			"zendesk": [{
				"url": %[1]q,
				"email": "ok@example.com",
				"api_token": "ok",
				"group_id": 122,
				"enable_software_vulnerabilities": true
			}]
		}
	}`, srvURL)), http.StatusOK)
	config = s.getConfig()
	require.Len(t, config.Integrations.Jira, 1)
	require.False(t, config.Integrations.Jira[0].EnableSoftwareVulnerabilities)
	require.Len(t, config.Integrations.Zendesk, 1)
	require.True(t, config.Integrations.Zendesk[0].EnableSoftwareVulnerabilities)

	// cannot have both jira enabled and zendesk enabled
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"jira": [{
				"url": %q,
				"username": "ok",
				"api_token": "bar",
				"project_key": "qux",
				"enable_software_vulnerabilities": true
			}],
			"zendesk": [{
				"url": %[1]q,
				"email": "ok@example.com",
				"api_token": "ok",
				"group_id": 122,
				"enable_software_vulnerabilities": true
			}]
		}
	}`, srvURL)), http.StatusUnprocessableEntity)

	// remove all integrations on exit, so that other tests can enable the
	// webhook as needed
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(`{
		"integrations": {
		"jira": [],
		"zendesk": []
		}
	}`), http.StatusOK)
	config = s.getConfig()
	require.Len(t, config.Integrations.Jira, 0)
	require.Len(t, config.Integrations.Zendesk, 0)
}

func (s *integrationTestSuite) TestQueriesBadRequests() {
	t := s.T()

	reqQuery := &fleet.QueryPayload{
		Name:  ptr.String("existing query"),
		Query: ptr.String("select 42;"),
	}
	createQueryResp := createQueryResponse{}
	s.DoJSON("POST", "/api/latest/fleet/queries", reqQuery, http.StatusOK, &createQueryResp)
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
			query: "",
		},
	} {
		t.Run(tc.tname, func(t *testing.T) {
			reqQuery := &fleet.QueryPayload{
				Name:  ptr.String(tc.name),
				Query: ptr.String(tc.query),
			}
			createQueryResp := createQueryResponse{}
			s.DoJSON("POST", "/api/latest/fleet/queries", reqQuery, http.StatusBadRequest, &createQueryResp)
			require.Nil(t, createQueryResp.Query)

			payload := fleet.QueryPayload{
				Name:  ptr.String(tc.name),
				Query: ptr.String(tc.query),
			}
			mResp := modifyQueryResponse{}
			s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/queries/%d", existingQueryID), &payload, http.StatusBadRequest, &mResp)
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
	s.DoJSON("POST", "/api/latest/fleet/packs", reqPacks, http.StatusOK, &createPackResp)
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
			s.DoJSON("POST", "/api/latest/fleet/packs", reqQuery, http.StatusBadRequest, &createPackResp)

			payload := fleet.PackPayload{
				Name: ptr.String(tc.name),
			}
			mResp := modifyPackResponse{}
			s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/packs/%d", existingPackID), &payload, http.StatusBadRequest, &mResp)
		})
	}
}

func (s *integrationTestSuite) TestPremiumEndpointsWithoutLicense() {
	t := s.T()

	// list teams, none
	var listResp listTeamsResponse
	s.DoJSON("GET", "/api/latest/fleet/teams", nil, http.StatusPaymentRequired, &listResp)
	assert.Len(t, listResp.Teams, 0)

	// get team
	var getResp getTeamResponse
	s.DoJSON("GET", "/api/latest/fleet/teams/123", nil, http.StatusPaymentRequired, &getResp)
	assert.Nil(t, getResp.Team)

	// create team
	var tmResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", &createTeamRequest{}, http.StatusPaymentRequired, &tmResp)
	assert.Nil(t, tmResp.Team)

	// modify team
	s.DoJSON("PATCH", "/api/latest/fleet/teams/123", fleet.TeamPayload{}, http.StatusPaymentRequired, &tmResp)
	assert.Nil(t, tmResp.Team)

	// delete team
	var delResp deleteTeamResponse
	s.DoJSON("DELETE", "/api/latest/fleet/teams/123", nil, http.StatusPaymentRequired, &delResp)

	// apply team specs
	var specResp applyTeamSpecsResponse
	teamSpecs := applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{Name: "newteam", Secrets: []fleet.EnrollSecret{{Secret: "ABC"}}}}}
	s.DoJSON("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusPaymentRequired, &specResp)

	// modify team agent options
	s.DoJSON("POST", "/api/latest/fleet/teams/123/agent_options", nil, http.StatusPaymentRequired, &tmResp)
	assert.Nil(t, tmResp.Team)

	// list team users
	var usersResp listUsersResponse
	s.DoJSON("GET", "/api/latest/fleet/teams/123/users", nil, http.StatusPaymentRequired, &usersResp, "page", "1")
	assert.Len(t, usersResp.Users, 0)

	// add team users
	s.DoJSON("PATCH", "/api/latest/fleet/teams/123/users", modifyTeamUsersRequest{Users: []fleet.TeamUser{{User: fleet.User{ID: 1}}}}, http.StatusPaymentRequired, &tmResp)
	assert.Nil(t, tmResp.Team)

	// delete team users
	s.DoJSON("DELETE", "/api/latest/fleet/teams/123/users", modifyTeamUsersRequest{Users: []fleet.TeamUser{{User: fleet.User{ID: 1}}}}, http.StatusPaymentRequired, &tmResp)
	assert.Nil(t, tmResp.Team)

	// get team enroll secrets
	var secResp teamEnrollSecretsResponse
	s.DoJSON("GET", "/api/latest/fleet/teams/123/secrets", nil, http.StatusPaymentRequired, &secResp)
	assert.Len(t, secResp.Secrets, 0)

	// modify team enroll secrets
	s.DoJSON("PATCH", "/api/latest/fleet/teams/123/secrets", modifyTeamEnrollSecretsRequest{Secrets: []fleet.EnrollSecret{{Secret: "DEF"}}}, http.StatusPaymentRequired, &secResp)
	assert.Len(t, secResp.Secrets, 0)

	// get apple BM configuration
	var appleBMResp getAppleBMResponse
	s.DoJSON("GET", "/api/latest/fleet/mdm/apple_bm", nil, http.StatusPaymentRequired, &appleBMResp)
	assert.Nil(t, appleBMResp.AppleBM)

	// batch-apply an empty set of MDM profiles succeeds even though MDM is not
	// enabled, because it wouldn't change anything (and it needs to support the
	// case where `fleetctl get config`'s output is used as input to `fleetctl
	// apply`).
	s.Do("POST", "/api/latest/fleet/mdm/apple/profiles/batch", nil, http.StatusNoContent)

	// batch-apply a non-empty set of MDM profiles fails
	res := s.Do("POST", "/api/latest/fleet/mdm/apple/profiles/batch",
		map[string]interface{}{"profiles": [][]byte{[]byte(`xyz`)}}, http.StatusUnprocessableEntity)
	errMsg := extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Fleet MDM is not configured")

	// update MDM settings, the endpoint returns an error if MDM is not enabled
	res = s.Do("PATCH", "/api/latest/fleet/mdm/apple/settings", fleet.MDMAppleSettingsPayload{}, fleet.ErrMDMNotConfigured.StatusCode())
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, fleet.ErrMDMNotConfigured.Error())

	// device migrate mdm endpoint returns an error if not premium
	createHostAndDeviceToken(t, s.ds, "some-token")
	s.Do("POST", fmt.Sprintf("/api/v1/fleet/device/%s/migrate_mdm", "some-token"), nil, http.StatusPaymentRequired)
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
	s.DoJSON("POST", "/api/latest/fleet/policies", gpParams0, http.StatusOK, &gpResp0)
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
	password := test.GoodPassword
	require.NoError(t, teamObserver.SetPassword(password, 10, 10))
	_, err = s.ds.NewUser(context.Background(), teamObserver)
	require.NoError(t, err)

	oldToken := s.token
	s.token = s.getTestToken(email, password)
	t.Cleanup(func() {
		s.token = oldToken
	})

	policiesResponse := listGlobalPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/policies", nil, http.StatusOK, &policiesResponse)
	require.Len(t, policiesResponse.Policies, 1)
	assert.Equal(t, "global policy", policiesResponse.Policies[0].Name)
	assert.Equal(t, "select * from osquery;", policiesResponse.Policies[0].Query)
}

func (s *integrationTestSuite) TestTeamPoliciesTeamNotExists() {
	t := s.T()

	teamPoliciesResponse := listTeamPoliciesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", 9999999), nil, http.StatusNotFound, &teamPoliciesResponse)
	require.Len(t, teamPoliciesResponse.Policies, 0)
}

func (s *integrationTestSuite) TestSessionInfo() {
	t := s.T()

	ssn := createSession(t, 1, s.ds)

	var meResp getUserResponse
	resp := s.DoRawWithHeaders("GET", "/api/latest/fleet/me", nil, http.StatusOK, map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", ssn.Key),
	})
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&meResp))
	assert.Equal(t, uint(1), meResp.User.ID)

	// get info about session
	var getResp getInfoAboutSessionResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/sessions/%d", ssn.ID), nil, http.StatusOK, &getResp)
	assert.Equal(t, ssn.ID, getResp.SessionID)
	assert.Equal(t, uint(1), getResp.UserID)

	// get info about session - non-existing: appears to deliberately return 500 due to forbidden,
	// which takes precedence vs the not found returned by the datastore (it still shouldn't be a
	// 500 though).
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/sessions/%d", ssn.ID+1), nil, http.StatusInternalServerError, &getResp)

	// delete session
	var delResp deleteSessionResponse
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/sessions/%d", ssn.ID), nil, http.StatusOK, &delResp)

	// delete session - non-existing: again, 500 due to forbidden instead of 404.
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/sessions/%d", ssn.ID), nil, http.StatusInternalServerError, &delResp)
}

func (s *integrationTestSuite) TestAppConfig() {
	t := s.T()
	ctx := context.Background()

	// get the app config
	var acResp appConfigResponse
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	assert.Equal(t, "free", acResp.License.Tier)
	assert.Equal(t, "FleetTest", acResp.OrgInfo.OrgName) // set in SetupSuite
	assert.False(t, acResp.MDM.AppleBMTermsExpired)
	assert.False(t, acResp.MDMEnabled)

	// set the apple BM terms expired flag, and the enabled and configured flags,
	// we'll check again at the end of this test to make sure they weren't
	// modified by any PATCH request (it cannot be set via this endpoint).
	appCfg, err := s.ds.AppConfig(ctx)
	require.NoError(t, err)
	appCfg.MDM.AppleBMTermsExpired = true
	appCfg.MDM.AppleBMEnabledAndConfigured = true
	appCfg.MDM.EnabledAndConfigured = true
	err = s.ds.SaveAppConfig(ctx, appCfg)
	require.NoError(t, err)

	acResp = appConfigResponse{}
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	assert.True(t, acResp.MDM.AppleBMTermsExpired)
	assert.True(t, acResp.MDM.AppleBMEnabledAndConfigured)
	assert.True(t, acResp.MDM.EnabledAndConfigured)

	// no server settings set for the URL, so not possible to test the
	// certificate endpoint
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
    "org_info": {
        "org_name": "test"
    }
  }`), http.StatusOK, &acResp)
	assert.Equal(t, "test", acResp.OrgInfo.OrgName)
	assert.True(t, acResp.MDM.AppleBMTermsExpired)
	assert.False(t, acResp.MDMEnabled)

	// the global agent options were not modified by the last call, so the
	// corresponding activity should not have been created.
	var listActivities listActivitiesResponse
	s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &listActivities, "order_key", "id", "order_direction", "desc")
	if len(listActivities.Activities) > 1 {
		// if there is an activity, make sure it is not edited_agent_options
		require.NotEqual(t, fleet.ActivityTypeEditedAgentOptions{}.ActivityName(), listActivities.Activities[0].Type)
	}

	// and it did not update the appconfig
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	require.Contains(t, string(*acResp.AgentOptions), `"logger_plugin": "tls"`) // default agent options has this setting

	// test a change that does clear the agent options (the field is provided but empty).
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"agent_options": {}
  }`), http.StatusOK, &acResp)
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	require.Equal(t, string(*acResp.AgentOptions), "{}")
	assert.True(t, acResp.MDM.AppleBMTermsExpired)

	// test a change that does modify the agent options.
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"agent_options": { "config": {"views": {"foo": "bar"}} }
  }`), http.StatusOK, &acResp)
	s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &listActivities, "order_key", "id", "order_direction", "desc")
	require.True(t, len(listActivities.Activities) > 1)
	require.Equal(t, fleet.ActivityTypeEditedAgentOptions{}.ActivityName(), listActivities.Activities[0].Type)
	require.NotNil(t, listActivities.Activities[0].Details)
	assert.JSONEq(t, `{"global": true, "team_id": null, "team_name": null}`, string(*listActivities.Activities[0].Details))

	// try to set invalid agent options
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"agent_options": { "config": {"nope": true} }
  }`), http.StatusBadRequest, &acResp)
	// did not update the appconfig
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	require.NotContains(t, string(*acResp.AgentOptions), `"nope"`)

	// try to set an invalid agent options logger_tls_endpoint (must start with "/")
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"agent_options": { "config": {"options": {"logger_tls_endpoint": "not-a-rooted-path"}} }
  }`), http.StatusBadRequest, &acResp)
	// did not update the appconfig
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	require.NotContains(t, string(*acResp.AgentOptions), `"not-a-rooted-path"`)

	// try to set a valid agent options logger_tls_endpoint
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"agent_options": { "config": {"options": {"logger_tls_endpoint": "/rooted-path"}} }
  }`), http.StatusOK, &acResp)
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	require.Contains(t, string(*acResp.AgentOptions), `"/rooted-path"`)

	// force-set invalid agent options
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"agent_options": { "config": {"nope": true} }
  }`), http.StatusOK, &acResp, "force", "true")
	require.Contains(t, string(*acResp.AgentOptions), `"nope"`)

	// dry-run valid agent options
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"agent_options": { "config": {"views":{"yep": "ok"}} }
  }`), http.StatusOK, &acResp, "dry_run", "true")
	require.NotContains(t, string(*acResp.AgentOptions), `"yep"`)
	require.Contains(t, string(*acResp.AgentOptions), `"nope"`)

	// dry-run invalid agent options
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"agent_options": { "config": {"invalid": true} }
  }`), http.StatusBadRequest, &acResp, "dry_run", "true")
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	require.NotContains(t, string(*acResp.AgentOptions), `"invalid"`)

	// set valid agent options command-line flag
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"agent_options": { "command_line_flags": {"enable_tables":"table1"}}
  }`), http.StatusOK, &acResp)
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	require.Contains(t, string(*acResp.AgentOptions), `"enable_tables": "table1"`)

	// set invalid agent options command-line flag
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"agent_options": { "command_line_flags": {"no_such_flag":true}}
  }`), http.StatusBadRequest, &acResp)
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	require.Contains(t, string(*acResp.AgentOptions), `"enable_tables": "table1"`)
	require.NotContains(t, string(*acResp.AgentOptions), `"no_such_flag"`)

	// set invalid value for a valid agent options command-line flag
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"agent_options": { "command_line_flags": {"enable_tables":true}}
  }`), http.StatusBadRequest, &acResp)
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	require.Contains(t, string(*acResp.AgentOptions), `"enable_tables": "table1"`)

	// force-set invalid value for a valid agent options command-line flag
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"agent_options": { "command_line_flags": {"enable_tables":true}}
  }`), http.StatusOK, &acResp, "force", "true")
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	require.NotContains(t, string(*acResp.AgentOptions), `"enable_tables": "table1"`)
	require.Contains(t, string(*acResp.AgentOptions), `"enable_tables": true`)

	// dry-run valid appconfig that uses legacy settings (returns error)
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"host_settings": { "additional_queries": {"foo": "bar"} }
  }`), http.StatusBadRequest, &acResp, "dry_run", "true")
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	require.Nil(t, acResp.Features.AdditionalQueries)

	// without dry-run, the valid appconfig that uses legacy settings is accepted
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"host_settings": { "additional_queries": {"foo": "bar"} }
  }`), http.StatusOK, &acResp, "dry_run", "false")
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	require.NotNil(t, acResp.Features.AdditionalQueries)
	require.Contains(t, string(*acResp.Features.AdditionalQueries), `"foo": "bar"`)

	var verResp versionResponse
	s.DoJSON("GET", "/api/latest/fleet/version", nil, http.StatusOK, &verResp)
	assert.NotEmpty(t, verResp.Branch)

	// get enroll secrets, none yet
	var specResp getEnrollSecretSpecResponse
	s.DoJSON("GET", "/api/latest/fleet/spec/enroll_secret", nil, http.StatusOK, &specResp)
	assert.Empty(t, specResp.Spec.Secrets)

	// apply spec, one secret
	var applyResp applyEnrollSecretSpecResponse
	s.DoJSON("POST", "/api/latest/fleet/spec/enroll_secret", applyEnrollSecretSpecRequest{
		Spec: &fleet.EnrollSecretSpec{
			Secrets: []*fleet.EnrollSecret{{Secret: "XYZ"}},
		},
	}, http.StatusOK, &applyResp)

	// apply spec, too many secrets
	s.DoJSON("POST", "/api/latest/fleet/spec/enroll_secret", applyEnrollSecretSpecRequest{
		Spec: &fleet.EnrollSecretSpec{
			Secrets: createEnrollSecrets(t, fleet.MaxEnrollSecretsCount+1),
		},
	}, http.StatusUnprocessableEntity, &applyResp)

	// get enroll secrets, one
	s.DoJSON("GET", "/api/latest/fleet/spec/enroll_secret", nil, http.StatusOK, &specResp)
	require.Len(t, specResp.Spec.Secrets, 1)
	assert.Equal(t, "XYZ", specResp.Spec.Secrets[0].Secret)

	// remove secret just to prevent affecting other tests
	s.DoJSON("POST", "/api/latest/fleet/spec/enroll_secret", applyEnrollSecretSpecRequest{
		Spec: &fleet.EnrollSecretSpec{},
	}, http.StatusOK, &applyResp)

	s.DoJSON("GET", "/api/latest/fleet/spec/enroll_secret", nil, http.StatusOK, &specResp)
	require.Len(t, specResp.Spec.Secrets, 0)

	// try to update the apple bm terms flag via PATCH /config
	// request is ok but modified value is ignored
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "apple_bm_terms_expired": false }
  }`), http.StatusOK, &acResp)
	assert.True(t, acResp.MDM.AppleBMTermsExpired)

	// try to update the mdm configured flags via PATCH /config
	// request is ok but modified value is ignored
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
	  "mdm": { "enabled_and_configured": false, "apple_bm_enabled_and_configured": false }
  }`), http.StatusOK, &acResp)
	assert.True(t, acResp.MDM.EnabledAndConfigured)
	assert.True(t, acResp.MDM.AppleBMEnabledAndConfigured)

	// set the macos disk encryption field, fails due to license
	res := s.Do("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "macos_settings": { "enable_disk_encryption": true } }
  }`), http.StatusUnprocessableEntity)
	errMsg := extractServerErrorText(res.Body)
	assert.Contains(t, errMsg, "missing or invalid license")

	// try to set the apple bm default team, which is premium only
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "apple_bm_default_team": "xyz" }
  }`), http.StatusUnprocessableEntity, &acResp)

	// try to enable Windows MDM, impossible without the feature flag
	// (only set in mdm integrations tests)
	res = s.Do("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "windows_enabled_and_configured": true }
  }`), http.StatusUnprocessableEntity)
	errMsg = extractServerErrorText(res.Body)
	assert.Contains(t, errMsg, "cannot enable Windows MDM without the feature flag explicitly enabled")

	// verify that the Apple BM terms expired flag was never modified
	acResp = appConfigResponse{}
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	assert.True(t, acResp.MDM.AppleBMTermsExpired)

	// set the apple BM terms back to false
	appCfg, err = s.ds.AppConfig(ctx)
	require.NoError(t, err)
	appCfg.MDM.AppleBMTermsExpired = false
	appCfg.MDM.AppleBMEnabledAndConfigured = false
	appCfg.MDM.EnabledAndConfigured = false
	err = s.ds.SaveAppConfig(ctx, appCfg)
	require.NoError(t, err)

	// set the macos custom settings fields, fails due to MDM not configured
	res = s.Do("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
			"mdm": { "macos_settings": { "custom_settings": ["foo", "bar"] } }
	  }`), http.StatusUnprocessableEntity)
	errMsg = extractServerErrorText(res.Body)
	assert.Contains(t, errMsg, "Couldn't update macos_settings because MDM features aren't turned on in Fleet.")

	// test setting the default app config we use for new installs (this check
	// ensures that the default config passes the validation)
	var defAppCfg fleet.AppConfig
	defAppCfg.ApplyDefaultsForNewInstalls()
	// must set org name and server settings
	defAppCfg.OrgInfo.OrgName = acResp.OrgInfo.OrgName
	defAppCfg.ServerSettings.ServerURL = acResp.ServerSettings.ServerURL
	s.DoRaw("PATCH", "/api/latest/fleet/config", jsonMustMarshal(t, defAppCfg), http.StatusOK)
}

// TODO(lucas): Add tests here.
func (s *integrationTestSuite) TestQuerySpecs() {
	t := s.T()

	// list specs, none yet
	var getSpecsResp getQuerySpecsResponse
	s.DoJSON("GET", "/api/latest/fleet/spec/queries", nil, http.StatusOK, &getSpecsResp)
	assert.Len(t, getSpecsResp.Specs, 0)

	// get unknown one
	var getSpecResp getQuerySpecResponse
	s.DoJSON("GET", "/api/latest/fleet/spec/queries/nonesuch", nil, http.StatusNotFound, &getSpecResp)

	// create some queries via apply specs
	q1 := strings.ReplaceAll(t.Name(), "/", "_")
	q2 := q1 + "_2"
	var applyResp applyQuerySpecsResponse
	s.DoJSON("POST", "/api/latest/fleet/spec/queries", applyQuerySpecsRequest{
		Specs: []*fleet.QuerySpec{
			{Name: q1, Query: "SELECT 1"},
			{Name: q2, Query: "SELECT 2"},
		},
	}, http.StatusOK, &applyResp)

	// get the queries back
	var listQryResp listQueriesResponse
	s.DoJSON("GET", "/api/latest/fleet/queries", nil, http.StatusOK, &listQryResp, "order_key", "name")
	require.Len(t, listQryResp.Queries, 2)
	assert.Equal(t, q1, listQryResp.Queries[0].Name)
	assert.Equal(t, q2, listQryResp.Queries[1].Name)
	q1ID, q2ID := listQryResp.Queries[0].ID, listQryResp.Queries[1].ID

	// list specs
	s.DoJSON("GET", "/api/latest/fleet/spec/queries", nil, http.StatusOK, &getSpecsResp)
	require.Len(t, getSpecsResp.Specs, 2)
	names := []string{getSpecsResp.Specs[0].Name, getSpecsResp.Specs[1].Name}
	assert.ElementsMatch(t, []string{q1, q2}, names)

	// get specific spec
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/spec/queries/%s", q1), nil, http.StatusOK, &getSpecResp)
	assert.Equal(t, getSpecResp.Spec.Query, "SELECT 1")

	// apply specs again - create q3 and update q2
	q3 := q1 + "_3"
	s.DoJSON("POST", "/api/latest/fleet/spec/queries", applyQuerySpecsRequest{
		Specs: []*fleet.QuerySpec{
			{Name: q2, Query: "SELECT -2"},
			{Name: q3, Query: "SELECT 3"},
		},
	}, http.StatusOK, &applyResp)

	// list specs - has 3, not 4 (one was an update)
	s.DoJSON("GET", "/api/latest/fleet/spec/queries", nil, http.StatusOK, &getSpecsResp)
	require.Len(t, getSpecsResp.Specs, 3)
	names = []string{getSpecsResp.Specs[0].Name, getSpecsResp.Specs[1].Name, getSpecsResp.Specs[2].Name}
	assert.ElementsMatch(t, []string{q1, q2, q3}, names)

	// get the queries back again
	s.DoJSON("GET", "/api/latest/fleet/queries", nil, http.StatusOK, &listQryResp, "order_key", "name")
	require.Len(t, listQryResp.Queries, 3)
	assert.Equal(t, q1ID, listQryResp.Queries[0].ID)
	assert.Equal(t, q2ID, listQryResp.Queries[1].ID)
	assert.Equal(t, "SELECT -2", listQryResp.Queries[1].Query)
	q3ID := listQryResp.Queries[2].ID

	// delete all queries created
	var delBatchResp deleteQueriesResponse
	s.DoJSON("POST", "/api/latest/fleet/queries/delete", map[string]interface{}{
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
			NodeKey:         ptr.String(t.Name() + strconv.Itoa(i)),
			OsqueryHostID:   ptr.String(t.Name() + strconv.Itoa(i)),
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
		_, err := s.ds.UpdateHostSoftware(context.Background(), h.ID, sws[i:])
		require.NoError(t, err)
		require.NoError(t, s.ds.LoadHostSoftware(context.Background(), h, false))

		if i == 0 {
			// this host has all software, refresh the list so we have the software.ID filled
			sws = make([]fleet.Software, 0, len(h.Software))
			for _, s := range h.Software {
				sws = append(sws, s.Software)
			}
		}
	}

	var cpes []fleet.SoftwareCPE
	for i, sw := range sws {
		cpes = append(cpes, fleet.SoftwareCPE{SoftwareID: sw.ID, CPE: "somecpe" + strconv.Itoa(i)})
	}

	_, err := s.ds.UpsertSoftwareCPEs(context.Background(), cpes)
	require.NoError(t, err)

	// Reload software to load GeneratedCPEID
	require.NoError(t, s.ds.LoadHostSoftware(context.Background(), hosts[0], false))

	// add CVEs for the first 10 software, which are the least used (lower hosts_count)
	for i, sw := range hosts[0].Software[:10] {
		inserted, err := s.ds.InsertSoftwareVulnerability(context.Background(), fleet.SoftwareVulnerability{
			SoftwareID: sw.ID,
			CVE:        fmt.Sprintf("cve-123-123-%03d", i),
		}, fleet.NVDSource)
		require.NoError(t, err)
		require.True(t, inserted)
	}

	// create a team and make the last 3 hosts part of it (meaning 3 that use
	// sws[19], 2 for sws[18], and 1 for sws[17])
	tm, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		Name: t.Name(),
	})
	require.NoError(t, err)
	require.NoError(t, s.ds.AddHostsToTeam(context.Background(), &tm.ID, []uint{hosts[19].ID, hosts[18].ID, hosts[17].ID}))

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
	s.DoJSON("GET", "/api/latest/fleet/software", nil, http.StatusOK, &lsResp, "order_key", "hosts_count", "order_direction", "desc")
	assertResp(lsResp, nil, time.Time{})

	// same with a team filter
	lsResp = listSoftwareResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software", nil, http.StatusOK, &lsResp, "order_key", "hosts_count", "order_direction", "desc", "team_id", fmt.Sprintf("%d", tm.ID))
	assertResp(lsResp, nil, time.Time{})

	// calculate hosts counts
	hostsCountTs := time.Now().UTC()
	require.NoError(t, s.ds.SyncHostsSoftware(context.Background(), hostsCountTs))

	// now the list software endpoint returns the software, get the first page without vulns
	lsResp = listSoftwareResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software", nil, http.StatusOK, &lsResp, "per_page", "5", "page", "0", "order_key", "hosts_count", "order_direction", "desc")
	assertResp(lsResp, []fleet.Software{sws[19], sws[18], sws[17], sws[16], sws[15]}, hostsCountTs, 20, 19, 18, 17, 16)

	// second page (page=1)
	lsResp = listSoftwareResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software", nil, http.StatusOK, &lsResp, "per_page", "5", "page", "1", "order_key", "hosts_count", "order_direction", "desc")
	assertResp(lsResp, []fleet.Software{sws[14], sws[13], sws[12], sws[11], sws[10]}, hostsCountTs, 15, 14, 13, 12, 11)

	// third page (page=2)
	lsResp = listSoftwareResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software", nil, http.StatusOK, &lsResp, "per_page", "5", "page", "2", "order_key", "hosts_count", "order_direction", "desc")
	assertResp(lsResp, []fleet.Software{sws[9], sws[8], sws[7], sws[6], sws[5]}, hostsCountTs, 10, 9, 8, 7, 6)

	// last page (page=3)
	lsResp = listSoftwareResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software", nil, http.StatusOK, &lsResp, "per_page", "5", "page", "3", "order_key", "hosts_count", "order_direction", "desc")
	assertResp(lsResp, []fleet.Software{sws[4], sws[3], sws[2], sws[1], sws[0]}, hostsCountTs, 5, 4, 3, 2, 1)

	// past the end
	lsResp = listSoftwareResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software", nil, http.StatusOK, &lsResp, "per_page", "5", "page", "4", "order_key", "hosts_count", "order_direction", "desc")
	assertResp(lsResp, nil, time.Time{})

	// no explicit sort order, defaults to hosts_count DESC
	lsResp = listSoftwareResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software", nil, http.StatusOK, &lsResp, "per_page", "2", "page", "0")
	assertResp(lsResp, []fleet.Software{sws[19], sws[18]}, hostsCountTs, 20, 19)

	// hosts_count ascending
	lsResp = listSoftwareResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software", nil, http.StatusOK, &lsResp, "per_page", "3", "page", "0", "order_key", "hosts_count", "order_direction", "asc")
	assertResp(lsResp, []fleet.Software{sws[0], sws[1], sws[2]}, hostsCountTs, 1, 2, 3)

	// vulnerable software only
	lsResp = listSoftwareResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software", nil, http.StatusOK, &lsResp, "vulnerable", "true", "per_page", "5", "page", "0", "order_key", "hosts_count", "order_direction", "desc")
	assertResp(lsResp, []fleet.Software{sws[9], sws[8], sws[7], sws[6], sws[5]}, hostsCountTs, 10, 9, 8, 7, 6)

	// vulnerable software only, next page
	lsResp = listSoftwareResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software", nil, http.StatusOK, &lsResp, "vulnerable", "true", "per_page", "5", "page", "1", "order_key", "hosts_count", "order_direction", "desc")
	assertResp(lsResp, []fleet.Software{sws[4], sws[3], sws[2], sws[1], sws[0]}, hostsCountTs, 5, 4, 3, 2, 1)

	// vulnerable software only, past last page
	lsResp = listSoftwareResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software", nil, http.StatusOK, &lsResp, "vulnerable", "true", "per_page", "5", "page", "2", "order_key", "hosts_count", "order_direction", "desc")
	assertResp(lsResp, nil, time.Time{})

	// filter by the team, 2 by page
	lsResp = listSoftwareResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software", nil, http.StatusOK, &lsResp, "per_page", "2", "page", "0", "order_key", "hosts_count", "order_direction", "desc", "team_id", fmt.Sprintf("%d", tm.ID))
	assertResp(lsResp, []fleet.Software{sws[19], sws[18]}, hostsCountTs, 3, 2)

	// filter by the team, 2 by page, next page
	lsResp = listSoftwareResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software", nil, http.StatusOK, &lsResp, "per_page", "2", "page", "1", "order_key", "hosts_count", "order_direction", "desc", "team_id", fmt.Sprintf("%d", tm.ID))
	assertResp(lsResp, []fleet.Software{sws[17]}, hostsCountTs, 1)
}

func (s *integrationTestSuite) TestChangeUserEmail() {
	t := s.T()

	// create a new test user
	user := &fleet.User{
		Name:       t.Name(),
		Email:      "testchangeemail@example.com",
		GlobalRole: ptr.String(fleet.RoleObserver),
	}
	userRawPwd := "foobarbaz1234!"
	err := user.SetPassword(userRawPwd, 10, 10)
	require.Nil(t, err)
	user, err = s.ds.NewUser(context.Background(), user)
	require.Nil(t, err)

	// try to change email with an invalid token
	var changeResp changeEmailResponse
	s.DoJSON("GET", "/api/latest/fleet/email/change/invalidtoken", nil, http.StatusNotFound, &changeResp)

	// create a valid token for the test user
	err = s.ds.PendingEmailChange(context.Background(), user.ID, "testchangeemail2@example.com", "validtoken")
	require.Nil(t, err)

	// try to change email with a valid token, but request made from different user
	changeResp = changeEmailResponse{}
	s.DoJSON("GET", "/api/latest/fleet/email/change/validtoken", nil, http.StatusNotFound, &changeResp)

	// switch to the test user and make the change email request
	s.token = s.getTestToken(user.Email, userRawPwd)
	defer func() { s.token = s.getTestAdminToken() }()

	changeResp = changeEmailResponse{}
	s.DoJSON("GET", "/api/latest/fleet/email/change/validtoken", nil, http.StatusOK, &changeResp)
	require.Equal(t, "testchangeemail2@example.com", changeResp.NewEmail)

	// using the token consumes it, so making another request with the same token fails
	changeResp = changeEmailResponse{}
	s.DoJSON("GET", "/api/latest/fleet/email/change/validtoken", nil, http.StatusNotFound, &changeResp)
}

func (s *integrationTestSuite) TestSearchTargets() {
	t := s.T()

	hosts := s.createHosts(t)

	lblIDs, err := s.ds.LabelIDsByName(context.Background(), []string{"All Hosts"})
	require.NoError(t, err)
	require.Len(t, lblIDs, 1)

	// no search criteria
	var searchResp searchTargetsResponse
	s.DoJSON("POST", "/api/latest/fleet/targets", searchTargetsRequest{}, http.StatusOK, &searchResp)
	require.Equal(t, uint(0), searchResp.TargetsCount)
	require.Len(t, searchResp.Targets.Hosts, len(hosts)) // the HostTargets.HostIDs are actually host IDs to *omit* from the search
	require.Len(t, searchResp.Targets.Labels, 1)
	require.Len(t, searchResp.Targets.Teams, 0)

	searchResp = searchTargetsResponse{}
	s.DoJSON("POST", "/api/latest/fleet/targets", searchTargetsRequest{Selected: fleet.HostTargets{LabelIDs: lblIDs}}, http.StatusOK, &searchResp)
	require.Equal(t, uint(0), searchResp.TargetsCount)
	require.Len(t, searchResp.Targets.Hosts, len(hosts)) // no omitted host id
	require.Len(t, searchResp.Targets.Labels, 0)         // labels have been omitted
	require.Len(t, searchResp.Targets.Teams, 0)

	searchResp = searchTargetsResponse{}
	s.DoJSON("POST", "/api/latest/fleet/targets", searchTargetsRequest{Selected: fleet.HostTargets{HostIDs: []uint{hosts[1].ID}}}, http.StatusOK, &searchResp)
	require.Equal(t, uint(1), searchResp.TargetsCount)
	require.Len(t, searchResp.Targets.Hosts, len(hosts)-1) // one omitted host id
	require.Len(t, searchResp.Targets.Labels, 1)           // labels have not been omitted
	require.Len(t, searchResp.Targets.Teams, 0)

	searchResp = searchTargetsResponse{}
	s.DoJSON("POST", "/api/latest/fleet/targets", searchTargetsRequest{MatchQuery: "foo.local1"}, http.StatusOK, &searchResp)
	require.Equal(t, uint(0), searchResp.TargetsCount)
	require.Len(t, searchResp.Targets.Hosts, 1)
	require.Len(t, searchResp.Targets.Labels, 1)
	require.Len(t, searchResp.Targets.Teams, 0)
	require.Contains(t, searchResp.Targets.Hosts[0].Hostname, "foo.local1")
}

func (s *integrationTestSuite) TestSearchHosts() {
	t := s.T()
	ctx := context.Background()

	hosts := s.createHosts(t)

	// set disk space information for hosts [0] and [1]
	require.NoError(t, s.ds.SetOrUpdateHostDisksSpace(ctx, hosts[0].ID, 1.0, 2.0))
	require.NoError(t, s.ds.SetOrUpdateHostDisksSpace(ctx, hosts[1].ID, 3.0, 4.0))

	// no search criteria
	var searchResp searchHostsResponse
	s.DoJSON("POST", "/api/latest/fleet/hosts/search", searchHostsRequest{}, http.StatusOK, &searchResp)
	require.Len(t, searchResp.Hosts, len(hosts)) // no request params
	for _, h := range searchResp.Hosts {
		switch h.ID {
		case hosts[0].ID:
			assert.Equal(t, 1.0, h.GigsDiskSpaceAvailable)
			assert.Equal(t, 2.0, h.PercentDiskSpaceAvailable)
		case hosts[1].ID:
			assert.Equal(t, 3.0, h.GigsDiskSpaceAvailable)
			assert.Equal(t, 4.0, h.PercentDiskSpaceAvailable)
		}
		assert.Equal(t, h.SoftwareUpdatedAt, h.CreatedAt)
	}

	searchResp = searchHostsResponse{}
	s.DoJSON("POST", "/api/latest/fleet/hosts/search", searchHostsRequest{ExcludedHostIDs: []uint{}}, http.StatusOK, &searchResp)
	require.Len(t, searchResp.Hosts, len(hosts)) // no omitted host id

	searchResp = searchHostsResponse{}
	s.DoJSON("POST", "/api/latest/fleet/hosts/search", searchHostsRequest{ExcludedHostIDs: []uint{hosts[1].ID}}, http.StatusOK, &searchResp)
	require.Len(t, searchResp.Hosts, len(hosts)-1) // one omitted host id

	searchResp = searchHostsResponse{}
	s.DoJSON("POST", "/api/latest/fleet/hosts/search", searchHostsRequest{MatchQuery: "foo.local1"}, http.StatusOK, &searchResp)
	require.Len(t, searchResp.Hosts, 1)
	require.Contains(t, searchResp.Hosts[0].Hostname, "foo.local1")

	// Update software and check that the software_updated_at is updated for the host returned by the search.
	time.Sleep(1 * time.Second)
	software := []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
	}
	_, err := s.ds.UpdateHostSoftware(context.Background(), hosts[0].ID, software)
	require.NoError(t, err)
	searchResp = searchHostsResponse{}
	s.DoJSON("POST", "/api/latest/fleet/hosts/search", searchHostsRequest{MatchQuery: "foo.local0"}, http.StatusOK, &searchResp)
	require.Len(t, searchResp.Hosts, 1)
	require.Greater(t, searchResp.Hosts[0].SoftwareUpdatedAt, searchResp.Hosts[0].CreatedAt)
}

func (s *integrationTestSuite) TestCountTargets() {
	t := s.T()

	team, err := s.ds.NewTeam(context.Background(), &fleet.Team{Name: "TestTeam"})
	require.NoError(t, err)
	require.Equal(t, "TestTeam", team.Name)

	hosts := s.createHosts(t)

	lblIDs, err := s.ds.LabelIDsByName(context.Background(), []string{"All Hosts"})
	require.NoError(t, err)
	require.Len(t, lblIDs, 1)

	for i := range hosts {
		err = s.ds.RecordLabelQueryExecutions(context.Background(), hosts[i], map[uint]*bool{lblIDs[0]: ptr.Bool(true)}, time.Now(), false)
		require.NoError(t, err)
	}

	var hostIDs []uint
	for _, h := range hosts {
		hostIDs = append(hostIDs, h.ID)
	}

	err = s.ds.AddHostsToTeam(context.Background(), ptr.Uint(team.ID), []uint{hostIDs[0]})
	require.NoError(t, err)

	var countResp countTargetsResponse
	// sleep to reduce flake in last seen time so that online/offline counts can be tested
	time.Sleep(1 * time.Second)

	// none selected
	s.DoJSON("POST", "/api/latest/fleet/targets/count", countTargetsRequest{}, http.StatusOK, &countResp)
	require.Equal(t, uint(0), countResp.TargetsCount)
	require.Equal(t, uint(0), countResp.TargetsOnline)
	require.Equal(t, uint(0), countResp.TargetsOffline)

	// all hosts label selected
	countResp = countTargetsResponse{}
	s.DoJSON("POST", "/api/latest/fleet/targets/count", countTargetsRequest{Selected: fleet.HostTargets{LabelIDs: lblIDs}}, http.StatusOK, &countResp)
	require.Equal(t, uint(3), countResp.TargetsCount)
	require.Equal(t, uint(1), countResp.TargetsOnline)
	require.Equal(t, uint(2), countResp.TargetsOffline)

	// team selected
	countResp = countTargetsResponse{}
	s.DoJSON("POST", "/api/latest/fleet/targets/count", countTargetsRequest{Selected: fleet.HostTargets{TeamIDs: []uint{team.ID}}}, http.StatusOK, &countResp)
	require.Equal(t, uint(1), countResp.TargetsCount)
	require.Equal(t, uint(1), countResp.TargetsOnline)
	require.Equal(t, uint(0), countResp.TargetsOffline)

	// host id selected
	countResp = countTargetsResponse{}
	s.DoJSON("POST", "/api/latest/fleet/targets/count", countTargetsRequest{Selected: fleet.HostTargets{HostIDs: []uint{hosts[1].ID}}}, http.StatusOK, &countResp)
	require.Equal(t, uint(1), countResp.TargetsCount)
	require.Equal(t, uint(0), countResp.TargetsOnline)
	require.Equal(t, uint(1), countResp.TargetsOffline)
}

func (s *integrationTestSuite) TestStatus() {
	var statusResp statusResponse
	s.DoJSON("GET", "/api/latest/fleet/status/result_store", nil, http.StatusOK, &statusResp)
	s.DoJSON("GET", "/api/latest/fleet/status/live_query", nil, http.StatusOK, &statusResp)
}

func (s *integrationTestSuite) TestOsqueryConfig() {
	t := s.T()

	hosts := s.createHosts(t)
	req := getClientConfigRequest{NodeKey: *hosts[0].NodeKey}
	var resp getClientConfigResponse
	s.DoJSON("POST", "/api/osquery/config", req, http.StatusOK, &resp)

	// test with invalid node key
	var errRes map[string]interface{}
	req.NodeKey += "zzzz"
	s.DoJSON("POST", "/api/osquery/config", req, http.StatusUnauthorized, &errRes)
	assert.Contains(t, errRes["error"], "invalid node key")
}

func (s *integrationTestSuite) TestEnrollHost() {
	t := s.T()

	// set the enroll secret
	var applyResp applyEnrollSecretSpecResponse
	s.DoJSON("POST", "/api/latest/fleet/spec/enroll_secret", applyEnrollSecretSpecRequest{
		Spec: &fleet.EnrollSecretSpec{
			Secrets: []*fleet.EnrollSecret{{Secret: t.Name()}},
		},
	}, http.StatusOK, &applyResp)

	// invalid enroll secret fails
	j, err := json.Marshal(&enrollAgentRequest{
		EnrollSecret:   "nosuchsecret",
		HostIdentifier: "abcd",
	})
	require.NoError(t, err)
	s.DoRawNoAuth("POST", "/api/osquery/enroll", j, http.StatusUnauthorized)

	// valid enroll secret succeeds
	j, err = json.Marshal(&enrollAgentRequest{
		EnrollSecret:   t.Name(),
		HostIdentifier: t.Name(),
	})
	require.NoError(t, err)

	var resp enrollAgentResponse
	hres := s.DoRawNoAuth("POST", "/api/osquery/enroll", j, http.StatusOK)
	defer hres.Body.Close()
	require.NoError(t, json.NewDecoder(hres.Body).Decode(&resp))
	require.NotEmpty(t, resp.NodeKey)
}

func (s *integrationTestSuite) TestReenrollHostCleansPolicies() {
	t := s.T()
	ctx := context.Background()
	host := s.createHosts(t)[0]

	// set the enroll secret
	var applyResp applyEnrollSecretSpecResponse
	s.DoJSON("POST", "/api/latest/fleet/spec/enroll_secret", applyEnrollSecretSpecRequest{
		Spec: &fleet.EnrollSecretSpec{
			Secrets: []*fleet.EnrollSecret{{Secret: t.Name()}},
		},
	}, http.StatusOK, &applyResp)

	var getHostResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.Empty(t, getHostResp.Host.Policies)

	// create a policy and make the host fail it
	pol, err := s.ds.NewGlobalPolicy(ctx, nil, fleet.PolicyPayload{Name: t.Name(), Query: "SELECT 1", Platform: host.FleetPlatform()})
	require.NoError(t, err)
	err = s.ds.RecordPolicyQueryExecutions(ctx, &fleet.Host{ID: host.ID}, map[uint]*bool{pol.ID: ptr.Bool(false)}, time.Now(), false)
	require.NoError(t, err)

	// refetch the host details
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.Len(t, *getHostResp.Host.Policies, 1)

	// re-enroll the host, but using a different platform
	j, err := json.Marshal(&enrollAgentRequest{
		EnrollSecret:   t.Name(),
		HostIdentifier: *host.OsqueryHostID,
		HostDetails:    map[string](map[string]string){"os_version": map[string]string{"platform": "windows"}},
	})
	require.NoError(t, err)

	// prevent the enroll cooldown from being applied
	mysql.ExecAdhocSQL(t, s.ds, func(db sqlx.ExtContext) error {
		_, err := db.ExecContext(
			context.Background(),
			"UPDATE hosts SET last_enrolled_at = DATE_SUB(NOW(), INTERVAL '1' HOUR) WHERE id = ?",
			host.ID,
		)
		return err
	})
	var resp enrollAgentResponse
	hres := s.DoRawNoAuth("POST", "/api/osquery/enroll", j, http.StatusOK)
	defer hres.Body.Close()
	require.NoError(t, json.NewDecoder(hres.Body).Decode(&resp))
	require.NotEmpty(t, resp.NodeKey)

	// refetch the host details
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)

	// policies should be gone
	require.Empty(t, getHostResp.Host.Policies)
}

func (s *integrationTestSuite) TestCarve() {
	t := s.T()
	hosts := s.createHosts(t)

	// begin a carve with an invalid node key
	var errRes map[string]interface{}
	s.DoJSON("POST", "/api/osquery/carve/begin", carveBeginRequest{
		NodeKey:    *hosts[0].NodeKey + "zzz",
		BlockCount: 1,
		BlockSize:  1,
		CarveSize:  1,
		CarveId:    "c1",
	}, http.StatusUnauthorized, &errRes)
	assert.Contains(t, errRes["error"], "invalid node key")

	// invalid carve size
	s.DoJSON("POST", "/api/osquery/carve/begin", carveBeginRequest{
		NodeKey:    *hosts[0].NodeKey,
		BlockCount: 3,
		BlockSize:  3,
		CarveSize:  0,
		CarveId:    "c1",
	}, http.StatusInternalServerError, &errRes) // TODO: should be 4xx, see #4406
	assert.Contains(t, errRes["error"], "carve_size must be greater")

	// invalid block size too big
	s.DoJSON("POST", "/api/osquery/carve/begin", carveBeginRequest{
		NodeKey:    *hosts[0].NodeKey,
		BlockCount: 3,
		BlockSize:  maxBlockSize + 1,
		CarveSize:  maxCarveSize,
		CarveId:    "c1",
	}, http.StatusInternalServerError, &errRes) // TODO: should be 4xx, see #4406
	assert.Contains(t, errRes["error"], "block_size exceeds max")

	// invalid carve size too big
	s.DoJSON("POST", "/api/osquery/carve/begin", carveBeginRequest{
		NodeKey:    *hosts[0].NodeKey,
		BlockCount: 3,
		BlockSize:  maxBlockSize,
		CarveSize:  maxCarveSize + 1,
		CarveId:    "c1",
	}, http.StatusInternalServerError, &errRes) // TODO: should be 4xx, see #4406
	assert.Contains(t, errRes["error"], "carve_size exceeds max")

	// invalid carve size, does not match blocks
	s.DoJSON("POST", "/api/osquery/carve/begin", carveBeginRequest{
		NodeKey:    *hosts[0].NodeKey,
		BlockCount: 3,
		BlockSize:  3,
		CarveSize:  1,
		CarveId:    "c1",
	}, http.StatusInternalServerError, &errRes) // TODO: should be 4xx, see #4406
	assert.Contains(t, errRes["error"], "carve_size does not match")

	// valid carve begin
	var beginResp carveBeginResponse
	s.DoJSON("POST", "/api/osquery/carve/begin", carveBeginRequest{
		NodeKey:    *hosts[0].NodeKey,
		BlockCount: 3,
		BlockSize:  3,
		CarveSize:  8,
		CarveId:    "c1",
		RequestId:  "r1",
	}, http.StatusOK, &beginResp)
	require.NotEmpty(t, beginResp.SessionId)
	sid := beginResp.SessionId

	// sending a block with invalid session id
	var blockResp carveBlockResponse
	s.DoJSON("POST", "/api/osquery/carve/block", carveBlockRequest{
		BlockId:   1,
		SessionId: sid + "zz",
		RequestId: "??",
		Data:      []byte("p1."),
	}, http.StatusNotFound, &blockResp)

	// sending a block with valid session id but invalid request id
	s.DoJSON("POST", "/api/osquery/carve/block", carveBlockRequest{
		BlockId:   1,
		SessionId: sid,
		RequestId: "??",
		Data:      []byte("p1."),
	}, http.StatusInternalServerError, &blockResp) // TODO: should be 400, see #4406

	checkCarveError := func(id uint, err string) {
		var getResp getCarveResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/carves/%d", id), nil, http.StatusOK, &getResp)
		require.Equal(t, err, *getResp.Carve.Error)
	}

	// sending a block with unexpected block id (expects 0, got 1)
	s.DoJSON("POST", "/api/osquery/carve/block", carveBlockRequest{
		BlockId:   1,
		SessionId: sid,
		RequestId: "r1",
		Data:      []byte("p1."),
	}, http.StatusInternalServerError, &blockResp) // TODO: should be 400, see #4406
	checkCarveError(1, "block_id does not match expected block (0): 1")

	// sending a block with valid payload, block 0
	s.DoJSON("POST", "/api/osquery/carve/block", carveBlockRequest{
		BlockId:   0,
		SessionId: sid,
		RequestId: "r1",
		Data:      []byte("p1."),
	}, http.StatusOK, &blockResp)
	require.True(t, blockResp.Success)

	// sending next block
	blockResp = carveBlockResponse{}
	s.DoJSON("POST", "/api/osquery/carve/block", carveBlockRequest{
		BlockId:   1,
		SessionId: sid,
		RequestId: "r1",
		Data:      []byte("p2."),
	}, http.StatusOK, &blockResp)
	require.True(t, blockResp.Success)

	// sending already-sent block again
	blockResp = carveBlockResponse{}
	s.DoJSON("POST", "/api/osquery/carve/block", carveBlockRequest{
		BlockId:   1,
		SessionId: sid,
		RequestId: "r1",
		Data:      []byte("p2."),
	}, http.StatusInternalServerError, &blockResp) // TODO: should be 400, see #4406
	checkCarveError(1, "block_id does not match expected block (2): 1")

	// sending final block with too many bytes
	blockResp = carveBlockResponse{}
	s.DoJSON("POST", "/api/osquery/carve/block", carveBlockRequest{
		BlockId:   2,
		SessionId: sid,
		RequestId: "r1",
		Data:      []byte("p3extra"),
	}, http.StatusInternalServerError, &blockResp) // TODO: should be 400, see #4406
	checkCarveError(1, "exceeded declared block size 3: 7")

	// sending actual final block
	blockResp = carveBlockResponse{}
	s.DoJSON("POST", "/api/osquery/carve/block", carveBlockRequest{
		BlockId:   2,
		SessionId: sid,
		RequestId: "r1",
		Data:      []byte("p3"),
	}, http.StatusOK, &blockResp)
	require.True(t, blockResp.Success)

	// sending unexpected block
	blockResp = carveBlockResponse{}
	s.DoJSON("POST", "/api/osquery/carve/block", carveBlockRequest{
		BlockId:   3,
		SessionId: sid,
		RequestId: "r1",
		Data:      []byte("p4."),
	}, http.StatusInternalServerError, &blockResp) // TODO: should be 400, see #4406
	checkCarveError(1, "block_id exceeds expected max (2): 3")
}

func (s *integrationTestSuite) TestLogLoginAttempts() {
	t := s.T()

	// create a new user
	var createResp createUserResponse
	params := fleet.UserPayload{
		Name:       ptr.String("foobar"),
		Email:      ptr.String("foobar@example.com"),
		Password:   ptr.String(test.GoodPassword),
		GlobalRole: ptr.String(fleet.RoleObserver),
	}
	s.DoJSON("POST", "/api/latest/fleet/users/admin", params, http.StatusOK, &createResp)
	require.NotZero(t, createResp.User.ID)
	u := *createResp.User

	// Register current number of activities.
	activitiesResp := listActivitiesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &activitiesResp)
	require.NoError(t, activitiesResp.Err)
	oldActivitiesCount := len(activitiesResp.Activities)

	// Login with invalid passwordm, should fail.
	res := s.DoRawNoAuth("POST", "/api/latest/fleet/login",
		jsonMustMarshal(t, loginRequest{Email: u.Email, Password: test.GoodPassword2}),
		http.StatusUnauthorized,
	)
	res.Body.Close()

	// A new activity item for the failed login attempt is created.
	activitiesResp = listActivitiesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &activitiesResp)
	require.NoError(t, activitiesResp.Err)
	require.Len(t, activitiesResp.Activities, oldActivitiesCount+1)
	sort.Slice(activitiesResp.Activities, func(i, j int) bool {
		return activitiesResp.Activities[i].ID < activitiesResp.Activities[j].ID
	})
	activity := activitiesResp.Activities[len(activitiesResp.Activities)-1]
	require.Equal(t, activity.Type, fleet.ActivityTypeUserFailedLogin{}.ActivityName())
	require.NotNil(t, activity.Details)
	actDetails := fleet.ActivityTypeUserFailedLogin{}
	err := json.Unmarshal(*activity.Details, &actDetails)
	require.NoError(t, err)
	require.Equal(t, actDetails.Email, "foobar@example.com")

	// login with good password, should succeed
	res = s.DoRawNoAuth("POST", "/api/latest/fleet/login",
		jsonMustMarshal(t, loginRequest{
			Email:    u.Email,
			Password: test.GoodPassword,
		}), http.StatusOK,
	)
	res.Body.Close()

	// A new activity item for the successful login is created.
	activitiesResp = listActivitiesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &activitiesResp)
	require.NoError(t, activitiesResp.Err)
	require.Len(t, activitiesResp.Activities, oldActivitiesCount+2)
	sort.Slice(activitiesResp.Activities, func(i, j int) bool {
		return activitiesResp.Activities[i].ID < activitiesResp.Activities[j].ID
	})
	activity = activitiesResp.Activities[len(activitiesResp.Activities)-1]
	require.Equal(t, activity.Type, fleet.ActivityTypeUserLoggedIn{}.ActivityName())
	require.NotNil(t, activity.Details)
	err = json.Unmarshal(*activity.Details, &fleet.ActivityTypeUserLoggedIn{})
	require.NoError(t, err)
}

func (s *integrationTestSuite) TestPasswordReset() {
	t := s.T()

	// create a new user
	var createResp createUserResponse
	userRawPwd := test.GoodPassword
	params := fleet.UserPayload{
		Name:       ptr.String("forgotpwd"),
		Email:      ptr.String("forgotpwd@example.com"),
		Password:   ptr.String(userRawPwd),
		GlobalRole: ptr.String(fleet.RoleObserver),
	}
	s.DoJSON("POST", "/api/latest/fleet/users/admin", params, http.StatusOK, &createResp)
	require.NotZero(t, createResp.User.ID)
	u := *createResp.User

	// request forgot password, invalid email
	res := s.DoRawNoAuth("POST", "/api/latest/fleet/forgot_password", jsonMustMarshal(t, forgotPasswordRequest{Email: "invalid@asd.com"}), http.StatusAccepted)
	res.Body.Close()

	// TODO: tested manually (adds too much time to the test), works but hitting the rate
	// limit returns 500 instead of 429, see #4406. We get the authz check missing error instead.
	//// trigger the rate limit with a batch of requests in a short burst
	//for i := 0; i < 20; i++ {
	//	s.DoJSON("POST", "/api/latest/fleet/forgot_password", forgotPasswordRequest{Email: "invalid@asd.com"}, http.StatusAccepted, &forgotResp)
	//}

	// request forgot password, valid email
	res = s.DoRawNoAuth("POST", "/api/latest/fleet/forgot_password", jsonMustMarshal(t, forgotPasswordRequest{Email: u.Email}), http.StatusAccepted)
	res.Body.Close()

	var token string
	mysql.ExecAdhocSQL(t, s.ds, func(db sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), db, &token, "SELECT token FROM password_reset_requests WHERE user_id = ?", u.ID)
	})

	// proceed with reset password
	userNewPwd := test.GoodPassword2
	res = s.DoRawNoAuth("POST", "/api/latest/fleet/reset_password", jsonMustMarshal(t, resetPasswordRequest{PasswordResetToken: token, NewPassword: userNewPwd}), http.StatusOK)
	res.Body.Close()

	// attempt it again with already-used token
	userUnusedPwd := "unusedpassw0rd!"
	res = s.DoRawNoAuth("POST", "/api/latest/fleet/reset_password", jsonMustMarshal(t, resetPasswordRequest{PasswordResetToken: token, NewPassword: userUnusedPwd}), http.StatusUnauthorized)
	res.Body.Close()

	// login with the old password, should not succeed
	res = s.DoRawNoAuth("POST", "/api/latest/fleet/login", jsonMustMarshal(t, loginRequest{Email: u.Email, Password: userRawPwd}), http.StatusUnauthorized)
	res.Body.Close()

	// login with the new password, should succeed
	res = s.DoRawNoAuth("POST", "/api/latest/fleet/login", jsonMustMarshal(t, loginRequest{Email: u.Email, Password: userNewPwd}), http.StatusOK)
	res.Body.Close()
}

func (s *integrationTestSuite) TestModifyUser() {
	t := s.T()

	// create a new user
	var createResp createUserResponse
	userRawPwd := test.GoodPassword
	params := fleet.UserPayload{
		Name:                     ptr.String("moduser"),
		Email:                    ptr.String("moduser@example.com"),
		Password:                 ptr.String(userRawPwd),
		GlobalRole:               ptr.String(fleet.RoleObserver),
		AdminForcedPasswordReset: ptr.Bool(false),
	}
	s.DoJSON("POST", "/api/latest/fleet/users/admin", params, http.StatusOK, &createResp)
	require.NotZero(t, createResp.User.ID)
	u := *createResp.User

	s.token = s.getTestToken(u.Email, userRawPwd)
	require.NotEmpty(t, s.token)
	defer func() { s.token = s.getTestAdminToken() }()

	// as the user: modify email without providing current password
	var modResp modifyUserResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/users/%d", u.ID), fleet.UserPayload{
		Email: ptr.String("moduser2@example.com"),
	}, http.StatusUnprocessableEntity, &modResp)

	// as the user: modify email with invalid password
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/users/%d", u.ID), fleet.UserPayload{
		Email:    ptr.String("moduser2@example.com"),
		Password: ptr.String("nosuchpwd"),
	}, http.StatusForbidden, &modResp)

	// as the user: modify email with current password
	newEmail := "moduser2@example.com"
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/users/%d", u.ID), fleet.UserPayload{
		Email:    ptr.String(newEmail),
		Password: ptr.String(userRawPwd),
	}, http.StatusOK, &modResp)
	require.Equal(t, u.ID, modResp.User.ID)
	require.Equal(t, u.Email, modResp.User.Email) // new email is pending confirmation, not changed immediately

	// as the user: set new password without providing current one
	newRawPwd := test.GoodPassword2
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/users/%d", u.ID), fleet.UserPayload{
		NewPassword: ptr.String(newRawPwd),
	}, http.StatusUnprocessableEntity, &modResp)

	// as the user: set new password with an invalid current password
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/users/%d", u.ID), fleet.UserPayload{
		NewPassword: ptr.String(newRawPwd),
		Password:    ptr.String("nosuchpwd"),
	}, http.StatusForbidden, &modResp)

	// as the user: set new password and change name, with a valid current password
	modResp = modifyUserResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/users/%d", u.ID), fleet.UserPayload{
		NewPassword: ptr.String(newRawPwd),
		Password:    ptr.String(userRawPwd),
		Name:        ptr.String("moduser2"),
	}, http.StatusOK, &modResp)
	require.Equal(t, u.ID, modResp.User.ID)
	require.Equal(t, "moduser2", modResp.User.Name)

	s.token = s.getTestToken(testUsers["user2"].Email, testUsers["user2"].PlaintextPassword)

	// as a different user: set new password with different user's old password (ensure
	// any other user that is not admin cannot change another user's password)
	newRawPwd = userRawPwd + "3"
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/users/%d", u.ID), fleet.UserPayload{
		NewPassword: ptr.String(newRawPwd),
		Password:    ptr.String(testUsers["user2"].PlaintextPassword),
	}, http.StatusForbidden, &modResp)

	s.token = s.getTestAdminToken()

	// as an admin, set a new email, name and password without a current password
	newRawPwd = userRawPwd + "4"
	modResp = modifyUserResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/users/%d", u.ID), fleet.UserPayload{
		NewPassword: ptr.String(newRawPwd),
		Email:       ptr.String("moduser3@example.com"),
		Name:        ptr.String("moduser3"),
	}, http.StatusOK, &modResp)
	require.Equal(t, u.ID, modResp.User.ID)
	require.Equal(t, "moduser3", modResp.User.Name)

	// as an admin, set new password that doesn't meet requirements
	invalidUserPwd := "abc"
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/users/%d", u.ID), fleet.UserPayload{
		NewPassword: ptr.String(invalidUserPwd),
	}, http.StatusUnprocessableEntity, &modResp)

	// login as the user, with the last password successfully set (to confirm it is the current one)
	var loginResp loginResponse
	resp := s.DoRawNoAuth("POST", "/api/latest/fleet/login", jsonMustMarshal(t, loginRequest{
		Email:    u.Email, // all email changes made are still pending, never confirmed
		Password: newRawPwd,
	}), http.StatusOK)
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&loginResp))
	resp.Body.Close()
	require.Equal(t, u.ID, loginResp.User.ID)
}

func (s *integrationTestSuite) TestGetHostLastOpenedAt() {
	t := s.T()

	host, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String(t.Name() + "1"),
		UUID:            t.Name() + "1",
		Hostname:        t.Name() + "foo.local",
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-58",
	})
	require.NoError(t, err)
	require.NotNil(t, host)

	today := time.Now()
	yesterday := today.Add(-24 * time.Hour)
	software := []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
		{Name: "bar", Version: "0.0.3", Source: "apps", LastOpenedAt: &today},
		{Name: "baz", Version: "0.0.4", Source: "apps", LastOpenedAt: &yesterday},
	}
	_, err = s.ds.UpdateHostSoftware(context.Background(), host.ID, software)
	require.NoError(t, err)

	var getHostResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.Equal(t, host.ID, getHostResp.Host.ID)
	require.Len(t, getHostResp.Host.Software, len(software))

	sort.Slice(getHostResp.Host.Software, func(l, r int) bool {
		lsw, rsw := getHostResp.Host.Software[l], getHostResp.Host.Software[r]
		return lsw.Name < rsw.Name
	})
	// bar, baz, foo, in this order
	wantTs := []time.Time{today, yesterday, {}}
	for i, want := range wantTs {
		sw := getHostResp.Host.Software[i]
		if want.IsZero() {
			require.Nil(t, sw.LastOpenedAt)
		} else {
			require.WithinDuration(t, want, *sw.LastOpenedAt, time.Second)
		}
	}

	// listing hosts does not return the last opened at timestamp, only the GET /hosts/{id} endpoint
	var listHostsResp listHostsResponse
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listHostsResp)

	var hostSeen bool
	for _, h := range listHostsResp.Hosts {
		if h.ID == host.ID {
			hostSeen = true
		}
		for _, sw := range h.Software {
			require.Nil(t, sw.LastOpenedAt)
		}
	}
	require.True(t, hostSeen)
}

func (s *integrationTestSuite) TestGetHostSoftwareUpdatedAt() {
	t := s.T()

	host, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String(t.Name() + "1"),
		UUID:            t.Name() + "1",
		Hostname:        t.Name() + "foo.local",
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-58",
	})
	require.NoError(t, err)
	require.NotNil(t, host)

	var getHostResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.Equal(t, host.ID, getHostResp.Host.ID)
	require.Empty(t, getHostResp.Host.Software)
	require.Equal(t, getHostResp.Host.SoftwareUpdatedAt, getHostResp.Host.CreatedAt)

	// Sleep for 1 second to have software_updated_at be bigger than created_at.
	time.Sleep(1 * time.Second)

	software := []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
	}
	_, err = s.ds.UpdateHostSoftware(context.Background(), host.ID, software)
	require.NoError(t, err)

	getHostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.Equal(t, host.ID, getHostResp.Host.ID)
	require.Len(t, getHostResp.Host.Software, len(software))
	require.Greater(t, getHostResp.Host.SoftwareUpdatedAt, getHostResp.Host.CreatedAt)
}

func (s *integrationTestSuite) TestHostsReportDownload() {
	t := s.T()
	ctx := context.Background()

	hosts := s.createHosts(t)
	err := s.ds.ApplyLabelSpecs(context.Background(), []*fleet.LabelSpec{
		{Name: t.Name(), LabelMembershipType: fleet.LabelMembershipTypeManual, Query: "select 1", Hosts: []string{hosts[2].Hostname}},
	})
	require.NoError(t, err)
	lids, err := s.ds.LabelIDsByName(context.Background(), []string{t.Name()})
	require.NoError(t, err)
	require.Len(t, lids, 1)
	customLabelID := lids[0]

	// create a policy and make host[1] fail that policy
	pol, err := s.ds.NewGlobalPolicy(ctx, nil, fleet.PolicyPayload{Name: t.Name(), Query: "SELECT 1"})
	require.NoError(t, err)
	err = s.ds.RecordPolicyQueryExecutions(ctx, hosts[1], map[uint]*bool{pol.ID: ptr.Bool(false)}, time.Now(), false)
	require.NoError(t, err)

	// create some device mappings for host[2]
	err = s.ds.ReplaceHostDeviceMapping(ctx, hosts[2].ID, []*fleet.HostDeviceMapping{
		{HostID: hosts[2].ID, Email: "a@b.c", Source: "google_chrome_profiles"},
		{HostID: hosts[2].ID, Email: "b@b.c", Source: "google_chrome_profiles"},
	})
	require.NoError(t, err)

	// set disk space information for hosts [0] and [1]
	require.NoError(t, s.ds.SetOrUpdateHostDisksSpace(ctx, hosts[0].ID, 1.0, 2.0))
	require.NoError(t, s.ds.SetOrUpdateHostDisksSpace(ctx, hosts[1].ID, 3.0, 4.0))

	res := s.DoRaw("GET", "/api/latest/fleet/hosts/report", nil, http.StatusUnsupportedMediaType, "format", "gzip")
	var errs validationErrResp
	require.NoError(t, json.NewDecoder(res.Body).Decode(&errs))
	res.Body.Close()
	require.Len(t, errs.Errors, 1)
	assert.Equal(t, "format", errs.Errors[0].Name)

	// valid format, no column specified so all columns returned
	res = s.DoRaw("GET", "/api/latest/fleet/hosts/report", nil, http.StatusOK, "format", "csv")
	rows, err := csv.NewReader(res.Body).ReadAll()
	res.Body.Close()
	require.NoError(t, err)
	require.Len(t, rows, len(hosts)+1) // all hosts + header row
	require.Len(t, rows[0], 48)        // total number of cols
	t.Log(rows[0])

	const (
		idCol       = 3
		issuesCol   = 41
		gigsDiskCol = 39
		pctDiskCol  = 40
	)

	// find the row for hosts[1], it should have issues=1 (1 failing policy) and the expected disk space
	for _, row := range rows[1:] {
		if row[idCol] == fmt.Sprint(hosts[1].ID) {
			require.Equal(t, "1", row[issuesCol], row)
			require.Equal(t, "3", row[gigsDiskCol], row)
			require.Equal(t, "4", row[pctDiskCol], row)
		} else {
			require.Equal(t, "0", row[issuesCol], row)
		}
	}

	// valid format, some columns
	res = s.DoRaw("GET", "/api/latest/fleet/hosts/report", nil, http.StatusOK, "format", "csv", "columns", "hostname", "gigs_disk_space_available", "percent_disk_space_available")
	rows, err = csv.NewReader(res.Body).ReadAll()
	res.Body.Close()
	require.NoError(t, err)
	require.Len(t, rows, len(hosts)+1)
	require.Contains(t, rows[0], "hostname") // first row contains headers
	require.Contains(t, res.Header, "Content-Disposition")
	require.Contains(t, res.Header, "Content-Type")
	require.Contains(t, res.Header, "X-Content-Type-Options")
	require.Contains(t, res.Header.Get("Content-Disposition"), "attachment;")
	require.Contains(t, res.Header.Get("Content-Type"), "text/csv")
	require.Contains(t, res.Header.Get("X-Content-Type-Options"), "nosniff")

	// pagination does not apply to this endpoint, it returns the complete list of hosts
	res = s.DoRaw("GET", "/api/latest/fleet/hosts/report", nil, http.StatusOK, "format", "csv", "page", "1", "per_page", "2", "columns", "hostname")
	rows, err = csv.NewReader(res.Body).ReadAll()
	res.Body.Close()
	require.NoError(t, err)
	require.Len(t, rows, len(hosts)+1)

	// search criteria are applied
	res = s.DoRaw("GET", "/api/latest/fleet/hosts/report", nil, http.StatusOK, "format", "csv", "query", "local0", "columns", "hostname")
	rows, err = csv.NewReader(res.Body).ReadAll()
	res.Body.Close()
	require.NoError(t, err)
	require.Len(t, rows, 2) // headers + matching host
	require.Contains(t, rows[1], hosts[0].Hostname)

	// with device mapping results
	res = s.DoRaw("GET", "/api/latest/fleet/hosts/report", nil, http.StatusOK, "format", "csv", "columns", "id,hostname,device_mapping")
	rawCSV, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	require.Contains(t, string(rawCSV), `"a@b.c,b@b.c"`) // inside quotes because it contains a comma
	rows, err = csv.NewReader(bytes.NewReader(rawCSV)).ReadAll()
	res.Body.Close()
	require.NoError(t, err)
	require.Len(t, rows, len(hosts)+1)
	for _, row := range rows[1:] {
		if row[0] == fmt.Sprint(hosts[2].ID) {
			require.Equal(t, "a@b.c,b@b.c", row[2], row)
		} else {
			require.Equal(t, "", row[2], row)
		}
	}

	// with a label id
	res = s.DoRaw("GET", "/api/latest/fleet/hosts/report", nil, http.StatusOK, "format", "csv", "columns", "hostname", "label_id", fmt.Sprintf("%d", customLabelID))
	rows, err = csv.NewReader(res.Body).ReadAll()
	res.Body.Close()
	require.NoError(t, err)
	require.Len(t, rows, 2) // headers + member host
	require.Contains(t, rows[1], hosts[2].Hostname)

	// valid format but an invalid column is provided
	res = s.DoRaw("GET", "/api/latest/fleet/hosts/report", nil, http.StatusBadRequest, "format", "csv", "columns", "memory,hostname,status,nosuchcolumn")
	require.NoError(t, json.NewDecoder(res.Body).Decode(&errs))
	res.Body.Close()
	require.Len(t, errs.Errors, 1)
	require.Contains(t, errs.Errors[0].Reason, "nosuchcolumn")

	// valid format, valid columns, order is respected, sorted
	res = s.DoRaw("GET", "/api/latest/fleet/hosts/report", nil, http.StatusOK, "format", "csv", "order_key", "hostname", "order_direction", "desc", "columns", "memory,hostname,status")
	rows, err = csv.NewReader(res.Body).ReadAll()
	res.Body.Close()
	require.NoError(t, err)
	require.Len(t, rows, len(hosts)+1)
	require.Equal(t, []string{"memory", "hostname", "status"}, rows[0]) // first row contains headers
	require.Len(t, rows[1], 3)
	// status is timing-dependent, ignore in the assertion
	require.Equal(t, []string{"0", "TestIntegrations/TestHostsReportDownloadfoo.local2"}, rows[1][:2])
	require.Len(t, rows[2], 3)
	require.Equal(t, []string{"0", "TestIntegrations/TestHostsReportDownloadfoo.local1"}, rows[2][:2])
	require.Len(t, rows[3], 3)
	require.Equal(t, []string{"0", "TestIntegrations/TestHostsReportDownloadfoo.local0"}, rows[3][:2])
	t.Log(rows)
}

func (s *integrationTestSuite) TestSSODisabled() {
	t := s.T()

	var initiateResp initiateSSOResponse
	s.DoJSON("POST", "/api/v1/fleet/sso", struct{}{}, http.StatusBadRequest, &initiateResp)

	var callbackResp callbackSSOResponse
	// callback without SAML response
	s.DoJSON("POST", "/api/v1/fleet/sso/callback", nil, http.StatusBadRequest, &callbackResp)
	// callback with invalid SAML response
	s.DoJSON("POST", "/api/v1/fleet/sso/callback?SAMLResponse=zz", nil, http.StatusBadRequest, &callbackResp)
	// callback with valid SAML response (<samlp:AuthnRequest></samlp:AuthnRequest>)
	res := s.DoRaw("POST", "/api/v1/fleet/sso/callback?SAMLResponse=PHNhbWxwOkF1dGhuUmVxdWVzdD48L3NhbWxwOkF1dGhuUmVxdWVzdD4%3D", nil, http.StatusOK)
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	require.Contains(t, string(body), "/login?status=org_disabled") // html contains a script that redirects to this path
}

func (s *integrationTestSuite) TestSandboxEndpoints() {
	t := s.T()
	validEmail := testUsers["user1"].Email
	validPwd := testUsers["user1"].PlaintextPassword
	hdrs := map[string]string{"Content-Type": "application/x-www-form-urlencoded"}

	// demo login endpoint always fails
	formBody := make(url.Values)
	formBody.Set("email", validEmail)
	formBody.Set("password", validPwd)
	res := s.DoRawWithHeaders("POST", "/api/v1/fleet/demologin", []byte(formBody.Encode()), http.StatusInternalServerError, hdrs)
	require.NotEqual(t, http.StatusOK, res.StatusCode)

	// installers endpoint is not enabled
	url, installersBody := installerPOSTReq(enrollSecret, "pkg", s.token, false)
	s.DoRaw("POST", url, installersBody, http.StatusInternalServerError)
}

func (s *integrationTestSuite) TestGetHostBatteries() {
	t := s.T()

	host, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String(strings.ReplaceAll(t.Name(), "/", "_") + "1"),
		UUID:            t.Name() + "1",
		Hostname:        t.Name() + "foo.local",
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-58",
	})
	require.NoError(t, err)

	bats := []*fleet.HostBattery{
		{HostID: host.ID, SerialNumber: "a", CycleCount: 1, Health: "Good"},
		{HostID: host.ID, SerialNumber: "b", CycleCount: 1002, Health: "Poor"},
	}
	require.NoError(t, s.ds.ReplaceHostBatteries(context.Background(), host.ID, bats))

	var getHostResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.Equal(t, host.ID, getHostResp.Host.ID)
	// only cycle count and health are returned
	require.ElementsMatch(t, []*fleet.HostBattery{
		{CycleCount: 1, Health: "Normal"},
		{CycleCount: 1002, Health: "Replacement recommended"},
	}, *getHostResp.Host.Batteries)

	// same for get host by identifier
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/identifier/%s", *host.NodeKey), nil, http.StatusOK, &getHostResp)
	require.Equal(t, host.ID, getHostResp.Host.ID)
	// only cycle count and health are returned
	require.ElementsMatch(t, []*fleet.HostBattery{
		{CycleCount: 1, Health: "Normal"},
		{CycleCount: 1002, Health: "Replacement recommended"},
	}, *getHostResp.Host.Batteries)
}

func (s *integrationTestSuite) TestHostByIdentifierSoftwareUpdatedAt() {
	t := s.T()

	host, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String(strings.ReplaceAll(t.Name(), "/", "_") + "1"),
		UUID:            t.Name() + "1",
		Hostname:        t.Name() + "foo.local",
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-58",
	})
	require.NoError(t, err)

	var getHostResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/identifier/%s", *host.NodeKey), nil, http.StatusOK, &getHostResp)
	require.Equal(t, host.ID, getHostResp.Host.ID)
	require.Equal(t, getHostResp.Host.SoftwareUpdatedAt, getHostResp.Host.CreatedAt)

	time.Sleep(1 * time.Second)

	software := []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
	}
	_, err = s.ds.UpdateHostSoftware(context.Background(), host.ID, software)
	require.NoError(t, err)

	getHostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/identifier/%s", *host.NodeKey), nil, http.StatusOK, &getHostResp)
	require.Greater(t, getHostResp.Host.SoftwareUpdatedAt, getHostResp.Host.CreatedAt)
}

func (s *integrationTestSuite) TestGetHostDiskEncryption() {
	t := s.T()

	// create Windows, mac and Linux hosts
	hostWin, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String(strings.ReplaceAll(t.Name(), "/", "_") + "1"),
		OsqueryHostID:   ptr.String(strings.ReplaceAll(t.Name(), "/", "_") + "1"),
		UUID:            t.Name() + "1",
		Hostname:        t.Name() + "foo.local",
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-58",
		Platform:        "windows",
	})
	require.NoError(t, err)

	hostMac, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String(strings.ReplaceAll(t.Name(), "/", "_") + "2"),
		OsqueryHostID:   ptr.String(strings.ReplaceAll(t.Name(), "/", "_") + "2"),
		UUID:            t.Name() + "2",
		Hostname:        t.Name() + "foo2.local",
		PrimaryIP:       "192.168.1.2",
		PrimaryMac:      "30-65-EC-6F-C4-59",
		Platform:        "darwin",
	})
	require.NoError(t, err)

	hostLin, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String(strings.ReplaceAll(t.Name(), "/", "_") + "3"),
		OsqueryHostID:   ptr.String(strings.ReplaceAll(t.Name(), "/", "_") + "3"),
		UUID:            t.Name() + "3",
		Hostname:        t.Name() + "foo3.local",
		PrimaryIP:       "192.168.1.3",
		PrimaryMac:      "30-65-EC-6F-C4-60",
		Platform:        "linux",
	})
	require.NoError(t, err)

	// before any disk encryption is received, all hosts report NULL (even if
	// some have disk space information, i.e. an entry exists in host_disks).
	require.NoError(t, s.ds.SetOrUpdateHostDisksSpace(context.Background(), hostWin.ID, 44.5, 55.6))

	var getHostResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", hostWin.ID), nil, http.StatusOK, &getHostResp)
	require.Equal(t, hostWin.ID, getHostResp.Host.ID)
	require.Nil(t, getHostResp.Host.DiskEncryptionEnabled)

	getHostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", hostMac.ID), nil, http.StatusOK, &getHostResp)
	require.Equal(t, hostMac.ID, getHostResp.Host.ID)
	require.Nil(t, getHostResp.Host.DiskEncryptionEnabled)

	getHostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", hostLin.ID), nil, http.StatusOK, &getHostResp)
	require.Equal(t, hostLin.ID, getHostResp.Host.ID)
	require.Nil(t, getHostResp.Host.DiskEncryptionEnabled)

	// set encrypted for all hosts
	require.NoError(t, s.ds.SetOrUpdateHostDisksEncryption(context.Background(), hostWin.ID, true))
	require.NoError(t, s.ds.SetOrUpdateHostDisksEncryption(context.Background(), hostMac.ID, true))
	require.NoError(t, s.ds.SetOrUpdateHostDisksEncryption(context.Background(), hostLin.ID, true))

	getHostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", hostWin.ID), nil, http.StatusOK, &getHostResp)
	require.Equal(t, hostWin.ID, getHostResp.Host.ID)
	require.True(t, *getHostResp.Host.DiskEncryptionEnabled)

	getHostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", hostMac.ID), nil, http.StatusOK, &getHostResp)
	require.Equal(t, hostMac.ID, getHostResp.Host.ID)
	require.True(t, *getHostResp.Host.DiskEncryptionEnabled)

	getHostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", hostLin.ID), nil, http.StatusOK, &getHostResp)
	require.Equal(t, hostLin.ID, getHostResp.Host.ID)
	require.True(t, *getHostResp.Host.DiskEncryptionEnabled)

	// set unencrypted for all hosts
	require.NoError(t, s.ds.SetOrUpdateHostDisksEncryption(context.Background(), hostWin.ID, false))
	require.NoError(t, s.ds.SetOrUpdateHostDisksEncryption(context.Background(), hostMac.ID, false))
	require.NoError(t, s.ds.SetOrUpdateHostDisksEncryption(context.Background(), hostLin.ID, false))

	getHostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", hostWin.ID), nil, http.StatusOK, &getHostResp)
	require.Equal(t, hostWin.ID, getHostResp.Host.ID)
	require.False(t, *getHostResp.Host.DiskEncryptionEnabled)

	getHostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", hostMac.ID), nil, http.StatusOK, &getHostResp)
	require.Equal(t, hostMac.ID, getHostResp.Host.ID)
	require.False(t, *getHostResp.Host.DiskEncryptionEnabled)

	// Linux does not return false, it omits the field when false
	getHostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", hostLin.ID), nil, http.StatusOK, &getHostResp)
	require.Equal(t, hostLin.ID, getHostResp.Host.ID)
	require.Nil(t, getHostResp.Host.DiskEncryptionEnabled)
}

func (s *integrationTestSuite) TestOSVersions() {
	t := s.T()

	testOS := fleet.OperatingSystem{Name: "barOS", Version: "4.2", Arch: "64bit", KernelVersion: "13.37", Platform: "foo"}

	hosts := s.createHosts(t)

	var resp listHostsResponse
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp)
	require.Len(t, resp.Hosts, len(hosts))

	// set operating system information on a host
	require.NoError(t, s.ds.UpdateHostOperatingSystem(context.Background(), hosts[0].ID, testOS))
	var osID uint
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), q, &osID,
			`SELECT id FROM operating_systems WHERE name = ? AND version = ? AND arch = ? AND kernel_version = ? AND platform = ?`,
			testOS.Name, testOS.Version, testOS.Arch, testOS.KernelVersion, testOS.Platform)
	})
	require.Greater(t, osID, uint(0))

	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "os_name", testOS.Name, "os_version", testOS.Version)
	require.Len(t, resp.Hosts, 1)

	expected := resp.Hosts[0]
	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "os_id", fmt.Sprintf("%d", osID))
	require.Len(t, resp.Hosts, 1)
	require.Equal(t, expected, resp.Hosts[0])

	// generate aggregated stats
	require.NoError(t, s.ds.UpdateOSVersions(context.Background()))

	var osVersionsResp osVersionsResponse
	s.DoJSON("GET", "/api/latest/fleet/os_versions", nil, http.StatusOK, &osVersionsResp)
	require.Len(t, osVersionsResp.OSVersions, 1)
	require.Equal(t, fleet.OSVersion{HostsCount: 1, Name: fmt.Sprintf("%s %s", testOS.Name, testOS.Version), NameOnly: testOS.Name, Version: testOS.Version, Platform: testOS.Platform}, osVersionsResp.OSVersions[0])
}

func (s *integrationTestSuite) TestPingEndpoints() {
	s.DoRaw("HEAD", "/api/fleet/orbit/ping", nil, http.StatusOK)
	// unauthenticated works too
	s.DoRawNoAuth("HEAD", "/api/fleet/orbit/ping", nil, http.StatusOK)

	s.DoRaw("HEAD", "/api/fleet/device/ping", nil, http.StatusOK)
	// unauthenticated works too
	s.DoRawNoAuth("HEAD", "/api/fleet/device/ping", nil, http.StatusOK)
}

func (s *integrationTestSuite) TestAppleMDMNotConfigured() {
	t := s.T()

	// create a host with device token to test device authenticated routes
	tkn := "D3V1C370K3N"
	createHostAndDeviceToken(t, s.ds, tkn)

	for _, route := range mdmAppleConfigurationRequiredEndpoints() {
		which := fmt.Sprintf("%s %s", route.method, route.path)
		log.Print(which)
		var expectedErr fleet.ErrWithStatusCode = fleet.ErrMDMNotConfigured
		if route.premiumOnly && route.deviceAuthenticated {
			// user-authenticated premium-only routes will never see the ErrMissingLicense error
			// if mdm is not configured, as the MDM middleware will intercept and fail the call.
			expectedErr = fleet.ErrMissingLicense
		}
		path := route.path
		if route.deviceAuthenticated {
			path = fmt.Sprintf(route.path, tkn)
		}
		res := s.Do(route.method, path, nil, expectedErr.StatusCode())
		errMsg := extractServerErrorText(res.Body)
		assert.Contains(t, errMsg, expectedErr.Error(), which)
	}

	fleetdmSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Setenv("TEST_FLEETDM_API_URL", fleetdmSrv.URL)
	t.Cleanup(fleetdmSrv.Close)

	// Always accessible
	var reqCSRResp requestMDMAppleCSRResponse
	s.DoJSON("POST", "/api/latest/fleet/mdm/apple/request_csr", requestMDMAppleCSRRequest{EmailAddress: "a@b.c", Organization: "test"}, http.StatusOK, &reqCSRResp)
	s.Do("POST", "/api/latest/fleet/mdm/apple/dep/key_pair", nil, http.StatusOK)
}

func (s *integrationTestSuite) TestOrbitConfigNotifications() {
	t := s.T()
	ctx := context.Background()

	// set the enabled and configured flags,
	appCfg, err := s.ds.AppConfig(ctx)
	require.NoError(t, err)
	origEnabledAndConfigured := appCfg.MDM.EnabledAndConfigured
	appCfg.MDM.EnabledAndConfigured = true
	err = s.ds.SaveAppConfig(ctx, appCfg)
	require.NoError(t, err)
	defer func() {
		appCfg.MDM.EnabledAndConfigured = origEnabledAndConfigured
		err = s.ds.SaveAppConfig(ctx, appCfg)
		require.NoError(t, err)
	}()

	var resp orbitGetConfigResponse
	// missing orbit key
	s.DoJSON("POST", "/api/fleet/orbit/config", nil, http.StatusUnauthorized, &resp)

	hNoMDM := createOrbitEnrolledHost(t, "darwin", "nomdm", s.ds)
	resp = orbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *hNoMDM.OrbitNodeKey)), http.StatusOK, &resp)
	require.False(t, resp.Notifications.RenewEnrollmentProfile)

	hSimpleMDM := createOrbitEnrolledHost(t, "darwin", "simplemdm", s.ds)
	err = s.ds.SetOrUpdateMDMData(context.Background(), hSimpleMDM.ID, false, true, "https://simplemdm.com", false, fleet.WellKnownMDMSimpleMDM)
	require.NoError(t, err)
	resp = orbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *hSimpleMDM.OrbitNodeKey)), http.StatusOK, &resp)
	require.False(t, resp.Notifications.RenewEnrollmentProfile)

	// not yet assigned in ABM
	hFleetMDM := createOrbitEnrolledHost(t, "darwin", "fleetmdm", s.ds)
	err = s.ds.SetOrUpdateMDMData(context.Background(), hFleetMDM.ID, false, false, "https://fleetdm.com", true, fleet.WellKnownMDMFleet)
	require.NoError(t, err)

	resp = orbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *hFleetMDM.OrbitNodeKey)), http.StatusOK, &resp)
	require.False(t, resp.Notifications.RenewEnrollmentProfile)

	// simulate ABM assignment
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		insertAppConfigQuery := `INSERT INTO host_dep_assignments (host_id) VALUES (?)`
		_, err = q.ExecContext(context.Background(), insertAppConfigQuery, hFleetMDM.ID)
		return err
	})
	err = s.ds.SetOrUpdateMDMData(context.Background(), hSimpleMDM.ID, false, true, "https://simplemdm.com", false, fleet.WellKnownMDMSimpleMDM)
	require.NoError(t, err)
	resp = orbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *hFleetMDM.OrbitNodeKey)), http.StatusOK, &resp)
	require.True(t, resp.Notifications.RenewEnrollmentProfile)

	// if the fleet mdm host is fully enrolled (not pending anymore), then the notification is false
	err = s.ds.SetOrUpdateMDMData(context.Background(), hFleetMDM.ID, false, true, "https://fleetdm.com", true, fleet.WellKnownMDMFleet)
	require.NoError(t, err)
	resp = orbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *hFleetMDM.OrbitNodeKey)), http.StatusOK, &resp)
	require.False(t, resp.Notifications.RenewEnrollmentProfile)
}

func (s *integrationTestSuite) TestTryingToEnrollWithTheWrongSecret() {
	t := s.T()
	ctx := context.Background()

	h, err := s.ds.NewHost(ctx, &fleet.Host{
		HardwareSerial:   uuid.New().String(),
		Platform:         "darwin",
		LastEnrolledAt:   time.Now(),
		DetailUpdatedAt:  time.Now(),
		RefetchRequested: true,
	})
	require.NoError(t, err)

	var resp jsonError
	s.DoJSON("POST", "/api/fleet/orbit/enroll", EnrollOrbitRequest{
		EnrollSecret:   uuid.New().String(),
		HardwareUUID:   h.UUID,
		HardwareSerial: h.HardwareSerial,
	}, http.StatusUnauthorized, &resp)

	require.Equal(t, resp.Message, "Authentication failed")
}

func (s *integrationTestSuite) TestEnrollOrbitExistingHostNoSerialMatch() {
	t := s.T()
	ctx := context.Background()

	// create a host with minimal information and the serial, no uuid/osquery id
	// (as when created via DEP sync).
	dbZeroTime := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	h, err := s.ds.NewHost(ctx, &fleet.Host{
		HardwareSerial:   uuid.New().String(),
		Platform:         "darwin",
		LastEnrolledAt:   dbZeroTime,
		DetailUpdatedAt:  dbZeroTime,
		RefetchRequested: true,
	})
	require.NoError(t, err)

	// create an enroll secret
	secret := uuid.New().String()
	var applyResp applyEnrollSecretSpecResponse
	s.DoJSON("POST", "/api/latest/fleet/spec/enroll_secret", applyEnrollSecretSpecRequest{
		Spec: &fleet.EnrollSecretSpec{
			Secrets: []*fleet.EnrollSecret{{Secret: secret}},
		},
	}, http.StatusOK, &applyResp)

	// enroll the host from orbit, it will NOT match the existing host since MDM
	// is not configured (it will only look for a match by osquery_host_id with
	// the provided uuid).
	var resp EnrollOrbitResponse
	hostUUID := uuid.New().String()
	s.DoJSON("POST", "/api/fleet/orbit/enroll", EnrollOrbitRequest{
		EnrollSecret:   secret,
		HardwareUUID:   hostUUID, // will not match any existing host
		HardwareSerial: h.HardwareSerial,
	}, http.StatusOK, &resp)
	require.NotEmpty(t, resp.OrbitNodeKey)

	// fetch the host, it will NOT match the one created above
	orbitHost, err := s.ds.LoadHostByOrbitNodeKey(ctx, resp.OrbitNodeKey)
	require.NoError(t, err)
	require.NotEqual(t, h.ID, orbitHost.ID)

	// enroll the host from osquery, it should match the Orbit-enrolled host
	var osqueryResp enrollAgentResponse

	// NOTE(mna): using an osquery_host_id that is NOT the host's UUID would not work,
	// because we haven't enabled lookup by UUID due to not having an index and possible
	// side-effects of this on host ingestion performance. However, this should not happen
	// anyway in MDM-enabled environments as we will recommend using the UUID as osquery
	// host identifier.
	// See https://github.com/fleetdm/fleet/issues/9033#issuecomment-1411150758

	osqueryID := hostUUID

	s.DoJSON("POST", "/api/osquery/enroll", enrollAgentRequest{
		EnrollSecret:   secret,
		HostIdentifier: osqueryID,
		HostDetails: map[string]map[string]string{
			"system_info": {
				"uuid":            hostUUID,
				"hardware_serial": h.HardwareSerial,
			},
		},
	}, http.StatusOK, &osqueryResp)
	require.NotEmpty(t, osqueryResp.NodeKey)

	// load the host by osquery node key, should match the orbit host
	got, err := s.ds.LoadHostByNodeKey(ctx, osqueryResp.NodeKey)
	require.NoError(t, err)
	require.Equal(t, orbitHost.ID, got.ID)
}

// this test can be deleted once the "v1" version is removed.
func (s *integrationTestSuite) TestAPIVersion_v1_2022_04() {
	t := s.T()

	// create a query that can be scheduled
	qr, err := s.ds.NewQuery(context.Background(), &fleet.Query{
		Name:           "TestQuery2",
		Query:          "select * from osquery;",
		ObserverCanRun: true,
		Saved:          true,
	})
	require.NoError(t, err)

	// try to schedule that query on the endpoint that is deprecated
	// in that version
	gsParams := fleet.ScheduledQueryPayload{QueryID: ptr.Uint(qr.ID), Interval: ptr.Uint(42)}
	res := s.DoRaw("POST", "/api/2022-04/fleet/global/schedule", jsonMustMarshal(t, gsParams), http.StatusNotFound)
	res.Body.Close()
	// use the correct version for that deprecated API
	createResp := globalScheduleQueryResponse{}
	s.DoJSON("POST", "/api/v1/fleet/global/schedule", gsParams, http.StatusOK, &createResp)
	require.NotZero(t, createResp.Scheduled.ID)

	// list the scheduled queries with the new endpoint, but the old version
	res = s.DoRaw("GET", "/api/v1/fleet/schedule", nil, http.StatusMethodNotAllowed)
	res.Body.Close()

	// list again, this time with the correct version
	gs := fleet.GlobalSchedulePayload{}
	s.DoJSON("GET", "/api/2022-04/fleet/schedule", nil, http.StatusOK, &gs)
	require.Len(t, gs.GlobalSchedule, 1)

	// delete using the old endpoint but on the wrong new version
	res = s.DoRaw("DELETE", fmt.Sprintf("/api/2022-04/fleet/global/schedule/%d", createResp.Scheduled.ID), nil, http.StatusNotFound)
	res.Body.Close()
	// properly delete with old endpoint and old version
	var delResp deleteGlobalScheduleResponse
	s.DoJSON("DELETE", fmt.Sprintf("/api/v1/fleet/global/schedule/%d", createResp.Scheduled.ID), nil, http.StatusOK, &delResp)
}

type validationErrResp struct {
	Message string `json:"message"`
	Errors  []struct {
		Name   string `json:"name"`
		Reason string `json:"reason"`
	} `json:"errors"`
}

func createOrbitEnrolledHost(t *testing.T, os, suffix string, ds fleet.Datastore) *fleet.Host {
	name := t.Name() + suffix
	h, err := ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-time.Minute),
		OsqueryHostID:   ptr.String(name),
		NodeKey:         ptr.String(name),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%s.local", name),
		HardwareSerial:  uuid.New().String(),
		Platform:        os,
	})
	require.NoError(t, err)
	orbitKey := uuid.New().String()
	_, err = ds.EnrollOrbit(context.Background(), false, fleet.OrbitHostInfo{
		HardwareUUID:   *h.OsqueryHostID,
		HardwareSerial: h.HardwareSerial,
	}, orbitKey, nil)
	require.NoError(t, err)
	h.OrbitNodeKey = &orbitKey
	return h
}

// creates a session and returns it, its key is to be passed as authorization header.
func createSession(t *testing.T, uid uint, ds fleet.Datastore) *fleet.Session {
	key := make([]byte, 64)
	_, err := rand.Read(key)
	require.NoError(t, err)

	sessionKey := base64.StdEncoding.EncodeToString(key)
	ssn, err := ds.NewSession(context.Background(), uid, sessionKey)
	require.NoError(t, err)

	return ssn
}

func cleanupQuery(s *integrationTestSuite, queryID uint) {
	var delResp deleteQueryByIDResponse
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/queries/id/%d", queryID), nil, http.StatusOK, &delResp)
}

func jsonMustMarshal(t testing.TB, v interface{}) []byte {
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return b
}

// starts a test web server that mocks responses to requests to external
// services with a valid payload (if the request is valid) or a status code
// error. It returns the URL to use to make requests to that server.
//
// For Jira, the project keys "qux" and "qux2" are supported.
// For Zendesk, the group IDs "122" and "123" are supported.
//
// The basic auth's user (or password for Zendesk) "ok" means that auth is
// allowed, while "fail" means unauthorized and anything else results in status
// 502.
func startExternalServiceWebServer(t *testing.T) string {
	// create a test http server to act as the Jira and Zendesk server
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.WriteHeader(501)
			return
		}

		switch r.URL.Path {
		case "/rest/api/2/project/qux":
			switch usr, _, _ := r.BasicAuth(); usr {
			case "ok":
				_, err := w.Write([]byte(jiraProjectResponsePayload))
				require.NoError(t, err)
			case "fail":
				w.WriteHeader(http.StatusUnauthorized)
			default:
				w.WriteHeader(502)
			}
		case "/rest/api/2/project/qux2":
			switch usr, _, _ := r.BasicAuth(); usr {
			case "ok":
				_, err := w.Write([]byte(jiraProjectResponsePayload))
				require.NoError(t, err)
			case "fail":
				w.WriteHeader(http.StatusUnauthorized)
			default:
				w.WriteHeader(502)
			}
		case "/api/v2/groups/122.json":
			switch _, pwd, _ := r.BasicAuth(); pwd {
			case "ok":
				_, err := w.Write([]byte(`{"group": {"id": 122,"name": "test122"}}`))
				require.NoError(t, err)
			case "fail":
				w.WriteHeader(http.StatusUnauthorized)
			default:
				w.WriteHeader(502)
			}
		case "/api/v2/groups/123.json":
			switch _, pwd, _ := r.BasicAuth(); pwd {
			case "ok":
				_, err := w.Write([]byte(`{"group": {"id": 123,"name": "test123"}}`))
				require.NoError(t, err)
			case "fail":
				w.WriteHeader(http.StatusUnauthorized)
			default:
				w.WriteHeader(502)
			}
		default:
			w.WriteHeader(502)
		}
	}))
	t.Cleanup(srv.Close)

	return srv.URL
}

const (
	// example response from the Jira docs
	jiraProjectResponsePayload = `{
  "self": "https://your-domain.atlassian.net/rest/api/2/project/EX",
  "id": "10000",
  "key": "EX",
  "description": "This project was created as an example for REST.",
  "lead": {
    "self": "https://your-domain.atlassian.net/rest/api/2/user?accountId=5b10a2844c20165700ede21g",
    "key": "",
    "accountId": "5b10a2844c20165700ede21g",
    "accountType": "atlassian",
    "name": "",
    "avatarUrls": {
      "48x48": "https://avatar-management--avatars.server-location.prod.public.atl-paas.net/initials/MK-5.png?size=48&s=48",
      "24x24": "https://avatar-management--avatars.server-location.prod.public.atl-paas.net/initials/MK-5.png?size=24&s=24",
      "16x16": "https://avatar-management--avatars.server-location.prod.public.atl-paas.net/initials/MK-5.png?size=16&s=16",
      "32x32": "https://avatar-management--avatars.server-location.prod.public.atl-paas.net/initials/MK-5.png?size=32&s=32"
    },
    "displayName": "Mia Krystof",
    "active": false
  },
  "components": [
    {
      "self": "https://your-domain.atlassian.net/rest/api/2/component/10000",
      "id": "10000",
      "name": "Component 1",
      "description": "This is a Jira component",
      "lead": {
        "self": "https://your-domain.atlassian.net/rest/api/2/user?accountId=5b10a2844c20165700ede21g",
        "key": "",
        "accountId": "5b10a2844c20165700ede21g",
        "accountType": "atlassian",
        "name": "",
        "avatarUrls": {
          "48x48": "https://avatar-management--avatars.server-location.prod.public.atl-paas.net/initials/MK-5.png?size=48&s=48",
          "24x24": "https://avatar-management--avatars.server-location.prod.public.atl-paas.net/initials/MK-5.png?size=24&s=24",
          "16x16": "https://avatar-management--avatars.server-location.prod.public.atl-paas.net/initials/MK-5.png?size=16&s=16",
          "32x32": "https://avatar-management--avatars.server-location.prod.public.atl-paas.net/initials/MK-5.png?size=32&s=32"
        },
        "displayName": "Mia Krystof",
        "active": false
      },
      "assigneeType": "PROJECT_LEAD",
      "assignee": {
        "self": "https://your-domain.atlassian.net/rest/api/2/user?accountId=5b10a2844c20165700ede21g",
        "key": "",
        "accountId": "5b10a2844c20165700ede21g",
        "accountType": "atlassian",
        "name": "",
        "avatarUrls": {
          "48x48": "https://avatar-management--avatars.server-location.prod.public.atl-paas.net/initials/MK-5.png?size=48&s=48",
          "24x24": "https://avatar-management--avatars.server-location.prod.public.atl-paas.net/initials/MK-5.png?size=24&s=24",
          "16x16": "https://avatar-management--avatars.server-location.prod.public.atl-paas.net/initials/MK-5.png?size=16&s=16",
          "32x32": "https://avatar-management--avatars.server-location.prod.public.atl-paas.net/initials/MK-5.png?size=32&s=32"
        },
        "displayName": "Mia Krystof",
        "active": false
      },
      "realAssigneeType": "PROJECT_LEAD",
      "realAssignee": {
        "self": "https://your-domain.atlassian.net/rest/api/2/user?accountId=5b10a2844c20165700ede21g",
        "key": "",
        "accountId": "5b10a2844c20165700ede21g",
        "accountType": "atlassian",
        "name": "",
        "avatarUrls": {
          "48x48": "https://avatar-management--avatars.server-location.prod.public.atl-paas.net/initials/MK-5.png?size=48&s=48",
          "24x24": "https://avatar-management--avatars.server-location.prod.public.atl-paas.net/initials/MK-5.png?size=24&s=24",
          "16x16": "https://avatar-management--avatars.server-location.prod.public.atl-paas.net/initials/MK-5.png?size=16&s=16",
          "32x32": "https://avatar-management--avatars.server-location.prod.public.atl-paas.net/initials/MK-5.png?size=32&s=32"
        },
        "displayName": "Mia Krystof",
        "active": false
      },
      "isAssigneeTypeValid": false,
      "project": "HSP",
      "projectId": 10000
    }
  ],
  "issueTypes": [
    {
      "self": "https://your-domain.atlassian.net/rest/api/2/issueType/3",
      "id": "3",
      "description": "A task that needs to be done.",
      "iconUrl": "https://your-domain.atlassian.net/secure/viewavatar?size=xsmall&avatarId=10299&avatarType=issuetype\",",
      "name": "Task",
      "subtask": false,
      "avatarId": 1,
      "hierarchyLevel": 0
    },
    {
      "self": "https://your-domain.atlassian.net/rest/api/2/issueType/1",
      "id": "1",
      "description": "A problem with the software.",
      "iconUrl": "https://your-domain.atlassian.net/secure/viewavatar?size=xsmall&avatarId=10316&avatarType=issuetype\",",
      "name": "Bug",
      "subtask": false,
      "avatarId": 10002,
      "entityId": "9d7dd6f7-e8b6-4247-954b-7b2c9b2a5ba2",
      "hierarchyLevel": 0,
      "scope": {
        "type": "PROJECT",
        "project": {
          "id": "10000",
          "key": "KEY",
          "name": "Next Gen Project"
        }
      }
    }
  ],
  "url": "https://www.example.com",
  "email": "from-jira@example.com",
  "assigneeType": "PROJECT_LEAD",
  "versions": [],
  "name": "Example",
  "roles": {
    "Developers": "https://your-domain.atlassian.net/rest/api/2/project/EX/role/10000"
  },
  "avatarUrls": {
    "48x48": "https://your-domain.atlassian.net/secure/projectavatar?size=large&pid=10000",
    "24x24": "https://your-domain.atlassian.net/secure/projectavatar?size=small&pid=10000",
    "16x16": "https://your-domain.atlassian.net/secure/projectavatar?size=xsmall&pid=10000",
    "32x32": "https://your-domain.atlassian.net/secure/projectavatar?size=medium&pid=10000"
  },
  "projectCategory": {
    "self": "https://your-domain.atlassian.net/rest/api/2/projectCategory/10000",
    "id": "10000",
    "name": "FIRST",
    "description": "First Project Category"
  },
  "simplified": false,
  "style": "classic",
  "properties": {
    "propertyKey": "propertyValue"
  },
  "insight": {
    "totalIssueCount": 100,
    "lastIssueUpdateTime": "2022-04-05T04:51:35.670+0000"
  }
}`
)
