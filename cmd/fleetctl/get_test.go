package main

import (
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/assert"
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
