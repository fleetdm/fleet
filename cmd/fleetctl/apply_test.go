package main

import (
	"database/sql"
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var userRoleSpecList = []*fleet.User{
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
		Teams:      []fleet.UserTeam{},
	},
}

func TestApplyUserRoles(t *testing.T) {
	server, ds := runServerWithMockedDS(t)
	defer server.Close()

	ds.ListUsersFunc = func(opt fleet.UserListOptions) ([]*fleet.User, error) {
		return userRoleSpecList, nil
	}

	ds.UserByEmailFunc = func(email string) (*fleet.User, error) {
		if email == "admin1@example.com" {
			return userRoleSpecList[0], nil
		}
		return userRoleSpecList[1], nil
	}

	ds.TeamByNameFunc = func(name string) (*fleet.Team, error) {
		return &fleet.Team{
			ID:        1,
			CreatedAt: time.Now(),
			Name:      "team1",
		}, nil
	}

	ds.SaveUsersFunc = func(users []*fleet.User) error {
		for _, u := range users {
			switch u.Email {
			case "admin1@example.com":
				userRoleList[0] = u
			case "admin2@example.com":
				userRoleList[1] = u
			}
		}
		return nil
	}

	tmpFile, err := ioutil.TempFile(os.TempDir(), "*.yml")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	tmpFile.WriteString(`
---
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
`)

	assert.Equal(t, "[+] applied user roles\n", runAppForTest(t, []string{"apply", "-f", tmpFile.Name()}))
	require.Len(t, userRoleSpecList[1].Teams, 1)
	assert.Equal(t, fleet.RoleMaintainer, userRoleSpecList[1].Teams[0].Role)
}

func TestApplyTeamSpecs(t *testing.T) {
	server, ds := runServerWithMockedDS(t, service.TestServerOpts{Tier: fleet.TierBasic})
	defer server.Close()

	teamsByName := map[string]*fleet.Team{
		"team1": &fleet.Team{
			ID:          42,
			Name:        "team1",
			Description: "team1 description",
		},
	}

	ds.TeamByNameFunc = func(name string) (*fleet.Team, error) {
		team, ok := teamsByName[name]
		if !ok {
			return nil, sql.ErrNoRows
		}
		return team, nil
	}

	i := 1
	ds.NewTeamFunc = func(team *fleet.Team) (*fleet.Team, error) {
		team.ID = uint(i)
		i++
		teamsByName[team.Name] = team
		return team, nil
	}

	agentOpts := json.RawMessage(`{"config":{"foo":"bar"},"overrides":{"platforms":{"darwin":{"foo":"override"}}}}`)
	ds.AppConfigFunc = func() (*fleet.AppConfig, error) {
		return &fleet.AppConfig{AgentOptions: &agentOpts}, nil
	}

	ds.SaveTeamFunc = func(team *fleet.Team) (*fleet.Team, error) {
		teamsByName[team.Name] = team
		return team, nil
	}

	enrolledSecretsCalled := make(map[uint][]*fleet.EnrollSecret)
	ds.ApplyEnrollSecretsFunc = func(teamID *uint, secrets []*fleet.EnrollSecret) error {
		enrolledSecretsCalled[*teamID] = secrets
		return nil
	}

	tmpFile, err := ioutil.TempFile(os.TempDir(), "*.yml")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	tmpFile.WriteString(`
---
apiVersion: v1
kind: team
spec:
  team:
    name: team2
---
apiVersion: v1
kind: team
spec:
  team:
    agent_options:
      config:
        something: else
    name: team1
    secrets:
      - secret: AAA
`)

	newAgentOpts := json.RawMessage("{\"config\":{\"something\":\"else\"}}")

	assert.Equal(t, "[+] applied 2 teams\n", runAppForTest(t, []string{"apply", "-f", tmpFile.Name()}))
	assert.Equal(t, &agentOpts, teamsByName["team2"].AgentOptions)
	assert.Equal(t, &newAgentOpts, teamsByName["team1"].AgentOptions)
	assert.Equal(t, []*fleet.EnrollSecret{{Secret: "AAA"}}, enrolledSecretsCalled[uint(42)])
}
