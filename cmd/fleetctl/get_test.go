package main

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var userRoleList = []*fleet.User{
	&fleet.User{
		UpdateCreateTimestamps: fleet.UpdateCreateTimestamps{
			CreateTimestamp: fleet.CreateTimestamp{CreatedAt: time.Now()},
			UpdateTimestamp: fleet.UpdateTimestamp{UpdatedAt: time.Now()},
		},
		ID:         42,
		Name:       "Test Name admin1@example.com",
		Email:      "admin1@example.com",
		GlobalRole: ptr.String(fleet.RoleAdmin),
	},
	&fleet.User{
		UpdateCreateTimestamps: fleet.UpdateCreateTimestamps{
			CreateTimestamp: fleet.CreateTimestamp{CreatedAt: time.Now()},
			UpdateTimestamp: fleet.UpdateTimestamp{UpdatedAt: time.Now()},
		},
		ID:         23,
		Name:       "Test Name2 admin2@example.com",
		Email:      "admin2@example.com",
		GlobalRole: nil,
		Teams: []fleet.UserTeam{
			fleet.UserTeam{
				Team: fleet.Team{
					ID:        1,
					CreatedAt: time.Now(),
					Name:      "team1",
					UserCount: 1,
					HostCount: 1,
				},
				Role: fleet.RoleMaintainer,
			},
		},
	},
}

func TestGetUserRoles(t *testing.T) {
	server, ds := runServerWithMockedDS(t)
	defer server.Close()

	ds.ListUsersFunc = func(opt fleet.UserListOptions) ([]*fleet.User, error) {
		return userRoleList, nil
	}

	expectedText := `+-------------------------------+-------------+
|             USER              | GLOBAL ROLE |
+-------------------------------+-------------+
| Test Name admin1@example.com  | admin       |
+-------------------------------+-------------+
| Test Name2 admin2@example.com |             |
+-------------------------------+-------------+
`
	expectedYaml := `---
apiVersion: v1
kind: user_roles
spec:
  roles:
    admin1@example.com:
      global_role: admin
      teams: null
    admin2@example.com:
      global_role: null
      teams:
      - role: maintainer
        team: team1
`
	expectedJson := `{"kind":"user_roles","apiVersion":"v1","spec":{"roles":{"admin1@example.com":{"global_role":"admin","teams":null},"admin2@example.com":{"global_role":null,"teams":[{"team":"team1","role":"maintainer"}]}}}}
`

	assert.Equal(t, expectedText, runAppForTest(t, []string{"get", "user_roles"}))
	assert.Equal(t, expectedYaml, runAppForTest(t, []string{"get", "user_roles", "--yaml"}))
	assert.Equal(t, expectedJson, runAppForTest(t, []string{"get", "user_roles", "--json"}))
}

func TestGetTeams(t *testing.T) {
	server, ds := runServerWithMockedDS(t, service.TestServerOpts{Tier: fleet.TierBasic})
	defer server.Close()

	agentOpts := json.RawMessage(`{"config":{"foo":"bar"},"overrides":{"platforms":{"darwin":{"foo":"override"}}}}`)
	ds.ListTeamsFunc = func(filter fleet.TeamFilter, opt fleet.ListOptions) ([]*fleet.Team, error) {
		created_at, err := time.Parse(time.RFC3339, "1999-03-10T02:45:06.371Z")
		require.NoError(t, err)
		return []*fleet.Team{
			&fleet.Team{
				ID:          42,
				CreatedAt:   created_at,
				Name:        "team1",
				Description: "team1 description",
				UserCount:   99,
			},
			&fleet.Team{
				ID:           43,
				CreatedAt:    created_at,
				Name:         "team2",
				Description:  "team2 description",
				UserCount:    87,
				AgentOptions: &agentOpts,
			},
		}, nil
	}

	expectedText := `+-----------+-------------------+------------+
| TEAM NAME |    DESCRIPTION    | USER COUNT |
+-----------+-------------------+------------+
| team1     | team1 description |         99 |
+-----------+-------------------+------------+
| team2     | team2 description |         87 |
+-----------+-------------------+------------+
`
	expectedYaml := `---
apiVersion: v1
kind: team
spec:
  team:
    agent_options: null
    created_at: "1999-03-10T02:45:06.371Z"
    description: team1 description
    host_count: 0
    id: 42
    name: team1
    user_count: 99
---
apiVersion: v1
kind: team
spec:
  team:
    agent_options:
      config:
        foo: bar
      overrides:
        platforms:
          darwin:
            foo: override
    created_at: "1999-03-10T02:45:06.371Z"
    description: team2 description
    host_count: 0
    id: 43
    name: team2
    user_count: 87
`
	expectedJson := `{"kind":"team","apiVersion":"v1","spec":{"team":{"id":42,"created_at":"1999-03-10T02:45:06.371Z","name":"team1","description":"team1 description","agent_options":null,"user_count":99,"host_count":0}}}
{"kind":"team","apiVersion":"v1","spec":{"team":{"id":43,"created_at":"1999-03-10T02:45:06.371Z","name":"team2","description":"team2 description","agent_options":{"config":{"foo":"bar"},"overrides":{"platforms":{"darwin":{"foo":"override"}}}},"user_count":87,"host_count":0}}}
`

	assert.Equal(t, expectedText, runAppForTest(t, []string{"get", "teams"}))
	assert.Equal(t, expectedYaml, runAppForTest(t, []string{"get", "teams", "--yaml"}))
	assert.Equal(t, expectedJson, runAppForTest(t, []string{"get", "teams", "--json"}))
}
