package service

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/WatchBeam/clock"
	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/live_query/live_query_mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/service/async"
	"github.com/fleetdm/fleet/v4/server/service/osquery_utils"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/ghodss/yaml"
	"github.com/go-kit/log"
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
	testingSuite.withServer.s = &testingSuite.Suite
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
			EnableCachedDS:   true,
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

	assertBodyContains(t, respSecond, `Error 1062`)
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

func (s *integrationTestSuite) TestUserPasswordLengthValidation() {
	params := fleet.UserPayload{
		Name:  ptr.String("user_invalid_email"),
		Email: ptr.String("test@example.com"),
		// This is 73 characters long
		Password:   ptr.String("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaX@1"),
		GlobalRole: ptr.String(fleet.RoleObserver),
	}

	resp := s.Do("POST", "/api/latest/fleet/users/admin", &params, http.StatusUnprocessableEntity)
	assertBodyContains(s.T(), resp, "Could not create user. Password is over the 48 characters limit. If the password is under 48 characters, please check the auth_salt_key_size in your Fleet server config.")
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
				ID: 9999, // non-existent team
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
	assertBodyContains(t, resp, `team with id 9999 does not exist`)
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
	defer s.cleanupQuery(createQueryResp.Query.ID)

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

func (s *integrationTestSuite) TestCreatingAPIOnlyUserReturnsAPIToken() {
	t := s.T()

	defer func() {
		s.token = s.getTestAdminToken()
	}()

	var createResp createUserResponse
	params := fleet.UserPayload{
		Name:       ptr.String("someadmin"),
		Email:      ptr.String("someadmin@example.com"),
		Password:   ptr.String(test.GoodPassword),
		GlobalRole: ptr.String(fleet.RoleAdmin),
		APIOnly:    ptr.Bool(false),
	}
	s.DoJSON("POST", "/api/latest/fleet/users/admin", params, http.StatusOK, &createResp)
	assert.NotZero(t, createResp.User.ID)
	assert.Nil(t, createResp.Token)

	params = fleet.UserPayload{
		Name:       ptr.String("apionly"),
		Email:      ptr.String("apionly@example.com"),
		Password:   ptr.String(test.GoodPassword),
		GlobalRole: ptr.String(fleet.RoleObserver),
		APIOnly:    ptr.Bool(true),
		// AdminForcedPasswordReset is set to false when creating api-only users via `fleetctl user create --api-only`.
		AdminForcedPasswordReset: ptr.Bool(false),
	}
	s.DoJSON("POST", "/api/latest/fleet/users/admin", params, http.StatusOK, &createResp)
	assert.NotZero(t, createResp.User.ID)
	assert.NotNil(t, createResp.Token)

	s.token = *createResp.Token
	var chr countHostsResponse
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", countHostsRequest{}, http.StatusOK, &chr)
	assert.Equal(t, 0, chr.Count)
}

func (s *integrationTestSuite) TestActivityUserEmailPersistsAfterDeletion() {
	t := s.T()

	// create a new user
	var createResp createUserResponse
	userRawPwd := test.GoodPassword
	params := fleet.UserPayload{
		Name:       ptr.String("Gonna B Deleted"),
		Email:      ptr.String("goingto@delete.com"),
		Password:   ptr.String(userRawPwd),
		GlobalRole: ptr.String(fleet.RoleObserver),
	}
	s.DoJSON("POST", "/api/latest/fleet/users/admin", params, http.StatusOK, &createResp)
	assert.NotZero(t, createResp.User.ID)
	assert.True(t, createResp.User.AdminForcedPasswordReset)
	u := *createResp.User

	var loginResp loginResponse
	s.DoJSON("POST", "/api/latest/fleet/login", params, http.StatusOK, &loginResp)
	require.Equal(t, loginResp.User.ID, u.ID)

	activities := listActivitiesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &activities)

	assert.GreaterOrEqual(t, len(activities.Activities), 1)
	found := false
	for _, activity := range activities.Activities {
		if activity.Type == "user_logged_in" && *activity.ActorFullName == u.Name {
			found = true
			assert.Equal(t, u.Email, *activity.ActorEmail)
		}
	}
	require.True(t, found)

	err := s.ds.DeleteUser(context.Background(), u.ID)
	require.NoError(t, err)

	activities = listActivitiesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &activities)

	assert.GreaterOrEqual(t, len(activities.Activities), 1)
	found = false
	for _, activity := range activities.Activities {
		if activity.Type == "user_logged_in" && *activity.ActorFullName == u.Name {
			found = true
			assert.Equal(t, u.Email, *activity.ActorEmail)
		}
	}
	require.True(t, found)

	// ensure that on exit, the admin token is used
	s.token = s.getTestAdminToken()
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
		Logging:        fleet.LoggingSnapshot,
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

	// empty body
	s.DoJSON("POST", "/api/latest/fleet/translate", &translatorRequest{}, http.StatusBadRequest, &payload)

	s.DoJSON("POST", "/api/latest/fleet/translate", &translatorRequest{List: []fleet.TranslatePayload{{Type: "notavalidtype", Payload: fleet.StringIdentifierToIDPayload{}}}}, http.StatusBadRequest, &payload)
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
		OSVersion:       "Mac OS X 10.14.6",
	})
	require.NoError(t, err)
	require.NotNil(t, host)

	software := []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "chrome_extensions", ExtensionID: "abc", Browser: "edge"},
		{Name: "bar", Version: "0.0.3", Source: "apps", ExtensionID: "xyz", Browser: "chrome"},
		{Name: "baz", Version: "0.0.4", Source: "apps"},
	}
	_, err = s.ds.UpdateHostSoftware(context.Background(), host.ID, software)
	require.NoError(t, err)
	require.NoError(t, s.ds.LoadHostSoftware(context.Background(), host, false))

	soft1 := host.Software[0]
	for _, item := range host.Software {
		if item.Name == "bar" {
			soft1 = item
			break
		}
	}

	cpes := []fleet.SoftwareCPE{{SoftwareID: soft1.ID, CPE: "somecpe"}}
	_, err = s.ds.UpsertSoftwareCPEs(context.Background(), cpes)
	require.NoError(t, err)

	// Reload software so that 'GeneratedCPEID is set.
	require.NoError(t, s.ds.LoadHostSoftware(context.Background(), host, false))
	soft1 = host.Software[0]
	for _, item := range host.Software {
		if item.Name == "bar" {
			soft1 = item
			break
		}
	}

	inserted, err := s.ds.InsertSoftwareVulnerability(
		context.Background(), fleet.SoftwareVulnerability{
			SoftwareID: soft1.ID,
			CVE:        "cve-123-123-132",
		}, fleet.NVDSource,
	)
	require.NoError(t, err)
	require.True(t, inserted)

	var hostResponse getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &hostResponse)

	assertSoftware := func(t *testing.T, software []fleet.HostSoftwareEntry, contains *fleet.Software) {
		t.Helper()
		var found bool
		for _, s := range software {
			if s.Name == contains.Name {
				found = true
				assert.Equal(t, s.Name, contains.Name)
				assert.Equal(t, s.Version, contains.Version)
				assert.Equal(t, s.Source, contains.Source)
				assert.Equal(t, s.ExtensionID, contains.ExtensionID)
				assert.Equal(t, s.Browser, contains.Browser)
				assert.Equal(t, s.GenerateCPE, contains.GenerateCPE)
				assert.Len(t, contains.Vulnerabilities, len(s.Vulnerabilities))
				for i, vuln := range s.Vulnerabilities {
					assert.Equal(t, vuln.CVE, contains.Vulnerabilities[i].CVE)
					assert.Equal(t, vuln.DetailsLink, contains.Vulnerabilities[i].DetailsLink)
				}
			}
		}
		if !found {
			t.Fatalf("software not found")
		}
	}

	expectedSoft2 := &fleet.Software{
		Name:        "bar",
		Version:     "0.0.3",
		Source:      "apps",
		ExtensionID: "xyz",
		Browser:     "chrome",
		GenerateCPE: "somecpe",
		Vulnerabilities: fleet.Vulnerabilities{
			{
				CVE:         "cve-123-123-132",
				DetailsLink: "https://nvd.nist.gov/vuln/detail/cve-123-123-132",
			},
		},
	}

	expectedSoft1 := &fleet.Software{
		Name:            "foo",
		Version:         "0.0.1",
		Source:          "chrome_extensions",
		ExtensionID:     "abc",
		Browser:         "edge",
		GenerateCPE:     "",
		Vulnerabilities: nil,
	}

	assertSoftware(t, hostResponse.Host.Software, expectedSoft1)
	assertSoftware(t, hostResponse.Host.Software, expectedSoft2)

	// no software host counts have been calculated yet, so this returns nothing
	var lsResp listSoftwareResponse
	resp := s.Do("GET", "/api/latest/fleet/software", nil, http.StatusOK, "vulnerable", "true", "order_key", "generated_cpe", "order_direction", "desc")
	bodyBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(bodyBytes), `"counts_updated_at": null`)
	require.NoError(t, json.Unmarshal(bodyBytes, &lsResp))
	require.Len(t, lsResp.Software, 0)
	assert.Nil(t, lsResp.CountsUpdatedAt)

	var versionsResp listSoftwareVersionsResponse
	resp = s.Do("GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, "vulnerable", "true", "order_key", "generated_cpe", "order_direction", "desc")
	bodyBytes, err = io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(bodyBytes), `"counts_updated_at": null`)
	require.NoError(t, json.Unmarshal(bodyBytes, &versionsResp))
	require.Len(t, versionsResp.Software, 0)
	require.Equal(t, 0, versionsResp.Count)
	assert.Nil(t, versionsResp.CountsUpdatedAt)

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
	assert.Equal(t, soft1.ExtensionID, lsResp.Software[0].ExtensionID)
	assert.Equal(t, soft1.Browser, lsResp.Software[0].Browser)
	assert.Len(t, lsResp.Software[0].Vulnerabilities, 1)
	require.NotNil(t, lsResp.CountsUpdatedAt)
	assert.WithinDuration(t, hostsCountTs, *lsResp.CountsUpdatedAt, time.Second)

	versionsResp = listSoftwareVersionsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, &versionsResp, "vulnerable", "true", "order_key", "generated_cpe", "order_direction", "desc")
	require.Len(t, versionsResp.Software, 1)
	require.Equal(t, 1, versionsResp.Count)
	assert.Equal(t, soft1.ID, versionsResp.Software[0].ID)
	assert.Equal(t, soft1.ExtensionID, versionsResp.Software[0].ExtensionID)
	assert.Equal(t, soft1.Browser, versionsResp.Software[0].Browser)
	assert.Len(t, versionsResp.Software[0].Vulnerabilities, 1)
	require.NotNil(t, versionsResp.CountsUpdatedAt)
	assert.WithinDuration(t, hostsCountTs, *versionsResp.CountsUpdatedAt, time.Second)

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

	versionsResp = listSoftwareVersionsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, &versionsResp)
	require.True(t, len(versionsResp.Software) >= len(software))
	require.True(t, versionsResp.Count >= len(software))
	require.NotNil(t, versionsResp.CountsUpdatedAt)
	assert.WithinDuration(t, hostsCountTs, *versionsResp.CountsUpdatedAt, time.Second)

	// request with a per_page limit (see #4058)
	lsResp = listSoftwareResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software", nil, http.StatusOK, &lsResp, "page", "0", "per_page", "2", "order_key", "hosts_count", "order_direction", "desc")
	require.Len(t, lsResp.Software, 2)
	require.NotNil(t, lsResp.CountsUpdatedAt)
	assert.WithinDuration(t, hostsCountTs, *lsResp.CountsUpdatedAt, time.Second)

	versionsResp = listSoftwareVersionsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, &versionsResp, "page", "0", "per_page", "2", "order_key", "hosts_count", "order_direction", "desc")
	require.Len(t, versionsResp.Software, 2)
	require.True(t, versionsResp.Count >= 2)
	require.NotNil(t, versionsResp.CountsUpdatedAt)
	assert.WithinDuration(t, hostsCountTs, *versionsResp.CountsUpdatedAt, time.Second)

	// request next page, with per_page limit
	lsResp = listSoftwareResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software", nil, http.StatusOK, &lsResp, "per_page", "2", "page", "1", "order_key", "hosts_count", "order_direction", "desc")
	require.Len(t, lsResp.Software, 1)
	require.NotNil(t, lsResp.CountsUpdatedAt)
	assert.WithinDuration(t, hostsCountTs, *lsResp.CountsUpdatedAt, time.Second)

	versionsResp = listSoftwareVersionsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, &versionsResp, "per_page", "2", "page", "1", "order_key", "hosts_count", "order_direction", "desc")
	require.Len(t, versionsResp.Software, 1)
	require.True(t, versionsResp.Count >= 2)
	require.NotNil(t, versionsResp.CountsUpdatedAt)
	assert.WithinDuration(t, hostsCountTs, *versionsResp.CountsUpdatedAt, time.Second)

	// request one past the last page
	lsResp = listSoftwareResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software", nil, http.StatusOK, &lsResp, "per_page", "2", "page", "2", "order_key", "hosts_count", "order_direction", "desc")
	require.Len(t, lsResp.Software, 0)
	require.Nil(t, lsResp.CountsUpdatedAt)

	versionsResp = listSoftwareVersionsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, &versionsResp, "per_page", "2", "page", "2", "order_key", "hosts_count", "order_direction", "desc")
	require.Len(t, versionsResp.Software, 0)
	require.True(t, versionsResp.Count >= 2)
	require.Nil(t, versionsResp.CountsUpdatedAt) // CONFIRM: legacy counts updated at is calculated by the server based on the software entries in the paginated response so how should we handle now?

	s.DoJSON("GET", "/api/latest/fleet/software", nil, http.StatusBadRequest, &lsResp, "per_page", "2", "page", "-10")
	s.DoJSON("GET", "/api/latest/fleet/software/versions", nil, http.StatusBadRequest, &lsResp, "per_page", "-2", "page", "2")
	s.DoJSON("GET", "/api/latest/fleet/software/count", nil, http.StatusBadRequest, &lsResp, "per_page", "-2", "page", "2")
}

func (s *integrationTestSuite) TestGlobalPolicies() {
	t := s.T()

	// create 3 hosts
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
		Logging:        fleet.LoggingSnapshot,
	})
	require.NoError(t, err)

	// create a global policy
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

	// list global policies
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

	// count global policies
	cGPRes := countGlobalPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/policies/count", nil, http.StatusOK, &cGPRes)
	assert.Equal(t, 1, cGPRes.Count)

	// count global policies with matching search query
	cGPRes = countGlobalPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/policies/count", nil, http.StatusOK, &cGPRes, "query", "estQue")
	assert.Equal(t, 1, cGPRes.Count)

	// count global policies with matching search query containing leading/trailing whitespace
	cGPRes = countGlobalPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/policies/count", nil, http.StatusOK, &cGPRes, "query", " estQue    ")
	assert.Equal(t, 1, cGPRes.Count)

	// count global policies with non-matching search query
	cGPRes = countGlobalPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/policies/count", nil, http.StatusOK, &cGPRes, "query", "Query4")
	assert.Equal(t, 0, cGPRes.Count)

	// delete the policy
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
		Filters: &map[string]interface{}{"team_id": float64(team1.ID)},
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
		Filters: &map[string]interface{}{"label_id": float64(label.ID)},
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

func (s *integrationTestSuite) TestBulkDeleteHostByIDsWithTimeout() {
	t := s.T()

	hosts := s.createHosts(t, "debian")

	req := deleteHostsRequest{
		IDs: []uint{hosts[0].ID},
	}
	resp := deleteHostsResponse{}
	originalTimeout := deleteHostsTimeout
	deleteHostsTimeout = 0
	deleteHostsSkipAuthorization = true
	defer func() {
		deleteHostsTimeout = originalTimeout
		deleteHostsSkipAuthorization = false
	}()
	s.DoJSON("POST", "/api/latest/fleet/hosts/delete", req, http.StatusAccepted, &resp)

	// Make sure the host was actually deleted.
	deleteDone := make(chan bool)
	go func() {
		for {
			_, err := s.ds.Host(context.Background(), hosts[0].ID)
			if err != nil {
				deleteDone <- true
				break
			}
		}
	}()
	select {
	case <-deleteDone:
		return
	case <-time.After(2 * time.Second):
		t.Log("http.StatusAccepted (202) means that delete should continue in the background, but we did not see the host deleted after 2 seconds.")
		t.Error("Timeout: delete did not occur.")
	}
}

func (s *integrationTestSuite) TestBulkDeleteHostsAll() {
	t := s.T()

	hosts := s.createHosts(t)

	// All hosts should be deleted when an empty filter is specified
	req := deleteHostsRequest{
		Filters: &map[string]interface{}{},
	}
	resp := deleteHostsResponse{}
	s.DoJSON("POST", "/api/latest/fleet/hosts/delete", req, http.StatusOK, &resp)

	_, err := s.ds.Host(context.Background(), hosts[0].ID)
	require.Error(t, err)
	_, err = s.ds.Host(context.Background(), hosts[1].ID)
	require.Error(t, err)
	_, err = s.ds.Host(context.Background(), hosts[2].ID)
	require.Error(t, err)
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
		IDs:     []uint{hosts[0].ID, hosts[1].ID},
		Filters: &map[string]interface{}{"label_id": float64(1)},
	}
	resp := deleteHostsResponse{}
	s.DoJSON("POST", "/api/latest/fleet/hosts/delete", req, http.StatusBadRequest, &resp)

	req = deleteHostsRequest{}
	// No ids or filter specified
	s.DoJSON("POST", "/api/latest/fleet/hosts/delete", req, http.StatusBadRequest, &resp)
}

func (s *integrationTestSuite) TestHostsCount() {
	t := s.T()

	hosts := s.createHosts(t, "darwin", "darwin", "darwin")

	// set disk space information for some hosts
	require.NoError(t, s.ds.SetOrUpdateHostDisksSpace(context.Background(), hosts[0].ID, 10.0, 2.0, 500.0))  // low disk
	require.NoError(t, s.ds.SetOrUpdateHostDisksSpace(context.Background(), hosts[1].ID, 40.0, 4.0, 1000.0)) // not low disk

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

	// there are 3 hosts, whos names end with ...local0, ...local1, ...local2
	// query by host name

	req = countHostsRequest{}
	resp = countHostsResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/hosts/count", req, http.StatusOK, &resp,
		"query", "local0",
	)
	assert.Equal(t, 1, resp.Count)

	req = countHostsRequest{}
	resp = countHostsResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/hosts/count", req, http.StatusOK, &resp,
		"query", "local",
	)
	assert.Equal(t, 3, resp.Count)

	// query by host name with leading/trailing whitespace
	req = countHostsRequest{}
	resp = countHostsResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/hosts/count", req, http.StatusOK, &resp,
		"query", " local0  ",
	)
	assert.Equal(t, 1, resp.Count)

	req = countHostsRequest{}
	resp = countHostsResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/hosts/count", req, http.StatusOK, &resp,
		"query", " local  ",
	)
	assert.Equal(t, 3, resp.Count)

	// query by host name leading/trailing whitespace and label
	req = countHostsRequest{}
	resp = countHostsResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/hosts/count", req, http.StatusOK, &resp,
		"label_id", fmt.Sprint(label.ID),
		"query", "   local0	",
	)
	assert.Equal(t, 1, resp.Count)

	req = countHostsRequest{}
	resp = countHostsResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/hosts/count", req, http.StatusOK, &resp,
		"label_id", fmt.Sprint(label.ID),
		// only host 0 has the label
		"query", "   local1	",
	)
	assert.Equal(t, 0, resp.Count)

	// filter by low_disk_space criteria is ignored (premium-only filter)
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &resp, "low_disk_space", "32")
	require.Equal(t, len(hosts), resp.Count)
	// but it is still validated for a correct value when provided (as that happens in a middleware before the handler)
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusBadRequest, &resp, "low_disk_space", "123456")

	// filter by MDM criteria without any host having such information
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &resp, "mdm_id", fmt.Sprint(999))
	require.Equal(t, 0, resp.Count)
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &resp, "mdm_enrollment_status", "manual")
	require.Equal(t, 0, resp.Count)

	// set MDM information on a host
	require.NoError(t, s.ds.SetOrUpdateMDMData(context.Background(), hosts[1].ID, false, true, "https://simplemdm.com", false, fleet.WellKnownMDMSimpleMDM, ""))
	// also create server with MDM information, which is ignored.
	require.NoError(t, s.ds.SetOrUpdateMDMData(context.Background(), hosts[2].ID, true, true, "https://simplemdm.com", false, fleet.WellKnownMDMSimpleMDM, ""))
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
	require.NoError(t, s.ds.SetOrUpdateMDMData(context.Background(), pendingMDMHost.ID, false, false, "https://fleetdm.com", true, fleet.WellKnownMDMFleet, ""))

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
	require.NoError(t, s.ds.SetOrUpdateHostDisksSpace(context.Background(), hosts[0].ID, 10.0, 2.0, 500.0))  // low disk
	require.NoError(t, s.ds.SetOrUpdateHostDisksSpace(context.Background(), hosts[1].ID, 40.0, 4.0, 1000.0)) // not low disk

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

	// create some software for various hosts
	host2 := hosts[2]
	software := []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
	}
	_, err := s.ds.UpdateHostSoftware(context.Background(), host2.ID, software)
	require.NoError(t, err)
	require.NoError(t, s.ds.LoadHostSoftware(context.Background(), host2, false))

	host1 := hosts[1]
	software = []fleet.Software{
		{Name: "foo", Version: "0.0.2", Source: "chrome_extensions"},
		{Name: "bar", Version: "0.1.0", Source: "application"},
	}
	_, err = s.ds.UpdateHostSoftware(context.Background(), host1.ID, software)
	require.NoError(t, err)
	require.NoError(t, s.ds.LoadHostSoftware(context.Background(), host1, false))

	host0 := hosts[0]
	software = []fleet.Software{
		{Name: "foo", Version: "0.0.2", Source: "chrome_extensions"},
		{Name: "bar", Version: "0.2.0", Source: "not_application"},
	}
	_, err = s.ds.UpdateHostSoftware(context.Background(), host0.ID, software)
	require.NoError(t, err)
	require.NoError(t, s.ds.LoadHostSoftware(context.Background(), host0, false))

	err = s.ds.SyncHostsSoftware(context.Background(), time.Now())
	require.NoError(t, err)
	err = s.ds.ReconcileSoftwareTitles(context.Background())
	require.NoError(t, err)
	err = s.ds.SyncHostsSoftwareTitles(context.Background(), time.Now())
	require.NoError(t, err)

	var fooV1ID, fooV2ID, barAppTitleID, fooTitleID uint
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		err := sqlx.GetContext(context.Background(), q, &fooV1ID,
			`SELECT id FROM software WHERE name = ? AND source = ? AND version = ?`, "foo", "chrome_extensions", "0.0.1")
		if err != nil {
			return err
		}
		err = sqlx.GetContext(context.Background(), q, &fooV2ID,
			`SELECT id FROM software WHERE name = ? AND source = ? AND version = ?`, "foo", "chrome_extensions", "0.0.2")
		if err != nil {
			return err
		}
		err = sqlx.GetContext(context.Background(), q, &barAppTitleID,
			`SELECT id FROM software_titles WHERE name = ? AND source = ?`, "bar", "application")
		if err != nil {
			return err
		}
		err = sqlx.GetContext(context.Background(), q, &fooTitleID,
			`SELECT id FROM software_titles WHERE name = ? AND source = ?`, "foo", "chrome_extensions")
		if err != nil {
			return err
		}
		return nil
	})

	// foo v0.0.1 is only installed on host2
	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "software_id", fmt.Sprint(fooV1ID))
	require.Len(t, resp.Hosts, 1)
	assert.Equal(t, host2.ID, resp.Hosts[0].ID)
	assert.Equal(t, "foo", resp.Software.Name)
	assert.Greater(t, resp.Hosts[0].SoftwareUpdatedAt, resp.Hosts[0].CreatedAt)
	assert.Nil(t, resp.SoftwareTitle)

	var countResp countHostsResponse
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp, "software_id", fmt.Sprint(fooV1ID))
	require.Equal(t, 1, countResp.Count)

	// foo v0.0.2 is installed on hosts 0 and 1
	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "software_version_id", fmt.Sprint(fooV2ID))
	require.Len(t, resp.Hosts, 2)
	require.ElementsMatch(t, []uint{host0.ID, host1.ID}, []uint{resp.Hosts[0].ID, resp.Hosts[1].ID})

	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp, "software_version_id", fmt.Sprint(fooV2ID))
	require.Equal(t, 2, countResp.Count)

	// bar/application title is only on host1
	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "software_title_id", fmt.Sprint(barAppTitleID))
	require.Len(t, resp.Hosts, 1)
	require.ElementsMatch(t, []uint{host1.ID}, []uint{resp.Hosts[0].ID})
	assert.Equal(t, "bar", resp.SoftwareTitle.Name)
	assert.Equal(t, "application", resp.SoftwareTitle.Source)
	assert.Equal(t, uint(1), resp.SoftwareTitle.HostsCount)
	require.Len(t, resp.SoftwareTitle.Versions, 1)
	assert.Equal(t, "0.1.0", resp.SoftwareTitle.Versions[0].Version)
	assert.Nil(t, resp.Software)

	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp, "software_title_id", fmt.Sprint(barAppTitleID))
	require.Equal(t, 1, countResp.Count)

	// foo title is on all 3 hosts
	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "software_title_id", fmt.Sprint(fooTitleID))
	require.Len(t, resp.Hosts, 3)
	require.ElementsMatch(t, []uint{host0.ID, host1.ID, host2.ID}, []uint{resp.Hosts[0].ID, resp.Hosts[1].ID, resp.Hosts[2].ID})

	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp, "software_title_id", fmt.Sprint(fooTitleID))
	require.Equal(t, 3, countResp.Count)

	// verify invalid combinations of software filters
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusBadRequest, &resp, "software_title_id", fmt.Sprint(fooTitleID), "software_id", fmt.Sprint(fooV1ID))
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusBadRequest, &resp, "software_title_id", fmt.Sprint(fooTitleID), "software_version_id", fmt.Sprint(fooV1ID))
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusBadRequest, &resp, "software_id", fmt.Sprint(fooV1ID), "software_version_id", fmt.Sprint(fooV1ID))
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusBadRequest, &resp, "software_id", fmt.Sprint(fooV1ID), "software_version_id", fmt.Sprint(fooV1ID), "software_title_id", fmt.Sprint(fooTitleID))
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusBadRequest, &countResp, "software_title_id", fmt.Sprint(fooTitleID), "software_id", fmt.Sprint(fooV1ID))
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusBadRequest, &countResp, "software_title_id", fmt.Sprint(fooTitleID), "software_version_id", fmt.Sprint(fooV1ID))
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusBadRequest, &countResp, "software_id", fmt.Sprint(fooV1ID), "software_version_id", fmt.Sprint(fooV1ID))
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusBadRequest, &countResp, "software_id", fmt.Sprint(fooV1ID), "software_version_id", fmt.Sprint(fooV1ID), "software_title_id", fmt.Sprint(fooTitleID))

	user1 := test.NewUser(t, s.ds, "Alice", "alice@example.com", true)
	q := test.NewQuery(t, s.ds, nil, "query1", "select 1", 0, true)
	defer s.cleanupQuery(q.ID)
	globalPolicy0, err := s.ds.NewGlobalPolicy(
		context.Background(), &user1.ID, fleet.PolicyPayload{
			QueryID: &q.ID,
		})
	require.NoError(t, err)

	require.NoError(
		t,
		s.ds.RecordPolicyQueryExecutions(context.Background(), host2, map[uint]*bool{globalPolicy0.ID: ptr.Bool(false)}, time.Now(), false),
	)

	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "software_id", fmt.Sprint(fooV1ID))
	require.Len(t, resp.Hosts, 1)
	assert.Equal(t, uint64(1), resp.Hosts[0].HostIssues.FailingPoliciesCount)
	assert.Equal(t, uint64(1), resp.Hosts[0].HostIssues.TotalIssuesCount)
	assert.Nil(t, resp.Hosts[0].HostIssues.CriticalVulnerabilitiesCount)

	resp = listHostsResponse{}
	// disable_failing_policies has been deprecated and is no longer documented; it is an alias for disable_issues
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "software_version_id", fmt.Sprint(fooV1ID), "disable_failing_policies", "true")
	require.Len(t, resp.Hosts, 1)
	assert.Zero(t, resp.Hosts[0].HostIssues.FailingPoliciesCount)
	assert.Zero(t, resp.Hosts[0].HostIssues.TotalIssuesCount)
	assert.Nil(t, resp.Hosts[0].HostIssues.CriticalVulnerabilitiesCount)

	resp = listHostsResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "software_version_id", fmt.Sprint(fooV1ID), "disable_issues", "true",
	)
	require.Len(t, resp.Hosts, 1)
	assert.Zero(t, resp.Hosts[0].HostIssues.FailingPoliciesCount)
	assert.Zero(t, resp.Hosts[0].HostIssues.TotalIssuesCount)
	assert.Nil(t, resp.Hosts[0].HostIssues.CriticalVulnerabilitiesCount)

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
	require.NoError(t, s.ds.SetOrUpdateMDMData(context.Background(), host2.ID, false, true, "https://simplemdm.com", false, fleet.WellKnownMDMSimpleMDM, ""))
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
	require.NoError(t, s.ds.SetOrUpdateMDMData(context.Background(), pendingMDMHost.ID, false, false, "https://fleetdm.com", true, fleet.WellKnownMDMFleet, ""))

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
	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusNotFound, &resp, "software_version_id", fmt.Sprint(9999))
	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusNotFound, &resp, "software_title_id", fmt.Sprint(9999))

	// Filter by non-existent team.
	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusBadRequest, &resp, "team_id", fmt.Sprint(9999))

	// set munki information on a host
	require.NoError(t, s.ds.SetOrUpdateMunkiInfo(context.Background(), host2.ID, "1.2.3", []string{"err"}, []string{"warn"}))
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
	require.NoError(t, s.ds.UpdateHostOperatingSystem(context.Background(), host2.ID, testOS))
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

	// populate software for hosts
	now := time.Now()

	inserted, err := s.ds.InsertSoftwareVulnerability(context.Background(), fleet.SoftwareVulnerability{
		SoftwareID: host2.Software[0].ID,
		CVE:        "cve-123-123-123",
	}, fleet.NVDSource)
	require.NoError(t, err)
	require.True(t, inserted)

	require.NoError(t, s.ds.InsertCVEMeta(context.Background(), []fleet.CVEMeta{{
		CVE:              "cve-123-123-123",
		CVSSScore:        ptr.Float64(5.4),
		EPSSProbability:  ptr.Float64(0.5),
		CISAKnownExploit: ptr.Bool(true),
		Published:        &now,
		Description:      "a long description of the cve",
	}}))

	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "populate_software", "true")
	require.Len(t, resp.Hosts, 4)
	for _, h := range resp.Hosts {
		if h.ID == hosts[2].ID {
			require.NotEmpty(t, h.Software)
			require.Len(t, h.Software, 1)
			require.NotEmpty(t, h.Software[0].Vulnerabilities)

			// all these should be nil because this isn't Premium
			require.Nil(t, h.Software[0].Vulnerabilities[0].CVSSScore)
			require.Nil(t, h.Software[0].Vulnerabilities[0].EPSSProbability)
			require.Nil(t, h.Software[0].Vulnerabilities[0].CISAKnownExploit)
			require.Nil(t, h.Software[0].Vulnerabilities[0].CVEPublished)
			require.Nil(t, h.Software[0].Vulnerabilities[0].Description)
			require.Nil(t, h.Software[0].Vulnerabilities[0].ResolvedInVersion)
		}
		assert.Nil(t, h.Policies)
	}

	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "populate_software", "false", "populate_policies", "false")
	require.Len(t, resp.Hosts, 4)
	for _, h := range resp.Hosts {
		require.Empty(t, h.Software)
		assert.Nil(t, h.Policies)
	}

	// Populate policies for hosts. One policy was created earlier.
	ctx := context.Background()
	globalPolicy1, err := s.ds.NewGlobalPolicy(
		ctx, &test.UserAdmin.ID, fleet.PolicyPayload{
			Name:  "foobar0",
			Query: "SELECT 0;",
		},
	)
	require.NoError(t, err)

	for _, host := range hosts {
		// All hosts pass the globalPolicy1
		err := s.ds.RecordPolicyQueryExecutions(
			context.Background(), host, map[uint]*bool{globalPolicy1.ID: ptr.Bool(true)}, time.Now(), false,
		)
		require.NoError(t, err)
	}

	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "populate_policies", "true")
	require.Len(t, resp.Hosts, len(hosts)+1) // +1 for the pending MDM host
	for _, h := range resp.Hosts {
		if h.ID == hosts[0].ID {
			policies := *h.Policies
			require.Len(t, policies, 2)
			assert.Equal(t, globalPolicy0.Name, policies[0].Name)
			assert.Equal(t, "", policies[0].Response)
			assert.Equal(t, globalPolicy1.Name, policies[1].Name)
			assert.Equal(t, "pass", policies[1].Response)
		} else if h.ID == hosts[2].ID {
			policies := *h.Policies
			require.Len(t, policies, 2)
			assert.Equal(t, globalPolicy0.Name, policies[0].Name)
			assert.Equal(t, "fail", policies[0].Response)
			assert.Equal(t, globalPolicy1.Name, policies[1].Name)
			assert.Equal(t, "pass", policies[1].Response)
		}
	}

	// there are 3 hosts, whos names end with ...local0, ...local1, ...local2
	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "query", "local0")
	require.Len(t, resp.Hosts, 1)
	require.Contains(t, resp.Hosts[0].Hostname, "local0")
	resp = listHostsResponse{}
	// now with leading/trailing whitespace
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "query", " local0 ")
	require.Len(t, resp.Hosts, 1)
	require.Contains(t, resp.Hosts[0].Hostname, "local0")
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

	// list invites filtered by search query with leading/trailing whitespace
	// matches name
	listResp = listInvitesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/invites", nil, http.StatusOK, &listResp, "query", " some name                     ")
	require.Len(t, listResp.Invites, 1)
	require.Equal(t, validInvite.ID, listResp.Invites[0].ID)

	// list invites filtered by search query with leading/trailing whitespace
	// matches email
	listResp = listInvitesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/invites", nil, http.StatusOK, &listResp, "query", " some email                     ")
	require.Len(t, listResp.Invites, 1)
	require.Equal(t, validInvite.ID, listResp.Invites[0].ID)

	// list invites filtered by search query with leading/trailing whitespace
	// matches nothing
	listResp = listInvitesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/invites", nil, http.StatusOK, &listResp, "query", " no match                     ")
	require.Len(t, listResp.Invites, 0)

	// list invites, next page is empty
	listResp = listInvitesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/invites", nil, http.StatusOK, &listResp, "page", "1", "per_page", "2")
	require.Len(t, listResp.Invites, 0)

	// update a non-existing invite
	updateInviteReq := updateInviteRequest{InvitePayload: fleet.InvitePayload{
		Teams: []fleet.UserTeam{
			{Team: fleet.Team{ID: team.ID}, Role: fleet.RoleObserver},
		},
		MFAEnabled: ptr.Bool(true),
	}}
	updateInviteResp := updateInviteResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/invites/%d", validInvite.ID+1), updateInviteReq, http.StatusNotFound, &updateInviteResp)

	// update the valid invite created earlier, make it an observer of a team
	updateInviteResp = updateInviteResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/invites/%d", validInvite.ID), updateInviteReq, http.StatusPaymentRequired, &updateInviteResp)
	updateInviteReq.MFAEnabled = nil
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
	require.NoError(t, s.ds.SetOrUpdateHostDisksSpace(ctx, hosts[0].ID, 1.0, 2.0, 500.0))
	require.NoError(t, s.ds.SetOrUpdateHostDisksSpace(ctx, hosts[1].ID, 3.0, 4.0, 1000.0))

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

	// 'after' param is not supported for labels
	s.DoJSON("GET", "/api/latest/fleet/labels", nil, http.StatusBadRequest, &listResp, "order_key", "id", "after", "1")

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
	s.DoJSON("GET", "/api/latest/fleet/host_summary", nil, http.StatusBadRequest, &resp, "low_disk_space", "1234")
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
		Logging:        fleet.LoggingSnapshot,
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

	response := s.DoRaw("PATCH", fmt.Sprintf("/api/latest/fleet/policies/%d", gpResp.Policy.ID), []byte(`{
		"name": "TestQuery4",
		"query": "select * from osquery_info;",
		"description": "Some description updated",
		"resolution": "some global resolution updated"
	}`), http.StatusOK)
	var mgpResp modifyGlobalPolicyResponse
	responseBody, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	err = json.Unmarshal(responseBody, &mgpResp)
	require.NoError(t, err)

	require.NotNil(t, gpResp.Policy)
	assert.Equal(t, "TestQuery4", mgpResp.Policy.Name)
	assert.Equal(t, "select * from osquery_info;", mgpResp.Policy.Query)
	assert.Equal(t, "Some description updated", mgpResp.Policy.Description)
	require.NotNil(t, mgpResp.Policy.Resolution)
	assert.Equal(t, "some global resolution updated", *mgpResp.Policy.Resolution)
	assert.Equal(t, "darwin", mgpResp.Policy.Platform)
	assert.Equal(t, uint(0), mgpResp.Policy.FailingHostCount)
	assert.Equal(t, uint(0), mgpResp.Policy.PassingHostCount)

	ggpResp := getPolicyByIDResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/policies/%d", gpResp.Policy.ID), getPolicyByIDRequest{}, http.StatusOK, &ggpResp)
	require.NotNil(t, ggpResp.Policy)
	assert.Equal(t, "TestQuery4", ggpResp.Policy.Name)
	assert.Equal(t, "select * from osquery_info;", ggpResp.Policy.Query)
	assert.Equal(t, "Some description updated", ggpResp.Policy.Description)
	require.NotNil(t, ggpResp.Policy.Resolution)
	assert.Equal(t, "some global resolution updated", *ggpResp.Policy.Resolution)
	assert.Equal(t, "darwin", mgpResp.Policy.Platform)
	assert.Equal(t, uint(0), mgpResp.Policy.FailingHostCount)
	assert.Equal(t, uint(0), mgpResp.Policy.PassingHostCount)

	policiesResponse := listGlobalPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/policies", nil, http.StatusOK, &policiesResponse)
	require.Len(t, policiesResponse.Policies, 1)
	assert.Equal(t, "TestQuery4", policiesResponse.Policies[0].Name)
	assert.Equal(t, "select * from osquery_info;", policiesResponse.Policies[0].Query)
	assert.Equal(t, "Some description updated", policiesResponse.Policies[0].Description)
	require.NotNil(t, policiesResponse.Policies[0].Resolution)
	assert.Equal(t, "some global resolution updated", *policiesResponse.Policies[0].Resolution)
	assert.Equal(t, "darwin", policiesResponse.Policies[0].Platform)
	assert.Equal(t, uint(0), policiesResponse.Policies[0].FailingHostCount)
	assert.Equal(t, uint(0), policiesResponse.Policies[0].PassingHostCount)

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

	listHostsURL = fmt.Sprintf("/api/latest/fleet/hosts?policy_id=%d&policy_response=failing", policiesResponse.Policies[0].ID)
	listHostsResp = listHostsResponse{}
	s.DoJSON("GET", listHostsURL, nil, http.StatusOK, &listHostsResp)
	require.Len(t, listHostsResp.Hosts, 0)

	require.NoError(t, s.ds.RecordPolicyQueryExecutions(context.Background(), h1.Host, map[uint]*bool{policiesResponse.Policies[0].ID: ptr.Bool(true)}, time.Now(), false))
	require.NoError(t, s.ds.RecordPolicyQueryExecutions(context.Background(), h2.Host, map[uint]*bool{policiesResponse.Policies[0].ID: nil}, time.Now(), false))

	listHostsURL = fmt.Sprintf("/api/latest/fleet/hosts?policy_id=%d&policy_response=passing", policiesResponse.Policies[0].ID)
	listHostsResp = listHostsResponse{}
	s.DoJSON("GET", listHostsURL, nil, http.StatusOK, &listHostsResp)
	require.Len(t, listHostsResp.Hosts, 1)

	response = s.DoRaw("PATCH", fmt.Sprintf("/api/latest/fleet/policies/%d", gpResp.Policy.ID), []byte(`{
		"query": "select * from users;"
	}`), http.StatusOK)
	responseBody, err = io.ReadAll(response.Body)
	require.NoError(t, err)
	err = json.Unmarshal(responseBody, &mgpResp)
	require.NoError(t, err)

	require.NotNil(t, gpResp.Policy)
	assert.Equal(t, "TestQuery4", mgpResp.Policy.Name)
	assert.Equal(t, "select * from users;", mgpResp.Policy.Query)
	assert.Equal(t, "Some description updated", mgpResp.Policy.Description)
	require.NotNil(t, mgpResp.Policy.Resolution)
	assert.Equal(t, "some global resolution updated", *mgpResp.Policy.Resolution)
	assert.Equal(t, "darwin", mgpResp.Policy.Platform)
	assert.Equal(t, uint(0), mgpResp.Policy.FailingHostCount)
	assert.Equal(t, uint(0), mgpResp.Policy.PassingHostCount)

	listHostsURL = fmt.Sprintf("/api/latest/fleet/hosts?policy_id=%d&policy_response=passing", policiesResponse.Policies[0].ID)
	listHostsResp = listHostsResponse{}
	s.DoJSON("GET", listHostsURL, nil, http.StatusOK, &listHostsResp)
	require.Len(t, listHostsResp.Hosts, 0)

	listHostsURL = fmt.Sprintf("/api/latest/fleet/hosts?policy_id=%d&policy_response=failing", policiesResponse.Policies[0].ID)
	listHostsResp = listHostsResponse{}
	s.DoJSON("GET", listHostsURL, nil, http.StatusOK, &listHostsResp)
	require.Len(t, listHostsResp.Hosts, 0)

	policiesResponse = listGlobalPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/policies", nil, http.StatusOK, &policiesResponse)
	require.Len(t, policiesResponse.Policies, 1)
	assert.Equal(t, "TestQuery4", policiesResponse.Policies[0].Name)
	assert.Equal(t, "select * from users;", policiesResponse.Policies[0].Query)
	assert.Equal(t, "Some description updated", policiesResponse.Policies[0].Description)
	require.NotNil(t, policiesResponse.Policies[0].Resolution)
	assert.Equal(t, "some global resolution updated", *policiesResponse.Policies[0].Resolution)
	assert.Equal(t, "darwin", policiesResponse.Policies[0].Platform)
	assert.Equal(t, uint(0), policiesResponse.Policies[0].FailingHostCount)
	assert.Equal(t, uint(0), policiesResponse.Policies[0].PassingHostCount)

	// Record query executions
	require.NoError(
		t, s.ds.RecordPolicyQueryExecutions(
			context.Background(), h1.Host, map[uint]*bool{policiesResponse.Policies[0].ID: ptr.Bool(true)}, time.Now(), false,
		),
	)
	require.NoError(
		t, s.ds.RecordPolicyQueryExecutions(
			context.Background(), h2.Host, map[uint]*bool{policiesResponse.Policies[0].ID: nil}, time.Now(), false,
		),
	)
	// Update policy stats
	require.NoError(t, s.ds.UpdateHostPolicyCounts(context.Background()))

	// Fetch policy to make sure stats are updated
	s.DoJSON("GET", "/api/latest/fleet/policies", nil, http.StatusOK, &policiesResponse)
	require.Len(t, policiesResponse.Policies, 1)
	assert.Equal(t, uint(0), policiesResponse.Policies[0].FailingHostCount)
	assert.Equal(t, uint(1), policiesResponse.Policies[0].PassingHostCount)

	listHostsURL = fmt.Sprintf("/api/latest/fleet/hosts?policy_id=%d&policy_response=passing", policiesResponse.Policies[0].ID)
	listHostsResp = listHostsResponse{}
	s.DoJSON("GET", listHostsURL, nil, http.StatusOK, &listHostsResp)
	require.Len(t, listHostsResp.Hosts, 1)

	// Modify the platform for the policy, which should clear the policy stats
	response = s.DoRaw("PATCH", fmt.Sprintf("/api/latest/fleet/policies/%d", gpResp.Policy.ID), []byte(`{
		"platform": "linux"
	}`), http.StatusOK)
	responseBody, err = io.ReadAll(response.Body)
	require.NoError(t, err)
	err = json.Unmarshal(responseBody, &mgpResp)
	require.NoError(t, err)

	require.NotNil(t, gpResp.Policy)
	assert.Equal(t, "TestQuery4", mgpResp.Policy.Name)
	assert.Equal(t, "select * from users;", mgpResp.Policy.Query)
	assert.Equal(t, "Some description updated", mgpResp.Policy.Description)
	require.NotNil(t, mgpResp.Policy.Resolution)
	assert.Equal(t, "some global resolution updated", *mgpResp.Policy.Resolution)
	assert.Equal(t, "linux", mgpResp.Policy.Platform)
	assert.Equal(t, uint(0), mgpResp.Policy.FailingHostCount)
	assert.Equal(t, uint(0), mgpResp.Policy.PassingHostCount)

	// Fetch policy to make sure stats are updated
	s.DoJSON("GET", "/api/latest/fleet/policies", nil, http.StatusOK, &policiesResponse)
	require.Len(t, policiesResponse.Policies, 1)
	assert.Equal(t, uint(0), policiesResponse.Policies[0].FailingHostCount)
	assert.Equal(t, uint(0), policiesResponse.Policies[0].PassingHostCount)

	listHostsURL = fmt.Sprintf("/api/latest/fleet/hosts?policy_id=%d&policy_response=passing", policiesResponse.Policies[0].ID)
	listHostsResp = listHostsResponse{}
	s.DoJSON("GET", listHostsURL, nil, http.StatusOK, &listHostsResp)
	require.Len(t, listHostsResp.Hosts, 0)

	listHostsURL = fmt.Sprintf("/api/latest/fleet/hosts?policy_id=%d&policy_response=failing", policiesResponse.Policies[0].ID)
	listHostsResp = listHostsResponse{}
	s.DoJSON("GET", listHostsURL, nil, http.StatusOK, &listHostsResp)
	require.Len(t, listHostsResp.Hosts, 0)

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

	tpName := "TestPolicy3"
	tpParams := teamPolicyRequest{
		Name:        tpName,
		Query:       "select * from osquery;",
		Description: "Some description",
		Resolution:  "some team resolution",
		Platform:    "darwin",
	}
	tpResp := teamPolicyResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team1.ID), tpParams, http.StatusOK, &tpResp)
	require.NotNil(t, tpResp.Policy)
	require.NotEmpty(t, tpResp.Policy.ID)
	assert.Equal(t, tpName, tpResp.Policy.Name)
	assert.Equal(t, "select * from osquery;", tpResp.Policy.Query)
	assert.Equal(t, "Some description", tpResp.Policy.Description)
	require.NotNil(t, tpResp.Policy.Resolution)
	assert.Equal(t, "some team resolution", *tpResp.Policy.Resolution)
	assert.NotNil(t, tpResp.Policy.AuthorID)
	assert.Equal(t, "Test Name admin1@example.com", tpResp.Policy.AuthorName)
	assert.Equal(t, "admin1@example.com", tpResp.Policy.AuthorEmail)

	tpNameNew := "TestPolicy4"

	response := s.DoRaw("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team1.ID, tpResp.Policy.ID), []byte(fmt.Sprintf(`{
		"name": "%s",
		"query": "select * from osquery_info;",
		"description": "Some description updated",
		"resolution": "some team resolution updated"
	}`, tpNameNew)), http.StatusOK)
	var mtpResp modifyGlobalPolicyResponse
	responseBody, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	err = json.Unmarshal(responseBody, &mtpResp)
	require.NoError(t, err)

	require.NotNil(t, mtpResp.Policy)
	assert.Equal(t, tpNameNew, mtpResp.Policy.Name)
	assert.Equal(t, "select * from osquery_info;", mtpResp.Policy.Query)
	assert.Equal(t, "Some description updated", mtpResp.Policy.Description)
	require.NotNil(t, mtpResp.Policy.Resolution)
	assert.Equal(t, "some team resolution updated", *mtpResp.Policy.Resolution)
	assert.Equal(t, "darwin", mtpResp.Policy.Platform)

	gtpResp := getPolicyByIDResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team1.ID, tpResp.Policy.ID), getPolicyByIDRequest{}, http.StatusOK, &gtpResp)
	require.NotNil(t, gtpResp.Policy)
	assert.Equal(t, tpNameNew, gtpResp.Policy.Name)
	assert.Equal(t, "select * from osquery_info;", gtpResp.Policy.Query)
	assert.Equal(t, "Some description updated", gtpResp.Policy.Description)
	require.NotNil(t, gtpResp.Policy.Resolution)
	assert.Equal(t, "some team resolution updated", *gtpResp.Policy.Resolution)
	assert.Equal(t, "darwin", gtpResp.Policy.Platform)

	policiesResponse := listTeamPoliciesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team1.ID), nil, http.StatusOK, &policiesResponse)
	require.Len(t, policiesResponse.Policies, 1)
	assert.Equal(t, tpNameNew, policiesResponse.Policies[0].Name)
	assert.Equal(t, "select * from osquery_info;", policiesResponse.Policies[0].Query)
	assert.Equal(t, "Some description updated", policiesResponse.Policies[0].Description)
	require.NotNil(t, policiesResponse.Policies[0].Resolution)
	assert.Equal(t, "some team resolution updated", *policiesResponse.Policies[0].Resolution)
	assert.Equal(t, "darwin", policiesResponse.Policies[0].Platform)
	require.Len(t, policiesResponse.InheritedPolicies, 0)

	// test team policy count endpoint
	tpCountResp := countTeamPoliciesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/count", team1.ID), nil, http.StatusOK, &tpCountResp)
	assert.Equal(t, 1, tpCountResp.Count)

	tpCountResp = countTeamPoliciesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/count", team1.ID), nil, http.StatusOK, &tpCountResp, "query", tpNameNew)
	assert.Equal(t, 1, tpCountResp.Count)

	tpCountResp = countTeamPoliciesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/count", team1.ID), nil, http.StatusOK, &tpCountResp, "query", " "+tpNameNew+" ")
	assert.Equal(t, 1, tpCountResp.Count)

	tpCountResp = countTeamPoliciesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/count", team1.ID), nil, http.StatusOK, &tpCountResp, "query", " nomatch")
	assert.Equal(t, 0, tpCountResp.Count)

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
	b, err := io.ReadAll(resp.Body)
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
	// Policies that did not run are listed before passing policies
	require.True(t, reflect.DeepEqual(tpResp.Policy.PolicyData, policies[0].PolicyData))
	require.Equal(t, policies[0].Response, "") // policy didn't "run"

	require.True(t, reflect.DeepEqual(gpResp.Policy.PolicyData, policies[1].PolicyData))
	require.Equal(t, policies[1].Response, "pass")

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

	timestamp := time.Now()
	ctx = context.WithValue(ctx, fleet.ActivityWebhookContextKey, true)
	err = s.ds.NewActivity(ctx, &u, fleet.ActivityTypeAppliedSpecPack{}, nil, timestamp)
	require.NoError(t, err)

	err = s.ds.NewActivity(ctx, &u, fleet.ActivityTypeDeletedPack{}, nil, timestamp)
	require.NoError(t, err)

	err = s.ds.NewActivity(ctx, &u, fleet.ActivityTypeEditedPack{}, nil, timestamp)
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

	// with 'after' param
	s.DoJSON(
		"GET", "/api/latest/fleet/carves", nil, http.StatusOK, &listResp, "per_page", "2", "order_key", "id", "after",
		strconv.FormatInt(c1.ID, 10),
	)
	require.Len(t, listResp.Carves, 1)
	assert.Equal(t, c3.ID, listResp.Carves[0].ID)

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

	// assign host0 and host1 to team 1
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

	// check that hosts are now part of team 1
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", hosts[0].ID), nil, http.StatusOK, &getResp)
	require.NotNil(t, getResp.Host.TeamID)
	require.Equal(t, tm1.ID, *getResp.Host.TeamID)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", hosts[1].ID), nil, http.StatusOK, &getResp)
	require.NotNil(t, getResp.Host.TeamID)
	require.Equal(t, tm1.ID, *getResp.Host.TeamID)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", hosts[2].ID), nil, http.StatusOK, &getResp)
	require.Nil(t, getResp.Host.TeamID)

	// assign host2 to team 2 with filter
	var addfResp addHostsToTeamByFilterResponse
	req := addHostsToTeamByFilterRequest{
		TeamID:  &tm2.ID,
		Filters: &map[string]interface{}{"query": hosts[2].Hostname},
	}

	s.DoJSON("POST", "/api/latest/fleet/hosts/transfer/filter", req, http.StatusOK, &addfResp)
	s.lastActivityOfTypeMatches(
		fleet.ActivityTypeTransferredHostsToTeam{}.ActivityName(),
		fmt.Sprintf(`{"team_id": %d, "team_name": %q, "host_ids": [%d], "host_display_names": [%q]}`,
			tm2.ID, tm2.Name, hosts[2].ID, hosts[2].DisplayName()),
		0,
	)

	// check that host2 is now part of team 2
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", hosts[2].ID), nil, http.StatusOK, &getResp)
	require.NotNil(t, getResp.Host.TeamID)
	require.Equal(t, tm2.ID, *getResp.Host.TeamID)

	// get all hosts label
	lblIDs, err := s.ds.LabelIDsByName(context.Background(), []string{"All Hosts"})
	require.NoError(t, err)
	labelID := lblIDs["All Hosts"]

	// Add label to host0
	err = s.ds.RecordLabelQueryExecutions(context.Background(), hosts[0], map[uint]*bool{labelID: ptr.Bool(true)}, time.Now(), false)
	require.NoError(t, err)

	// offline status filter request should not move hosts
	req = addHostsToTeamByFilterRequest{
		TeamID:  &tm2.ID,
		Filters: &map[string]interface{}{"status": "offline", "label_id": float64(labelID)},
	}
	var hostsResp listHostsResponse
	s.DoJSON("POST", "/api/latest/fleet/hosts/transfer/filter", req, http.StatusOK, &addfResp)
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &hostsResp)
	require.Len(t, hostsResp.Hosts, 3)
	require.Equal(t, tm1.ID, *hostsResp.Hosts[0].TeamID)
	require.Equal(t, tm1.ID, *hostsResp.Hosts[1].TeamID)
	require.Equal(t, tm2.ID, *hostsResp.Hosts[2].TeamID)

	// assign host0 to team 2 with filter
	req = addHostsToTeamByFilterRequest{
		TeamID:  &tm2.ID,
		Filters: &map[string]interface{}{"status": "online", "label_id": float64(labelID)},
	}
	s.DoJSON("POST", "/api/latest/fleet/hosts/transfer/filter", req, http.StatusOK, &addfResp)

	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", hosts[0].ID), nil, http.StatusOK, &getResp)
	require.NotNil(t, getResp.Host.TeamID)
	require.Equal(t, tm2.ID, *getResp.Host.TeamID)

	// status filter request should not delete hosts
	dreq := deleteHostsRequest{
		Filters: &map[string]interface{}{"status": "offline", "label_id": float64(labelID)},
	}
	s.DoJSON("POST", "/api/latest/fleet/hosts/transfer/filter", req, http.StatusOK, &dreq)
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &hostsResp)
	require.Len(t, hostsResp.Hosts, 3)

	// delete host 0 with filter
	dreq = deleteHostsRequest{
		Filters: &map[string]interface{}{"status": "online", "label_id": float64(labelID)},
	}
	var delHostsResp deleteHostsResponse
	s.DoJSON("POST", "/api/latest/fleet/hosts/delete", dreq, http.StatusOK, &delHostsResp)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", hosts[0].ID), nil, http.StatusNotFound, &getResp)

	// delete non-existing host
	var delResp deleteHostResponse
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

func (s *integrationTestSuite) TestGetHostByIdentifier() {
	t := s.T()
	ctx := context.Background()

	hosts := make([]*fleet.Host, 6)
	for i := 0; i < len(hosts); i++ {
		h, err := s.ds.NewHost(ctx, &fleet.Host{
			Hostname:       fmt.Sprintf("test-host%d-name", i),
			OsqueryHostID:  ptr.String(fmt.Sprintf("osquery-%d", i)),
			NodeKey:        ptr.String(fmt.Sprintf("nodekey-%d", i)),
			UUID:           fmt.Sprintf("test-uuid-%d", i),
			Platform:       "darwin",
			HardwareSerial: fmt.Sprintf("serial-%d", i),
		})
		require.NoError(t, err)
		hosts[i] = h
	}

	var resp getHostResponse
	s.DoJSON("GET", "/api/v1/fleet/hosts/identifier/osquery-1", nil, http.StatusOK, &resp)
	require.Equal(t, hosts[1].ID, resp.Host.ID)

	s.DoJSON("GET", "/api/v1/fleet/hosts/identifier/serial-2", nil, http.StatusOK, &resp)
	require.Equal(t, hosts[2].ID, resp.Host.ID)

	s.DoJSON("GET", "/api/v1/fleet/hosts/identifier/nodekey-3", nil, http.StatusOK, &resp)
	require.Equal(t, hosts[3].ID, resp.Host.ID)

	s.DoJSON("GET", "/api/v1/fleet/hosts/identifier/test-uuid-4", nil, http.StatusOK, &resp)
	require.Equal(t, hosts[4].ID, resp.Host.ID)

	s.DoJSON("GET", "/api/v1/fleet/hosts/identifier/test-host5-name", nil, http.StatusOK, &resp)
	require.Equal(t, hosts[5].ID, resp.Host.ID)

	s.DoJSON("GET", "/api/v1/fleet/hosts/identifier/no-such-host", nil, http.StatusNotFound, &resp)
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

	// listing with matching name returns that query
	s.DoJSON("GET", "/api/latest/fleet/queries", nil, http.StatusOK, &listQryResp, "query", query.Name)
	require.Len(t, listQryResp.Queries, 1)
	assert.Equal(t, query.Name, listQryResp.Queries[0].Name)

	// listing with matching name plus whitespace returns that query
	s.DoJSON("GET", "/api/latest/fleet/queries", nil, http.StatusOK, &listQryResp, "query", "  "+query.Name+" ")
	require.Len(t, listQryResp.Queries, 1)
	assert.Equal(t, query.Name, listQryResp.Queries[0].Name)

	// listing with non-matching name returns nothing
	s.DoJSON("GET", "/api/latest/fleet/queries", nil, http.StatusOK, &listQryResp, "query", "  nomatch")
	require.Len(t, listQryResp.Queries, 0)

	// Return that query by name
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries?query=%s", query.Name), nil, http.StatusOK, &listQryResp)
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

	// TODO(jahziel): check that the query results were deleted

	// TODO(jahziel): check that the query results were deleted after setting `discard_data`

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

	orbitHost := createOrbitEnrolledHost(t, "windows", "device_mapping", s.ds)
	hosts := s.createHosts(t)

	// get host device mappings of invalid host
	var listResp listHostDeviceMappingResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/device_mapping", hosts[2].ID+1), nil, http.StatusNotFound, &listResp)

	// existing host but none yet
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/device_mapping", hosts[0].ID), nil, http.StatusOK, &listResp)
	require.Len(t, listResp.DeviceMapping, 0)

	// create a custom mapping of a non-existing host
	var putResp putHostDeviceMappingResponse
	s.DoJSON("PUT", fmt.Sprintf("/api/latest/fleet/hosts/%d/device_mapping", hosts[2].ID+1), nil, http.StatusNotFound, &putResp)

	// create some google mappings
	require.NoError(t, s.ds.ReplaceHostDeviceMapping(ctx, hosts[0].ID, []*fleet.HostDeviceMapping{
		{HostID: hosts[0].ID, Email: "a@b.c", Source: fleet.DeviceMappingGoogleChromeProfiles},
		{HostID: hosts[0].ID, Email: "b@b.c", Source: fleet.DeviceMappingGoogleChromeProfiles},
	}, fleet.DeviceMappingGoogleChromeProfiles))

	// create a custom mapping
	s.DoJSON("PUT", fmt.Sprintf("/api/latest/fleet/hosts/%d/device_mapping", hosts[0].ID), putHostDeviceMappingRequest{Email: "c@b.c"}, http.StatusOK, &putResp)
	require.Equal(t, hosts[0].ID, putResp.HostID)
	require.ElementsMatch(t, putResp.DeviceMapping, []*fleet.HostDeviceMapping{
		{Email: "a@b.c", Source: fleet.DeviceMappingGoogleChromeProfiles},
		{Email: "b@b.c", Source: fleet.DeviceMappingGoogleChromeProfiles},
		{Email: "c@b.c", Source: fleet.DeviceMappingCustomReplacement},
	})

	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/device_mapping", hosts[0].ID), nil, http.StatusOK, &listResp)
	require.Equal(t, hosts[0].ID, listResp.HostID)
	require.ElementsMatch(t, listResp.DeviceMapping, []*fleet.HostDeviceMapping{
		{Email: "a@b.c", Source: fleet.DeviceMappingGoogleChromeProfiles},
		{Email: "b@b.c", Source: fleet.DeviceMappingGoogleChromeProfiles},
		{Email: "c@b.c", Source: fleet.DeviceMappingCustomReplacement},
	})

	// other host still has none
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/device_mapping", hosts[1].ID), nil, http.StatusOK, &listResp)
	require.Len(t, listResp.DeviceMapping, 0)

	var listHosts listHostsResponse
	// list hosts response includes device mappings
	s.DoJSON("GET", "/api/latest/fleet/hosts?device_mapping=true", nil, http.StatusOK, &listHosts)
	require.Len(t, listHosts.Hosts, len(hosts)+1)
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
	require.ElementsMatch(t, dm, []*fleet.HostDeviceMapping{
		{Email: "a@b.c", Source: fleet.DeviceMappingGoogleChromeProfiles},
		{Email: "b@b.c", Source: fleet.DeviceMappingGoogleChromeProfiles},
		{Email: "c@b.c", Source: fleet.DeviceMappingCustomReplacement},
	})

	// no device mapping for other hosts
	assert.Nil(t, hostsByID[hosts[1].ID].DeviceMapping)
	assert.Nil(t, hostsByID[hosts[2].ID].DeviceMapping)
	assert.Nil(t, hostsByID[orbitHost.ID].DeviceMapping)

	// update custom email for hosts[0]
	s.DoJSON("PUT", fmt.Sprintf("/api/latest/fleet/hosts/%d/device_mapping", hosts[0].ID), putHostDeviceMappingRequest{Email: "d@b.c"}, http.StatusOK, &putResp)
	require.Equal(t, hosts[0].ID, putResp.HostID)
	require.ElementsMatch(t, putResp.DeviceMapping, []*fleet.HostDeviceMapping{
		{Email: "a@b.c", Source: fleet.DeviceMappingGoogleChromeProfiles},
		{Email: "b@b.c", Source: fleet.DeviceMappingGoogleChromeProfiles},
		{Email: "d@b.c", Source: fleet.DeviceMappingCustomReplacement},
	})

	// create a custom_installer email for orbit host
	s.Do("PUT", "/api/fleet/orbit/device_mapping", orbitPutDeviceMappingRequest{
		OrbitNodeKey: *orbitHost.OrbitNodeKey,
		Email:        "e@b.c",
	}, http.StatusOK)

	// search host by email address finds the corresponding host
	s.DoJSON("GET", "/api/latest/fleet/hosts?device_mapping=true", nil, http.StatusOK, &listHosts, "query", "a@b.c")
	require.Len(t, listHosts.Hosts, 1)
	require.Equal(t, host1.ID, listHosts.Hosts[0].ID)
	require.NotNil(t, listHosts.Hosts[0].DeviceMapping)

	err = json.Unmarshal(*listHosts.Hosts[0].DeviceMapping, &dm)
	require.NoError(t, err)
	require.ElementsMatch(t, putResp.DeviceMapping, []*fleet.HostDeviceMapping{
		{Email: "a@b.c", Source: fleet.DeviceMappingGoogleChromeProfiles},
		{Email: "b@b.c", Source: fleet.DeviceMappingGoogleChromeProfiles},
		{Email: "d@b.c", Source: fleet.DeviceMappingCustomReplacement},
	})

	// search host by the custom email address finds the corresponding host
	s.DoJSON("GET", "/api/latest/fleet/hosts?device_mapping=true", nil, http.StatusOK, &listHosts, "query", "d@b.c")
	require.Len(t, listHosts.Hosts, 1)
	require.Equal(t, hosts[0].ID, listHosts.Hosts[0].ID)

	s.DoJSON("GET", "/api/latest/fleet/hosts?device_mapping=true", nil, http.StatusOK, &listHosts, "query", "e@b.c")
	require.Len(t, listHosts.Hosts, 1)
	require.Equal(t, orbitHost.ID, listHosts.Hosts[0].ID)

	// override the custom email for the orbit host
	s.DoJSON("PUT", fmt.Sprintf("/api/latest/fleet/hosts/%d/device_mapping", orbitHost.ID), putHostDeviceMappingRequest{Email: "f@b.c"}, http.StatusOK, &putResp)

	// update the custom_installer email for orbit host, will get ignored (because a custom_override exists)
	s.Do("PUT", "/api/fleet/orbit/device_mapping", orbitPutDeviceMappingRequest{
		OrbitNodeKey: *orbitHost.OrbitNodeKey,
		Email:        "g@b.c",
	}, http.StatusOK)

	// searching by the old custom installer email doesn't work anymore
	s.DoJSON("GET", "/api/latest/fleet/hosts?device_mapping=true", nil, http.StatusOK, &listHosts, "query", "e@b.c")
	require.Len(t, listHosts.Hosts, 0)

	// searching by the new custom email address finds it
	s.DoJSON("GET", "/api/latest/fleet/hosts?device_mapping=true", nil, http.StatusOK, &listHosts, "query", "f@b.c")
	require.Len(t, listHosts.Hosts, 1)
	require.Equal(t, orbitHost.ID, listHosts.Hosts[0].ID)

	// searching by a never-used email returns nothing
	s.DoJSON("GET", "/api/latest/fleet/hosts?device_mapping=true", nil, http.StatusOK, &listHosts, "query", "Z@b.c")
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

	require.NoError(t, s.ds.ReplaceHostDeviceMapping(ctx, hosts[0].ID, mappings, "google_chrome_profiles"))

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

	require.NoError(t, s.ds.SetOrUpdateMDMData(ctx, hostAll.ID, false, true, "url", false, "", ""))
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

	require.NoError(t, s.ds.SetOrUpdateMDMData(ctx, hostAll.ID, false, true, "https://simplemdm.com", true, fleet.WellKnownMDMSimpleMDM, ""))
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

	require.NoError(t, s.ds.SetOrUpdateMDMData(ctx, hostAll.ID, false, false, "url2", false, "", ""))

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
	require.NoError(t, s.ds.SetOrUpdateMDMData(ctx, hostOnlyMDM.ID, false, true, "https://kandji.io", true, fleet.WellKnownMDMKandji, ""))
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

	// Delete Munki from host -- no munki, but issues stick.
	require.NoError(t, s.ds.SetOrUpdateMunkiInfo(ctx, hostAll.ID, "", []string{"error1", "error3"}, []string{}))
	macadminsData = macadminsDataResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/macadmins", hostAll.ID), nil, http.StatusOK, &macadminsData)
	require.NotNil(t, macadminsData.Macadmins)
	assert.Equal(t, "Off", macadminsData.Macadmins.MDM.EnrollmentStatus)
	assert.Nil(t, macadminsData.Macadmins.MDM.Name)
	require.NotNil(t, macadminsData.Macadmins.MDM.ID)
	assert.NotZero(t, *macadminsData.Macadmins.MDM.ID)
	require.Nil(t, macadminsData.Macadmins.Munki)
	require.Len(t, macadminsData.Macadmins.MunkiIssues, 2)

	// Bring Munki back, with same issues.
	require.NoError(t, s.ds.SetOrUpdateMunkiInfo(ctx, hostAll.ID, "6.4", []string{"error1", "error3"}, []string{}))
	macadminsData = macadminsDataResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/macadmins", hostAll.ID), nil, http.StatusOK, &macadminsData)
	require.NotNil(t, macadminsData.Macadmins)
	assert.Equal(t, "Off", macadminsData.Macadmins.MDM.EnrollmentStatus)
	assert.Nil(t, macadminsData.Macadmins.MDM.Name)
	require.NotNil(t, macadminsData.Macadmins.MDM.ID)
	assert.NotZero(t, *macadminsData.Macadmins.MDM.ID)
	assert.NotNil(t, macadminsData.Macadmins.Munki)
	require.NotNil(t, macadminsData.Macadmins.Munki.Version, "6.4")
	require.Len(t, macadminsData.Macadmins.MunkiIssues, 2)

	// Delete Munki from host without MDM -- nothing is returned
	require.NoError(t, s.ds.SetOrUpdateMunkiInfo(ctx, hostOnlyMunki.ID, "", nil, []string{}))
	macadminsData = macadminsDataResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/macadmins", hostOnlyMunki.ID), nil, http.StatusOK, &macadminsData)
	require.Nil(t, macadminsData.Macadmins)

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

	// create some hosts to use for manual labels
	hosts := s.createHosts(t, "debian", "linux", "fedora", "darwin", "darwin", "darwin")
	manualHosts := hosts[:3]
	lbl2Hosts := hosts[3:]

	// list labels, has the built-in ones
	builtinsMap := fleet.ReservedLabelNames()
	var listResp listLabelsResponse
	s.DoJSON("GET", "/api/latest/fleet/labels", nil, http.StatusOK, &listResp)
	assert.True(t, len(listResp.Labels) > 0)
	var builtinLbl fleet.Label
	for _, lbl := range listResp.Labels {
		_, ok := builtinsMap[lbl.Name]
		assert.True(t, ok)
		assert.Equal(t, fleet.LabelTypeBuiltIn, lbl.LabelType)
		builtinLbl = lbl.Label
	}
	builtInsCount := len(listResp.Labels)
	require.Equal(t, builtInsCount, len(builtinsMap))

	// labels summary has the built-in ones
	var summaryResp getLabelsSummaryResponse
	s.DoJSON("GET", "/api/latest/fleet/labels/summary", nil, http.StatusOK, &summaryResp)
	assert.Len(t, summaryResp.Labels, builtInsCount)
	for _, lbl := range summaryResp.Labels {
		_, ok := builtinsMap[lbl.Name]
		assert.True(t, ok)
		assert.Equal(t, fleet.LabelTypeBuiltIn, lbl.LabelType)
	}

	// create a label without name, an error
	var createResp createLabelResponse
	s.DoJSON("POST", "/api/latest/fleet/labels", &fleet.LabelPayload{Query: "select 1"}, http.StatusUnprocessableEntity, &createResp)

	// create a label with both a query and hosts, error
	res := s.Do("POST", "/api/latest/fleet/labels", &fleet.LabelPayload{Name: t.Name(), Query: "select 1", Hosts: []string{manualHosts[0].UUID}}, http.StatusUnprocessableEntity)
	errMsg := extractServerErrorText(res.Body)
	require.Contains(t, errMsg, `Only one of either "query" or "hosts" can be included in the request.`)

	// create invalid label, conflicts with builtin name
	for n := range builtinsMap {
		s.DoJSON("POST", "/api/latest/fleet/labels", &fleet.LabelPayload{Name: n, Query: "select 1"}, http.StatusUnprocessableEntity, &createResp)
	}

	// create a valid dynamic label
	s.DoJSON("POST", "/api/latest/fleet/labels", &fleet.LabelPayload{Name: t.Name(), Query: "select 1"}, http.StatusOK, &createResp)
	assert.NotZero(t, createResp.Label.ID)
	assert.Equal(t, t.Name(), createResp.Label.Name)
	assert.Empty(t, createResp.Label.HostIDs)
	lbl1 := createResp.Label.Label

	// try to create a manual label with the same name
	s.DoJSON("POST", "/api/latest/fleet/labels", &fleet.LabelPayload{Name: lbl1.Name, Hosts: []string{manualHosts[0].UUID}}, http.StatusConflict, &createResp)
	// try to create a dynamic label with the same name
	s.DoJSON("POST", "/api/latest/fleet/labels", &fleet.LabelPayload{Name: lbl1.Name, Query: "select 2"}, http.StatusConflict, &createResp)

	// get the label
	var getResp getLabelResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/labels/%d", lbl1.ID), nil, http.StatusOK, &getResp)
	assert.Equal(t, lbl1.ID, getResp.Label.ID)
	assert.Empty(t, getResp.Label.HostIDs)

	// get a non-existing label
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/labels/%d", lbl1.ID+1), nil, http.StatusNotFound, &getResp)

	// create a valid manual label
	createResp = createLabelResponse{}
	s.DoJSON("POST", "/api/latest/fleet/labels", &fleet.LabelPayload{Name: t.Name() + "manual", Hosts: []string{manualHosts[0].UUID, manualHosts[1].Hostname, *manualHosts[2].NodeKey}}, http.StatusOK, &createResp)
	assert.NotZero(t, createResp.Label.ID)
	assert.Equal(t, t.Name()+"manual", createResp.Label.Name)
	assert.ElementsMatch(t, []uint{manualHosts[0].ID, manualHosts[1].ID, manualHosts[2].ID}, createResp.Label.HostIDs)
	manualLbl1 := createResp.Label.Label

	// get the label
	getResp = getLabelResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/labels/%d", manualLbl1.ID), nil, http.StatusOK, &getResp)
	assert.Equal(t, manualLbl1.ID, getResp.Label.ID)
	assert.Equal(t, fleet.LabelTypeRegular, getResp.Label.LabelType)
	assert.Equal(t, fleet.LabelMembershipTypeManual, getResp.Label.LabelMembershipType)
	assert.ElementsMatch(t, []uint{manualHosts[0].ID, manualHosts[1].ID, manualHosts[2].ID}, getResp.Label.HostIDs)
	assert.EqualValues(t, 3, getResp.Label.HostCount)

	// create a valid empty manual label
	createResp = createLabelResponse{}
	s.DoJSON("POST", "/api/latest/fleet/labels", &fleet.LabelPayload{Name: strings.ReplaceAll(t.Name(), "/", "_") + "manual2"}, http.StatusOK, &createResp)
	assert.NotZero(t, createResp.Label.ID)
	assert.Equal(t, strings.ReplaceAll(t.Name(), "/", "_")+"manual2", createResp.Label.Name)
	assert.Empty(t, createResp.Label.HostIDs)
	manualLbl2 := createResp.Label.Label

	// try to create a manual label with the same name
	s.DoJSON("POST", "/api/latest/fleet/labels", &fleet.LabelPayload{Name: manualLbl2.Name, Hosts: []string{manualHosts[0].UUID}}, http.StatusConflict, &createResp)
	// try to create a dynamic label with the same name
	s.DoJSON("POST", "/api/latest/fleet/labels", &fleet.LabelPayload{Name: manualLbl2.Name, Query: "select 2"}, http.StatusConflict, &createResp)

	// get the label
	getResp = getLabelResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/labels/%d", manualLbl2.ID), nil, http.StatusOK, &getResp)
	assert.Equal(t, manualLbl2.ID, getResp.Label.ID)
	assert.Equal(t, fleet.LabelTypeRegular, getResp.Label.LabelType)
	assert.Equal(t, fleet.LabelMembershipTypeManual, getResp.Label.LabelMembershipType)
	assert.Empty(t, getResp.Label.HostIDs)
	assert.EqualValues(t, 0, getResp.Label.HostCount)

	// get a non-existing label
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/labels/%d", 9999), nil, http.StatusNotFound, &getResp)

	// modify dynamic label lbl1
	var modResp modifyLabelResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/labels/%d", lbl1.ID), &fleet.ModifyLabelPayload{Name: ptr.String(t.Name() + "zzz")}, http.StatusOK, &modResp)
	assert.Equal(t, lbl1.ID, modResp.Label.ID)
	assert.Empty(t, modResp.Label.HostIDs)
	assert.NotEqual(t, lbl1.Name, modResp.Label.Name)

	// attempt to modify a label to a reserved name
	for n := range builtinsMap {
		s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/labels/%d", lbl1.ID), &fleet.ModifyLabelPayload{Name: ptr.String(n)}, http.StatusUnprocessableEntity, &modResp)
	}

	// modify a non-existing label
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/labels/%d", 9999), &fleet.ModifyLabelPayload{Name: ptr.String("zzz")}, http.StatusNotFound, &modResp)
	// modify a built-in label
	res = s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/labels/%d", builtinLbl.ID), &fleet.ModifyLabelPayload{Name: ptr.String("zzz")}, http.StatusUnprocessableEntity)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "cannot modify built-in label")

	// modify manual label 1 without modifying its hosts
	modResp = modifyLabelResponse{}
	newName := "modified_manual_label1"
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/labels/%d", manualLbl1.ID), &fleet.ModifyLabelPayload{Name: &newName}, http.StatusOK,
		&modResp)
	assert.Equal(t, manualLbl1.ID, modResp.Label.ID)
	assert.Equal(t, fleet.LabelTypeRegular, modResp.Label.LabelType)
	assert.Equal(t, fleet.LabelMembershipTypeManual, modResp.Label.LabelMembershipType)
	assert.ElementsMatch(t, []uint{manualHosts[0].ID, manualHosts[1].ID, manualHosts[2].ID}, modResp.Label.HostIDs)
	assert.EqualValues(t, 3, modResp.Label.HostCount)
	assert.Equal(t, newName, modResp.Label.Name)

	// modify manual label 2 adding some hosts
	modResp = modifyLabelResponse{}
	newName = "modified_manual_label2"
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/labels/%d", manualLbl2.ID),
		&fleet.ModifyLabelPayload{Name: &newName, Hosts: []string{manualHosts[0].UUID}}, http.StatusOK, &modResp)
	assert.Equal(t, manualLbl2.ID, modResp.Label.ID)
	assert.Equal(t, fleet.LabelTypeRegular, modResp.Label.LabelType)
	assert.Equal(t, fleet.LabelMembershipTypeManual, modResp.Label.LabelMembershipType)
	assert.ElementsMatch(t, []uint{manualHosts[0].ID}, modResp.Label.HostIDs)
	assert.EqualValues(t, 1, modResp.Label.HostCount)
	assert.Equal(t, newName, modResp.Label.Name)
	manualLbl2.Name = newName

	// modify manual label 2 clearing its hosts
	modResp = modifyLabelResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/labels/%d", manualLbl2.ID), &fleet.ModifyLabelPayload{Hosts: []string{}, Description: ptr.String("desc")}, http.StatusOK, &modResp)
	assert.Equal(t, manualLbl2.ID, modResp.Label.ID)
	assert.Equal(t, "desc", modResp.Label.Description)
	assert.Empty(t, modResp.Label.HostIDs)
	assert.EqualValues(t, 0, modResp.Label.HostCount)

	// list labels
	dynamicLabels := []fleet.Label{lbl1}
	manualLabels := []fleet.Label{manualLbl1, manualLbl2}
	s.DoJSON("GET", "/api/latest/fleet/labels", nil, http.StatusOK, &listResp, "per_page", strconv.Itoa(100))
	assert.Len(t, listResp.Labels, builtInsCount+len(dynamicLabels)+len(manualLabels))

	// labels summary
	s.DoJSON("GET", "/api/latest/fleet/labels/summary", nil, http.StatusOK, &summaryResp)
	assert.Len(t, summaryResp.Labels, builtInsCount+len(dynamicLabels)+len(manualLabels))

	// next page is empty
	s.DoJSON("GET", "/api/latest/fleet/labels", nil, http.StatusOK, &listResp, "per_page", "100", "page", "1")
	assert.Len(t, listResp.Labels, 0)

	// list labels with invalid query params
	s.DoJSON("GET", "/api/latest/fleet/labels", nil, http.StatusBadRequest, &listResp, "per_page", strconv.Itoa(builtInsCount+1), "order_key", "id", "after", "1")
	s.DoJSON("GET", "/api/latest/fleet/labels", nil, http.StatusBadRequest, &listResp, "per_page", strconv.Itoa(builtInsCount+1), "query", "no match query for this endpoint")

	// create another dynamic label
	s.DoJSON("POST", "/api/latest/fleet/labels", &fleet.LabelPayload{Name: strings.ReplaceAll(t.Name(), "/", "_"), Query: "select 1"}, http.StatusOK, &createResp)
	assert.NotZero(t, createResp.Label.ID)
	lbl2 := createResp.Label.Label
	dynamicLabels = append(dynamicLabels, lbl2)
	require.Len(t, dynamicLabels, 2) // to make linter happy (dynamicLabels is not used past this point)

	// add lbl2 hosts to that label
	for _, h := range lbl2Hosts {
		err := s.ds.RecordLabelQueryExecutions(context.Background(), h, map[uint]*bool{lbl2.ID: ptr.Bool(true)}, time.Now(), false)
		require.NoError(t, err)
	}

	// list hosts in dynamic label lbl2
	var listHostsResp listHostsResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/labels/%d/hosts", lbl2.ID), nil, http.StatusOK, &listHostsResp)
	assert.Len(t, listHostsResp.Hosts, len(lbl2Hosts))

	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/labels/%d/hosts", lbl2.ID), nil, http.StatusOK, &listHostsResp, "order_key", "id", "after", fmt.Sprintf("%d", lbl2Hosts[0].ID))
	assert.Len(t, listHostsResp.Hosts, 2)
	assert.Equal(t, lbl2Hosts[1].ID, listHostsResp.Hosts[0].ID)
	assert.Equal(t, lbl2Hosts[2].ID, listHostsResp.Hosts[1].ID)

	// list hosts in manual label 1
	listHostsResp = listHostsResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/labels/%d/hosts", manualLbl1.ID), nil, http.StatusOK, &listHostsResp, "order_key", "id")
	assert.Len(t, listHostsResp.Hosts, manualLbl1.HostCount)
	assert.Equal(t, manualHosts[0].ID, listHostsResp.Hosts[0].ID)
	assert.Equal(t, manualHosts[1].ID, listHostsResp.Hosts[1].ID)
	assert.Equal(t, manualHosts[2].ID, listHostsResp.Hosts[2].ID)

	// list hosts in manual label 2
	listHostsResp = listHostsResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/labels/%d/hosts", manualLbl2.ID), nil, http.StatusOK, &listHostsResp, "order_key", "id")
	assert.Len(t, listHostsResp.Hosts, 0)

	// list hosts in dynamic label 2 searching by display_name
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/labels/%d/hosts", lbl2.ID), nil, http.StatusOK, &listHostsResp, "order_key", "display_name", "order_direction", "desc")
	assert.Len(t, listHostsResp.Hosts, len(lbl2Hosts))
	// first in the list is the last one, as the names are ordered with the index
	// of creation, and vice-versa
	assert.Equal(t, lbl2Hosts[len(lbl2Hosts)-1].ID, listHostsResp.Hosts[0].ID)
	assert.Equal(t, lbl2Hosts[0].ID, listHostsResp.Hosts[len(lbl2Hosts)-1].ID)

	mysql.ExecAdhocSQL(t, s.ds, func(db sqlx.ExtContext) error {
		_, err := db.ExecContext(
			context.Background(),
			`INSERT INTO host_emails (host_id, email, source) VALUES (?, ?, ?)`,
			lbl2Hosts[0].ID, "a@b.c", "src1")

		return err
	})

	// list hosts in label searching by email address
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/labels/%d/hosts", lbl2.ID), nil, http.StatusOK, &listHostsResp, "query", "a@b.c")
	assert.Len(t, listHostsResp.Hosts, 1)
	assert.Equal(t, lbl2Hosts[0].ID, listHostsResp.Hosts[0].ID)

	// list hosts in label searching by email address with leading/trailing whitespace
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/labels/%d/hosts", lbl2.ID), nil, http.StatusOK, &listHostsResp, "query", "    a@b.c   ")
	assert.Len(t, listHostsResp.Hosts, 1)
	assert.Equal(t, lbl2Hosts[0].ID, listHostsResp.Hosts[0].ID)

	// count hosts in label order by display_name
	var countResp countHostsResponse
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp, "label_id", fmt.Sprint(lbl2.ID), "order_key", "display_name", "order_direction", "desc")
	assert.Equal(t, len(lbl2Hosts), countResp.Count)

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
	require.NoError(t, s.ds.SetOrUpdateMDMData(context.Background(), lbl2Hosts[0].ID, false, true, "https://simplemdm.com", false, fleet.WellKnownMDMSimpleMDM, ""))
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

	// delete a manual label by id
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/labels/id/%d", manualLbl1.ID), nil, http.StatusOK, &delIDResp)

	// delete a manual label by name
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/labels/%s", url.PathEscape(manualLbl2.Name)), nil, http.StatusOK, &delResp)

	// list labels, only the built-ins remain
	s.DoJSON("GET", "/api/latest/fleet/labels", nil, http.StatusOK, &listResp, "per_page", strconv.Itoa(builtInsCount+1))
	assert.Len(t, listResp.Labels, builtInsCount)
	idsByName := make(map[string]uint, len(listResp.Labels))
	for _, lbl := range listResp.Labels {
		_, ok := builtinsMap[lbl.Name]
		assert.True(t, ok)
		assert.Equal(t, fleet.LabelTypeBuiltIn, lbl.LabelType)
		idsByName[lbl.Name] = lbl.ID
	}

	// labels summary, only the built-ins remains
	s.DoJSON("GET", "/api/latest/fleet/labels/summary", nil, http.StatusOK, &summaryResp)
	assert.Len(t, summaryResp.Labels, builtInsCount)
	for _, lbl := range summaryResp.Labels {
		_, ok := builtinsMap[lbl.Name]
		assert.True(t, ok)
		assert.Equal(t, fleet.LabelTypeBuiltIn, lbl.LabelType)
		assert.Equal(t, idsByName[lbl.Name], lbl.ID)
	}

	// host summary matches built-ins count
	var hostSummaryResp getHostSummaryResponse
	s.DoJSON("GET", "/api/latest/fleet/host_summary", nil, http.StatusOK, &hostSummaryResp)
	assert.Len(t, hostSummaryResp.BuiltinLabels, builtInsCount)
	for _, lbl := range hostSummaryResp.BuiltinLabels {
		_, ok := builtinsMap[lbl.Name]
		assert.True(t, ok)
		assert.Equal(t, fleet.LabelTypeBuiltIn, lbl.LabelType)
		assert.Equal(t, idsByName[lbl.Name], lbl.ID)
	}

	require.Len(t, idsByName, len(builtinsMap))
	for name := range builtinsMap {
		id, ok := idsByName[name]
		require.True(t, ok)

		// attempt to delete by name
		s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/labels/%s", url.PathEscape(name)), nil, http.StatusUnprocessableEntity, &delResp)

		// attempt to delete by id
		s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/labels/id/%d", id), nil, http.StatusUnprocessableEntity, &delIDResp)
	}
}

// Sanity test to make sure fleet/labels/<all>/hosts and fleet/hosts return the same thing.
func (s *integrationTestSuite) TestListHostsByLabel() {
	t := s.T()

	lblIDs, err := s.ds.LabelIDsByName(context.Background(), []string{"All Hosts"})
	require.NoError(t, err)
	require.Len(t, lblIDs, 1)
	labelID := lblIDs["All Hosts"]

	hosts := s.createHosts(t, "darwin")
	host := hosts[0]

	// Update label
	mysql.ExecAdhocSQL(
		t, s.ds, func(db sqlx.ExtContext) error {
			_, err := db.ExecContext(
				context.Background(),
				"INSERT IGNORE INTO label_membership (host_id, label_id) VALUES (?, (SELECT id FROM labels WHERE name = 'All Hosts' AND label_type = 1))",
				host.ID,
			)
			return err
		},
	)

	// set disk space information
	require.NoError(t, s.ds.SetOrUpdateHostDisksSpace(context.Background(), host.ID, 10.0, 2.0, 500.0)) // low disk

	// Update host fields
	host.Uptime = 30 * time.Second
	host.RefetchRequested = true
	host.OSVersion = "macOS 14.2"
	host.Build = "abc"
	host.PlatformLike = "darwin"
	host.CodeName = "sky"
	host.Memory = 1000
	host.CPUType = "arm64"
	host.CPUSubtype = "ARM64e"
	host.CPUBrand = "Apple M2 Pro"
	host.CPUPhysicalCores = 12
	host.CPULogicalCores = 14
	host.HardwareVendor = "Apple Inc."
	host.HardwareModel = "Mac14,10"
	host.HardwareVersion = "23"
	host.HardwareSerial = "ABC123"
	host.ComputerName = "MBP"
	host.PublicIP = "1.1.1.1"
	host.PrimaryIP = "10.10.10.10"
	host.PrimaryMac = "11:22:33"
	host.DistributedInterval = 10
	host.ConfigTLSRefresh = 9
	host.OsqueryVersion = "5.10"
	err = s.ds.UpdateHost(context.Background(), host)
	require.NoError(t, err)

	// Add team
	team, err := s.ds.NewTeam(
		context.Background(), &fleet.Team{
			Name: uuid.New().String(),
		},
	)
	require.NoError(t, err)
	require.NoError(t, s.ds.AddHostsToTeam(context.Background(), &team.ID, []uint{host.ID}))

	// Add pack
	_, err = s.ds.NewPack(
		context.Background(), &fleet.Pack{
			Name: t.Name(),
			Hosts: []fleet.Target{
				{
					Type:     fleet.TargetHost,
					TargetID: hosts[0].ID,
				},
			},
		},
	)
	require.NoError(t, err)

	// Add policy
	qr, err := s.ds.NewQuery(
		context.Background(), &fleet.Query{
			Name:           t.Name(),
			Description:    "Some description",
			Query:          "select * from osquery;",
			ObserverCanRun: true,
			Logging:        fleet.LoggingSnapshot,
		},
	)
	require.NoError(t, err)

	gpParams := globalPolicyRequest{
		QueryID:    &qr.ID,
		Resolution: "some global resolution",
	}
	gpResp := globalPolicyResponse{}
	s.DoJSON("POST", "/api/latest/fleet/policies", gpParams, http.StatusOK, &gpResp)
	require.NotNil(t, gpResp.Policy)
	require.NoError(
		t,
		s.ds.RecordPolicyQueryExecutions(context.Background(), host, map[uint]*bool{gpResp.Policy.ID: ptr.Bool(false)}, time.Now(), false),
	)

	// Add MDM info
	require.NoError(
		t,
		s.ds.SetOrUpdateMDMData(
			context.Background(), host.ID, false, true, "https://simplemdm.com", false, fleet.WellKnownMDMSimpleMDM, "",
		),
	)

	// Add device mapping
	require.NoError(
		t, s.ds.ReplaceHostDeviceMapping(
			context.Background(), host.ID, []*fleet.HostDeviceMapping{
				{HostID: hosts[0].ID, Email: "a@b.c", Source: fleet.DeviceMappingGoogleChromeProfiles},
				{HostID: hosts[0].ID, Email: "b@b.c", Source: fleet.DeviceMappingGoogleChromeProfiles},
			}, fleet.DeviceMappingGoogleChromeProfiles,
		),
	)

	// Now do the actual API calls that we will compare.
	var hostsResp, labelsResp listHostsResponse
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &hostsResp, "device_mapping", "true")
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/labels/%d/hosts", labelID), nil, http.StatusOK, &labelsResp, "device_mapping", "true")

	// Converting to formatted JSON for easier diffs
	hostsJson, _ := json.MarshalIndent(hostsResp, "", "  ")
	labelsJson, _ := json.MarshalIndent(labelsResp, "", "  ")
	assert.Equal(t, string(hostsJson), string(labelsJson))
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
	}, http.StatusUnprocessableEntity, &applyResp,
	)

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
	}, http.StatusUnprocessableEntity, &applyResp,
	)

	// apply an invalid label spec - builtin label type
	s.DoJSON("POST", "/api/latest/fleet/spec/labels", applyLabelSpecsRequest{
		Specs: []*fleet.LabelSpec{
			{
				Name:                name,
				Query:               "select 1",
				Platform:            "linux",
				LabelMembershipType: fleet.LabelMembershipTypeDynamic,
				LabelType:           fleet.LabelTypeBuiltIn,
			},
		},
	}, http.StatusUnprocessableEntity, &applyResp)

	// apply an invalid label spec - builtin label name
	for n := range fleet.ReservedLabelNames() {
		s.DoJSON("POST", "/api/latest/fleet/spec/labels", applyLabelSpecsRequest{
			Specs: []*fleet.LabelSpec{
				{
					Name:                n,
					Query:               "select 1",
					Platform:            "linux",
					LabelMembershipType: fleet.LabelMembershipTypeDynamic,
				},
			},
		}, http.StatusUnprocessableEntity, &applyResp)
	}

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

	// existing users:
	// {ID: 1, Name: "Test Name admin1@example.com", Email: "admin1@example.com", ...}
	// {ID: 2, Name: "Test Name user1@example.com", Email: "user1@example.com", ...}
	// {ID: 3, Name: "Test Name user2@example.com", Email: "user2@example.com", ...}

	// list existing users
	var listResp listUsersResponse
	s.DoJSON("GET", "/api/latest/fleet/users", nil, http.StatusOK, &listResp)
	assert.Len(t, listResp.Users, len(s.users))

	// with non-matching query
	s.DoJSON("GET", "/api/latest/fleet/users", nil, http.StatusOK, &listResp, "query", "noone")
	assert.Len(t, listResp.Users, 0)

	// with matching query containing leading/trailing whitespaces
	s.DoJSON("GET", "/api/latest/fleet/users", nil, http.StatusOK, &listResp, "query", " user 	")
	assert.Len(t, listResp.Users, 2)
	assert.Equal(t, uint(2), listResp.Users[0].ID)
	assert.Equal(t, uint(3), listResp.Users[1].ID)

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
		MFAEnabled: ptr.Bool(true),
	}
	// mailer isn't set up, which fails prior to silently ignoring MFA
	s.DoJSON("POST", "/api/latest/fleet/users/admin", params, http.StatusBadRequest, &createResp)
	params.MFAEnabled = nil
	s.DoJSON("POST", "/api/latest/fleet/users/admin", params, http.StatusOK, &createResp)
	assert.NotZero(t, createResp.User.ID)
	assert.True(t, createResp.User.AdminForcedPasswordReset)
	u := *createResp.User

	var loginResp loginResponse

	// try MFA
	mysql.ExecAdhocSQL(t, s.ds, func(db sqlx.ExtContext) error {
		_, err := db.ExecContext(context.Background(), `UPDATE users SET mfa_enabled = TRUE WHERE id = ?`, u.ID)
		return err
	})
	s.DoJSONWithoutAuth("POST", "/api/latest/fleet/sessions", sessionCreateRequest{Token: "foo"}, http.StatusUnauthorized, &loginResp)
	// MFA unsupported client
	s.DoJSONWithoutAuth("POST", "/api/latest/fleet/login", params, http.StatusBadRequest, &loginResp)
	// MFA supported; send email
	s.DoJSONWithoutAuth("POST", "/api/latest/fleet/login", loginRequest{Email: "extra@asd.com", Password: userRawPwd, SupportsEmailVerification: true}, http.StatusAccepted, &loginResp)
	var mfaToken string
	mysql.ExecAdhocSQL(t, s.ds, func(tx sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), tx, &mfaToken, `SELECT token FROM verification_tokens WHERE user_id = ? LIMIT 1`, createResp.User.ID)
	})
	// create session from MFA token
	s.DoJSONWithoutAuth("POST", "/api/latest/fleet/sessions", sessionCreateRequest{Token: mfaToken}, http.StatusOK, &loginResp)
	// can't use the same MFA token twice
	s.DoJSONWithoutAuth("POST", "/api/latest/fleet/sessions", sessionCreateRequest{Token: mfaToken}, http.StatusUnauthorized, &loginResp)

	// send another email, which we'll expire the token for
	s.DoJSONWithoutAuth("POST", "/api/latest/fleet/login", loginRequest{Email: "extra@asd.com", Password: userRawPwd, SupportsEmailVerification: true}, http.StatusAccepted, &loginResp)
	mysql.ExecAdhocSQL(t, s.ds, func(db sqlx.ExtContext) error {
		_, err := db.ExecContext(
			context.Background(),
			`UPDATE verification_tokens SET created_at = NOW() - INTERVAL ? SECOND - INTERVAL 0.5 SECOND WHERE user_id = ?`,
			(time.Minute * 15).Seconds(),
			u.ID,
		)
		if err != nil {
			return err
		}

		return sqlx.GetContext(context.Background(), db, &mfaToken, `SELECT token FROM verification_tokens WHERE user_id = ? LIMIT 1`, createResp.User.ID)
	})
	s.DoJSONWithoutAuth("POST", "/api/latest/fleet/sessions", sessionCreateRequest{Token: mfaToken}, http.StatusUnauthorized, &loginResp)

	// turn off MFA
	var modResp modifyUserResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/users/%d", u.ID), fleet.UserPayload{MFAEnabled: ptr.Bool(false)}, http.StatusOK, &modResp)
	require.False(t, modResp.User.MFAEnabled)

	// can't turn MFA back on
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/users/%d", u.ID), fleet.UserPayload{MFAEnabled: ptr.Bool(true)}, http.StatusPaymentRequired, &modResp)

	// login as that user and check that teams info is empty
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
		MFAEnabled: ptr.Bool(true),
	}}
	createInviteResp := createInviteResponse{}
	s.DoJSON("POST", "/api/latest/fleet/invites", createInviteReq, http.StatusPaymentRequired, &createInviteResp)
	createInviteReq.MFAEnabled = nil
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

	var perfPwdResetResp performRequiredPasswordResetResponse
	newRawPwd := test.GoodPassword2
	// Try a required password change without authentication
	s.DoJSON(
		"POST", "/api/latest/fleet/perform_required_password_reset", performRequiredPasswordResetRequest{
			Password: newRawPwd,
			ID:       u.ID,
		}, http.StatusForbidden, &perfPwdResetResp,
	)

	// perform a required password change as the user themselves
	s.token = s.getTestToken(u.Email, userRawPwd)
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
	}, http.StatusForbidden, &perfPwdResetResp)
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

func (s *integrationTestSuite) TestActivitiesWebhookConfig() {
	t := s.T()

	s.DoRaw(
		"PATCH", "/api/latest/fleet/config", []byte(
			`{
		"webhook_settings": {
			"activities_webhook": {
				"enable_activities_webhook": true,
				"destination_url": "http://some/url"
    		}
  		}
	}`,
		), http.StatusOK,
	)

	s.lastActivityOfTypeMatches(
		fleet.ActivityTypeEnabledActivityAutomations{}.ActivityName(),
		`{"webhook_url": "http://some/url"}`,
		0,
	)

	appConfig := s.getConfig()
	require.True(t, appConfig.WebhookSettings.ActivitiesWebhook.Enable)
	require.Equal(t, "http://some/url", appConfig.WebhookSettings.ActivitiesWebhook.DestinationURL)

	s.DoRaw(
		"PATCH", "/api/latest/fleet/config", []byte(
			`{
		"webhook_settings": {
			"activities_webhook": {
				"enable_activities_webhook": true,
				"destination_url": "http://some/other/url"
    		}
  		}
	}`,
		), http.StatusOK,
	)

	s.lastActivityOfTypeMatches(
		fleet.ActivityTypeEditedActivityAutomations{}.ActivityName(),
		`{"webhook_url": "http://some/other/url"}`,
		0,
	)

	s.DoRaw(
		"PATCH", "/api/latest/fleet/config", []byte(
			`{
		"webhook_settings": {
			"activities_webhook": {
				"enable_activities_webhook": true,
				"destination_url": "invalid-url"
    		}
  		}
	}`,
		), http.StatusUnprocessableEntity,
	)

	s.lastActivityOfTypeMatches(
		fleet.ActivityTypeEditedActivityAutomations{}.ActivityName(),
		`{"webhook_url": "http://some/other/url"}`,
		0,
	)

	s.DoRaw(
		"PATCH", "/api/latest/fleet/config", []byte(
			`{
		"webhook_settings": {
			"activities_webhook": {
				"enable_activities_webhook": false
    		}
  		}
	}`,
		), http.StatusOK,
	)

	s.lastActivityOfTypeMatches(
		fleet.ActivityTypeDisabledActivityAutomations{}.ActivityName(),
		``,
		0,
	)

	s.DoRaw(
		"PATCH", "/api/latest/fleet/config", []byte(
			`{
		"webhook_settings": {
			"activities_webhook": {
				"enable_activities_webhook": true,
				"destination_url": "foo.baz"
    		}
  		}
	}`,
		), http.StatusUnprocessableEntity,
	)

	s.lastActivityOfTypeMatches(
		fleet.ActivityTypeEnabledActivityAutomations{}.ActivityName(),
		`{"webhook_url": "http://some/url"}`,
		0,
	)
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

	// make an unrelated appconfig change, should not remove the integrations
	var appCfgResp appConfigResponse
	s.DoJSON("PATCH", "/api/v1/fleet/config", json.RawMessage(`{
		"org_info": {
			"org_name": "test-integrations"
		}
	}`), http.StatusOK, &appCfgResp)
	require.Equal(t, "test-integrations", appCfgResp.OrgInfo.OrgName)
	require.Len(t, appCfgResp.Integrations.Jira, 2)

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

	// if no jira nor zendesk integrations are provided, does not remove integrations
	appCfgResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/v1/fleet/config", json.RawMessage(`{
		"integrations": {}
	}`), http.StatusOK, &appCfgResp)
	require.Len(t, appCfgResp.Integrations.Jira, 1)

	// if explicitly-empty arrays are provided, remove all integrations
	appCfgResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/v1/fleet/config", json.RawMessage(`{
		"integrations": {
			"jira": [],
			"zendesk": []
		}
	}`), http.StatusOK, &appCfgResp)
	require.Len(t, appCfgResp.Integrations.Jira, 0)

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

	// make an unrelated appconfig change, should not remove the integrations
	appCfgResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/v1/fleet/config", json.RawMessage(`{
		"org_info": {
			"org_name": "test-integrations-zendesk"
		}
	}`), http.StatusOK, &appCfgResp)
	require.Equal(t, "test-integrations-zendesk", appCfgResp.OrgInfo.OrgName)
	require.Len(t, appCfgResp.Integrations.Zendesk, 2)

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

	// if no jira nor zendesk integrations are provided, does not remove integrations
	appCfgResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/v1/fleet/config", json.RawMessage(`{
		"integrations": {}
	}`), http.StatusOK, &appCfgResp)
	require.Len(t, appCfgResp.Integrations.Jira, 1)
	require.Len(t, appCfgResp.Integrations.Zendesk, 1)

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

func (s *integrationTestSuite) TestGoogleCalendarIntegrations() {
	t := s.T()
	email := "service-account@example.com"
	privateKey := "-----BEGIN PRIVATE KEY-----\nXXXXX\n-----END"
	domain := "example.com"
	s.DoRaw(
		"PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(
			`{
		"integrations": {
			"google_calendar": [{
				"api_key_json": {
					"client_email": %q,
					"private_key": %q
				},
				"domain": %q
			}]
		}
	}`, email, privateKey, domain,
		)), http.StatusOK,
	)

	appConfig := s.getConfig()
	require.Len(t, appConfig.Integrations.GoogleCalendar, 1)
	assert.Equal(t, email, appConfig.Integrations.GoogleCalendar[0].ApiKey[fleet.GoogleCalendarEmail])
	assert.Equal(t, privateKey, appConfig.Integrations.GoogleCalendar[0].ApiKey[fleet.GoogleCalendarPrivateKey])
	assert.Equal(t, domain, appConfig.Integrations.GoogleCalendar[0].Domain)

	// Add 2nd config -- not allowed at this time
	s.DoRaw(
		"PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(
			`{
		"integrations": {
			"google_calendar": [{
				"api_key_json": {
					"client_email": %q,
					"private_key": %q
				},
				"domain": %q
			},
			{
				"api_key_json": {
					"client_email": "bozo@example.com",
					"private_key": "abc"
				},
				"domain": "example.com"
			}]
		}
	}`, email, privateKey, domain,
		)), http.StatusUnprocessableEntity,
	)

	// Make an unrelated config change, should not remove the integrations
	var appCfgResp appConfigResponse
	s.DoJSON(
		"PATCH", "/api/v1/fleet/config", json.RawMessage(
			`{
		"org_info": {
			"org_name": "test-google-calendar-integrations"
		}
	}`,
		), http.StatusOK, &appCfgResp,
	)
	require.Equal(t, "test-google-calendar-integrations", appCfgResp.OrgInfo.OrgName)
	require.Len(t, appCfgResp.Integrations.GoogleCalendar, 1)

	// Update calendar config
	domain = "new.com"
	s.DoRaw(
		"PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(
			`{
		"integrations": {
			"google_calendar": [{
				"api_key_json": {
					"client_email": %q,
					"private_key": %q
				},
				"domain": %q
			}]
		}
	}`, email, privateKey, domain,
		)), http.StatusOK,
	)
	appConfig = s.getConfig()
	require.Len(t, appConfig.Integrations.GoogleCalendar, 1)
	assert.Equal(t, email, appConfig.Integrations.GoogleCalendar[0].ApiKey[fleet.GoogleCalendarEmail])
	assert.Equal(t, privateKey, appConfig.Integrations.GoogleCalendar[0].ApiKey[fleet.GoogleCalendarPrivateKey])
	assert.Equal(t, domain, appConfig.Integrations.GoogleCalendar[0].Domain)

	// Clearing other integrations does not clear Google Calendar integration
	appCfgResp = appConfigResponse{}
	s.DoJSON(
		"PATCH", "/api/v1/fleet/config", json.RawMessage(
			`{
		"integrations": {
			"jira": [],
			"zendesk": []
		}
	}`,
		), http.StatusOK, &appCfgResp,
	)
	require.Len(t, appCfgResp.Integrations.GoogleCalendar, 1)

	// Clearing Google Calendar integration
	appCfgResp = appConfigResponse{}
	s.DoJSON(
		"PATCH", "/api/v1/fleet/config", json.RawMessage(
			`{
		"integrations": {
			"google_calendar": []
		}
	}`,
		), http.StatusOK, &appCfgResp,
	)
	assert.Empty(t, appCfgResp.Integrations.GoogleCalendar)

	// Try adding Google Calendar integration without sending private key -- not allowed
	s.DoRaw(
		"PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(
			`{
		"integrations": {
			"google_calendar": [{
				"api_key_json": {
					"client_email": %q
				},
				"domain": %q
			}]
		}
	}`, email, domain,
		)), http.StatusUnprocessableEntity,
	)

	// Empty email -- not allowed
	s.DoRaw(
		"PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(
			`{
		"integrations": {
			"google_calendar": [{
				"api_key_json": {
					"client_email": " ",
					"private_key": %q
				},
				"domain": %q
			}]
		}
	}`, privateKey, domain,
		)), http.StatusUnprocessableEntity,
	)

	// Empty domain -- not allowed
	s.DoRaw(
		"PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(
			`{
		"integrations": {
			"google_calendar": [{
				"api_key_json": {
					"client_email": %q,
					"private_key": %q
				},
				"domain": ""
			}]
		}
	}`, email, privateKey,
		)), http.StatusUnprocessableEntity,
	)

	// Unknown fields fails as bad request
	s.DoRaw(
		"PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(
			`{
		"integrations": {
			"google_calendar": [{
				"api_key_json": {
					"client_email": %q,
					"private_key": %q
				},
				"domain": %q,
				"foo": "bar"
			}]
		}
	}`, email, privateKey, domain,
		)), http.StatusBadRequest,
	)

	// Null api_key_json -- fails validation
	s.DoRaw(
		"PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(
			`{
		"integrations": {
			"google_calendar": [{
				"api_key_json": null,
				"domain": %q
			}]
		}
	}`, domain,
		)), http.StatusUnprocessableEntity,
	)
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
	defer s.cleanupQuery(existingQueryID)

	for _, tc := range []struct {
		tname    string
		name     string
		query    string
		platform string
		logging  string
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
		{
			tname:    "unsupported platform",
			name:     "bad query",
			query:    "select 1",
			platform: "oops",
		},
		{
			tname:    "unsupported platform",
			name:     "bad query",
			query:    "select 1",
			platform: "charles,darwin",
		},
		{
			tname:    "missing platform comma delimeter",
			name:     "bad query",
			query:    "select 1",
			platform: "linuxdarwin",
		},
		{
			tname:    "missing platform comma delimeter",
			name:     "bad query",
			query:    "select 1",
			platform: "windows darwin",
		},
		{
			tname:   "invalid logging value",
			name:    "bad query",
			query:   "select 1",
			logging: "foobar",
		},
	} {
		t.Run(tc.tname, func(t *testing.T) {
			reqQuery := &fleet.QueryPayload{
				Name:     ptr.String(tc.name),
				Query:    ptr.String(tc.query),
				Platform: ptr.String(tc.platform),
				Logging:  ptr.String(tc.logging),
			}
			createQueryResp := createQueryResponse{}
			s.DoJSON("POST", "/api/latest/fleet/queries", reqQuery, http.StatusBadRequest, &createQueryResp)
			require.Nil(t, createQueryResp.Query)

			payload := fleet.QueryPayload{
				Name:     ptr.String(tc.name),
				Query:    ptr.String(tc.query),
				Platform: ptr.String(tc.platform),
				Logging:  ptr.String(tc.logging),
			}
			mResp := modifyQueryResponse{}
			s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/queries/%d", existingQueryID), &payload, http.StatusBadRequest, &mResp)
			require.Nil(t, mResp.Query)
			// TODO – add checks for specific errors
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
	teamSpecs := applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{Name: "newteam", Secrets: &[]fleet.EnrollSecret{{Secret: "ABC"}}}}}
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

	// update MDM disk encryption
	_ = s.Do("POST", "/api/latest/fleet/disk_encryption", fleet.MDMAppleSettingsPayload{}, http.StatusPaymentRequired)

	// device migrate mdm endpoint returns an error if not premium
	createHostAndDeviceToken(t, s.ds, "some-token")
	s.Do("POST", fmt.Sprintf("/api/v1/fleet/device/%s/migrate_mdm", "some-token"), nil, http.StatusPaymentRequired)

	// software titles
	// a normal request works fine
	var resp listSoftwareTitlesResponse
	s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{}, http.StatusOK, &resp)
	// TODO: there's a race condition that makes this number change from
	// 0-3, commenting for now since it's not really relevant for this
	// test (we only care about the response status)
	// require.NotEmpty(t, 0, resp.Count)
	// require.Nil(t, resp.SoftwareTitles)

	// a request with a team_id parameter returns a license error
	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{}, http.StatusPaymentRequired, &resp,
		"team_id", "1",
	)

	// a request with a premium vulnerability filter returns a license error
	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{fleet.SoftwareTitleListOptions{VulnerableOnly: true, MinimumCVSS: 7.5}}, http.StatusPaymentRequired, &resp,
	)
	verResp := listSoftwareVersionsResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/versions",
		listSoftwareRequest{fleet.SoftwareListOptions{VulnerableOnly: true, MinimumCVSS: 7.5}}, http.StatusPaymentRequired, &verResp,
	)
	countResp := countSoftwareResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/count",
		listSoftwareRequest{fleet.SoftwareListOptions{VulnerableOnly: true, MinimumCVSS: 7.5}}, http.StatusPaymentRequired, &countResp,
	)

	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{fleet.SoftwareTitleListOptions{VulnerableOnly: true, MaximumCVSS: 7.5}}, http.StatusPaymentRequired, &resp,
	)
	verResp = listSoftwareVersionsResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/versions",
		listSoftwareRequest{fleet.SoftwareListOptions{VulnerableOnly: true, MaximumCVSS: 7.5}}, http.StatusPaymentRequired, &verResp,
	)
	countResp = countSoftwareResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/count",
		listSoftwareRequest{fleet.SoftwareListOptions{VulnerableOnly: true, MaximumCVSS: 7.5}}, http.StatusPaymentRequired, &countResp,
	)

	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{fleet.SoftwareTitleListOptions{VulnerableOnly: true, KnownExploit: true}}, http.StatusPaymentRequired, &resp,
	)
	verResp = listSoftwareVersionsResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/versions",
		listSoftwareRequest{fleet.SoftwareListOptions{VulnerableOnly: true, KnownExploit: true}}, http.StatusPaymentRequired, &verResp,
	)
	countResp = countSoftwareResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/count",
		listSoftwareRequest{fleet.SoftwareListOptions{VulnerableOnly: true, KnownExploit: true}}, http.StatusPaymentRequired, &countResp,
	)

	// lock/unlock/wipe a host
	s.Do("POST", "/api/v1/fleet/hosts/123/lock", nil, http.StatusPaymentRequired)
	s.Do("POST", "/api/v1/fleet/hosts/123/unlock", nil, http.StatusPaymentRequired)
	s.Do("POST", "/api/v1/fleet/hosts/123/wipe", nil, http.StatusPaymentRequired)

	// try to update the enable_release_device_manually setting, requires premium
	// (but /setup_experience catches the error of the MDM middleware check, so not
	// StatusPaymentRequired)
	res = s.Do("PATCH", "/api/v1/fleet/setup_experience", fleet.MDMAppleSetupPayload{EnableReleaseDeviceManually: ptr.Bool(true)}, http.StatusBadRequest)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, fleet.ErrMDMNotConfigured.Error())

	res = s.Do("PATCH", "/api/v1/fleet/config", json.RawMessage(`{
		"mdm": { "macos_setup": { "enable_release_device_manually": true } }
	}`), http.StatusUnprocessableEntity)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "missing or invalid license")

	res = s.Do("PATCH", "/api/v1/fleet/config", json.RawMessage(`{
		"mdm": { "windows_migration_enabled": true }
	}`), http.StatusUnprocessableEntity)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "missing or invalid license")
}

func (s *integrationTestSuite) TestScriptsEndpointsWithoutLicense() {
	t := s.T()

	// this is just checking that the endpoints do not fail with "no license", the actual tests
	// for scripts endpoints are in the enterprise integrations tests.

	// run a script
	var runResp runScriptResponse
	s.DoJSON("POST", "/api/latest/fleet/scripts/run", fleet.HostScriptRequestPayload{HostID: 1, ScriptContents: "echo foo"}, http.StatusNotFound, &runResp)

	// run a script sync
	s.DoJSON("POST", "/api/latest/fleet/scripts/run/sync", fleet.HostScriptRequestPayload{HostID: 1, ScriptContents: "echo foo"}, http.StatusNotFound, &runResp)

	// get script result
	var scriptResultResp getScriptResultResponse
	s.DoJSON("GET", "/api/latest/fleet/scripts/results/test-id", nil, http.StatusNotFound, &scriptResultResp)

	// create a saved script
	body, headers := generateNewScriptMultipartRequest(t,
		"myscript.sh", []byte(`echo "hello"`), s.token, nil)
	s.DoRawWithHeaders("POST", "/api/latest/fleet/scripts", body.Bytes(), http.StatusOK, headers)

	// run a saved script by name without team id (should fail host not found)
	res := s.Do("POST", "/api/latest/fleet/scripts/run/sync", runScriptSyncRequest{ScriptName: "myscript.sh"}, http.StatusNotFound)
	errMsg := extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Host was not found in the datastore")

	// run a saved script by name with team id (should fail with license error)
	res = s.Do("POST", "/api/latest/fleet/scripts/run/sync", runScriptSyncRequest{ScriptName: "myscript.sh", TeamID: 1}, http.StatusPaymentRequired)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Requires Fleet Premium license")

	// delete a saved script
	var delScriptResp deleteScriptResponse
	s.DoJSON("DELETE", "/api/latest/fleet/scripts/123", nil, http.StatusNotFound, &delScriptResp)

	// list saved scripts
	var listScriptsResp listScriptsResponse
	s.DoJSON("GET", "/api/latest/fleet/scripts", nil, http.StatusOK, &listScriptsResp, "per_page", "10")

	// get a saved script
	var getScriptResp getScriptResponse
	s.DoJSON("GET", "/api/latest/fleet/scripts/123", nil, http.StatusNotFound, &getScriptResp)

	// get host script details
	var getHostScriptDetailsResp getHostScriptDetailsResponse
	s.DoJSON("GET", "/api/latest/fleet/hosts/123/scripts", nil, http.StatusNotFound, &getHostScriptDetailsResp)

	// batch set scripts
	s.Do("POST", "/api/v1/fleet/scripts/batch", batchSetScriptsRequest{Scripts: nil}, http.StatusOK)
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

	deleteTeamPoliciesResponse := deleteTeamPoliciesResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/delete", 9999999), deleteTeamPoliciesRequest{IDs: []uint{1, 1000}}, http.StatusNotFound, &deleteTeamPoliciesResponse)
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

	// get info about session
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/sessions/%d", ssn.ID+1), nil, http.StatusNotFound, &getResp)

	// delete session
	var delResp deleteSessionResponse
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/sessions/%d", ssn.ID), nil, http.StatusOK, &delResp)

	// delete session - non-existing
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/sessions/%d", ssn.ID), nil, http.StatusNotFound, &delResp)
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
	assert.False(t, acResp.ActivityExpirySettings.ActivityExpiryEnabled)
	assert.Zero(t, acResp.ActivityExpirySettings.ActivityExpiryWindow)
	assert.False(t, acResp.ServerSettings.AIFeaturesDisabled)

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

	// Invalid activity expiry window.
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
    "activity_expiry_settings": {
        "activity_expiry_enabled": true,
        "activity_expiry_window": -1
    }
  }`), http.StatusUnprocessableEntity, &acResp)
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	require.False(t, acResp.ActivityExpirySettings.ActivityExpiryEnabled)
	require.Zero(t, acResp.ActivityExpirySettings.ActivityExpiryWindow)

	// Valid activity expiry window.
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
    "activity_expiry_settings": {
        "activity_expiry_enabled": true,
        "activity_expiry_window": 42
    }
  }`), http.StatusOK, &acResp)
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	require.True(t, acResp.ActivityExpirySettings.ActivityExpiryEnabled)
	require.Equal(t, 42, acResp.ActivityExpirySettings.ActivityExpiryWindow)

	// Disable AI features.
	acResp = appConfigResponse{}
	s.DoJSON(
		"PATCH", "/api/latest/fleet/config", json.RawMessage(
			`{
    "server_settings": {
        "ai_features_disabled": true
    }
  }`,
		), http.StatusOK, &acResp,
	)
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	assert.True(t, acResp.ServerSettings.AIFeaturesDisabled)

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
		"mdm": { "enable_disk_encryption": true }
  }`), http.StatusUnprocessableEntity)
	errMsg := extractServerErrorText(res.Body)
	assert.Contains(t, errMsg, "missing or invalid license")

	// legacy config
	res = s.Do("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "macos_settings": { "enable_disk_encryption": true } }
  }`), http.StatusUnprocessableEntity)
	errMsg = extractServerErrorText(res.Body)
	assert.Contains(t, errMsg, "missing or invalid license")

	// try to set the apple bm default team, which is premium only
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "apple_bm_default_team": "xyz" }
  }`), http.StatusUnprocessableEntity, &acResp)

	// try to set the windows updates, which is premium only
	res = s.Do("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "windows_updates": {"deadline_days": 1, "grace_period_days": 0} }
  }`), http.StatusUnprocessableEntity)
	errMsg = extractServerErrorText(res.Body)
	assert.Contains(t, errMsg, "missing or invalid license")

	// try to enable Windows MDM, impossible without the WSTEP certs
	// (only set in mdm integrations tests)
	res = s.Do("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "windows_enabled_and_configured": true }
  }`), http.StatusUnprocessableEntity)
	errMsg = extractServerErrorText(res.Body)
	assert.Contains(t, errMsg, "Please configure Fleet with a certificate and key pair first.")

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

	// try to create a query with invalid platform, fail
	q4 := q1 + "_4"
	s.DoJSON("POST", "/api/latest/fleet/spec/queries", applyQuerySpecsRequest{
		Specs: []*fleet.QuerySpec{
			{Name: q4, Query: "SELECT 4", Platform: "not valid"},
		},
	}, http.StatusBadRequest, &applyResp)

	// try to edit a query with invalid platform, fail
	s.DoJSON("POST", "/api/latest/fleet/spec/queries", applyQuerySpecsRequest{
		Specs: []*fleet.QuerySpec{
			{Name: q3, Query: "SELECT 3", Platform: "charles darwin"},
		},
	}, http.StatusBadRequest, &applyResp)

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

func (s *integrationTestSuite) TestListSoftwareAndSoftwareDetails() {
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
		sw := fleet.Software{Name: fmt.Sprintf("sw%02d", i), Version: fmt.Sprintf("0.0.%02d", i), Source: "apps"}
		if i%2 == 0 {
			sw.Source = "chrome_extensions"
			sw.Browser = "chrome"
		}
		sws[i] = sw
	}

	sortByNameAlphanumeric := func(sw []fleet.Software, a, b int) bool {
		aNum, _ := strconv.Atoi(strings.TrimPrefix(sw[a].Name, "sw"))
		bNum, _ := strconv.Atoi(strings.TrimPrefix(sw[b].Name, "sw"))
		return aNum < bNum
	}
	sortEntryByNameAlphanumeric := func(sw []fleet.HostSoftwareEntry, a, b int) bool {
		aNum, _ := strconv.Atoi(strings.TrimPrefix(sw[a].Name, "sw"))
		bNum, _ := strconv.Atoi(strings.TrimPrefix(sw[b].Name, "sw"))
		return aNum < bNum
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
			// Sort software by Name (alphanumeric)
			sort.Slice(
				sws, func(a, b int) bool {
					return sortByNameAlphanumeric(sws, a, b)
				},
			)
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
	// Sort software by Name (alphanumeric)
	sort.Slice(
		hosts[0].Software, func(a, b int) bool {
			return sortEntryByNameAlphanumeric(hosts[0].Software, a, b)
		},
	)
	testCvePrefix := "cve-123-123"
	for i, sw := range hosts[0].Software[:10] {
		inserted, err := s.ds.InsertSoftwareVulnerability(context.Background(), fleet.SoftwareVulnerability{
			SoftwareID: sw.ID,
			CVE:        fmt.Sprintf(testCvePrefix+"-%03d", i),
		}, fleet.NVDSource)
		require.NoError(t, err)
		require.True(t, inserted)
	}
	expectedVulnVersionsCount := 10

	// create a team and make the last 3 hosts part of it (meaning 3 that use
	// sws[19], 2 for sws[18], and 1 for sws[17])
	tm, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		Name: t.Name(),
	})
	require.NoError(t, err)
	require.NoError(t, s.ds.AddHostsToTeam(context.Background(), &tm.ID, []uint{hosts[19].ID, hosts[18].ID, hosts[17].ID}))
	expectedTeamVersionsCount := 3

	assertSoftwareDetails := func(expectedSoftware []fleet.Software, team string) {
		t.Helper()
		// this is just a basic sanity check of the software details endpoints and doesn't test all of the
		// fields that may be present in the response (e.g., vulnerabilities)
		for _, sw := range expectedSoftware {
			var detailsResp getSoftwareResponse
			s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/%d", sw.ID), nil, http.StatusOK, &detailsResp, "team_id", team)
			assert.Equal(t, sw.ID, detailsResp.Software.ID)
			assert.Equal(t, sw.Name, detailsResp.Software.Name)
			assert.Equal(t, sw.Version, detailsResp.Software.Version)
			assert.Equal(t, sw.Source, detailsResp.Software.Source)
			assert.Equal(t, sw.Browser, detailsResp.Software.Browser)

			detailsResp = getSoftwareResponse{}
			s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/versions/%d", sw.ID), nil, http.StatusOK, &detailsResp, "team_id", team)
			assert.Equal(t, sw.ID, detailsResp.Software.ID)
			assert.Equal(t, sw.Name, detailsResp.Software.Name)
			assert.Equal(t, sw.Version, detailsResp.Software.Version)
			assert.Equal(t, sw.Source, detailsResp.Software.Source)
			assert.Equal(t, sw.Browser, detailsResp.Software.Browser)
			if len(sw.Vulnerabilities) > 0 {
				assert.Len(t, detailsResp.Software.Vulnerabilities, len(sw.Vulnerabilities))
				assert.Greater(t, detailsResp.Software.Vulnerabilities[0].CreatedAt, time.Now().Add(-time.Hour)) // asserting a non-zero time
			}
		}
	}

	assertResp := func(resp listSoftwareResponse, want []fleet.Software, ts time.Time, team string, counts ...int) {
		t.Helper()
		require.Len(t, resp.Software, len(want))
		for i := range resp.Software {
			wantID, gotID := want[i].ID, resp.Software[i].ID
			assert.Equal(t, wantID, gotID, "want.Name: %s got.Name: %s", want[i].Name, resp.Software[i].Name)
			wantName, gotName := want[i].Name, resp.Software[i].Name
			assert.Equal(t, wantName, gotName)
			wantVersion, gotVersion := want[i].Version, resp.Software[i].Version
			assert.Equal(t, wantVersion, gotVersion)
			wantSource, gotSource := want[i].Source, resp.Software[i].Source
			assert.Equal(t, wantSource, gotSource)
			wantBrowser, gotBrowser := want[i].Browser, resp.Software[i].Browser
			assert.Equal(t, wantBrowser, gotBrowser)
			wantCount, gotCount := counts[i], resp.Software[i].HostsCount
			assert.Equal(t, wantCount, gotCount)
		}
		if ts.IsZero() {
			assert.Nil(t, resp.CountsUpdatedAt)
		} else {
			require.NotNil(t, resp.CountsUpdatedAt)
			assert.WithinDuration(t, ts, *resp.CountsUpdatedAt, time.Second)
		}
		assertSoftwareDetails(resp.Software, team)
	}

	assertVersionsResp := func(
		resp listSoftwareVersionsResponse, want []fleet.Software, ts time.Time, team string, swCount int, hostCounts ...int,
	) {
		require.Equal(t, swCount, resp.Count)
		require.Len(t, resp.Software, len(want))
		for i := range resp.Software {
			wantID, gotID := want[i].ID, resp.Software[i].ID
			assert.Equal(t, wantID, gotID)
			wantCount, gotCount := hostCounts[i], resp.Software[i].HostsCount
			assert.Equal(t, wantCount, gotCount)
			wantName, gotName := want[i].Name, resp.Software[i].Name
			assert.Equal(t, wantName, gotName)
			wantVersion, gotVersion := want[i].Version, resp.Software[i].Version
			assert.Equal(t, wantVersion, gotVersion)
			wantSource, gotSource := want[i].Source, resp.Software[i].Source
			assert.Equal(t, wantSource, gotSource)
			wantBrowser, gotBrowser := want[i].Browser, resp.Software[i].Browser
			assert.Equal(t, wantBrowser, gotBrowser)
		}
		if ts.IsZero() {
			assert.Nil(t, resp.CountsUpdatedAt)
		} else {
			require.NotNil(t, resp.CountsUpdatedAt)
			assert.WithinDuration(t, ts, *resp.CountsUpdatedAt, time.Second)
		}
		assertSoftwareDetails(resp.Software, team)
	}

	// no software host counts have been calculated yet, so this returns nothing
	var lsResp listSoftwareResponse
	s.DoJSON("GET", "/api/latest/fleet/software", nil, http.StatusOK, &lsResp, "order_key", "hosts_count", "order_direction", "desc")
	assertResp(lsResp, nil, time.Time{}, "")
	var versResp listSoftwareVersionsResponse
	s.DoJSON("GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, &versResp, "order_key", "hosts_count", "order_direction", "desc")
	assertVersionsResp(versResp, nil, time.Time{}, "", 0)

	// same with a team filter
	teamStr := fmt.Sprintf("%d", tm.ID)
	lsResp = listSoftwareResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software", nil, http.StatusOK, &lsResp, "order_key", "hosts_count", "order_direction", "desc", "team_id",
		teamStr,
	)
	assertResp(lsResp, nil, time.Time{}, teamStr)
	versResp = listSoftwareVersionsResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, &versResp, "order_key", "hosts_count", "order_direction", "desc",
		"team_id", teamStr,
	)
	assertVersionsResp(versResp, nil, time.Time{}, teamStr, 0)

	// calculate hosts counts
	hostsCountTs := time.Now().UTC()
	require.NoError(t, s.ds.SyncHostsSoftware(context.Background(), hostsCountTs))

	// now the list software endpoint returns the software, get the first page without vulns
	lsResp = listSoftwareResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software", nil, http.StatusOK, &lsResp, "per_page", "5", "page", "0", "order_key", "hosts_count", "order_direction", "desc")
	assertResp(lsResp, []fleet.Software{sws[19], sws[18], sws[17], sws[16], sws[15]}, hostsCountTs, "", 20, 19, 18, 17, 16)
	versResp = listSoftwareVersionsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, &versResp, "per_page", "5", "page", "0", "order_key", "hosts_count", "order_direction", "desc")
	assertVersionsResp(
		versResp, []fleet.Software{sws[19], sws[18], sws[17], sws[16], sws[15]}, hostsCountTs, "", len(sws), 20, 19, 18, 17, 16,
	)
	require.False(t, versResp.Meta.HasPreviousResults)
	require.True(t, versResp.Meta.HasNextResults)

	// second page (page=1)
	lsResp = listSoftwareResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software", nil, http.StatusOK, &lsResp, "per_page", "5", "page", "1", "order_key", "hosts_count", "order_direction", "desc")
	assertResp(lsResp, []fleet.Software{sws[14], sws[13], sws[12], sws[11], sws[10]}, hostsCountTs, "", 15, 14, 13, 12, 11)
	versResp = listSoftwareVersionsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, &versResp, "per_page", "5", "page", "1", "order_key", "hosts_count", "order_direction", "desc")
	assertVersionsResp(
		versResp, []fleet.Software{sws[14], sws[13], sws[12], sws[11], sws[10]}, hostsCountTs, "", len(sws), 15, 14, 13, 12, 11,
	)
	require.True(t, versResp.Meta.HasPreviousResults)
	require.True(t, versResp.Meta.HasNextResults)

	// third page (page=2)
	lsResp = listSoftwareResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software", nil, http.StatusOK, &lsResp, "per_page", "5", "page", "2", "order_key", "hosts_count", "order_direction", "desc")
	assertResp(lsResp, []fleet.Software{sws[9], sws[8], sws[7], sws[6], sws[5]}, hostsCountTs, "", 10, 9, 8, 7, 6)
	versResp = listSoftwareVersionsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, &versResp, "per_page", "5", "page", "2", "order_key", "hosts_count", "order_direction", "desc")
	assertVersionsResp(versResp, []fleet.Software{sws[9], sws[8], sws[7], sws[6], sws[5]}, hostsCountTs, "", len(sws), 10, 9, 8, 7, 6)
	require.True(t, versResp.Meta.HasPreviousResults)
	require.True(t, versResp.Meta.HasNextResults)

	// last page (page=3)
	lsResp = listSoftwareResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software", nil, http.StatusOK, &lsResp, "per_page", "5", "page", "3", "order_key", "hosts_count", "order_direction", "desc")
	assertResp(lsResp, []fleet.Software{sws[4], sws[3], sws[2], sws[1], sws[0]}, hostsCountTs, "", 5, 4, 3, 2, 1)
	versResp = listSoftwareVersionsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, &versResp, "per_page", "5", "page", "3", "order_key", "hosts_count", "order_direction", "desc")
	assertVersionsResp(versResp, []fleet.Software{sws[4], sws[3], sws[2], sws[1], sws[0]}, hostsCountTs, "", len(sws), 5, 4, 3, 2, 1)
	require.True(t, versResp.Meta.HasPreviousResults)
	require.False(t, versResp.Meta.HasNextResults)

	// past the end
	lsResp = listSoftwareResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software", nil, http.StatusOK, &lsResp, "per_page", "5", "page", "4", "order_key", "hosts_count", "order_direction", "desc")
	assertResp(lsResp, nil, time.Time{}, "")
	versResp = listSoftwareVersionsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, &versResp, "per_page", "5", "page", "4", "order_key", "hosts_count", "order_direction", "desc")
	assertVersionsResp(versResp, nil, time.Time{}, "", len(sws))

	// no explicit sort order, defaults to hosts_count DESC
	lsResp = listSoftwareResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software", nil, http.StatusOK, &lsResp, "per_page", "2", "page", "0")
	assertResp(lsResp, []fleet.Software{sws[19], sws[18]}, hostsCountTs, "", 20, 19)
	versResp = listSoftwareVersionsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, &versResp, "per_page", "2", "page", "0")
	assertVersionsResp(versResp, []fleet.Software{sws[19], sws[18]}, hostsCountTs, "", len(sws), 20, 19)

	// hosts_count ascending
	lsResp = listSoftwareResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software", nil, http.StatusOK, &lsResp, "per_page", "3", "page", "0", "order_key", "hosts_count", "order_direction", "asc")
	assertResp(lsResp, []fleet.Software{sws[0], sws[1], sws[2]}, hostsCountTs, "", 1, 2, 3)
	versResp = listSoftwareVersionsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, &versResp, "per_page", "3", "page", "0", "order_key", "hosts_count", "order_direction", "asc")
	assertVersionsResp(versResp, []fleet.Software{sws[0], sws[1], sws[2]}, hostsCountTs, "", len(sws), 1, 2, 3)

	// vulnerable software only
	lsResp = listSoftwareResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software", nil, http.StatusOK, &lsResp, "vulnerable", "true", "per_page", "5", "page", "0", "order_key", "hosts_count", "order_direction", "desc")
	assertResp(lsResp, []fleet.Software{sws[9], sws[8], sws[7], sws[6], sws[5]}, hostsCountTs, "", 10, 9, 8, 7, 6)
	versResp = listSoftwareVersionsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, &versResp, "vulnerable", "true", "per_page", "5", "page", "0", "order_key", "hosts_count", "order_direction", "desc")
	assertVersionsResp(
		versResp, []fleet.Software{sws[9], sws[8], sws[7], sws[6], sws[5]}, hostsCountTs, "", expectedVulnVersionsCount, 10, 9, 8, 7, 6,
	)

	// vulnerable software only, next page
	lsResp = listSoftwareResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software", nil, http.StatusOK, &lsResp, "vulnerable", "true", "per_page", "5", "page", "1", "order_key", "hosts_count", "order_direction", "desc")
	assertResp(lsResp, []fleet.Software{sws[4], sws[3], sws[2], sws[1], sws[0]}, hostsCountTs, "", 5, 4, 3, 2, 1)
	versResp = listSoftwareVersionsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, &versResp, "vulnerable", "true", "per_page", "5", "page", "1", "order_key", "hosts_count", "order_direction", "desc")
	assertVersionsResp(
		versResp, []fleet.Software{sws[4], sws[3], sws[2], sws[1], sws[0]}, hostsCountTs, "", expectedVulnVersionsCount, 5, 4, 3, 2, 1,
	)

	// vulnerable software only, past last page
	lsResp = listSoftwareResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software", nil, http.StatusOK, &lsResp, "vulnerable", "true", "per_page", "5", "page", "2", "order_key", "hosts_count", "order_direction", "desc")
	assertResp(lsResp, nil, time.Time{}, "")
	versResp = listSoftwareVersionsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, &versResp, "vulnerable", "true", "per_page", "5", "page", "2", "order_key", "hosts_count", "order_direction", "desc")
	assertVersionsResp(versResp, nil, time.Time{}, "", expectedVulnVersionsCount)

	// /software/versions  filtered by name, version, cve (`/software` is deprecated)
	versionsResp := listSoftwareVersionsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, &versionsResp, "query", sws[0].Name)
	assertVersionsResp(versionsResp, []fleet.Software{sws[0]}, hostsCountTs, "", 1, 1)
	// with whitespace
	s.DoJSON("GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, &versionsResp, "query", " "+sws[0].Name+"\n")
	assertVersionsResp(versionsResp, []fleet.Software{sws[0]}, hostsCountTs, "", 1, 1)

	s.DoJSON("GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, &versionsResp, "query", sws[0].Version)
	assertVersionsResp(versionsResp, []fleet.Software{sws[0]}, hostsCountTs, "", 1, 1)
	// with whitespace
	s.DoJSON("GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, &versionsResp, "query", "\n"+sws[0].Version+"  ")
	assertVersionsResp(versionsResp, []fleet.Software{sws[0]}, hostsCountTs, "", 1, 1)

	// All 10 CVEs added to the first 10 software have the same cvePrefix, so should return all
	// 10 vulnerable software versions
	s.DoJSON("GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, &versionsResp, "query", testCvePrefix)
	require.Len(t, versionsResp.Software, 10)
	require.Equal(t, 10, versionsResp.Count)
	// TODO(jacob) use `assertVersionsResp`
	// assertVersionsResp(versionsResp, sws[:10], hostsCountTs, "", 10, 1)
	// with whitespace
	s.DoJSON("GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, &versionsResp, "query", "  "+testCvePrefix+"\n")
	require.Len(t, versionsResp.Software, 10)
	require.Equal(t, 10, versionsResp.Count)
	// TODO(jacob) use `assertVersionsResp`
	// assertVersionsResp(versionsResp, sws[:10], hostsCountTs, "", 10, 1)

	// filter by the team, 2 by page
	lsResp = listSoftwareResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software", nil, http.StatusOK, &lsResp, "per_page", "2", "page", "0", "order_key", "hosts_count",
		"order_direction", "desc", "team_id", teamStr,
	)
	assertResp(lsResp, []fleet.Software{sws[19], sws[18]}, hostsCountTs, teamStr, 3, 2)
	versResp = listSoftwareVersionsResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, &versResp, "per_page", "2", "page", "0", "order_key",
		"hosts_count", "order_direction", "desc", "team_id", teamStr,
	)
	assertVersionsResp(versResp, []fleet.Software{sws[19], sws[18]}, hostsCountTs, teamStr, expectedTeamVersionsCount, 3, 2)

	// filter by the team, 2 by page, next page
	lsResp = listSoftwareResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software", nil, http.StatusOK, &lsResp, "per_page", "2", "page", "1", "order_key", "hosts_count",
		"order_direction", "desc", "team_id", teamStr,
	)
	assertResp(lsResp, []fleet.Software{sws[17]}, hostsCountTs, teamStr, 1)
	versResp = listSoftwareVersionsResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, &versResp, "per_page", "2", "page", "1", "order_key",
		"hosts_count", "order_direction", "desc", "team_id", teamStr,
	)
	assertVersionsResp(versResp, []fleet.Software{sws[17]}, hostsCountTs, teamStr, expectedTeamVersionsCount, 1)

	// filter by no team, 2 by page
	lsResp = listSoftwareResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software", nil, http.StatusOK, &lsResp, "per_page", "2", "page", "0", "order_key", "name",
		"order_direction", "desc", "team_id", "0",
	)
	fmt.Printf("lsResp: %+v\n", lsResp)
	assertResp(lsResp, []fleet.Software{sws[19], sws[18]}, hostsCountTs, "", 17, 17)

	// Invalid software team -- admin gets a 404, team users get a 403
	detailsResp := getSoftwareResponse{}
	s.DoJSON(
		"GET", fmt.Sprintf("/api/latest/fleet/software/versions/%d", versResp.Software[0].ID), nil, http.StatusNotFound, &detailsResp,
		"team_id", "999999",
	)

	// a request with without_vulnerability_details set to false does not return extra details
	respVersions := listSoftwareVersionsResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/versions",
		listSoftwareRequest{},
		http.StatusOK, &respVersions,
		"without_vulnerability_details", "false",
	)
	for _, s := range respVersions.Software {
		for _, cve := range s.Vulnerabilities {
			require.Nil(t, cve.CVSSScore)
			require.Nil(t, cve.EPSSProbability)
			require.Nil(t, cve.CISAKnownExploit)
			require.Nil(t, cve.CVEPublished)
			require.Nil(t, cve.Description)
			require.Nil(t, cve.ResolvedInVersion)
		}
	}
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

	var builtinNames []string
	for name := range fleet.ReservedLabelNames() {
		builtinNames = append(builtinNames, name)
	}
	lblMap, err := s.ds.LabelIDsByName(context.Background(), builtinNames)
	require.NoError(t, err)
	require.Len(t, lblMap, len(builtinNames))

	// no search criteria
	var searchResp searchTargetsResponse
	s.DoJSON("POST", "/api/latest/fleet/targets", searchTargetsRequest{}, http.StatusOK, &searchResp)
	require.Equal(t, uint(0), searchResp.TargetsCount)
	require.Len(t, searchResp.Targets.Hosts, len(hosts)) // the HostTargets.HostIDs are actually host IDs to *omit* from the search
	require.Len(t, searchResp.Targets.Labels, len(lblMap))
	require.Len(t, searchResp.Targets.Teams, 0)

	var lblIDs []uint
	for _, labelID := range lblMap {
		lblIDs = append(lblIDs, labelID)
	}

	searchResp = searchTargetsResponse{}
	s.DoJSON("POST", "/api/latest/fleet/targets", searchTargetsRequest{Selected: fleet.HostTargets{LabelIDs: lblIDs}}, http.StatusOK, &searchResp)
	require.Equal(t, uint(0), searchResp.TargetsCount)
	require.Len(t, searchResp.Targets.Hosts, len(hosts)) // no omitted host id
	require.Len(t, searchResp.Targets.Labels, 0)         // All built-in labels have been omitted (pre-selected)
	require.Len(t, searchResp.Targets.Teams, 0)

	searchResp = searchTargetsResponse{}
	s.DoJSON("POST", "/api/latest/fleet/targets", searchTargetsRequest{Selected: fleet.HostTargets{HostIDs: []uint{hosts[1].ID}}}, http.StatusOK, &searchResp)
	require.Equal(t, uint(1), searchResp.TargetsCount)
	require.Len(t, searchResp.Targets.Hosts, len(hosts)-1) // one omitted host id
	require.Len(t, searchResp.Targets.Labels, len(lblMap)) // labels have not been omitted
	require.Len(t, searchResp.Targets.Teams, 0)

	searchResp = searchTargetsResponse{}
	s.DoJSON("POST", "/api/latest/fleet/targets", searchTargetsRequest{MatchQuery: "foo.local1"}, http.StatusOK, &searchResp)
	require.Equal(t, uint(0), searchResp.TargetsCount)
	require.Len(t, searchResp.Targets.Hosts, 1)
	require.Len(t, searchResp.Targets.Labels, 1) // with a match query, only matching label names and "All Hosts" can be returned (here, only all hosts)
	require.Len(t, searchResp.Targets.Teams, 0)
	require.Contains(t, searchResp.Targets.Hosts[0].Hostname, "foo.local1")
}

func (s *integrationTestSuite) TestSearchHosts() {
	t := s.T()
	ctx := context.Background()

	hosts := s.createHosts(t)

	// set disk space information for hosts [0] and [1]
	require.NoError(t, s.ds.SetOrUpdateHostDisksSpace(ctx, hosts[0].ID, 1.0, 2.0, 500.0))
	require.NoError(t, s.ds.SetOrUpdateHostDisksSpace(ctx, hosts[1].ID, 3.0, 4.0, 1000.0))

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

	mysql.ExecAdhocSQL(t, s.ds, func(db sqlx.ExtContext) error {
		_, err := db.ExecContext(
			context.Background(),
			`INSERT INTO host_emails (host_id, email, source) VALUES (?, ?, ?)`,
			hosts[0].ID, "a@b.c", "src1")

		return err
	})

	s.DoJSON("POST", "/api/latest/fleet/hosts/search", searchHostsRequest{MatchQuery: "a@b.c"}, http.StatusOK, &searchResp)
	require.Len(t, searchResp.Hosts, 1)

	// search for non-existent email, shouldn't get anything back
	s.DoJSON("POST", "/api/latest/fleet/hosts/search", searchHostsRequest{MatchQuery: "not@found.com"}, http.StatusOK, &searchResp)
	require.Len(t, searchResp.Hosts, 0)
}

func (s *integrationTestSuite) TestCountTargets() {
	t := s.T()

	team, err := s.ds.NewTeam(context.Background(), &fleet.Team{Name: "TestTeam"})
	require.NoError(t, err)
	require.Equal(t, "TestTeam", team.Name)

	hosts := s.createHosts(t)

	lblMap, err := s.ds.LabelIDsByName(context.Background(), []string{"All Hosts"})
	require.NoError(t, err)
	require.Len(t, lblMap, 1)

	for i := range hosts {
		err = s.ds.RecordLabelQueryExecutions(context.Background(), hosts[i], map[uint]*bool{lblMap["All Hosts"]: ptr.Bool(true)}, time.Now(), false)
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

	var lblIDs []uint
	for _, labelID := range lblMap {
		lblIDs = append(lblIDs, labelID)
	}
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

	// 'No team' selected
	countResp = countTargetsResponse{}
	s.DoJSON(
		"POST", "/api/latest/fleet/targets/count", countTargetsRequest{Selected: fleet.HostTargets{TeamIDs: []uint{0}}},
		http.StatusOK, &countResp,
	)
	assert.Equal(t, uint(2), countResp.TargetsCount)
	assert.Equal(t, uint(0), countResp.TargetsOnline)
	assert.Equal(t, uint(2), countResp.TargetsOffline)

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
	}, http.StatusBadRequest, &blockResp)
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
	}, http.StatusBadRequest, &blockResp)
	checkCarveError(1, "block_id does not match expected block (2): 1")

	// sending final block with too many bytes
	blockResp = carveBlockResponse{}
	s.DoJSON("POST", "/api/osquery/carve/block", carveBlockRequest{
		BlockId:   2,
		SessionId: sid,
		RequestId: "r1",
		Data:      []byte("p3extra"),
	}, http.StatusBadRequest, &blockResp)
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
	}, http.StatusBadRequest, &blockResp)
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

func (s *integrationTestSuite) TestChangePassword() {
	t := s.T()

	endpoint := "/api/latest/fleet/change_password"
	// also the default password for the default logged in admin user
	startPwd := test.GoodPassword

	testCases := []struct {
		oldPw          string
		newPw          string
		expectedStatus int
	}{
		// valid changes – 12-48 characters, with at least 1 number (e.g. 0 - 9) and 1 symbol (e.g. &*#).
		{startPwd, "password123$", http.StatusOK},
		{"password123$", "Password$321", http.StatusOK},

		// invalid changes
		// empty old
		{"", "PassworD$321", http.StatusUnprocessableEntity},
		// empty new
		{"password123$", "", http.StatusUnprocessableEntity},
		// too short
		{"password123$", "Password$21", http.StatusUnprocessableEntity},
		// too long
		{"password123$", "Password$321Password$321Password$321Password$321Password$321", http.StatusUnprocessableEntity},
		// no numbers
		{"password123$", "Password$!@#", http.StatusUnprocessableEntity},
		// no symbols
		{"password123$", "Password4321", http.StatusUnprocessableEntity},
		// new pw is same as old
		{"password123$", "password123$", http.StatusUnprocessableEntity},
		// wrong old pw
		{"passgord123$", "Password$321", http.StatusUnprocessableEntity},
	}

	runTestCases := func(name string) {
		for _, tc := range testCases {
			t.Run(name, func(t *testing.T) {
				var changePwResp changePasswordResponse
				s.DoJSON("POST", endpoint, changePasswordRequest{OldPassword: tc.oldPw, NewPassword: tc.newPw}, tc.expectedStatus, &changePwResp)
			})
		}
	}

	runTestCases("test change passwords as admin")

	// create a new user
	testUserEmail := "changepwd@example.com"
	var createResp createUserResponse
	params := fleet.UserPayload{
		Name:                     ptr.String("Test Change Password"),
		Email:                    ptr.String(testUserEmail),
		Password:                 ptr.String(startPwd),
		GlobalRole:               ptr.String(fleet.RoleObserver),
		AdminForcedPasswordReset: ptr.Bool(false),
	}
	s.DoJSON("POST", "/api/latest/fleet/users/admin", params, http.StatusOK, &createResp)
	require.NotZero(t, createResp.User.ID)

	// schedule cleanup with admin user's token before changing it
	oldToken := s.token
	t.Cleanup(func() { s.token = oldToken })

	// login and run the change password tests as the user
	s.token = s.getTestToken(testUserEmail, startPwd)
	runTestCases("test change passwords as user")
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
	// // trigger the rate limit with a batch of requests in a short burst
	// for i := 0; i < 20; i++ {
	//	s.DoJSON("POST", "/api/latest/fleet/forgot_password", forgotPasswordRequest{Email: "invalid@asd.com"}, http.StatusAccepted, &forgotResp)
	// }

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
		NodeKey:         ptr.String(strings.ReplaceAll(t.Name(), "/", "_") + "1"),
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

	getHostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp, "exclude_software", "true")
	require.Equal(t, host.ID, getHostResp.Host.ID)
	require.Empty(t, getHostResp.Host.Software)
	require.Greater(t, getHostResp.Host.SoftwareUpdatedAt, getHostResp.Host.CreatedAt)

	getHostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/identifier/%s", *host.NodeKey), nil, http.StatusOK, &getHostResp)
	require.Equal(t, host.ID, getHostResp.Host.ID)
	require.Len(t, getHostResp.Host.Software, len(software))
	require.Greater(t, getHostResp.Host.SoftwareUpdatedAt, getHostResp.Host.CreatedAt)

	getHostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/identifier/%s", *host.NodeKey), nil, http.StatusOK, &getHostResp, "exclude_software", "true")
	require.Equal(t, host.ID, getHostResp.Host.ID)
	require.Empty(t, getHostResp.Host.Software)
	require.Greater(t, getHostResp.Host.SoftwareUpdatedAt, getHostResp.Host.CreatedAt)
}

func (s *integrationTestSuite) TestHostsReportDownload() {
	t := s.T()
	ctx := context.Background()

	// create 3 hosts (deb, rhel, linux)
	hosts := s.createHosts(t)
	err := s.ds.ApplyLabelSpecs(context.Background(), []*fleet.LabelSpec{
		{Name: t.Name(), LabelMembershipType: fleet.LabelMembershipTypeManual, Query: "select 1", Hosts: []string{hosts[2].Hostname}},
	})
	require.NoError(t, err)
	lids, err := s.ds.LabelIDsByName(context.Background(), []string{t.Name()})
	require.NoError(t, err)
	require.Len(t, lids, 1)
	customLabelID := lids[t.Name()]

	// create a policy and make host[1] fail that policy
	pol, err := s.ds.NewGlobalPolicy(ctx, nil, fleet.PolicyPayload{Name: t.Name(), Query: "SELECT 1"})
	require.NoError(t, err)
	err = s.ds.RecordPolicyQueryExecutions(ctx, hosts[1], map[uint]*bool{pol.ID: ptr.Bool(false)}, time.Now(), false)
	require.NoError(t, err)

	// create some device mappings for host[2]
	err = s.ds.ReplaceHostDeviceMapping(ctx, hosts[2].ID, []*fleet.HostDeviceMapping{
		{HostID: hosts[2].ID, Email: "a@b.c", Source: "google_chrome_profiles"},
		{HostID: hosts[2].ID, Email: "b@b.c", Source: "google_chrome_profiles"},
	}, "google_chrome_profiles")
	require.NoError(t, err)

	// set disk space information for hosts [0] and [1]
	require.NoError(t, s.ds.SetOrUpdateHostDisksSpace(ctx, hosts[0].ID, 1.0, 2.0, 500.0))
	require.NoError(t, s.ds.SetOrUpdateHostDisksSpace(ctx, hosts[1].ID, 3.0, 4.0, 1000.0))

	// create software for host [0]
	software := []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
	}
	_, err = s.ds.UpdateHostSoftware(ctx, hosts[0].ID, software)
	require.NoError(t, err)
	require.NoError(t, s.ds.LoadHostSoftware(ctx, hosts[0], false))

	err = s.ds.ReconcileSoftwareTitles(ctx)
	require.NoError(t, err)

	var fooV1ID, fooTitleID uint
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		err := sqlx.GetContext(context.Background(), q, &fooV1ID,
			`SELECT id FROM software WHERE name = ? AND source = ? AND version = ?`, "foo", "chrome_extensions", "0.0.1")
		if err != nil {
			return err
		}
		err = sqlx.GetContext(context.Background(), q, &fooTitleID,
			`SELECT id FROM software_titles WHERE name = ? AND source = ?`, "foo", "chrome_extensions")
		if err != nil {
			return err
		}
		return nil
	})

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
	assert.Len(t, rows[0], 54)         // total number of cols

	const (
		idCol        = 3
		issuesCol    = 45
		gigsDiskCol  = 42
		pctDiskCol   = 43
		gigsTotalCol = 44
	)

	// find the row for hosts[1], it should have issues=1 (1 failing policy) and the expected disk space
	for _, row := range rows[1:] {
		if row[idCol] == fmt.Sprint(hosts[1].ID) {
			assert.Equal(t, "1", row[issuesCol], row)
			assert.Equal(t, "3", row[gigsDiskCol], row)
			assert.Equal(t, "4", row[pctDiskCol], row)
			assert.Equal(t, "1000", row[gigsTotalCol], row)
		} else {
			assert.Equal(t, "0", row[issuesCol], row)
		}
	}

	// valid format, some columns
	res = s.DoRaw(
		"GET", "/api/latest/fleet/hosts/report", nil, http.StatusOK, "format", "csv",
		"columns", "hostname,gigs_disk_space_available,percent_disk_space_available,gigs_total_disk_space",
	)
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

	// search criteria including search query with leading/trailing whitespace are applied
	res = s.DoRaw("GET", "/api/latest/fleet/hosts/report", nil, http.StatusOK, "format", "csv", "query", "   local0 ", "columns", "hostname")
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

	// with a label id and a search query with leading/trailing whitespace
	res = s.DoRaw("GET", "/api/latest/fleet/hosts/report", nil, http.StatusOK, "format", "csv", "columns", "hostname", "label_id", fmt.Sprintf("%d", customLabelID), "query", "  local2 ")
	rows, err = csv.NewReader(res.Body).ReadAll()
	res.Body.Close()
	require.NoError(t, err)
	require.Len(t, rows, 2) // headers + member host
	// hosts[2] is both matched by the trimmed query and in the provided label
	require.Contains(t, rows[1], hosts[2].Hostname)

	// with a software version id
	res = s.DoRaw("GET", "/api/latest/fleet/hosts/report", nil, http.StatusOK, "format", "csv", "columns", "hostname", "software_version_id", fmt.Sprint(fooV1ID))
	rows, err = csv.NewReader(res.Body).ReadAll()
	res.Body.Close()
	require.NoError(t, err)
	require.Len(t, rows, 2) // headers + member host
	require.Contains(t, rows[1], hosts[0].Hostname)

	// with a software title id
	res = s.DoRaw("GET", "/api/latest/fleet/hosts/report", nil, http.StatusOK, "format", "csv", "columns", "hostname", "software_title_id", fmt.Sprint(fooTitleID))
	rows, err = csv.NewReader(res.Body).ReadAll()
	res.Body.Close()
	require.NoError(t, err)
	require.Len(t, rows, 2) // headers + member host
	require.Contains(t, rows[1], hosts[0].Hostname)

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

	// invalid combinations of software filters
	s.DoRaw("GET", "/api/latest/fleet/hosts/report", nil, http.StatusBadRequest, "software_title_id", "123", "software_id", "456")
	s.DoRaw("GET", "/api/latest/fleet/hosts/report", nil, http.StatusBadRequest, "software_title_id", "123", "software_version_id", "456")
	s.DoRaw("GET", "/api/latest/fleet/hosts/report", nil, http.StatusBadRequest, "software_id", "123", "software_version_id", "456")
	s.DoRaw("GET", "/api/latest/fleet/hosts/report", nil, http.StatusBadRequest, "software_id", "123", "software_version_id", "456", "software_title_id", "789")
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
		{HostID: host.ID, SerialNumber: "a", CycleCount: 1, Health: "Normal"},
		{HostID: host.ID, SerialNumber: "b", CycleCount: 1002, Health: "Service recommended"},
	}
	require.NoError(t, s.ds.ReplaceHostBatteries(context.Background(), host.ID, bats))

	var getHostResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.Equal(t, host.ID, getHostResp.Host.ID)
	// only cycle count and health are returned
	require.ElementsMatch(t, []*fleet.HostBattery{
		{CycleCount: 1, Health: "Normal"},
		{CycleCount: 1002, Health: "Service recommended"},
	}, *getHostResp.Host.Batteries)

	// same for get host by identifier
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/identifier/%s", *host.NodeKey), nil, http.StatusOK, &getHostResp)
	require.Equal(t, host.ID, getHostResp.Host.ID)
	// only cycle count and health are returned
	require.ElementsMatch(t, []*fleet.HostBattery{
		{CycleCount: 1, Health: "Normal"},
		{CycleCount: 1002, Health: "Service recommended"},
	}, *getHostResp.Host.Batteries)
}

func (s *integrationTestSuite) TestGetHostMaintenanceWindow() {
	t := s.T()
	ctx := context.Background()

	host, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String("1"),
		UUID:            "1",
		Hostname:        "foo.local",
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-58",
	})
	require.NoError(t, err)
	err = s.ds.ReplaceHostDeviceMapping(ctx, host.ID, []*fleet.HostDeviceMapping{
		{
			HostID: host.ID,
			Email:  "foo@example.com",
			Source: "google_chrome_profiles",
		},
	}, "google_chrome_profiles")
	require.NoError(t, err)

	startTime := time.Now().Add(time.Minute).In(time.UTC)
	endTime := startTime.Add(time.Minute * 30)
	testEvent := fleet.CalendarEvent{
		Email:     "foo@example.com",
		StartTime: startTime,
		EndTime:   endTime,
		Data:      []byte(`{}`),
		TimeZone:  nil,
		UUID:      uuid.New().String(),
	}

	dsEvent, err := s.ds.CreateOrUpdateCalendarEvent(ctx, testEvent.UUID, testEvent.Email, testEvent.StartTime, testEvent.EndTime,
		testEvent.Data, testEvent.TimeZone, host.ID, fleet.CalendarWebhookStatusNone)
	require.NoError(t, err)

	time.Sleep(1 * time.Second)

	// DB methods don't allow nil timezone, since we only allow it for the edge case that the db has
	// just undergone a migration and the calendar_cron has not run to populate the new `time_zone`
	// column yet. This means we need to manually set the timezone to nil.
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, "UPDATE calendar_events SET timezone = NULL WHERE id = ?", dsEvent.ID)
		return err
	})

	// GET host, check maintenance window
	var getHostResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.Equal(t, host.ID, getHostResp.Host.ID)
	// Round to account for sub-second precision differences between DB and Go
	require.Equal(t, testEvent.StartTime.Round(time.Second), getHostResp.Host.MaintenanceWindow.StartsAt)
	require.Nil(t, getHostResp.Host.MaintenanceWindow.TimeZone)

	timeZone := "America/Argentina/Buenos_Aires"
	// get a time.Location from the timezone string
	tZLoc, err := time.LoadLocation(timeZone)
	require.NoError(t, err)

	// use the time.Location to update the start time for the timezone
	zonedStartsAt := startTime.In(tZLoc).Round(time.Second)

	// update the timezone
	_, err = s.ds.CreateOrUpdateCalendarEvent(ctx, testEvent.UUID, testEvent.Email, testEvent.StartTime, testEvent.EndTime, testEvent.Data,
		&timeZone, host.ID, fleet.CalendarWebhookStatusNone)
	require.NoError(t, err)

	time.Sleep(1 * time.Second)

	// GET it again
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.Equal(t, host.ID, getHostResp.Host.ID)
	require.Equal(t, timeZone, *getHostResp.Host.MaintenanceWindow.TimeZone)

	// for equality comparison with original Go-derived start time, add a Location to the DB-derived start time, which only has an offset
	respStartsAt := getHostResp.Host.MaintenanceWindow.StartsAt
	respSAWithLoc, err := time.ParseInLocation("2006-01-02T15:04:05", respStartsAt.Format("2006-01-02T15:04:05"), tZLoc)
	require.NoError(t, err)

	require.Equal(t, zonedStartsAt, respSAWithLoc)
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
	require.NoError(t, s.ds.SetOrUpdateHostDisksSpace(context.Background(), hostWin.ID, 44.5, 55.6, 90.0))

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

	// should succeed as we no longer require MDM to access this endpoint, as Linux encryption doesn't require MDM
	var profiles getMDMProfilesSummaryResponse
	s.DoJSON("GET", "/api/latest/fleet/configuration_profiles/summary", getMDMProfilesSummaryRequest{}, http.StatusOK, &profiles)
	s.DoJSON("GET", "/api/latest/fleet/mdm/profiles/summary", getMDMProfilesSummaryRequest{}, http.StatusOK, &profiles)

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

	// Linux may omit the field when false
	getHostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", hostLin.ID), nil, http.StatusOK, &getHostResp)
	require.Equal(t, hostLin.ID, getHostResp.Host.ID)
	require.Nil(t, getHostResp.Host.DiskEncryptionEnabled)

	// the orbit endpoint to set the disk encryption key always fails in this
	// suite because MDM is not configured.
	orbitHost := createOrbitEnrolledHost(t, "windows", "diskenc", s.ds)
	res := s.Do("POST", "/api/fleet/orbit/disk_encryption_key", orbitPostDiskEncryptionKeyRequest{
		OrbitNodeKey:  *orbitHost.OrbitNodeKey,
		EncryptionKey: []byte("testkey"),
	}, http.StatusBadRequest)
	errMsg := extractServerErrorText(res.Body)
	require.Contains(t, errMsg, fleet.ErrMDMNotConfigured.Error())
}

func (s *integrationTestSuite) TestListVulnerabilities() {
	t := s.T()
	var resp listVulnerabilitiesResponse
	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities", nil, http.StatusOK, &resp)

	// Invalid Order Key
	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities", nil, http.StatusBadRequest, &resp, "order_key", "foo", "order_direction", "asc")

	// EE Order Key is an invalid order key
	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities", nil, http.StatusBadRequest, &resp, "order_key", "cvss_score", "order_direction", "asc")

	// Exploit is an EE only filter
	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities", nil, http.StatusPaymentRequired, &resp, "exploit", "true")

	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities", nil, http.StatusOK, &resp)
	require.Len(s.T(), resp.Vulnerabilities, 0)

	host, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String(strings.ReplaceAll(t.Name(), "/", "_") + "1"),
		OsqueryHostID:   ptr.String(strings.ReplaceAll(t.Name(), "/", "_") + "1"),
		UUID:            t.Name() + "1",
		Hostname:        t.Name() + "foo1.local",
		PrimaryIP:       "192.168.1.2",
		PrimaryMac:      "30-65-EC-6F-C4-59",
		Platform:        "windows",
	})
	require.NoError(t, err)

	err = s.ds.UpdateHostOperatingSystem(context.Background(), host.ID, fleet.OperatingSystem{
		Name:     "Windows 11 Enterprise 22H2",
		Version:  "10.0.19042.1234",
		Platform: "windows",
	})
	require.NoError(t, err)
	allos, err := s.ds.ListOperatingSystems(context.Background())
	require.NoError(t, err)
	var os fleet.OperatingSystem
	for _, o := range allos {
		if o.ID > os.ID {
			os = o
		}
	}

	err = s.ds.UpdateOSVersions(context.Background())
	require.NoError(t, err)

	_, err = s.ds.InsertOSVulnerability(context.Background(), fleet.OSVulnerability{
		OSID:              os.ID,
		CVE:               "CVE-2021-12345",
		ResolvedInVersion: *ptr.StringPtr("10.0.19043.2013"),
	}, fleet.MSRCSource)
	require.NoError(t, err)

	res, err := s.ds.UpdateHostSoftware(context.Background(), host.ID, []fleet.Software{
		{Name: "Google Chrome", Version: "0.0.1", Source: "programs"},
	})
	require.NoError(t, err)
	sw := res.Inserted[0]

	_, err = s.ds.UpsertSoftwareCPEs(context.Background(), []fleet.SoftwareCPE{
		{
			SoftwareID: sw.ID,
			CPE:        "cpe:2.3:a:google:chrome:1.0.0:*:*:*:*:*:*:*:*",
		},
	})
	require.NoError(t, err)

	_, err = s.ds.InsertSoftwareVulnerability(context.Background(), fleet.SoftwareVulnerability{
		SoftwareID: sw.ID,
		CVE:        "CVE-2021-1235",
	}, fleet.NVDSource)
	require.NoError(t, err)

	err = s.ds.SyncHostsSoftware(context.Background(), time.Now())
	require.NoError(t, err)

	host2, err := s.ds.NewHost(context.Background(), &fleet.Host{
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
		Platform:        "windows",
	})
	require.NoError(t, err)

	res2, err := s.ds.UpdateHostSoftware(context.Background(), host2.ID, []fleet.Software{
		{Name: "Firefox", Version: "0.0.1", Source: "programs"},
	})
	require.NoError(t, err)
	sw2 := res2.Inserted[0]

	// insert software vuln outside of host scope
	_, err = s.ds.InsertSoftwareVulnerability(context.Background(), fleet.SoftwareVulnerability{
		SoftwareID: sw2.ID,
		CVE:        "CVE-2021-1246",
	}, fleet.NVDSource)
	require.NoError(t, err)

	// insert CVEMeta
	knownCVE := "cve-2021-12999"
	mockTime := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	err = s.ds.InsertCVEMeta(context.Background(), []fleet.CVEMeta{
		{
			CVE:              "CVE-2021-12345",
			CVSSScore:        ptr.Float64(7.5),
			EPSSProbability:  ptr.Float64(0.5),
			CISAKnownExploit: ptr.Bool(true),
			Published:        ptr.Time(mockTime),
			Description:      "Test CVE 2021-12345",
		},
		{
			CVE:              "CVE-2021-1235",
			CVSSScore:        ptr.Float64(5.4),
			EPSSProbability:  ptr.Float64(0.6),
			CISAKnownExploit: ptr.Bool(false),
			Published:        ptr.Time(mockTime),
			Description:      "Test CVE 2021-1235",
		},
		{
			CVE:              "CVE-2021-1246",
			CVSSScore:        ptr.Float64(5.4),
			EPSSProbability:  ptr.Float64(0.6),
			CISAKnownExploit: ptr.Bool(false),
			Published:        ptr.Time(mockTime),
			Description:      "Test CVE 2021-1246",
		},
		{
			CVE:              knownCVE,
			CVSSScore:        ptr.Float64(6.4),
			EPSSProbability:  ptr.Float64(0.61),
			CISAKnownExploit: ptr.Bool(true),
			Published:        ptr.Time(mockTime),
			Description:      fmt.Sprintf("Test %s", knownCVE),
		},
	})
	require.NoError(t, err)

	err = s.ds.UpdateVulnerabilityHostCounts(context.Background())
	require.NoError(t, err)

	// test list
	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities", nil, http.StatusOK, &resp)
	require.Empty(t, resp.Err)
	require.Len(s.T(), resp.Vulnerabilities, 3)
	require.Equal(t, resp.Count, uint(3))
	require.False(t, resp.Meta.HasPreviousResults)
	require.False(t, resp.Meta.HasNextResults)

	expected := map[string]struct {
		fleet.CVEMeta
		HostCount   uint
		DetailsLink string
		Source      fleet.VulnerabilitySource
	}{
		"CVE-2021-12345": {
			HostCount:   1,
			DetailsLink: "https://nvd.nist.gov/vuln/detail/CVE-2021-12345",
		},
		"CVE-2021-1235": {
			HostCount:   1,
			DetailsLink: "https://nvd.nist.gov/vuln/detail/CVE-2021-1235",
		},
		"CVE-2021-1246": {
			HostCount:   1,
			DetailsLink: "https://nvd.nist.gov/vuln/detail/CVE-2021-1246",
		},
	}

	for _, vuln := range resp.Vulnerabilities {
		expectedVuln, ok := expected[vuln.CVE.CVE]
		require.True(t, ok, vuln.CVE.CVE)
		require.Equal(t, expectedVuln.HostCount, vuln.HostsCount)
		require.Equal(t, expectedVuln.DetailsLink, vuln.DetailsLink)
		require.Empty(t, vuln.CVSSScore)
	}

	// test list with matching query containing leading/trailing whitespace
	// TODO(jacob) - this may be another parsing bug
	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities", nil, http.StatusOK, &resp, "query", "  123	")
	require.Empty(t, resp.Err)
	require.Len(s.T(), resp.Vulnerabilities, 2)
	require.Equal(t, resp.Count, uint(2))
	require.False(t, resp.Meta.HasPreviousResults)
	require.False(t, resp.Meta.HasNextResults)

	expected = map[string]struct {
		fleet.CVEMeta
		HostCount   uint
		DetailsLink string
		Source      fleet.VulnerabilitySource
	}{
		"CVE-2021-12345": {
			HostCount:   1,
			DetailsLink: "https://nvd.nist.gov/vuln/detail/CVE-2021-12345",
		},
		"CVE-2021-1235": {
			HostCount:   1,
			DetailsLink: "https://nvd.nist.gov/vuln/detail/CVE-2021-1235",
		},
		// ...1246 should not match the query
	}

	for _, vuln := range resp.Vulnerabilities {
		expectedVuln, ok := expected[vuln.CVE.CVE]
		require.True(t, ok)
		require.Equal(t, expectedVuln.HostCount, vuln.HostsCount)
		require.Equal(t, expectedVuln.DetailsLink, vuln.DetailsLink)
		require.Empty(t, vuln.CVSSScore)
	}

	// test list with non-matching query
	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities", nil, http.StatusOK, &resp, "query", "CVB")
	require.Empty(t, resp.Err)
	require.Len(s.T(), resp.Vulnerabilities, 0)
	require.Equal(t, resp.Count, uint(0))
	require.False(t, resp.Meta.HasPreviousResults)
	require.False(t, resp.Meta.HasNextResults)

	// test with a known CVE that does not match on software/OS
	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities", nil, http.StatusOK, &resp, "query", knownCVE)
	require.Empty(t, resp.Err)
	assert.Len(s.T(), resp.Vulnerabilities, 0)
	assert.Equal(t, resp.Count, uint(0))
	assert.False(t, resp.Meta.HasPreviousResults)
	assert.False(t, resp.Meta.HasNextResults)

	// test with a substring of a known CVE -- results are returned
	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities", nil, http.StatusOK, &resp, "query", "CVE-2021-1234")
	require.Empty(t, resp.Err)
	assert.Len(s.T(), resp.Vulnerabilities, 1)
	assert.Equal(t, resp.Count, uint(1))
	assert.False(t, resp.Meta.HasPreviousResults)
	assert.False(t, resp.Meta.HasNextResults)
	_ = s.Do("GET", "/api/latest/fleet/vulnerabilities/CVE-2021-1234", nil, http.StatusNotFound)

	// Team 1 Filter
	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities", nil, http.StatusOK, &resp, "team_id", "1")
	require.Len(s.T(), resp.Vulnerabilities, 0)

	team, err := s.ds.NewTeam(context.Background(), &fleet.Team{Name: "team1"})
	require.NoError(t, err)
	err = s.ds.AddHostsToTeam(context.Background(), &team.ID, []uint{host.ID})
	require.NoError(t, err)

	err = s.ds.UpdateVulnerabilityHostCounts(context.Background())
	require.NoError(t, err)

	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities", nil, http.StatusOK, &resp, "team_id", fmt.Sprintf("%d", team.ID))
	require.Len(t, resp.Vulnerabilities, 2)
	require.Equal(t, uint(2), resp.Count)
	require.False(t, resp.Meta.HasPreviousResults)
	require.False(t, resp.Meta.HasNextResults)
	require.Empty(t, resp.Err)

	for _, vuln := range resp.Vulnerabilities {
		expectedVuln, ok := expected[vuln.CVE.CVE]
		require.True(t, ok)
		require.Equal(t, expectedVuln.HostCount, vuln.HostsCount)
		require.Equal(t, expectedVuln.DetailsLink, vuln.DetailsLink)
		require.Empty(t, vuln.CVSSScore)
	}

	// No filter (global)
	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities", nil, http.StatusOK, &resp)
	require.Len(t, resp.Vulnerabilities, 3)
	require.Equal(t, uint(3), resp.Count)
	require.Equal(t, uint(1), resp.Vulnerabilities[0].HostsCount)
	require.Equal(t, uint(1), resp.Vulnerabilities[1].HostsCount)
	require.Equal(t, uint(1), resp.Vulnerabilities[2].HostsCount)

	// Team 0 Filter
	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities", nil, http.StatusOK, &resp, "team_id", "0")
	require.Len(t, resp.Vulnerabilities, 1)
	require.Equal(t, uint(1), resp.Count)
	require.Equal(t, "CVE-2021-1246", resp.Vulnerabilities[0].CVE.CVE)
	require.Equal(t, uint(1), resp.Vulnerabilities[0].HostsCount)

	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities", nil, http.StatusOK, &resp, "team_id", "0")
	require.Len(t, resp.Vulnerabilities, 1)

	var gResp getVulnerabilityResponse
	// invalid cve
	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities/foobar", nil, http.StatusBadRequest, &gResp)

	// Valid CVE but not in team scope
	s.Do("GET", "/api/latest/fleet/vulnerabilities/CVE-2021-1246", nil, http.StatusNoContent, "team_id",
		fmt.Sprintf("%d", team.ID))

	// Valid CVE in "no team" scope
	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities/CVE-2021-1246", nil, http.StatusOK, &gResp, "team_id", "0")

	// Valid CVE not in "no team" scope
	s.Do("GET", "/api/latest/fleet/vulnerabilities/CVE-2021-12345", nil, http.StatusNoContent, "team_id", "0")

	// Invalid TeamID
	s.Do("GET", "/api/latest/fleet/vulnerabilities/CVE-2021-12345", nil, http.StatusForbidden, "team_id", "100")

	// Valid Global Request
	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities/CVE-2021-12345", nil, http.StatusOK, &gResp)
	require.Empty(t, gResp.Err)
	require.Equal(t, "CVE-2021-12345", gResp.Vulnerability.CVE.CVE)
	require.Equal(t, uint(1), gResp.Vulnerability.HostsCount)
	require.Equal(t, "https://nvd.nist.gov/vuln/detail/CVE-2021-12345", gResp.Vulnerability.DetailsLink)
	require.Empty(t, gResp.Vulnerability.Description)
	require.Empty(t, gResp.Vulnerability.CVSSScore)
	require.Empty(t, gResp.Vulnerability.CISAKnownExploit)
	require.Empty(t, gResp.Vulnerability.EPSSProbability)
	require.Empty(t, gResp.Vulnerability.CVEPublished)
	require.Len(t, gResp.OSVersions, 1)
	require.Equal(t, "Windows 11 Enterprise 22H2 10.0.19042.1234", gResp.OSVersions[0].Name)
	require.Equal(t, "Windows 11 Enterprise 22H2", gResp.OSVersions[0].NameOnly)
	require.Equal(t, "windows", gResp.OSVersions[0].Platform)
	require.Equal(t, "10.0.19042.1234", gResp.OSVersions[0].Version)
	require.Equal(t, 1, gResp.OSVersions[0].HostsCount)
	require.Equal(t, "10.0.19043.2013", *gResp.OSVersions[0].ResolvedInVersion)

	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities/CVE-2021-1235", nil, http.StatusOK, &gResp)
	require.Empty(t, gResp.Err)
	require.Equal(t, "CVE-2021-1235", gResp.Vulnerability.CVE.CVE)
	require.Equal(t, uint(1), gResp.Vulnerability.HostsCount)
	require.Equal(t, "https://nvd.nist.gov/vuln/detail/CVE-2021-1235", gResp.Vulnerability.DetailsLink)
	require.Empty(t, gResp.Vulnerability.Description)
	require.Empty(t, gResp.Vulnerability.CVSSScore)
	require.Empty(t, gResp.Vulnerability.CISAKnownExploit)
	require.Empty(t, gResp.Vulnerability.EPSSProbability)
	require.Empty(t, gResp.Vulnerability.CVEPublished)
	require.Len(t, gResp.Software, 1)
	require.Equal(t, "Google Chrome", gResp.Software[0].Name)
	require.Equal(t, "0.0.1", gResp.Software[0].Version)
	require.Equal(t, "programs", gResp.Software[0].Source)
	require.Equal(t, "cpe:2.3:a:google:chrome:1.0.0:*:*:*:*:*:*:*:*", gResp.Software[0].GenerateCPE)
	require.Equal(t, 1, gResp.Software[0].HostsCount)
}

func (s *integrationTestSuite) TestOSVersions() {
	t := s.T()

	testOSes := []fleet.OperatingSystem{
		{Name: "macOS", Version: "14.1.2", Arch: "64bit", KernelVersion: "13.37", Platform: "darwin"},                             // os_version_id=1
		{Name: "macOS", Version: "13.2.1", Arch: "64bit", KernelVersion: "18.12", Platform: "darwin"},                             // os_version_id=2
		{Name: "macOS", Version: "13.2.1", Arch: "64bit", KernelVersion: "18.12", Platform: "darwin"},                             // os_version_id=2
		{Name: "Windows 11 Pro 21H2", Version: "10.0.22000.1", Arch: "64bit", KernelVersion: "10.0.22000.1", Platform: "windows"}, // os_version_id=3
		{Name: "Windows 11 Pro 21H2", Version: "10.0.22000.1", Arch: "64bit", KernelVersion: "10.0.22000.1", Platform: "windows"}, // os_version_id=3
		{Name: "Windows 11 Pro 21H2", Version: "10.0.22000.1", Arch: "64bit", KernelVersion: "10.0.22000.1", Platform: "windows"}, // os_version_id=3
		{Name: "Windows 11 Pro 21H2", Version: "10.0.22000.2", Arch: "64bit", KernelVersion: "10.0.22000.2", Platform: "windows"}, // os_version_id=4
		{Name: "Windows 11 Pro 21H2", Version: "10.0.22000.2", Arch: "64bit", KernelVersion: "10.0.22000.2", Platform: "windows"}, // os_version_id=4
		{Name: "Windows 11 Pro 21H2", Version: "10.0.22000.2", Arch: "ARM64", KernelVersion: "10.0.22000.2", Platform: "windows"}, // os_version_id=4
		{Name: "Windows 11 Pro 21H2", Version: "10.0.22000.2", Arch: "ARM64", KernelVersion: "10.0.22000.2", Platform: "windows"}, // os_version_id=4
	}

	var platforms []string
	for _, os := range testOSes {
		platforms = append(platforms, os.Platform)
	}

	hosts := s.createHosts(t, platforms...)

	var resp listHostsResponse
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp)
	require.Len(t, resp.Hosts, len(hosts))

	// set operating system information on a host
	for i, os := range testOSes {
		require.NoError(t, s.ds.UpdateHostOperatingSystem(context.Background(), hosts[i].ID, os))
	}

	// get OS versions
	osv, err := s.ds.ListOperatingSystems(context.Background())
	require.NoError(t, err)

	osvMap := make(map[string]fleet.OperatingSystem)
	for _, os := range osv {
		key := fmt.Sprintf("%s %s %s", os.Name, os.Version, os.Arch)
		osvMap[key] = os
	}

	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "os_name", testOSes[1].Name, "os_version", testOSes[1].Version)
	require.Len(t, resp.Hosts, 2)

	expected := hosts[1].Hostname
	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "os_version_id", fmt.Sprintf("%d", osvMap["macOS 13.2.1 64bit"].OSVersionID))
	require.Len(t, resp.Hosts, 2)
	require.Equal(t, expected, resp.Hosts[0].Hostname)

	countResp := countHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp, "os_version_id", fmt.Sprintf("%d", osvMap["macOS 13.2.1 64bit"].OSVersionID))
	require.Equal(t, 2, countResp.Count)

	// generate aggregated stats
	require.NoError(t, s.ds.UpdateOSVersions(context.Background()))

	// insert Vuln for Win x64
	_, err = s.ds.InsertOSVulnerability(context.Background(), fleet.OSVulnerability{
		OSID: osvMap["Windows 11 Pro 21H2 10.0.22000.2 64bit"].ID,
		CVE:  "CVE-2021-1234",
	}, fleet.MSRCSource)
	require.NoError(t, err)

	// insert duplicate Vuln for Win ARM64
	_, err = s.ds.InsertOSVulnerability(context.Background(), fleet.OSVulnerability{
		OSID: osvMap["Windows 11 Pro 21H2 10.0.22000.2 ARM64"].ID,
		CVE:  "CVE-2021-1234",
	}, fleet.MSRCSource)
	require.NoError(t, err)

	// insert different Vuln for Win ARM64
	_, err = s.ds.InsertOSVulnerability(context.Background(), fleet.OSVulnerability{
		OSID: osvMap["Windows 11 Pro 21H2 10.0.22000.2 ARM64"].ID,
		CVE:  "CVE-2021-5678",
	}, fleet.MSRCSource)
	require.NoError(t, err)

	assertOSVersion := func(t *testing.T, expected fleet.OSVersion, actual fleet.OSVersion) {
		require.Equal(t, expected.HostsCount, actual.HostsCount)
		require.Equal(t, expected.Name, actual.Name)
		require.Equal(t, expected.NameOnly, actual.NameOnly)
		require.Equal(t, expected.Version, actual.Version)
		require.Equal(t, expected.Platform, actual.Platform)
		require.Equal(t, expected.OSVersionID, actual.OSVersionID)
		require.Len(t, actual.Vulnerabilities, len(expected.Vulnerabilities))
		for i, vuln := range expected.Vulnerabilities {
			require.Equal(t, vuln.CVE, actual.Vulnerabilities[i].CVE)
			require.Equal(t, vuln.DetailsLink, actual.Vulnerabilities[i].DetailsLink)
			require.Greater(t, actual.Vulnerabilities[i].CreatedAt, time.Now().Add(-time.Hour)) // assert non-zero value
		}
	}

	var osVersionsResp osVersionsResponse
	s.DoJSON("GET", "/api/latest/fleet/os_versions", nil, http.StatusOK, &osVersionsResp)
	require.Len(t, osVersionsResp.OSVersions, 4) // different archs are grouped together

	// Default sort is by hosts count, descending
	expectedVersion := fleet.OSVersion{
		HostsCount:  4,
		Name:        "Windows 11 Pro 21H2 10.0.22000.2",
		NameOnly:    "Windows 11 Pro 21H2",
		Version:     "10.0.22000.2",
		Platform:    "windows",
		OSVersionID: osvMap["Windows 11 Pro 21H2 10.0.22000.2 ARM64"].OSVersionID,
		Vulnerabilities: fleet.Vulnerabilities{
			{
				CVE:         "CVE-2021-1234",
				DetailsLink: "https://nvd.nist.gov/vuln/detail/CVE-2021-1234",
			},
			{
				CVE:         "CVE-2021-5678", // vulns are aggregated by OS name and version
				DetailsLink: "https://nvd.nist.gov/vuln/detail/CVE-2021-5678",
			},
		},
	}

	// Default sort is by hosts count, descending
	assertOSVersion(t, expectedVersion, osVersionsResp.OSVersions[0])

	// get OS version by id
	var osVersionResp getOSVersionResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/os_versions/%d", osvMap["Windows 11 Pro 21H2 10.0.22000.2 ARM64"].OSVersionID), nil, http.StatusOK, &osVersionResp)
	assertOSVersion(t, expectedVersion, *osVersionResp.OSVersion)

	// invalid id
	s.DoJSON("GET", "/api/latest/fleet/os_versions/999", nil, http.StatusOK, &osVersionResp)
	assert.Zero(t, osVersionResp.OSVersion.HostsCount)

	// name and version filters
	s.DoJSON("GET", "/api/latest/fleet/os_versions", nil, http.StatusOK, &osVersionsResp, "os_name", "Windows 11 Pro 21H2", "os_version", "10.0.22000.2")
	require.Len(t, osVersionsResp.OSVersions, 1)
	require.Equal(t, "Windows 11 Pro 21H2 10.0.22000.2", osVersionsResp.OSVersions[0].Name)
	require.Len(t, osVersionsResp.OSVersions[0].Vulnerabilities, 2)

	// name without version
	s.DoJSON("GET", "/api/latest/fleet/os_versions", nil, http.StatusBadRequest, &osVersionsResp, "os_name", "Windows 11 Pro 21H2")

	// version without name
	s.DoJSON("GET", "/api/latest/fleet/os_versions", nil, http.StatusBadRequest, &osVersionsResp, "os_version", "10.0.22000.1")

	// invalid order key
	s.DoJSON("GET", "/api/latest/fleet/os_versions", nil, http.StatusBadRequest, &osVersionsResp, "order_key", "nosuchkey")

	// ascending order by hosts count
	s.DoJSON("GET", "/api/latest/fleet/os_versions", nil, http.StatusOK, &osVersionsResp, "order_key", "hosts_count", "order_direction", "asc")
	require.Equal(t, 1, osVersionsResp.OSVersions[0].HostsCount)
	require.Equal(t, "macOS 14.1.2", osVersionsResp.OSVersions[0].Name)

	// test pagination
	s.DoJSON("GET", "/api/latest/fleet/os_versions", nil, http.StatusOK, &osVersionsResp, "page", "0", "per_page", "2")
	require.Len(t, osVersionsResp.OSVersions, 2)
	require.Equal(t, "Windows 11 Pro 21H2 10.0.22000.2", osVersionsResp.OSVersions[0].Name)
	require.Equal(t, "Windows 11 Pro 21H2 10.0.22000.1", osVersionsResp.OSVersions[1].Name)
	require.Equal(t, 4, osVersionsResp.Count)
	require.True(t, osVersionsResp.Meta.HasNextResults)
	require.False(t, osVersionsResp.Meta.HasPreviousResults)

	s.DoJSON("GET", "/api/latest/fleet/os_versions", nil, http.StatusOK, &osVersionsResp, "page", "1", "per_page", "2")
	require.Len(t, osVersionsResp.OSVersions, 2)
	require.Equal(t, "macOS 13.2.1", osVersionsResp.OSVersions[0].Name)
	require.Equal(t, "macOS 14.1.2", osVersionsResp.OSVersions[1].Name)
	require.Equal(t, 4, osVersionsResp.Count)
	require.False(t, osVersionsResp.Meta.HasNextResults)
	require.True(t, osVersionsResp.Meta.HasPreviousResults)

	// same results with team_id=0
	s.DoJSON("GET", "/api/latest/fleet/os_versions", nil, http.StatusOK, &osVersionsResp, "page", "1", "per_page", "2", "team_id", "0")
	require.Len(t, osVersionsResp.OSVersions, 2)
	require.Equal(t, "macOS 13.2.1", osVersionsResp.OSVersions[0].Name)
	require.Equal(t, "macOS 14.1.2", osVersionsResp.OSVersions[1].Name)
	require.Equal(t, 4, osVersionsResp.Count)
	require.False(t, osVersionsResp.Meta.HasNextResults)
	require.True(t, osVersionsResp.Meta.HasPreviousResults)
}

func (s *integrationTestSuite) TestPingEndpoints() {
	t := s.T()

	s.DoRaw("HEAD", "/api/fleet/orbit/ping", nil, http.StatusOK)
	// unauthenticated works too
	s.DoRawNoAuth("HEAD", "/api/fleet/orbit/ping", nil, http.StatusOK)

	s.DoRaw("HEAD", "/api/fleet/device/ping", nil, http.StatusOK)
	// unauthenticated works too
	s.DoRawNoAuth("HEAD", "/api/fleet/device/ping", nil, http.StatusOK)

	// device authenticated ping
	createHostAndDeviceToken(t, s.ds, "ping-token")
	s.DoRaw("HEAD", fmt.Sprintf("/api/v1/fleet/device/%s/ping", "ping-token"), nil, http.StatusOK)
	s.DoRawNoAuth("HEAD", fmt.Sprintf("/api/v1/fleet/device/%s/ping", "ping-token"), nil, http.StatusOK)
	s.DoRaw("HEAD", fmt.Sprintf("/api/v1/fleet/device/%s/ping", "bozo-token"), nil, http.StatusUnauthorized)
	s.DoRawNoAuth("HEAD", fmt.Sprintf("/api/v1/fleet/device/%s/ping", "bozo-token"), nil, http.StatusUnauthorized)
}

func (s *integrationTestSuite) TestMDMNotConfiguredEndpoints() {
	t := s.T()

	// create a host with device token to test device authenticated routes
	tkn := "D3V1C370K3N"
	createHostAndDeviceToken(t, s.ds, tkn)

	for _, route := range mdmConfigurationRequiredEndpoints() {
		which := fmt.Sprintf("%s %s", route.method, route.path)
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
	err = s.ds.SetOrUpdateMDMData(context.Background(), hSimpleMDM.ID, false, true, "https://simplemdm.com", false, fleet.WellKnownMDMSimpleMDM, "")
	require.NoError(t, err)
	resp = orbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *hSimpleMDM.OrbitNodeKey)), http.StatusOK, &resp)
	require.False(t, resp.Notifications.RenewEnrollmentProfile)

	// not yet assigned in ABM
	hFleetMDM := createOrbitEnrolledHost(t, "darwin", "fleetmdm", s.ds)
	err = s.ds.SetOrUpdateMDMData(context.Background(), hFleetMDM.ID, false, false, "https://fleetdm.com", true, fleet.WellKnownMDMFleet, "")
	require.NoError(t, err)

	resp = orbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *hFleetMDM.OrbitNodeKey)), http.StatusOK, &resp)
	require.False(t, resp.Notifications.RenewEnrollmentProfile)

	// simulate ABM assignment
	encTok := uuid.NewString()
	abmToken, err := s.ds.InsertABMToken(ctx, &fleet.ABMToken{OrganizationName: "unused", EncryptedToken: []byte(encTok)})
	require.NoError(t, err)
	require.NotEmpty(t, abmToken.ID)
	err = s.ds.UpsertMDMAppleHostDEPAssignments(ctx, []fleet.Host{*hFleetMDM}, abmToken.ID)
	require.NoError(t, err)
	err = s.ds.SetOrUpdateMDMData(context.Background(), hSimpleMDM.ID, false, true, "https://simplemdm.com", false, fleet.WellKnownMDMSimpleMDM, "")
	require.NoError(t, err)
	resp = orbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *hFleetMDM.OrbitNodeKey)), http.StatusOK, &resp)
	require.True(t, resp.Notifications.RenewEnrollmentProfile)

	// if the fleet mdm host is fully enrolled (not pending anymore), then the notification is false
	err = s.ds.SetOrUpdateMDMData(context.Background(), hFleetMDM.ID, false, true, "https://fleetdm.com", true, fleet.WellKnownMDMFleet, "")
	require.NoError(t, err)
	resp = orbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *hFleetMDM.OrbitNodeKey)), http.StatusOK, &resp)
	require.False(t, resp.Notifications.RenewEnrollmentProfile)

	// the scripts orbit endpoints are accessible without license
	s.Do("POST", "/api/fleet/orbit/scripts/request", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *hFleetMDM.OrbitNodeKey)), http.StatusNotFound)
	s.Do("POST", "/api/fleet/orbit/scripts/result", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *hFleetMDM.OrbitNodeKey)), http.StatusBadRequest)
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
		Logging:        fleet.LoggingSnapshot,
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

func setOrbitEnrollment(t *testing.T, h *fleet.Host, ds fleet.Datastore) string {
	orbitKey := uuid.New().String()
	_, err := ds.EnrollOrbit(context.Background(), false, fleet.OrbitHostInfo{
		HardwareUUID:   *h.OsqueryHostID,
		HardwareSerial: h.HardwareSerial,
	}, orbitKey, h.TeamID)
	require.NoError(t, err)
	err = ds.SetOrUpdateHostOrbitInfo(
		context.Background(), h.ID, "1.22.0", sql.NullString{String: "42", Valid: true}, sql.NullBool{Bool: true, Valid: true},
	)
	require.NoError(t, err)
	return orbitKey
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

	orbitKey := setOrbitEnrollment(t, h, ds)
	h.OrbitNodeKey = &orbitKey
	return h
}

// creates a session and returns it, its key is to be passed as authorization header.
func createSession(t *testing.T, uid uint, ds fleet.Datastore) *fleet.Session {
	ssn, err := ds.NewSession(context.Background(), uid, 64)
	require.NoError(t, err)

	return ssn
}

func (s *integrationTestSuite) cleanupQuery(queryID uint) {
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

func (s *integrationTestSuite) TestDirectIngestScheduledQueryStats() {
	t := s.T()

	team1, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		Name: "Foobar",
	})
	require.NoError(t, err)
	team2, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		Name: "Zoo",
	})
	require.NoError(t, err)
	globalHost, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   ptr.String(uuid.New().String()),
		NodeKey:         ptr.String(uuid.New().String()),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%sfoo.global", t.Name()),
		Platform:        "darwin",
	})
	require.NoError(t, err)
	team1Host, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   ptr.String(uuid.New().String()),
		NodeKey:         ptr.String(uuid.New().String()),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%sfoo.team", t.Name()),
		Platform:        "darwin",
		TeamID:          &team1.ID,
	})
	require.NoError(t, err)
	scheduledGlobalQuery, err := s.ds.NewQuery(context.Background(), &fleet.Query{
		Name:               "scheduled-global-query",
		TeamID:             nil,
		Interval:           10,
		Platform:           "darwin",
		AutomationsEnabled: true,
		Logging:            fleet.LoggingSnapshot,
		Description:        "foobar",
		Query:              "SELECT * from time;",
		Saved:              true,
	})
	require.NoError(t, err)
	nonScheduledGlobalQuery, err := s.ds.NewQuery(context.Background(), &fleet.Query{
		Name:               "non-scheduled-global-query",
		TeamID:             nil,
		Interval:           0,
		Platform:           "darwin",
		AutomationsEnabled: false,
		Logging:            fleet.LoggingSnapshot,
		Description:        "foobar",
		Query:              "SELECT * from osquery_info;",
		Saved:              true,
	})
	require.NoError(t, err)
	scheduledTeam1Query1, err := s.ds.NewQuery(context.Background(), &fleet.Query{
		Name:               "scheduled-team1-query1",
		TeamID:             &team1.ID,
		Interval:           20,
		Platform:           "",
		AutomationsEnabled: true,
		Logging:            fleet.LoggingSnapshot,
		Description:        "foobar",
		Query:              "SELECT * from other;",
		Saved:              true,
	})
	require.NoError(t, err)
	scheduledTeam1Query2, err := s.ds.NewQuery(context.Background(), &fleet.Query{
		Name:               "scheduled-team1-query2",
		TeamID:             &team1.ID,
		Interval:           90,
		Platform:           "",
		AutomationsEnabled: true,
		Logging:            fleet.LoggingSnapshot,
		Description:        "foobar",
		Query:              "SELECT * from other;",
		Saved:              true,
	})
	require.NoError(t, err)
	// Create a non-scheduled query to test that we filter it out when providing
	// the queries in the osquery/config endpoint.
	_, err = s.ds.NewQuery(context.Background(), &fleet.Query{
		Name:               "non-scheduled-team1-query",
		TeamID:             &team1.ID,
		Interval:           0,
		Platform:           "",
		AutomationsEnabled: false,
		Logging:            "snapshot",
		Description:        "foobar",
		Query:              "SELECT * from foobar;",
		Saved:              true,
	})
	require.NoError(t, err)
	// Create a scheduled query but on another team to test that we filter it
	// out when providing the queries in the osquery/config endpoint.
	_, err = s.ds.NewQuery(context.Background(), &fleet.Query{
		Name:               "scheduled-team2-query",
		TeamID:             &team2.ID,
		Interval:           40,
		Platform:           "",
		AutomationsEnabled: true,
		Logging:            fleet.LoggingSnapshot,
		Description:        "foobar",
		Query:              "SELECT * from other;",
		Saved:              true,
	})
	require.NoError(t, err)
	// Create a legacy 2017 user pack with one query.
	userPack1TargetTeam1, err := s.ds.NewPack(context.Background(), &fleet.Pack{
		Name:    "2017 Pack",
		Type:    nil,
		Teams:   []fleet.Target{{TargetID: team1.ID, Type: fleet.TargetTeam}},
		TeamIDs: []uint{team1.ID},
	})
	require.NoError(t, err)
	scheduledQueryOnPack1, err := s.ds.NewScheduledQuery(context.Background(), &fleet.ScheduledQuery{
		Name:     "scheduled-query-pack1",
		PackID:   userPack1TargetTeam1.ID,
		QueryID:  nonScheduledGlobalQuery.ID,
		Interval: 60,
		Snapshot: ptr.Bool(true),
		Removed:  ptr.Bool(true),
	})
	require.NoError(t, err)

	// Simulate the osquery instance of the global host calling the osquery/config endpoint
	// and test the returned scheduled queries.
	req := getClientConfigRequest{NodeKey: *globalHost.NodeKey}
	var resp getClientConfigResponse
	s.DoJSON("POST", "/api/osquery/config", req, http.StatusOK, &resp)
	packs := resp.Config["packs"].(map[string]interface{})
	require.Len(t, packs, 1)
	globalQueries := packs["Global"].(map[string]interface{})["queries"].(map[string]interface{})
	require.Len(t, globalQueries, 1)
	require.Contains(t, globalQueries, scheduledGlobalQuery.Name)

	// Simulate the osquery instance of the team host calling the osquery/config endpoint
	// and test the returned scheduled queries.
	req = getClientConfigRequest{NodeKey: *team1Host.NodeKey}
	resp = getClientConfigResponse{}
	s.DoJSON("POST", "/api/osquery/config", req, http.StatusOK, &resp)
	packs = resp.Config["packs"].(map[string]interface{})
	require.Len(t, packs, 3)
	globalQueries = packs["Global"].(map[string]interface{})["queries"].(map[string]interface{})
	require.Len(t, globalQueries, 1)
	require.Contains(t, globalQueries, scheduledGlobalQuery.Name)
	team1Queries := packs[fmt.Sprintf("team-%d", team1.ID)].(map[string]interface{})["queries"].(map[string]interface{})
	require.Len(t, team1Queries, 2)
	require.Contains(t, team1Queries, scheduledTeam1Query1.Name)
	require.Contains(t, team1Queries, scheduledTeam1Query2.Name)
	userPack1Queries := packs[userPack1TargetTeam1.Name].(map[string]interface{})["queries"].(map[string]interface{})
	require.Len(t, userPack1Queries, 1)
	require.Contains(t, userPack1Queries, scheduledQueryOnPack1.Name)

	// Now let's simulate a osquery instance running in the team host returning the
	// stats in the distributed/write (osquery_schedule table)
	rows := []map[string]string{
		{
			"name":              "pack/Global/scheduled-global-query",
			"query":             "SELECT * FROM time;",
			"interval":          "10",
			"executions":        "2",
			"last_executed":     "1693476753",
			"denylisted":        "0",
			"output_size":       "576",
			"wall_time":         "1",
			"wall_time_ms":      "2",
			"last_wall_time_ms": "3",
			"user_time":         "4",
			"last_user_time":    "5",
			"system_time":       "6",
			"last_system_time":  "7",
			"average_memory":    "8",
			"last_memory":       "9",
			"delimiter":         "/",
		},
		{
			"name":              "pack/2017 Pack/scheduled-query-pack1",
			"query":             "SELECT * FROM osquery_info;",
			"interval":          "60",
			"executions":        "20",
			"last_executed":     "1693476842",
			"denylisted":        "0",
			"output_size":       "9620",
			"wall_time":         "9",
			"wall_time_ms":      "8",
			"last_wall_time_ms": "7",
			"user_time":         "6",
			"last_user_time":    "5",
			"system_time":       "4",
			"last_system_time":  "3",
			"average_memory":    "2",
			"last_memory":       "1",
			"delimiter":         "/",
		},
		{
			"name":              fmt.Sprintf("pack/team-%d/scheduled-team1-query1", team1.ID),
			"query":             "SELECT * FROM other;",
			"interval":          "20",
			"executions":        "1",
			"last_executed":     "1693476561",
			"denylisted":        "0",
			"output_size":       "10",
			"wall_time":         "11",
			"wall_time_ms":      "12",
			"last_wall_time_ms": "13",
			"user_time":         "14",
			"last_user_time":    "15",
			"system_time":       "16",
			"last_system_time":  "17",
			"average_memory":    "18",
			"last_memory":       "19",
			"delimiter":         "/",
		},
		{
			"name":              fmt.Sprintf("pack/team-%d/scheduled-team1-query2", team1.ID),
			"query":             "SELECT * FROM other;",
			"interval":          "90",
			"executions":        "5",
			"last_executed":     "1693476666",
			"denylisted":        "0",
			"output_size":       "20",
			"wall_time":         "21",
			"wall_time_ms":      "22",
			"last_wall_time_ms": "23",
			"user_time":         "24",
			"last_user_time":    "25",
			"system_time":       "26",
			"last_system_time":  "27",
			"average_memory":    "28",
			"last_memory":       "29",
			"delimiter":         "/",
		},
	}

	appConfig, err := s.ds.AppConfig(context.Background())
	require.NoError(t, err)
	detailQueries := osquery_utils.GetDetailQueries(context.Background(), config.FleetConfig{
		App: config.AppConfig{
			EnableScheduledQueryStats: true,
		},
	}, appConfig, &appConfig.Features)
	task := async.NewTask(s.ds, nil, clock.C, config.OsqueryConfig{})
	err = detailQueries["scheduled_query_stats"].DirectTaskIngestFunc(
		context.Background(),
		log.NewNopLogger(),
		team1Host,
		task,
		rows,
	)
	require.NoError(t, err)

	// Check that the received stats were stored in the DB as expected.
	var scheduledQueriesStats []fleet.ScheduledQueryStats
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.SelectContext(context.Background(), q, &scheduledQueriesStats,
			`SELECT
				scheduled_query_id, q.name AS scheduled_query_name, average_memory, denylisted,
				executions, q.schedule_interval, last_executed,
				output_size, system_time, user_time, wall_time
			FROM scheduled_query_stats sqs
			JOIN queries q ON sqs.scheduled_query_id = q.id
			WHERE host_id = ?;`,
			team1Host.ID,
		)
	})
	require.Len(t, scheduledQueriesStats, 4)
	rowsMap := make(map[string]map[string]string)
	for _, row := range rows {
		parts := strings.Split(row["name"], "/")
		queryName := parts[len(parts)-1]
		// we need to map this because 2017 packs send the name of the schedule and not
		// the name of the query.
		if queryName == "scheduled-query-pack1" {
			queryName = "non-scheduled-global-query"
		}
		rowsMap[queryName] = row
	}
	for _, sqs := range scheduledQueriesStats {
		row := rowsMap[sqs.ScheduledQueryName]
		require.Equal(t, fmt.Sprint(sqs.AverageMemory), row["average_memory"])
		require.Equal(t, fmt.Sprint(sqs.Executions), row["executions"])
		interval := row["interval"]
		if sqs.ScheduledQueryName == "non-scheduled-global-query" {
			interval = "0" // this query has metrics because it runs on a pack.
		}
		require.Equal(t, strconv.FormatInt(int64(sqs.Interval), 10), interval)
		lastExecuted, err := strconv.ParseInt(row["last_executed"], 10, 64)
		require.NoError(t, err)
		require.WithinDuration(t, sqs.LastExecuted, time.Unix(lastExecuted, 0), 1*time.Second)
		require.Equal(t, fmt.Sprint(sqs.OutputSize), row["output_size"])
		require.Equal(t, fmt.Sprint(sqs.SystemTime), row["system_time"])
		require.Equal(t, fmt.Sprint(sqs.UserTime), row["user_time"])
		assert.Equal(t, fmt.Sprint(sqs.WallTime), row["wall_time_ms"])
	}

	// Now let's simulate a osquery instance running in the global host returning the
	// stats in the distributed/write (osquery_schedule table)
	rows = []map[string]string{
		{
			"name":              "pack/Global/scheduled-global-query",
			"query":             "SELECT * FROM time;",
			"interval":          "10",
			"executions":        "2",
			"last_executed":     "1693476753",
			"denylisted":        "0",
			"output_size":       "576",
			"wall_time":         "1",
			"wall_time_ms":      "2",
			"last_wall_time_ms": "3",
			"user_time":         "4",
			"last_user_time":    "5",
			"system_time":       "6",
			"last_system_time":  "7",
			"average_memory":    "8",
			"last_memory":       "9",
			"delimiter":         "/",
		},
	}

	err = detailQueries["scheduled_query_stats"].DirectTaskIngestFunc(
		context.Background(),
		log.NewNopLogger(),
		globalHost,
		task,
		rows,
	)
	require.NoError(t, err)

	// Check that the received stats were stored in the DB as expected.
	scheduledQueriesStats = []fleet.ScheduledQueryStats{}
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.SelectContext(context.Background(), q, &scheduledQueriesStats,
			`SELECT
				scheduled_query_id, q.name AS scheduled_query_name, average_memory, denylisted,
				executions, q.schedule_interval, last_executed,
				output_size, system_time, user_time, wall_time
			FROM scheduled_query_stats sqs
			JOIN queries q ON sqs.scheduled_query_id = q.id
			WHERE host_id = ?;`,
			globalHost.ID,
		)
	})
	require.Len(t, scheduledQueriesStats, 1)
	row := rows[0]
	parts := strings.Split(row["name"], "/")
	queryName := parts[len(parts)-1]
	sqs := scheduledQueriesStats[0]
	require.Equal(t, scheduledQueriesStats[0].ScheduledQueryName, queryName)
	require.Equal(t, fmt.Sprint(sqs.AverageMemory), row["average_memory"])
	require.Equal(t, fmt.Sprint(sqs.Executions), row["executions"])
	require.Equal(t, fmt.Sprint(sqs.Interval), row["interval"])
	lastExecuted, err := strconv.ParseInt(row["last_executed"], 10, 64)
	require.NoError(t, err)
	require.WithinDuration(t, sqs.LastExecuted, time.Unix(lastExecuted, 0), 1*time.Second)
	require.Equal(t, fmt.Sprint(sqs.OutputSize), row["output_size"])
	require.Equal(t, fmt.Sprint(sqs.SystemTime), row["system_time"])
	require.Equal(t, fmt.Sprint(sqs.UserTime), row["user_time"])
	require.Equal(t, fmt.Sprint(sqs.WallTime), row["wall_time_ms"])
}

// TestDirectIngestSoftwareWithLongFields tests that software with reported long fields
// are inserted properly and subsequent reports of the same software do not generate new
// entries in the `software` table. (It mainly tests the comparison between the currenly
// inserted software and the incoming software from a host.)
func (s *integrationTestSuite) TestDirectIngestSoftwareWithLongFields() {
	t := s.T()

	appConfig, err := s.ds.AppConfig(context.Background())
	require.NoError(t, err)
	appConfig.Features.EnableSoftwareInventory = true

	globalHost, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   ptr.String(uuid.New().String()),
		NodeKey:         ptr.String(uuid.New().String()),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%sfoo.global", t.Name()),
		Platform:        "darwin",
	})
	require.NoError(t, err)

	// Simulate a osquery agent on Windows reporting a software row for Wireshark.
	rows := []map[string]string{
		{
			"name":           "Wireshark 4.0.8 64-bit",
			"version":        "4.0.8",
			"type":           "Program (Windows)",
			"source":         "programs",
			"vendor":         "The Wireshark developer community, https://www.wireshark.org",
			"installed_path": "C:\\Program Files\\Wireshark",
		},
	}
	detailQueries := osquery_utils.GetDetailQueries(context.Background(), config.FleetConfig{}, appConfig, &appConfig.Features)
	err = detailQueries["software_windows"].DirectIngestFunc(
		context.Background(),
		log.NewNopLogger(),
		globalHost,
		s.ds,
		rows,
	)
	require.NoError(t, err)

	// Check that the software was properly ingested.
	softwareQueryByName := "SELECT id, name, version, source, bundle_identifier, `release`, arch, vendor FROM software WHERE name = ?;"
	var wiresharkSoftware fleet.Software
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), q, &wiresharkSoftware, softwareQueryByName, "Wireshark 4.0.8 64-bit")
	})
	require.NotZero(t, wiresharkSoftware.ID)
	require.Equal(t, "Wireshark 4.0.8 64-bit", wiresharkSoftware.Name)
	require.Equal(t, "4.0.8", wiresharkSoftware.Version)
	require.Equal(t, "programs", wiresharkSoftware.Source)
	require.Empty(t, wiresharkSoftware.BundleIdentifier)
	require.Empty(t, wiresharkSoftware.Release)
	require.Empty(t, wiresharkSoftware.Arch)
	require.Equal(t, "The Wireshark developer community, https://www.wireshark.org", wiresharkSoftware.Vendor)
	hostSoftwareInstalledPathsQuery := `SELECT installed_path FROM host_software_installed_paths WHERE software_id = ?;`
	var wiresharkSoftwareInstalledPath string
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), q, &wiresharkSoftwareInstalledPath, hostSoftwareInstalledPathsQuery, wiresharkSoftware.ID)
	})
	require.Equal(t, "C:\\Program Files\\Wireshark", wiresharkSoftwareInstalledPath)

	// We now check that the same software is not created again as a new row when it is received again during software ingestion.
	err = detailQueries["software_windows"].DirectIngestFunc(
		context.Background(),
		log.NewNopLogger(),
		globalHost,
		s.ds,
		rows,
	)
	require.NoError(t, err)
	var wiresharkSoftware2 fleet.Software
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), q, &wiresharkSoftware2, softwareQueryByName, "Wireshark 4.0.8 64-bit")
	})
	require.NotZero(t, wiresharkSoftware2.ID)
	require.Equal(t, wiresharkSoftware.ID, wiresharkSoftware2.ID)

	// Simulate a osquery agent on Windows reporting a software row with a longer than 114 chars vendor field.
	rows = []map[string]string{
		{
			"name":           "Foobar" + strings.Repeat("A", fleet.SoftwareNameMaxLength),
			"version":        "4.0.8" + strings.Repeat("B", fleet.SoftwareVersionMaxLength),
			"type":           "Program (Windows)",
			"source":         "programs" + strings.Repeat("C", fleet.SoftwareSourceMaxLength),
			"vendor":         strings.Repeat("D", fleet.SoftwareVendorMaxLength+1),
			"installed_path": "C:\\Program Files\\Foobar",
			// Test UTF-8 encoded strings.
			"bundle_identifier": strings.Repeat("⌘", fleet.SoftwareBundleIdentifierMaxLength+1),
			"release":           strings.Repeat("F", fleet.SoftwareReleaseMaxLength-1) + "⌘⌘",
			"arch":              strings.Repeat("G", fleet.SoftwareArchMaxLength+1),
		},
	}

	err = detailQueries["software_windows"].DirectIngestFunc(
		context.Background(),
		log.NewNopLogger(),
		globalHost,
		s.ds,
		rows,
	)
	require.NoError(t, err)

	var foobarSoftware fleet.Software
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), q, &foobarSoftware, softwareQueryByName, "Foobar"+strings.Repeat("A", fleet.SoftwareNameMaxLength-6))
	})
	require.NotZero(t, foobarSoftware.ID)
	require.Equal(t, "Foobar"+strings.Repeat("A", fleet.SoftwareNameMaxLength-6), foobarSoftware.Name)
	require.Equal(t, "4.0.8"+strings.Repeat("B", fleet.SoftwareNameMaxLength-5), foobarSoftware.Version)
	require.Equal(t, "programs"+strings.Repeat("C", fleet.SoftwareSourceMaxLength-8), foobarSoftware.Source)
	// Vendor field is currenty trimmed with a different method (... appended at the end)
	require.Equal(t, strings.Repeat("D", fleet.SoftwareVendorMaxLength-3)+"...", foobarSoftware.Vendor)
	require.Equal(t, strings.Repeat("⌘", fleet.SoftwareBundleIdentifierMaxLength), foobarSoftware.BundleIdentifier)
	require.Equal(t, strings.Repeat("F", fleet.SoftwareReleaseMaxLength-1)+"⌘", foobarSoftware.Release)
	require.Equal(t, strings.Repeat("G", fleet.SoftwareArchMaxLength), foobarSoftware.Arch)

	// We now check that the same software with long (to be trimmed) fields is not created again as a new row.
	err = detailQueries["software_windows"].DirectIngestFunc(
		context.Background(),
		log.NewNopLogger(),
		globalHost,
		s.ds,
		rows,
	)
	require.NoError(t, err)

	var foobarSoftware2 fleet.Software
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), q, &foobarSoftware2, softwareQueryByName, "Foobar"+strings.Repeat("A", fleet.SoftwareNameMaxLength-6))
	})
	require.Equal(t, foobarSoftware.ID, foobarSoftware2.ID)
}

func (s *integrationTestSuite) TestDirectIngestSoftwareWithInvalidFields() {
	t := s.T()

	appConfig, err := s.ds.AppConfig(context.Background())
	require.NoError(t, err)
	appConfig.Features.EnableSoftwareInventory = true

	globalHost, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   ptr.String(uuid.New().String()),
		NodeKey:         ptr.String(uuid.New().String()),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%sfoo.global", t.Name()),
		Platform:        "darwin",
	})
	require.NoError(t, err)

	// Ingesting software without name should not fail, but the software won't be inserted.
	rows := []map[string]string{
		{
			"version":        "4.0.8",
			"type":           "Program (Windows)",
			"source":         "programs",
			"vendor":         "The Wireshark developer community, https://www.wireshark.org",
			"installed_path": "C:\\Program Files\\Wireshark",
			"last_opened_at": "foobar",
		},
	}
	var w1 bytes.Buffer
	logger1 := log.NewJSONLogger(&w1)
	detailQueries := osquery_utils.GetDetailQueries(context.Background(), config.FleetConfig{}, appConfig, &appConfig.Features)
	err = detailQueries["software_windows"].DirectIngestFunc(
		context.Background(),
		logger1,
		globalHost,
		s.ds,
		rows,
	)
	require.NoError(t, err)
	logs1, err := io.ReadAll(&w1)
	require.NoError(t, err)
	require.Contains(t, string(logs1), "host reported software with empty name", string(logs1))
	require.Contains(t, string(logs1), "debug")

	// Check that the software was not ingested.
	softwareQueryByVendor := "SELECT id, name, version, source, bundle_identifier, `release`, arch, vendor FROM software WHERE vendor = ?;"
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		var wiresharkSoftware fleet.Software
		if sqlx.GetContext(context.Background(), q, &wiresharkSoftware, softwareQueryByVendor, "The Wireshark developer community, https://www.wireshark.org") != sql.ErrNoRows {
			return errors.New("expected no results")
		}
		return nil
	})

	// Ingesting software without source should not fail, but the software won't be inserted.
	rows = []map[string]string{
		{
			"name":           "Wireshark 4.0.8 64-bit",
			"version":        "4.0.8",
			"type":           "Program (Windows)",
			"vendor":         "The Wireshark developer community, https://www.wireshark.org",
			"installed_path": "C:\\Program Files\\Wireshark",
			"last_opened_at": "foobar",
		},
	}
	detailQueries = osquery_utils.GetDetailQueries(context.Background(), config.FleetConfig{}, appConfig, &appConfig.Features)
	var w2 bytes.Buffer
	logger2 := log.NewJSONLogger(&w2)
	err = detailQueries["software_windows"].DirectIngestFunc(
		context.Background(),
		logger2,
		globalHost,
		s.ds,
		rows,
	)
	require.NoError(t, err)
	logs2, err := io.ReadAll(&w2)
	require.NoError(t, err)
	require.Contains(t, string(logs2), "host reported software with empty source")
	require.Contains(t, string(logs2), "debug")

	// Check that the software was not ingested.
	softwareQueryByName := "SELECT id, name, version, source, bundle_identifier, `release`, arch, vendor FROM software WHERE name = ?;"
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		var wiresharkSoftware fleet.Software
		if sqlx.GetContext(context.Background(), q, &wiresharkSoftware, softwareQueryByName, "Wireshark 4.0.8 64-bit") != sql.ErrNoRows {
			return errors.New("expected no results")
		}
		return nil
	})

	// Ingesting software with invalid last_opened_at should not fail (only log a debug error)
	rows = []map[string]string{
		{
			"name":           "Wireshark 4.0.8 64-bit",
			"version":        "4.0.8",
			"type":           "Program (Windows)",
			"source":         "programs",
			"vendor":         "The Wireshark developer community, https://www.wireshark.org",
			"installed_path": "C:\\Program Files\\Wireshark",
			"last_opened_at": "foobar",
		},
	}
	var w3 bytes.Buffer
	logger3 := log.NewJSONLogger(&w3)
	detailQueries = osquery_utils.GetDetailQueries(context.Background(), config.FleetConfig{}, appConfig, &appConfig.Features)
	err = detailQueries["software_windows"].DirectIngestFunc(
		context.Background(),
		logger3,
		globalHost,
		s.ds,
		rows,
	)
	require.NoError(t, err)
	logs3, err := io.ReadAll(&w3)
	require.NoError(t, err)
	require.Contains(t, string(logs3), "host reported software with invalid last opened timestamp")
	require.Contains(t, string(logs3), "debug")

	// Check that the software was properly ingested.
	var wiresharkSoftware fleet.Software
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), q, &wiresharkSoftware, softwareQueryByName, "Wireshark 4.0.8 64-bit")
	})
	require.NotZero(t, wiresharkSoftware.ID)
}

func (s *integrationTestSuite) TestOrbitConfigExtensions() {
	t := s.T()
	ctx := context.Background()

	appCfg, err := s.ds.AppConfig(ctx)
	require.NoError(t, err)
	defer func() {
		err = s.ds.SaveAppConfig(ctx, appCfg)
		require.NoError(t, err)
	}()

	// Orbit client gets no extensions if extensions are not configured.
	orbitLinuxClient := createOrbitEnrolledHost(t, "linux", "foobar1", s.ds)
	resp := orbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *orbitLinuxClient.OrbitNodeKey)), http.StatusOK, &resp)
	require.Empty(t, resp.Extensions)

	// Attempt to add extensions (should succeed).
	s.DoRaw("PATCH", "/api/latest/fleet/config", []byte(`{
	"agent_options": {
		"config": {
			"options": {
				"pack_delimiter": "/",
				"logger_tls_period": 10,
				"distributed_plugin": "tls",
				"disable_distributed": false,
				"logger_tls_endpoint": "/api/osquery/log",
				"distributed_interval": 10,
				"distributed_tls_max_attempts": 3
			}
		},
		"extensions": {
			"hello_world_linux": {
				"channel": "stable",
				"platform": "linux"
			},
			"hello_mars_linux": {
				"channel": "stable",
				"platform": "linux"
			},
			"hello_world_macos": {
				"channel": "stable",
				"platform": "macos"
			}
		}
	}
}`), http.StatusOK)

	// Attempt to add labels to extensions (only available on premium).
	s.DoRaw("PATCH", "/api/latest/fleet/config", []byte(`{
  "agent_options": {
	"config": {
		"options": {
		"pack_delimiter": "/",
		"logger_tls_period": 10,
		"distributed_plugin": "tls",
		"disable_distributed": false,
		"logger_tls_endpoint": "/api/osquery/log",
		"distributed_interval": 10,
		"distributed_tls_max_attempts": 3
		}
	},
	"extensions": {
		"hello_world_linux": {
			"channel": "stable",
			"platform": "linux"
		},
		"hello_world_macos": {
			"labels": [
				"All hosts",
				"Some label"
			],
			"channel": "stable",
			"platform": "macos"
		},
		"hello_world_windows": {
			"channel": "stable",
			"platform": "windows"
		}
	}
  }
}`), http.StatusBadRequest)

	// Orbit client gets extensions configured for its platform.
	orbitDarwinClient := createOrbitEnrolledHost(t, "darwin", "foobar2", s.ds)
	resp = orbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *orbitDarwinClient.OrbitNodeKey)), http.StatusOK, &resp)
	require.JSONEq(t, `{
    "hello_world_macos": {
      "platform": "macos",
      "channel": "stable"
    }
  }`, string(resp.Extensions))

	orbitWindowsClient := createOrbitEnrolledHost(t, "windows", "foobar3", s.ds)

	// Orbit client gets no extensions if none of the platforms target it.
	resp = orbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *orbitWindowsClient.OrbitNodeKey)), http.StatusOK, &resp)
	require.Empty(t, resp.Extensions)

	// Orbit client gets the two extensions configured for its platform.
	resp = orbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *orbitLinuxClient.OrbitNodeKey)), http.StatusOK, &resp)
	require.JSONEq(t, `{
	"hello_world_linux": {
		"channel": "stable",
		"platform": "linux"
	},
	"hello_mars_linux": {
		"channel": "stable",
		"platform": "linux"
	}
  }`, string(resp.Extensions))
}

func (s *integrationTestSuite) TestHostsReportWithPolicyResults() {
	t := s.T()
	ctx := context.Background()

	newHostFunc := func(name string) *fleet.Host {
		host, err := s.ds.NewHost(ctx, &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now(),
			NodeKey:         ptr.String(name),
			UUID:            name,
			Hostname:        "foo.local." + name,
		})
		require.NoError(t, err)
		require.NotNil(t, host)
		return host
	}

	hostCount := 10
	hosts := make([]*fleet.Host, 0, hostCount)
	for i := 0; i < hostCount; i++ {
		hosts = append(hosts, newHostFunc(fmt.Sprintf("h%d", i)))
	}

	globalPolicy0, err := s.ds.NewGlobalPolicy(ctx, &test.UserAdmin.ID, fleet.PolicyPayload{
		Name:  "foobar0",
		Query: "SELECT 0;",
	})
	require.NoError(t, err)
	globalPolicy1, err := s.ds.NewGlobalPolicy(ctx, &test.UserAdmin.ID, fleet.PolicyPayload{
		Name:  "foobar1",
		Query: "SELECT 1;",
	})
	require.NoError(t, err)
	globalPolicy2, err := s.ds.NewGlobalPolicy(ctx, &test.UserAdmin.ID, fleet.PolicyPayload{
		Name:  "foobar2",
		Query: "SELECT 2;",
	})
	require.NoError(t, err)

	for i, host := range hosts {
		// All hosts pass the globalPolicy0
		err := s.ds.RecordPolicyQueryExecutions(context.Background(), host, map[uint]*bool{globalPolicy0.ID: ptr.Bool(true)}, time.Now(), false)
		require.NoError(t, err)

		if i%2 == 0 {
			// Half of the hosts pass the globalPolicy1 and fail the globalPolicy2
			err := s.ds.RecordPolicyQueryExecutions(context.Background(), host, map[uint]*bool{globalPolicy1.ID: ptr.Bool(true)}, time.Now(), false)
			require.NoError(t, err)
			err = s.ds.RecordPolicyQueryExecutions(context.Background(), host, map[uint]*bool{globalPolicy2.ID: ptr.Bool(false)}, time.Now(), false)
			require.NoError(t, err)
		} else {
			// Half of the hosts pass the globalPolicy2 and fail the globalPolicy1
			err := s.ds.RecordPolicyQueryExecutions(context.Background(), host, map[uint]*bool{globalPolicy1.ID: ptr.Bool(false)}, time.Now(), false)
			require.NoError(t, err)
			err = s.ds.RecordPolicyQueryExecutions(context.Background(), host, map[uint]*bool{globalPolicy2.ID: ptr.Bool(true)}, time.Now(), false)
			require.NoError(t, err)
		}
	}

	// The hosts/report endpoint uses svc.ds.ListHosts with page=0, per_page=0, thus we are
	// testing the non optimized for pagination queries for failing policies calculation.
	res := s.DoRaw("GET", "/api/latest/fleet/hosts/report", nil, http.StatusOK, "format", "csv")
	rows1, err := csv.NewReader(res.Body).ReadAll()
	res.Body.Close()
	require.NoError(t, err)
	require.Len(t, rows1, len(hosts)+1) // all hosts + header row
	assert.Len(t, rows1[0], 54)         // total number of cols

	var (
		idIdx     int
		issuesIdx int
	)
	for colIdx, column := range rows1[0] {
		switch column {
		case "issues":
			issuesIdx = colIdx
		case "id":
			idIdx = colIdx
		}
	}

	for i := 1; i < len(hosts)+1; i++ {
		row := rows1[i]
		require.Equal(t, row[issuesIdx], "1")
	}

	// Running with disable_issues=true (which overrides disable_failing_policies=false) disable the counting of failed policies for a host.
	// Thus, all "issues" values should be 0.
	res = s.DoRaw(
		"GET", "/api/latest/fleet/hosts/report", nil, http.StatusOK, "format", "csv", "disable_failing_policies", "false", "disable_issues",
		"true",
	)
	rows2, err := csv.NewReader(res.Body).ReadAll()
	res.Body.Close()
	require.NoError(t, err)
	require.Len(t, rows2, len(hosts)+1) // all hosts + header row
	assert.Len(t, rows2[0], 54)         // total number of cols

	// Check that all hosts have 0 issues and that they match the previous call to `/hosts/report`.
	for i := 1; i < len(hosts)+1; i++ {
		row := rows2[i]
		require.Equal(t, row[issuesIdx], "0")
		row1 := rows1[i]
		require.Equal(t, row[idIdx], row1[idIdx])
	}

	for _, tc := range []struct {
		name      string
		args      []string
		checkRows func(t *testing.T, rows [][]string)
	}{
		{
			name: "get hosts that fail globalPolicy0",
			args: []string{"policy_id", fmt.Sprint(globalPolicy0.ID), "policy_response", "failing"},
			checkRows: func(t *testing.T, rows [][]string) {
				require.Len(t, rows, 1) // just header row, all hosts pass such policy.
			},
		},
		{
			name: "get hosts that pass globalPolicy0",
			args: []string{"policy_id", fmt.Sprint(globalPolicy0.ID), "policy_response", "passing"},
			checkRows: func(t *testing.T, rows [][]string) {
				require.Len(t, rows, len(hosts)+1) // all hosts + header row, all hosts pass such policy.
			},
		},
		{
			name: "get hosts that fail globalPolicy1",
			args: []string{"policy_id", fmt.Sprint(globalPolicy1.ID), "policy_response", "failing"},
			checkRows: func(t *testing.T, rows [][]string) {
				require.Len(t, rows, len(hosts)/2+1) // half of hosts + header row.
			},
		},
		{
			name: "get hosts that pass globalPolicy1",
			args: []string{"policy_id", fmt.Sprint(globalPolicy1.ID), "policy_response", "passing"},
			checkRows: func(t *testing.T, rows [][]string) {
				require.Len(t, rows, len(hosts)/2+1) // half of hosts + header row.
			},
		},
		{
			name: "get hosts that fail globalPolicy2",
			args: []string{"policy_id", fmt.Sprint(globalPolicy2.ID), "policy_response", "failing"},
			checkRows: func(t *testing.T, rows [][]string) {
				require.Len(t, rows, len(hosts)/2+1) // half of hosts + header row.
			},
		},
		{
			name: "get hosts that pass globalPolicy2",
			args: []string{"policy_id", fmt.Sprint(globalPolicy2.ID), "policy_response", "passing"},
			checkRows: func(t *testing.T, rows [][]string) {
				require.Len(t, rows, len(hosts)/2+1) // half of hosts + header row.
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			res = s.DoRaw("GET", "/api/latest/fleet/hosts/report", nil, http.StatusOK, append(tc.args, "format", "csv")...)
			rows, err := csv.NewReader(res.Body).ReadAll()
			res.Body.Close()
			require.NoError(t, err)
			tc.checkRows(t, rows)
			// Test the same with "disable_issues=true" which should not change the result.
			res = s.DoRaw(
				"GET", "/api/latest/fleet/hosts/report", nil, http.StatusOK, append(tc.args, "format", "csv", "disable_issues", "true")...,
			)
			rows, err = csv.NewReader(res.Body).ReadAll()
			res.Body.Close()
			require.NoError(t, err)
			tc.checkRows(t, rows)
		})
	}
}

func (s *integrationTestSuite) TestQueryReports() {
	t := s.T()
	ctx := context.Background()

	team1, err := s.ds.NewTeam(ctx, &fleet.Team{
		ID:          42,
		Name:        "team1",
		Description: "desc team1",
	})
	require.NoError(t, err)

	host1Global, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String("1"),
		UUID:            "1",
		Hostname:        "foo.local1",
		OsqueryHostID:   ptr.String("1"),
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-58",
		Platform:        "ubuntu",
	})
	require.NoError(t, err)

	host2Global, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String("2"),
		UUID:            "2",
		Hostname:        "foo.local2",
		OsqueryHostID:   ptr.String("2"),
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-59",
		Platform:        "ubuntu",
	})
	require.NoError(t, err)

	host2Team1, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String("3"),
		UUID:            "3",
		ComputerName:    "Foo Local3",
		Hostname:        "foo.local3",
		OsqueryHostID:   ptr.String("3"),
		PrimaryIP:       "192.168.1.3",
		PrimaryMac:      "30-65-EC-6F-C4-60",
		Platform:        "darwin",
	})
	require.NoError(t, err)

	err = s.ds.AddHostsToTeam(ctx, &team1.ID, []uint{host2Team1.ID})
	require.NoError(t, err)

	osqueryInfoQuery, err := s.ds.NewQuery(ctx, &fleet.Query{
		Name:               "Osquery info",
		Description:        "osquery_info table",
		Query:              "select * from osquery_info;",
		Saved:              true,
		Interval:           30,
		AutomationsEnabled: true,
		DiscardData:        false,
		TeamID:             nil,
		Logging:            fleet.LoggingSnapshot,
	})
	require.NoError(t, err)

	usbDevicesQuery, err := s.ds.NewQuery(ctx, &fleet.Query{
		Name:               "USB devices",
		Description:        "usb_devices table",
		Query:              "select * from usb_devices;",
		Saved:              true,
		Interval:           60,
		AutomationsEnabled: true,
		DiscardData:        false,
		TeamID:             ptr.Uint(team1.ID),
		Logging:            fleet.LoggingSnapshot,
	})
	require.NoError(t, err)

	// Should return no results.
	var gqrr getQueryReportResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d/report", usbDevicesQuery.ID), getQueryReportRequest{}, http.StatusOK, &gqrr)
	require.NoError(t, gqrr.Err)
	require.Equal(t, usbDevicesQuery.ID, gqrr.QueryID)
	require.NotNil(t, gqrr.Results)
	require.Len(t, gqrr.Results, 0)

	var ghqrr getHostQueryReportResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/queries/%d", host1Global.ID, usbDevicesQuery.ID), getHostQueryReportRequest{}, http.StatusOK, &ghqrr)
	require.NoError(t, ghqrr.Err)
	require.Equal(t, usbDevicesQuery.ID, ghqrr.QueryID)
	require.Equal(t, host1Global.ID, ghqrr.HostID)
	require.Nil(t, ghqrr.LastFetched)
	require.False(t, ghqrr.ReportClipped)
	require.NotNil(t, ghqrr.Results)
	require.Len(t, ghqrr.Results, 0)

	slreq := submitLogsRequest{
		NodeKey: *host2Team1.NodeKey,
		LogType: "result",
		Data: json.RawMessage(`[{
  "snapshot": [
    {
      "class": "239",
      "model": "HD Pro Webcam C920",
      "model_id": "0892",
      "protocol": "",
      "removable": "1",
      "serial": "zoobar",
      "subclass": "2",
      "usb_address": "3",
      "usb_port": "1",
      "vendor": "",
      "vendor_id": "046d",
      "version": "0.19"
    },
    {
      "class": "0",
      "model": "Apple Internal Keyboard / Trackpad",
      "model_id": "027e",
      "protocol": "",
      "removable": "0",
      "serial": "foobar",
      "subclass": "0",
      "usb_address": "8",
      "usb_port": "5",
      "vendor": "Apple Inc.",
      "vendor_id": "05ac",
      "version": "9.33"
    }
  ],
  "action": "snapshot",
  "name": "pack/team-` + usbDevicesQuery.TeamIDStr() + `/` + usbDevicesQuery.Name + `",
  "hostIdentifier": "` + *host2Team1.OsqueryHostID + `",
  "calendarTime": "Fri Oct  6 17:32:08 2023 UTC",
  "unixTime": 1696613528,
  "epoch": 0,
  "counter": 0,
  "numerics": false,
  "decorations": {
    "host_uuid": "` + host2Team1.UUID + `",
    "hostname": "` + host2Team1.Hostname + `"
  }
},
{
  "snapshot": [
    {
      "build_distro": "10.14",
      "build_platform": "darwin",
      "config_hash": "eed0d8296e5f90b790a23814a9db7a127b13498d",
      "config_valid": "1",
      "extensions": "active",
      "instance_id": "7f02ff0f-f8a7-4ba9-a1d2-66836b154f4a",
      "pid": "95637",
      "platform_mask": "21",
      "start_time": "1696611201",
      "uuid": "` + host2Team1.UUID + `",
      "version": "5.9.1",
      "watcher": "95636"
    }
  ],
  "action": "snapshot",
  "name": "pack/Global/` + osqueryInfoQuery.Name + `",
  "hostIdentifier": "` + *host2Team1.OsqueryHostID + `",
  "calendarTime": "Fri Oct  6 18:08:18 2023 UTC",
  "unixTime": 1696615698,
  "epoch": 0,
  "counter": 0,
  "numerics": false,
  "decorations": {
    "host_uuid": "` + host2Team1.UUID + `",
    "hostname": "` + host2Team1.Hostname + `"
  }
}
]`),
	}
	slres := submitLogsResponse{}
	s.DoJSON("POST", "/api/osquery/log", slreq, http.StatusOK, &slres)
	require.NoError(t, slres.Err)

	slreq = submitLogsRequest{
		NodeKey: *host1Global.NodeKey,
		LogType: "result",
		Data: json.RawMessage(`[{
  "snapshot": [
    {
      "build_distro": "centos7",
      "build_platform": "linux",
      "config_hash": "eed0d8296e5f90b790a23814a9db7a127b13498d",
      "config_valid": "1",
      "extensions": "active",
      "instance_id": "e5799132-85ab-4cfa-89f3-03e0dd3c509a",
      "pid": "3574",
      "platform_mask": "9",
      "start_time": "1696502961",
      "uuid": "` + host1Global.UUID + `",
      "version": "5.9.2",
      "watcher": "3570"
    }
  ],
  "action": "snapshot",
  "name": "pack/Global/` + osqueryInfoQuery.Name + `",
  "hostIdentifier": "` + *host1Global.OsqueryHostID + `",
  "calendarTime": "Fri Oct  6 18:13:04 2023 UTC",
  "unixTime": 1696615984,
  "epoch": 0,
  "counter": 0,
  "numerics": false,
  "decorations": {
    "host_uuid": "187c4d56-8e45-1a9d-8513-ac17efd2f0fd",
    "hostname": "` + host1Global.Hostname + `"
  }
}]`),
	}
	slres = submitLogsResponse{}
	s.DoJSON("POST", "/api/osquery/log", slreq, http.StatusOK, &slres)
	require.NoError(t, slres.Err)

	emptyslreq := submitLogsRequest{
		NodeKey: *host2Global.NodeKey,
		LogType: "result",
		Data: json.RawMessage(`[{
			  "snapshot": [],
			  "action": "snapshot",
			  "name": "pack/Global/` + osqueryInfoQuery.Name + `",
			  "hostIdentifier": "` + *host1Global.OsqueryHostID + `",
			  "calendarTime": "Fri Oct  6 18:13:04 2023 UTC",
			  "unixTime": 1696615984,
			  "epoch": 0,
			  "counter": 0,
			  "numerics": false,
			  "decorations": {
				"host_uuid": "187c4d56-8e45-1a9d-8513-ac17efd2f0fd",
				"hostname": "` + host1Global.Hostname + `"
			  }
			}]`),
	}
	emptyslres := submitLogsResponse{}
	s.DoJSON("POST", "/api/osquery/log", emptyslreq, http.StatusOK, &emptyslres)
	require.NoError(t, emptyslres.Err)

	gqrr = getQueryReportResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d/report", usbDevicesQuery.ID), getQueryReportRequest{}, http.StatusOK, &gqrr)
	require.NoError(t, gqrr.Err)
	require.Equal(t, usbDevicesQuery.ID, gqrr.QueryID)
	require.Len(t, gqrr.Results, 2)
	sort.Slice(gqrr.Results, func(i, j int) bool {
		// Let's just pick a known column of the query to sort.
		return gqrr.Results[i].Columns["usb_port"] < gqrr.Results[j].Columns["usb_port"]
	})
	require.Equal(t, host2Team1.ID, gqrr.Results[0].HostID)
	require.Equal(t, host2Team1.DisplayName(), gqrr.Results[0].Hostname)
	require.NotZero(t, gqrr.Results[0].LastFetched)
	require.Equal(t, map[string]string{
		"class":       "239",
		"model":       "HD Pro Webcam C920",
		"model_id":    "0892",
		"protocol":    "",
		"removable":   "1",
		"serial":      "zoobar",
		"subclass":    "2",
		"usb_address": "3",
		"usb_port":    "1",
		"vendor":      "",
		"vendor_id":   "046d",
		"version":     "0.19",
	}, gqrr.Results[0].Columns)
	require.Equal(t, host2Team1.ID, gqrr.Results[1].HostID)
	require.Equal(t, host2Team1.DisplayName(), gqrr.Results[1].Hostname)
	require.NotZero(t, gqrr.Results[1].LastFetched)
	require.Equal(t, map[string]string{
		"class":       "0",
		"model":       "Apple Internal Keyboard / Trackpad",
		"model_id":    "027e",
		"protocol":    "",
		"removable":   "0",
		"serial":      "foobar",
		"subclass":    "0",
		"usb_address": "8",
		"usb_port":    "5",
		"vendor":      "Apple Inc.",
		"vendor_id":   "05ac",
		"version":     "9.33",
	}, gqrr.Results[1].Columns)

	ghqrr = getHostQueryReportResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/queries/%d", host2Team1.ID, usbDevicesQuery.ID), getHostQueryReportRequest{}, http.StatusOK, &ghqrr)
	require.NoError(t, ghqrr.Err)
	require.Equal(t, usbDevicesQuery.ID, ghqrr.QueryID)
	require.Equal(t, host2Team1.ID, ghqrr.HostID)
	require.NotNil(t, ghqrr.LastFetched)
	require.False(t, ghqrr.ReportClipped)
	require.Len(t, ghqrr.Results, 2)
	sort.Slice(gqrr.Results, func(i, j int) bool {
		// Let's just pick a known column of the query to sort.
		return gqrr.Results[i].Columns["usb_port"] < gqrr.Results[j].Columns["usb_port"]
	})
	require.Equal(t, map[string]string{
		"class":       "239",
		"model":       "HD Pro Webcam C920",
		"model_id":    "0892",
		"protocol":    "",
		"removable":   "1",
		"serial":      "zoobar",
		"subclass":    "2",
		"usb_address": "3",
		"usb_port":    "1",
		"vendor":      "",
		"vendor_id":   "046d",
		"version":     "0.19",
	}, ghqrr.Results[0].Columns)
	require.Equal(t, map[string]string{
		"class":       "0",
		"model":       "Apple Internal Keyboard / Trackpad",
		"model_id":    "027e",
		"protocol":    "",
		"removable":   "0",
		"serial":      "foobar",
		"subclass":    "0",
		"usb_address": "8",
		"usb_port":    "5",
		"vendor":      "Apple Inc.",
		"vendor_id":   "05ac",
		"version":     "9.33",
	}, ghqrr.Results[1].Columns)

	gqrr = getQueryReportResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d/report", osqueryInfoQuery.ID), getQueryReportRequest{}, http.StatusOK, &gqrr)
	require.NoError(t, gqrr.Err)
	require.Equal(t, osqueryInfoQuery.ID, gqrr.QueryID)
	require.Len(t, gqrr.Results, 2)
	sort.Slice(gqrr.Results, func(i, j int) bool {
		// Let's just pick a known column of the query to sort.
		return gqrr.Results[i].Columns["version"] > gqrr.Results[j].Columns["version"]
	})
	require.Equal(t, host1Global.ID, gqrr.Results[0].HostID)
	require.Equal(t, host1Global.DisplayName(), gqrr.Results[0].Hostname)
	require.NotZero(t, gqrr.Results[0].LastFetched)
	require.Equal(t, map[string]string{
		"build_distro":   "centos7",
		"build_platform": "linux",
		"config_hash":    "eed0d8296e5f90b790a23814a9db7a127b13498d",
		"config_valid":   "1",
		"extensions":     "active",
		"instance_id":    "e5799132-85ab-4cfa-89f3-03e0dd3c509a",
		"pid":            "3574",
		"platform_mask":  "9",
		"start_time":     "1696502961",
		"uuid":           host1Global.UUID,
		"version":        "5.9.2",
		"watcher":        "3570",
	}, gqrr.Results[0].Columns)
	require.Equal(t, host2Team1.ID, gqrr.Results[1].HostID)
	require.Equal(t, host2Team1.DisplayName(), gqrr.Results[1].Hostname)
	require.NotZero(t, gqrr.Results[1].LastFetched)
	require.Equal(t, map[string]string{
		"build_distro":   "10.14",
		"build_platform": "darwin",
		"config_hash":    "eed0d8296e5f90b790a23814a9db7a127b13498d",
		"config_valid":   "1",
		"extensions":     "active",
		"instance_id":    "7f02ff0f-f8a7-4ba9-a1d2-66836b154f4a",
		"pid":            "95637",
		"platform_mask":  "21",
		"start_time":     "1696611201",
		"uuid":           host2Team1.UUID,
		"version":        "5.9.1",
		"watcher":        "95636",
	}, gqrr.Results[1].Columns)

	ghqrr = getHostQueryReportResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/queries/%d", host1Global.ID, osqueryInfoQuery.ID), getHostQueryReportRequest{}, http.StatusOK, &ghqrr)
	require.NoError(t, ghqrr.Err)
	require.Equal(t, osqueryInfoQuery.ID, ghqrr.QueryID)
	require.Equal(t, host1Global.ID, ghqrr.HostID)
	require.NotNil(t, ghqrr.LastFetched)
	require.False(t, ghqrr.ReportClipped)
	require.Len(t, ghqrr.Results, 1)
	require.Equal(t, map[string]string{
		"build_distro":   "centos7",
		"build_platform": "linux",
		"config_hash":    "eed0d8296e5f90b790a23814a9db7a127b13498d",
		"config_valid":   "1",
		"extensions":     "active",
		"instance_id":    "e5799132-85ab-4cfa-89f3-03e0dd3c509a",
		"pid":            "3574",
		"platform_mask":  "9",
		"start_time":     "1696502961",
		"uuid":           host1Global.UUID,
		"version":        "5.9.2",
		"watcher":        "3570",
	}, ghqrr.Results[0].Columns)

	ghqrr = getHostQueryReportResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/queries/%d", host2Global.ID, osqueryInfoQuery.ID), getHostQueryReportRequest{}, http.StatusOK, &ghqrr)
	require.NoError(t, ghqrr.Err)
	require.Equal(t, osqueryInfoQuery.ID, ghqrr.QueryID)
	require.Equal(t, host2Global.ID, ghqrr.HostID)
	require.NotNil(t, ghqrr.LastFetched)
	require.False(t, ghqrr.ReportClipped)
	require.Len(t, ghqrr.Results, 0)

	// verify that certain modifications to queries don't cause result deletion
	modifyQueryResp := modifyQueryResponse{}
	updatedDesc := "Updated description"
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/queries/%d", osqueryInfoQuery.ID), modifyQueryRequest{ID: osqueryInfoQuery.ID, QueryPayload: fleet.QueryPayload{Description: &updatedDesc}}, http.StatusOK, &modifyQueryResp)
	require.Equal(t, updatedDesc, modifyQueryResp.Query.Description)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d/report", osqueryInfoQuery.ID), getQueryReportRequest{}, http.StatusOK, &gqrr)
	require.Len(t, gqrr.Results, 2)

	// now update the query and verify that results are deleted
	updatedQuery := "SELECT * FROM some_new_table;"
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/queries/%d", osqueryInfoQuery.ID), modifyQueryRequest{ID: osqueryInfoQuery.ID, QueryPayload: fleet.QueryPayload{Query: &updatedQuery}}, http.StatusOK, &modifyQueryResp)
	require.Equal(t, updatedQuery, modifyQueryResp.Query.Query)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d/report", osqueryInfoQuery.ID), getQueryReportRequest{}, http.StatusOK, &gqrr)
	require.Len(t, gqrr.Results, 0)

	// Re-add results to our query and check that they're actually there
	s.DoJSON("POST", "/api/osquery/log", slreq, http.StatusOK, &slres)
	require.NoError(t, slres.Err)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d/report", osqueryInfoQuery.ID), getQueryReportRequest{}, http.StatusOK, &gqrr)
	require.Len(t, gqrr.Results, 1)

	// now update the platform and verify that results are deleted
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/queries/%d", osqueryInfoQuery.ID), modifyQueryRequest{
		ID: osqueryInfoQuery.ID,
		QueryPayload: fleet.QueryPayload{
			Platform: ptr.String("linux"),
		},
	},
		http.StatusOK,
		&modifyQueryResp,
	)
	require.Equal(t, "linux", modifyQueryResp.Query.Platform)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d/report", osqueryInfoQuery.ID), getQueryReportRequest{}, http.StatusOK, &gqrr)
	require.Len(t, gqrr.Results, 0)

	// Re-add results to our query and check that they're actually there
	s.DoJSON("POST", "/api/osquery/log", slreq, http.StatusOK, &slres)
	require.NoError(t, slres.Err)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d/report", osqueryInfoQuery.ID), getQueryReportRequest{}, http.StatusOK, &gqrr)
	require.Len(t, gqrr.Results, 1)

	// now update the platform to the same value and verify that results are not deleted
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/queries/%d", osqueryInfoQuery.ID), modifyQueryRequest{
		ID: osqueryInfoQuery.ID,
		QueryPayload: fleet.QueryPayload{
			Platform: ptr.String("linux"),
		},
	},
		http.StatusOK,
		&modifyQueryResp,
	)
	require.Equal(t, "linux", modifyQueryResp.Query.Platform)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d/report", osqueryInfoQuery.ID), getQueryReportRequest{}, http.StatusOK, &gqrr)
	require.Len(t, gqrr.Results, 1)

	// now update the min_osquery_version and verify that results are deleted
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/queries/%d", osqueryInfoQuery.ID), modifyQueryRequest{
		ID: osqueryInfoQuery.ID,
		QueryPayload: fleet.QueryPayload{
			MinOsqueryVersion: ptr.String("5.9.1"),
		},
	},
		http.StatusOK,
		&modifyQueryResp,
	)
	require.Equal(t, "5.9.1", modifyQueryResp.Query.MinOsqueryVersion)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d/report", osqueryInfoQuery.ID), getQueryReportRequest{}, http.StatusOK, &gqrr)
	require.Len(t, gqrr.Results, 0)

	// Re-add results to our query and check that they're actually there
	s.DoJSON("POST", "/api/osquery/log", slreq, http.StatusOK, &slres)
	require.NoError(t, slres.Err)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d/report", osqueryInfoQuery.ID), getQueryReportRequest{}, http.StatusOK, &gqrr)
	require.Len(t, gqrr.Results, 1)

	// now update the min_osquery_version to another value and verify that results are deleted
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/queries/%d", osqueryInfoQuery.ID), modifyQueryRequest{
		ID: osqueryInfoQuery.ID,
		QueryPayload: fleet.QueryPayload{
			MinOsqueryVersion: ptr.String("5.11.0"),
		},
	},
		http.StatusOK,
		&modifyQueryResp,
	)
	require.Equal(t, "5.11.0", modifyQueryResp.Query.MinOsqueryVersion)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d/report", osqueryInfoQuery.ID), getQueryReportRequest{}, http.StatusOK, &gqrr)
	require.Len(t, gqrr.Results, 0)

	// Re-add results to our query and check that they're actually there
	s.DoJSON("POST", "/api/osquery/log", slreq, http.StatusOK, &slres)
	require.NoError(t, slres.Err)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d/report", osqueryInfoQuery.ID), getQueryReportRequest{}, http.StatusOK, &gqrr)
	require.Len(t, gqrr.Results, 1)

	// now update the min_osquery_version to the same value and verify that results are not deleted
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/queries/%d", osqueryInfoQuery.ID), modifyQueryRequest{
		ID: osqueryInfoQuery.ID,
		QueryPayload: fleet.QueryPayload{
			MinOsqueryVersion: ptr.String("5.11.0"),
		},
	},
		http.StatusOK,
		&modifyQueryResp,
	)
	require.Equal(t, "5.11.0", modifyQueryResp.Query.MinOsqueryVersion)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d/report", osqueryInfoQuery.ID), getQueryReportRequest{}, http.StatusOK, &gqrr)
	require.Len(t, gqrr.Results, 1)

	// now update the query via specs and change the min_osquery_version, results should be deleted.
	osqueryInfoQuerySpec := &fleet.QuerySpec{
		Name:               osqueryInfoQuery.Name,
		Description:        osqueryInfoQuery.Description,
		Query:              osqueryInfoQuery.Query,
		Interval:           osqueryInfoQuery.Interval,
		ObserverCanRun:     osqueryInfoQuery.ObserverCanRun,
		Platform:           osqueryInfoQuery.Platform,
		MinOsqueryVersion:  osqueryInfoQuery.MinOsqueryVersion,
		AutomationsEnabled: osqueryInfoQuery.AutomationsEnabled,
		Logging:            osqueryInfoQuery.Logging,
		DiscardData:        osqueryInfoQuery.DiscardData,
	}
	osqueryInfoQuerySpec.MinOsqueryVersion = "5.12.0"
	var applyResp applyQuerySpecsResponse
	s.DoJSON("POST", "/api/latest/fleet/spec/queries", applyQuerySpecsRequest{
		Specs: []*fleet.QuerySpec{osqueryInfoQuerySpec},
	}, http.StatusOK, &applyResp)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d/report", osqueryInfoQuery.ID), getQueryReportRequest{}, http.StatusOK, &gqrr)
	require.Len(t, gqrr.Results, 0)
	require.False(t, gqrr.ReportClipped)

	// Re-add results to our query and check that they're actually there
	s.DoJSON("POST", "/api/osquery/log", slreq, http.StatusOK, &slres)
	require.NoError(t, slres.Err)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d/report", osqueryInfoQuery.ID), getQueryReportRequest{}, http.StatusOK, &gqrr)
	require.Len(t, gqrr.Results, 1)
	require.False(t, gqrr.ReportClipped)

	// don't change platform or min_osquery_version and results should not be deleted
	s.DoJSON("POST", "/api/latest/fleet/spec/queries", applyQuerySpecsRequest{
		Specs: []*fleet.QuerySpec{osqueryInfoQuerySpec},
	}, http.StatusOK, &applyResp)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d/report", osqueryInfoQuery.ID), getQueryReportRequest{}, http.StatusOK, &gqrr)
	require.Len(t, gqrr.Results, 1)
	require.False(t, gqrr.ReportClipped)

	// now update the platform and results should be deleted.
	osqueryInfoQuerySpec.Platform = "darwin"
	s.DoJSON("POST", "/api/latest/fleet/spec/queries", applyQuerySpecsRequest{
		Specs: []*fleet.QuerySpec{osqueryInfoQuerySpec},
	}, http.StatusOK, &applyResp)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d/report", osqueryInfoQuery.ID), getQueryReportRequest{}, http.StatusOK, &gqrr)
	require.Len(t, gqrr.Results, 0)
	require.False(t, gqrr.ReportClipped)

	// Update logging type, which should cause results deletion
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/queries/%d", usbDevicesQuery.ID), modifyQueryRequest{ID: usbDevicesQuery.ID, QueryPayload: fleet.QueryPayload{Logging: &fleet.LoggingDifferential}}, http.StatusOK, &modifyQueryResp)
	require.Equal(t, fleet.LoggingDifferential, modifyQueryResp.Query.Logging)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d/report", usbDevicesQuery.ID), getQueryReportRequest{}, http.StatusOK, &gqrr)
	require.Len(t, gqrr.Results, 0)
	require.False(t, gqrr.ReportClipped)

	// Re-add results to our query and check that they're actually there
	s.DoJSON("POST", "/api/osquery/log", slreq, http.StatusOK, &slres)
	require.NoError(t, slres.Err)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d/report", osqueryInfoQuery.ID), getQueryReportRequest{}, http.StatusOK, &gqrr)
	require.Len(t, gqrr.Results, 1)
	require.False(t, gqrr.ReportClipped)

	discardData := true
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/queries/%d", osqueryInfoQuery.ID), modifyQueryRequest{ID: osqueryInfoQuery.ID, QueryPayload: fleet.QueryPayload{DiscardData: &discardData}}, http.StatusOK, &modifyQueryResp)
	require.True(t, modifyQueryResp.Query.DiscardData)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d/report", osqueryInfoQuery.ID), getQueryReportRequest{}, http.StatusOK, &gqrr)
	require.Len(t, gqrr.Results, 0)
	require.False(t, gqrr.ReportClipped)

	// check that now that discardData is set, we don't add new results
	s.DoJSON("POST", "/api/osquery/log", slreq, http.StatusOK, &slres)
	require.NoError(t, slres.Err)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d/report", osqueryInfoQuery.ID), getQueryReportRequest{}, http.StatusOK, &gqrr)
	require.Len(t, gqrr.Results, 0)
	require.False(t, gqrr.ReportClipped)

	// Verify that we can't have more than 1k results

	discardData = false
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/queries/%d", osqueryInfoQuery.ID), modifyQueryRequest{ID: osqueryInfoQuery.ID, QueryPayload: fleet.QueryPayload{DiscardData: &discardData}}, http.StatusOK, &modifyQueryResp)
	require.False(t, modifyQueryResp.Query.DiscardData)

	slreq = submitLogsRequest{
		NodeKey: *host1Global.NodeKey,
		LogType: "result",
		Data: json.RawMessage(`[{
  "snapshot": [` + results(fleet.DefaultMaxQueryReportRows, host1Global.UUID) + `
  ],
  "action": "snapshot",
  "name": "pack/Global/` + osqueryInfoQuery.Name + `",
  "hostIdentifier": "` + *host1Global.OsqueryHostID + `",
  "calendarTime": "Fri Oct  6 18:13:04 2023 UTC",
  "unixTime": 1696615984,
  "epoch": 0,
  "counter": 0,
  "numerics": false,
  "decorations": {
    "host_uuid": "187c4d56-8e45-1a9d-8513-ac17efd2f0fd",
    "hostname": "` + host1Global.Hostname + `"
  }
}]`),
	}
	slres = submitLogsResponse{}
	s.DoJSON("POST", "/api/osquery/log", slreq, http.StatusOK, &slres)
	require.NoError(t, slres.Err)

	s.DoJSON("POST", "/api/osquery/log", slreq, http.StatusOK, &slres)
	require.NoError(t, slres.Err)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d/report", osqueryInfoQuery.ID), getQueryReportRequest{}, http.StatusOK, &gqrr)
	require.Len(t, gqrr.Results, fleet.DefaultMaxQueryReportRows)
	require.True(t, gqrr.ReportClipped)

	ghqrr = getHostQueryReportResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/queries/%d", host1Global.ID, osqueryInfoQuery.ID), getHostQueryReportRequest{}, http.StatusOK, &ghqrr)
	require.NoError(t, ghqrr.Err)
	require.Len(t, ghqrr.Results, fleet.DefaultMaxQueryReportRows)
	require.True(t, ghqrr.ReportClipped)

	slreq.Data = json.RawMessage(`[{
  "snapshot": [` + results(1, host1Global.UUID) + `
  ],
  "action": "snapshot",
  "name": "pack/Global/` + osqueryInfoQuery.Name + `",
  "hostIdentifier": "` + *host1Global.OsqueryHostID + `",
  "calendarTime": "Fri Oct  6 18:13:04 2023 UTC",
  "unixTime": 1696615984,
  "epoch": 0,
  "counter": 0,
  "numerics": false,
  "decorations": {
    "host_uuid": "187c4d56-8e45-1a9d-8513-ac17efd2f0fd",
    "hostname": "` + host1Global.Hostname + `"
  }
}]`)

	s.DoJSON("POST", "/api/osquery/log", slreq, http.StatusOK, &slres)
	require.NoError(t, slres.Err)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d/report", osqueryInfoQuery.ID), getQueryReportRequest{}, http.StatusOK, &gqrr)
	require.Len(t, gqrr.Results, fleet.DefaultMaxQueryReportRows)
	require.True(t, gqrr.ReportClipped)

	appConfigSpec := map[string]map[string]int{
		"server_settings": {"query_report_cap": fleet.DefaultMaxQueryReportRows + 1},
	}
	s.Do("PATCH", "/api/latest/fleet/config", appConfigSpec, http.StatusOK)

	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d/report", osqueryInfoQuery.ID), getQueryReportRequest{}, http.StatusOK, &gqrr)
	require.Len(t, gqrr.Results, fleet.DefaultMaxQueryReportRows)
	require.False(t, gqrr.ReportClipped)

	slreq.Data = json.RawMessage(`[{
  "snapshot": [` + results(1002, host1Global.UUID) + `
  ],
  "action": "snapshot",
  "name": "pack/Global/` + osqueryInfoQuery.Name + `",
  "hostIdentifier": "` + *host1Global.OsqueryHostID + `",
  "calendarTime": "Fri Oct  6 18:13:04 2023 UTC",
  "unixTime": 1696615984,
  "epoch": 0,
  "counter": 0,
  "numerics": false,
  "decorations": {
    "host_uuid": "187c4d56-8e45-1a9d-8513-ac17efd2f0fd",
    "hostname": "` + host1Global.Hostname + `"
  }
}]`)

	s.DoJSON("POST", "/api/osquery/log", slreq, http.StatusOK, &slres)
	require.NoError(t, slres.Err)

	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d/report", osqueryInfoQuery.ID), getQueryReportRequest{}, http.StatusOK, &gqrr)
	require.Len(t, gqrr.Results, fleet.DefaultMaxQueryReportRows+1)
	require.True(t, gqrr.ReportClipped)

	// TODO: Set global discard flag and verify that all data is gone.
}

// Creates a set of results for use in tests for Query Results.
func results(num int, hostID string) string {
	b := strings.Builder{}
	for i := 0; i < num; i++ {
		b.WriteString(`    {
      "build_distro": "centos7",
      "build_platform": "linux",
      "config_hash": "eed0d8296e5f90b790a23814a9db7a127b13498d",
      "config_valid": "1",
      "extensions": "active",
      "instance_id": "e5799132-85ab-4cfa-89f3-03e0dd3c509a",
      "pid": "3574",
      "platform_mask": "9",
      "start_time": "1696502961",
      "uuid": "` + hostID + `",
      "version": "5.9.2",
      "watcher": "3570"
    }`)
		if i != num-1 {
			b.WriteString(",")
		}
	}

	return b.String()
}

func (s *integrationTestSuite) TestHostHealth() {
	t := s.T()

	host, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		OsqueryHostID:   ptr.String(t.Name() + "hostid1"),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String(t.Name() + "nodekey1"),
		UUID:            t.Name() + "uuid1",
		Hostname:        t.Name() + "foo.local",
		PrimaryIP:       "192.168.1.1",
		PrimaryMac:      "30-65-EC-6F-C4-58",
		OSVersion:       "Mac OS X 10.14.6",
		Platform:        "darwin",
		CPUType:         "cpuType",
		TeamID:          nil,
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
	for _, item := range host.Software {
		if item.Name == "bar" {
			soft1 = item
			break
		}
	}

	cpes := []fleet.SoftwareCPE{{SoftwareID: soft1.ID, CPE: "somecpe"}}
	_, err = s.ds.UpsertSoftwareCPEs(context.Background(), cpes)
	require.NoError(t, err)

	// Reload software so that 'GeneratedCPEID is set.
	require.NoError(t, s.ds.LoadHostSoftware(context.Background(), host, false))
	soft1 = host.Software[0]
	for _, item := range host.Software {
		if item.Name == "bar" {
			soft1 = item
			break
		}
	}

	inserted, err := s.ds.InsertSoftwareVulnerability(
		context.Background(), fleet.SoftwareVulnerability{
			SoftwareID: soft1.ID,
			CVE:        "cve-123-123-132",
		}, fleet.NVDSource,
	)
	require.NoError(t, err)
	require.True(t, inserted)

	passingPolicy, err := s.ds.NewGlobalPolicy(context.Background(), nil, fleet.PolicyPayload{
		Name:       "passing_policy",
		Query:      "select 1",
		Resolution: "Run this command to fix it",
	})
	require.NoError(t, err)

	failingPolicy, err := s.ds.NewGlobalPolicy(context.Background(), nil, fleet.PolicyPayload{
		Name:       "failing_policy",
		Query:      "select 0",
		Resolution: "Run this command to fix it",
	})
	require.NoError(t, err)

	require.NoError(t, s.ds.RecordPolicyQueryExecutions(context.Background(), host, map[uint]*bool{failingPolicy.ID: ptr.Bool(false)}, time.Now(), false))
	require.NoError(t, s.ds.RecordPolicyQueryExecutions(context.Background(), host, map[uint]*bool{passingPolicy.ID: ptr.Bool(true)}, time.Now(), false))

	require.NoError(t, s.ds.SetOrUpdateHostDisksEncryption(context.Background(), host.ID, true))

	// Get host health
	hh := getHostHealthResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/health", host.ID), nil, http.StatusOK, &hh)
	assert.Equal(t, host.ID, hh.HostID)
	assert.NotNil(t, hh.HostHealth)
	assert.Equal(t, host.OSVersion, hh.HostHealth.OsVersion)
	assert.Len(t, hh.HostHealth.VulnerableSoftware, 1)
	assert.Equal(t, hh.HostHealth.VulnerableSoftware[0], fleet.HostHealthVulnerableSoftware{
		ID:      soft1.ID,
		Name:    soft1.Name,
		Version: soft1.Version,
	})
	assert.Equal(t, 1, hh.HostHealth.FailingPoliciesCount)
	assert.Nil(t, hh.HostHealth.FailingCriticalPoliciesCount)
	assert.Len(t, hh.HostHealth.FailingPolicies, 1)
	assert.Equal(t, hh.HostHealth.FailingPolicies[0], &fleet.HostHealthFailingPolicy{
		ID:         failingPolicy.ID,
		Name:       failingPolicy.Name,
		Resolution: failingPolicy.Resolution,
		Critical:   nil,
	})
	assert.True(t, *hh.HostHealth.DiskEncryptionEnabled)
	// Check that the TeamID didn't make it into the response
	assert.Nil(t, hh.HostHealth.TeamID)

	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/health", 0), nil, http.StatusNotFound, &hh)

	resp := getHostHealthResponse{}
	host1, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		OsqueryHostID:   ptr.String(t.Name() + "hostid2"),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String(t.Name() + "nodekey2"),
		UUID:            t.Name() + "uuid2",
		Hostname:        t.Name() + "foo2.local",
		PrimaryIP:       "192.168.2.2",
		PrimaryMac:      "32-62-E2-62-C2-52",
		OSVersion:       "Mac OS X 10.14.2",
		Platform:        "darwin",
		CPUType:         "cpuType",
	})
	require.NoError(t, err)
	require.NotNil(t, host1)

	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/health", host1.ID), nil, http.StatusOK, &resp)
	assert.Equal(t, host1.ID, resp.HostID)
	assert.NotNil(t, resp.HostHealth)
	assert.Equal(t, host1.OSVersion, resp.HostHealth.OsVersion)
	assert.Nil(t, resp.HostHealth.DiskEncryptionEnabled)
	assert.Empty(t, resp.HostHealth.VulnerableSoftware)
	assert.Empty(t, resp.HostHealth.FailingPolicies)
	assert.Nil(t, resp.HostHealth.TeamID)
}

func (s *integrationTestSuite) TestHostDeviceToken() {
	t := s.T()
	type response struct {
		Err string `json:"error"`
	}

	orbitHost := createOrbitEnrolledHost(t, "windows", "device_token", s.ds)

	// Write empty token
	body := setOrUpdateDeviceTokenRequest{
		OrbitNodeKey:    *orbitHost.OrbitNodeKey,
		DeviceAuthToken: "",
	}
	s.DoJSON("POST", "/api/fleet/orbit/device_token", body, http.StatusBadRequest, &response{})

	// Use illegal characters
	body = setOrUpdateDeviceTokenRequest{
		OrbitNodeKey:    *orbitHost.OrbitNodeKey,
		DeviceAuthToken: "../.",
	}
	s.DoJSON("POST", "/api/fleet/orbit/device_token", body, http.StatusBadRequest, &response{})

	// Write bad node key
	body = setOrUpdateDeviceTokenRequest{
		OrbitNodeKey:    "",
		DeviceAuthToken: "token",
	}
	s.DoJSON("POST", "/api/fleet/orbit/device_token", body, http.StatusUnauthorized, &response{})

	// Write a good token.
	body = setOrUpdateDeviceTokenRequest{
		OrbitNodeKey:    *orbitHost.OrbitNodeKey,
		DeviceAuthToken: "token",
	}
	s.DoJSON("POST", "/api/fleet/orbit/device_token", body, http.StatusOK, &response{})

	// Try to write the token again for a different host.
	// First write a valid token.
	orbitHost2 := createOrbitEnrolledHost(t, "darwin", "device_token2", s.ds)
	body = setOrUpdateDeviceTokenRequest{
		OrbitNodeKey:    *orbitHost2.OrbitNodeKey,
		DeviceAuthToken: "token2",
	}
	s.DoJSON("POST", "/api/fleet/orbit/device_token", body, http.StatusOK, &response{})

	// Now write a duplicate token, which will result in a conflict with the first host.
	body = setOrUpdateDeviceTokenRequest{
		OrbitNodeKey:    *orbitHost2.OrbitNodeKey,
		DeviceAuthToken: "token",
	}
	s.DoJSON("POST", "/api/fleet/orbit/device_token", body, http.StatusConflict, &response{})
}

func (s *integrationTestSuite) TestHostPastActivities() {
	t := s.T()
	ctx := context.Background()
	user := s.users["admin1@example.com"]
	getDetails := func(a *fleet.Activity) fleet.ActivityTypeRanScript {
		var details fleet.ActivityTypeRanScript
		err := json.Unmarshal([]byte(*a.Details), &details)
		require.NoError(t, err)

		return details
	}

	host := createOrbitEnrolledHost(t, "linux", "", s.ds)
	err := s.ds.MarkHostsSeen(ctx, []uint{host.ID}, time.Now())
	require.NoError(t, err)

	// create a valid script execution request
	savedScript, err := s.ds.NewScript(ctx, &fleet.Script{
		TeamID:         nil,
		Name:           "saved.sh",
		ScriptContents: "echo 'hello world'",
	})
	require.NoError(t, err)

	var runResp runScriptResponse
	s.DoJSON("POST", "/api/latest/fleet/scripts/run", fleet.HostScriptRequestPayload{HostID: host.ID, ScriptID: &savedScript.ID}, http.StatusAccepted, &runResp)
	require.Equal(t, host.ID, runResp.HostID)
	require.NotEmpty(t, runResp.ExecutionID)

	execID1 := runResp.ExecutionID

	result, err := s.ds.GetHostScriptExecutionResult(ctx, runResp.ExecutionID)
	require.NoError(t, err)
	require.Equal(t, host.ID, result.HostID)
	require.Equal(t, "echo 'hello world'", result.ScriptContents)
	require.Nil(t, result.ExitCode)

	var orbitPostScriptResp orbitPostScriptResultResponse
	s.DoJSON("POST", "/api/fleet/orbit/scripts/result",
		json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q, "execution_id": %q, "exit_code": 0, "output": "ok"}`, *host.OrbitNodeKey, result.ExecutionID)),
		http.StatusOK, &orbitPostScriptResp)

	var listResp listActivitiesResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities", host.ID), nil, http.StatusOK, &listResp)

	require.Len(t, listResp.Activities, 1)
	require.Equal(t, user.Email, *listResp.Activities[0].ActorEmail)
	require.Equal(t, user.Name, *listResp.Activities[0].ActorFullName)
	require.Equal(t, user.GravatarURL, *listResp.Activities[0].ActorGravatar)
	require.Equal(t, "ran_script", listResp.Activities[0].Type)
	d := getDetails(listResp.Activities[0])
	require.Equal(t, execID1, d.ScriptExecutionID)
	require.Equal(t, savedScript.Name, d.ScriptName)
	require.Equal(t, host.DisplayName(), d.HostDisplayName)
	require.Equal(t, host.ID, d.HostID)
	require.Equal(t, true, d.Async)

	// sleep to have the created_at timestamps differ
	time.Sleep(time.Second)

	// Execute another script in order to test query params
	s.DoJSON("POST", "/api/latest/fleet/scripts/run", fleet.HostScriptRequestPayload{HostID: host.ID, ScriptContents: "echo 'foobar'"}, http.StatusAccepted, &runResp)
	require.Equal(t, host.ID, runResp.HostID)
	require.NotEmpty(t, runResp.ExecutionID)

	execID2 := runResp.ExecutionID

	result, err = s.ds.GetHostScriptExecutionResult(ctx, runResp.ExecutionID)
	require.NoError(t, err)
	require.Equal(t, host.ID, result.HostID)
	require.Equal(t, "echo 'foobar'", result.ScriptContents)
	require.Nil(t, result.ExitCode)

	s.DoJSON("POST", "/api/fleet/orbit/scripts/result",
		json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q, "execution_id": %q, "exit_code": 0, "output": "ok"}`, *host.OrbitNodeKey, result.ExecutionID)),
		http.StatusOK, &orbitPostScriptResp)

	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities", host.ID), nil, http.StatusOK, &listResp, "page", "0", "per_page", "1")

	require.Len(t, listResp.Activities, 1)
	d = getDetails(listResp.Activities[0])

	require.Equal(t, execID2, d.ScriptExecutionID)

	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities", host.ID), nil, http.StatusOK, &listResp, "page", "1", "per_page", "1")

	require.Len(t, listResp.Activities, 1)
	d = getDetails(listResp.Activities[0])
	require.Equal(t, execID1, d.ScriptExecutionID)
}

func (s *integrationTestSuite) TestListHostUpcomingActivities() {
	t := s.T()
	ctx := context.Background()

	adminUser, err := s.ds.UserByEmail(ctx, "admin1@example.com")
	require.NoError(t, err)

	// there is already a datastore-layer test that verifies that correct values
	// are returned for users, saved scripts, etc. so this is more focused on
	// verifying that the service layer passes the proper options and the
	// rendering of the response.

	host1, err := s.ds.NewHost(ctx, &fleet.Host{
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

	// create script execution requests
	hsr, err := s.ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{HostID: host1.ID, ScriptContents: "A", SyncRequest: true})
	require.NoError(t, err)
	h1A := hsr.ExecutionID
	hsr, err = s.ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{HostID: host1.ID, ScriptContents: "B"})
	require.NoError(t, err)
	h1B := hsr.ExecutionID
	hsr, err = s.ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{HostID: host1.ID, ScriptContents: "C"})
	require.NoError(t, err)
	h1C := hsr.ExecutionID
	hsr, err = s.ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{HostID: host1.ID, ScriptContents: "D", SyncRequest: true})
	require.NoError(t, err)
	h1D := hsr.ExecutionID
	hsr, err = s.ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{HostID: host1.ID, ScriptContents: "E"})
	require.NoError(t, err)
	h1E := hsr.ExecutionID

	// create a software installation request
	tfr1, err := fleet.NewTempFileReader(strings.NewReader("echo"), t.TempDir)
	require.NoError(t, err)
	sw1, _, err := s.ds.MatchOrCreateSoftwareInstaller(ctx, &fleet.UploadSoftwareInstallerPayload{
		InstallScript: "install foo",
		InstallerFile: tfr1,
		StorageID:     uuid.NewString(),
		Filename:      "foo.pkg",
		Title:         "foo",
		Source:        "apps",
		Version:       "0.0.1",
		UserID:        adminUser.ID,
	})
	require.NoError(t, err)
	s1Meta, err := s.ds.GetSoftwareInstallerMetadataByID(ctx, sw1)
	require.NoError(t, err)
	h1Foo, err := s.ds.InsertSoftwareInstallRequest(ctx, host1.ID, s1Meta.InstallerID, false, nil)
	require.NoError(t, err)

	// force an order to the activities
	endTime := mysql.SetOrderedCreatedAtTimestamps(t, s.ds, time.Now(), "host_script_results", "execution_id", h1A, h1B)
	endTime = mysql.SetOrderedCreatedAtTimestamps(t, s.ds, endTime, "host_software_installs", "execution_id", h1Foo)
	mysql.SetOrderedCreatedAtTimestamps(t, s.ds, endTime, "host_script_results", "execution_id", h1C, h1D, h1E)

	// modify the timestamp h1A and h1B to simulate an script that has been
	// pending for a long time (h1A is a sync request, so it will be ignored for
	// upcoming activities)
	mysql.ExecAdhocSQL(t, s.ds, func(tx sqlx.ExtContext) error {
		_, err := tx.ExecContext(ctx, "UPDATE host_script_results SET created_at = ? WHERE execution_id IN (?, ?)", time.Now().Add(-24*time.Hour), h1A, h1B)
		return err
	})

	cases := []struct {
		queries   []string // alternate query name and value
		wantExecs []string
		wantMeta  *fleet.PaginationMetadata
	}{
		{
			wantExecs: []string{h1B, h1Foo, h1C, h1D, h1E},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: false, HasPreviousResults: false},
		},
		{
			queries:   []string{"per_page", "2"},
			wantExecs: []string{h1B, h1Foo},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: true, HasPreviousResults: false},
		},
		{
			queries:   []string{"per_page", "2", "page", "1"},
			wantExecs: []string{h1C, h1D},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: true, HasPreviousResults: true},
		},
		{
			queries:   []string{"per_page", "2", "page", "2"},
			wantExecs: []string{h1E},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: false, HasPreviousResults: true},
		},
		{
			queries:   []string{"per_page", "2", "page", "3"},
			wantExecs: nil,
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: false, HasPreviousResults: true},
		},
		{
			queries:   []string{"per_page", "3"},
			wantExecs: []string{h1B, h1Foo, h1C},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: true, HasPreviousResults: false},
		},
		{
			queries:   []string{"per_page", "3", "page", "1"},
			wantExecs: []string{h1D, h1E},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: false, HasPreviousResults: true},
		},
		{
			queries:   []string{"per_page", "3", "page", "2"},
			wantExecs: nil,
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: false, HasPreviousResults: true},
		},
	}
	for _, c := range cases {
		t.Run(fmt.Sprintf("%#v", c.queries), func(t *testing.T) {
			var listResp listHostUpcomingActivitiesResponse
			queryArgs := c.queries
			s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming", host1.ID), nil, http.StatusOK, &listResp, queryArgs...)

			require.Equal(t, uint(5), listResp.Count)
			require.Equal(t, len(c.wantExecs), len(listResp.Activities))
			require.Equal(t, c.wantMeta, listResp.Meta)

			var gotExecs []string
			if len(listResp.Activities) > 0 {
				gotExecs = make([]string, len(listResp.Activities))
				for i, a := range listResp.Activities {
					require.Zero(t, a.ID)
					require.NotEmpty(t, a.UUID)
					require.Contains(t, []string{
						fleet.ActivityTypeRanScript{}.ActivityName(),
						fleet.ActivityTypeInstalledSoftware{}.ActivityName(),
					}, a.Type)

					var details map[string]any
					require.NotNil(t, a.Details)
					require.NoError(t, json.Unmarshal(*a.Details, &details))
					switch a.Type {
					case fleet.ActivityTypeRanScript{}.ActivityName():
						gotExecs[i] = details["script_execution_id"].(string)
					case fleet.ActivityTypeInstalledSoftware{}.ActivityName():
						gotExecs[i] = details["install_uuid"].(string)
					}
				}
			}
			require.Equal(t, c.wantExecs, gotExecs)
		})
	}

	// Test with a host that has no upcoming activities
	host2, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   ptr.String(t.Name() + "2"),
		NodeKey:         ptr.String(t.Name() + "2"),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%sfoo2.local", t.Name()),
		Platform:        "darwin",
	})
	require.NoError(t, err)

	var listResp listHostUpcomingActivitiesResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming", host2.ID), nil, http.StatusOK, &listResp)
	require.Equal(t, uint(0), listResp.Count)
	require.Empty(t, listResp.Activities)
	require.Equal(t, &fleet.PaginationMetadata{HasNextResults: false, HasPreviousResults: false}, listResp.Meta)
}

func (s *integrationTestSuite) TestAddingRemovingManualLabels() {
	t := s.T()
	ctx := context.Background()

	team1, err := s.ds.NewTeam(ctx, &fleet.Team{
		Name: "team1",
	})
	require.NoError(t, err)

	newGlobalUserFunc := func(email string, globalRole string) *fleet.User {
		user := &fleet.User{
			Name:       email,
			Email:      email,
			GlobalRole: &globalRole,
		}
		err = user.SetPassword(test.GoodPassword, 10, 10)
		require.NoError(t, err)
		user, err = s.ds.NewUser(context.Background(), user)
		require.NoError(t, err)
		return user
	}
	newTeamUserFunc := func(email string, team *fleet.Team, teamRole string) *fleet.User {
		user := &fleet.User{
			Name:  email,
			Email: email,
			Teams: []fleet.UserTeam{
				{
					Team: *team,
					Role: teamRole,
				},
			},
		}
		err = user.SetPassword(test.GoodPassword, 10, 10)
		require.NoError(t, err)
		user, err = s.ds.NewUser(context.Background(), user)
		require.NoError(t, err)
		return user
	}
	globalObserver := newGlobalUserFunc("global.observer@example.com", fleet.RoleObserver)
	teamAdmin := newTeamUserFunc("team.admin@example.com", team1, fleet.RoleAdmin)
	teamObserver := newTeamUserFunc("team.observer@example.com", team1, fleet.RoleObserver)

	newHostFunc := func(name string, teamID *uint) *fleet.Host {
		host, err := s.ds.NewHost(ctx, &fleet.Host{
			NodeKey:  ptr.String(name),
			UUID:     name,
			Hostname: "foo.local." + name,
			TeamID:   teamID,
		})
		require.NoError(t, err)
		require.NotNil(t, host)
		return host
	}
	host1 := newHostFunc("host1", nil)
	host2 := newHostFunc("host2", nil)
	teamHost2 := newHostFunc("teamHost2", &team1.ID)

	ls, err := s.ds.LabelIDsByName(ctx, []string{"All Hosts"})
	require.NoError(t, err)
	require.Len(t, ls, 1)
	allHostsLabelID, ok := ls["All Hosts"]
	require.True(t, ok)
	require.NotZero(t, allHostsLabelID)

	dynamicLabel1, err := s.ds.NewLabel(ctx, &fleet.Label{
		Name:                "dynamicLabel1",
		Query:               "SELECT 1;",
		LabelMembershipType: fleet.LabelMembershipTypeDynamic,
	})
	require.NoError(t, err)
	manualLabel1, err := s.ds.NewLabel(ctx, &fleet.Label{
		Name:                "manualLabel1",
		Query:               "SELECT 2;",
		LabelMembershipType: fleet.LabelMembershipTypeManual,
	})
	require.NoError(t, err)
	manualLabel2, err := s.ds.NewLabel(ctx, &fleet.Label{
		Name:                "manualLabel2",
		Query:               "SELECT 3;",
		LabelMembershipType: fleet.LabelMembershipTypeManual,
	})
	require.NoError(t, err)

	err = s.ds.RecordLabelQueryExecutions(context.Background(), host1, map[uint]*bool{allHostsLabelID: ptr.Bool(true)}, time.Now(), false)
	require.NoError(t, err)

	getHostLabels := func(host *fleet.Host) []string {
		var hostResp getHostResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &hostResp)
		var labels []string
		for _, label := range hostResp.Host.Labels {
			labels = append(labels, label.Name)
		}
		return labels
	}

	hostLabels1 := getHostLabels(host1)
	require.Len(t, hostLabels1, 1)
	require.Equal(t, "All Hosts", hostLabels1[0])

	// No labels or empty labels is a no-op.
	var addLabelsToHostResp addLabelsToHostResponse
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", host1.ID),
		json.RawMessage(`{}`), http.StatusOK, &addLabelsToHostResp,
	)
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", host1.ID), addLabelsToHostRequest{
		Labels: []string{},
	}, http.StatusOK, &addLabelsToHostResp)
	var removeLabelsFromHostResp removeLabelsFromHostResponse
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", host1.ID), removeLabelsFromHostRequest{
		Labels: []string{},
	}, http.StatusOK, &removeLabelsFromHostResp)
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", host1.ID), addLabelsToHostRequest{
		Labels: []string{""},
	}, http.StatusOK, &addLabelsToHostResp)
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", host1.ID), addLabelsToHostRequest{
		Labels: []string{"", ""},
	}, http.StatusOK, &addLabelsToHostResp)

	// A dynamic buitin label should fail to be added.
	res := s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", host1.ID), addLabelsToHostRequest{
		Labels: []string{"All Hosts"},
	}, http.StatusBadRequest)
	errMsg := extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Couldn't add labels. Labels are dynamic: \"All Hosts\". Dynamic labels can not be assigned to hosts manually.")
	// An inexistent label should fail to be added.
	res = s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", host1.ID), addLabelsToHostRequest{
		Labels: []string{"manualLabel2", "does not exist"},
	}, http.StatusBadRequest)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Couldn't add labels. Labels not found: \"does not exist\". All labels must exist.")
	// Multiple inexistent labels should fail to be added.
	res = s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", host1.ID), addLabelsToHostRequest{
		Labels: []string{"manualLabel2", "does not exist", "does not exist 2"},
	}, http.StatusBadRequest)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Couldn't add labels. Labels not found: \"does not exist\", \"does not exist 2\". All labels must exist.")
	// A dynamic non-builtin label should fail to be added.
	res = s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", host1.ID), addLabelsToHostRequest{
		Labels: []string{dynamicLabel1.Name},
	}, http.StatusBadRequest)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Couldn't add labels. Labels are dynamic: \"dynamicLabel1\". Dynamic labels can not be assigned to hosts manually.")
	// Multiple dynamic labels should fail to be added.
	res = s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", host1.ID), addLabelsToHostRequest{
		Labels: []string{"All Hosts", dynamicLabel1.Name},
	}, http.StatusBadRequest)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Couldn't add labels. Labels are dynamic: \"All Hosts\", \"dynamicLabel1\". Dynamic labels can not be assigned to hosts manually.")

	// A dynamic builtin label should fail to be deleted.
	res = s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", host1.ID), removeLabelsFromHostRequest{
		Labels: []string{"All Hosts"},
	}, http.StatusBadRequest)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Couldn't remove labels. Labels are dynamic: \"All Hosts\". Dynamic labels can not be assigned to hosts manually.")
	// An inexistent label should fail to be deleted.
	res = s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", host1.ID), removeLabelsFromHostRequest{
		Labels: []string{manualLabel2.Name, "does not exist"},
	}, http.StatusBadRequest)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Couldn't remove labels. Labels not found: \"does not exist\". All labels must exist.")
	// Multiple inexistent labels should fail to be deleted.
	res = s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", host1.ID), removeLabelsFromHostRequest{
		Labels: []string{manualLabel2.Name, "does not exist", "does not exist 2"},
	}, http.StatusBadRequest)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Couldn't remove labels. Labels not found: \"does not exist\", \"does not exist 2\". All labels must exist.")
	// Multiple dynamic labels should fail to be deleted.
	res = s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", host1.ID), removeLabelsFromHostRequest{
		Labels: []string{manualLabel2.Name, dynamicLabel1.Name, "All Hosts"},
	}, http.StatusBadRequest)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Couldn't remove labels. Labels are dynamic: \"All Hosts\", \"dynamicLabel1\". Dynamic labels can not be assigned to hosts manually.")

	// Add two manual labels to a host.
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", host1.ID), addLabelsToHostRequest{
		Labels: []string{manualLabel1.Name, manualLabel2.Name},
	}, http.StatusOK, &addLabelsToHostResp)
	// Add the same manual labels to a host should succeed.
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", host1.ID), addLabelsToHostRequest{
		Labels: []string{manualLabel1.Name, manualLabel2.Name},
	}, http.StatusOK, &addLabelsToHostResp)

	hostLabels1 = getHostLabels(host1)
	require.Len(t, hostLabels1, 3)
	require.Equal(t, "All Hosts", hostLabels1[0])
	require.Equal(t, manualLabel1.Name, hostLabels1[1])
	require.Equal(t, manualLabel2.Name, hostLabels1[2])
	hostLabels2 := getHostLabels(host2)
	require.Empty(t, hostLabels2)

	// Remove the two manual labels from the host.
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", host1.ID), removeLabelsFromHostRequest{
		Labels: []string{manualLabel1.Name, manualLabel2.Name},
	}, http.StatusOK, &removeLabelsFromHostResp)
	// Remove the same manual labels from the host again.
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", host1.ID), removeLabelsFromHostRequest{
		Labels: []string{manualLabel1.Name, manualLabel2.Name},
	}, http.StatusOK, &removeLabelsFromHostResp)

	hostLabels1 = getHostLabels(host1)
	require.Len(t, hostLabels1, 1)
	require.Equal(t, "All Hosts", hostLabels1[0])

	// Add same label, should deduplicate.
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", host1.ID), addLabelsToHostRequest{
		Labels: []string{manualLabel1.Name, manualLabel1.Name},
	}, http.StatusOK, &addLabelsToHostResp)

	hostLabels1 = getHostLabels(host1)
	require.Len(t, hostLabels1, 2)
	require.Equal(t, "All Hosts", hostLabels1[0])
	require.Equal(t, manualLabel1.Name, hostLabels1[1])

	// Adding an already added label should work.
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", host1.ID), addLabelsToHostRequest{
		Labels: []string{manualLabel1.Name, manualLabel2.Name},
	}, http.StatusOK, &addLabelsToHostResp)

	hostLabels1 = getHostLabels(host1)
	require.Len(t, hostLabels1, 3)
	require.Equal(t, "All Hosts", hostLabels1[0])
	require.Equal(t, manualLabel1.Name, hostLabels1[1])
	require.Equal(t, manualLabel2.Name, hostLabels1[2])

	// Delete same label, should deduplicate.
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", host1.ID), removeLabelsFromHostRequest{
		Labels: []string{manualLabel1.Name, manualLabel1.Name},
	}, http.StatusOK, &removeLabelsFromHostResp)

	// Deleting a non-member label (manualLabel1) should work.
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", host1.ID), removeLabelsFromHostRequest{
		Labels: []string{manualLabel1.Name, manualLabel2.Name},
	}, http.StatusOK, &removeLabelsFromHostResp)

	hostLabels1 = getHostLabels(host1)
	require.Len(t, hostLabels1, 1)
	require.Equal(t, "All Hosts", hostLabels1[0])

	// Add to non-existent host
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", 999), addLabelsToHostRequest{
		Labels: []string{manualLabel1.Name, manualLabel2.Name},
	}, http.StatusNotFound, &addLabelsToHostResp)
	// Delete from non-existent host
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", 999), removeLabelsFromHostRequest{
		Labels: []string{manualLabel1.Name, manualLabel2.Name},
	}, http.StatusNotFound, &removeLabelsFromHostResp)

	// Add labels to team host.
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", teamHost2.ID), addLabelsToHostRequest{
		Labels: []string{manualLabel1.Name},
	}, http.StatusOK, &addLabelsToHostResp)

	// A global observer should not be allowed to add/remove a label.
	oldToken := s.token
	s.token = s.getTestToken(globalObserver.Email, test.GoodPassword)
	t.Cleanup(func() {
		s.token = oldToken
	})
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", teamHost2.ID), addLabelsToHostRequest{
		Labels: []string{manualLabel1.Name},
	}, http.StatusForbidden, &addLabelsToHostResp)
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", teamHost2.ID), removeLabelsFromHostRequest{
		Labels: []string{manualLabel1.Name},
	}, http.StatusForbidden, &removeLabelsFromHostResp)

	// A team observer should not be allowed to add/remove a label.
	s.token = s.getTestToken(teamObserver.Email, test.GoodPassword)
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", teamHost2.ID), addLabelsToHostRequest{
		Labels: []string{manualLabel1.Name},
	}, http.StatusForbidden, &addLabelsToHostResp)
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", teamHost2.ID), removeLabelsFromHostRequest{
		Labels: []string{manualLabel1.Name},
	}, http.StatusForbidden, &removeLabelsFromHostResp)

	// A team admin should not be allowed to add/remove a label for a global host.
	s.token = s.getTestToken(teamAdmin.Email, test.GoodPassword)
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", host1.ID), addLabelsToHostRequest{
		Labels: []string{manualLabel1.Name},
	}, http.StatusForbidden, &addLabelsToHostResp)
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", host1.ID), removeLabelsFromHostRequest{
		Labels: []string{manualLabel1.Name},
	}, http.StatusForbidden, &removeLabelsFromHostResp)

	// A team admin should be allowed to add/remove a label for a team host.
	s.token = s.getTestToken(teamAdmin.Email, test.GoodPassword)
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", teamHost2.ID), addLabelsToHostRequest{
		Labels: []string{manualLabel1.Name},
	}, http.StatusOK, &addLabelsToHostResp)
	teamHost2Labels := getHostLabels(teamHost2)
	require.Len(t, teamHost2Labels, 1)
	require.Equal(t, manualLabel1.Name, teamHost2Labels[0])
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", teamHost2.ID), removeLabelsFromHostRequest{
		Labels: []string{manualLabel1.Name},
	}, http.StatusOK, &removeLabelsFromHostResp)
	teamHost2Labels = getHostLabels(teamHost2)
	require.Empty(t, teamHost2Labels)
}

func (s *integrationTestSuite) TestDebugDB() {
	t := s.T()
	var response map[string]string
	s.DoJSON("GET", "/debug/db/locks", nil, http.StatusOK, &response)
	assert.Empty(t, response)

	var responseString string
	s.DoJSON("GET", "/debug/db/innodb-status", nil, http.StatusOK, &responseString)
	assert.Contains(t, responseString, "INNODB MONITOR OUTPUT")
}

func (s *integrationTestSuite) TestAutofillPolicies() {
	t := s.T()
	startMockServer := func(t *testing.T) string {
		// create a test http server
		srv := httptest.NewServer(
			http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					if r.Method != "POST" {
						w.WriteHeader(http.StatusMethodNotAllowed)
						return
					}
					switch r.URL.Path {
					case "/ok":
						var body map[string]interface{}
						err := json.NewDecoder(r.Body).Decode(&body)
						if err != nil {
							t.Log(err)
							w.WriteHeader(http.StatusBadRequest)
							return
						}
						_, _ = w.Write([]byte(`{"risks":"description", "whatWillProbablyHappenDuringMaintenance":"resolution"}`))
					case "/error":
						w.WriteHeader(http.StatusTeapot)
						_, _ = w.Write([]byte(`{}`))
					case "/badBody":
						_, _ = w.Write([]byte(`{bad json}`))
					case "/timeout":
						time.Sleep(2 * time.Second)
						_, _ = w.Write([]byte(`{"risks":"description", "whatWillProbablyHappenDuringMaintenance":"resolution"}`))
					default:
						w.WriteHeader(http.StatusNotFound)
					}
				},
			),
		)
		t.Cleanup(srv.Close)
		return srv.URL
	}
	mockUrl := startMockServer(t)
	originalUrl := getHumanInterpretationFromOsquerySqlUrl
	originalTimeout := getHumanInterpretationFromOsquerySqlTimeout
	t.Cleanup(
		func() {
			getHumanInterpretationFromOsquerySqlUrl = originalUrl
			getHumanInterpretationFromOsquerySqlTimeout = originalTimeout
		},
	)

	req := autofillPoliciesRequest{
		SQL: "  ", // empty
	}
	getHumanInterpretationFromOsquerySqlUrl = mockUrl + "/ok"
	// empty sql
	resp := s.Do("POST", "/api/latest/fleet/autofill/policy", req, http.StatusBadRequest)
	assertBodyContains(t, resp, "cannot be empty")

	// good request
	req.SQL = "select 1"
	var res autofillPoliciesResponse
	s.DoJSON("POST", "/api/latest/fleet/autofill/policy", req, http.StatusOK, &res)
	assert.Equal(t, "description", res.Description)
	assert.Equal(t, "resolution", res.Resolution)

	// good request with weird characters
	req.SQL = `select * from " with ' and "" \"`
	res = autofillPoliciesResponse{}
	s.DoJSON("POST", "/api/latest/fleet/autofill/policy", req, http.StatusOK, &res)
	assert.Equal(t, "description", res.Description)
	assert.Equal(t, "resolution", res.Resolution)

	getHumanInterpretationFromOsquerySqlUrl = mockUrl + "/error"
	resp = s.Do("POST", "/api/latest/fleet/autofill/policy", req, http.StatusUnprocessableEntity)
	assertBodyContains(t, resp, "error from human interpretation of osquery sql")

	getHumanInterpretationFromOsquerySqlUrl = mockUrl + "/badBody"
	resp = s.Do("POST", "/api/latest/fleet/autofill/policy", req, http.StatusUnprocessableEntity)
	assertBodyContains(t, resp, "error unmarshaling response body from human interpretation of osquery sql")

	getHumanInterpretationFromOsquerySqlUrl = mockUrl + "/timeout"
	getHumanInterpretationFromOsquerySqlTimeout = 1 * time.Millisecond
	resp = s.Do("POST", "/api/latest/fleet/autofill/policy", req, http.StatusUnprocessableEntity)
	assertBodyContains(t, resp, "error sending request to get human interpretation from osquery sql")

	// disable AI features
	appConfigSpec := map[string]map[string]bool{
		"server_settings": {"ai_features_disabled": true},
	}
	s.Do("PATCH", "/api/latest/fleet/config", appConfigSpec, http.StatusOK)
	resp = s.Do("POST", "/api/latest/fleet/autofill/policy", req, http.StatusBadRequest)
	assertBodyContains(t, resp, "AI features are disabled")
}

func (s *integrationTestSuite) TestHostWithNoPoliciesClearsPolicyCounts() {
	t := s.T()
	ctx := context.Background()

	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "Zoobar"})
	require.NoError(t, err)

	host, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now(),
		NodeKey:         ptr.String("foobar"),
		UUID:            "foobar",
		Hostname:        "com.foobar.local",
		Platform:        "linux",
		TeamID:          &team.ID,
	})
	require.NoError(t, err)

	policy, err := s.ds.NewTeamPolicy(ctx, team.ID, nil, fleet.PolicyPayload{
		Name:  "Barfoo",
		Query: "SELECT 1;",
	})
	require.NoError(t, err)

	distributedWriteResp := submitDistributedQueryResultsResponse{}
	s.DoJSON("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host,
		map[uint]*bool{
			policy.ID: ptr.Bool(false),
		},
	), http.StatusOK, &distributedWriteResp)

	listHostsResp := listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listHostsResp)
	require.Len(t, listHostsResp.Hosts, 1)
	require.Equal(t, uint64(1), listHostsResp.Hosts[0].FailingPoliciesCount)

	_, err = s.ds.DeleteTeamPolicies(ctx, team.ID, []uint{policy.ID})
	require.NoError(t, err)

	distributedWriteResp = submitDistributedQueryResultsResponse{}
	results := make(map[string]json.RawMessage)
	results[hostNoPoliciesWildcard] = json.RawMessage("{\"1\": \"1\"}")
	statuses := make(map[string]interface{})
	statuses[hostNoPoliciesWildcard] = 0
	s.DoJSON("POST", "/api/osquery/distributed/write", submitDistributedQueryResultsRequestShim{
		NodeKey:  *host.NodeKey,
		Results:  results,
		Statuses: statuses,
		Stats:    map[string]*fleet.Stats{},
	}, http.StatusOK, &distributedWriteResp)

	listHostsResp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listHostsResp)
	require.Len(t, listHostsResp.Hosts, 1)
	require.Equal(t, uint64(0), listHostsResp.Hosts[0].FailingPoliciesCount)
}

func (s *integrationTestSuite) TestHostSoftwareWithTeamIdentifier() {
	t := s.T()
	ctx := context.Background()

	host, err := s.ds.NewHost(ctx, &fleet.Host{
		NodeKey:       ptr.String(t.Name()),
		OsqueryHostID: ptr.String(t.Name()),
		UUID:          t.Name(),
		Hostname:      t.Name() + "foo.local",
		Platform:      "darwin",
	})
	require.NoError(t, err)

	safariApp := fleet.Software{
		Name:             "Safari.app",
		BundleIdentifier: "com.apple.safari",
		Version:          "18.1",
		Source:           "apps",
	}
	googleChromeApp := fleet.Software{
		Name:             "Google Chrome.app",
		BundleIdentifier: "com.google.Chrome",
		Version:          "130.0.6723.117",
		Source:           "apps",
	}
	ghCli := fleet.Software{
		Name:   "gh",
		Source: "homebrew_packages",
	}

	// Update the host's software.
	software := []fleet.Software{
		safariApp, googleChromeApp, ghCli,
	}
	hostSoftware, err := s.ds.UpdateHostSoftware(context.Background(), host.ID, software)
	require.NoError(t, err)
	require.Len(t, hostSoftware.CurrInstalled(), 3)

	// Update the host's software installed paths for the software above.
	// Google Chrome.app will have two installed paths one with team identifier set
	// the other one set to empty.
	swPaths := map[string]struct{}{}
	for _, s := range software {
		pathItems := [][2]string{{fmt.Sprintf("/some/path/%s", s.Name), ""}}
		if s.Name == "Google Chrome.app" {
			pathItems = [][2]string{
				{fmt.Sprintf("/some/path/%s", s.Name), "EQHXZ8M8AV"},
				{fmt.Sprintf("/some/other/path/%s", s.Name), ""},
			}
		}
		for _, pathItem := range pathItems {
			path := pathItem[0]
			teamIdentifier := pathItem[1]
			key := fmt.Sprintf(
				"%s%s%s%s%s",
				path, fleet.SoftwareFieldSeparator, teamIdentifier, fleet.SoftwareFieldSeparator, s.ToUniqueStr(),
			)
			swPaths[key] = struct{}{}
		}
	}
	err = s.ds.UpdateHostSoftwareInstalledPaths(ctx, host.ID, swPaths, hostSoftware)
	require.NoError(t, err)

	hostsCountTs := time.Now().UTC()
	err = s.ds.SyncHostsSoftware(context.Background(), hostsCountTs)
	require.NoError(t, err)

	getHostSoftwareResp := getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host.ID),
		nil, http.StatusOK, &getHostSoftwareResp,
		"per_page", "5", "page", "0", "order_key", "name", "order_direction", "desc",
	)
	require.Len(t, getHostSoftwareResp.Software, 3)
	require.Equal(t, "Safari.app", getHostSoftwareResp.Software[0].Name)
	require.Len(t, getHostSoftwareResp.Software[0].InstalledVersions, 1)
	require.Len(t, getHostSoftwareResp.Software[0].InstalledVersions[0].InstalledPaths, 1)
	require.Equal(t, "/some/path/Safari.app", getHostSoftwareResp.Software[0].InstalledVersions[0].InstalledPaths[0])
	require.Len(t, getHostSoftwareResp.Software[0].InstalledVersions[0].SignatureInformation, 1)
	require.Equal(t, "/some/path/Safari.app", getHostSoftwareResp.Software[0].InstalledVersions[0].SignatureInformation[0].InstalledPath)
	require.Empty(t, getHostSoftwareResp.Software[0].InstalledVersions[0].SignatureInformation[0].TeamIdentifier)

	require.Equal(t, "Google Chrome.app", getHostSoftwareResp.Software[1].Name)
	require.Len(t, getHostSoftwareResp.Software[1].InstalledVersions, 1)
	require.Len(t, getHostSoftwareResp.Software[1].InstalledVersions[0].InstalledPaths, 2)
	sort.Slice(getHostSoftwareResp.Software[1].InstalledVersions[0].InstalledPaths, func(i, j int) bool {
		return getHostSoftwareResp.Software[1].InstalledVersions[0].InstalledPaths[i] < getHostSoftwareResp.Software[1].InstalledVersions[0].InstalledPaths[j]
	})
	require.Equal(t, "/some/other/path/Google Chrome.app", getHostSoftwareResp.Software[1].InstalledVersions[0].InstalledPaths[0])
	require.Equal(t, "/some/path/Google Chrome.app", getHostSoftwareResp.Software[1].InstalledVersions[0].InstalledPaths[1])
	require.Len(t, getHostSoftwareResp.Software[1].InstalledVersions[0].SignatureInformation, 2)
	sort.Slice(getHostSoftwareResp.Software[1].InstalledVersions[0].SignatureInformation, func(i, j int) bool {
		return getHostSoftwareResp.Software[1].InstalledVersions[0].SignatureInformation[i].InstalledPath < getHostSoftwareResp.Software[1].InstalledVersions[0].SignatureInformation[j].InstalledPath
	})
	require.Equal(t, "/some/other/path/Google Chrome.app", getHostSoftwareResp.Software[1].InstalledVersions[0].SignatureInformation[0].InstalledPath)
	require.Equal(t, "", getHostSoftwareResp.Software[1].InstalledVersions[0].SignatureInformation[0].TeamIdentifier)
	require.Equal(t, "/some/path/Google Chrome.app", getHostSoftwareResp.Software[1].InstalledVersions[0].SignatureInformation[1].InstalledPath)
	require.Equal(t, "EQHXZ8M8AV", getHostSoftwareResp.Software[1].InstalledVersions[0].SignatureInformation[1].TeamIdentifier)

	require.Equal(t, "gh", getHostSoftwareResp.Software[2].Name)
	require.Len(t, getHostSoftwareResp.Software[2].InstalledVersions, 1)
	require.Len(t, getHostSoftwareResp.Software[2].InstalledVersions[0].InstalledPaths, 1)
	require.Equal(t, "/some/path/gh", getHostSoftwareResp.Software[2].InstalledVersions[0].InstalledPaths[0])
	require.Nil(t, getHostSoftwareResp.Software[2].InstalledVersions[0].SignatureInformation)
}

func (s *integrationTestSuite) TestSecretVariables() {
	t := s.T()
	ctx := context.Background()

	// Create the global GitOps user we'll use in tests.
	u := &fleet.User{
		Name:       "GitOps",
		Email:      "gitops1@example.com",
		GlobalRole: ptr.String(fleet.RoleGitOps),
	}
	require.NoError(t, u.SetPassword(test.GoodPassword, 10, 10))
	_, err := s.ds.NewUser(ctx, u)
	require.NoError(t, err)
	s.setTokenForTest(t, "gitops1@example.com", test.GoodPassword)

	// Empty request
	req := secretVariablesRequest{}
	var resp secretVariablesResponse
	s.DoJSON("PUT", "/api/latest/fleet/spec/secret_variables", req, http.StatusOK, &resp)

	// Secret variable name too long
	req = secretVariablesRequest{
		SecretVariables: []fleet.SecretVariable{
			{
				Name:  strings.Repeat("a", 256),
				Value: "value",
			},
		},
	}
	httpResp := s.Do("PUT", "/api/latest/fleet/spec/secret_variables", req, http.StatusUnprocessableEntity)
	assertBodyContains(t, httpResp, "secret variable name is too long")

	// Secret variable name empty
	req = secretVariablesRequest{
		SecretVariables: []fleet.SecretVariable{
			{
				Name:  "  ",
				Value: "value",
			},
		},
	}
	httpResp = s.Do("PUT", "/api/latest/fleet/spec/secret_variables", req, http.StatusUnprocessableEntity)
	assertBodyContains(t, httpResp, "secret variable name cannot be empty")

	validName := strings.Repeat("g", 255)
	req = secretVariablesRequest{
		SecretVariables: []fleet.SecretVariable{
			{
				Name:  "FLEET_SECRET_" + validName,
				Value: "value",
			},
		},
	}
	s.DoJSON("PUT", "/api/latest/fleet/spec/secret_variables", req, http.StatusOK, &resp)

	secrets, err := s.ds.GetSecretVariables(ctx, []string{validName})
	require.NoError(t, err)
	require.Len(t, secrets, 1)
	assert.Equal(t, "value", secrets[0].Value)
}
