package service

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/datastore/redis/redistest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/live_query/live_query_mock"
	"github.com/fleetdm/fleet/v4/server/mdm"
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
	testingSuite.withServer.s = &testingSuite.Suite
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
		Pool:           s.redisPool,
		Lq:             s.lq,
		Logger:         log.NewLogfmtLogger(os.Stdout),
		EnableCachedDS: true,
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
		WindowsUpdates: fleet.WindowsUpdates{
			DeadlineDays:    optjson.Int{Set: true},
			GracePeriodDays: optjson.Int{Set: true},
		},
		MacOSSetup: fleet.MacOSSetup{
			// because the MacOSSetup was marshalled to JSON to be saved in the DB,
			// it did get marshalled, and then when unmarshalled it was set (but
			// null).
			MacOSSetupAssistant: optjson.String{Set: true},
			BootstrapPackage:    optjson.String{Set: true},
		},
		// because the WindowsSettings was marshalled to JSON to be saved in the DB,
		// it did get marshalled, and then when unmarshalled it was set (but
		// empty).
		WindowsSettings: fleet.WindowsSettings{
			CustomSettings: optjson.Slice[fleet.MDMProfileSpec]{Set: true, Value: []fleet.MDMProfileSpec{}},
		},
	}, team.Config.MDM)

	// an activity was created for team spec applied
	s.lastActivityMatches(fleet.ActivityTypeAppliedSpecTeam{}.ActivityName(), fmt.Sprintf(`{"teams": [{"id": %d, "name": %q}]}`, team.ID, team.Name), 0)

	// dry-run with invalid windows updates
	teamSpecs = map[string]any{
		"specs": []any{
			map[string]any{
				"name": teamName,
				"mdm": map[string]any{
					"windows_updates": map[string]any{
						"deadline_days":     -1,
						"grace_period_days": 1,
					},
				},
			},
		},
	}
	res := s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusUnprocessableEntity, "dry_run", "true")
	errMsg := extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "deadline_days must be an integer between 0 and 30")

	// apply valid windows updates settings
	teamSpecs = map[string]any{
		"specs": []any{
			map[string]any{
				"name": teamName,
				"mdm": map[string]any{
					"windows_updates": map[string]any{
						"deadline_days":     1,
						"grace_period_days": 1,
					},
				},
			},
		},
	}
	s.DoJSON("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK, &applyResp)
	require.Len(t, applyResp.TeamIDsByName, 1)
	team, err = s.ds.TeamByName(context.Background(), teamName)
	require.NoError(t, err)
	require.Equal(t, applyResp.TeamIDsByName[teamName], team.ID)
	require.Equal(t, fleet.TeamMDM{
		MacOSUpdates: fleet.MacOSUpdates{
			MinimumVersion: optjson.SetString("10.15.0"),
			Deadline:       optjson.SetString("2021-01-01"),
		},
		WindowsUpdates: fleet.WindowsUpdates{
			DeadlineDays:    optjson.SetInt(1),
			GracePeriodDays: optjson.SetInt(1),
		},
		MacOSSetup: fleet.MacOSSetup{
			MacOSSetupAssistant: optjson.String{Set: true},
			BootstrapPackage:    optjson.String{Set: true},
		},
		WindowsSettings: fleet.WindowsSettings{
			CustomSettings: optjson.Slice[fleet.MDMProfileSpec]{Set: true, Value: []fleet.MDMProfileSpec{}},
		},
	}, team.Config.MDM)

	// get the team via the GET endpoint, check that it properly returns the mdm settings
	var getTmResp getTeamResponse
	s.DoJSON("GET", "/api/latest/fleet/teams/"+fmt.Sprint(team.ID), nil, http.StatusOK, &getTmResp)
	require.Equal(t, fleet.TeamMDM{
		MacOSUpdates: fleet.MacOSUpdates{
			MinimumVersion: optjson.SetString("10.15.0"),
			Deadline:       optjson.SetString("2021-01-01"),
		},
		WindowsUpdates: fleet.WindowsUpdates{
			DeadlineDays:    optjson.SetInt(1),
			GracePeriodDays: optjson.SetInt(1),
		},
		MacOSSetup: fleet.MacOSSetup{
			MacOSSetupAssistant: optjson.String{Set: true},
			BootstrapPackage:    optjson.String{Set: true},
		},
		WindowsSettings: fleet.WindowsSettings{
			CustomSettings: optjson.Slice[fleet.MDMProfileSpec]{Set: true, Value: []fleet.MDMProfileSpec{}},
		},
	}, getTmResp.Team.Config.MDM)

	// get the team via the list teams endpoint, check that it properly returns the mdm settings
	var listTmResp listTeamsResponse
	s.DoJSON("GET", "/api/latest/fleet/teams", nil, http.StatusOK, &listTmResp, "query", teamName)
	require.True(t, len(listTmResp.Teams) > 0)
	require.Equal(t, team.ID, listTmResp.Teams[0].ID)
	require.Equal(t, fleet.TeamMDM{
		MacOSUpdates: fleet.MacOSUpdates{
			MinimumVersion: optjson.SetString("10.15.0"),
			Deadline:       optjson.SetString("2021-01-01"),
		},
		WindowsUpdates: fleet.WindowsUpdates{
			DeadlineDays:    optjson.SetInt(1),
			GracePeriodDays: optjson.SetInt(1),
		},
		MacOSSetup: fleet.MacOSSetup{
			MacOSSetupAssistant: optjson.String{Set: true},
			BootstrapPackage:    optjson.String{Set: true},
		},
		WindowsSettings: fleet.WindowsSettings{
			CustomSettings: optjson.Slice[fleet.MDMProfileSpec]{Set: true, Value: []fleet.MDMProfileSpec{}},
		},
	}, listTmResp.Teams[0].Config.MDM)

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
	res = s.DoRaw("POST", "/api/latest/fleet/spec/teams", nil, http.StatusBadRequest, "force", "true")
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
	errMsg = extractServerErrorText(res.Body)
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
	assert.Equal(t, map[string]uint{teamName: team.ID}, applyResp.TeamIDsByName)

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

	// dry-run with invalid host_expiry_settings.host_expiry_window
	teamSpecs = map[string]any{
		"specs": []map[string]any{
			{
				"name": teamName,
				"host_expiry_settings": map[string]any{
					"host_expiry_window":  0,
					"host_expiry_enabled": true,
				},
			},
		},
	}
	// Update team
	res = s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusUnprocessableEntity, "dry_run", "true")
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "host expiry window")
	// Create team (coverage should show that this validation was covered for both update and create)
	teamSpecs["specs"].([]map[string]any)[0]["name"] = teamName + "invalid host expiry window"
	res = s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusUnprocessableEntity, "dry_run", "true")
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "host expiry window")

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
	require.False(t, team.Config.MDM.EnableDiskEncryption)                 // unchanged

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
	assert.Len(t, team.Secrets, 1) // secret gets created automatically for a new team when none is supplied.
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
			Logging:        fleet.LoggingSnapshot,
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

	qr, err := s.ds.NewQuery(context.Background(), &fleet.Query{
		Name:           "TestQuery2",
		Description:    "Some description",
		Query:          "select * from osquery;",
		ObserverCanRun: true,
		Logging:        fleet.LoggingSnapshot,
	})
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

	// create a team with invalid host expiry window
	team4 := &fleet.TeamPayload{
		Name:        ptr.String(name + "invalid host_expiry_window"),
		Description: ptr.String("Team4 description"),
		Secrets:     []*fleet.EnrollSecret{{Secret: "TEAM4"}},
		HostExpirySettings: &fleet.HostExpirySettings{
			HostExpiryEnabled: true,
			HostExpiryWindow:  -1,
		},
	}
	s.DoJSON("POST", "/api/latest/fleet/teams", team4, http.StatusUnprocessableEntity, &tmResp)

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
			EnableDiskEncryption: optjson.SetBool(true),
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
	assert.False(t, tmResp.Team.Config.HostExpirySettings.HostExpiryEnabled)

	// modify non-existing team
	tmResp.Team = nil
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", tm1ID+1), team, http.StatusNotFound, &tmResp)

	// modify team host expiry
	modifyExpiry := fleet.TeamPayload{
		HostExpirySettings: &fleet.HostExpirySettings{
			HostExpiryEnabled: true,
			HostExpiryWindow:  10,
		},
	}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", tm1ID), modifyExpiry, http.StatusOK, &tmResp)
	assert.Equal(t, *modifyExpiry.HostExpirySettings, tmResp.Team.Config.HostExpirySettings)

	// invalid team host expiry (<= 0)
	modifyExpiry.HostExpirySettings.HostExpiryWindow = 0
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", tm1ID), modifyExpiry, http.StatusUnprocessableEntity, &tmResp)

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

	// modify team agent options with invalid platform options
	tmResp.Team = nil
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/agent_options", tm1ID), json.RawMessage(
		`{"overrides": {
			"platforms": {
				"linux": null
			}
		}}`,
	), http.StatusBadRequest, &tmResp)

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
	tmResp = teamResponse{}
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

	// update the team with an unrelated field, should not change integrations
	tmResp = teamResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{
		Description: ptr.String("team-desc"),
	}, http.StatusOK, &tmResp)
	require.Len(t, tmResp.Team.Config.Integrations.Jira, 1)
	require.Equal(t, "team-desc", tmResp.Team.Description)

	// make an unrelated appconfig change, should not remove the global integrations nor the teams'
	var appCfgResp appConfigResponse
	s.DoJSON("PATCH", "/api/v1/fleet/config", json.RawMessage(`{
		"org_info": {
			"org_name": "test-integrations"
		}
	}`), http.StatusOK, &appCfgResp)
	require.Equal(t, "test-integrations", appCfgResp.OrgInfo.OrgName)
	require.Len(t, appCfgResp.Integrations.Jira, 2)

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

	// update the team with an unrelated field, should not change integrations
	tmResp = teamResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), fleet.TeamPayload{
		Description: ptr.String("team-desc-2"),
	}, http.StatusOK, &tmResp)
	require.Len(t, tmResp.Team.Config.Integrations.Zendesk, 2)
	require.Equal(t, "team-desc-2", tmResp.Team.Description)

	// make an unrelated appconfig change, should not remove the global integrations nor the teams'
	appCfgResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/v1/fleet/config", json.RawMessage(`{
		"org_info": {
			"org_name": "test-integrations-2"
		}
	}`), http.StatusOK, &appCfgResp)
	require.Equal(t, "test-integrations-2", appCfgResp.OrgInfo.OrgName)
	require.Len(t, appCfgResp.Integrations.Zendesk, 2)
	require.Len(t, appCfgResp.Integrations.Jira, 2)

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

	appCfgResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/v1/fleet/config", json.RawMessage(`{
		"integrations": {
			"jira": [],
			"zendesk": []
		}
	}`), http.StatusOK, &appCfgResp)
	require.Len(t, appCfgResp.Integrations.Jira, 0)
	require.Len(t, appCfgResp.Integrations.Zendesk, 0)
}

func (s *integrationEnterpriseTestSuite) TestWindowsUpdatesTeamConfig() {
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

	checkWindowsOSUpdatesProfile(t, s.ds, &team.ID, nil)

	// modify the team's config
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), map[string]any{
		"mdm": map[string]any{
			"windows_updates": &fleet.WindowsUpdates{
				DeadlineDays:    optjson.SetInt(5),
				GracePeriodDays: optjson.SetInt(2),
			},
		},
	}, http.StatusOK, &tmResp)
	require.Equal(t, 5, tmResp.Team.Config.MDM.WindowsUpdates.DeadlineDays.Value)
	require.Equal(t, 2, tmResp.Team.Config.MDM.WindowsUpdates.GracePeriodDays.Value)
	s.lastActivityMatches(fleet.ActivityTypeEditedWindowsUpdates{}.ActivityName(), fmt.Sprintf(`{"team_id": %d, "team_name": %q, "deadline_days": 5, "grace_period_days": 2}`, team.ID, team.Name), 0)

	checkWindowsOSUpdatesProfile(t, s.ds, &team.ID, &fleet.WindowsUpdates{
		DeadlineDays:    optjson.SetInt(5),
		GracePeriodDays: optjson.SetInt(2),
	})

	// get the team via the GET endpoint, check that it properly returns the mdm
	// settings.
	var getTmResp getTeamResponse
	s.DoJSON("GET", "/api/latest/fleet/teams/"+fmt.Sprint(team.ID), nil, http.StatusOK, &getTmResp)
	require.Equal(t, fleet.TeamMDM{
		MacOSUpdates: fleet.MacOSUpdates{
			MinimumVersion: optjson.String{Set: true},
			Deadline:       optjson.String{Set: true},
		},
		WindowsUpdates: fleet.WindowsUpdates{
			DeadlineDays:    optjson.SetInt(5),
			GracePeriodDays: optjson.SetInt(2),
		},
		MacOSSetup: fleet.MacOSSetup{
			MacOSSetupAssistant: optjson.String{Set: true},
			BootstrapPackage:    optjson.String{Set: true},
		},
		WindowsSettings: fleet.WindowsSettings{
			CustomSettings: optjson.Slice[fleet.MDMProfileSpec]{Set: true, Value: []fleet.MDMProfileSpec{}},
		},
	}, getTmResp.Team.Config.MDM)

	// only update the deadline
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), map[string]any{
		"mdm": map[string]any{
			"windows_updates": &fleet.WindowsUpdates{
				DeadlineDays:    optjson.SetInt(6),
				GracePeriodDays: optjson.SetInt(2),
			},
		},
	}, http.StatusOK, &tmResp)
	require.Equal(t, 6, tmResp.Team.Config.MDM.WindowsUpdates.DeadlineDays.Value)
	require.Equal(t, 2, tmResp.Team.Config.MDM.WindowsUpdates.GracePeriodDays.Value)
	lastActivity := s.lastActivityMatches(fleet.ActivityTypeEditedWindowsUpdates{}.ActivityName(), fmt.Sprintf(`{"team_id": %d, "team_name": %q, "deadline_days": 6, "grace_period_days": 2}`, team.ID, team.Name), 0)

	checkWindowsOSUpdatesProfile(t, s.ds, &team.ID, &fleet.WindowsUpdates{
		DeadlineDays:    optjson.SetInt(6),
		GracePeriodDays: optjson.SetInt(2),
	})

	// setting the macos updates doesn't alter the windows updates
	tmResp = teamResponse{}
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
	require.Equal(t, 6, tmResp.Team.Config.MDM.WindowsUpdates.DeadlineDays.Value)
	require.Equal(t, 2, tmResp.Team.Config.MDM.WindowsUpdates.GracePeriodDays.Value)
	// did not create a new activity for windows updates
	s.lastActivityOfTypeMatches(fleet.ActivityTypeEditedWindowsUpdates{}.ActivityName(), "", lastActivity)
	lastActivity = s.lastActivityMatches(fleet.ActivityTypeEditedMacOSMinVersion{}.ActivityName(), ``, 0)

	checkWindowsOSUpdatesProfile(t, s.ds, &team.ID, &fleet.WindowsUpdates{
		DeadlineDays:    optjson.SetInt(6),
		GracePeriodDays: optjson.SetInt(2),
	})

	// sending a nil MDM or WindowsUpdates config doesn't modify anything
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), map[string]any{
		"mdm": nil,
	}, http.StatusOK, &tmResp)
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), map[string]any{
		"mdm": map[string]any{
			"windows_updates": nil,
		},
	}, http.StatusOK, &tmResp)
	require.Equal(t, "10.15.0", tmResp.Team.Config.MDM.MacOSUpdates.MinimumVersion.Value)
	require.Equal(t, "2021-01-01", tmResp.Team.Config.MDM.MacOSUpdates.Deadline.Value)
	require.Equal(t, 6, tmResp.Team.Config.MDM.WindowsUpdates.DeadlineDays.Value)
	require.Equal(t, 2, tmResp.Team.Config.MDM.WindowsUpdates.GracePeriodDays.Value)
	// no new activity is created
	s.lastActivityMatches("", "", lastActivity)

	checkWindowsOSUpdatesProfile(t, s.ds, &team.ID, &fleet.WindowsUpdates{
		DeadlineDays:    optjson.SetInt(6),
		GracePeriodDays: optjson.SetInt(2),
	})

	// sending empty WindowsUpdates fields empties both fields
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), map[string]any{
		"mdm": map[string]any{
			"windows_updates": map[string]any{
				"deadline_days":     nil,
				"grace_period_days": nil,
			},
		},
	}, http.StatusOK, &tmResp)
	require.False(t, tmResp.Team.Config.MDM.WindowsUpdates.DeadlineDays.Valid)
	require.False(t, tmResp.Team.Config.MDM.WindowsUpdates.GracePeriodDays.Valid)
	s.lastActivityMatches(fleet.ActivityTypeEditedWindowsUpdates{}.ActivityName(), fmt.Sprintf(`{"team_id": %d, "team_name": %q, "deadline_days": null, "grace_period_days": null}`, team.ID, team.Name), 0)

	checkWindowsOSUpdatesProfile(t, s.ds, &team.ID, nil)

	// error checks:

	// try to set an invalid deadline
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), map[string]any{
		"mdm": map[string]any{
			"windows_updates": map[string]any{
				"deadline_days":     1000,
				"grace_period_days": 1,
			},
		},
	}, http.StatusUnprocessableEntity, &tmResp)

	// try to set an invalid grace period
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), map[string]any{
		"mdm": map[string]any{
			"windows_updates": map[string]any{
				"deadline_days":     1,
				"grace_period_days": 1000,
			},
		},
	}, http.StatusUnprocessableEntity, &tmResp)

	// try to set a deadline but not a grace period
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), map[string]any{
		"mdm": map[string]any{
			"windows_updates": map[string]any{
				"deadline_days": 1,
			},
		},
	}, http.StatusUnprocessableEntity, &tmResp)

	// try to set a grace period but no deadline
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), map[string]any{
		"mdm": map[string]any{
			"windows_updates": map[string]any{
				"grace_period_days": 1,
			},
		},
	}, http.StatusUnprocessableEntity, &tmResp)

	// try to set an empty grace period but a non-empty deadline
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), map[string]any{
		"mdm": map[string]any{
			"windows_updates": map[string]any{
				"deadline_days":     1,
				"grace_period_days": nil,
			},
		},
	}, http.StatusUnprocessableEntity, &tmResp)
}

func (s *integrationEnterpriseTestSuite) TestMacOSUpdatesTeamConfig() {
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

	// setting the windows updates doesn't alter the macos updates
	tmResp = teamResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), map[string]any{
		"mdm": map[string]any{
			"windows_updates": &fleet.WindowsUpdates{
				DeadlineDays:    optjson.SetInt(10),
				GracePeriodDays: optjson.SetInt(2),
			},
		},
	}, http.StatusOK, &tmResp)
	require.Equal(t, "10.15.0", tmResp.Team.Config.MDM.MacOSUpdates.MinimumVersion.Value)
	require.Equal(t, "2025-10-01", tmResp.Team.Config.MDM.MacOSUpdates.Deadline.Value)
	require.Equal(t, 10, tmResp.Team.Config.MDM.WindowsUpdates.DeadlineDays.Value)
	require.Equal(t, 2, tmResp.Team.Config.MDM.WindowsUpdates.GracePeriodDays.Value)
	// did not create a new activity for macos updates
	s.lastActivityOfTypeMatches(fleet.ActivityTypeEditedMacOSMinVersion{}.ActivityName(), "", lastActivity)
	lastActivity = s.lastActivityMatches(fleet.ActivityTypeEditedWindowsUpdates{}.ActivityName(), ``, 0)

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

	// set the logo via the modify appconfig endpoint, so that the cache is
	// properly updated.
	var acResp appConfigResponse
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{"org_info":{"org_logo_url": "http://example.com/logo"}}`), http.StatusOK, &acResp)
	require.Equal(t, "http://example.com/logo", acResp.OrgInfo.OrgLogoURL)

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
		Logging:        fleet.LoggingSnapshot,
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
	err = res.Body.Close()
	require.NoError(t, err)

	// GET `/api/_version_/fleet/device/{token}/policies`
	listDevicePoliciesResp := listDevicePoliciesResponse{}
	res = s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/policies", nil, http.StatusOK)
	err = json.NewDecoder(res.Body).Decode(&listDevicePoliciesResp)
	require.NoError(t, err)
	err = res.Body.Close()
	require.NoError(t, err)
	require.Len(t, listDevicePoliciesResp.Policies, 2)
	require.NoError(t, listDevicePoliciesResp.Err)

	// GET `/api/_version_/fleet/device/{token}`
	getDeviceHostResp := getDeviceHostResponse{}
	res = s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token, nil, http.StatusOK)
	err = json.NewDecoder(res.Body).Decode(&getDeviceHostResp)
	require.NoError(t, err)
	err = res.Body.Close()
	require.NoError(t, err)
	require.NoError(t, getDeviceHostResp.Err)
	require.Equal(t, host.ID, getDeviceHostResp.Host.ID)
	require.False(t, getDeviceHostResp.Host.RefetchRequested)
	require.Equal(t, "http://example.com/logo", getDeviceHostResp.OrgLogoURL)
	require.Len(t, *getDeviceHostResp.Host.Policies, 2)

	// GET `/api/_version_/fleet/device/{token}/desktop`
	getDesktopResp := fleetDesktopResponse{}
	res = s.DoRawNoAuth("GET", "/api/latest/fleet/device/"+token+"/desktop", nil, http.StatusOK)
	err = json.NewDecoder(res.Body).Decode(&getDesktopResp)
	require.NoError(t, err)
	err = res.Body.Close()
	require.NoError(t, err)
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

func (s *integrationEnterpriseTestSuite) TestMDMWindowsUpdates() {
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
		require.Equal(t, fleet.WindowsUpdates{DeadlineDays: optjson.Int{Set: true}, GracePeriodDays: optjson.Int{Set: true}}, acResp.MDM.WindowsUpdates)

		// no activity got created
		activitiesResp = listActivitiesResponse{}
		s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &activitiesResp, "order_key", "a.id", "order_direction", "desc")
		require.Condition(t, func() bool {
			return (lastActivity == 0 && len(activitiesResp.Activities) == 0) ||
				(len(activitiesResp.Activities) > 0 && activitiesResp.Activities[0].ID == lastActivity)
		})
	}

	// missing grace period
	checkInvalidConfig(`{"mdm": {
		"windows_updates": {
			"deadline_days": 1
		}
	}}`)

	// missing deadline
	checkInvalidConfig(`{"mdm": {
		"windows_updates": {
			"grace_period_days": 1
		}
	}}`)

	// invalid deadline
	checkInvalidConfig(`{"mdm": {
		"windows_updates": {
			"grace_period_days": 1,
			"deadline_days": -1
		}
	}}`)

	// invalid grace period
	checkInvalidConfig(`{"mdm": {
		"windows_updates": {
			"grace_period_days": -1,
			"deadline_days": 1
		}
	}}`)

	checkWindowsOSUpdatesProfile(t, s.ds, nil, nil)

	// valid config
	acResp := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
			"mdm": {
				"windows_updates": {
					"deadline_days": 5,
					"grace_period_days": 1
				}
			}
		}`), http.StatusOK, &acResp)
	require.Equal(t, 5, acResp.MDM.WindowsUpdates.DeadlineDays.Value)
	require.Equal(t, 1, acResp.MDM.WindowsUpdates.GracePeriodDays.Value)

	checkWindowsOSUpdatesProfile(t, s.ds, nil, &fleet.WindowsUpdates{
		DeadlineDays:    optjson.SetInt(5),
		GracePeriodDays: optjson.SetInt(1),
	})

	// edited windows updates activity got created
	s.lastActivityMatches(fleet.ActivityTypeEditedWindowsUpdates{}.ActivityName(), `{"deadline_days":5, "grace_period_days":1, "team_id": null, "team_name": null}`, 0)

	// get the appconfig
	acResp = appConfigResponse{}
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	require.Equal(t, 5, acResp.MDM.WindowsUpdates.DeadlineDays.Value)
	require.Equal(t, 1, acResp.MDM.WindowsUpdates.GracePeriodDays.Value)

	// update the deadline
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
			"mdm": {
				"windows_updates": {
					"deadline_days": 6,
					"grace_period_days": 1
				}
			}
		}`), http.StatusOK, &acResp)
	require.Equal(t, 6, acResp.MDM.WindowsUpdates.DeadlineDays.Value)
	require.Equal(t, 1, acResp.MDM.WindowsUpdates.GracePeriodDays.Value)

	checkWindowsOSUpdatesProfile(t, s.ds, nil, &fleet.WindowsUpdates{
		DeadlineDays:    optjson.SetInt(6),
		GracePeriodDays: optjson.SetInt(1),
	})

	// another edited windows updates activity got created
	lastActivity = s.lastActivityMatches(fleet.ActivityTypeEditedWindowsUpdates{}.ActivityName(), `{"deadline_days":6, "grace_period_days":1, "team_id": null, "team_name": null}`, 0)

	// update something unrelated - the transparency url
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{"fleet_desktop":{"transparency_url": "customURL"}}`), http.StatusOK, &acResp)
	require.Equal(t, 6, acResp.MDM.WindowsUpdates.DeadlineDays.Value)
	require.Equal(t, 1, acResp.MDM.WindowsUpdates.GracePeriodDays.Value)

	// no activity got created
	s.lastActivityMatches("", ``, lastActivity)

	checkWindowsOSUpdatesProfile(t, s.ds, nil, &fleet.WindowsUpdates{
		DeadlineDays:    optjson.SetInt(6),
		GracePeriodDays: optjson.SetInt(1),
	})

	// clear the Windows requirement
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
			"mdm": {
				"windows_updates": {
					"deadline_days": null,
					"grace_period_days": null
				}
			}
		}`), http.StatusOK, &acResp)
	require.False(t, acResp.MDM.WindowsUpdates.DeadlineDays.Valid)
	require.False(t, acResp.MDM.WindowsUpdates.GracePeriodDays.Valid)

	// edited windows updates activity got created with empty requirement
	lastActivity = s.lastActivityMatches(fleet.ActivityTypeEditedWindowsUpdates{}.ActivityName(), `{"deadline_days":null, "grace_period_days":null, "team_id": null, "team_name": null}`, 0)

	checkWindowsOSUpdatesProfile(t, s.ds, nil, nil)

	// update again with empty windows requirement
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
			"mdm": {
				"windows_updates": {
					"deadline_days": null,
					"grace_period_days": null
				}
			}
		}`), http.StatusOK, &acResp)
	require.False(t, acResp.MDM.WindowsUpdates.DeadlineDays.Valid)
	require.False(t, acResp.MDM.WindowsUpdates.GracePeriodDays.Valid)

	// no activity got created
	s.lastActivityMatches("", ``, lastActivity)
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
	require.NoError(t, s.ds.SetOrUpdateHostDisksSpace(context.Background(), host1.ID, 10.0, 2.0, 500.0))
	require.NoError(t, s.ds.SetOrUpdateHostDisksSpace(context.Background(), host2.ID, 40.0, 4.0, 1000.0))

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

	// returns an error when the low_disk_space value is invalid (outside 1-100)
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusBadRequest, &resp, "low_disk_space", "101")
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusBadRequest, &resp, "low_disk_space", "0")

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

	// populate software for hosts
	now := time.Now()

	software := []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
	}
	_, err = s.ds.UpdateHostSoftware(context.Background(), host1.ID, software)
	require.NoError(t, err)

	require.NoError(t, s.ds.LoadHostSoftware(context.Background(), host1, false))

	inserted, err := s.ds.InsertSoftwareVulnerability(context.Background(), fleet.SoftwareVulnerability{
		SoftwareID: host1.Software[0].ID,
		CVE:        "cve-123-123-123",
	}, fleet.NVDSource)
	require.NoError(t, err)
	require.True(t, inserted)

	vulnMeta := []fleet.CVEMeta{{
		CVE:              "cve-123-123-123",
		CVSSScore:        ptr.Float64(5.4),
		EPSSProbability:  ptr.Float64(0.5),
		CISAKnownExploit: ptr.Bool(true),
		Published:        &now,
		Description:      "a long description of the cve",
	}}

	require.NoError(t, s.ds.InsertCVEMeta(context.Background(), vulnMeta))

	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "populate_software", "true")
	require.Len(t, resp.Hosts, 3)
	for _, h := range resp.Hosts {
		if h.ID == host1.ID {
			require.NotEmpty(t, h.Software)
			require.Len(t, h.Software, 1)
			require.NotEmpty(t, h.Software[0].Vulnerabilities)

			s := &vulnMeta[0].Description
			require.Equal(t, &vulnMeta[0].CVSSScore, h.Software[0].Vulnerabilities[0].CVSSScore)
			require.Equal(t, &vulnMeta[0].EPSSProbability, h.Software[0].Vulnerabilities[0].EPSSProbability)
			require.Equal(t, &vulnMeta[0].CISAKnownExploit, h.Software[0].Vulnerabilities[0].CISAKnownExploit)
			require.Equal(t, &s, h.Software[0].Vulnerabilities[0].Description)
		}
	}

	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "populate_software", "false")
	require.Len(t, resp.Hosts, 3)
	for _, h := range resp.Hosts {
		require.Empty(t, h.Software)
	}
}

func (s *integrationEnterpriseTestSuite) TestListVulnerabilities() {
	t := s.T()
	var resp listVulnerabilitiesResponse
	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities", nil, http.StatusOK, &resp)

	// Invalid Order Key
	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities", nil, http.StatusBadRequest, &resp, "order_key", "foo", "order_direction", "asc")

	// EE Only Order Key
	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities", nil, http.StatusOK, &resp, "order_key", "cvss_score", "order_direction", "asc")

	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities", nil, http.StatusOK, &resp)
	require.Len(s.T(), resp.Vulnerabilities, 0)

	host, err := s.ds.NewHost(context.Background(), &fleet.Host{
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
		CVE:               "CVE-2021-1234",
		ResolvedInVersion: ptr.String("10.0.19043.2013"),
	}, fleet.MSRCSource)
	require.NoError(t, err)

	res, err := s.ds.UpdateHostSoftware(context.Background(), host.ID, []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
	})
	require.NoError(t, err)
	sw := res.Inserted[0]

	_, err = s.ds.InsertSoftwareVulnerability(context.Background(), fleet.SoftwareVulnerability{
		SoftwareID: sw.ID,
		CVE:        "CVE-2021-1235",
	}, fleet.NVDSource)
	require.NoError(t, err)

	// insert CVEMeta
	mockTime := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	err = s.ds.InsertCVEMeta(context.Background(), []fleet.CVEMeta{
		{
			CVE:              "CVE-2021-1234",
			CVSSScore:        ptr.Float64(7.5),
			EPSSProbability:  ptr.Float64(0.5),
			CISAKnownExploit: ptr.Bool(true),
			Published:        ptr.Time(mockTime),
			Description:      "Test CVE 2021-1234",
		},
		{
			CVE:              "CVE-2021-1235",
			CVSSScore:        ptr.Float64(5.4),
			EPSSProbability:  ptr.Float64(0.6),
			CISAKnownExploit: ptr.Bool(false),
			Published:        ptr.Time(mockTime),
			Description:      "Test CVE 2021-1235",
		},
	})
	require.NoError(t, err)

	err = s.ds.UpdateVulnerabilityHostCounts(context.Background())
	require.NoError(t, err)

	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities", nil, http.StatusOK, &resp)
	require.Len(s.T(), resp.Vulnerabilities, 2)
	require.Equal(t, resp.Count, uint(2))
	require.False(t, resp.Meta.HasPreviousResults)
	require.False(t, resp.Meta.HasNextResults)
	require.Empty(t, resp.Err)

	expected := map[string]struct {
		fleet.CVEMeta
		HostCount   uint
		DetailsLink string
		Source      fleet.VulnerabilitySource
	}{
		"CVE-2021-1234": {
			HostCount:   1,
			DetailsLink: "https://msrc.microsoft.com/update-guide/en-US/vulnerability/CVE-2021-1234",
			CVEMeta: fleet.CVEMeta{
				CVE:              "CVE-2021-1234",
				CVSSScore:        ptr.Float64(7.5),
				EPSSProbability:  ptr.Float64(0.5),
				CISAKnownExploit: ptr.Bool(true),
				Published:        ptr.Time(mockTime),
				Description:      "Test CVE 2021-1234",
			},
		},
		"CVE-2021-1235": {
			HostCount:   1,
			DetailsLink: "https://nvd.nist.gov/vuln/detail/CVE-2021-1235",
			CVEMeta: fleet.CVEMeta{
				CVE:              "CVE-2021-1235",
				CVSSScore:        ptr.Float64(5.4),
				EPSSProbability:  ptr.Float64(0.6),
				CISAKnownExploit: ptr.Bool(false),
				Published:        ptr.Time(mockTime),
				Description:      "Test CVE 2021-1235",
			},
		},
	}

	for _, vuln := range resp.Vulnerabilities {
		expectedVuln, ok := expected[vuln.CVE]
		require.True(t, ok)
		require.Equal(t, expectedVuln.HostCount, vuln.HostsCount)
		require.Equal(t, expectedVuln.DetailsLink, vuln.DetailsLink)
		require.Equal(t, expectedVuln.CVEMeta, vuln.CVEMeta)
	}

	// EE Exploit Filter
	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities", nil, http.StatusOK, &resp, "exploit", "true")
	require.Len(t, resp.Vulnerabilities, 1)
	require.Equal(t, "CVE-2021-1234", resp.Vulnerabilities[0].CVE)

	// Test Team Filter
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
		expectedVuln, ok := expected[vuln.CVE]
		require.True(t, ok)
		require.Equal(t, expectedVuln.HostCount, vuln.HostsCount)
		require.Equal(t, expectedVuln.DetailsLink, vuln.DetailsLink)
		require.Equal(t, expectedVuln.CVEMeta, vuln.CVEMeta)
	}

	var gResp getVulnerabilityResponse
	s.DoJSON("GET", "/api/latest/fleet/vulnerabilities/CVE-2021-1234", nil, http.StatusOK, &gResp)
	require.Empty(t, gResp.Err)
	require.Equal(t, "CVE-2021-1234", gResp.Vulnerability.CVE)
	require.Equal(t, uint(1), gResp.Vulnerability.HostsCount)
	require.Equal(t, "https://msrc.microsoft.com/update-guide/en-US/vulnerability/CVE-2021-1234", gResp.Vulnerability.DetailsLink)
	require.Equal(t, "Test CVE 2021-1234", gResp.Vulnerability.Description)
	require.Equal(t, ptr.Float64(7.5), gResp.Vulnerability.CVSSScore)
	require.Equal(t, ptr.Bool(true), gResp.Vulnerability.CISAKnownExploit)
	require.Equal(t, ptr.Float64(0.5), gResp.Vulnerability.EPSSProbability)
	require.Equal(t, ptr.Time(mockTime), gResp.Vulnerability.Published)
	require.Len(t, gResp.OSVersions, 1)
	require.Equal(t, "Windows 11 Enterprise 22H2 10.0.19042.1234", gResp.OSVersions[0].Name)
	require.Equal(t, "Windows 11 Enterprise 22H2", gResp.OSVersions[0].NameOnly)
	require.Equal(t, "windows", gResp.OSVersions[0].Platform)
	require.Equal(t, "10.0.19042.1234", gResp.OSVersions[0].Version)
	require.Equal(t, 1, gResp.OSVersions[0].HostsCount)
	require.Equal(t, "10.0.19043.2013", *gResp.OSVersions[0].ResolvedInVersion)
}

func (s *integrationEnterpriseTestSuite) TestOSVersions() {
	t := s.T()

	testOS := fleet.OperatingSystem{Name: "Windows 11 Pro", Version: "10.0.22621.2861", Arch: "x86_64", KernelVersion: "10.0.22621.2861", Platform: "windows"}

	hosts := s.createHosts(t)

	var resp listHostsResponse
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp)
	require.Len(t, resp.Hosts, len(hosts))

	// set operating system information on a host
	require.NoError(t, s.ds.UpdateHostOperatingSystem(context.Background(), hosts[0].ID, testOS))
	var osinfo struct {
		ID          uint `db:"id"`
		OSVersionID uint `db:"os_version_id"`
	}
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), q, &osinfo,
			`SELECT id, os_version_id FROM operating_systems WHERE name = ? AND version = ? AND arch = ? AND kernel_version = ? AND platform = ?`,
			testOS.Name, testOS.Version, testOS.Arch, testOS.KernelVersion, testOS.Platform)
	})
	require.Greater(t, osinfo.ID, uint(0))

	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "os_name", testOS.Name, "os_version", testOS.Version)
	require.Len(t, resp.Hosts, 1)

	expected := resp.Hosts[0]
	resp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &resp, "os_id", fmt.Sprintf("%d", osinfo.ID))
	require.Len(t, resp.Hosts, 1)
	require.Equal(t, expected, resp.Hosts[0])

	// generate aggregated stats
	require.NoError(t, s.ds.UpdateOSVersions(context.Background()))

	// insert OS Vulns
	_, err := s.ds.InsertOSVulnerability(context.Background(), fleet.OSVulnerability{
		OSID: osinfo.ID,
		CVE:  "CVE-2021-1234",
	}, fleet.MSRCSource)
	require.NoError(t, err)

	// insert CVE MEta
	vulnMeta := []fleet.CVEMeta{
		{
			CVE:              "CVE-2021-1234",
			CVSSScore:        ptr.Float64(5.4),
			EPSSProbability:  ptr.Float64(0.5),
			CISAKnownExploit: ptr.Bool(true),
			Published:        ptr.Time(time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)),
			Description:      "a long description of the cve",
		},
	}
	require.NoError(t, s.ds.InsertCVEMeta(context.Background(), vulnMeta))

	var osVersionsResp osVersionsResponse
	s.DoJSON("GET", "/api/latest/fleet/os_versions", nil, http.StatusOK, &osVersionsResp)
	require.Len(t, osVersionsResp.OSVersions, 1)
	require.Equal(t, 1, osVersionsResp.OSVersions[0].HostsCount)
	require.Equal(t, fmt.Sprintf("%s %s", testOS.Name, testOS.Version), osVersionsResp.OSVersions[0].Name)
	require.Equal(t, testOS.Name, osVersionsResp.OSVersions[0].NameOnly)
	require.Equal(t, testOS.Version, osVersionsResp.OSVersions[0].Version)
	require.Equal(t, testOS.Platform, osVersionsResp.OSVersions[0].Platform)
	require.Len(t, osVersionsResp.OSVersions[0].Vulnerabilities, 1)
	require.Equal(t, "CVE-2021-1234", osVersionsResp.OSVersions[0].Vulnerabilities[0].CVE)
	require.Equal(t, "https://msrc.microsoft.com/update-guide/en-US/vulnerability/CVE-2021-1234", osVersionsResp.OSVersions[0].Vulnerabilities[0].DetailsLink)
	require.Equal(t, *vulnMeta[0].CVSSScore, **osVersionsResp.OSVersions[0].Vulnerabilities[0].CVSSScore)
	require.Equal(t, *vulnMeta[0].EPSSProbability, **osVersionsResp.OSVersions[0].Vulnerabilities[0].EPSSProbability)
	require.Equal(t, *vulnMeta[0].CISAKnownExploit, **osVersionsResp.OSVersions[0].Vulnerabilities[0].CISAKnownExploit)
	require.Equal(t, *vulnMeta[0].Published, **osVersionsResp.OSVersions[0].Vulnerabilities[0].CVEPublished)
	require.Equal(t, vulnMeta[0].Description, **osVersionsResp.OSVersions[0].Vulnerabilities[0].Description)

	var osVersionResp getOSVersionResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/os_versions/%d", osinfo.OSVersionID), nil, http.StatusOK, &osVersionResp)
	require.Equal(t, &osVersionsResp.OSVersions[0], osVersionResp.OSVersion)

	// OS versions with invalid team
	s.DoJSON(
		"GET", fmt.Sprintf("/api/latest/fleet/os_versions/%d", osinfo.OSVersionID), nil, http.StatusForbidden, &osVersionResp, "team_id",
		"99999",
	)

	// Create team and ask for the OS versions from the team (with no hosts) -- should get 404.
	tr := teamResponse{}
	s.DoJSON(
		"POST", "/api/latest/fleet/teams", createTeamRequest{
			TeamPayload: fleet.TeamPayload{
				Name: ptr.String("os_versions_team"),
			},
		}, http.StatusOK, &tr,
	)
	osVersionResp = getOSVersionResponse{}
	s.DoJSON(
		"GET", fmt.Sprintf("/api/latest/fleet/os_versions/%d", osinfo.OSVersionID), nil, http.StatusNotFound, &osVersionResp, "team_id",
		fmt.Sprintf("%d", tr.Team.ID),
	)

	// return empty json if UpdateOSVersions cron hasn't run yet for new team
	team, err := s.ds.NewTeam(context.Background(), &fleet.Team{Name: "new team"})
	require.NoError(t, err)
	require.NoError(t, s.ds.AddHostsToTeam(context.Background(), &team.ID, []uint{hosts[0].ID}))
	s.DoJSON("GET", "/api/latest/fleet/os_versions", nil, http.StatusOK, &osVersionsResp, "team_id", fmt.Sprintf("%d", team.ID))
	require.Len(t, osVersionsResp.OSVersions, 0)

	// return err if team_id is invalid
	s.DoJSON("GET", "/api/latest/fleet/os_versions", nil, http.StatusBadRequest, &osVersionsResp, "team_id", "invalid")
}

func (s *integrationEnterpriseTestSuite) TestMDMNotConfiguredEndpoints() {
	t := s.T()

	// create a host with device token to test device authenticated routes
	tkn := "D3V1C370K3N"
	createHostAndDeviceToken(t, s.ds, tkn)

	for _, route := range mdmConfigurationRequiredEndpoints() {
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
			SoftwareID:        bar.ID,
			CVE:               "cve-123",
			ResolvedInVersion: ptr.String("1.2.3"),
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
		Description:      "a long description of the cve",
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
	require.Equal(t, barPayload.Vulnerabilities[0].Description, ptr.StringPtr("a long description of the cve"))
	require.Equal(t, barPayload.Vulnerabilities[0].ResolvedInVersion, ptr.StringPtr("1.2.3"))

	var respVersions listSoftwareVersionsResponse
	s.DoJSON("GET", "/api/latest/fleet/software/versions", nil, http.StatusOK, &respVersions)
	require.NotNil(t, resp)

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
	require.Equal(t, barPayload.Vulnerabilities[0].Description, ptr.StringPtr("a long description of the cve"))
	require.Equal(t, barPayload.Vulnerabilities[0].ResolvedInVersion, ptr.StringPtr("1.2.3"))
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
		Name:    "Foo",
		Query:   "SELECT * from time;",
		Logging: fleet.LoggingSnapshot,
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
	s.DoJSON("GET", "/api/latest/fleet/software/versions", listSoftwareRequest{}, http.StatusForbidden, &listSoftwareVersionsResponse{})
	s.DoJSON("GET", "/api/latest/fleet/software", listSoftwareRequest{}, http.StatusForbidden, &listSoftwareResponse{})
	s.DoJSON("GET", "/api/latest/fleet/software/count", countSoftwareRequest{}, http.StatusForbidden, &countSoftwareResponse{})
	s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{}, http.StatusForbidden, &listSoftwareTitlesResponse{})
	s.DoJSON("GET", "/api/latest/fleet/software/titles/1", getSoftwareTitleRequest{}, http.StatusForbidden, &getSoftwareTitleResponse{})

	// Attempt to list a software, should fail.
	s.DoJSON("GET", "/api/latest/fleet/software/1", getSoftwareRequest{}, http.StatusForbidden, &getSoftwareResponse{})
	s.DoJSON("GET", "/api/latest/fleet/software/versions/1", getSoftwareRequest{}, http.StatusForbidden, &getSoftwareResponse{})

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
	s.DoJSON("GET", "/api/latest/fleet/queries/run", runLiveQueryRequest{
		HostIDs:  []uint{h1.ID},
		QueryIDs: []uint{q1.ID},
	}, http.StatusForbidden, &runLiveQueryResponse{},
	)

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

	// Attempt to view a query, should work.
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d", cqr.Query.ID), getQueryRequest{}, http.StatusOK, &getQueryResponse{})

	// Attempt to list all queries, should work.
	s.DoJSON("GET", "/api/latest/fleet/queries", listQueriesRequest{}, http.StatusOK, &listQueriesResponse{})

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

	// Attempt to read the global schedule, should allow.
	s.DoJSON("GET", "/api/latest/fleet/schedule", nil, http.StatusOK, &getGlobalScheduleResponse{})

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

	// Attempt to read a global policy, should allow.
	s.DoJSON(
		"GET", fmt.Sprintf("/api/latest/fleet/policies/%d", gplr.Policy.ID), getPolicyByIDRequest{}, http.StatusOK,
		&getPolicyByIDResponse{},
	)

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

	// Attempt to view a team policy, should allow.
	s.DoJSON(
		"GET", fmt.Sprintf("/api/latest/fleet/team/%d/policies/%d", t1.ID, tplr.Policy.ID), getTeamPolicyByIDRequest{}, http.StatusOK,
		&getTeamPolicyByIDResponse{},
	)

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

	// Attempt to read the team's schedule, should allow.
	s.DoJSON(
		"GET", fmt.Sprintf("/api/latest/fleet/teams/%d/schedule", t1.ID), getTeamScheduleRequest{}, http.StatusOK,
		&getTeamScheduleResponse{},
	)

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

	// Attempt to view a team policy, should allow.
	s.DoJSON(
		"GET", fmt.Sprintf("/api/latest/fleet/team/%d/policies/%d", t1.ID, ttplr.Policy.ID), getTeamPolicyByIDRequest{}, http.StatusOK,
		&getTeamPolicyByIDResponse{},
	)

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

func (s *integrationEnterpriseTestSuite) TestRunHostScript() {
	t := s.T()

	testRunScriptWaitForResult = 2 * time.Second
	defer func() { testRunScriptWaitForResult = 0 }()

	ctx := context.Background()

	host := createOrbitEnrolledHost(t, "linux", "", s.ds)
	otherHost := createOrbitEnrolledHost(t, "linux", "other", s.ds)

	// attempt to run a script on a non-existing host
	var runResp runScriptResponse
	s.DoJSON("POST", "/api/latest/fleet/scripts/run", fleet.HostScriptRequestPayload{HostID: host.ID + 100, ScriptContents: "echo"}, http.StatusNotFound, &runResp)

	// attempt to run an empty script
	res := s.Do("POST", "/api/latest/fleet/scripts/run", fleet.HostScriptRequestPayload{HostID: host.ID, ScriptContents: ""}, http.StatusUnprocessableEntity)
	errMsg := extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Script contents must not be empty.")

	// attempt to run an overly long script
	res = s.Do("POST", "/api/latest/fleet/scripts/run", fleet.HostScriptRequestPayload{HostID: host.ID, ScriptContents: strings.Repeat("a", 10001)}, http.StatusUnprocessableEntity)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Script is too large.")

	// make sure the host is still seen as "online"
	err := s.ds.MarkHostsSeen(ctx, []uint{host.ID}, time.Now())
	require.NoError(t, err)

	// create a valid script execution request
	s.DoJSON("POST", "/api/latest/fleet/scripts/run", fleet.HostScriptRequestPayload{HostID: host.ID, ScriptContents: "echo"}, http.StatusAccepted, &runResp)
	require.Equal(t, host.ID, runResp.HostID)
	require.NotEmpty(t, runResp.ExecutionID)

	result, err := s.ds.GetHostScriptExecutionResult(ctx, runResp.ExecutionID)
	require.NoError(t, err)
	require.Equal(t, host.ID, result.HostID)
	require.Equal(t, "echo", result.ScriptContents)
	require.Nil(t, result.ExitCode)

	// get script result
	var scriptResultResp getScriptResultResponse
	s.DoJSON("GET", "/api/latest/fleet/scripts/results/"+runResp.ExecutionID, nil, http.StatusOK, &scriptResultResp)
	require.Equal(t, host.ID, scriptResultResp.HostID)
	require.Equal(t, "echo", scriptResultResp.ScriptContents)
	require.Nil(t, scriptResultResp.ExitCode)
	require.False(t, scriptResultResp.HostTimeout)
	require.Contains(t, scriptResultResp.Message, fleet.RunScriptAsyncScriptEnqueuedErrMsg)

	// an async script doesn't care about timeouts
	now := time.Now()
	mysql.ExecAdhocSQL(t, s.ds, func(tx sqlx.ExtContext) error {
		_, err := tx.ExecContext(ctx, `UPDATE host_script_results SET created_at = ? WHERE execution_id = ?`,
			now.Add(-1*time.Hour),
			runResp.ExecutionID,
		)
		return err
	})
	scriptResultResp = getScriptResultResponse{}
	s.DoJSON("GET", "/api/latest/fleet/scripts/results/"+runResp.ExecutionID, nil, http.StatusOK, &scriptResultResp)
	require.Equal(t, host.ID, scriptResultResp.HostID)
	require.Equal(t, "echo", scriptResultResp.ScriptContents)
	require.Nil(t, scriptResultResp.ExitCode)
	require.False(t, scriptResultResp.HostTimeout)
	require.Contains(t, scriptResultResp.Message, fleet.RunScriptAsyncScriptEnqueuedErrMsg)

	// Disable scripts and verify that there are no Orbit notifs
	acr := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"server_settings": {
			"scripts_disabled": true
		}
	}`), http.StatusOK, &acr)
	require.True(t, acr.AppConfig.ServerSettings.ScriptsDisabled)

	var orbitResp orbitGetConfigResponse
	s.DoJSON("POST", "/api/fleet/orbit/config",
		json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *host.OrbitNodeKey)),
		http.StatusOK, &orbitResp)
	require.Empty(t, orbitResp.Notifications.PendingScriptExecutionIDs)

	// Verify that endpoints related to scripts are disabled
	srResp := s.Do("POST", "/api/latest/fleet/scripts/run/sync", fleet.HostScriptRequestPayload{HostID: host.ID, ScriptContents: "echo"}, http.StatusForbidden)
	assertBodyContains(t, srResp, fleet.RunScriptScriptsDisabledGloballyErrMsg)

	srResp = s.Do("POST", "/api/latest/fleet/scripts/run", fleet.HostScriptRequestPayload{HostID: host.ID, ScriptContents: "echo"}, http.StatusForbidden)
	assertBodyContains(t, srResp, fleet.RunScriptScriptsDisabledGloballyErrMsg)

	acr = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"server_settings": {
			"scripts_disabled": false
		}
	}`), http.StatusOK, &acr)
	require.False(t, acr.AppConfig.ServerSettings.ScriptsDisabled)

	// verify that orbit would get the notification that it has a script to run
	s.DoJSON("POST", "/api/fleet/orbit/config",
		json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *host.OrbitNodeKey)),
		http.StatusOK, &orbitResp)
	require.Equal(t, []string{scriptResultResp.ExecutionID}, orbitResp.Notifications.PendingScriptExecutionIDs)

	// the orbit endpoint to get a pending script to execute returns it
	var orbitGetScriptResp orbitGetScriptResponse
	s.DoJSON("POST", "/api/fleet/orbit/scripts/request",
		json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q, "execution_id": %q}`, *host.OrbitNodeKey, scriptResultResp.ExecutionID)),
		http.StatusOK, &orbitGetScriptResp)
	require.Equal(t, host.ID, orbitGetScriptResp.HostID)
	require.Equal(t, scriptResultResp.ExecutionID, orbitGetScriptResp.ExecutionID)
	require.Equal(t, "echo", orbitGetScriptResp.ScriptContents)

	// trying to get that script via its execution ID but a different host returns not found
	s.DoJSON("POST", "/api/fleet/orbit/scripts/request",
		json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q, "execution_id": %q}`, *otherHost.OrbitNodeKey, scriptResultResp.ExecutionID)),
		http.StatusNotFound, &orbitGetScriptResp)

	// trying to get an unknown execution id returns not found
	s.DoJSON("POST", "/api/fleet/orbit/scripts/request",
		json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q, "execution_id": %q}`, *host.OrbitNodeKey, scriptResultResp.ExecutionID+"no-such")),
		http.StatusNotFound, &orbitGetScriptResp)

	// attempt to run a sync script on a non-existing host
	var runSyncResp runScriptSyncResponse
	s.DoJSON("POST", "/api/latest/fleet/scripts/run/sync", fleet.HostScriptRequestPayload{HostID: host.ID + 100, ScriptContents: "echo"}, http.StatusNotFound, &runSyncResp)

	// attempt to sync run an empty script
	res = s.Do("POST", "/api/latest/fleet/scripts/run/sync", fleet.HostScriptRequestPayload{HostID: host.ID, ScriptContents: ""}, http.StatusUnprocessableEntity)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Script contents must not be empty.")

	// attempt to sync run an overly long script
	res = s.Do("POST", "/api/latest/fleet/scripts/run/sync", fleet.HostScriptRequestPayload{HostID: host.ID, ScriptContents: strings.Repeat("a", 10001)}, http.StatusUnprocessableEntity)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Script is too large.")

	// make sure the host is still seen as "online"
	err = s.ds.MarkHostsSeen(ctx, []uint{host.ID}, time.Now())
	require.NoError(t, err)

	// attempt to create a valid sync script execution request, fails because the
	// host has a pending script execution
	res = s.Do("POST", "/api/latest/fleet/scripts/run/sync", fleet.HostScriptRequestPayload{HostID: host.ID, ScriptContents: "echo"}, http.StatusConflict)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, fleet.RunScriptAlreadyRunningErrMsg)

	// save a result via the orbit endpoint
	var orbitPostScriptResp orbitPostScriptResultResponse
	s.DoJSON("POST", "/api/fleet/orbit/scripts/result",
		json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q, "execution_id": %q, "exit_code": 0, "output": "ok"}`, *host.OrbitNodeKey, scriptResultResp.ExecutionID)),
		http.StatusOK, &orbitPostScriptResp)

	// verify that orbit does not receive any pending script anymore
	orbitResp = orbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config",
		json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *host.OrbitNodeKey)),
		http.StatusOK, &orbitResp)
	require.Empty(t, orbitResp.Notifications.PendingScriptExecutionIDs)

	// create a valid sync script execution request, fails because the
	// request will time-out waiting for a result.
	runSyncResp = runScriptSyncResponse{}
	s.DoJSON("POST", "/api/latest/fleet/scripts/run/sync", fleet.HostScriptRequestPayload{HostID: host.ID, ScriptContents: "echo"}, http.StatusRequestTimeout, &runSyncResp)
	require.Equal(t, host.ID, runSyncResp.HostID)
	require.NotEmpty(t, runSyncResp.ExecutionID)
	require.True(t, runSyncResp.HostTimeout)
	require.Contains(t, runSyncResp.Message, fleet.RunScriptHostTimeoutErrMsg)

	s.DoJSON("POST", "/api/fleet/orbit/scripts/result",
		json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q, "execution_id": %q, "exit_code": 0, "output": "ok"}`, *host.OrbitNodeKey, runSyncResp.ExecutionID)),
		http.StatusOK, &orbitPostScriptResp)

	// create a valid sync script execution request, and simulate a result
	// arriving before timeout.
	testRunScriptWaitForResult = 5 * time.Second
	ctx, cancel := context.WithTimeout(ctx, testRunScriptWaitForResult)
	defer cancel()

	resultsCh := make(chan *fleet.HostScriptResultPayload, 1)
	go func() {
		for range time.Tick(300 * time.Millisecond) {
			pending, err := s.ds.ListPendingHostScriptExecutions(ctx, host.ID)
			if err != nil {
				t.Log(err)
				return
			}
			if len(pending) > 0 {
				select {
				case <-ctx.Done():
					return
				case r := <-resultsCh:
					r.ExecutionID = pending[0].ExecutionID
					// ignoring errors in this goroutine, the HTTP request below will fail if this fails
					_, err = s.ds.SetHostScriptExecutionResult(ctx, r)
					if err != nil {
						t.Log(err)
					}
				}
			}
		}
	}()

	// simulate a successful script result
	resultsCh <- &fleet.HostScriptResultPayload{
		HostID:   host.ID,
		Output:   "ok",
		Runtime:  1,
		ExitCode: 0,
	}
	runSyncResp = runScriptSyncResponse{}
	s.DoJSON("POST", "/api/latest/fleet/scripts/run/sync", fleet.HostScriptRequestPayload{HostID: host.ID, ScriptContents: "echo"}, http.StatusOK, &runSyncResp)
	require.Equal(t, host.ID, runSyncResp.HostID)
	require.NotEmpty(t, runSyncResp.ExecutionID)
	require.Equal(t, "ok", runSyncResp.Output)
	require.NotNil(t, runSyncResp.ExitCode)
	require.Equal(t, int64(0), *runSyncResp.ExitCode)
	require.False(t, runSyncResp.HostTimeout)

	// simulate a scripts disabled result
	resultsCh <- &fleet.HostScriptResultPayload{
		HostID:   host.ID,
		Output:   "",
		Runtime:  0,
		ExitCode: -2,
	}
	runSyncResp = runScriptSyncResponse{}
	s.DoJSON("POST", "/api/latest/fleet/scripts/run/sync", fleet.HostScriptRequestPayload{HostID: host.ID, ScriptContents: "echo"}, http.StatusOK, &runSyncResp)
	require.Equal(t, host.ID, runSyncResp.HostID)
	require.NotEmpty(t, runSyncResp.ExecutionID)
	require.Empty(t, runSyncResp.Output)
	require.NotNil(t, runSyncResp.ExitCode)
	require.Equal(t, int64(-2), *runSyncResp.ExitCode)
	require.False(t, runSyncResp.HostTimeout)
	require.Contains(t, runSyncResp.Message, "Scripts are disabled")

	// create a sync execution request.
	runSyncResp = runScriptSyncResponse{}
	s.DoJSON("POST", "/api/latest/fleet/scripts/run/sync", fleet.HostScriptRequestPayload{HostID: host.ID, ScriptContents: "echo"}, http.StatusRequestTimeout, &runSyncResp)

	// modify the timestamp of the script to simulate an script that has
	// been pending for a long time
	mysql.ExecAdhocSQL(t, s.ds, func(tx sqlx.ExtContext) error {
		_, err := tx.ExecContext(context.Background(), "UPDATE host_script_results SET created_at = ? WHERE execution_id = ?", time.Now().Add(-24*time.Hour), runSyncResp.ExecutionID)
		return err
	})

	// fetch the results for the timed-out script
	scriptResultResp = getScriptResultResponse{}
	s.DoJSON("GET", "/api/latest/fleet/scripts/results/"+runSyncResp.ExecutionID, nil, http.StatusOK, &scriptResultResp)
	require.Equal(t, host.ID, scriptResultResp.HostID)
	require.Equal(t, "echo", scriptResultResp.ScriptContents)
	require.Nil(t, scriptResultResp.ExitCode)
	require.True(t, scriptResultResp.HostTimeout)
	require.Contains(t, scriptResultResp.Message, fleet.RunScriptHostTimeoutErrMsg)

	// make the host "offline"
	err = s.ds.MarkHostsSeen(context.Background(), []uint{host.ID}, time.Now().Add(-time.Hour))
	require.NoError(t, err)

	// attempt to create a sync script execution request, fails because the host
	// is offline.
	res = s.Do("POST", "/api/latest/fleet/scripts/run/sync", fleet.HostScriptRequestPayload{HostID: host.ID, ScriptContents: "echo"}, http.StatusUnprocessableEntity)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, fleet.RunScriptHostOfflineErrMsg)

	// attempt to create an async script execution request, succeeds because script is added to queue.
	s.Do("POST", "/api/latest/fleet/scripts/run", fleet.HostScriptRequestPayload{HostID: host.ID, ScriptContents: "echo"}, http.StatusAccepted)

	// attempt to run a script on a plain osquery host
	plainOsqueryHost, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-time.Minute),
		OsqueryHostID:   ptr.String("plain-osquery-host"),
		NodeKey:         ptr.String("plain-osquery-host"),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%s.local", "plain-osquery-host"),
		HardwareSerial:  uuid.New().String(),
		Platform:        "linux",
	})
	require.NoError(t, err)
	res = s.Do("POST", "/api/latest/fleet/scripts/run", fleet.HostScriptRequestPayload{HostID: plainOsqueryHost.ID, ScriptContents: "echo"}, http.StatusUnprocessableEntity)
	require.Contains(t, extractServerErrorText(res.Body), fleet.RunScriptDisabledErrMsg)
	res = s.Do("POST", "/api/latest/fleet/scripts/run/sync", fleet.HostScriptRequestPayload{HostID: plainOsqueryHost.ID, ScriptContents: "echo"}, http.StatusUnprocessableEntity)
	require.Contains(t, extractServerErrorText(res.Body), fleet.RunScriptDisabledErrMsg)
}

func (s *integrationEnterpriseTestSuite) TestRunHostSavedScript() {
	t := s.T()

	testRunScriptWaitForResult = 2 * time.Second
	defer func() { testRunScriptWaitForResult = 0 }()

	ctx := context.Background()

	host := createOrbitEnrolledHost(t, "linux", "", s.ds)
	tm, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "team 1"})
	require.NoError(t, err)
	savedNoTmScript, err := s.ds.NewScript(ctx, &fleet.Script{
		TeamID:         nil,
		Name:           "no_team_script.sh",
		ScriptContents: "echo 'no team'",
	})
	require.NoError(t, err)
	savedTmScript, err := s.ds.NewScript(ctx, &fleet.Script{
		TeamID:         &tm.ID,
		Name:           "team_script.sh",
		ScriptContents: "echo 'team'",
	})
	require.NoError(t, err)

	// attempt to run a script on a non-existing host
	var runResp runScriptResponse
	s.DoJSON("POST", "/api/latest/fleet/scripts/run", fleet.HostScriptRequestPayload{HostID: host.ID + 100, ScriptID: &savedNoTmScript.ID}, http.StatusNotFound, &runResp)

	// attempt to run with both script contents and id
	res := s.Do("POST", "/api/latest/fleet/scripts/run", fleet.HostScriptRequestPayload{HostID: host.ID, ScriptContents: "echo", ScriptID: ptr.Uint(savedTmScript.ID + 999)}, http.StatusUnprocessableEntity)
	errMsg := extractServerErrorText(res.Body)
	require.Contains(t, errMsg, `Only one of "script_id" or "script_contents" can be provided.`)

	// attempt to run with unknown script id
	res = s.Do("POST", "/api/latest/fleet/scripts/run", fleet.HostScriptRequestPayload{HostID: host.ID, ScriptID: ptr.Uint(savedTmScript.ID + 999)}, http.StatusNotFound)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, `No script exists for the provided "script_id".`)

	// make sure the host is still seen as "online"
	err = s.ds.MarkHostsSeen(ctx, []uint{host.ID}, time.Now())
	require.NoError(t, err)

	// attempt to run a team script on a non-team host
	res = s.Do("POST", "/api/latest/fleet/scripts/run", fleet.HostScriptRequestPayload{HostID: host.ID, ScriptID: &savedTmScript.ID}, http.StatusUnprocessableEntity)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, `The script does not belong to the same team`)

	// make sure the host is still seen as "online"
	err = s.ds.MarkHostsSeen(ctx, []uint{host.ID}, time.Now())
	require.NoError(t, err)

	// create a valid script execution request
	s.DoJSON("POST", "/api/latest/fleet/scripts/run", fleet.HostScriptRequestPayload{HostID: host.ID, ScriptID: &savedNoTmScript.ID}, http.StatusAccepted, &runResp)
	require.Equal(t, host.ID, runResp.HostID)
	require.NotEmpty(t, runResp.ExecutionID)

	var scriptResultResp getScriptResultResponse
	s.DoJSON("GET", "/api/latest/fleet/scripts/results/"+runResp.ExecutionID, nil, http.StatusOK, &scriptResultResp)
	require.Equal(t, host.ID, scriptResultResp.HostID)
	require.Equal(t, "echo 'no team'", scriptResultResp.ScriptContents)
	require.Nil(t, scriptResultResp.ExitCode)
	require.False(t, scriptResultResp.HostTimeout)
	require.Contains(t, scriptResultResp.Message, fleet.RunScriptAsyncScriptEnqueuedErrMsg)
	require.NotNil(t, scriptResultResp.ScriptID)
	require.Equal(t, savedNoTmScript.ID, *scriptResultResp.ScriptID)

	// verify that orbit would get the notification that it has a script to run
	var orbitResp orbitGetConfigResponse
	s.DoJSON("POST", "/api/fleet/orbit/config",
		json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *host.OrbitNodeKey)),
		http.StatusOK, &orbitResp)
	require.Equal(t, []string{scriptResultResp.ExecutionID}, orbitResp.Notifications.PendingScriptExecutionIDs)

	// the orbit endpoint to get a pending script to execute returns it
	var orbitGetScriptResp orbitGetScriptResponse
	s.DoJSON("POST", "/api/fleet/orbit/scripts/request",
		json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q, "execution_id": %q}`, *host.OrbitNodeKey, scriptResultResp.ExecutionID)),
		http.StatusOK, &orbitGetScriptResp)
	require.Equal(t, host.ID, orbitGetScriptResp.HostID)
	require.Equal(t, scriptResultResp.ExecutionID, orbitGetScriptResp.ExecutionID)
	require.Equal(t, "echo 'no team'", orbitGetScriptResp.ScriptContents)

	// make sure the host is still seen as "online"
	err = s.ds.MarkHostsSeen(ctx, []uint{host.ID}, time.Now())
	require.NoError(t, err)

	// save a result via the orbit endpoint
	var orbitPostScriptResp orbitPostScriptResultResponse
	s.DoJSON("POST", "/api/fleet/orbit/scripts/result",
		json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q, "execution_id": %q, "exit_code": 0, "output": "ok"}`, *host.OrbitNodeKey, scriptResultResp.ExecutionID)),
		http.StatusOK, &orbitPostScriptResp)

	// an activity was created for the script execution
	s.lastActivityMatches(
		fleet.ActivityTypeRanScript{}.ActivityName(),
		fmt.Sprintf(
			`{"host_id": %d, "host_display_name": %q, "script_name": %q, "script_execution_id": %q, "async": true}`,
			host.ID, host.DisplayName(), savedNoTmScript.Name, scriptResultResp.ExecutionID,
		),
		0,
	)

	// verify that orbit does not receive any pending script anymore
	orbitResp = orbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config",
		json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *host.OrbitNodeKey)),
		http.StatusOK, &orbitResp)
	require.Empty(t, orbitResp.Notifications.PendingScriptExecutionIDs)

	// create a valid sync script execution request, fails because the
	// request will time-out waiting for a result.
	var runSyncResp runScriptSyncResponse
	s.DoJSON("POST", "/api/latest/fleet/scripts/run/sync", fleet.HostScriptRequestPayload{HostID: host.ID, ScriptID: &savedNoTmScript.ID}, http.StatusRequestTimeout, &runSyncResp)
	require.Equal(t, host.ID, runSyncResp.HostID)
	require.NotEmpty(t, runSyncResp.ExecutionID)
	require.NotNil(t, runSyncResp.ScriptID)
	require.Equal(t, savedNoTmScript.ID, *runSyncResp.ScriptID)
	require.Equal(t, "echo 'no team'", runSyncResp.ScriptContents)
	require.True(t, runSyncResp.HostTimeout)
	require.Contains(t, runSyncResp.Message, fleet.RunScriptHostTimeoutErrMsg)

	// deleting the saved script does not impact the pending script
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/scripts/%d", savedNoTmScript.ID), nil, http.StatusNoContent)

	// script id is now nil, but otherwise execution request is the same
	scriptResultResp = getScriptResultResponse{}
	s.DoJSON("GET", "/api/latest/fleet/scripts/results/"+runSyncResp.ExecutionID, nil, http.StatusOK, &scriptResultResp)
	require.Equal(t, host.ID, scriptResultResp.HostID)
	require.Equal(t, "echo 'no team'", scriptResultResp.ScriptContents)
	require.Nil(t, scriptResultResp.ExitCode)
	require.False(t, scriptResultResp.HostTimeout)
	require.Contains(t, scriptResultResp.Message, fleet.RunScriptAlreadyRunningErrMsg)
	require.Nil(t, scriptResultResp.ScriptID)

	// Verify that we can't enqueue more than 1k scripts

	// Make the host offline so that scripts enqueue
	err = s.ds.MarkHostsSeen(ctx, []uint{host.ID}, time.Now().Add(-time.Hour))
	require.NoError(t, err)
	for i := 0; i < 1000; i++ {
		script, err := s.ds.NewScript(ctx, &fleet.Script{
			TeamID:         nil,
			Name:           fmt.Sprintf("script_1k_%d.sh", i),
			ScriptContents: fmt.Sprintf("echo %d", i),
		})
		require.NoError(t, err)

		_, err = s.ds.NewHostScriptExecutionRequest(ctx, &fleet.HostScriptRequestPayload{HostID: host.ID, ScriptID: &script.ID})
		require.NoError(t, err)
	}

	script, err := s.ds.NewScript(ctx, &fleet.Script{
		TeamID:         nil,
		Name:           "script_1k_1000.sh",
		ScriptContents: "echo 1000",
	})
	require.NoError(t, err)

	s.DoJSON("POST", "/api/latest/fleet/scripts/run", fleet.HostScriptRequestPayload{HostID: host.ID, ScriptID: &script.ID}, http.StatusConflict, &runResp)

	// attempt to run a script on a plain osquery host
	plainOsqueryHost, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-time.Minute),
		OsqueryHostID:   ptr.String("plain-osquery-host-2"),
		NodeKey:         ptr.String("plain-osquery-host-2"),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%s.local", "plain-osquery-host-2"),
		HardwareSerial:  uuid.New().String(),
		Platform:        "linux",
	})
	require.NoError(t, err)
	res = s.Do("POST", "/api/latest/fleet/scripts/run", fleet.HostScriptRequestPayload{HostID: plainOsqueryHost.ID, ScriptID: &script.ID}, http.StatusUnprocessableEntity)
	require.Contains(t, extractServerErrorText(res.Body), fleet.RunScriptDisabledErrMsg)
	res = s.Do("POST", "/api/latest/fleet/scripts/run/sync", fleet.HostScriptRequestPayload{HostID: plainOsqueryHost.ID, ScriptID: &script.ID}, http.StatusUnprocessableEntity)
	require.Contains(t, extractServerErrorText(res.Body), fleet.RunScriptDisabledErrMsg)
}

func (s *integrationEnterpriseTestSuite) TestEnqueueSameScriptTwice() {
	t := s.T()
	ctx := context.Background()

	host := createOrbitEnrolledHost(t, "linux", "", s.ds)
	script, err := s.ds.NewScript(ctx, &fleet.Script{
		TeamID:         nil,
		Name:           "script.sh",
		ScriptContents: "echo 'hi from script'",
	})
	require.NoError(t, err)

	// Make the host offline so that scripts enqueue
	err = s.ds.MarkHostsSeen(ctx, []uint{host.ID}, time.Now().Add(-time.Hour))
	require.NoError(t, err)

	var runResp runScriptResponse
	s.DoJSON("POST", "/api/latest/fleet/scripts/run", fleet.HostScriptRequestPayload{HostID: host.ID, ScriptID: &script.ID}, http.StatusAccepted, &runResp)
	require.Equal(t, host.ID, runResp.HostID)
	require.NotEmpty(t, runResp.ExecutionID)

	// Should fail because the same script is already enqueued for this host.
	resp := s.Do("POST", "/api/latest/fleet/scripts/run", fleet.HostScriptRequestPayload{HostID: host.ID, ScriptID: &script.ID}, http.StatusConflict)
	errorMsg := extractServerErrorText(resp.Body)
	require.Contains(t, errorMsg, "The script is already queued on the given host")
}

func (s *integrationEnterpriseTestSuite) TestOrbitConfigExtensions() {
	t := s.T()
	ctx := context.Background()

	appCfg, err := s.ds.AppConfig(ctx)
	require.NoError(t, err)
	defer func() {
		err = s.ds.SaveAppConfig(ctx, appCfg)
		require.NoError(t, err)
	}()

	foobarLabel, err := s.ds.NewLabel(ctx, &fleet.Label{
		Name:  "Foobar",
		Query: "SELECT 1;",
	})
	require.NoError(t, err)
	zoobarLabel, err := s.ds.NewLabel(ctx, &fleet.Label{
		Name:  "Zoobar",
		Query: "SELECT 1;",
	})
	require.NoError(t, err)
	allHostsLabel, err := s.ds.GetLabelSpec(ctx, "All hosts")
	require.NoError(t, err)

	orbitDarwinClient := createOrbitEnrolledHost(t, "darwin", "foobar1", s.ds)
	orbitLinuxClient := createOrbitEnrolledHost(t, "linux", "foobar2", s.ds)
	orbitWindowsClient := createOrbitEnrolledHost(t, "windows", "foobar3", s.ds)

	// orbitDarwinClient is member of 'All hosts' and 'Zoobar' labels.
	err = s.ds.RecordLabelQueryExecutions(ctx, orbitDarwinClient, map[uint]*bool{
		allHostsLabel.ID: ptr.Bool(true),
		zoobarLabel.ID:   ptr.Bool(true),
	}, time.Now(), false)
	require.NoError(t, err)
	// orbitLinuxClient is member of 'All hosts' and 'Foobar' labels.
	err = s.ds.RecordLabelQueryExecutions(ctx, orbitLinuxClient, map[uint]*bool{
		allHostsLabel.ID: ptr.Bool(true),
		foobarLabel.ID:   ptr.Bool(true),
	}, time.Now(), false)
	require.NoError(t, err)
	// orbitWindowsClient is member of the 'All hosts' label only.
	err = s.ds.RecordLabelQueryExecutions(ctx, orbitWindowsClient, map[uint]*bool{
		allHostsLabel.ID: ptr.Bool(true),
	}, time.Now(), false)
	require.NoError(t, err)

	// Attempt to add labels to extensions.
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
			"labels": [
				"All hosts",
				"Foobar"
			],
			"channel": "stable",
			"platform": "linux"
		},
		"hello_world_macos": {
			"labels": [
				"All hosts",
				"Foobar"
			],
			"channel": "stable",
			"platform": "macos"
		},
		"hello_mars_macos": {
			"labels": [
				"All hosts",
				"Zoobar"
			],
			"channel": "stable",
			"platform": "macos"
		},
		"hello_world_windows": {
			"labels": [
				"Zoobar"
			],
			"channel": "stable",
			"platform": "windows"
		},
		"hello_mars_windows": {
			"labels": [
				"Foobar"
			],
			"channel": "stable",
			"platform": "windows"
		}
	}
  }
}`), http.StatusOK)

	resp := orbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *orbitDarwinClient.OrbitNodeKey)), http.StatusOK, &resp)
	require.JSONEq(t, `{
	"hello_mars_macos": {
		"channel": "stable",
		"platform": "macos"
	}
  }`, string(resp.Extensions))

	resp = orbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *orbitLinuxClient.OrbitNodeKey)), http.StatusOK, &resp)
	require.JSONEq(t, `{
	"hello_world_linux": {
		"channel": "stable",
		"platform": "linux"
	}
  }`, string(resp.Extensions))

	resp = orbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *orbitWindowsClient.OrbitNodeKey)), http.StatusOK, &resp)
	require.Empty(t, string(resp.Extensions))

	// orbitDarwinClient is now also a member of the 'Foobar' label.
	err = s.ds.RecordLabelQueryExecutions(ctx, orbitDarwinClient, map[uint]*bool{
		foobarLabel.ID: ptr.Bool(true),
	}, time.Now(), false)
	require.NoError(t, err)

	resp = orbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *orbitDarwinClient.OrbitNodeKey)), http.StatusOK, &resp)
	require.JSONEq(t, `{
	"hello_world_macos": {
		"channel": "stable",
		"platform": "macos"
	},
	"hello_mars_macos": {
		"channel": "stable",
		"platform": "macos"
	}
  }`, string(resp.Extensions))

	// orbitLinuxClient is no longer a member of the 'Foobar' label.
	err = s.ds.RecordLabelQueryExecutions(ctx, orbitLinuxClient, map[uint]*bool{
		foobarLabel.ID: nil,
	}, time.Now(), false)
	require.NoError(t, err)

	resp = orbitGetConfigResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/config", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *orbitLinuxClient.OrbitNodeKey)), http.StatusOK, &resp)
	require.Empty(t, string(resp.Extensions))

	// Attempt to set non-existent labels in the config.
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
			"labels": [
				"All hosts",
				"Doesn't exist"
			],
			"channel": "stable",
			"platform": "linux"
		}
	}
  }
}`), http.StatusBadRequest)
}

func (s *integrationEnterpriseTestSuite) TestSavedScripts() {
	t := s.T()
	ctx := context.Background()

	// create a saved script for no team
	var newScriptResp createScriptResponse
	body, headers := generateNewScriptMultipartRequest(t,
		"script1.sh", []byte(`echo "hello"`), s.token, nil)
	res := s.DoRawWithHeaders("POST", "/api/latest/fleet/scripts", body.Bytes(), http.StatusOK, headers)
	err := json.NewDecoder(res.Body).Decode(&newScriptResp)
	require.NoError(t, err)
	require.NotZero(t, newScriptResp.ScriptID)
	noTeamScriptID := newScriptResp.ScriptID
	s.lastActivityMatches("added_script", fmt.Sprintf(`{"script_name": %q, "team_name": null, "team_id": null}`, "script1.sh"), 0)

	// get the script
	var getScriptResp getScriptResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/scripts/%d", noTeamScriptID), nil, http.StatusOK, &getScriptResp)
	require.Equal(t, noTeamScriptID, getScriptResp.ID)
	require.Nil(t, getScriptResp.TeamID)
	require.Equal(t, "script1.sh", getScriptResp.Name)
	require.NotZero(t, getScriptResp.CreatedAt)
	require.NotZero(t, getScriptResp.UpdatedAt)
	require.Empty(t, getScriptResp.ScriptContents)

	// download the script's content
	res = s.Do("GET", fmt.Sprintf("/api/latest/fleet/scripts/%d", noTeamScriptID), nil, http.StatusOK, "alt", "media")
	b, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	require.Equal(t, `echo "hello"`, string(b))
	require.Equal(t, int64(len(`echo "hello"`)), res.ContentLength)
	require.Equal(t, fmt.Sprintf("attachment;filename=\"%s %s\"", time.Now().Format(time.DateOnly), "script1.sh"), res.Header.Get("Content-Disposition"))

	// get a non-existing script
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/scripts/%d", noTeamScriptID+999), nil, http.StatusNotFound, &getScriptResp)
	// download a non-existing script
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/scripts/%d", noTeamScriptID+999), nil, http.StatusNotFound, &getScriptResp, "alt", "media")

	// file name is empty
	body, headers = generateNewScriptMultipartRequest(t,
		"", []byte(`echo "hello"`), s.token, nil)
	res = s.DoRawWithHeaders("POST", "/api/latest/fleet/scripts", body.Bytes(), http.StatusBadRequest, headers)
	errMsg := extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "no file headers for script")

	// file name is not .sh
	body, headers = generateNewScriptMultipartRequest(t,
		"not_sh.txt", []byte(`echo "hello"`), s.token, nil)
	res = s.DoRawWithHeaders("POST", "/api/latest/fleet/scripts", body.Bytes(), http.StatusUnprocessableEntity, headers)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Validation Failed: File type not supported. Only .sh and .ps1 file type is allowed.")

	// file content is empty
	body, headers = generateNewScriptMultipartRequest(t,
		"script2.sh", []byte(``), s.token, nil)
	res = s.DoRawWithHeaders("POST", "/api/latest/fleet/scripts", body.Bytes(), http.StatusUnprocessableEntity, headers)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Script contents must not be empty")

	// file content is too large
	body, headers = generateNewScriptMultipartRequest(t,
		"script2.sh", []byte(strings.Repeat("a", 10001)), s.token, nil)
	res = s.DoRawWithHeaders("POST", "/api/latest/fleet/scripts", body.Bytes(), http.StatusUnprocessableEntity, headers)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Script is too large. It's limited to 10,000 characters")

	// invalid hashbang
	body, headers = generateNewScriptMultipartRequest(t,
		"script2.sh", []byte(`#!/bin/python`), s.token, nil)
	res = s.DoRawWithHeaders("POST", "/api/latest/fleet/scripts", body.Bytes(), http.StatusUnprocessableEntity, headers)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Interpreter not supported.")

	// script already exists with this name for this no-team
	body, headers = generateNewScriptMultipartRequest(t,
		"script1.sh", []byte(`echo "hello"`), s.token, nil)
	res = s.DoRawWithHeaders("POST", "/api/latest/fleet/scripts", body.Bytes(), http.StatusConflict, headers)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "A script with this name already exists")

	// team id does not exist
	body, headers = generateNewScriptMultipartRequest(t,
		"script1.sh", []byte(`echo "hello"`), s.token, map[string][]string{"team_id": {"123"}})
	res = s.DoRawWithHeaders("POST", "/api/latest/fleet/scripts", body.Bytes(), http.StatusNotFound, headers)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "The team does not exist.")

	// create a team
	tm, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "team1"})
	require.NoError(t, err)

	// create with existing name for this time for a team
	body, headers = generateNewScriptMultipartRequest(t,
		"script1.sh", []byte(`echo "team"`), s.token, map[string][]string{"team_id": {fmt.Sprintf("%d", tm.ID)}})
	res = s.DoRawWithHeaders("POST", "/api/latest/fleet/scripts", body.Bytes(), http.StatusOK, headers)
	err = json.NewDecoder(res.Body).Decode(&newScriptResp)
	require.NoError(t, err)
	require.NotZero(t, newScriptResp.ScriptID)
	require.NotEqual(t, noTeamScriptID, newScriptResp.ScriptID)
	tmScriptID := newScriptResp.ScriptID
	s.lastActivityMatches("added_script", fmt.Sprintf(`{"script_name": %q, "team_name": %q, "team_id": %d}`, "script1.sh", tm.Name, tm.ID), 0)

	// create a windows script
	body, headers = generateNewScriptMultipartRequest(t,
		"script2.ps1", []byte(`Write-Host "Hello, World!"`), s.token, map[string][]string{"team_id": {fmt.Sprintf("%d", tm.ID)}})

	res = s.DoRawWithHeaders("POST", "/api/latest/fleet/scripts", body.Bytes(), http.StatusOK, headers)
	err = json.NewDecoder(res.Body).Decode(&newScriptResp)
	require.NoError(t, err)
	require.NotZero(t, newScriptResp.ScriptID)
	require.NotEqual(t, noTeamScriptID, newScriptResp.ScriptID)
	s.lastActivityMatches("added_script", fmt.Sprintf(`{"script_name": %q, "team_name": %q, "team_id": %d}`, "script2.ps1", tm.Name, tm.ID), 0)

	// get team's script
	getScriptResp = getScriptResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/scripts/%d", tmScriptID), nil, http.StatusOK, &getScriptResp)
	require.Equal(t, tmScriptID, getScriptResp.ID)
	require.NotNil(t, getScriptResp.TeamID)
	require.Equal(t, tm.ID, *getScriptResp.TeamID)
	require.Equal(t, "script1.sh", getScriptResp.Name)
	require.NotZero(t, getScriptResp.CreatedAt)
	require.NotZero(t, getScriptResp.UpdatedAt)
	require.Empty(t, getScriptResp.ScriptContents)

	// download the team's script's content
	res = s.Do("GET", fmt.Sprintf("/api/latest/fleet/scripts/%d", tmScriptID), nil, http.StatusOK, "alt", "media")
	b, err = io.ReadAll(res.Body)
	require.NoError(t, err)
	require.Equal(t, `echo "team"`, string(b))
	require.Equal(t, int64(len(`echo "team"`)), res.ContentLength)
	require.Equal(t, fmt.Sprintf("attachment;filename=\"%s %s\"", time.Now().Format(time.DateOnly), "script1.sh"), res.Header.Get("Content-Disposition"))

	// script already exists with this name for this team
	body, headers = generateNewScriptMultipartRequest(t,
		"script1.sh", []byte(`echo "hello"`), s.token, map[string][]string{"team_id": {fmt.Sprintf("%d", tm.ID)}})

	res = s.DoRawWithHeaders("POST", "/api/latest/fleet/scripts", body.Bytes(), http.StatusConflict, headers)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "A script with this name already exists")

	// create with a different name for this team
	body, headers = generateNewScriptMultipartRequest(t,
		"script2.sh", []byte(`echo "hello"`), s.token, map[string][]string{"team_id": {fmt.Sprintf("%d", tm.ID)}})

	res = s.DoRawWithHeaders("POST", "/api/latest/fleet/scripts", body.Bytes(), http.StatusOK, headers)
	err = json.NewDecoder(res.Body).Decode(&newScriptResp)
	require.NoError(t, err)
	require.NotZero(t, newScriptResp.ScriptID)
	require.NotEqual(t, noTeamScriptID, newScriptResp.ScriptID)
	require.NotEqual(t, tmScriptID, newScriptResp.ScriptID)
	s.lastActivityMatches("added_script", fmt.Sprintf(`{"script_name": %q, "team_name": %q, "team_id": %d}`, "script2.sh", tm.Name, tm.ID), 0)

	// delete the no-team script
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/scripts/%d", noTeamScriptID), nil, http.StatusNoContent)
	s.lastActivityMatches("deleted_script", fmt.Sprintf(`{"script_name": %q, "team_name": null, "team_id": null}`, "script1.sh"), 0)

	// delete the initial team script
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/scripts/%d", tmScriptID), nil, http.StatusNoContent)
	s.lastActivityMatches("deleted_script", fmt.Sprintf(`{"script_name": %q, "team_name": %q, "team_id": %d}`, "script1.sh", tm.Name, tm.ID), 0)

	// delete a non-existing script
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/scripts/%d", noTeamScriptID), nil, http.StatusNotFound)
}

func (s *integrationEnterpriseTestSuite) TestListSavedScripts() {
	t := s.T()
	ctx := context.Background()

	// create some teams
	tm1, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "team1"})
	require.NoError(t, err)
	tm2, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "team2"})
	require.NoError(t, err)
	tm3, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "team3"})
	require.NoError(t, err)

	// create 5 scripts for no team and team 1
	for i := 0; i < 5; i++ {
		_, err = s.ds.NewScript(ctx, &fleet.Script{
			Name:           string('a' + byte(i)), // i.e. "a", "b", "c", ...
			ScriptContents: "echo",
		})
		require.NoError(t, err)
		_, err = s.ds.NewScript(ctx, &fleet.Script{Name: string('a' + byte(i)), TeamID: &tm1.ID, ScriptContents: "echo"})
		require.NoError(t, err)
	}

	// create a single script for team 2
	_, err = s.ds.NewScript(ctx, &fleet.Script{Name: "a", TeamID: &tm2.ID, ScriptContents: "echo"})
	require.NoError(t, err)

	cases := []struct {
		queries   []string // alternate query name and value
		teamID    *uint
		wantNames []string
		wantMeta  *fleet.PaginationMetadata
	}{
		{
			wantNames: []string{"a", "b", "c", "d", "e"},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: false, HasPreviousResults: false},
		},
		{
			queries:   []string{"per_page", "2"},
			wantNames: []string{"a", "b"},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: true, HasPreviousResults: false},
		},
		{
			queries:   []string{"per_page", "2", "page", "1"},
			wantNames: []string{"c", "d"},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: true, HasPreviousResults: true},
		},
		{
			queries:   []string{"per_page", "2", "page", "2"},
			wantNames: []string{"e"},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: false, HasPreviousResults: true},
		},
		{
			queries:   []string{"per_page", "3"},
			teamID:    &tm1.ID,
			wantNames: []string{"a", "b", "c"},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: true, HasPreviousResults: false},
		},
		{
			queries:   []string{"per_page", "3", "page", "1"},
			teamID:    &tm1.ID,
			wantNames: []string{"d", "e"},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: false, HasPreviousResults: true},
		},
		{
			queries:   []string{"per_page", "3", "page", "2"},
			teamID:    &tm1.ID,
			wantNames: nil,
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: false, HasPreviousResults: true},
		},
		{
			queries:   []string{"per_page", "3"},
			teamID:    &tm2.ID,
			wantNames: []string{"a"},
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: false, HasPreviousResults: false},
		},
		{
			queries:   []string{"per_page", "2"},
			teamID:    &tm3.ID,
			wantNames: nil,
			wantMeta:  &fleet.PaginationMetadata{HasNextResults: false, HasPreviousResults: false},
		},
	}
	for _, c := range cases {
		t.Run(fmt.Sprintf("%v: %#v", c.teamID, c.queries), func(t *testing.T) {
			var listResp listScriptsResponse
			queryArgs := c.queries
			if c.teamID != nil {
				queryArgs = append(queryArgs, "team_id", fmt.Sprint(*c.teamID))
			}
			s.DoJSON("GET", "/api/latest/fleet/scripts", nil, http.StatusOK, &listResp, queryArgs...)

			require.Equal(t, len(c.wantNames), len(listResp.Scripts))
			require.Equal(t, c.wantMeta, listResp.Meta)

			var gotNames []string
			if len(listResp.Scripts) > 0 {
				gotNames = make([]string, len(listResp.Scripts))
				for i, s := range listResp.Scripts {
					gotNames[i] = s.Name
					if c.teamID == nil {
						require.Nil(t, s.TeamID)
					} else {
						require.NotNil(t, s.TeamID)
						require.Equal(t, *c.teamID, *s.TeamID)
					}
				}
			}
			require.Equal(t, c.wantNames, gotNames)
		})
	}
}

func (s *integrationEnterpriseTestSuite) TestHostScriptDetails() {
	t := s.T()
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	// create some teams
	tm1, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "test-script-details-team1"})
	require.NoError(t, err)
	tm2, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "test-script-details-team2"})
	require.NoError(t, err)
	tm3, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "test-script-details-team3"})
	require.NoError(t, err)
	tm4, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "test-script-details-team4-windows"})
	require.NoError(t, err)

	// create 5 scripts for no team and team 1
	for i := 0; i < 5; i++ {
		_, err = s.ds.NewScript(ctx, &fleet.Script{Name: fmt.Sprintf("test-script-details-%d.sh", i), ScriptContents: "echo"})
		require.NoError(t, err)
		_, err = s.ds.NewScript(ctx, &fleet.Script{Name: fmt.Sprintf("test-script-details-%d.sh", i), TeamID: &tm1.ID, ScriptContents: "echo"})
		require.NoError(t, err)
	}

	// add a windows script to team 4
	_, err = s.ds.NewScript(ctx, &fleet.Script{Name: "test-script-details-windows.ps1", TeamID: &tm4.ID, ScriptContents: `Write-Host "Hello, World!"`})
	require.NoError(t, err)

	// create a single script for team 2
	_, err = s.ds.NewScript(ctx, &fleet.Script{Name: "test-script-details-team-2.sh", TeamID: &tm2.ID, ScriptContents: "echo"})
	require.NoError(t, err)

	// create a host without a team
	host0, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   ptr.String("host0"),
		NodeKey:         ptr.String("host0"),
		UUID:            uuid.New().String(),
		Hostname:        "host0",
		Platform:        "darwin",
	})
	require.NoError(t, err)

	// create a host for team 1
	host1, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   ptr.String("host1"),
		NodeKey:         ptr.String("host1"),
		UUID:            uuid.New().String(),
		Hostname:        "host1",
		Platform:        "darwin",
		TeamID:          &tm1.ID,
	})
	require.NoError(t, err)

	// create a host for team 3
	host2, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   ptr.String("host2"),
		NodeKey:         ptr.String("host2"),
		UUID:            uuid.New().String(),
		Hostname:        "host2",
		Platform:        "darwin",
		TeamID:          &tm3.ID,
	})
	require.NoError(t, err)

	// create a Windows host
	host3, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   ptr.String("host3"),
		NodeKey:         ptr.String("host3"),
		UUID:            uuid.New().String(),
		Hostname:        "host3",
		Platform:        "windows",
		TeamID:          &tm4.ID,
	})
	require.NoError(t, err)

	// create a Linux host
	host4, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   ptr.String("host4"),
		NodeKey:         ptr.String("host4"),
		UUID:            uuid.New().String(),
		Hostname:        "host4",
		Platform:        "ubuntu",
		TeamID:          nil,
	})
	require.NoError(t, err)

	// create a chrome host
	host5, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   ptr.String("host5"),
		NodeKey:         ptr.String("host5"),
		UUID:            uuid.New().String(),
		Hostname:        "host5",
		Platform:        "chrome",
		TeamID:          nil,
	})
	require.NoError(t, err)

	insertResults := func(t *testing.T, hostID uint, script *fleet.Script, createdAt time.Time, execID string, exitCode *int64) {
		stmt := `
INSERT INTO
	host_script_results (%s host_id, created_at, execution_id, exit_code, script_contents, output, sync_request)
VALUES
	(%s ?,?,?,?,?,?, 1)`

		args := []interface{}{}
		if script.ID == 0 {
			stmt = fmt.Sprintf(stmt, "", "")
		} else {
			stmt = fmt.Sprintf(stmt, "script_id,", "?,")
			args = append(args, script.ID)
		}
		args = append(args, hostID, createdAt, execID, exitCode, script.ScriptContents, "")

		mysql.ExecAdhocSQL(t, s.ds, func(tx sqlx.ExtContext) error {
			_, err := tx.ExecContext(ctx, stmt, args...)
			return err
		})
	}

	// insert some ad hoc script results, these are never included in the host script details
	insertResults(t, host0.ID, &fleet.Script{Name: "ad hoc script", ScriptContents: "echo foo"}, now, "ad-hoc-0", ptr.Int64(0))
	insertResults(t, host1.ID, &fleet.Script{Name: "ad hoc script", ScriptContents: "echo foo"}, now.Add(-1*time.Hour), "ad-hoc-1", ptr.Int64(1))

	t.Run("no team", func(t *testing.T) {
		noTeamScripts, _, err := s.ds.ListScripts(ctx, nil, fleet.ListOptions{})
		require.NoError(t, err)
		require.Len(t, noTeamScripts, 5)

		// insert saved script results for host0
		insertResults(t, host0.ID, noTeamScripts[0], now, "exec0-0", ptr.Int64(0))                   // expect status ran
		insertResults(t, host0.ID, noTeamScripts[1], now.Add(-1*time.Hour), "exec0-1", ptr.Int64(1)) // expect status error
		insertResults(t, host0.ID, noTeamScripts[2], now.Add(-2*time.Hour), "exec0-2", nil)          // expect status pending

		// insert some ad hoc script results, these are never included in the host script details
		insertResults(t, host0.ID, &fleet.Script{Name: "ad hoc script", ScriptContents: "echo foo"}, now.Add(-3*time.Hour), "exec0-3", ptr.Int64(0))

		// check host script details, should include all no team scripts
		var resp getHostScriptDetailsResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/scripts", host0.ID), nil, http.StatusOK, &resp)
		require.Len(t, resp.Scripts, len(noTeamScripts))
		byScriptID := make(map[uint]*fleet.HostScriptDetail, len(resp.Scripts))
		for _, s := range resp.Scripts {
			byScriptID[s.ScriptID] = s
		}
		for i, s := range noTeamScripts {
			gotScript, ok := byScriptID[s.ID]
			require.True(t, ok)
			require.Equal(t, s.Name, gotScript.Name)
			switch i {
			case 0:
				require.NotNil(t, gotScript.LastExecution)
				require.Equal(t, "exec0-0", gotScript.LastExecution.ExecutionID)
				require.Equal(t, now, gotScript.LastExecution.ExecutedAt)
				require.Equal(t, "ran", gotScript.LastExecution.Status)
			case 1:
				require.NotNil(t, gotScript.LastExecution)
				require.Equal(t, "exec0-1", gotScript.LastExecution.ExecutionID)
				require.Equal(t, now.Add(-1*time.Hour), gotScript.LastExecution.ExecutedAt)
				require.Equal(t, "error", gotScript.LastExecution.Status)
			case 2:
				require.NotNil(t, gotScript.LastExecution)
				require.Equal(t, "exec0-2", gotScript.LastExecution.ExecutionID)
				require.Equal(t, now.Add(-2*time.Hour), gotScript.LastExecution.ExecutedAt)
				require.Equal(t, "pending", gotScript.LastExecution.Status)
			default:
				require.Nil(t, gotScript.LastExecution)
			}
		}
	})

	t.Run("team 1", func(t *testing.T) {
		tm1Scripts, _, err := s.ds.ListScripts(ctx, &tm1.ID, fleet.ListOptions{})
		require.NoError(t, err)
		require.Len(t, tm1Scripts, 5)

		// insert results for host1
		insertResults(t, host1.ID, tm1Scripts[0], now, "exec1-0", ptr.Int64(0)) // expect status ran

		// check host script details, should match team 1
		var resp getHostScriptDetailsResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/scripts", host1.ID), nil, http.StatusOK, &resp)
		require.Len(t, resp.Scripts, len(tm1Scripts))
		byScriptID := make(map[uint]*fleet.HostScriptDetail, len(resp.Scripts))
		for _, s := range resp.Scripts {
			byScriptID[s.ScriptID] = s
		}
		for i, s := range tm1Scripts {
			gotScript, ok := byScriptID[s.ID]
			require.True(t, ok)
			require.Equal(t, s.Name, gotScript.Name)
			switch i {
			case 0:
				require.NotNil(t, gotScript.LastExecution)
				require.Equal(t, "exec1-0", gotScript.LastExecution.ExecutionID)
				require.Equal(t, now, gotScript.LastExecution.ExecutedAt)
				require.Equal(t, "ran", gotScript.LastExecution.Status)
			default:
				require.Nil(t, gotScript.LastExecution)
			}
		}
	})

	t.Run("deleted script", func(t *testing.T) {
		noTeamScripts, _, err := s.ds.ListScripts(ctx, nil, fleet.ListOptions{})
		require.NoError(t, err)
		require.Len(t, noTeamScripts, 5)

		// delete a script
		s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/scripts/%d", noTeamScripts[0].ID), nil, http.StatusNoContent)

		// check host script details, should not include deleted script
		var resp getHostScriptDetailsResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/scripts", host0.ID), nil, http.StatusOK, &resp)
		require.Len(t, resp.Scripts, len(noTeamScripts)-1)
		byScriptID := make(map[uint]*fleet.HostScriptDetail, len(resp.Scripts))
		for _, s := range resp.Scripts {
			require.NotEqual(t, noTeamScripts[0].ID, s.ScriptID)
			byScriptID[s.ScriptID] = s
		}
		for i, s := range noTeamScripts {
			gotScript, ok := byScriptID[s.ID]
			if i == 0 {
				require.False(t, ok)
			} else {
				require.True(t, ok)
				require.Equal(t, s.Name, gotScript.Name)
				switch i {
				case 1:
					require.NotNil(t, gotScript.LastExecution)
					require.Equal(t, "exec0-1", gotScript.LastExecution.ExecutionID)
					require.Equal(t, now.Add(-1*time.Hour), gotScript.LastExecution.ExecutedAt)
					require.Equal(t, "error", gotScript.LastExecution.Status)
				case 2:
					require.NotNil(t, gotScript.LastExecution)
					require.Equal(t, "exec0-2", gotScript.LastExecution.ExecutionID)
					require.Equal(t, now.Add(-2*time.Hour), gotScript.LastExecution.ExecutedAt)
					require.Equal(t, "pending", gotScript.LastExecution.Status)
				case 3, 4:
					require.Nil(t, gotScript.LastExecution)
				default:
					require.Fail(t, "unexpected script")
				}
			}
		}
	})

	t.Run("transfer team", func(t *testing.T) {
		s.DoJSON("POST", "/api/latest/fleet/hosts/transfer", addHostsToTeamRequest{
			TeamID:  &tm2.ID,
			HostIDs: []uint{host1.ID},
		}, http.StatusOK, &addHostsToTeamResponse{})

		tm2Scripts, _, err := s.ds.ListScripts(ctx, &tm2.ID, fleet.ListOptions{})
		require.NoError(t, err)
		require.Len(t, tm2Scripts, 1)

		// check host script details, should not include prior team's scripts
		var resp getHostScriptDetailsResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/scripts", host1.ID), nil, http.StatusOK, &resp)
		require.Len(t, resp.Scripts, len(tm2Scripts))
		byScriptID := make(map[uint]*fleet.HostScriptDetail, len(resp.Scripts))
		for _, s := range resp.Scripts {
			byScriptID[s.ScriptID] = s
		}
		for _, s := range tm2Scripts {
			gotScript, ok := byScriptID[s.ID]
			require.True(t, ok)
			require.Equal(t, s.Name, gotScript.Name)
			require.Nil(t, gotScript.LastExecution)
		}
	})

	t.Run("no scripts", func(t *testing.T) {
		var resp getHostScriptDetailsResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/scripts", host2.ID), nil, http.StatusOK, &resp)
		require.NotNil(t, resp.Scripts)
		require.Len(t, resp.Scripts, 0)
	})

	t.Run("windows", func(t *testing.T) {
		team4Scripts, _, err := s.ds.ListScripts(ctx, &tm4.ID, fleet.ListOptions{})
		require.NoError(t, err)
		require.Len(t, team4Scripts, 1)

		var resp getHostScriptDetailsResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/scripts", host3.ID), nil, http.StatusOK, &resp)
		require.NotNil(t, resp.Scripts)
		require.Len(t, resp.Scripts, 1)
	})

	t.Run("linux", func(t *testing.T) {
		require.Nil(t, host4.TeamID)
		noTeamScripts, _, err := s.ds.ListScripts(ctx, nil, fleet.ListOptions{})
		require.NoError(t, err)
		require.True(t, len(noTeamScripts) > 0)

		var resp getHostScriptDetailsResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/scripts", host4.ID), nil, http.StatusOK, &resp)
		require.NotNil(t, resp.Scripts)
		require.Len(t, resp.Scripts, 4)

		for _, s := range resp.Scripts {
			require.Nil(t, s.LastExecution)
			require.Contains(t, s.Name, ".sh")
		}
	})

	// NOTE: Scripts are specified only for platforms other than macOS, Linux,
	// and Windows; however, we default to listing all scripts for unspecified platforms.
	// Separately, the UI restricts scripts related functionality to only macOS,
	// Linux, and Windows.
	t.Run("unspecified platform", func(t *testing.T) {
		var resp getHostScriptDetailsResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/scripts", host5.ID), nil, http.StatusOK, &resp)
		require.NotNil(t, resp.Scripts)
		require.Len(t, resp.Scripts, 4)
	})

	t.Run("get script results user message", func(t *testing.T) {
		// add a script with an older created_at timestamp
		var oldScriptID uint
		mysql.ExecAdhocSQL(t, s.ds, func(tx sqlx.ExtContext) error {
			res, err := tx.ExecContext(ctx, `
INSERT INTO
	scripts (name, script_contents, created_at, updated_at)
VALUES
	(?,?,?,?)`,
				"test-script-details-timeout.sh",
				"echo test-script-details-timeout",
				now.Add(-1*time.Hour),
				now.Add(-1*time.Hour),
			)
			if err != nil {
				return err
			}
			id, err := res.LastInsertId()
			if err != nil {
				return err
			}
			oldScriptID = uint(id)
			return nil
		})

		for _, c := range []struct {
			name       string
			exitCode   *int64
			executedAt time.Time
			expected   string
		}{
			{
				name:       "host-timeout",
				exitCode:   nil,
				executedAt: now.Add(-1 * time.Hour),
				expected:   fleet.RunScriptHostTimeoutErrMsg,
			},
			{
				name:       "script-timeout",
				exitCode:   ptr.Int64(-1),
				executedAt: now.Add(-1 * time.Hour),
				expected:   fleet.RunScriptScriptTimeoutErrMsg,
			},
			{
				name:       "pending",
				exitCode:   nil,
				executedAt: now.Add(-1 * time.Minute),
				expected:   fleet.RunScriptAlreadyRunningErrMsg,
			},
			{
				name:       "success",
				exitCode:   ptr.Int64(0),
				executedAt: now.Add(-1 * time.Hour),
				expected:   "",
			},
			{
				name:       "error",
				exitCode:   ptr.Int64(1),
				executedAt: now.Add(-1 * time.Hour),
				expected:   "",
			},
			{
				name:       "disabled",
				exitCode:   ptr.Int64(-2),
				executedAt: now.Add(-1 * time.Hour),
				expected:   fleet.RunScriptDisabledErrMsg,
			},
		} {
			t.Run(c.name, func(t *testing.T) {
				insertResults(t, host0.ID, &fleet.Script{ID: oldScriptID, Name: "test-script-details-timeout.sh"}, c.executedAt, "test-user-message_"+c.name, c.exitCode)

				var resp getScriptResultResponse
				s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/scripts/results/%s", "test-user-message_"+c.name), nil, http.StatusOK, &resp)
				require.Equal(t, c.expected, resp.Message)
			})
		}
	})
}

// generates the body and headers part of a multipart request ready to be
// used via s.DoRawWithHeaders to POST /api/_version_/fleet/scripts.
func generateNewScriptMultipartRequest(t *testing.T,
	fileName string, fileContent []byte, token string, extraFields map[string][]string,
) (*bytes.Buffer, map[string]string) {
	return generateMultipartRequest(t, "script", fileName, fileContent, token, extraFields)
}

func (s *integrationEnterpriseTestSuite) TestAppConfigScripts() {
	t := s.T()

	// set the script fields
	acResp := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{ "scripts": ["foo", "bar"] }`), http.StatusOK, &acResp)
	assert.ElementsMatch(t, []string{"foo", "bar"}, acResp.Scripts.Value)

	// check that they are returned by a GET /config
	acResp = appConfigResponse{}
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	assert.ElementsMatch(t, []string{"foo", "bar"}, acResp.Scripts.Value)

	// patch without specifying the scripts fields, should not remove them
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{}`), http.StatusOK, &acResp)
	assert.ElementsMatch(t, []string{"foo", "bar"}, acResp.Scripts.Value)

	// patch with explicitly empty scripts fields, would remove
	// them but this is a dry-run
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{ "scripts": null }`), http.StatusOK, &acResp, "dry_run", "true")
	assert.ElementsMatch(t, []string{"foo", "bar"}, acResp.Scripts.Value)

	// patch with explicitly empty scripts fields, removes them
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{ "scripts": null }`), http.StatusOK, &acResp)
	assert.Empty(t, acResp.Scripts.Value)

	// set the script fields again
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{ "scripts": ["foo", "bar"] }`), http.StatusOK, &acResp)
	assert.ElementsMatch(t, []string{"foo", "bar"}, acResp.Scripts.Value)

	// patch with an empty array sets the scripts to an empty array as well
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{ "scripts": [] }`), http.StatusOK, &acResp)
	assert.Empty(t, acResp.Scripts.Value)

	// patch with an invalid array returns an error
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{ "scripts": ["foo", 1] }`), http.StatusBadRequest, &acResp)
	assert.Empty(t, acResp.Scripts.Value)
}

func (s *integrationEnterpriseTestSuite) TestApplyTeamsScriptsConfig() {
	t := s.T()

	// create a team through the service so it initializes the agent ops
	teamName := t.Name() + "team1"
	team := &fleet.Team{
		Name:        teamName,
		Description: "desc team1",
	}
	var createTeamResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", team, http.StatusOK, &createTeamResp)
	require.NotZero(t, createTeamResp.Team.ID)
	team = createTeamResp.Team

	// apply with scripts
	// must not use applyTeamSpecsRequest and marshal it as JSON, as it will set
	// all keys to their zerovalue, and some are only valid with mdm enabled.
	teamSpecs := map[string]any{
		"specs": []any{
			map[string]any{
				"name":    teamName,
				"scripts": []string{"foo", "bar"},
			},
		},
	}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK)

	// retrieving the team returns the scripts
	var teamResp getTeamResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &teamResp)
	require.Equal(t, []string{"foo", "bar"}, teamResp.Team.Config.Scripts.Value)

	// apply without custom scripts specified, should not replace existing scripts
	teamSpecs = map[string]any{
		"specs": []any{
			map[string]any{
				"name": teamName,
			},
		},
	}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK)
	teamResp = getTeamResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &teamResp)
	require.Equal(t, []string{"foo", "bar"}, teamResp.Team.Config.Scripts.Value)

	// apply with explicitly empty custom scripts would clear the existing
	// scripts, but dry-run
	teamSpecs = map[string]any{
		"specs": []any{
			map[string]any{
				"name":    teamName,
				"scripts": nil,
			},
		},
	}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK, "dry_run", "true")
	teamResp = getTeamResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &teamResp)
	require.Equal(t, []string{"foo", "bar"}, teamResp.Team.Config.Scripts.Value)

	// apply with explicitly empty scripts clears the existing scripts
	teamSpecs = map[string]any{
		"specs": []any{
			map[string]any{
				"name":    teamName,
				"scripts": nil,
			},
		},
	}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusOK)
	teamResp = getTeamResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &teamResp)
	require.Empty(t, teamResp.Team.Config.Scripts.Value)

	// patch with an invalid array returns an error
	teamSpecs = map[string]any{
		"specs": []any{
			map[string]any{
				"name":    teamName,
				"scripts": []any{"foo", 1},
			},
		},
	}
	s.Do("POST", "/api/latest/fleet/spec/teams", teamSpecs, http.StatusBadRequest)
	teamResp = getTeamResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d", team.ID), nil, http.StatusOK, &teamResp)
	require.Empty(t, teamResp.Team.Config.Scripts.Value)
}

func (s *integrationEnterpriseTestSuite) TestBatchApplyScriptsEndpoints() {
	t := s.T()
	ctx := context.Background()

	saveAndCheckScripts := func(team *fleet.Team, scripts []fleet.ScriptPayload) {
		var teamID *uint
		teamIDStr := ""
		teamActivity := `{"team_id": null, "team_name": null}`
		if team != nil {
			teamID = &team.ID
			teamIDStr = strconv.Itoa(int(team.ID))
			teamActivity = fmt.Sprintf(`{"team_id": %d, "team_name": %q}`, team.ID, team.Name)
		}

		// create and check activities
		s.Do("POST", "/api/v1/fleet/scripts/batch", batchSetScriptsRequest{Scripts: scripts}, http.StatusNoContent, "team_id", teamIDStr)
		s.lastActivityMatches(
			fleet.ActivityTypeEditedScript{}.ActivityName(),
			teamActivity,
			0,
		)

		// check that the right values got stored in the db
		var listResp listScriptsResponse
		s.DoJSON("GET", "/api/latest/fleet/scripts", nil, http.StatusOK, &listResp, "team_id", teamIDStr)
		require.Len(t, listResp.Scripts, len(scripts))

		got := make([]fleet.ScriptPayload, len(scripts))
		for i, gotScript := range listResp.Scripts {
			// add the script contents
			res := s.Do("GET", fmt.Sprintf("/api/latest/fleet/scripts/%d", gotScript.ID), nil, http.StatusOK, "alt", "media")
			b, err := io.ReadAll(res.Body)
			require.NoError(t, err)
			got[i] = fleet.ScriptPayload{
				Name:           gotScript.Name,
				ScriptContents: b,
			}
			// check that it belongs to the right team
			require.Equal(t, teamID, gotScript.TeamID)
		}

		require.ElementsMatch(t, scripts, got)
	}

	// create a new team
	tm, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "batch_set_scripts"})
	require.NoError(t, err)

	// apply an empty set to no-team
	saveAndCheckScripts(nil, nil)

	// apply to both team id and name
	s.Do("POST", "/api/v1/fleet/scripts/batch", batchSetScriptsRequest{Scripts: nil},
		http.StatusUnprocessableEntity, "team_id", strconv.Itoa(int(tm.ID)), "team_name", tm.Name)

	// invalid team name
	s.Do("POST", "/api/v1/fleet/scripts/batch", batchSetScriptsRequest{Scripts: nil},
		http.StatusNotFound, "team_name", uuid.New().String())

	// duplicate script names
	s.Do("POST", "/api/v1/fleet/scripts/batch", batchSetScriptsRequest{Scripts: []fleet.ScriptPayload{
		{Name: "N1.sh", ScriptContents: []byte("foo")},
		{Name: "N1.sh", ScriptContents: []byte("bar")},
	}}, http.StatusUnprocessableEntity, "team_id", strconv.Itoa(int(tm.ID)))

	// invalid script name
	s.Do("POST", "/api/v1/fleet/scripts/batch", batchSetScriptsRequest{Scripts: []fleet.ScriptPayload{
		{Name: "N1", ScriptContents: []byte("foo")},
	}}, http.StatusUnprocessableEntity, "team_id", strconv.Itoa(int(tm.ID)))

	// empty script name
	s.Do("POST", "/api/v1/fleet/scripts/batch", batchSetScriptsRequest{Scripts: []fleet.ScriptPayload{
		{Name: "", ScriptContents: []byte("foo")},
	}}, http.StatusUnprocessableEntity, "team_id", strconv.Itoa(int(tm.ID)))

	// successfully apply a scripts for the team
	saveAndCheckScripts(tm, []fleet.ScriptPayload{
		{Name: "N1.sh", ScriptContents: []byte("foo")},
		{Name: "N2.sh", ScriptContents: []byte("bar")},
	})

	// successfully apply scripts for "no team"
	saveAndCheckScripts(nil, []fleet.ScriptPayload{
		{Name: "N1.sh", ScriptContents: []byte("foo")},
		{Name: "N2.sh", ScriptContents: []byte("bar")},
	})

	// edit, delete and add a new one for "no team"
	saveAndCheckScripts(nil, []fleet.ScriptPayload{
		{Name: "N2.sh", ScriptContents: []byte("bar-edited")},
		{Name: "N3.sh", ScriptContents: []byte("baz")},
	})

	// edit, delete and add a new one for the team
	saveAndCheckScripts(tm, []fleet.ScriptPayload{
		{Name: "N2.sh", ScriptContents: []byte("bar-edited")},
		{Name: "N3.sh", ScriptContents: []byte("baz")},
	})

	// remove all scripts for a team
	saveAndCheckScripts(tm, nil)

	// remove all scripts for "no team"
	saveAndCheckScripts(nil, nil)
}

func (s *integrationEnterpriseTestSuite) TestTeamConfigDetailQueriesOverrides() {
	ctx := context.Background()
	t := s.T()

	teamName := t.Name() + "team1"
	team := &fleet.Team{
		Name:        teamName,
		Description: "desc team1",
	}
	s.Do("POST", "/api/latest/fleet/teams", team, http.StatusOK)

	spec := []byte(fmt.Sprintf(`
  name: %s
  features:
    additional_queries:
      time: SELECT * FROM time
    enable_host_users: true
    detail_query_overrides:
      users: null
      software_linux: "select * from blah;"
      disk_encryption_linux: null
`, teamName))

	s.applyTeamSpec(spec)
	team, err := s.ds.TeamByName(ctx, teamName)
	require.NoError(t, err)
	require.NotNil(t, team.Config.Features.DetailQueryOverrides)
	require.Nil(t, team.Config.Features.DetailQueryOverrides["users"])
	require.Nil(t, team.Config.Features.DetailQueryOverrides["disk_encryption_linux"])
	require.NotNil(t, team.Config.Features.DetailQueryOverrides["software_linux"])
	require.Equal(t, "select * from blah;", *team.Config.Features.DetailQueryOverrides["software_linux"])

	// create a linux host
	linuxHost, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now().Add(-10 * time.Hour),
		LabelUpdatedAt:  time.Now().Add(-10 * time.Hour),
		PolicyUpdatedAt: time.Now().Add(-10 * time.Hour),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   ptr.String(t.Name()),
		NodeKey:         ptr.String(t.Name()),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%sfoo.local", t.Name()),
		Platform:        "linux",
	})
	require.NoError(t, err)

	// add the host to team1
	err = s.ds.AddHostsToTeam(context.Background(), &team.ID, []uint{linuxHost.ID})
	require.NoError(t, err)

	// get distributed queries for the host
	s.lq.On("QueriesForHost", linuxHost.ID).Return(map[string]string{t.Name(): "select 1 from osquery;"}, nil)
	req := getDistributedQueriesRequest{NodeKey: *linuxHost.NodeKey}
	var dqResp getDistributedQueriesResponse
	s.DoJSON("POST", "/api/osquery/distributed/read", req, http.StatusOK, &dqResp)
	require.NotContains(t, dqResp.Queries, "fleet_detail_query_users")
	require.NotContains(t, dqResp.Queries, "fleet_detail_query_disk_encryption_linux")
	require.Contains(t, dqResp.Queries, "fleet_detail_query_software_linux")
	require.Contains(t, dqResp.Queries, fmt.Sprintf("fleet_distributed_query_%s", t.Name()))

	spec = []byte(fmt.Sprintf(`
  name: %s
  features:
    additional_queries:
      time: SELECT * FROM time
    enable_host_users: true
    detail_query_overrides:
      software_linux: "select * from blah;"
`, teamName))

	s.applyTeamSpec(spec)
	team, err = s.ds.TeamByName(ctx, teamName)
	require.NoError(t, err)
	require.NotNil(t, team.Config.Features.DetailQueryOverrides)
	require.Nil(t, team.Config.Features.DetailQueryOverrides["users"])
	require.Nil(t, team.Config.Features.DetailQueryOverrides["disk_encryption_linux"])
	require.NotNil(t, team.Config.Features.DetailQueryOverrides["software_linux"])
	require.Equal(t, "select * from blah;", *team.Config.Features.DetailQueryOverrides["software_linux"])

	// get distributed queries for the host
	req = getDistributedQueriesRequest{NodeKey: *linuxHost.NodeKey}
	dqResp = getDistributedQueriesResponse{}
	s.DoJSON("POST", "/api/osquery/distributed/read", req, http.StatusOK, &dqResp)
	require.Contains(t, dqResp.Queries, "fleet_detail_query_users")
	require.Contains(t, dqResp.Queries, "fleet_detail_query_disk_encryption_linux")
	require.Contains(t, dqResp.Queries, "fleet_detail_query_software_linux")
	require.Contains(t, dqResp.Queries, fmt.Sprintf("fleet_distributed_query_%s", t.Name()))
}

func (s *integrationEnterpriseTestSuite) TestAllSoftwareTitles() {
	ctx := context.Background()
	t := s.T()

	softwareTitlesMatch := func(want, got []fleet.SoftwareTitle) {
		// compare only the fields we care about
		for i := range got {
			require.NotZero(t, got[i].ID)
			got[i].ID = 0

			for j := range got[i].Versions {
				require.NotZero(t, got[i].Versions[j].ID)
				got[i].Versions[j].ID = 0
			}
		}

		// sort and use EqualValues instead of ElementsMatch in order
		// to do a deep comparison of nested structures
		sort.Slice(got, func(i, j int) bool {
			return got[i].Name < got[j].Name
		})
		sort.Slice(want, func(i, j int) bool {
			return want[i].Name < want[j].Name
		})

		require.EqualValues(t, want, got)
	}

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

	tmHost, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   ptr.String(t.Name() + "tm"),
		NodeKey:         ptr.String(t.Name() + "tm"),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%sfoo.local", t.Name()+"tm"),
		Platform:        "linux",
	})
	require.NoError(t, err)

	// create a couple of teams and add tmHost to one
	team1, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "team1"})
	require.NoError(t, err)
	team2, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "team2"})
	require.NoError(t, err)
	require.NoError(t, s.ds.AddHostsToTeam(ctx, &team1.ID, []uint{tmHost.ID}))

	software := []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "homebrew"},
		{Name: "foo", Version: "0.0.3", Source: "homebrew"},
		{Name: "bar", Version: "0.0.4", Source: "apps"},
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

	// calculate hosts counts
	hostsCountTs := time.Now().UTC()
	require.NoError(t, s.ds.SyncHostsSoftware(ctx, hostsCountTs))
	require.NoError(t, s.ds.ReconcileSoftwareTitles(ctx))
	require.NoError(t, s.ds.SyncHostsSoftwareTitles(ctx, hostsCountTs))

	var resp listSoftwareTitlesResponse
	s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{}, http.StatusOK, &resp)
	require.Equal(t, 2, resp.Count)
	require.NotEmpty(t, resp.CountsUpdatedAt)
	softwareTitlesMatch([]fleet.SoftwareTitle{
		{
			Name:          "foo",
			Source:        "homebrew",
			VersionsCount: 2,
			HostsCount:    1,
			Versions: []fleet.SoftwareVersion{
				{Version: "0.0.1", Vulnerabilities: nil},
				{Version: "0.0.3", Vulnerabilities: nil},
			},
		},
		{
			Name:          "bar",
			Source:        "apps",
			VersionsCount: 1,
			HostsCount:    1,
			Versions: []fleet.SoftwareVersion{
				{Version: "0.0.4", Vulnerabilities: &fleet.SliceString{"cve-123-123-132"}},
			},
		},
	}, resp.SoftwareTitles)

	// per_page equals 1, so we get only one item, but the total count is
	// still 2
	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"per_page", "1",
		"order_key", "name",
		"order_direction", "desc",
	)
	require.Equal(t, 2, resp.Count)
	require.NotEmpty(t, resp.CountsUpdatedAt)
	softwareTitlesMatch([]fleet.SoftwareTitle{
		{
			Name:          "foo",
			Source:        "homebrew",
			VersionsCount: 2,
			HostsCount:    1,
			Versions: []fleet.SoftwareVersion{
				{Version: "0.0.1", Vulnerabilities: nil},
				{Version: "0.0.3", Vulnerabilities: nil},
			},
		},
	}, resp.SoftwareTitles)

	// get the second item
	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"per_page", "1",
		"page", "1",
		"order_key", "name",
		"order_direction", "desc",
	)
	require.Equal(t, 2, resp.Count)
	require.NotEmpty(t, resp.CountsUpdatedAt)
	softwareTitlesMatch([]fleet.SoftwareTitle{
		{
			Name:          "bar",
			Source:        "apps",
			VersionsCount: 1,
			HostsCount:    1,
			Versions: []fleet.SoftwareVersion{
				{Version: "0.0.4", Vulnerabilities: &fleet.SliceString{"cve-123-123-132"}},
			},
		},
	}, resp.SoftwareTitles)

	// asking for a non-existent page returns an empty list
	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"per_page", "1",
		"page", "4",
		"order_key", "name",
		"order_direction", "desc",
	)
	require.Equal(t, 2, resp.Count)
	require.Empty(t, resp.CountsUpdatedAt)
	softwareTitlesMatch(nil, resp.SoftwareTitles)

	// asking for vulnerable only software returns the expected values
	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"vulnerable", "true",
	)
	require.Equal(t, 1, resp.Count)
	require.NotEmpty(t, resp.CountsUpdatedAt)
	softwareTitlesMatch([]fleet.SoftwareTitle{
		{
			Name:          "bar",
			Source:        "apps",
			VersionsCount: 1,
			HostsCount:    1,
			Versions: []fleet.SoftwareVersion{
				{Version: "0.0.4", Vulnerabilities: &fleet.SliceString{"cve-123-123-132"}},
			},
		},
	}, resp.SoftwareTitles)

	// request titles for team1, nothing there yet
	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"team_id", "1",
	)
	require.Equal(t, 0, resp.Count)
	require.Empty(t, resp.CountsUpdatedAt)
	softwareTitlesMatch(nil, resp.SoftwareTitles)

	// add new software for tmHost
	software = []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "homebrew"},
		{Name: "baz", Version: "0.0.5", Source: "deb_packages"},
	}
	_, err = s.ds.UpdateHostSoftware(context.Background(), tmHost.ID, software)
	require.NoError(t, err)

	// calculate hosts counts
	hostsCountTs = time.Now().UTC()
	require.NoError(t, s.ds.SyncHostsSoftware(context.Background(), hostsCountTs))
	require.NoError(t, s.ds.ReconcileSoftwareTitles(ctx))
	require.NoError(t, s.ds.SyncHostsSoftwareTitles(ctx, hostsCountTs))

	// request software for the team, this time we get results
	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"team_id", "1",
		"order_key", "name",
		"order_direction", "desc",
	)
	require.Equal(t, 2, resp.Count)
	require.NotEmpty(t, resp.CountsUpdatedAt)
	softwareTitlesMatch([]fleet.SoftwareTitle{
		{
			Name:          "baz",
			Source:        "deb_packages",
			VersionsCount: 1,
			HostsCount:    1,
			Versions: []fleet.SoftwareVersion{
				{Version: "0.0.5", Vulnerabilities: nil},
			},
		},
		{
			Name:          "foo",
			Source:        "homebrew",
			VersionsCount: 1, // NOTE: this value is 1 because the team has only 1 matching host in the team
			HostsCount:    1, // NOTE: this value is 1 because the team has only 1 matching host in the team
			Versions: []fleet.SoftwareVersion{
				{Version: "0.0.1", Vulnerabilities: nil}, // NOTE: this only includes versions present in the team
			},
		},
	}, resp.SoftwareTitles)

	// request software for no-team, we get all results and 2 hosts for
	// `"foo"`
	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"order_key", "name",
		"order_direction", "desc",
	)
	require.Equal(t, 3, resp.Count)
	require.NotEmpty(t, resp.CountsUpdatedAt)
	softwareTitlesMatch([]fleet.SoftwareTitle{
		{
			Name:          "baz",
			Source:        "deb_packages",
			VersionsCount: 1,
			HostsCount:    1,
			Versions: []fleet.SoftwareVersion{
				{Version: "0.0.5", Vulnerabilities: nil},
			},
		},
		{
			Name:          "foo",
			Source:        "homebrew",
			VersionsCount: 2, // NOTE: this value is 2, important because no team filter was applied
			HostsCount:    2, // NOTE: this value is 2, important because no team filter was applied
			Versions: []fleet.SoftwareVersion{
				{Version: "0.0.1", Vulnerabilities: nil},
				{Version: "0.0.3", Vulnerabilities: nil},
			},
		},
		{
			Name:          "bar",
			Source:        "apps",
			VersionsCount: 1,
			HostsCount:    1,
			Versions: []fleet.SoftwareVersion{
				{Version: "0.0.4", Vulnerabilities: &fleet.SliceString{"cve-123-123-132"}},
			},
		},
	}, resp.SoftwareTitles)

	// match cve by name
	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"query", "123",
	)
	require.Equal(t, 1, resp.Count)
	softwareTitlesMatch([]fleet.SoftwareTitle{
		{
			Name:          "bar",
			Source:        "apps",
			VersionsCount: 1,
			HostsCount:    1,
			Versions: []fleet.SoftwareVersion{
				{Version: "0.0.4", Vulnerabilities: &fleet.SliceString{"cve-123-123-132"}},
			},
		},
	}, resp.SoftwareTitles)

	// match software title by name
	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"query", "ba",
	)
	require.Equal(t, 2, resp.Count)
	require.NotEmpty(t, resp.CountsUpdatedAt)
	softwareTitlesMatch([]fleet.SoftwareTitle{
		{
			Name:          "bar",
			Source:        "apps",
			VersionsCount: 1,
			HostsCount:    1,
			Versions: []fleet.SoftwareVersion{
				{Version: "0.0.4", Vulnerabilities: &fleet.SliceString{"cve-123-123-132"}},
			},
		},
		{
			Name:          "baz",
			Source:        "deb_packages",
			VersionsCount: 1,
			HostsCount:    1,
			Versions: []fleet.SoftwareVersion{
				{Version: "0.0.5", Vulnerabilities: nil},
			},
		},
	}, resp.SoftwareTitles)

	// find the ID of "foo"
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"query", "foo",
	)
	require.Equal(t, 1, resp.Count)
	require.Len(t, resp.SoftwareTitles, 1)
	require.NotEmpty(t, resp.CountsUpdatedAt)
	fooTitle := resp.SoftwareTitles[0]
	require.Equal(t, "foo", fooTitle.Name)

	// non-existent id is a 404
	var stResp getSoftwareTitleResponse
	s.DoJSON("GET", "/api/latest/fleet/software/titles/999", getSoftwareTitleRequest{}, http.StatusNotFound, &stResp)

	// valid title
	stResp = getSoftwareTitleResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", fooTitle.ID), getSoftwareTitleRequest{}, http.StatusOK, &stResp)
	softwareTitlesMatch([]fleet.SoftwareTitle{
		{
			Name:          "foo",
			Source:        "homebrew",
			VersionsCount: 2,
			HostsCount:    2,
			Versions: []fleet.SoftwareVersion{
				{Version: "0.0.1", Vulnerabilities: nil, HostsCount: ptr.Uint(2)},
				{Version: "0.0.3", Vulnerabilities: nil, HostsCount: ptr.Uint(1)},
			},
		},
	}, []fleet.SoftwareTitle{*stResp.SoftwareTitle})

	// valid title for team
	stResp = getSoftwareTitleResponse{}
	s.DoJSON(
		"GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", fooTitle.ID), getSoftwareTitleRequest{}, http.StatusOK, &stResp,
		"team_id", fmt.Sprintf("%d", team1.ID),
	)
	softwareTitlesMatch(
		[]fleet.SoftwareTitle{
			{
				Name:          "foo",
				Source:        "homebrew",
				VersionsCount: 1,
				HostsCount:    1,
				Versions: []fleet.SoftwareVersion{
					{Version: "0.0.1", Vulnerabilities: nil, HostsCount: ptr.Uint(1)},
				},
			},
		}, []fleet.SoftwareTitle{*stResp.SoftwareTitle},
	)

	// find the ID of "bar"
	resp = listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &resp,
		"query", "bar",
	)
	require.Equal(t, 1, resp.Count)
	require.Len(t, resp.SoftwareTitles, 1)
	barTitle := resp.SoftwareTitles[0]
	require.Equal(t, "bar", barTitle.Name)

	// valid title with vulnerabilities
	stResp = getSoftwareTitleResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", barTitle.ID), getSoftwareTitleRequest{}, http.StatusOK, &stResp)
	softwareTitlesMatch([]fleet.SoftwareTitle{
		{
			Name:          "bar",
			Source:        "apps",
			VersionsCount: 1,
			HostsCount:    1,
			Versions: []fleet.SoftwareVersion{
				{
					Version:         "0.0.4",
					Vulnerabilities: &fleet.SliceString{"cve-123-123-132"},
					HostsCount:      ptr.Uint(1),
				},
			},
		},
	}, []fleet.SoftwareTitle{*stResp.SoftwareTitle})

	// invalid title for team
	stResp = getSoftwareTitleResponse{}
	s.DoJSON(
		"GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", barTitle.ID), getSoftwareTitleRequest{}, http.StatusNotFound, &stResp,
		"team_id", fmt.Sprintf("%d", team1.ID),
	)

	// add bar tmHost
	software = []fleet.Software{
		{Name: "bar", Version: "0.0.4", Source: "apps"},
	}
	_, err = s.ds.UpdateHostSoftware(context.Background(), tmHost.ID, software)
	require.NoError(t, err)

	// calculate hosts counts
	hostsCountTs = time.Now().UTC()
	require.NoError(t, s.ds.SyncHostsSoftware(context.Background(), hostsCountTs))
	require.NoError(t, s.ds.ReconcileSoftwareTitles(ctx))
	require.NoError(t, s.ds.SyncHostsSoftwareTitles(ctx, hostsCountTs))

	// valid title with vulnerabilities
	stResp = getSoftwareTitleResponse{}
	s.DoJSON(
		"GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", barTitle.ID), getSoftwareTitleRequest{}, http.StatusOK, &stResp,
		"team_id", fmt.Sprintf("%d", team1.ID),
	)
	softwareTitlesMatch(
		[]fleet.SoftwareTitle{
			{
				Name:          "bar",
				Source:        "apps",
				VersionsCount: 1,
				HostsCount:    1,
				Versions: []fleet.SoftwareVersion{
					{
						Version:         "0.0.4",
						Vulnerabilities: &fleet.SliceString{"cve-123-123-132"},
						HostsCount:      ptr.Uint(1),
					},
				},
			},
		}, []fleet.SoftwareTitle{*stResp.SoftwareTitle},
	)

	// Team without hosts
	s.DoJSON(
		"GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", barTitle.ID), getSoftwareTitleRequest{}, http.StatusNotFound, &stResp,
		"team_id", fmt.Sprintf("%d", team2.ID),
	)

	// Non-existent team
	s.DoJSON(
		"GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", barTitle.ID), getSoftwareTitleRequest{}, http.StatusNotFound, &stResp,
		"team_id", "99999",
	)

}

func (s *integrationEnterpriseTestSuite) TestLockUnlockWindowsLinux() {
	ctx := context.Background()
	t := s.T()

	// create a Windows and a Linux hosts
	winHost := createOrbitEnrolledHost(t, "windows", "win_lock_unlock", s.ds)
	linuxHost := createOrbitEnrolledHost(t, "linux", "linux_lock_unlock", s.ds)

	// get the host's information
	var getHostResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", winHost.ID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host.MDM.DeviceStatus)
	require.Equal(t, "unlocked", *getHostResp.Host.MDM.DeviceStatus)
	require.NotNil(t, getHostResp.Host.MDM.PendingAction)
	require.Equal(t, "", *getHostResp.Host.MDM.PendingAction)

	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", linuxHost.ID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host.MDM.DeviceStatus)
	require.Equal(t, "unlocked", *getHostResp.Host.MDM.DeviceStatus)
	require.NotNil(t, getHostResp.Host.MDM.PendingAction)
	require.Equal(t, "", *getHostResp.Host.MDM.PendingAction)

	// try to lock/unlock the Windows host, fails because Windows MDM must be enabled
	res := s.DoRaw("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/lock", winHost.ID), nil, http.StatusBadRequest)
	errMsg := extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Windows MDM isn't turned on.")
	res = s.DoRaw("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/unlock", winHost.ID), nil, http.StatusBadRequest)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Windows MDM isn't turned on.")

	// try to lock/unlock the Linux host succeeds, no MDM constraints
	s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/lock", linuxHost.ID), nil, http.StatusNoContent)

	// simulate a successful script result for the lock command
	status, err := s.ds.GetHostLockWipeStatus(ctx, linuxHost.ID, linuxHost.FleetPlatform())
	require.NoError(t, err)

	var orbitScriptResp orbitPostScriptResultResponse
	s.DoJSON("POST", "/api/fleet/orbit/scripts/result",
		json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q, "execution_id": %q, "exit_code": 0, "output": "ok"}`, *linuxHost.OrbitNodeKey, status.LockScript.ExecutionID)),
		http.StatusOK, &orbitScriptResp)

	s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/unlock", linuxHost.ID), nil, http.StatusNoContent)

	// windows host status is unchanged, linux is locked pending unlock
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", winHost.ID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host.MDM.DeviceStatus)
	require.Equal(t, "unlocked", *getHostResp.Host.MDM.DeviceStatus)
	require.NotNil(t, getHostResp.Host.MDM.PendingAction)
	require.Equal(t, "", *getHostResp.Host.MDM.PendingAction)

	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", linuxHost.ID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host.MDM.DeviceStatus)
	require.Equal(t, "locked", *getHostResp.Host.MDM.DeviceStatus)
	require.NotNil(t, getHostResp.Host.MDM.PendingAction)
	require.Equal(t, "unlock", *getHostResp.Host.MDM.PendingAction)
}

// checks that the specified team/no-team has the Windows OS Updates profile with
// the specified deadline/grace settings (or checks that it doesn't have the
// profile if wantSettings is nil). It returns the profile_uuid if it exists,
// empty string otherwise.
func checkWindowsOSUpdatesProfile(t *testing.T, ds *mysql.Datastore, teamID *uint, wantSettings *fleet.WindowsUpdates) string {
	ctx := context.Background()

	var prof fleet.MDMWindowsConfigProfile
	mysql.ExecAdhocSQL(t, ds, func(tx sqlx.ExtContext) error {
		var globalOrTeamID uint
		if teamID != nil {
			globalOrTeamID = *teamID
		}
		err := sqlx.GetContext(ctx, tx, &prof, `SELECT profile_uuid, syncml FROM mdm_windows_configuration_profiles WHERE team_id = ? AND name = ?`, globalOrTeamID, mdm.FleetWindowsOSUpdatesProfileName)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	})
	if wantSettings == nil {
		require.Empty(t, prof.ProfileUUID)
	} else {
		require.NotEmpty(t, prof.ProfileUUID)
		require.Contains(t, string(prof.SyncML), fmt.Sprintf(`<Data>%d</Data>`, wantSettings.DeadlineDays.Value))
		require.Contains(t, string(prof.SyncML), fmt.Sprintf(`<Data>%d</Data>`, wantSettings.GracePeriodDays.Value))
	}

	if len(prof.ProfileUUID) > 0 {
		require.Equal(t, byte('w'), prof.ProfileUUID[0])
	}

	return prof.ProfileUUID
}

func (s *integrationEnterpriseTestSuite) createHosts(t *testing.T, platforms ...string) []*fleet.Host {
	var hosts []*fleet.Host
	if len(platforms) == 0 {
		platforms = []string{"debian", "rhel", "linux", "windows", "darwin"}
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

func (s *integrationEnterpriseTestSuite) TestSoftwareAuth() {
	t := s.T()
	ctx := context.Background()
	// create two hosts, one belongs to team1 and one has no team
	host, err := s.ds.NewHost(ctx, &fleet.Host{
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

	tmHost, err := s.ds.NewHost(ctx, &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   ptr.String(t.Name() + "tm"),
		NodeKey:         ptr.String(t.Name() + "tm"),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%sfoo.local", t.Name()+"tm"),
		Platform:        "linux",
	})
	require.NoError(t, err)

	// Create two teams, team1 and team2.
	team1, err := s.ds.NewTeam(ctx, &fleet.Team{
		ID:          42,
		Name:        "team1",
		Description: "desc team1",
	})
	require.NoError(t, err)
	require.NoError(t, s.ds.AddHostsToTeam(ctx, &team1.ID, []uint{tmHost.ID}))
	team2, err := s.ds.NewTeam(ctx, &fleet.Team{
		ID:          43,
		Name:        "team2",
		Description: "desc team2",
	})
	require.NoError(t, err)

	allSoftware := []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "homebrew"},
		{Name: "foo", Version: "0.0.3", Source: "homebrew"},
		{Name: "bar", Version: "0.0.4", Source: "apps"},
	}
	// add all the software entries to the "no team host"
	_, err = s.ds.UpdateHostSoftware(ctx, host.ID, allSoftware)
	require.NoError(t, err)
	require.NoError(t, s.ds.LoadHostSoftware(ctx, host, false))

	// add only one version of "foo" to the team host
	_, err = s.ds.UpdateHostSoftware(ctx, tmHost.ID, []fleet.Software{allSoftware[0]})
	require.NoError(t, err)
	require.NoError(t, s.ds.LoadHostSoftware(ctx, tmHost, false))

	// calculate hosts counts
	hostsCountTs := time.Now().UTC()
	require.NoError(t, s.ds.SyncHostsSoftware(ctx, hostsCountTs))
	require.NoError(t, s.ds.ReconcileSoftwareTitles(ctx))
	require.NoError(t, s.ds.SyncHostsSoftwareTitles(ctx, hostsCountTs))

	// add variations of user roles to different teams
	extraTestUsers := make(map[string]fleet.User)
	for k, u := range map[string]struct {
		Email      string
		GlobalRole *string
		Teams      *[]fleet.UserTeam
	}{
		"team-1-admin": {
			Email: "team-1-admin@example.com",
			Teams: &([]fleet.UserTeam{{
				Team: *team1,
				Role: fleet.RoleAdmin,
			}}),
		},
		"team-1-maintainer": {
			Email: "team-1-maintainer@example.com",
			Teams: &([]fleet.UserTeam{{
				Team: *team1,
				Role: fleet.RoleMaintainer,
			}}),
		},
		"team-1-observer": {
			Email: "team-1-observer@example.com",
			Teams: &([]fleet.UserTeam{{
				Team: *team1,
				Role: fleet.RoleObserver,
			}}),
		},
		"team-2-admin": {
			Email: "team-2-admin@example.com",
			Teams: &([]fleet.UserTeam{{
				Team: *team2,
				Role: fleet.RoleAdmin,
			}}),
		},
		"team-2-maintainer": {
			Email: "team-2-maintainer@example.com",
			Teams: &([]fleet.UserTeam{{
				Team: *team2,
				Role: fleet.RoleMaintainer,
			}}),
		},
		"team-2-observer": {
			Email: "team-2-observer@example.com",
			Teams: &([]fleet.UserTeam{{
				Team: *team2,
				Role: fleet.RoleObserver,
			}}),
		},
	} {
		uu := u
		cur := createUserResponse{}
		s.DoJSON("POST", "/api/latest/fleet/users/admin", createUserRequest{
			UserPayload: fleet.UserPayload{
				Email:                    &uu.Email,
				Password:                 &test.GoodPassword,
				Name:                     &uu.Email,
				Teams:                    uu.Teams,
				AdminForcedPasswordReset: ptr.Bool(false),
			},
		}, http.StatusOK, &cur)
		extraTestUsers[k] = *cur.User
	}

	// List all software titles with an admin
	var listSoftwareTitlesResp listSoftwareTitlesResponse
	s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{}, http.StatusOK, &listSoftwareTitlesResp)

	var softwareFoo, softwareBar *fleet.SoftwareTitle
	for _, s := range listSoftwareTitlesResp.SoftwareTitles {
		s := s
		switch s.Name {
		case "foo":
			softwareFoo = &s
		case "bar":
			softwareBar = &s
		}
	}
	require.NotNil(t, softwareFoo)
	require.NotNil(t, softwareBar)

	var teamFooVersion *fleet.SoftwareVersion
	for _, sv := range softwareFoo.Versions {
		sv := sv
		if sv.Version == "0.0.1" {
			teamFooVersion = &sv
		}
	}
	require.NotNil(t, teamFooVersion)

	for _, tc := range []struct {
		name                 string
		user                 fleet.User
		shouldFailGlobalRead bool
		shouldFailTeamRead   bool
	}{
		{
			name:                 "global-admin",
			user:                 s.users["admin1@example.com"],
			shouldFailGlobalRead: false,
			shouldFailTeamRead:   false,
		},
		{
			name:                 "global-maintainer",
			user:                 s.users["user1@example.com"],
			shouldFailGlobalRead: false,
			shouldFailTeamRead:   false,
		},
		{
			name:                 "global-observer",
			user:                 s.users["user2@example.com"],
			shouldFailGlobalRead: false,
			shouldFailTeamRead:   false,
		},
		{
			name:                 "team-admin-belongs-to-team",
			user:                 extraTestUsers["team-1-admin"],
			shouldFailGlobalRead: true,
			shouldFailTeamRead:   false,
		},
		{
			name:                 "team-maintainer-belongs-to-team",
			user:                 extraTestUsers["team-1-maintainer"],
			shouldFailGlobalRead: true,
			shouldFailTeamRead:   false,
		},
		{
			name:                 "team-observer-belongs-to-team",
			user:                 extraTestUsers["team-1-observer"],
			shouldFailGlobalRead: true,
			shouldFailTeamRead:   false,
		},
		{
			name:                 "team-admin-does-not-belong-to-team",
			user:                 extraTestUsers["team-2-admin"],
			shouldFailGlobalRead: true,
			shouldFailTeamRead:   true,
		},
		{
			name:                 "team-maintainer-does-not-belong-to-team",
			user:                 extraTestUsers["team-2-maintainer"],
			shouldFailGlobalRead: true,
			shouldFailTeamRead:   true,
		},
		{
			name:                 "team-observer-does-not-belong-to-team",
			user:                 extraTestUsers["team-2-observer"],
			shouldFailGlobalRead: true,
			shouldFailTeamRead:   true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// to make the request as the user
			s.token = s.getTestToken(tc.user.Email, test.GoodPassword)

			if tc.shouldFailGlobalRead {
				// List all software titles
				var listSoftwareTitlesResp listSoftwareTitlesResponse
				s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{}, http.StatusForbidden, &listSoftwareTitlesResp)

				// List all software versions
				var resp listSoftwareVersionsResponse
				s.DoJSON("GET", "/api/latest/fleet/software/versions", listSoftwareTitlesRequest{}, http.StatusForbidden, &resp)

				// Get a global software title
				var getSoftwareTitleResp getSoftwareTitleResponse
				s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", softwareBar.ID), getSoftwareTitleRequest{}, http.StatusForbidden, &getSoftwareTitleResp)

				// Get a global software version
				var getSoftwareResp getSoftwareResponse
				s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/versions/%d", softwareBar.Versions[0].ID), getSoftwareRequest{}, http.StatusForbidden, &getSoftwareResp)

				// Get a global software vesion using the deprecated endpoint
				getSoftwareResp = getSoftwareResponse{}
				s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/%d", softwareBar.Versions[0].ID), getSoftwareRequest{}, http.StatusForbidden, &getSoftwareResp)

				// Get a count of software vesions using the deprecated endpoint
				var countSoftwareResp countSoftwareResponse
				s.DoJSON("GET", "/api/latest/fleet/software/count", getSoftwareRequest{}, http.StatusForbidden, &countSoftwareResp)

				// List all software versions using the deprecated endpoint
				var softwareListResp listSoftwareResponse
				s.DoJSON("GET", "/api/latest/fleet/software", listSoftwareRequest{}, http.StatusForbidden, &softwareListResp)
			} else {
				// List all software titles
				var listSoftwareTitlesResp listSoftwareTitlesResponse
				s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{}, http.StatusOK, &listSoftwareTitlesResp)
				require.Equal(t, 2, listSoftwareTitlesResp.Count)
				require.NotEmpty(t, listSoftwareTitlesResp.CountsUpdatedAt)

				// List all software versions
				var resp listSoftwareVersionsResponse
				s.DoJSON("GET", "/api/latest/fleet/software/versions", listSoftwareRequest{}, http.StatusOK, &resp)
				require.Equal(t, 3, resp.Count)
				require.NotEmpty(t, resp.CountsUpdatedAt)

				// Get a global software title
				var getSoftwareTitleResp getSoftwareTitleResponse
				s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", softwareBar.ID), getSoftwareTitleRequest{}, http.StatusOK, &getSoftwareTitleResp)

				// Get a global software version
				var getSoftwareResp getSoftwareResponse
				s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/versions/%d", softwareBar.Versions[0].ID), getSoftwareRequest{}, http.StatusOK, &getSoftwareResp)

				// Get a global software vesion using the deprecated endpoint
				getSoftwareResp = getSoftwareResponse{}
				s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/%d", softwareBar.Versions[0].ID), getSoftwareRequest{}, http.StatusOK, &getSoftwareResp)

				// Get a global count of software vesions using the deprecated endpoint
				var countSoftwareResp countSoftwareResponse
				s.DoJSON("GET", "/api/latest/fleet/software/count", countSoftwareRequest{}, http.StatusOK, &countSoftwareResp)
				require.Equal(t, 3, countSoftwareResp.Count)

				// List all software versions using the deprecated endpoint
				var softwareListResp listSoftwareResponse
				s.DoJSON("GET", "/api/latest/fleet/software", listSoftwareRequest{}, http.StatusOK, &softwareListResp)
				require.Equal(t, countSoftwareResp.Count, 3)
			}

			if tc.shouldFailTeamRead {
				// List all software titles for a team
				var listSoftwareTitlesResp listSoftwareTitlesResponse
				s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{SoftwareTitleListOptions: fleet.SoftwareTitleListOptions{TeamID: &team1.ID}}, http.StatusForbidden, &listSoftwareTitlesResp)

				// List software versions for a team.
				var resp listSoftwareTitlesResponse
				s.DoJSON("GET", "/api/latest/fleet/software/versions", listSoftwareRequest{SoftwareListOptions: fleet.SoftwareListOptions{TeamID: &team1.ID}}, http.StatusForbidden, &resp)

				// Get a team software title
				var getSoftwareTitleResp getSoftwareTitleResponse
				s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", softwareFoo.ID), getSoftwareTitleRequest{}, http.StatusForbidden, &getSoftwareTitleResp)

				// Get a team software version
				var getSoftwareResp getSoftwareResponse
				s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/versions/%d", teamFooVersion.ID), getSoftwareRequest{}, http.StatusForbidden, &getSoftwareResp)

				// Get a team software vesion using the deprecated endpoint
				getSoftwareResp = getSoftwareResponse{}
				s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/%d", teamFooVersion.ID), getSoftwareRequest{}, http.StatusForbidden, &getSoftwareResp)

				// Get a count of team software vesions using the deprecated endpoint
				var countSoftwareResp countSoftwareResponse
				s.DoJSON("GET", "/api/latest/fleet/software/count", getSoftwareRequest{}, http.StatusForbidden, &countSoftwareResp)

				// List all software versions using the deprecated endpoint for a team
				var softwareListResp listSoftwareResponse
				s.DoJSON("GET", "/api/latest/fleet/software", listSoftwareRequest{}, http.StatusForbidden, &softwareListResp)
			} else {
				// List all software titles for a team
				var listSoftwareTitlesResp listSoftwareTitlesResponse
				s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{SoftwareTitleListOptions: fleet.SoftwareTitleListOptions{TeamID: &team1.ID}}, http.StatusOK, &listSoftwareTitlesResp)
				require.Equal(t, 1, listSoftwareTitlesResp.Count)
				require.NotEmpty(t, listSoftwareTitlesResp.CountsUpdatedAt)

				// List software versions for a team.
				var resp listSoftwareTitlesResponse
				s.DoJSON("GET", "/api/latest/fleet/software/versions", listSoftwareRequest{SoftwareListOptions: fleet.SoftwareListOptions{TeamID: &team1.ID}}, http.StatusOK, &resp)
				require.Equal(t, 1, resp.Count)
				require.NotEmpty(t, resp.CountsUpdatedAt)

				// Get a team software title
				var getSoftwareTitleResp getSoftwareTitleResponse
				s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", softwareFoo.ID), getSoftwareTitleRequest{}, http.StatusOK, &getSoftwareTitleResp)

				// Get a team software version
				var getSoftwareResp getSoftwareResponse
				s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/versions/%d", teamFooVersion.ID), getSoftwareRequest{}, http.StatusOK, &getSoftwareResp)

				// Get a team software vesion using the deprecated endpoint
				getSoftwareResp = getSoftwareResponse{}
				s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/%d", teamFooVersion.ID), getSoftwareRequest{}, http.StatusOK, &getSoftwareResp)

				// Get a team count of software vesions using the deprecated endpoint
				var countSoftwareResp countSoftwareResponse
				s.DoJSON("GET", "/api/latest/fleet/software/count", countSoftwareRequest{SoftwareListOptions: fleet.SoftwareListOptions{TeamID: &team1.ID}}, http.StatusOK, &countSoftwareResp)
				require.Equal(t, 1, countSoftwareResp.Count)

				// List all software versions using the deprecated endpoint for a team
				var softwareListResp listSoftwareResponse
				s.DoJSON("GET", "/api/latest/fleet/software", listSoftwareRequest{SoftwareListOptions: fleet.SoftwareListOptions{TeamID: &team1.ID}}, http.StatusOK, &softwareListResp)
				require.Equal(t, countSoftwareResp.Count, 1)
			}
		})
	}

	// set the admin token again to avoid breaking other tests
	s.token = s.getTestAdminToken()

}
