package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/datastore/redis/redistest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/live_query/live_query_mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/go-kit/log"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestIntegrationsEnterprise(t *testing.T) {
	testingSuite := new(integrationEnterpriseTestSuite)
	testingSuite.s = &testingSuite.Suite
	suite.Run(t, testingSuite)
}

type integrationEnterpriseTestSuite struct {
	withServer
	suite.Suite
	redisPool fleet.RedisPool

	lq *live_query_mock.MockLiveQuery
}

func (s *integrationEnterpriseTestSuite) SetupSuite() {
	s.withDS.SetupSuite("integrationEnterpriseTestSuite")

	s.redisPool = redistest.SetupRedis(s.T(), "integration_enterprise", false, false, false)
	s.lq = live_query_mock.New(s.T())
	config := TestServerOpts{
		License: &fleet.LicenseInfo{
			Tier: fleet.TierPremium,
		},
		Pool:   s.redisPool,
		Lq:     s.lq,
		Logger: log.NewLogfmtLogger(os.Stdout),
	}
	users, server := RunServerForTestsWithDS(s.T(), s.ds, &config)
	s.server = server
	s.users = users
	s.token = s.getTestAdminToken()
	s.cachedTokens = make(map[string]string)
}

func (s *integrationEnterpriseTestSuite) TearDownTest() {
	// reset the mock
	s.lq.Mock = mock.Mock{}
	s.withServer.commonTearDownTest(s.T())
}

func (s *integrationEnterpriseTestSuite) TestTeamSpecs() {
	t := s.T()

	// create a team through the service so it initializes the agent ops
	teamName := t.Name() + "team1"
	team := &fleet.Team{
		Name:        teamName,
		Description: "desc team1",
	}

	s.Do("POST", "/api/latest/fleet/teams", team, http.StatusOK)

	// updates a team, no secret is provided so it will keep the one generated
	// automatically when the team was created.
	agentOpts := json.RawMessage(`{"config": {"views": {"foo": "bar"}}, "overrides": {"platforms": {"darwin": {"views": {"bar": "qux"}}}}}`)
	features := json.RawMessage(`{
    "enable_host_users": false,
    "enable_software_inventory": false,
    "additional_queries": {"foo": "bar"}
  }`)
	// must not use applyTeamSpecsRequest and marshal it as JSON, as it will set
	// all keys to their zerovalue, and some are only valid with mdm enabled.
	teamSpecs := map[string]any{
		"specs": []any{
			map[string]any{
				"name":          teamName,
				"agent_options": agentOpts,
				"features":      &features,
				"mdm": map[string]any{
					"macos_updates": map[string]any{
						"minimum_version": "10.15.0",
						"deadline":        "2021-01-01",
					},
				},
			},
		},
	}
	var applyResp applyTeamSpecsResponse
	s.DoJSON("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK, &applyResp)
	require.Len(t, applyResp.TeamIDsByName, 1)

	team, err := s.ds.TeamByName(context.Background(), teamName)
	require.NoError(t, err)
	require.Equal(t, applyResp.TeamIDsByName[teamName], team.ID)
	assert.Len(t, team.Secrets, 1)
	require.JSONEq(t, string(agentOpts), string(*team.Config.AgentOptions))
	require.Equal(t, fleet.Features{
		EnableHostUsers:         false,
		EnableSoftwareInventory: false,
		AdditionalQueries:       ptr.RawMessage(json.RawMessage(`{"foo": "bar"}`)),
	}, team.Config.Features)
	require.Equal(t, fleet.TeamMDM{
		MacOSUpdates: fleet.MacOSUpdates{
			MinimumVersion: optjson.SetString("10.15.0"),
			Deadline:       optjson.SetString("2021-01-01"),
		},
		MacOSSetup: fleet.MacOSSetup{
			// because the MacOSSetup was marshalled to JSON to be saved in the DB,
			// it did get marshalled, and then when unmarshalled it was set (but
			// null).
			MacOSSetupAssistant: optjson.String{Set: true},
			BootstrapPackage:    optjson.String{Set: true},
		},
	}, team.Config.MDM)

	// an activity was created for team spec applied
	s.lastActivityMatches(fleet.ActivityTypeAppliedSpecTeam{}.ActivityName(), fmt.Sprintf(`{"teams": [{"id": %d, "name": %q}]}`, team.ID, team.Name), 0)

	// dry-run with invalid agent options
	agentOpts = json.RawMessage(`{"config": {"nope": 1}}`)
	teamSpecs = map[string]any{
		"specs": []any{
			map[string]any{
				"name":          teamName,
				"agent_options": agentOpts,
			},
		},
	}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusBadRequest, "dry_run", "true")

	// dry-run with empty body
	res := s.DoRaw("POST", "/api/latest/fleet/spec/teams", nil, http.StatusBadRequest, "force", "true")
	errBody, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	require.Contains(t, string(errBody), `"Expected JSON Body"`)

	// dry-run with invalid top-level key
	s.Do("POST", "/api/latest/fleet/spec/teams", json.RawMessage(`{
		"specs": [
			{"name": "team_name_1", "unknown_key": true}
		]
	}`), http.StatusBadRequest, "dry_run", "true")

	team, err = s.ds.TeamByName(context.Background(), teamName)
	require.NoError(t, err)
	require.Contains(t, string(*team.Config.AgentOptions), `"foo": "bar"`) // unchanged

	// dry-run with valid agent options and custom macos settings
	agentOpts = json.RawMessage(`{"config": {"views": {"foo": "qux"}}}`)
	teamSpecs = map[string]any{
		"specs": []any{
			map[string]any{
				"name":          teamName,
				"agent_options": agentOpts,
				"mdm": map[string]any{
					"macos_settings": map[string]any{
						"custom_settings": []string{"foo", "bar"},
					},
				},
			},
		},
	}
	res = s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusUnprocessableEntity, "dry_run", "true")
	errMsg := extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Couldn't update macos_settings because MDM features aren't turned on in Fleet.")

	// dry-run with macos disk encryption set to false, no error
	teamSpecs = map[string]any{
		"specs": []any{
			map[string]any{
				"name": teamName,
				"mdm": map[string]any{
					"macos_settings": map[string]any{
						"enable_disk_encryption": false,
					},
				},
			},
		},
	}
	applyResp = applyTeamSpecsResponse{}
	s.DoJSON("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK, &applyResp, "dry_run", "true")
	// dry-run never returns id to name mappings as it may not have them
	require.Empty(t, applyResp.TeamIDsByName)

	// dry-run with macos disk encryption set to true
	teamSpecs = map[string]any{
		"specs": []any{
			map[string]any{
				"name": teamName,
				"mdm": map[string]any{
					"macos_settings": map[string]any{
						"enable_disk_encryption": true,
					},
				},
			},
		},
	}
	res = s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusUnprocessableEntity, "dry_run", "true")
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Couldn't update macos_settings because MDM features aren't turned on in Fleet.")

	// dry-run with valid agent options only
	agentOpts = json.RawMessage(`{"config": {"views": {"foo": "qux"}}}`)
	teamSpecs = map[string]any{
		"specs": []any{
			map[string]any{
				"name":          teamName,
				"agent_options": agentOpts,
			},
		},
	}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK, "dry_run", "true")

	team, err = s.ds.TeamByName(context.Background(), teamName)
	require.NoError(t, err)
	require.Contains(t, string(*team.Config.AgentOptions), `"foo": "bar"`) // unchanged
	require.Empty(t, team.Config.MDM.MacOSSettings.CustomSettings)         // unchanged
	require.False(t, team.Config.MDM.MacOSSettings.EnableDiskEncryption)   // unchanged

	// apply without agent options specified
	teamSpecs = map[string]any{
		"specs": []any{
			map[string]any{
				"name": teamName,
			},
		},
	}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK)

	// agent options are unchanged, not cleared
	team, err = s.ds.TeamByName(context.Background(), teamName)
	require.NoError(t, err)
	require.Contains(t, string(*team.Config.AgentOptions), `"foo": "bar"`) // unchanged

	// apply with agent options specified but null
	teamSpecs = map[string]any{
		"specs": []any{
			map[string]any{
				"name":          teamName,
				"agent_options": json.RawMessage(`null`),
			},
		},
	}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK)

	// agent options are cleared
	team, err = s.ds.TeamByName(context.Background(), teamName)
	require.NoError(t, err)
	require.Nil(t, team.Config.AgentOptions)

	// force with invalid agent options
	agentOpts = json.RawMessage(`{"config": {"foo": "qux"}}`)
	teamSpecs = map[string]any{
		"specs": []any{
			map[string]any{
				"name":          teamName,
				"agent_options": agentOpts,
			},
		},
	}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK, "force", "true")

	team, err = s.ds.TeamByName(context.Background(), teamName)
	require.NoError(t, err)
	require.Contains(t, string(*team.Config.AgentOptions), `"foo": "qux"`)

	// force create new team with invalid top-level key
	s.Do("POST", "/api/latest/fleet/spec/teams", json.RawMessage(`{
		"specs": [
			{"name": "team_with_invalid_key", "unknown_key": true}
		]
	}`), http.StatusOK, "force", "true")

	_, err = s.ds.TeamByName(context.Background(), "team_with_invalid_key")
	require.NoError(t, err)

	// invalid agent options command-line flag
	agentOpts = json.RawMessage(`{"command_line_flags": {"nope": 1}}`)
	teamSpecs = map[string]any{
		"specs": []any{
			map[string]any{
				"name":          teamName,
				"agent_options": agentOpts,
			},
		},
	}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusBadRequest)

	// valid agent options command-line flag
	agentOpts = json.RawMessage(`{"command_line_flags": {"enable_tables": "abcd"}}`)
	teamSpecs = map[string]any{
		"specs": []any{
			map[string]any{
				"name":          teamName,
				"agent_options": agentOpts,
			},
		},
	}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK)

	team, err = s.ds.TeamByName(context.Background(), teamName)
	require.NoError(t, err)
	require.Contains(t, string(*team.Config.AgentOptions), `"enable_tables": "abcd"`)

	// creates a team with default agent options
	user, err := s.ds.UserByEmail(context.Background(), "admin1@example.com")
	require.NoError(t, err)

	teams, err := s.ds.ListTeams(context.Background(), fleet.TeamFilter{User: user}, fleet.ListOptions{})
	require.NoError(t, err)
	require.True(t, len(teams) >= 1)

	teamSpecs = map[string]any{
		"specs": []any{
			map[string]any{
				"name": "team2",
			},
		},
	}
	applyResp = applyTeamSpecsResponse{}
	s.DoJSON("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK, &applyResp)
	require.Len(t, applyResp.TeamIDsByName, 1)

	teams, err = s.ds.ListTeams(context.Background(), fleet.TeamFilter{User: user}, fleet.ListOptions{})
	require.NoError(t, err)
	assert.True(t, len(teams) >= 2)

	team, err = s.ds.TeamByName(context.Background(), "team2")
	require.NoError(t, err)
	require.Equal(t, applyResp.TeamIDsByName["team2"], team.ID)

	appConfig, err := s.ds.AppConfig(context.Background())
	require.NoError(t, err)
	defaultOpts := `{"config": {"options": {"logger_plugin": "tls", "pack_delimiter": "/", "logger_tls_period": 10, "distributed_plugin": "tls", "disable_distributed": false, "logger_tls_endpoint": "/api/osquery/log", "distributed_interval": 10, "distributed_tls_max_attempts": 3}, "decorators": {"load": ["SELECT uuid AS host_uuid FROM system_info;", "SELECT hostname AS hostname FROM system_info;"]}}, "overrides": {}}`
	assert.Len(t, team.Secrets, 0) // no secret gets created automatically when creating a team via apply spec
	require.NotNil(t, team.Config.AgentOptions)
	require.JSONEq(t, defaultOpts, string(*team.Config.AgentOptions))
	require.Equal(t, appConfig.Features, team.Config.Features)

	// an activity was created for the newly created team via the applied spec
	s.lastActivityMatches(fleet.ActivityTypeAppliedSpecTeam{}.ActivityName(), fmt.Sprintf(`{"teams": [{"id": %d, "name": %q}]}`, team.ID, team.Name), 0)

	// updates
	teamSpecs = map[string]any{
		"specs": []any{
			map[string]any{
				"name":     "team2",
				"secrets":  []fleet.EnrollSecret{{Secret: "ABC"}},
				"features": nil,
			},
		},
	}
	applyResp = applyTeamSpecsResponse{}
	s.DoJSON("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK, &applyResp)
	require.Len(t, applyResp.TeamIDsByName, 1)

	team, err = s.ds.TeamByName(context.Background(), "team2")
	require.NoError(t, err)
	require.Equal(t, applyResp.TeamIDsByName["team2"], team.ID)

	require.Len(t, team.Secrets, 1)
	assert.Equal(t, "ABC", team.Secrets[0].Secret)
}

func (s *integrationEnterpriseTestSuite) TestTeamSpecsPermissions() {
	t := s.T()

	//
	// Setup test
	//

	// Create two teams, team1 and team2.
	team1, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		ID:          42,
		Name:        "team1",
		Description: "desc team1",
	})
	require.NoError(t, err)
	team2, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		ID:          43,
		Name:        "team2",
		Description: "desc team2",
	})
	require.NoError(t, err)
	// Create a new admin for team1.
	password := test.GoodPassword
	email := "admin-team1@example.com"
	u := &fleet.User{
		Name:       "admin team1",
		Email:      email,
		GlobalRole: nil,
		Teams: []fleet.UserTeam{
			{
				Team: *team1,
				Role: fleet.RoleAdmin,
			},
		},
	}
	require.NoError(t, u.SetPassword(password, 10, 10))
	_, err = s.ds.NewUser(context.Background(), u)
	require.NoError(t, err)

	//
	// Start testing team specs with admin of team1.
	//

	s.setTokenForTest(t, "admin-team1@example.com", test.GoodPassword)

	// Should allow editing own team.
	agentOpts := json.RawMessage(`{"config": {"views": {"foo": "bar2"}}, "overrides": {"platforms": {"darwin": {"views": {"bar": "qux"}}}}}`)
	editTeam1Spec := map[string]any{
		"specs": []any{
			map[string]any{
				"name":          team1.Name,
				"agent_options": agentOpts,
			},
		},
	}
	s.Do("POST", "/api/latest/fleet/spec/teams", editTeam1Spec, http.StatusOK)
	team1b, err := s.ds.Team(context.Background(), team1.ID)
	require.NoError(t, err)
	require.Equal(t, *team1b.Config.AgentOptions, agentOpts)

	// Should not allow editing other teams.
	editTeam2Spec := map[string]any{
		"specs": []any{
			map[string]any{
				"name":          team2.Name,
				"agent_options": agentOpts,
			},
		},
	}
	s.Do("POST", "/api/latest/fleet/spec/teams", editTeam2Spec, http.StatusForbidden)
}

func (s *integrationEnterpriseTestSuite) TestTeamSchedule() {
	t := s.T()

	team1, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		ID:          42,
		Name:        "team1",
		Description: "desc team1",
	})
	require.NoError(t, err)

	ts := getTeamScheduleResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/schedule", team1.ID), nil, http.StatusOK, &ts)
	require.Len(t, ts.Scheduled, 0)

	qr, err := s.ds.NewQuery(
		context.Background(),
		&fleet.Query{
			Name:           "TestQueryTeamPolicy",
			Description:    "Some description",
			Query:          "select * from osquery;",
			ObserverCanRun: true,
			Saved:          true,
		},
	)
	require.NoError(t, err)

	gsParams := teamScheduleQueryRequest{ScheduledQueryPayload: fleet.ScheduledQueryPayload{
		QueryID:  &qr.ID,
		Interval: ptr.Uint(42),
	}}
	r := teamScheduleQueryResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/schedule", team1.ID), gsParams, http.StatusOK, &r)

	ts = getTeamScheduleResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/schedule", team1.ID), nil, http.StatusOK, &ts)
	require.Len(t, ts.Scheduled, 1)
	assert.Equal(t, uint(42), ts.Scheduled[0].Interval)
	assert.Contains(t, ts.Scheduled[0].Name, "Copy of TestQueryTeamPolicy")
	assert.NotEqual(t, qr.ID, ts.Scheduled[0].QueryID) // it creates a new query (copy)
	id := ts.Scheduled[0].ID

	modifyResp := modifyTeamScheduleResponse{}
	modifyParams := modifyTeamScheduleRequest{ScheduledQueryPayload: fleet.ScheduledQueryPayload{Interval: ptr.Uint(55)}}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/schedule/%d", team1.ID, id), modifyParams, http.StatusOK, &modifyResp)

	// just to satisfy my paranoia, wanted to make sure the contents of the json would work
	s.DoRaw("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/schedule/%d", team1.ID, id), []byte(`{"interval": 77}`), http.StatusOK)

	ts = getTeamScheduleResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/schedule", team1.ID), nil, http.StatusOK, &ts)
	require.Len(t, ts.Scheduled, 1)
	assert.Equal(t, uint(77), ts.Scheduled[0].Interval)

	deleteResp := deleteTeamScheduleResponse{}
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/teams/%d/schedule/%d", team1.ID, id), nil, http.StatusOK, &deleteResp)

	ts = getTeamScheduleResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/schedule", team1.ID), nil, http.StatusOK, &ts)
	require.Len(t, ts.Scheduled, 0)
}

func (s *integrationEnterpriseTestSuite) TestTeamPolicies() {
	t := s.T()

	team1, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		ID:          42,
		Name:        "team1" + t.Name(),
		Description: "desc team1",
	})
	require.NoError(t, err)

	oldToken := s.token
	t.Cleanup(func() {
		s.token = oldToken
	})

	password := test.GoodPassword
	email := "testteam@user.com"

	u := &fleet.User{
		Name:       "test team user",
		Email:      email,
		GlobalRole: nil,
		Teams: []fleet.UserTeam{
			{
				Team: *team1,
				Role: fleet.RoleMaintainer,
			},
		},
	}
	require.NoError(t, u.SetPassword(password, 10, 10))
	_, err = s.ds.NewUser(context.Background(), u)
	require.NoError(t, err)

	s.token = s.getTestToken(email, password)

	ts := listTeamPoliciesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team1.ID), nil, http.StatusOK, &ts)
	require.Len(t, ts.Policies, 0)
	require.Len(t, ts.InheritedPolicies, 0)

	// create a global policy
	gpol, err := s.ds.NewGlobalPolicy(context.Background(), nil, fleet.PolicyPayload{Name: "TestGlobalPolicy", Query: "SELECT 1"})
	require.NoError(t, err)
	defer func() {
		_, err := s.ds.DeleteGlobalPolicies(context.Background(), []uint{gpol.ID})
		require.NoError(t, err)
	}()

	qr, err := s.ds.NewQuery(context.Background(), &fleet.Query{Name: "TestQuery2", Description: "Some description", Query: "select * from osquery;", ObserverCanRun: true})
	require.NoError(t, err)

	tpParams := teamPolicyRequest{
		QueryID:    &qr.ID,
		Resolution: "some team resolution",
	}
	r := teamPolicyResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team1.ID), tpParams, http.StatusOK, &r)

	ts = listTeamPoliciesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team1.ID), nil, http.StatusOK, &ts)
	require.Len(t, ts.Policies, 1)
	assert.Equal(t, "TestQuery2", ts.Policies[0].Name)
	assert.Equal(t, "select * from osquery;", ts.Policies[0].Query)
	assert.Equal(t, "Some description", ts.Policies[0].Description)
	require.NotNil(t, ts.Policies[0].Resolution)
	assert.Equal(t, "some team resolution", *ts.Policies[0].Resolution)
	require.Len(t, ts.InheritedPolicies, 1)
	assert.Equal(t, gpol.Name, ts.InheritedPolicies[0].Name)
	assert.Equal(t, gpol.ID, ts.InheritedPolicies[0].ID)

	deletePolicyParams := deleteTeamPoliciesRequest{IDs: []uint{ts.Policies[0].ID}}
	deletePolicyResp := deleteTeamPoliciesResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/delete", team1.ID), deletePolicyParams, http.StatusOK, &deletePolicyResp)

	ts = listTeamPoliciesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team1.ID), nil, http.StatusOK, &ts)
	require.Len(t, ts.Policies, 0)
}

func (s *integrationEnterpriseTestSuite) TestModifyTeamEnrollSecrets() {
	t := s.T()

	// Create new team and set initial secret
	teamName := t.Name() + "secretTeam"
	team := &fleet.Team{
		Name:        teamName,
		Description: "secretTeam description",
		Secrets:     []*fleet.EnrollSecret{{Secret: "initialSecret"}},
	}

	s.Do("POST", "/api/latest/fleet/teams", team, http.StatusOK)

	team, err := s.ds.TeamByName(context.Background(), teamName)
	require.NoError(t, err)
	assert.Equal(t, team.Secrets[0].Secret, "initialSecret")

	// Test replace existing secrets
	req := json.RawMessage(`{"secrets": [{"secret": "testSecret1"},{"secret": "testSecret2"}]}`)
	var resp teamEnrollSecretsResponse

	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/secrets", team.ID), req, http.StatusOK, &resp)
	require.Len(t, resp.Secrets, 2)

	team, err = s.ds.TeamByName(context.Background(), teamName)
	require.NoError(t, err)
	assert.Equal(t, "testSecret1", team.Secrets[0].Secret)
	assert.Equal(t, "testSecret2", team.Secrets[1].Secret)

	// Test delete all enroll secrets
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/secrets", team.ID), json.RawMessage(`{"secrets": []}`), http.StatusOK, &resp)
	require.Len(t, resp.Secrets, 0)

	// Test bad requests
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/secrets", team.ID), json.RawMessage(`{"foo": [{"secret": "testSecret3"}]}`), http.StatusUnprocessableEntity, &resp)
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/secrets", team.ID), json.RawMessage(`{}`), http.StatusUnprocessableEntity, &resp)

	// too many secrets
	secrets := createEnrollSecrets(t, fleet.MaxEnrollSecretsCount+1)
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/secrets", team.ID), json.RawMessage(`{"secrets": `+string(jsonMustMarshal(t, secrets))+`}`), http.StatusUnprocessableEntity, &resp)
}

func (s *integrationEnterpriseTestSuite) TestAvailableTeams() {
	t := s.T()

	// create a new team
	team := &fleet.Team{
		Name:        "Available Team",
		Description: "Available Team description",
	}

	s.Do("POST", "/api/latest/fleet/teams", team, http.StatusOK)

	team, err := s.ds.TeamByName(context.Background(), "Available Team")
	require.NoError(t, err)

	// create a new user
	user := &fleet.User{
		Name:       "Available Teams User",
		Email:      "available@example.com",
		GlobalRole: ptr.String("observer"),
	}
	err = user.SetPassword(test.GoodPassword, 10, 10)
	require.Nil(t, err)
	user, err = s.ds.NewUser(context.Background(), user)
	require.Nil(t, err)

	// test available teams for user assigned to global role
	var getResp getUserResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/users/%d", user.ID), nil, http.StatusOK, &getResp)
	assert.Equal(t, user.ID, getResp.User.ID)
	assert.Equal(t, ptr.String("observer"), getResp.User.GlobalRole)
	assert.Len(t, getResp.User.Teams, 0)     // teams is empty if user has a global role
	assert.Len(t, getResp.AvailableTeams, 1) // available teams includes all teams if user has a global role
	assert.Equal(t, getResp.AvailableTeams[0].Name, "Available Team")

	// assign user to a team
	user.GlobalRole = nil
	user.Teams = []fleet.UserTeam{{Team: *team, Role: "maintainer"}}
	err = s.ds.SaveUser(context.Background(), user)
	require.NoError(t, err)

	// test available teams for user assigned to team role
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/users/%d", user.ID), nil, http.StatusOK, &getResp)
	assert.Equal(t, user.ID, getResp.User.ID)
	assert.Nil(t, getResp.User.GlobalRole)
	assert.Len(t, getResp.User.Teams, 1)
	assert.Equal(t, getResp.User.Teams[0].Name, "Available Team")
	assert.Len(t, getResp.AvailableTeams, 1)
	assert.Equal(t, getResp.AvailableTeams[0].Name, "Available Team")

	// test available teams returned by `/me` endpoint
	key := make([]byte, 64)
	sessionKey := base64.StdEncoding.EncodeToString(key)
	_, err = s.ds.NewSession(context.Background(), user.ID, sessionKey)
	require.NoError(t, err)
	resp := s.DoRawWithHeaders("GET", "/api/latest/fleet/me", []byte(""), http.StatusOK, map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", sessionKey),
	})
	err = json.NewDecoder(resp.Body).Decode(&getResp)
	require.NoError(t, err)
	assert.Equal(t, user.ID, getResp.User.ID)
	assert.Nil(t, getResp.User.GlobalRole)
	assert.Len(t, getResp.User.Teams, 1)
	assert.Equal(t, getResp.User.Teams[0].Name, "Available Team")
	assert.Len(t, getResp.AvailableTeams, 1)
	assert.Equal(t, getResp.AvailableTeams[0].Name, "Available Team")
}

func (s *integrationEnterpriseTestSuite) TestTeamEndpoints() {
	t := s.T()

	name := strings.ReplaceAll(t.Name(), "/", "_")
	// create a new team
	team := &fleet.Team{
		Name:        name,
		Description: "Team description",
		Secrets:     []*fleet.EnrollSecret{{Secret: "DEF"}},
	}

	var tmResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", team, http.StatusOK, &tmResp)
	assert.Equal(t, team.Name, tmResp.Team.Name)
	require.Len(t, tmResp.Team.Secrets, 1)
	assert.Equal(t, "DEF", tmResp.Team.Secrets[0].Secret)

	// create a duplicate team (same name)
	team2 := &fleet.Team{
		Name:        name,
		Description: "Team2 description",
		Secrets:     []*fleet.EnrollSecret{{Secret: "GHI"}},
	}
	tmResp.Team = nil
	s.DoJSON("POST", "/api/latest/fleet/teams", team2, http.StatusConflict, &tmResp)

	// create a team with too many secrets
	team3 := &fleet.Team{
		Name:        name + "lots_of_secrets",
		Description: "Team3 description",
		Secrets:     createEnrollSecrets(t, fleet.MaxEnrollSecretsCount+1),
	}
	tmResp.Team = nil
	s.DoJSON("POST", "/api/latest/fleet/teams", team3, http.StatusUnprocessableEntity, &tmResp)

	// list teams
	var listResp listTeamsResponse
	s.DoJSON("GET", "/api/latest/fleet/teams", nil, http.StatusOK, &listResp, "query", name, "per_page", "2")
	require.Len(t, listResp.Teams, 1)
	assert.Equal(t, team.Name, listResp.Teams[0].Name)
	assert.NotNil(t, listResp.Teams[0].Config.AgentOptions)
	tm1ID := listResp.Teams[0].ID

	// get team
	var getResp getTeamResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", tm1ID), nil, http.StatusOK, &getResp)
	assert.Equal(t, team.Name, getResp.Team.Name)
	assert.NotNil(t, getResp.Team.Config.AgentOptions)

	// modify team
	team.Description = "Alt " + team.Description
	tmResp.Team = nil
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", tm1ID), team, http.StatusOK, &tmResp)
	assert.Contains(t, tmResp.Team.Description, "Alt ")

	// modify team's disk encryption, impossible without mdm enabled
	res := s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", tm1ID), fleet.TeamPayload{
		MDM: &fleet.TeamPayloadMDM{
			MacOSSettings: &fleet.MacOSSettings{EnableDiskEncryption: true},
		},
	}, http.StatusUnprocessableEntity)
	errMsg := extractServerErrorText(res.Body)
	assert.Contains(t, errMsg, `Couldn't update macos_settings because MDM features aren't turned on in Fleet.`)

	// modify a team with a NULL config
	defaultFeatures := fleet.Features{}
	defaultFeatures.ApplyDefaultsForNewInstalls()
	mysql.ExecAdhocSQL(t, s.ds, func(db sqlx.ExtContext) error {
		_, err := db.ExecContext(context.Background(), `UPDATE teams SET config = NULL WHERE id = ? `, tm1ID)
		return err
	})
	tmResp.Team = nil
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", tm1ID), team, http.StatusOK, &tmResp)
	assert.Equal(t, defaultFeatures, tmResp.Team.Config.Features)

	// modify a team with an empty config
	mysql.ExecAdhocSQL(t, s.ds, func(db sqlx.ExtContext) error {
		_, err := db.ExecContext(context.Background(), `UPDATE teams SET config = '{}' WHERE id = ? `, tm1ID)
		return err
	})
	tmResp.Team = nil
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", tm1ID), team, http.StatusOK, &tmResp)
	assert.Equal(t, defaultFeatures, tmResp.Team.Config.Features)

	// modify non-existing team
	tmResp.Team = nil
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", tm1ID+1), team, http.StatusNotFound, &tmResp)

	// list team users
	var usersResp listUsersResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/users", tm1ID), nil, http.StatusOK, &usersResp)
	assert.Len(t, usersResp.Users, 0)

	// list team users - non-existing team
	usersResp.Users = nil
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/users", tm1ID+1), nil, http.StatusNotFound, &usersResp)

	// create a new user
	user := &fleet.User{
		Name:       "Team User",
		Email:      "user@example.com",
		GlobalRole: ptr.String("observer"),
	}
	require.NoError(t, user.SetPassword(test.GoodPassword, 10, 10))
	user, err := s.ds.NewUser(context.Background(), user)
	require.NoError(t, err)

	// add a team user
	tmResp.Team = nil
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/users", tm1ID), modifyTeamUsersRequest{Users: []fleet.TeamUser{{User: *user, Role: fleet.RoleObserver}}}, http.StatusOK, &tmResp)
	require.Len(t, tmResp.Team.Users, 1)
	assert.Equal(t, user.ID, tmResp.Team.Users[0].ID)

	// add a team user - non-existing team
	tmResp.Team = nil
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/users", tm1ID+1), modifyTeamUsersRequest{Users: []fleet.TeamUser{{User: *user, Role: fleet.RoleObserver}}}, http.StatusNotFound, &tmResp)

	// add a team user - invalid user role
	tmResp.Team = nil
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/users", tm1ID), modifyTeamUsersRequest{Users: []fleet.TeamUser{{User: *user, Role: "foobar"}}}, http.StatusUnprocessableEntity, &tmResp)

	// search for that user
	usersResp.Users = nil
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/users", tm1ID), nil, http.StatusOK, &usersResp, "query", "user")
	require.Len(t, usersResp.Users, 1)
	assert.Equal(t, user.ID, usersResp.Users[0].ID)

	// search for unknown user
	usersResp.Users = nil
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/users", tm1ID), nil, http.StatusOK, &usersResp, "query", "notauser")
	require.Len(t, usersResp.Users, 0)

	// delete team user
	tmResp.Team = nil
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/teams/%d/users", tm1ID), modifyTeamUsersRequest{Users: []fleet.TeamUser{{User: fleet.User{ID: user.ID}}}}, http.StatusOK, &tmResp)
	require.Len(t, tmResp.Team.Users, 0)

	// delete team user - unknown user
	tmResp.Team = nil
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/teams/%d/users", tm1ID), modifyTeamUsersRequest{Users: []fleet.TeamUser{{User: fleet.User{ID: user.ID + 1}}}}, http.StatusOK, &tmResp)
	require.Len(t, tmResp.Team.Users, 0)

	// delete team user - unknown team
	tmResp.Team = nil
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/teams/%d/users", tm1ID+1), modifyTeamUsersRequest{Users: []fleet.TeamUser{{User: fleet.User{ID: user.ID}}}}, http.StatusNotFound, &tmResp)

	// modify team agent options with invalid options
	tmResp.Team = nil
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/agent_options", tm1ID), json.RawMessage(`{
		"x": "y"
	}`), http.StatusBadRequest, &tmResp)

	// modify team agent options with invalid options, but force-apply them
	tmResp.Team = nil
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/agent_options", tm1ID), json.RawMessage(`{
		"config": {
			"x": "y"
		}
	}`), http.StatusOK, &tmResp, "force", "true")
	require.Contains(t, string(*tmResp.Team.Config.AgentOptions), `"x": "y"`)

	// modify team agent options with valid options
	tmResp.Team = nil
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/agent_options", tm1ID), json.RawMessage(`{
		"config": {
			"options": {
				"aws_debug": true
			}
		}
	}`), http.StatusOK, &tmResp)
	require.Contains(t, string(*tmResp.Team.Config.AgentOptions), `"aws_debug": true`)

	// modify team agent using invalid options with dry-run
	tmResp.Team = nil
	resp := s.DoRaw("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/agent_options", tm1ID), json.RawMessage(`{
		"config": {
			"options": {
				"aws_debug": "not-a-bool"
			}
		}
	}`), http.StatusBadRequest, "dry_run", "true")
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Contains(t, string(body), "invalid value type at 'options.aws_debug': expected bool but got string")

	// modify team agent using valid options with dry-run
	tmResp.Team = nil
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/agent_options", tm1ID), json.RawMessage(`{
		"config": {
			"options": {
				"aws_debug": false
			}
		}
	}`), http.StatusOK, &tmResp, "dry_run", "true")
	require.Contains(t, string(*tmResp.Team.Config.AgentOptions), `"aws_debug": true`) // left unchanged

	// list activities, it should have created one for edited_agent_options
	s.lastActivityMatches(fleet.ActivityTypeEditedAgentOptions{}.ActivityName(), fmt.Sprintf(`{"global": false, "team_id": %d, "team_name": %q}`, tm1ID, team.Name), 0)

	// modify team agent options - unknown team
	tmResp.Team = nil
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/agent_options", tm1ID+1), json.RawMessage(`{}`), http.StatusNotFound, &tmResp)

	// get team enroll secrets
	var secResp teamEnrollSecretsResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/secrets", tm1ID), nil, http.StatusOK, &secResp)
	require.Len(t, secResp.Secrets, 1)
	assert.Equal(t, team.Secrets[0].Secret, secResp.Secrets[0].Secret)

	// get team enroll secrets- unknown team: does not return 404 because reads directly
	// the secrets table, does not load the team first (which would be unnecessary except
	// for checking that it exists)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/secrets", tm1ID+1), nil, http.StatusOK, &secResp)
	assert.Len(t, secResp.Secrets, 0)

	// delete team
	var delResp deleteTeamResponse
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/teams/%d", tm1ID), nil, http.StatusOK, &delResp)

	// delete team again, now an unknown team
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/teams/%d", tm1ID), nil, http.StatusNotFound, &delResp)
}

func (s *integrationEnterpriseTestSuite) TestTeamSecretsAreObfuscated() {
	t := s.T()

	// -----------------
	// Set up test data
	// -----------------
	teams := []*fleet.Team{
		{
			Name:        "Team One",
			Description: "Team description",
			Secrets:     []*fleet.EnrollSecret{{Secret: "DEF"}},
		},
		{
			Name:        "Team Two",
			Description: "Team Two description",
			Secrets:     []*fleet.EnrollSecret{{Secret: "ABC"}},
		},
	}
	for _, team := range teams {
		_, err := s.ds.NewTeam(context.Background(), team)
		require.NoError(t, err)
	}

	global_obs := &fleet.User{
		Name:       "Global Obs",
		Email:      "global_obs@example.com",
		GlobalRole: ptr.String(fleet.RoleObserver),
	}
	global_obs_plus := &fleet.User{
		Name:       "Global Obs+",
		Email:      "global_obs_plus@example.com",
		GlobalRole: ptr.String(fleet.RoleObserverPlus),
	}
	team_obs := &fleet.User{
		Name:  "Team Obs",
		Email: "team_obs@example.com",
		Teams: []fleet.UserTeam{
			{
				Team: *teams[0],
				Role: fleet.RoleObserver,
			},
			{
				Team: *teams[1],
				Role: fleet.RoleAdmin,
			},
		},
	}
	team_obs_plus := &fleet.User{
		Name:  "Team Obs Plus",
		Email: "team_obs_plus@example.com",
		Teams: []fleet.UserTeam{
			{
				Team: *teams[0],
				Role: fleet.RoleAdmin,
			},
			{
				Team: *teams[1],
				Role: fleet.RoleObserverPlus,
			},
		},
	}
	users := []*fleet.User{global_obs, global_obs_plus, team_obs, team_obs_plus}
	for _, u := range users {
		require.NoError(t, u.SetPassword(test.GoodPassword, 10, 10))
		_, err := s.ds.NewUser(context.Background(), u)
		require.NoError(t, err)
	}

	// --------------------------------------------------------------------
	// Global obs/obs+ should not be able to see any team secrets
	// --------------------------------------------------------------------
	for _, u := range []*fleet.User{global_obs, global_obs_plus} {

		s.setTokenForTest(t, u.Email, test.GoodPassword)

		// list all teams
		var listResp listTeamsResponse
		s.DoJSON("GET", "/api/latest/fleet/teams", nil, http.StatusOK, &listResp)

		require.Len(t, listResp.Teams, len(teams))
		require.NoError(t, listResp.Err)

		for _, team := range listResp.Teams {
			for _, secret := range team.Secrets {
				require.Equal(t, fleet.MaskedPassword, secret.Secret)
			}
		}

		// listing a team / team secrets
		for _, team := range teams {
			var getResp getTeamResponse
			s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &getResp)

			require.NoError(t, getResp.Err)
			for _, secret := range getResp.Team.Secrets {
				require.Equal(t, fleet.MaskedPassword, secret.Secret)
			}

			var secResp teamEnrollSecretsResponse
			s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/secrets", team.ID), nil, http.StatusOK, &secResp)

			require.Len(t, secResp.Secrets, 1)
			require.NoError(t, secResp.Err)
			for _, secret := range secResp.Secrets {
				require.Equal(t, fleet.MaskedPassword, secret.Secret)
			}
		}
	}

	// --------------------------------------------------------------------
	// Team obs/obs+ should not be able to see their team secrets
	// --------------------------------------------------------------------
	for _, u := range []*fleet.User{team_obs, team_obs_plus} {

		s.setTokenForTest(t, u.Email, test.GoodPassword)

		// list all teams
		var listResp listTeamsResponse
		s.DoJSON("GET", "/api/latest/fleet/teams", nil, http.StatusOK, &listResp)

		require.Len(t, listResp.Teams, len(u.Teams))
		require.NoError(t, listResp.Err)

		for _, team := range listResp.Teams {
			for _, secret := range team.Secrets {
				// team_obs has RoleObserver in Team 1, and an RoleAdmin in Team 2
				// so it should be able to see the secrets in Team 1
				if u.ID == team_obs.ID {
					require.Equal(t, fleet.MaskedPassword == secret.Secret, team.ID == teams[0].ID)
					require.Equal(t, fleet.MaskedPassword != secret.Secret, team.ID == teams[1].ID)
				}

				// team_obs_plus should not be able to see any Team Secret
				if u.ID == team_obs_plus.ID {
					require.Equal(t, fleet.MaskedPassword == secret.Secret, team.ID == teams[1].ID)
					require.Equal(t, fleet.MaskedPassword != secret.Secret, team.ID == teams[0].ID)
				}
			}
		}

		// listing a team / team secrets
		for _, team := range teams {
			var getResp getTeamResponse
			s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &getResp)

			require.NoError(t, getResp.Err)
			// team_obs has RoleObserver in Team 1, and an RoleAdmin in Team 2
			// so it should be able to see the secrets in Team 1
			for _, secret := range getResp.Team.Secrets {
				if u.ID == team_obs.ID {
					require.Equal(t, fleet.MaskedPassword == secret.Secret, team.ID == teams[0].ID)
					require.Equal(t, fleet.MaskedPassword != secret.Secret, team.ID == teams[1].ID)
				}

				if u.ID == team_obs_plus.ID {
					require.Equal(t, fleet.MaskedPassword == secret.Secret, team.ID == teams[1].ID)
					require.Equal(t, fleet.MaskedPassword != secret.Secret, team.ID == teams[0].ID)
				}
			}

			var secResp teamEnrollSecretsResponse
			s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/secrets", team.ID), nil, http.StatusOK, &secResp)

			require.Len(t, secResp.Secrets, 1)
			require.NoError(t, secResp.Err)
			for _, secret := range secResp.Secrets {
				if u.ID == team_obs.ID {
					require.Equal(t, fleet.MaskedPassword == secret.Secret, team.ID == teams[0].ID)
					require.Equal(t, fleet.MaskedPassword != secret.Secret, team.ID == teams[1].ID)
				}

				if u.ID == team_obs_plus.ID {
					require.Equal(t, fleet.MaskedPassword == secret.Secret, team.ID == teams[1].ID)
					require.Equal(t, fleet.MaskedPassword != secret.Secret, team.ID == teams[0].ID)
				}
			}
		}
	}
}

func (s *integrationEnterpriseTestSuite) TestExternalIntegrationsTeamConfig() {
	t := s.T()

	// create a test http server to act as the Jira and Zendesk server
	srvURL := startExternalServiceWebServer(t)

	// create a new team
	team := &fleet.Team{
		Name:        t.Name(),
		Description: "Team description",
		Secrets:     []*fleet.EnrollSecret{{Secret: "XYZ"}},
	}
	var tmResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", team, http.StatusOK, &tmResp)
	require.Equal(t, team.Name, tmResp.Team.Name)
	require.Len(t, tmResp.Team.Secrets, 1)
	require.Equal(t, "XYZ", tmResp.Team.Secrets[0].Secret)
	team.ID = tmResp.Team.ID

	// modify the team's config - enable the webhook
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{WebhookSettings: &fleet.TeamWebhookSettings{
		FailingPoliciesWebhook: fleet.FailingPoliciesWebhookSettings{
			Enable:         true,
			DestinationURL: "http://example.com",
		},
	}}, http.StatusOK, &tmResp)
	require.True(t, tmResp.Team.Config.WebhookSettings.FailingPoliciesWebhook.Enable)
	require.Equal(t, "http://example.com", tmResp.Team.Config.WebhookSettings.FailingPoliciesWebhook.DestinationURL)

	// add an unknown automation - does not exist at the global level
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{Integrations: &fleet.TeamIntegrations{
		Jira: []*fleet.TeamJiraIntegration{
			{
				URL:                   srvURL,
				ProjectKey:            "qux",
				EnableFailingPolicies: false,
			},
		},
	}}, http.StatusUnprocessableEntity, &tmResp)

	// add a couple Jira integrations at the global level (qux and qux2)
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"jira": [
				{
					"url": %q,
					"username": "ok",
					"api_token": "foo",
					"project_key": "qux"
				},
				{
					"url": %[1]q,
					"username": "ok",
					"api_token": "foo",
					"project_key": "qux2"
				}
			]
		}
	}`, srvURL)), http.StatusOK)

	// enable an automation - should fail as the webhook is enabled too
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{Integrations: &fleet.TeamIntegrations{
		Jira: []*fleet.TeamJiraIntegration{
			{
				URL:                   srvURL,
				ProjectKey:            "qux",
				EnableFailingPolicies: true,
			},
		},
	}}, http.StatusUnprocessableEntity, &tmResp)

	// get the team, no integration was saved
	var getResp getTeamResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &getResp)
	require.Len(t, getResp.Team.Config.Integrations.Jira, 0)
	require.Len(t, getResp.Team.Config.Integrations.Zendesk, 0)

	// disable the webhook and enable the automation
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{
		Integrations: &fleet.TeamIntegrations{
			Jira: []*fleet.TeamJiraIntegration{
				{
					URL:                   srvURL,
					ProjectKey:            "qux",
					EnableFailingPolicies: true,
				},
			},
		},
		WebhookSettings: &fleet.TeamWebhookSettings{
			FailingPoliciesWebhook: fleet.FailingPoliciesWebhookSettings{
				Enable:         false,
				DestinationURL: "http://example.com",
			},
		},
	}, http.StatusOK, &tmResp)
	require.Len(t, tmResp.Team.Config.Integrations.Jira, 1)
	require.Equal(t, "qux", tmResp.Team.Config.Integrations.Jira[0].ProjectKey)

	// enable the webhook without changing the integration should fail (an integration is already enabled)
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{WebhookSettings: &fleet.TeamWebhookSettings{
		FailingPoliciesWebhook: fleet.FailingPoliciesWebhookSettings{
			Enable:         true,
			DestinationURL: "http://example.com",
		},
	}}, http.StatusUnprocessableEntity, &tmResp)

	// add a second, disabled Jira integration
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{
		Integrations: &fleet.TeamIntegrations{
			Jira: []*fleet.TeamJiraIntegration{
				{
					URL:                   srvURL,
					ProjectKey:            "qux",
					EnableFailingPolicies: true,
				},
				{
					URL:                   srvURL,
					ProjectKey:            "qux2",
					EnableFailingPolicies: false,
				},
			},
		},
	}, http.StatusOK, &tmResp)
	require.Len(t, tmResp.Team.Config.Integrations.Jira, 2)
	require.Equal(t, "qux", tmResp.Team.Config.Integrations.Jira[0].ProjectKey)
	require.Equal(t, "qux2", tmResp.Team.Config.Integrations.Jira[1].ProjectKey)

	// enabling the second without disabling the first fails
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{
		Integrations: &fleet.TeamIntegrations{
			Jira: []*fleet.TeamJiraIntegration{
				{
					URL:                   srvURL,
					ProjectKey:            "qux",
					EnableFailingPolicies: true,
				},
				{
					URL:                   srvURL,
					ProjectKey:            "qux2",
					EnableFailingPolicies: true,
				},
			},
		},
	}, http.StatusUnprocessableEntity, &tmResp)

	// updating to use the same project key fails (must be unique)
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{
		Integrations: &fleet.TeamIntegrations{
			Jira: []*fleet.TeamJiraIntegration{
				{
					URL:                   srvURL,
					ProjectKey:            "qux",
					EnableFailingPolicies: true,
				},
				{
					URL:                   srvURL,
					ProjectKey:            "qux",
					EnableFailingPolicies: false,
				},
			},
		},
	}, http.StatusUnprocessableEntity, &tmResp)

	// remove second integration, disable first so that nothing is enabled now
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{
		Integrations: &fleet.TeamIntegrations{
			Jira: []*fleet.TeamJiraIntegration{
				{
					URL:                   srvURL,
					ProjectKey:            "qux",
					EnableFailingPolicies: false,
				},
			},
		},
	}, http.StatusOK, &tmResp)

	// enable the webhook now works
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{WebhookSettings: &fleet.TeamWebhookSettings{
		FailingPoliciesWebhook: fleet.FailingPoliciesWebhookSettings{
			Enable:         true,
			DestinationURL: "http://example.com",
		},
	}}, http.StatusOK, &tmResp)

	// set environmental varible to use Zendesk test client
	t.Setenv("TEST_ZENDESK_CLIENT", "true")

	// add an unknown automation - does not exist at the global level
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{Integrations: &fleet.TeamIntegrations{
		Zendesk: []*fleet.TeamZendeskIntegration{
			{
				URL:                   srvURL,
				GroupID:               122,
				EnableFailingPolicies: false,
			},
		},
	}}, http.StatusUnprocessableEntity, &tmResp)

	// add a couple Zendesk integrations at the global level (122 and 123), keep the jira ones too
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"zendesk": [
				{
					"url": %q,
					"email": "a@b.c",
					"api_token": "ok",
					"group_id": 122
				},
				{
					"url": %[1]q,
					"email": "b@b.c",
					"api_token": "ok",
					"group_id": 123
				}
			],
			"jira": [
				{
					"url": %[1]q,
					"username": "ok",
					"api_token": "foo",
					"project_key": "qux"
				},
				{
					"url": %[1]q,
					"username": "ok",
					"api_token": "foo",
					"project_key": "qux2"
				}
			]
		}
	}`, srvURL)), http.StatusOK)

	// enable a Zendesk automation - should fail as the webhook is enabled too
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{Integrations: &fleet.TeamIntegrations{
		Zendesk: []*fleet.TeamZendeskIntegration{
			{
				URL:                   srvURL,
				GroupID:               122,
				EnableFailingPolicies: true,
			},
		},
	}}, http.StatusUnprocessableEntity, &tmResp)

	// disable the webhook and enable the automation
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{
		Integrations: &fleet.TeamIntegrations{
			Zendesk: []*fleet.TeamZendeskIntegration{
				{
					URL:                   srvURL,
					GroupID:               122,
					EnableFailingPolicies: true,
				},
			},
		},
		WebhookSettings: &fleet.TeamWebhookSettings{
			FailingPoliciesWebhook: fleet.FailingPoliciesWebhookSettings{
				Enable:         false,
				DestinationURL: "http://example.com",
			},
		},
	}, http.StatusOK, &tmResp)
	require.Len(t, tmResp.Team.Config.Integrations.Zendesk, 1)
	require.Equal(t, int64(122), tmResp.Team.Config.Integrations.Zendesk[0].GroupID)

	// enable the webhook without changing the integration should fail (an integration is already enabled)
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{WebhookSettings: &fleet.TeamWebhookSettings{
		FailingPoliciesWebhook: fleet.FailingPoliciesWebhookSettings{
			Enable:         true,
			DestinationURL: "http://example.com",
		},
	}}, http.StatusUnprocessableEntity, &tmResp)

	// add a second, disabled Zendesk integration
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{
		Integrations: &fleet.TeamIntegrations{
			Zendesk: []*fleet.TeamZendeskIntegration{
				{
					URL:                   srvURL,
					GroupID:               122,
					EnableFailingPolicies: true,
				},
				{
					URL:                   srvURL,
					GroupID:               123,
					EnableFailingPolicies: false,
				},
			},
		},
	}, http.StatusOK, &tmResp)
	require.Len(t, tmResp.Team.Config.Integrations.Zendesk, 2)
	require.Equal(t, int64(122), tmResp.Team.Config.Integrations.Zendesk[0].GroupID)
	require.Equal(t, int64(123), tmResp.Team.Config.Integrations.Zendesk[1].GroupID)

	// enabling the second without disabling the first fails
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{
		Integrations: &fleet.TeamIntegrations{
			Zendesk: []*fleet.TeamZendeskIntegration{
				{
					URL:                   srvURL,
					GroupID:               122,
					EnableFailingPolicies: true,
				},
				{
					URL:                   srvURL,
					GroupID:               123,
					EnableFailingPolicies: true,
				},
			},
		},
	}, http.StatusUnprocessableEntity, &tmResp)

	// updating to use the same group ID fails (must be unique per group ID)
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{
		Integrations: &fleet.TeamIntegrations{
			Zendesk: []*fleet.TeamZendeskIntegration{
				{
					URL:                   srvURL,
					GroupID:               123,
					EnableFailingPolicies: true,
				},
				{
					URL:                   srvURL,
					GroupID:               123,
					EnableFailingPolicies: false,
				},
			},
		},
	}, http.StatusUnprocessableEntity, &tmResp)

	// remove second Zendesk integration, add disabled Jira integration
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{
		Integrations: &fleet.TeamIntegrations{
			Zendesk: []*fleet.TeamZendeskIntegration{
				{
					URL:                   srvURL,
					GroupID:               122,
					EnableFailingPolicies: true,
				},
			},
			Jira: []*fleet.TeamJiraIntegration{
				{
					URL:                   srvURL,
					ProjectKey:            "qux",
					EnableFailingPolicies: false,
				},
			},
		},
	}, http.StatusOK, &tmResp)
	require.Len(t, tmResp.Team.Config.Integrations.Jira, 1)
	require.Equal(t, "qux", tmResp.Team.Config.Integrations.Jira[0].ProjectKey)
	require.Len(t, tmResp.Team.Config.Integrations.Zendesk, 1)
	require.Equal(t, int64(122), tmResp.Team.Config.Integrations.Zendesk[0].GroupID)

	// enabling a Jira integration when a Zendesk one is enabled fails
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{
		Integrations: &fleet.TeamIntegrations{
			Zendesk: []*fleet.TeamZendeskIntegration{
				{
					URL:                   srvURL,
					GroupID:               122,
					EnableFailingPolicies: true,
				},
			},
			Jira: []*fleet.TeamJiraIntegration{
				{
					URL:                   srvURL,
					ProjectKey:            "qux",
					EnableFailingPolicies: true,
				},
			},
		},
	}, http.StatusUnprocessableEntity, &tmResp)

	// set additional integrations on the team
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{
		Integrations: &fleet.TeamIntegrations{
			Zendesk: []*fleet.TeamZendeskIntegration{
				{
					URL:                   srvURL,
					GroupID:               122,
					EnableFailingPolicies: true,
				},
				{
					URL:                   srvURL,
					GroupID:               123,
					EnableFailingPolicies: false,
				},
			},
			Jira: []*fleet.TeamJiraIntegration{
				{
					URL:                   srvURL,
					ProjectKey:            "qux",
					EnableFailingPolicies: false,
				},
			},
		},
	}, http.StatusOK, &tmResp)

	// removing Zendesk 122 from the global config removes it from the team too
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"zendesk": [
				{
					"url": %[1]q,
					"email": "b@b.c",
					"api_token": "ok",
					"group_id": 123
				}
			],
			"jira": [
				{
					"url": %[1]q,
					"username": "ok",
					"api_token": "foo",
					"project_key": "qux"
				},
				{
					"url": %[1]q,
					"username": "ok",
					"api_token": "foo",
					"project_key": "qux2"
				}
			]
		}
	}`, srvURL)), http.StatusOK)

	// get the team, only one Zendesk integration remains, none are enabled
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &getResp)
	require.Len(t, getResp.Team.Config.Integrations.Jira, 1)
	require.Equal(t, "qux", getResp.Team.Config.Integrations.Jira[0].ProjectKey)
	require.False(t, getResp.Team.Config.Integrations.Jira[0].EnableFailingPolicies)
	require.Len(t, getResp.Team.Config.Integrations.Zendesk, 1)
	require.Equal(t, int64(123), getResp.Team.Config.Integrations.Zendesk[0].GroupID)
	require.False(t, getResp.Team.Config.Integrations.Zendesk[0].EnableFailingPolicies)

	// removing Jira qux2 from the global config does not impact the team as it is unused.
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"zendesk": [
				{
					"url": %[1]q,
					"email": "b@b.c",
					"api_token": "ok",
					"group_id": 123
				}
			],
			"jira": [
				{
					"url": %[1]q,
					"username": "ok",
					"api_token": "foo",
					"project_key": "qux"
				}
			]
		}
	}`, srvURL)), http.StatusOK)

	// get the team, integrations are unchanged
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &getResp)
	require.Len(t, getResp.Team.Config.Integrations.Jira, 1)
	require.Equal(t, "qux", getResp.Team.Config.Integrations.Jira[0].ProjectKey)
	require.False(t, getResp.Team.Config.Integrations.Jira[0].EnableFailingPolicies)
	require.Len(t, getResp.Team.Config.Integrations.Zendesk, 1)
	require.Equal(t, int64(123), getResp.Team.Config.Integrations.Zendesk[0].GroupID)
	require.False(t, getResp.Team.Config.Integrations.Zendesk[0].EnableFailingPolicies)

	// enable Jira qux for the team
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{
		Integrations: &fleet.TeamIntegrations{
			Zendesk: []*fleet.TeamZendeskIntegration{
				{
					URL:                   srvURL,
					GroupID:               123,
					EnableFailingPolicies: false,
				},
			},
			Jira: []*fleet.TeamJiraIntegration{
				{
					URL:                   srvURL,
					ProjectKey:            "qux",
					EnableFailingPolicies: true,
				},
			},
		},
	}, http.StatusOK, &tmResp)

	// removing Zendesk 123 from the global config removes it from the team but
	// leaves the Jira integration enabled.
	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(fmt.Sprintf(`{
		"integrations": {
			"jira": [
				{
					"url": %[1]q,
					"username": "ok",
					"api_token": "foo",
					"project_key": "qux"
				}
			]
		}
	}`, srvURL)), http.StatusOK)

	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &getResp)
	require.Len(t, getResp.Team.Config.Integrations.Jira, 1)
	require.Equal(t, "qux", getResp.Team.Config.Integrations.Jira[0].ProjectKey)
	require.True(t, getResp.Team.Config.Integrations.Jira[0].EnableFailingPolicies)
	require.Len(t, getResp.Team.Config.Integrations.Zendesk, 0)

	// remove all integrations on exit, so that other tests can enable the
	// webhook as needed
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{
		Integrations: &fleet.TeamIntegrations{
			Zendesk: []*fleet.TeamZendeskIntegration{},
			Jira:    []*fleet.TeamJiraIntegration{},
		},
		WebhookSettings: &fleet.TeamWebhookSettings{},
	}, http.StatusOK, &tmResp)
	require.Len(t, tmResp.Team.Config.Integrations.Jira, 0)
	require.Len(t, tmResp.Team.Config.Integrations.Zendesk, 0)
	require.False(t, tmResp.Team.Config.WebhookSettings.FailingPoliciesWebhook.Enable)
	require.Empty(t, tmResp.Team.Config.WebhookSettings.FailingPoliciesWebhook.DestinationURL)

	s.DoRaw("PATCH", "/api/v1/fleet/config", []byte(`{
		"integrations": {}
	}`), http.StatusOK)
}

func (s *integrationEnterpriseTestSuite) TestMacOSUpdatesConfig() {
	t := s.T()

	// Create a team
	team := &fleet.Team{
		Name:        t.Name(),
		Description: "Team description",
		Secrets:     []*fleet.EnrollSecret{{Secret: "XYZ"}},
	}
	var tmResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", team, http.StatusOK, &tmResp)
	require.Equal(t, team.Name, tmResp.Team.Name)
	team.ID = tmResp.Team.ID

	// modify the team's config
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), map[string]any{
		"mdm": map[string]any{
			"macos_updates": &fleet.MacOSUpdates{
				MinimumVersion: optjson.SetString("10.15.0"),
				Deadline:       optjson.SetString("2021-01-01"),
			},
		},
	}, http.StatusOK, &tmResp)
	require.Equal(t, "10.15.0", tmResp.Team.Config.MDM.MacOSUpdates.MinimumVersion.Value)
	require.Equal(t, "2021-01-01", tmResp.Team.Config.MDM.MacOSUpdates.Deadline.Value)
	s.lastActivityMatches(fleet.ActivityTypeEditedMacOSMinVersion{}.ActivityName(), fmt.Sprintf(`{"team_id": %d, "team_name": %q, "minimum_version": "10.15.0", "deadline": "2021-01-01"}`, team.ID, team.Name), 0)

	// only update the deadline
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), map[string]any{
		"mdm": map[string]any{
			"macos_updates": &fleet.MacOSUpdates{
				MinimumVersion: optjson.SetString("10.15.0"),
				Deadline:       optjson.SetString("2025-10-01"),
			},
		},
	}, http.StatusOK, &tmResp)
	require.Equal(t, "10.15.0", tmResp.Team.Config.MDM.MacOSUpdates.MinimumVersion.Value)
	require.Equal(t, "2025-10-01", tmResp.Team.Config.MDM.MacOSUpdates.Deadline.Value)
	lastActivity := s.lastActivityMatches(fleet.ActivityTypeEditedMacOSMinVersion{}.ActivityName(), fmt.Sprintf(`{"team_id": %d, "team_name": %q, "minimum_version": "10.15.0", "deadline": "2025-10-01"}`, team.ID, team.Name), 0)

	// sending a nil MDM or MacOSUpdate config doesn't modify anything
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), map[string]any{
		"mdm": nil,
	}, http.StatusOK, &tmResp)
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), map[string]any{
		"mdm": map[string]any{
			"macos_updates": nil,
		},
	}, http.StatusOK, &tmResp)
	require.Equal(t, "10.15.0", tmResp.Team.Config.MDM.MacOSUpdates.MinimumVersion.Value)
	require.Equal(t, "2025-10-01", tmResp.Team.Config.MDM.MacOSUpdates.Deadline.Value)
	// no new activity is created
	s.lastActivityMatches("", "", lastActivity)

	// sending macos settings but no macos_updates does not change the macos updates
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), map[string]any{
		"mdm": map[string]any{
			"macos_settings": map[string]any{
				"custom_settings": nil,
			},
		},
	}, http.StatusOK, &tmResp)
	require.Equal(t, "10.15.0", tmResp.Team.Config.MDM.MacOSUpdates.MinimumVersion.Value)
	require.Equal(t, "2025-10-01", tmResp.Team.Config.MDM.MacOSUpdates.Deadline.Value)
	// no new activity is created
	s.lastActivityMatches("", "", lastActivity)

	// sending empty MacOSUpdate fields empties both fields
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), map[string]any{
		"mdm": map[string]any{
			"macos_updates": map[string]any{
				"minimum_version": "",
				"deadline":        nil,
			},
		},
	}, http.StatusOK, &tmResp)
	require.Empty(t, tmResp.Team.Config.MDM.MacOSUpdates.MinimumVersion.Value)
	require.Empty(t, tmResp.Team.Config.MDM.MacOSUpdates.Deadline.Value)
	s.lastActivityMatches(fleet.ActivityTypeEditedMacOSMinVersion{}.ActivityName(), fmt.Sprintf(`{"team_id": %d, "team_name": %q, "minimum_version": "", "deadline": ""}`, team.ID, team.Name), 0)

	// error checks:

	// try to set an invalid deadline
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), map[string]any{
		"mdm": map[string]any{
			"macos_updates": map[string]any{
				"minimum_version": "10.15.0",
				"deadline":        "2021-01-01T00:00:00Z",
			},
		},
	}, http.StatusUnprocessableEntity, &tmResp)

	// try to set an invalid minimum version
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), map[string]any{
		"mdm": map[string]any{
			"macos_updates": map[string]any{
				"minimum_version": "10.15.0 (19A583)",
				"deadline":        "2021-01-01T00:00:00Z",
			},
		},
	}, http.StatusUnprocessableEntity, &tmResp)

	// try to set a deadline but not a minimum version
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), map[string]any{
		"mdm": map[string]any{
			"macos_updates": map[string]any{
				"deadline": "2021-01-01T00:00:00Z",
			},
		},
	}, http.StatusUnprocessableEntity, &tmResp)

	// try to set an empty deadline but not a minimum version
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), map[string]any{
		"mdm": map[string]any{
			"macos_updates": map[string]any{
				"deadline": "",
			},
		},
	}, http.StatusUnprocessableEntity, &tmResp)

	// try to set a minimum version but not a deadline
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), map[string]any{
		"mdm": map[string]any{
			"macos_updates": map[string]any{
				"minimum_version": "10.15.0 (19A583)",
			},
		},
	}, http.StatusUnprocessableEntity, &tmResp)

	// try to set an empty minimum version but not a deadline
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), map[string]any{
		"mdm": map[string]any{
			"macos_updates": map[string]any{
				"minimum_version": "",
			},
		},
	}, http.StatusUnprocessableEntity, &tmResp)
}

func (s *integrationEnterpriseTestSuite) TestListDevicePolicies() {
	t := s.T()

	ac, err := s.ds.AppConfig(context.Background())
	require.NoError(t, err)
	ac.OrgInfo.OrgLogoURL = "http://example.com/logo"
	err = s.ds.SaveAppConfig(context.Background(), ac)
	require.NoError(t, err)

	team, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		ID:          51,
		Name:        "team1-policies",
		Description: "desc team1",
	})
	require.NoError(t, err)

	token := "much_valid"
	host := createHostAndDeviceToken(t, s.ds, token)
	err = s.ds.AddHostsToTeam(context.Background(), &team.ID, []uint{host.ID})
	require.NoError(t, err)

	qr, err := s.ds.NewQuery(context.Background(), &fleet.Query{
		Name:           "TestQueryEnterpriseGlobalPolicy",
		Description:    "Some description",
		Query:          "select * from osquery;",
		ObserverCanRun: true,
	})
	require.NoError(t, err)

	// add a global policy
	gpParams := globalPolicyRequest{
		QueryID:    &qr.ID,
		Resolution: "some global resolution",
	}
	gpResp := globalPolicyResponse{}
	s.DoJSON("POST", "/api/latest/fleet/policies", gpParams, http.StatusOK, &gpResp)
	require.NotNil(t, gpResp.Policy)

	// add a policy execution
	require.NoError(t, s.ds.RecordPolicyQueryExecutions(context.Background(), host,
		map[uint]*bool{gpResp.Policy.ID: ptr.Bool(false)}, time.Now(), false))

	// add a policy to team
	oldToken := s.token
	t.Cleanup(func() {
		s.token = oldToken
	})

	password := test.GoodPassword
	email := "test_enterprise_policies@user.com"

	u := &fleet.User{
		Name:       "test team user",
		Email:      email,
		GlobalRole: nil,
		Teams: []fleet.UserTeam{
			{
				Team: *team,
				Role: fleet.RoleMaintainer,
			},
		},
	}

	require.NoError(t, u.SetPassword(password, 10, 10))
	_, err = s.ds.NewUser(context.Background(), u)
	require.NoError(t, err)

	s.token = s.getTestToken(email, password)
	tpParams := teamPolicyRequest{
		Name:        "TestQueryEnterpriseTeamPolicy",
		Query:       "select * from osquery;",
		Description: "Some description",
		Resolution:  "some team resolution",
		Platform:    "darwin",
	}
	tpResp := teamPolicyResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team.ID), tpParams, http.StatusOK, &tpResp)

	// try with invalid token
	res := s.DoRawNoAuth("GET", "/api/latest/fleet/device/invalid_token/policies", nil, http.StatusUnauthorized)
	res.Body.Close()

	// GET `/api/_version_/fleet/device/{token}/policies`
	listDevicePoliciesResp := listDevicePoliciesResponse{}
	res = s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/policies", nil, http.StatusOK)
	json.NewDecoder(res.Body).Decode(&listDevicePoliciesResp) //nolint:errcheck
	res.Body.Close()                                          //nolint:errcheck
	require.Len(t, listDevicePoliciesResp.Policies, 2)
	require.NoError(t, listDevicePoliciesResp.Err)

	// GET `/api/_version_/fleet/device/{token}`
	getDeviceHostResp := getDeviceHostResponse{}
	res = s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token, nil, http.StatusOK)
	json.NewDecoder(res.Body).Decode(&getDeviceHostResp) //nolint:errcheck
	res.Body.Close()                                     //nolint:errcheck
	require.NoError(t, getDeviceHostResp.Err)
	require.Equal(t, host.ID, getDeviceHostResp.Host.ID)
	require.False(t, getDeviceHostResp.Host.RefetchRequested)
	require.Equal(t, "http://example.com/logo", getDeviceHostResp.OrgLogoURL)
	require.Len(t, *getDeviceHostResp.Host.Policies, 2)

	// GET `/api/_version_/fleet/device/{token}/desktop`
	getDesktopResp := fleetDesktopResponse{}
	res = s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/desktop", nil, http.StatusOK)
	require.NoError(t, json.NewDecoder(res.Body).Decode(&getDesktopResp))
	require.NoError(t, res.Body.Close())
	require.NoError(t, getDesktopResp.Err)
	require.Equal(t, *getDesktopResp.FailingPolicies, uint(1))
	require.False(t, getDesktopResp.Notifications.NeedsMDMMigration)
}

// TestCustomTransparencyURL tests that Fleet Premium licensees can use custom transparency urls.
func (s *integrationEnterpriseTestSuite) TestCustomTransparencyURL() {
	t := s.T()

	token := "token_test_custom_transparency_url"
	createHostAndDeviceToken(t, s.ds, token)

	// confirm intitial default url
	acResp := appConfigResponse{}
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	require.NotNil(t, acResp)
	require.Equal(t, fleet.DefaultTransparencyURL, acResp.FleetDesktop.TransparencyURL)

	// confirm device endpoint returns initial default url
	deviceResp := &transparencyURLResponse{}
	rawResp := s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/transparency", nil, http.StatusTemporaryRedirect)
	json.NewDecoder(rawResp.Body).Decode(deviceResp) //nolint:errcheck
	rawResp.Body.Close()                             //nolint:errcheck
	require.NoError(t, deviceResp.Err)
	require.Equal(t, fleet.DefaultTransparencyURL, rawResp.Header.Get("Location"))

	// set custom url
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{"fleet_desktop":{"transparency_url": "customURL"}}`), http.StatusOK, &acResp)
	require.NotNil(t, acResp)
	require.Equal(t, "customURL", acResp.FleetDesktop.TransparencyURL)

	// device endpoint returns custom url
	deviceResp = &transparencyURLResponse{}
	rawResp = s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/transparency", nil, http.StatusTemporaryRedirect)
	json.NewDecoder(rawResp.Body).Decode(deviceResp) //nolint:errcheck
	rawResp.Body.Close()                             //nolint:errcheck
	require.NoError(t, deviceResp.Err)
	require.Equal(t, "customURL", rawResp.Header.Get("Location"))

	// empty string applies default url
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{"fleet_desktop":{"transparency_url": ""}}`), http.StatusOK, &acResp)
	require.NotNil(t, acResp)
	require.Equal(t, fleet.DefaultTransparencyURL, acResp.FleetDesktop.TransparencyURL)

	// device endpoint returns default url
	deviceResp = &transparencyURLResponse{}
	rawResp = s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/transparency", nil, http.StatusTemporaryRedirect)
	json.NewDecoder(rawResp.Body).Decode(deviceResp) //nolint:errcheck
	rawResp.Body.Close()                             //nolint:errcheck
	require.NoError(t, deviceResp.Err)
	require.Equal(t, fleet.DefaultTransparencyURL, rawResp.Header.Get("Location"))
}

func (s *integrationEnterpriseTestSuite) TestDefaultAppleBMTeam() {
	t := s.T()

	tm, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		Name:        t.Name(),
		Description: "desc",
	})
	require.NoError(s.T(), err)

	var acResp appConfigResponse

	// try to set an invalid team name
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": {
			"apple_bm_default_team": "xyz"
		}
	}`), http.StatusUnprocessableEntity, &acResp)

	// get the appconfig, nothing changed
	acResp = appConfigResponse{}
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	require.Empty(t, acResp.MDM.AppleBMDefaultTeam)

	// set to a valid team name
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(fmt.Sprintf(`{
		"mdm": {
			"apple_bm_default_team": %q
		}
	}`, tm.Name)), http.StatusOK, &acResp)
	require.Equal(t, tm.Name, acResp.MDM.AppleBMDefaultTeam)

	// get the appconfig, set to that team name
	acResp = appConfigResponse{}
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	require.Equal(t, tm.Name, acResp.MDM.AppleBMDefaultTeam)
}

func (s *integrationEnterpriseTestSuite) TestMDMMacOSUpdates() {
	t := s.T()

	// keep the last activity, to detect newly created ones
	var activitiesResp listActivitiesResponse
	s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &activitiesResp, "order_key", "a.id", "order_direction", "desc")
	var lastActivity uint
	if len(activitiesResp.Activities) > 0 {
		lastActivity = activitiesResp.Activities[0].ID
	}

	checkInvalidConfig := func(config string) {
		// try to set an invalid config
		acResp := appConfigResponse{}
		s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(config), http.StatusUnprocessableEntity, &acResp)

		// get the appconfig, nothing changed
		acResp = appConfigResponse{}
		s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
		require.Equal(t, fleet.MacOSUpdates{MinimumVersion: optjson.String{Set: true}, Deadline: optjson.String{Set: true}}, acResp.MDM.MacOSUpdates)

		// no activity got created
		activitiesResp = listActivitiesResponse{}
		s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &activitiesResp, "order_key", "a.id", "order_direction", "desc")
		require.Condition(t, func() bool {
			return (lastActivity == 0 && len(activitiesResp.Activities) == 0) ||
				(len(activitiesResp.Activities) > 0 && activitiesResp.Activities[0].ID == lastActivity)
		})
	}

	// missing minimum_version
	checkInvalidConfig(`{"mdm": {
		"macos_updates": {
			"deadline": "2022-01-01"
		}
	}}`)

	// missing deadline
	checkInvalidConfig(`{"mdm": {
		"macos_updates": {
			"minimum_version": "12.1.1"
		}
	}}`)

	// invalid deadline
	checkInvalidConfig(`{"mdm": {
		"macos_updates": {
			"minimum_version": "12.1.1",
			"deadline": "2022"
		}
	}}`)

	// deadline includes timestamp
	checkInvalidConfig(`{"mdm": {
		"macos_updates": {
			"minimum_version": "12.1.1",
			"deadline": "2022-01-01T00:00:00Z"
		}
	}}`)

	// minimum_version includes build info
	checkInvalidConfig(`{"mdm": {
		"macos_updates": {
			"minimum_version": "12.1.1 (ABCD)",
			"deadline": "2022-01-01"
		}
	}}`)

	// valid config
	acResp := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
			"mdm": {
				"macos_updates": {
					"minimum_version": "12.3.1",
					"deadline": "2022-01-01"
				}
			}
		}`), http.StatusOK, &acResp)
	require.Equal(t, "12.3.1", acResp.MDM.MacOSUpdates.MinimumVersion.Value)
	require.Equal(t, "2022-01-01", acResp.MDM.MacOSUpdates.Deadline.Value)

	// edited macos min version activity got created
	s.lastActivityMatches(fleet.ActivityTypeEditedMacOSMinVersion{}.ActivityName(), `{"deadline":"2022-01-01", "minimum_version":"12.3.1", "team_id": null, "team_name": null}`, 0)

	// get the appconfig
	acResp = appConfigResponse{}
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	require.Equal(t, "12.3.1", acResp.MDM.MacOSUpdates.MinimumVersion.Value)
	require.Equal(t, "2022-01-01", acResp.MDM.MacOSUpdates.Deadline.Value)

	// update the deadline
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
			"mdm": {
				"macos_updates": {
					"minimum_version": "12.3.1",
					"deadline": "2024-01-01"
				}
			}
		}`), http.StatusOK, &acResp)
	require.Equal(t, "12.3.1", acResp.MDM.MacOSUpdates.MinimumVersion.Value)
	require.Equal(t, "2024-01-01", acResp.MDM.MacOSUpdates.Deadline.Value)

	// another edited macos min version activity got created
	lastActivity = s.lastActivityMatches(fleet.ActivityTypeEditedMacOSMinVersion{}.ActivityName(), `{"deadline":"2024-01-01", "minimum_version":"12.3.1", "team_id": null, "team_name": null}`, 0)

	// update something unrelated - the transparency url
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{"fleet_desktop":{"transparency_url": "customURL"}}`), http.StatusOK, &acResp)
	require.Equal(t, "12.3.1", acResp.MDM.MacOSUpdates.MinimumVersion.Value)
	require.Equal(t, "2024-01-01", acResp.MDM.MacOSUpdates.Deadline.Value)

	// no activity got created
	s.lastActivityMatches("", ``, lastActivity)

	// clear the macos requirement
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
			"mdm": {
				"macos_updates": {
					"minimum_version": "",
					"deadline": ""
				}
			}
		}`), http.StatusOK, &acResp)
	require.Empty(t, acResp.MDM.MacOSUpdates.MinimumVersion.Value)
	require.Empty(t, acResp.MDM.MacOSUpdates.Deadline.Value)

	// edited macos min version activity got created with empty requirement
	lastActivity = s.lastActivityMatches(fleet.ActivityTypeEditedMacOSMinVersion{}.ActivityName(), `{"deadline":"", "minimum_version":"", "team_id": null, "team_name": null}`, 0)

	// update again with empty macos requirement
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
			"mdm": {
				"macos_updates": {
					"minimum_version": "",
					"deadline": ""
				}
			}
		}`), http.StatusOK, &acResp)
	require.Empty(t, acResp.MDM.MacOSUpdates.MinimumVersion.Value)
	require.Empty(t, acResp.MDM.MacOSUpdates.Deadline.Value)

	// no activity got created
	s.lastActivityMatches("", ``, lastActivity)
}

func (s *integrationEnterpriseTestSuite) TestSSOJITProvisioning() {
	t := s.T()

	acResp := appConfigResponse{}
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	require.NotNil(t, acResp)
	require.False(t, acResp.SSOSettings.EnableJITProvisioning)

	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"sso_settings": {
			"enable_sso": true,
			"entity_id": "https://localhost:8080",
			"issuer_uri": "http://localhost:8080/simplesaml/saml2/idp/SSOService.php",
			"idp_name": "SimpleSAML",
			"metadata_url": "http://localhost:9080/simplesaml/saml2/idp/metadata.php",
			"enable_jit_provisioning": false
		}
	}`), http.StatusOK, &acResp)
	require.NotNil(t, acResp)
	require.False(t, acResp.SSOSettings.EnableJITProvisioning)

	// users can't be created if SSO is disabled
	auth, body := s.LoginSSOUser("sso_user", "user123#")
	require.Contains(t, body, "/login?status=account_invalid")
	// ensure theresn't a user in the DB
	_, err := s.ds.UserByEmail(context.Background(), auth.UserID())
	var nfe fleet.NotFoundError
	require.ErrorAs(t, err, &nfe)

	// If enable_jit_provisioning is enabled Roles won't be updated for existing SSO users.

	// enable JIT provisioning
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"sso_settings": {
			"enable_sso": true,
			"entity_id": "https://localhost:8080",
			"issuer_uri": "http://localhost:8080/simplesaml/saml2/idp/SSOService.php",
			"idp_name": "SimpleSAML",
			"metadata_url": "http://localhost:9080/simplesaml/saml2/idp/metadata.php",
			"enable_jit_provisioning": true
		}
	}`), http.StatusOK, &acResp)
	require.NotNil(t, acResp)
	require.True(t, acResp.SSOSettings.EnableJITProvisioning)

	// a new user is created and redirected accordingly
	auth, body = s.LoginSSOUser("sso_user", "user123#")
	// a successful redirect has this content
	require.Contains(t, body, "Redirecting to Fleet at  ...")
	user, err := s.ds.UserByEmail(context.Background(), auth.UserID())
	require.NoError(t, err)
	require.Equal(t, auth.UserID(), user.Email)

	// a new activity item is created
	activitiesResp := listActivitiesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &activitiesResp)
	require.NoError(t, activitiesResp.Err)
	require.NotEmpty(t, activitiesResp.Activities)
	require.Condition(t, func() bool {
		for _, a := range activitiesResp.Activities {
			if (a.Type == fleet.ActivityTypeUserAddedBySSO{}.ActivityName()) && *a.ActorEmail == auth.UserID() {
				return true
			}
		}
		return false
	})

	// Test that roles are not updated for an existing user when SSO attributes are not set.

	// Change role to global admin first.
	user.GlobalRole = ptr.String("admin")
	err = s.ds.SaveUser(context.Background(), user)
	require.NoError(t, err)
	// Login should NOT change the role to the default (global observer) because SSO attributes
	// are not set for this user (see ../../tools/saml/users.php).
	auth, body = s.LoginSSOUser("sso_user", "user123#")
	assert.Equal(t, "sso_user@example.com", auth.UserID())
	assert.Equal(t, "SSO User 1", auth.UserDisplayName())
	require.Contains(t, body, "Redirecting to Fleet at  ...")
	user, err = s.ds.UserByEmail(context.Background(), "sso_user@example.com")
	require.NoError(t, err)
	require.NotNil(t, user.GlobalRole)
	require.Equal(t, *user.GlobalRole, "admin")

	// A user with pre-configured roles can be created
	// see `tools/saml/users.php` for details.
	auth, body = s.LoginSSOUser("sso_user_3_global_admin", "user123#")
	assert.Equal(t, "sso_user_3_global_admin@example.com", auth.UserID())
	assert.Equal(t, "SSO User 3", auth.UserDisplayName())
	assert.Contains(t, auth.AssertionAttributes(), fleet.SAMLAttribute{
		Name: "FLEET_JIT_USER_ROLE_GLOBAL",
		Values: []fleet.SAMLAttributeValue{{
			Value: "admin",
		}},
	})
	require.Contains(t, body, "Redirecting to Fleet at  ...")

	// Test that roles are updated for an existing user when SSO attributes are set.

	// Change role to global maintainer first.
	user3, err := s.ds.UserByEmail(context.Background(), auth.UserID())
	require.NoError(t, err)
	require.Equal(t, auth.UserID(), user3.Email)
	user3.GlobalRole = ptr.String("maintainer")
	err = s.ds.SaveUser(context.Background(), user3)
	require.NoError(t, err)

	// Login should change the role to the configured role in the SSO attributes (global admin).
	auth, body = s.LoginSSOUser("sso_user_3_global_admin", "user123#")
	assert.Equal(t, "sso_user_3_global_admin@example.com", auth.UserID())
	assert.Equal(t, "SSO User 3", auth.UserDisplayName())
	require.Contains(t, body, "Redirecting to Fleet at  ...")
	user3, err = s.ds.UserByEmail(context.Background(), "sso_user_3_global_admin@example.com")
	require.NoError(t, err)
	require.NotNil(t, user3.GlobalRole)
	require.Equal(t, *user3.GlobalRole, "admin")

	// We cannot use NewTeam and must use adhoc SQL because the teams.id is
	// auto-incremented and other tests cause it to be different than what we need (ID=1).
	var execErr error
	mysql.ExecAdhocSQL(t, s.ds, func(db sqlx.ExtContext) error {
		_, execErr = db.ExecContext(context.Background(), `INSERT INTO teams (id, name) VALUES (1, 'Foobar') ON DUPLICATE KEY UPDATE name = VALUES(name);`)
		return execErr
	})
	require.NoError(t, execErr)

	// Create a team for the test below.
	_, err = s.ds.NewTeam(context.Background(), &fleet.Team{
		Name:        "team_" + t.Name(),
		Description: "desc team_" + t.Name(),
	})
	require.NoError(t, err)

	// A user with pre-configured roles can be created,
	// see `tools/saml/users.php` for details.
	auth, body = s.LoginSSOUser("sso_user_4_team_maintainer", "user123#")
	assert.Equal(t, "sso_user_4_team_maintainer@example.com", auth.UserID())
	assert.Equal(t, "SSO User 4", auth.UserDisplayName())
	assert.Contains(t, auth.AssertionAttributes(), fleet.SAMLAttribute{
		Name: "FLEET_JIT_USER_ROLE_TEAM_1",
		Values: []fleet.SAMLAttributeValue{{
			Value: "maintainer",
		}},
	})
	require.Contains(t, body, "Redirecting to Fleet at  ...")

	// A user with pre-configured roles can be created,
	// see `tools/saml/users.php` for details.
	auth, body = s.LoginSSOUser("sso_user_5_team_admin", "user123#")
	assert.Equal(t, "sso_user_5_team_admin@example.com", auth.UserID())
	assert.Equal(t, "SSO User 5", auth.UserDisplayName())
	assert.Contains(t, auth.AssertionAttributes(), fleet.SAMLAttribute{
		Name: "FLEET_JIT_USER_ROLE_TEAM_1",
		Values: []fleet.SAMLAttributeValue{{
			Value: "admin",
		}},
	})
	// FLEET_JIT_USER_ROLE_* attributes with value `null` are ignored by Fleet.
	assert.Contains(t, auth.AssertionAttributes(), fleet.SAMLAttribute{
		Name: "FLEET_JIT_USER_ROLE_GLOBAL",
		Values: []fleet.SAMLAttributeValue{{
			Value: "null",
		}},
	})
	// FLEET_JIT_USER_ROLE_* attributes with value `null` are ignored by Fleet.
	assert.Contains(t, auth.AssertionAttributes(), fleet.SAMLAttribute{
		Name: "FLEET_JIT_USER_ROLE_TEAM_2",
		Values: []fleet.SAMLAttributeValue{{
			Value: "null",
		}},
	})
	require.Contains(t, body, "Redirecting to Fleet at  ...")

	// A user with pre-configured roles can be created,
	// see `tools/saml/users.php` for details.
	auth, body = s.LoginSSOUser("sso_user_6_global_observer", "user123#")
	assert.Equal(t, "sso_user_6_global_observer@example.com", auth.UserID())
	assert.Equal(t, "SSO User 6", auth.UserDisplayName())
	// FLEET_JIT_USER_ROLE_* attributes with value `null` are ignored by Fleet.
	assert.Contains(t, auth.AssertionAttributes(), fleet.SAMLAttribute{
		Name: "FLEET_JIT_USER_ROLE_GLOBAL",
		Values: []fleet.SAMLAttributeValue{{
			Value: "null",
		}},
	})
	// FLEET_JIT_USER_ROLE_* attributes with value `null` are ignored by Fleet.
	assert.Contains(t, auth.AssertionAttributes(), fleet.SAMLAttribute{
		Name: "FLEET_JIT_USER_ROLE_TEAM_1",
		Values: []fleet.SAMLAttributeValue{{
			Value: "null",
		}},
	})
	require.Contains(t, body, "Redirecting to Fleet at  ...")
}

func (s *integrationEnterpriseTestSuite) TestDistributedReadWithFeatures() {
	t := s.T()

	// Global config has both features enabled
	spec := []byte(`
  features:
    additional_queries: null
    enable_host_users: true
    enable_software_inventory: true
`)
	s.applyConfig(spec)

	// Team config has only additional queries enabled
	a := json.RawMessage(`{"time": "SELECT * FROM time"}`)
	team, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		ID:          8324,
		Name:        "team1_" + t.Name(),
		Description: "desc team1_" + t.Name(),
		Config: fleet.TeamConfig{
			Features: fleet.Features{
				EnableHostUsers:         false,
				EnableSoftwareInventory: false,
				AdditionalQueries:       &a,
			},
		},
	})
	require.NoError(t, err)

	// Create a host without a team
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

	s.lq.On("QueriesForHost", host.ID).Return(map[string]string{fmt.Sprintf("%d", host.ID): "select 1 from osquery;"}, nil)

	// ensure we can read distributed queries for the host
	err = s.ds.UpdateHostRefetchRequested(context.Background(), host.ID, true)
	require.NoError(t, err)

	// get distributed queries for the host
	req := getDistributedQueriesRequest{NodeKey: *host.NodeKey}
	var dqResp getDistributedQueriesResponse
	s.DoJSON("POST", "/api/osquery/distributed/read", req, http.StatusOK, &dqResp)
	require.Contains(t, dqResp.Queries, "fleet_detail_query_users")
	require.Contains(t, dqResp.Queries, "fleet_detail_query_software_macos")
	require.NotContains(t, dqResp.Queries, "fleet_additional_query_time")

	// add the host to team1
	err = s.ds.AddHostsToTeam(context.Background(), &team.ID, []uint{host.ID})
	require.NoError(t, err)

	err = s.ds.UpdateHostRefetchRequested(context.Background(), host.ID, true)
	require.NoError(t, err)
	req = getDistributedQueriesRequest{NodeKey: *host.NodeKey}
	dqResp = getDistributedQueriesResponse{}
	s.DoJSON("POST", "/api/osquery/distributed/read", req, http.StatusOK, &dqResp)
	require.NotContains(t, dqResp.Queries, "fleet_detail_query_users")
	require.NotContains(t, dqResp.Queries, "fleet_detail_query_software_macos")
	require.Contains(t, dqResp.Queries, "fleet_additional_query_time")
}

func (s *integrationEnterpriseTestSuite) TestListHosts() {
	t := s.T()

	// create a couple of hosts
	host1, err := s.ds.NewHost(context.Background(), &fleet.Host{
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
	host2, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   ptr.String(t.Name() + "2"),
		NodeKey:         ptr.String(t.Name() + "2"),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%sbar.local", t.Name()),
		Platform:        "linux",
	})
	require.NoError(t, err)
	host3, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   ptr.String(t.Name() + "3"),
		NodeKey:         ptr.String(t.Name() + "3"),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%sbaz.local", t.Name()),
		Platform:        "windows",
	})
	require.NoError(t, err)
	require.NotNil(t, host3)

	// set disk space information for some hosts (none provided for host3)
	require.NoError(t, s.ds.SetOrUpdateHostDisksSpace(context.Background(), host1.ID, 10.0, 2.0))
	require.NoError(t, s.ds.SetOrUpdateHostDisksSpace(context.Background(), host2.ID, 40.0, 4.0))

	var resp listHostsResponse
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp)
	require.Len(t, resp.Hosts, 3)

	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "low_disk_space", "32")
	require.Len(t, resp.Hosts, 1)
	assert.Equal(t, host1.ID, resp.Hosts[0].ID)
	assert.Equal(t, 10.0, resp.Hosts[0].GigsDiskSpaceAvailable)

	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "low_disk_space", "100")
	require.Len(t, resp.Hosts, 2)

	// returns an error when the criteria is invalid (outside 1-100)
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusInternalServerError, &resp, "low_disk_space", "101") // TODO: status code to be fixed with #4406
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusInternalServerError, &resp, "low_disk_space", "0")   // TODO: status code to be fixed with #4406

	// counting hosts works with and without the filter too
	var countResp countHostsResponse
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp)
	require.Equal(t, 3, countResp.Count)
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp, "low_disk_space", "32")
	require.Equal(t, 1, countResp.Count)
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp, "low_disk_space", "100")
	require.Equal(t, 2, countResp.Count)

	// host summary returns counts for low disk space
	var summaryResp getHostSummaryResponse
	s.DoJSON("GET", "/api/latest/fleet/host_summary", nil, http.StatusOK, &summaryResp, "low_disk_space", "32")
	require.Equal(t, uint(3), summaryResp.TotalsHostsCount)
	require.NotNil(t, summaryResp.LowDiskSpaceCount)
	require.Equal(t, uint(1), *summaryResp.LowDiskSpaceCount)

	summaryResp = getHostSummaryResponse{}
	s.DoJSON("GET", "/api/latest/fleet/host_summary", nil, http.StatusOK, &summaryResp, "platform", "windows", "low_disk_space", "32")
	require.Equal(t, uint(1), summaryResp.TotalsHostsCount)
	require.NotNil(t, summaryResp.LowDiskSpaceCount)
	require.Equal(t, uint(0), *summaryResp.LowDiskSpaceCount)

	// all possible filters
	summaryResp = getHostSummaryResponse{}
	s.DoJSON("GET", "/api/latest/fleet/host_summary", nil, http.StatusOK, &summaryResp, "team_id", "1", "platform", "linux", "low_disk_space", "32")
	require.Equal(t, uint(0), summaryResp.TotalsHostsCount)
	require.NotNil(t, summaryResp.LowDiskSpaceCount)
	require.Equal(t, uint(0), *summaryResp.LowDiskSpaceCount)

	// without low_disk_space, does not return the count
	summaryResp = getHostSummaryResponse{}
	s.DoJSON("GET", "/api/latest/fleet/host_summary", nil, http.StatusOK, &summaryResp, "team_id", "1", "platform", "linux")
	require.Equal(t, uint(0), summaryResp.TotalsHostsCount)
	require.Nil(t, summaryResp.LowDiskSpaceCount)
}

func (s *integrationEnterpriseTestSuite) TestAppleMDMNotConfigured() {
	t := s.T()

	// create a host with device token to test device authenticated routes
	tkn := "D3V1C370K3N"
	createHostAndDeviceToken(t, s.ds, tkn)

	for _, route := range mdmAppleConfigurationRequiredEndpoints() {
		var expectedErr fleet.ErrWithStatusCode = fleet.ErrMDMNotConfigured
		path := route.path
		if route.deviceAuthenticated {
			path = fmt.Sprintf(route.path, tkn)
		}

		res := s.Do(route.method, path, nil, expectedErr.StatusCode())
		errMsg := extractServerErrorText(res.Body)
		assert.Contains(t, errMsg, expectedErr.Error())
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

func (s *integrationEnterpriseTestSuite) TestGlobalPolicyCreateReadPatch() {
	fields := []string{"Query", "Name", "Description", "Resolution", "Platform", "Critical"}

	createPol1 := &globalPolicyResponse{}
	createPol1Req := &globalPolicyRequest{
		Query:       "query",
		Name:        "name1",
		Description: "description",
		Resolution:  "resolution",
		Platform:    "linux",
		Critical:    true,
	}
	s.DoJSON("POST", "/api/latest/fleet/policies", createPol1Req, http.StatusOK, &createPol1)
	allEqual(s.T(), createPol1Req, createPol1.Policy, fields...)

	createPol2 := &globalPolicyResponse{}
	createPol2Req := &globalPolicyRequest{
		Query:       "query",
		Name:        "name2",
		Description: "description",
		Resolution:  "resolution",
		Platform:    "linux",
		Critical:    false,
	}
	s.DoJSON("POST", "/api/latest/fleet/policies", createPol2Req, http.StatusOK, &createPol2)
	allEqual(s.T(), createPol2Req, createPol2.Policy, fields...)

	listPol := &listGlobalPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/policies", nil, http.StatusOK, listPol)
	require.Len(s.T(), listPol.Policies, 2)
	sort.Slice(listPol.Policies, func(i, j int) bool {
		return listPol.Policies[i].Name < listPol.Policies[j].Name
	})
	require.Equal(s.T(), createPol1.Policy, listPol.Policies[0])
	require.Equal(s.T(), createPol2.Policy, listPol.Policies[1])

	patchPol1Req := &modifyGlobalPolicyRequest{
		ModifyPolicyPayload: fleet.ModifyPolicyPayload{
			Name:        ptr.String("newName1"),
			Query:       ptr.String("newQuery"),
			Description: ptr.String("newDescription"),
			Resolution:  ptr.String("newResolution"),
			Platform:    ptr.String("windows"),
			Critical:    ptr.Bool(false),
		},
	}
	patchPol1 := &modifyGlobalPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/policies/%d", createPol1.Policy.ID), patchPol1Req, http.StatusOK, patchPol1)
	allEqual(s.T(), patchPol1Req, patchPol1.Policy, fields...)

	patchPol2Req := &modifyGlobalPolicyRequest{
		ModifyPolicyPayload: fleet.ModifyPolicyPayload{
			Name:        ptr.String("newName2"),
			Query:       ptr.String("newQuery"),
			Description: ptr.String("newDescription"),
			Resolution:  ptr.String("newResolution"),
			Platform:    ptr.String("windows"),
			Critical:    ptr.Bool(true),
		},
	}
	patchPol2 := &modifyGlobalPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/policies/%d", createPol2.Policy.ID), patchPol2Req, http.StatusOK, patchPol2)
	allEqual(s.T(), patchPol2Req, patchPol2.Policy, fields...)

	listPol = &listGlobalPoliciesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/policies", nil, http.StatusOK, listPol)
	require.Len(s.T(), listPol.Policies, 2)
	sort.Slice(listPol.Policies, func(i, j int) bool {
		return listPol.Policies[i].Name < listPol.Policies[j].Name
	})
	// not using require.Equal because "PATCH policies" returns the wrong updated timestamp.
	allEqual(s.T(), patchPol1.Policy, listPol.Policies[0], fields...)
	allEqual(s.T(), patchPol2.Policy, listPol.Policies[1], fields...)

	getPol2 := &getPolicyByIDResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/policies/%d", createPol2.Policy.ID), nil, http.StatusOK, getPol2)
	require.Equal(s.T(), listPol.Policies[1], getPol2.Policy)
}

func (s *integrationEnterpriseTestSuite) TestTeamPolicyCreateReadPatch() {
	fields := []string{"Query", "Name", "Description", "Resolution", "Platform", "Critical"}

	team1, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		ID:          42,
		Name:        "team1",
		Description: "desc team1",
	})
	require.NoError(s.T(), err)

	createPol1 := &teamPolicyResponse{}
	createPol1Req := &teamPolicyRequest{
		Query:       "query",
		Name:        "name1",
		Description: "description",
		Resolution:  "resolution",
		Platform:    "linux",
		Critical:    true,
	}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team1.ID), createPol1Req, http.StatusOK, &createPol1)
	allEqual(s.T(), createPol1Req, createPol1.Policy, fields...)

	createPol2 := &teamPolicyResponse{}
	createPol2Req := &teamPolicyRequest{
		Query:       "query",
		Name:        "name2",
		Description: "description",
		Resolution:  "resolution",
		Platform:    "linux",
		Critical:    false,
	}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team1.ID), createPol2Req, http.StatusOK, &createPol2)
	allEqual(s.T(), createPol2Req, createPol2.Policy, fields...)

	listPol := &listTeamPoliciesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team1.ID), nil, http.StatusOK, listPol)
	require.Len(s.T(), listPol.Policies, 2)
	sort.Slice(listPol.Policies, func(i, j int) bool {
		return listPol.Policies[i].Name < listPol.Policies[j].Name
	})
	require.Equal(s.T(), createPol1.Policy, listPol.Policies[0])
	require.Equal(s.T(), createPol2.Policy, listPol.Policies[1])

	patchPol1Req := &modifyTeamPolicyRequest{
		ModifyPolicyPayload: fleet.ModifyPolicyPayload{
			Name:        ptr.String("newName1"),
			Query:       ptr.String("newQuery"),
			Description: ptr.String("newDescription"),
			Resolution:  ptr.String("newResolution"),
			Platform:    ptr.String("windows"),
			Critical:    ptr.Bool(false),
		},
	}
	patchPol1 := &modifyTeamPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team1.ID, createPol1.Policy.ID), patchPol1Req, http.StatusOK, patchPol1)
	allEqual(s.T(), patchPol1Req, patchPol1.Policy, fields...)

	patchPol2Req := &modifyTeamPolicyRequest{
		ModifyPolicyPayload: fleet.ModifyPolicyPayload{
			Name:        ptr.String("newName2"),
			Query:       ptr.String("newQuery"),
			Description: ptr.String("newDescription"),
			Resolution:  ptr.String("newResolution"),
			Platform:    ptr.String("windows"),
			Critical:    ptr.Bool(true),
		},
	}
	patchPol2 := &modifyTeamPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team1.ID, createPol2.Policy.ID), patchPol2Req, http.StatusOK, patchPol2)
	allEqual(s.T(), patchPol2Req, patchPol2.Policy, fields...)

	listPol = &listTeamPoliciesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team1.ID), nil, http.StatusOK, listPol)
	require.Len(s.T(), listPol.Policies, 2)
	sort.Slice(listPol.Policies, func(i, j int) bool {
		return listPol.Policies[i].Name < listPol.Policies[j].Name
	})
	// not using require.Equal because "PATCH policies" returns the wrong updated timestamp.
	allEqual(s.T(), patchPol1.Policy, listPol.Policies[0], fields...)
	allEqual(s.T(), patchPol2.Policy, listPol.Policies[1], fields...)

	getPol2 := &getPolicyByIDResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team1.ID, createPol2.Policy.ID), nil, http.StatusOK, getPol2)
	require.Equal(s.T(), listPol.Policies[1], getPol2.Policy)
}

func (s *integrationEnterpriseTestSuite) TestResetAutomation() {
	ctx := context.Background()

	team1, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		ID:          42,
		Name:        "team1",
		Description: "desc team1",
	})
	require.NoError(s.T(), err)

	createPol1 := &teamPolicyResponse{}
	createPol1Req := &teamPolicyRequest{
		Query:       "query",
		Name:        "name1",
		Description: "description",
		Resolution:  "resolution",
		Platform:    "linux",
		Critical:    true,
	}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team1.ID), createPol1Req, http.StatusOK, &createPol1)

	createPol2 := &teamPolicyResponse{}
	createPol2Req := &teamPolicyRequest{
		Query:       "query",
		Name:        "name2",
		Description: "description",
		Resolution:  "resolution",
		Platform:    "linux",
		Critical:    false,
	}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team1.ID), createPol2Req, http.StatusOK, &createPol2)

	createPol3 := &teamPolicyResponse{}
	createPol3Req := &teamPolicyRequest{
		Query:       "query",
		Name:        "name3",
		Description: "description",
		Resolution:  "resolution",
		Platform:    "linux",
		Critical:    false,
	}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", team1.ID), createPol3Req, http.StatusOK, &createPol3)

	var tmResp teamResponse
	// modify the team's config - enable the webhook
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team1.ID), fleet.TeamPayload{WebhookSettings: &fleet.TeamWebhookSettings{
		FailingPoliciesWebhook: fleet.FailingPoliciesWebhookSettings{
			Enable:         true,
			DestinationURL: "http://127/",
			PolicyIDs:      []uint{createPol1.Policy.ID, createPol2.Policy.ID},
			HostBatchSize:  12345,
		},
	}}, http.StatusOK, &tmResp)

	h1, err := s.ds.NewHost(ctx, &fleet.Host{})
	require.NoError(s.T(), err)

	err = s.ds.RecordPolicyQueryExecutions(ctx, h1, map[uint]*bool{
		createPol1.Policy.ID: ptr.Bool(false),
		createPol2.Policy.ID: ptr.Bool(false),
		createPol3.Policy.ID: ptr.Bool(false), // This policy is not activated for automation in config.
	}, time.Now(), false)
	require.NoError(s.T(), err)

	pfs, err := s.ds.OutdatedAutomationBatch(ctx)
	require.NoError(s.T(), err)
	require.Empty(s.T(), pfs)

	s.DoJSON("POST", "/api/latest/fleet/automations/reset", resetAutomationRequest{
		TeamIDs:   nil,
		PolicyIDs: []uint{},
	}, http.StatusOK, &tmResp)

	pfs, err = s.ds.OutdatedAutomationBatch(ctx)
	require.NoError(s.T(), err)
	require.Empty(s.T(), pfs)

	s.DoJSON("POST", "/api/latest/fleet/automations/reset", resetAutomationRequest{
		TeamIDs:   nil,
		PolicyIDs: []uint{createPol1.Policy.ID, createPol2.Policy.ID, createPol3.Policy.ID},
	}, http.StatusOK, &tmResp)

	pfs, err = s.ds.OutdatedAutomationBatch(ctx)
	require.NoError(s.T(), err)
	require.Len(s.T(), pfs, 2)

	s.DoJSON("POST", "/api/latest/fleet/automations/reset", resetAutomationRequest{
		TeamIDs:   []uint{team1.ID},
		PolicyIDs: nil,
	}, http.StatusOK, &tmResp)

	pfs, err = s.ds.OutdatedAutomationBatch(ctx)
	require.NoError(s.T(), err)
	require.Len(s.T(), pfs, 2)

	s.DoJSON("POST", "/api/latest/fleet/automations/reset", resetAutomationRequest{
		TeamIDs:   nil,
		PolicyIDs: []uint{createPol2.Policy.ID},
	}, http.StatusOK, &tmResp)

	pfs, err = s.ds.OutdatedAutomationBatch(ctx)
	require.NoError(s.T(), err)
	require.Len(s.T(), pfs, 1)
}

// allEqual compares all fields of a struct.
// If a field is a pointer on one side but not on the other, then it follows that pointer. This is useful for optional
// arguments.
func allEqual(t *testing.T, expect, actual interface{}, fields ...string) {
	require.NotEmpty(t, fields)
	t.Helper()
	expV := reflect.Indirect(reflect.ValueOf(expect))
	actV := reflect.Indirect(reflect.ValueOf(actual))
	for _, f := range fields {
		e, a := expV.FieldByName(f), actV.FieldByName(f)
		switch {
		case e.Kind() == reflect.Ptr && a.Kind() != reflect.Ptr && !e.IsZero():
			e = e.Elem()
		case a.Kind() == reflect.Ptr && e.Kind() != reflect.Ptr && !a.IsZero():
			a = a.Elem()
		}
		require.Equal(t, e.Interface(), a.Interface(), "%s", f)
	}
}

func createHostAndDeviceToken(t *testing.T, ds *mysql.Datastore, token string) *fleet.Host {
	host, err := ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   ptr.String(t.Name()),
		NodeKey:         ptr.String(t.Name()),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%sfoo.local", t.Name()),
		HardwareSerial:  uuid.New().String(),
		Platform:        "darwin",
	})
	require.NoError(t, err)

	createDeviceTokenForHost(t, ds, host.ID, token)

	return host
}

func createDeviceTokenForHost(t *testing.T, ds *mysql.Datastore, hostID uint, token string) {
	mysql.ExecAdhocSQL(t, ds, func(db sqlx.ExtContext) error {
		_, err := db.ExecContext(context.Background(), `INSERT INTO host_device_auth (host_id, token) VALUES (?, ?)`, hostID, token)
		return err
	})
}

func (s *integrationEnterpriseTestSuite) TestListSoftware() {
	t := s.T()
	now := time.Now().UTC().Truncate(time.Second)
	ctx := context.Background()

	host, err := s.ds.NewHost(ctx, &fleet.Host{
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

	software := []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
		{Name: "bar", Version: "0.0.3", Source: "apps"},
	}
	_, err = s.ds.UpdateHostSoftware(ctx, host.ID, software)
	require.NoError(t, err)
	require.NoError(t, s.ds.LoadHostSoftware(ctx, host, false))

	bar := host.Software[0]
	if bar.Name != "bar" {
		bar = host.Software[1]
	}

	inserted, err := s.ds.InsertSoftwareVulnerability(
		ctx, fleet.SoftwareVulnerability{
			SoftwareID: bar.ID,
			CVE:        "cve-123",
		}, fleet.NVDSource,
	)
	require.NoError(t, err)
	require.True(t, inserted)

	require.NoError(t, s.ds.InsertCVEMeta(ctx, []fleet.CVEMeta{{
		CVE:              "cve-123",
		CVSSScore:        ptr.Float64(5.4),
		EPSSProbability:  ptr.Float64(0.5),
		CISAKnownExploit: ptr.Bool(true),
		Published:        &now,
	}}))

	require.NoError(t, s.ds.SyncHostsSoftware(ctx, time.Now().UTC()))

	var resp listSoftwareResponse
	s.DoJSON("GET", "/api/latest/fleet/software", nil, http.StatusOK, &resp)
	require.NotNil(t, resp)

	var fooPayload, barPayload fleet.Software
	for _, s := range resp.Software {
		switch s.Name {
		case "foo":
			fooPayload = s
		case "bar":
			barPayload = s
		default:
			require.Failf(t, "unrecognized software %s", s.Name)

		}
	}

	require.Empty(t, fooPayload.Vulnerabilities)
	require.Len(t, barPayload.Vulnerabilities, 1)
	require.Equal(t, barPayload.Vulnerabilities[0].CVE, "cve-123")
	require.NotNil(t, barPayload.Vulnerabilities[0].CVSSScore, ptr.Float64Ptr(5.4))
	require.NotNil(t, barPayload.Vulnerabilities[0].EPSSProbability, ptr.Float64Ptr(0.5))
	require.NotNil(t, barPayload.Vulnerabilities[0].CISAKnownExploit, ptr.BoolPtr(true))
	require.Equal(t, barPayload.Vulnerabilities[0].CVEPublished, ptr.TimePtr(now))
}

// TestGitOpsUserActions tests the permissions listed in ../../docs/Using-Fleet/Permissions.md.
func (s *integrationEnterpriseTestSuite) TestGitOpsUserActions() {
	t := s.T()
	ctx := context.Background()

	//
	// Setup test data.
	// All setup actions are authored by a global admin.
	//

	admin, err := s.ds.UserByEmail(ctx, "admin1@example.com")
	require.NoError(t, err)
	h1, err := s.ds.NewHost(ctx, &fleet.Host{
		NodeKey:  ptr.String(t.Name() + "1"),
		UUID:     t.Name() + "1",
		Hostname: t.Name() + "foo.local",
	})
	require.NoError(t, err)
	t1, err := s.ds.NewTeam(ctx, &fleet.Team{
		Name: "Foo",
	})
	require.NoError(t, err)
	t2, err := s.ds.NewTeam(ctx, &fleet.Team{
		Name: "Bar",
	})
	require.NoError(t, err)
	t3, err := s.ds.NewTeam(ctx, &fleet.Team{
		Name: "Zoo",
	})
	require.NoError(t, err)
	acr := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"webhook_settings": {
			"vulnerabilities_webhook": {
				"enable_vulnerabilities_webhook": false
			}
		}
	}`), http.StatusOK, &acr)
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acr)
	require.False(t, acr.WebhookSettings.VulnerabilitiesWebhook.Enable)
	q1, err := s.ds.NewQuery(ctx, &fleet.Query{
		Name:  "Foo",
		Query: "SELECT * from time;",
	})
	require.NoError(t, err)
	ggsr := getGlobalScheduleResponse{}
	s.DoJSON("GET", "/api/latest/fleet/schedule", nil, http.StatusOK, &ggsr)
	require.NoError(t, ggsr.Err)
	cpar := createPackResponse{}
	var userPackID uint
	s.DoJSON("POST", "/api/latest/fleet/packs", createPackRequest{
		PackPayload: fleet.PackPayload{
			Name:     ptr.String("Foobar"),
			Disabled: ptr.Bool(false),
		},
	}, http.StatusOK, &cpar)
	userPackID = cpar.Pack.Pack.ID
	require.NotZero(t, userPackID)
	cur := createUserResponse{}
	s.DoJSON("POST", "/api/latest/fleet/users/admin", createUserRequest{
		UserPayload: fleet.UserPayload{
			Email:      ptr.String("foo42@example.com"),
			Password:   ptr.String("p4ssw0rd.123"),
			Name:       ptr.String("foo42"),
			GlobalRole: ptr.String("maintainer"),
		},
	}, http.StatusOK, &cur)
	maintainer := cur.User
	var carveBeginResp carveBeginResponse
	s.DoJSON("POST", "/api/osquery/carve/begin", carveBeginRequest{
		NodeKey:    *h1.NodeKey,
		BlockCount: 3,
		BlockSize:  3,
		CarveSize:  8,
		CarveId:    "c1",
		RequestId:  "r1",
	}, http.StatusOK, &carveBeginResp)
	require.NotEmpty(t, carveBeginResp.SessionId)
	lcr := listCarvesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/carves", listCarvesRequest{}, http.StatusOK, &lcr)
	require.NotEmpty(t, lcr.Carves)
	carveID := lcr.Carves[0].ID
	// Create the global GitOps user we'll use in tests.
	u := &fleet.User{
		Name:       "GitOps",
		Email:      "gitops1@example.com",
		GlobalRole: ptr.String(fleet.RoleGitOps),
	}
	require.NoError(t, u.SetPassword(test.GoodPassword, 10, 10))
	_, err = s.ds.NewUser(context.Background(), u)
	require.NoError(t, err)
	// Create a GitOps user for team t1 we'll use in tests.
	u2 := &fleet.User{
		Name:       "GitOps 2",
		Email:      "gitops2@example.com",
		GlobalRole: nil,
		Teams: []fleet.UserTeam{
			{
				Team: *t1,
				Role: fleet.RoleGitOps,
			},
			{
				Team: *t3,
				Role: fleet.RoleGitOps,
			},
		},
	}
	require.NoError(t, u2.SetPassword(test.GoodPassword, 10, 10))
	_, err = s.ds.NewUser(context.Background(), u2)
	require.NoError(t, err)
	gp2, err := s.ds.NewGlobalPolicy(ctx, &admin.ID, fleet.PolicyPayload{
		Name:  "Zoo",
		Query: "SELECT 0;",
	})
	require.NoError(t, err)
	t2p, err := s.ds.NewTeamPolicy(ctx, t2.ID, &admin.ID, fleet.PolicyPayload{
		Name:  "Zoo2",
		Query: "SELECT 2;",
	})
	require.NoError(t, err)
	// Create some test user to test moving from/to teams.
	u3 := &fleet.User{
		Name:       "Test Foo Observer",
		Email:      "test-foo-observer@example.com",
		GlobalRole: nil,
		Teams: []fleet.UserTeam{
			{
				Team: *t1,
				Role: fleet.RoleObserver,
			},
		},
	}
	require.NoError(t, u3.SetPassword(test.GoodPassword, 10, 10))
	_, err = s.ds.NewUser(context.Background(), u3)
	require.NoError(t, err)

	//
	// Start running permission tests with user gitops1.
	//
	s.setTokenForTest(t, "gitops1@example.com", test.GoodPassword)

	// Attempt to retrieve activities, should fail.
	s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusForbidden, &listActivitiesResponse{})

	// Attempt to retrieve hosts, should fail.
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusForbidden, &listHostsResponse{})

	// Attempt to filter hosts using labels, should fail (label ID 6 is the builtin label "All Hosts")
	s.DoJSON("GET", "/api/latest/fleet/labels/6/hosts", nil, http.StatusForbidden, &listHostsResponse{})

	// Attempt to delete hosts, should fail.
	s.DoJSON("DELETE", "/api/latest/fleet/hosts/1", nil, http.StatusForbidden, &deleteHostResponse{})

	// Attempt to transfer host from global to a team, should allow.
	s.DoJSON("POST", "/api/latest/fleet/hosts/transfer", addHostsToTeamRequest{
		TeamID:  &t1.ID,
		HostIDs: []uint{h1.ID},
	}, http.StatusOK, &addHostsToTeamResponse{})

	// Attempt to create a label, should allow.
	clr := createLabelResponse{}
	s.DoJSON("POST", "/api/latest/fleet/labels", createLabelRequest{
		LabelPayload: fleet.LabelPayload{
			Name:  ptr.String("foo"),
			Query: ptr.String("SELECT 1;"),
		},
	}, http.StatusOK, &clr)

	// Attempt to modify a label, should allow.
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/labels/%d", clr.Label.ID), modifyLabelRequest{
		ModifyLabelPayload: fleet.ModifyLabelPayload{
			Name: ptr.String("foo2"),
		},
	}, http.StatusOK, &modifyLabelResponse{})

	// Attempt to get a label, should fail.
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/labels/%d", clr.Label.ID), getLabelRequest{}, http.StatusForbidden, &getLabelResponse{})

	// Attempt to list all labels, should fail.
	s.DoJSON("GET", "/api/latest/fleet/labels", listLabelsRequest{}, http.StatusForbidden, &listLabelsResponse{})

	// Attempt to delete a label, should allow.
	s.DoJSON("DELETE", "/api/latest/fleet/labels/foo2", deleteLabelRequest{}, http.StatusOK, &deleteLabelResponse{})

	// Attempt to list all software, should fail.
	s.DoJSON("GET", "/api/latest/fleet/software", listSoftwareRequest{}, http.StatusForbidden, &listSoftwareResponse{})
	s.DoJSON("GET", "/api/latest/fleet/software/count", countSoftwareRequest{}, http.StatusForbidden, &countSoftwareResponse{})

	// Attempt to list a software, should fail.
	s.DoJSON("GET", "/api/latest/fleet/software/1", getSoftwareRequest{}, http.StatusForbidden, &getSoftwareResponse{})

	// Attempt to read app config, should fail.
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusForbidden, &appConfigResponse{})

	// Attempt to write app config, should allow.
	acr = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"webhook_settings": {
			"vulnerabilities_webhook": {
				"enable_vulnerabilities_webhook": true,
				"destination_url": "https://foobar.example.com"
			}
		}
	}`), http.StatusOK, &acr)
	require.True(t, acr.AppConfig.WebhookSettings.VulnerabilitiesWebhook.Enable)
	require.Equal(t, "https://foobar.example.com", acr.AppConfig.WebhookSettings.VulnerabilitiesWebhook.DestinationURL)

	// Attempt to run live queries synchronously, should fail.
	// TODO(lucas): This is a bug, the synchronous live query API should return 403 but currently returns 200.
	// It doesn't run the query but incorrectly returns a 200.
	s.DoJSON("GET", "/api/latest/fleet/queries/run", runLiveQueryRequest{
		HostIDs:  []uint{h1.ID},
		QueryIDs: []uint{q1.ID},
	}, http.StatusOK, &runLiveQueryResponse{})

	// Attempt to run live queries asynchronously (new unsaved query), should fail.
	s.DoJSON("POST", "/api/latest/fleet/queries/run", createDistributedQueryCampaignRequest{
		QuerySQL: "SELECT * FROM time;",
		Selected: fleet.HostTargets{
			HostIDs: []uint{h1.ID},
		},
	}, http.StatusForbidden, &runLiveQueryResponse{})

	// Attempt to run live queries asynchronously (saved query), should fail.
	s.DoJSON("POST", "/api/latest/fleet/queries/run", createDistributedQueryCampaignRequest{
		QueryID: ptr.Uint(q1.ID),
		Selected: fleet.HostTargets{
			HostIDs: []uint{h1.ID},
		},
	}, http.StatusForbidden, &runLiveQueryResponse{})

	// Attempt to create queries, should allow.
	cqr := createQueryResponse{}
	s.DoJSON("POST", "/api/latest/fleet/queries", createQueryRequest{
		QueryPayload: fleet.QueryPayload{
			Name:  ptr.String("foo4"),
			Query: ptr.String("SELECT * from osquery_info;"),
		},
	}, http.StatusOK, &cqr)
	cqr2 := createQueryResponse{}
	s.DoJSON("POST", "/api/latest/fleet/queries", createQueryRequest{
		QueryPayload: fleet.QueryPayload{
			Name:  ptr.String("foo5"),
			Query: ptr.String("SELECT * from os_version;"),
		},
	}, http.StatusOK, &cqr2)
	cqr3 := createQueryResponse{}
	s.DoJSON("POST", "/api/latest/fleet/queries", createQueryRequest{
		QueryPayload: fleet.QueryPayload{
			Name:  ptr.String("foo6"),
			Query: ptr.String("SELECT * from processes;"),
		},
	}, http.StatusOK, &cqr3)
	cqr4 := createQueryResponse{}
	s.DoJSON("POST", "/api/latest/fleet/queries", createQueryRequest{
		QueryPayload: fleet.QueryPayload{
			Name:  ptr.String("foo7"),
			Query: ptr.String("SELECT * from managed_policies;"),
		},
	}, http.StatusOK, &cqr4)

	// Attempt to edit queries, should allow.
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/queries/%d", cqr.Query.ID), modifyQueryRequest{
		QueryPayload: fleet.QueryPayload{
			Name:  ptr.String("foo4"),
			Query: ptr.String("SELECT * FROM system_info;"),
		},
	}, http.StatusOK, &modifyQueryResponse{})

	// Attempt to view a query, should fail.
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d", cqr.Query.ID), getQueryRequest{}, http.StatusForbidden, &getQueryResponse{})

	// Attempt to list all queries, should fail.
	s.DoJSON("GET", "/api/latest/fleet/queries", listQueriesRequest{}, http.StatusForbidden, &listQueriesResponse{})

	// Attempt to delete queries, should allow.
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/queries/id/%d", cqr.Query.ID), deleteQueryByIDRequest{}, http.StatusOK, &deleteQueryByIDResponse{})
	s.DoJSON("POST", "/api/latest/fleet/queries/delete", deleteQueriesRequest{IDs: []uint{cqr2.Query.ID}}, http.StatusOK, &deleteQueriesResponse{})
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/queries/%s", cqr3.Query.Name), deleteQueryRequest{}, http.StatusOK, &deleteQueryResponse{})

	// Attempt to add a query to a user pack, should allow.
	sqr := scheduleQueryResponse{}
	s.DoJSON("POST", "/api/latest/fleet/packs/schedule", scheduleQueryRequest{
		PackID:   userPackID,
		QueryID:  cqr4.Query.ID,
		Interval: 60,
	}, http.StatusOK, &sqr)

	// Attempt to edit a scheduled query in the global schedule, should allow.
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/packs/schedule/%d", sqr.Scheduled.ID), modifyScheduledQueryRequest{
		ScheduledQueryPayload: fleet.ScheduledQueryPayload{
			Interval: ptr.Uint(30),
		},
	}, http.StatusOK, &scheduleQueryResponse{})

	// Attempt to remove a query from the global schedule, should allow.
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/packs/schedule/%d", sqr.Scheduled.ID), deleteScheduledQueryRequest{}, http.StatusOK, &scheduleQueryResponse{})

	// Attempt to read the global schedule, should disallow.
	s.DoJSON("GET", "/api/latest/fleet/schedule", nil, http.StatusForbidden, &getGlobalScheduleResponse{})

	// Attempt to create a pack, should allow.
	cpr := createPackResponse{}
	s.DoJSON("POST", "/api/latest/fleet/packs", createPackRequest{
		PackPayload: fleet.PackPayload{
			Name: ptr.String("foo8"),
		},
	}, http.StatusOK, &cpr)

	// Attempt to edit a pack, should allow.
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/packs/%d", cpr.Pack.ID), modifyPackRequest{
		PackPayload: fleet.PackPayload{
			Name: ptr.String("foo9"),
		},
	}, http.StatusOK, &modifyPackResponse{})

	// Attempt to read a pack, should allow.
	// This is an exception to the "write only" nature of gitops (packs can be viewed by gitops).
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/packs/%d", cpr.Pack.ID), nil, http.StatusOK, &getPackResponse{})

	// Attempt to delete a pack, should allow.
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/packs/id/%d", cpr.Pack.ID), deletePackRequest{}, http.StatusOK, &deletePackResponse{})

	// Attempt to create a global policy, should allow.
	gplr := globalPolicyResponse{}
	s.DoJSON("POST", "/api/latest/fleet/policies", globalPolicyRequest{
		Name:  "foo9",
		Query: "SELECT * from plist;",
	}, http.StatusOK, &gplr)

	// Attempt to edit a global policy, should allow.
	mgplr := modifyGlobalPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/policies/%d", gplr.Policy.ID), modifyGlobalPolicyRequest{
		ModifyPolicyPayload: fleet.ModifyPolicyPayload{
			Query: ptr.String("SELECT * from plist WHERE path = 'foo';"),
		},
	}, http.StatusOK, &mgplr)

	// Attempt to read a global policy, should fail.
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/policies/%d", gplr.Policy.ID), getPolicyByIDRequest{}, http.StatusForbidden, &getPolicyByIDResponse{})

	// Attempt to delete a global policy, should allow.
	s.DoJSON("POST", "/api/latest/fleet/policies/delete", deleteGlobalPoliciesRequest{
		IDs: []uint{gplr.Policy.ID},
	}, http.StatusOK, &deleteGlobalPoliciesResponse{})

	// Attempt to create a team policy, should allow.
	tplr := teamPolicyResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/team/%d/policies", t1.ID), teamPolicyRequest{
		Name:  "foo10",
		Query: "SELECT * from file;",
	}, http.StatusOK, &tplr)

	// Attempt to edit a team policy, should allow.
	mtplr := modifyTeamPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", t1.ID, tplr.Policy.ID), modifyTeamPolicyRequest{
		ModifyPolicyPayload: fleet.ModifyPolicyPayload{
			Query: ptr.String("SELECT * from file WHERE path = 'foo';"),
		},
	}, http.StatusOK, &mtplr)

	// Attempt to view a team policy, should fail.
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/team/%d/policies/%d", t1.ID, tplr.Policy.ID), getTeamPolicyByIDRequest{}, http.StatusForbidden, &getTeamPolicyByIDResponse{})

	// Attempt to delete a team policy, should allow.
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/delete", t1.ID), deleteTeamPoliciesRequest{
		IDs: []uint{tplr.Policy.ID},
	}, http.StatusOK, &deleteTeamPoliciesResponse{})

	// Attempt to create a user, should fail.
	s.DoJSON("POST", "/api/latest/fleet/users/admin", createUserRequest{
		UserPayload: fleet.UserPayload{
			Email:      ptr.String("foo10@example.com"),
			Name:       ptr.String("foo10"),
			GlobalRole: ptr.String("admin"),
		},
	}, http.StatusForbidden, &createUserResponse{})

	// Attempt to modify a user, should fail.
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/users/%d", admin.ID), modifyUserRequest{
		UserPayload: fleet.UserPayload{
			GlobalRole: ptr.String("observer"),
		},
	}, http.StatusForbidden, &modifyUserResponse{})

	// Attempt to view a user, should fail.
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/users/%d", admin.ID), getUserRequest{}, http.StatusForbidden, &getUserResponse{})

	// Attempt to delete a user, should fail.
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/users/%d", admin.ID), deleteUserRequest{}, http.StatusForbidden, &deleteUserResponse{})

	// Attempt to add users to team, should allow.
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/users", t1.ID), modifyTeamUsersRequest{
		Users: []fleet.TeamUser{
			{
				User: *maintainer,
				Role: "admin",
			},
		},
	}, http.StatusOK, &teamResponse{})

	// Attempt to delete users from team, should allow.
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/teams/%d/users", t1.ID), modifyTeamUsersRequest{
		Users: []fleet.TeamUser{
			{
				User: *maintainer,
				Role: "admin",
			},
		},
	}, http.StatusOK, &teamResponse{})

	// Attempt to create a team, should allow.
	tr := teamResponse{}
	s.DoJSON("POST", "/api/latest/fleet/teams", createTeamRequest{
		TeamPayload: fleet.TeamPayload{
			Name: ptr.String("foo11"),
		},
	}, http.StatusOK, &tr)

	// Attempt to edit a team, should allow.
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", tr.Team.ID), modifyTeamRequest{
		TeamPayload: fleet.TeamPayload{
			Name: ptr.String("foo12"),
		},
	}, http.StatusOK, &teamResponse{})

	// Attempt to edit a team's agent options, should allow.
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/agent_options", tr.Team.ID), json.RawMessage(`{
		"config": {
			"options": {
				"aws_debug": true
			}
		}
	}`), http.StatusOK, &teamResponse{})

	// Attempt to view a team, should fail.
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", tr.Team.ID), getTeamRequest{}, http.StatusForbidden, &teamResponse{})

	// Attempt to delete a team, should allow.
	dtr := deleteTeamResponse{}
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/teams/%d", tr.Team.ID), deleteTeamRequest{}, http.StatusOK, &dtr)

	// Attempt to create/edit enroll secrets, should allow.
	s.DoJSON("POST", "/api/latest/fleet/spec/enroll_secret", applyEnrollSecretSpecRequest{
		Spec: &fleet.EnrollSecretSpec{
			Secrets: []*fleet.EnrollSecret{
				{
					Secret: "foo400",
					TeamID: nil,
				},
				{
					Secret: "foo500",
					TeamID: ptr.Uint(t1.ID),
				},
			},
		},
	}, http.StatusOK, &applyEnrollSecretSpecResponse{})

	// Attempt to get enroll secrets, should fail.
	s.DoJSON("GET", "/api/latest/fleet/spec/enroll_secret", nil, http.StatusForbidden, &getEnrollSecretSpecResponse{})

	// Attempt to get team enroll secret, should fail.
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/secrets", t1.ID), teamEnrollSecretsRequest{}, http.StatusForbidden, &teamEnrollSecretsResponse{})

	// Attempt to list carved files, should fail.
	s.DoJSON("GET", "/api/latest/fleet/carves", listCarvesRequest{}, http.StatusForbidden, &listCarvesResponse{})

	// Attempt to get a carved file, should fail.
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/carves/%d", carveID), listCarvesRequest{}, http.StatusForbidden, &listCarvesResponse{})

	// Attempt to search hosts, should fail.
	s.DoJSON("POST", "/api/latest/fleet/targets", searchTargetsRequest{
		MatchQuery: "foo",
		QueryID:    &q1.ID,
	}, http.StatusForbidden, &searchTargetsResponse{})

	// Attempt to count target hosts, should fail.
	s.DoJSON("POST", "/api/latest/fleet/targets/count", countTargetsRequest{
		Selected: fleet.HostTargets{
			HostIDs:  []uint{h1.ID},
			LabelIDs: []uint{clr.Label.ID},
			TeamIDs:  []uint{t1.ID},
		},
		QueryID: &q1.ID,
	}, http.StatusForbidden, &countTargetsResponse{})

	//
	// Start running permission tests with user gitops2 (which is a GitOps use for team t1).
	//

	s.setTokenForTest(t, "gitops2@example.com", test.GoodPassword)

	// Attempt to create queries in global domain, should fail.
	tcqr := createQueryResponse{}
	s.DoJSON("POST", "/api/latest/fleet/queries", createQueryRequest{
		QueryPayload: fleet.QueryPayload{
			Name:  ptr.String("foo600"),
			Query: ptr.String("SELECT * from orbit_info;"),
		},
	}, http.StatusForbidden, &tcqr)

	// Attempt to create queries in its team, should allow.
	tcqr = createQueryResponse{}
	s.DoJSON("POST", "/api/latest/fleet/queries", createQueryRequest{
		QueryPayload: fleet.QueryPayload{
			Name:   ptr.String("foo600"),
			Query:  ptr.String("SELECT * from orbit_info;"),
			TeamID: &t1.ID,
		},
	}, http.StatusOK, &tcqr)

	// Attempt to edit own query, should allow.
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/queries/%d", tcqr.Query.ID), modifyQueryRequest{
		QueryPayload: fleet.QueryPayload{
			Name:  ptr.String("foo4"),
			Query: ptr.String("SELECT * FROM system_info;"),
		},
	}, http.StatusOK, &modifyQueryResponse{})

	// Attempt to delete own query, should allow.
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/queries/id/%d", tcqr.Query.ID), deleteQueryByIDRequest{}, http.StatusOK, &deleteQueryByIDResponse{})

	// Attempt to edit query created by somebody else, should fail.
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/queries/%d", cqr4.Query.ID), modifyQueryRequest{
		QueryPayload: fleet.QueryPayload{
			Name:  ptr.String("foo4"),
			Query: ptr.String("SELECT * FROM system_info;"),
		},
	}, http.StatusForbidden, &modifyQueryResponse{})

	// Attempt to delete query created by somebody else, should fail.
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/queries/id/%d", cqr4.Query.ID), deleteQueryByIDRequest{}, http.StatusForbidden, &deleteQueryByIDResponse{})

	// Attempt to read the global schedule, should fail.
	s.DoJSON("GET", "/api/latest/fleet/schedule", nil, http.StatusForbidden, &getGlobalScheduleResponse{})

	// Attempt to read the team's schedule, should fail.
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/schedule", t1.ID), getTeamScheduleRequest{}, http.StatusForbidden, &getTeamScheduleResponse{})

	// Attempt to read other team's schedule, should fail.
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/schedule", t2.ID), getTeamScheduleRequest{}, http.StatusForbidden, &getTeamScheduleResponse{})

	// Attempt to add a query to a user pack, should fail.
	tsqr := scheduleQueryResponse{}
	s.DoJSON("POST", "/api/latest/fleet/packs/schedule", scheduleQueryRequest{
		PackID:   userPackID,
		QueryID:  cqr4.Query.ID,
		Interval: 60,
	}, http.StatusForbidden, &tsqr)

	// Attempt to add a query to the team's schedule, should allow.
	cqrt1 := createQueryResponse{}
	s.DoJSON("POST", "/api/latest/fleet/queries", createQueryRequest{
		QueryPayload: fleet.QueryPayload{
			Name:   ptr.String("foo8"),
			Query:  ptr.String("SELECT * from managed_policies;"),
			TeamID: &t1.ID,
		},
	}, http.StatusOK, &cqrt1)
	ttsqr := teamScheduleQueryResponse{}
	// Add a schedule with the deprecated APIs (by referencing a global query).
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/schedule", t1.ID), teamScheduleQueryRequest{
		ScheduledQueryPayload: fleet.ScheduledQueryPayload{
			QueryID:  ptr.Uint(q1.ID),
			Interval: ptr.Uint(60),
		},
	}, http.StatusOK, &ttsqr)

	// Attempt to remove a query from the team's schedule, should allow.
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/teams/%d/schedule/%d", t1.ID, ttsqr.Scheduled.ID), deleteTeamScheduleRequest{}, http.StatusOK, &deleteTeamScheduleResponse{})

	// Attempt to read the global schedule, should fail.
	s.DoJSON("GET", "/api/latest/fleet/schedule", nil, http.StatusForbidden, &getGlobalScheduleResponse{})

	// Attempt to read a global policy, should fail.
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/policies/%d", gp2.ID), getPolicyByIDRequest{}, http.StatusForbidden, &getPolicyByIDResponse{})

	// Attempt to delete a global policy, should fail.
	s.DoJSON("POST", "/api/latest/fleet/policies/delete", deleteGlobalPoliciesRequest{
		IDs: []uint{gp2.ID},
	}, http.StatusForbidden, &deleteGlobalPoliciesResponse{})

	// Attempt to create a team policy, should allow.
	ttplr := teamPolicyResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/team/%d/policies", t1.ID), teamPolicyRequest{
		Name:  "foo1000",
		Query: "SELECT * from file;",
	}, http.StatusOK, &ttplr)

	// Attempt to edit a team policy, should allow.
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", t1.ID, ttplr.Policy.ID), modifyTeamPolicyRequest{
		ModifyPolicyPayload: fleet.ModifyPolicyPayload{
			Query: ptr.String("SELECT * from file WHERE path = 'foobar';"),
		},
	}, http.StatusOK, &modifyTeamPolicyResponse{})

	// Attempt to edit another team's policy, should fail.
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", t2.ID, t2p.ID), modifyTeamPolicyRequest{
		ModifyPolicyPayload: fleet.ModifyPolicyPayload{
			Query: ptr.String("SELECT * from file WHERE path = 'foobar';"),
		},
	}, http.StatusForbidden, &modifyTeamPolicyResponse{})

	// Attempt to view a team policy, should fail.
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/team/%d/policies/%d", t1.ID, ttplr.Policy.ID), getTeamPolicyByIDRequest{}, http.StatusForbidden, &getTeamPolicyByIDResponse{})

	// Attempt to view another team's policy, should fail.
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/team/%d/policies/%d", t2.ID, t2p.ID), getTeamPolicyByIDRequest{}, http.StatusForbidden, &getTeamPolicyByIDResponse{})

	// Attempt to delete a team policy, should allow.
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/delete", t1.ID), deleteTeamPoliciesRequest{
		IDs: []uint{ttplr.Policy.ID},
	}, http.StatusOK, &deleteTeamPoliciesResponse{})

	// Attempt to edit own team, should allow.
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", t1.ID), modifyTeamRequest{
		TeamPayload: fleet.TeamPayload{
			Name: ptr.String("foo123456"),
		},
	}, http.StatusOK, &teamResponse{})

	// Attempt to edit another team, should fail.
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", t2.ID), modifyTeamRequest{
		TeamPayload: fleet.TeamPayload{
			Name: ptr.String("foo123456"),
		},
	}, http.StatusForbidden, &teamResponse{})

	// Attempt to edit own team's agent options, should allow.
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/agent_options", t1.ID), json.RawMessage(`{
		"config": {
			"options": {
				"aws_debug": true
			}
		}
	}`), http.StatusOK, &teamResponse{})

	// Attempt to edit another team's agent options, should fail.
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/agent_options", t2.ID), json.RawMessage(`{
		"config": {
			"options": {
				"aws_debug": true
			}
		}
	}`), http.StatusForbidden, &teamResponse{})

	// Attempt to add users from team it owns to another team it owns, should allow.
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/users", t3.ID), modifyTeamUsersRequest{
		Users: []fleet.TeamUser{
			{
				User: *u3,
				Role: "maintainer",
			},
		},
	}, http.StatusOK, &teamResponse{})

	// Attempt to delete users from team it owns, should allow.
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/teams/%d/users", t3.ID), modifyTeamUsersRequest{
		Users: []fleet.TeamUser{
			{
				User: *u3,
			},
		},
	}, http.StatusOK, &teamResponse{})

	// Attempt to add users to another team it doesn't own, should fail.
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/users", t2.ID), modifyTeamUsersRequest{
		Users: []fleet.TeamUser{
			{
				User: *u3,
				Role: "maintainer",
			},
		},
	}, http.StatusForbidden, &teamResponse{})

	// Attempt to delete users from team it doesn't own, should fail.
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/teams/%d/users", t2.ID), modifyTeamUsersRequest{
		Users: []fleet.TeamUser{
			{
				User: *u2,
			},
		},
	}, http.StatusForbidden, &teamResponse{})

	// Attempt to search hosts, should fail.
	s.DoJSON("POST", "/api/latest/fleet/targets", searchTargetsRequest{
		MatchQuery: "foo",
		QueryID:    &q1.ID,
	}, http.StatusForbidden, &searchTargetsResponse{})

	// Attempt to count target hosts, should fail.
	s.DoJSON("POST", "/api/latest/fleet/targets/count", countTargetsRequest{
		Selected: fleet.HostTargets{
			HostIDs:  []uint{h1.ID},
			LabelIDs: []uint{clr.Label.ID},
			TeamIDs:  []uint{t1.ID},
		},
		QueryID: &q1.ID,
	}, http.StatusForbidden, &countTargetsResponse{})
}

func (s *integrationEnterpriseTestSuite) setTokenForTest(t *testing.T, email, password string) {
	oldToken := s.token
	t.Cleanup(func() {
		s.token = oldToken
	})

	s.token = s.getCachedUserToken(email, password)
}

func (s *integrationEnterpriseTestSuite) TestDesktopEndpointWithInvalidPolicy() {
	t := s.T()

	token := "abcd123"
	host := createHostAndDeviceToken(t, s.ds, token)

	// Create an 'invalid' global policy for host
	admin := s.users["admin1@example.com"]
	err := s.ds.SaveUser(context.Background(), &admin)
	require.NoError(t, err)

	policy, err := s.ds.NewGlobalPolicy(context.Background(), &admin.ID, fleet.PolicyPayload{
		Query:       "SELECT 1 FROM table",
		Name:        "test",
		Description: "Some invalid Query",
		Resolution:  "",
		Platform:    host.Platform,
		Critical:    false,
	})
	require.NoError(t, err)
	require.NoError(t, s.ds.RecordPolicyQueryExecutions(context.Background(), host, map[uint]*bool{policy.ID: nil}, time.Now(), false))

	// Any 'invalid' policies should be ignored.
	desktopRes := fleetDesktopResponse{}
	res := s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/desktop", nil, http.StatusOK)
	require.NoError(t, json.NewDecoder(res.Body).Decode(&desktopRes))
	require.NoError(t, res.Body.Close())
	require.NoError(t, desktopRes.Err)
	require.Equal(t, uint(0), *desktopRes.FailingPolicies)
}
