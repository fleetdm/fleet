package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
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
	{
		UpdateCreateTimestamps: fleet.UpdateCreateTimestamps{
			CreateTimestamp: fleet.CreateTimestamp{CreatedAt: time.Now()},
			UpdateTimestamp: fleet.UpdateTimestamp{UpdatedAt: time.Now()},
		},
		ID:         42,
		Name:       "Test Name admin1@example.com",
		Email:      "admin1@example.com",
		GlobalRole: ptr.String(fleet.RoleAdmin),
	},
	{
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
	_, ds := runServerWithMockedDS(t)

	ds.ListUsersFunc = func(ctx context.Context, opt fleet.UserListOptions) ([]*fleet.User, error) {
		return userRoleSpecList, nil
	}

	ds.UserByEmailFunc = func(ctx context.Context, email string) (*fleet.User, error) {
		if email == "admin1@example.com" {
			return userRoleSpecList[0], nil
		}
		return userRoleSpecList[1], nil
	}

	ds.TeamByNameFunc = func(ctx context.Context, name string) (*fleet.Team, error) {
		return &fleet.Team{
			ID:        1,
			CreatedAt: time.Now(),
			Name:      "team1",
		}, nil
	}

	ds.SaveUsersFunc = func(ctx context.Context, users []*fleet.User) error {
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
	license := &fleet.LicenseInfo{Tier: fleet.TierPremium, Expiration: time.Now().Add(24 * time.Hour)}
	_, ds := runServerWithMockedDS(t, service.TestServerOpts{License: license})

	teamsByName := map[string]*fleet.Team{
		"team1": {
			ID:          42,
			Name:        "team1",
			Description: "team1 description",
		},
	}

	ds.TeamByNameFunc = func(ctx context.Context, name string) (*fleet.Team, error) {
		team, ok := teamsByName[name]
		if !ok {
			return nil, sql.ErrNoRows
		}
		return team, nil
	}

	i := 1
	ds.NewTeamFunc = func(ctx context.Context, team *fleet.Team) (*fleet.Team, error) {
		team.ID = uint(i)
		i++
		teamsByName[team.Name] = team
		return team, nil
	}

	agentOpts := json.RawMessage(`{"config":{"foo":"bar"},"overrides":{"platforms":{"darwin":{"foo":"override"}}}}`)
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{AgentOptions: &agentOpts}, nil
	}

	ds.SaveTeamFunc = func(ctx context.Context, team *fleet.Team) (*fleet.Team, error) {
		teamsByName[team.Name] = team
		return team, nil
	}

	enrolledSecretsCalled := make(map[uint][]*fleet.EnrollSecret)
	ds.ApplyEnrollSecretsFunc = func(ctx context.Context, teamID *uint, secrets []*fleet.EnrollSecret) error {
		enrolledSecretsCalled[*teamID] = secrets
		return nil
	}

	tmpFile, err := ioutil.TempFile(t.TempDir(), "*.yml")
	require.NoError(t, err)

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

func writeTmpYml(t *testing.T, contents string) string {
	tmpFile, err := ioutil.TempFile(t.TempDir(), "*.yml")
	require.NoError(t, err)
	_, err = tmpFile.WriteString(contents)
	require.NoError(t, err)
	return tmpFile.Name()
}

func TestApplyAppConfig(t *testing.T) {
	_, ds := runServerWithMockedDS(t)

	ds.ListUsersFunc = func(ctx context.Context, opt fleet.UserListOptions) ([]*fleet.User, error) {
		return userRoleSpecList, nil
	}

	ds.UserByEmailFunc = func(ctx context.Context, email string) (*fleet.User, error) {
		if email == "admin1@example.com" {
			return userRoleSpecList[0], nil
		}
		return userRoleSpecList[1], nil
	}

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}

	var savedAppConfig *fleet.AppConfig
	ds.SaveAppConfigFunc = func(ctx context.Context, config *fleet.AppConfig) error {
		savedAppConfig = config
		return nil
	}

	name := writeTmpYml(t, `---
apiVersion: v1
kind: config
spec:
  host_settings:
    enable_host_users: false
    enable_software_inventory: false
`)

	assert.Equal(t, "[+] applied fleet config\n", runAppForTest(t, []string{"apply", "-f", name}))
	require.NotNil(t, savedAppConfig)
	assert.False(t, savedAppConfig.HostSettings.EnableHostUsers)
	assert.False(t, savedAppConfig.HostSettings.EnableSoftwareInventory)

	name = writeTmpYml(t, `---
apiVersion: v1
kind: config
spec:
  host_settings:
    enable_host_users: true
    enable_software_inventory: true
`)

	assert.Equal(t, "[+] applied fleet config\n", runAppForTest(t, []string{"apply", "-f", name}))
	require.NotNil(t, savedAppConfig)
	assert.True(t, savedAppConfig.HostSettings.EnableHostUsers)
	assert.True(t, savedAppConfig.HostSettings.EnableSoftwareInventory)
}

func TestApplyAppConfigUnknownFields(t *testing.T) {
	_, ds := runServerWithMockedDS(t)

	ds.ListUsersFunc = func(ctx context.Context, opt fleet.UserListOptions) ([]*fleet.User, error) {
		return userRoleSpecList, nil
	}

	ds.UserByEmailFunc = func(ctx context.Context, email string) (*fleet.User, error) {
		if email == "admin1@example.com" {
			return userRoleSpecList[0], nil
		}
		return userRoleSpecList[1], nil
	}

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}

	var savedAppConfig *fleet.AppConfig
	ds.SaveAppConfigFunc = func(ctx context.Context, config *fleet.AppConfig) error {
		savedAppConfig = config
		return nil
	}

	name := writeTmpYml(t, `---
apiVersion: v1
kind: config
spec:
  host_settings:
    enabled_software_inventory: false # typo, correct config is enable_software_inventory
`)

	runAppCheckErr(t, []string{"apply", "-f", name},
		"applying fleet config: PATCH /api/v1/fleet/config received status 400 Bad request: json: unknown field \"enabled_software_inventory\"",
	)
	require.Nil(t, savedAppConfig)
}

func TestApplyPolicies(t *testing.T) {
	_, ds := runServerWithMockedDS(t)

	var appliedPolicySpecs []*fleet.PolicySpec
	ds.ApplyPolicySpecsFunc = func(ctx context.Context, authorID uint, specs []*fleet.PolicySpec) error {
		appliedPolicySpecs = specs
		return nil
	}
	ds.TeamByNameFunc = func(ctx context.Context, name string) (*fleet.Team, error) {
		if name == "Team1" {
			return &fleet.Team{ID: 123}, nil
		}
		return nil, fmt.Errorf("unexpected team name!")
	}

	name := writeTmpYml(t, `---
apiVersion: v1
kind: policy
spec:
  name: Is Gatekeeper enabled on macOS devices?
  query: SELECT 1 FROM gatekeeper WHERE assessments_enabled = 1;
  description: Checks to make sure that the Gatekeeper feature is enabled on macOS devices. Gatekeeper tries to ensure only trusted software is run on a mac machine.
  resolution: "Run the following command in the Terminal app: /usr/sbin/spctl --master-enable"
  platform: darwin
  team: Team1
---
apiVersion: v1
kind: policy
spec:
  name: Is disk encryption enabled on Windows devices?
  query: SELECT 1 FROM bitlocker_info where protection_status = 1;
  description: Checks to make sure that device encryption is enabled on Windows devices.
  resolution: "Option 1: Select the Start button. Select Settings  > Update & Security  > Device encryption. If Device encryption doesn't appear, skip to Option 2. If device encryption is turned off, select Turn on. Option 2: Select the Start button. Under Windows System, select Control Panel. Select System and Security. Under BitLocker Drive Encryption, select Manage BitLocker. Select Turn on BitLocker and then follow the instructions."
  platform: windows
---
apiVersion: v1
kind: policy
spec:
  name: Is Filevault enabled on macOS devices?
  query: SELECT 1 FROM disk_encryption WHERE user_uuid IS NOT “” AND filevault_status = ‘on’ LIMIT 1;
  description: Checks to make sure that the Filevault feature is enabled on macOS devices.
  resolution: "Choose Apple menu > System Preferences, then click Security & Privacy. Click the FileVault tab. Click the Lock icon, then enter an administrator name and password. Click Turn On FileVault."
  platform: darwin
`)

	assert.Equal(t, "[+] applied 3 policies\n", runAppForTest(t, []string{"apply", "-f", name}))
	assert.True(t, ds.ApplyPolicySpecsFuncInvoked)
	assert.Len(t, appliedPolicySpecs, 3)
	for _, p := range appliedPolicySpecs {
		assert.NotEmpty(t, p.Platform)
	}
	assert.True(t, ds.TeamByNameFuncInvoked)
}

func TestApplyEnrollSecrets(t *testing.T) {
	_, ds := runServerWithMockedDS(t)

	var appliedSecrets []*fleet.EnrollSecret
	ds.ApplyEnrollSecretsFunc = func(ctx context.Context, teamID *uint, secrets []*fleet.EnrollSecret) error {
		appliedSecrets = secrets
		return nil
	}

	name := writeTmpYml(t, `---
apiVersion: v1
kind: enroll_secret
spec:
  secrets:
    - secret: RzTlxPvugG4o4O5IKS/HqEDJUmI1hwBoffff
    - secret: reallyworks
    - secret: thissecretwontwork!
`)

	assert.Equal(t, "[+] applied enroll secrets\n", runAppForTest(t, []string{"apply", "-f", name}))
	assert.True(t, ds.ApplyEnrollSecretsFuncInvoked)
	assert.Len(t, appliedSecrets, 3)
	for _, s := range appliedSecrets {
		assert.NotEmpty(t, s.Secret)
	}
}

func TestApplyLabels(t *testing.T) {
	_, ds := runServerWithMockedDS(t)

	var appliedLabels []*fleet.LabelSpec
	ds.ApplyLabelSpecsFunc = func(ctx context.Context, specs []*fleet.LabelSpec) error {
		appliedLabels = specs
		return nil
	}

	name := writeTmpYml(t, `---
apiVersion: v1
kind: label
spec:
  name: pending_updates
  query: select 1;
  platforms:
    - darwin
`)

	assert.Equal(t, "[+] applied 1 labels\n", runAppForTest(t, []string{"apply", "-f", name}))
	assert.True(t, ds.ApplyLabelSpecsFuncInvoked)
	require.Len(t, appliedLabels, 1)
	assert.Equal(t, "pending_updates", appliedLabels[0].Name)
	assert.Equal(t, "select 1;", appliedLabels[0].Query)
}

func TestApplyPacks(t *testing.T) {
	_, ds := runServerWithMockedDS(t)

	ds.ListPacksFunc = func(ctx context.Context, opt fleet.PackListOptions) ([]*fleet.Pack, error) {
		return nil, nil
	}
	ds.NewActivityFunc = func(ctx context.Context, user *fleet.User, activityType string, details *map[string]interface{}) error {
		return nil
	}

	var appliedPacks []*fleet.PackSpec
	ds.ApplyPackSpecsFunc = func(ctx context.Context, specs []*fleet.PackSpec) error {
		appliedPacks = specs
		return nil
	}

	name := writeTmpYml(t, `---
apiVersion: v1
kind: pack
spec:
  name: osquery_monitoring
  queries:
    - query: osquery_version
      name: osquery_version_snapshot
      interval: 7200
      snapshot: true
    - query: osquery_version
      name: osquery_version_differential
      interval: 7200
`)

	assert.Equal(t, "[+] applied 1 packs\n", runAppForTest(t, []string{"apply", "-f", name}))
	assert.True(t, ds.ApplyPackSpecsFuncInvoked)
	require.Len(t, appliedPacks, 1)
	assert.Equal(t, "osquery_monitoring", appliedPacks[0].Name)
	require.Len(t, appliedPacks[0].Queries, 2)
}

func TestApplyQueries(t *testing.T) {
	_, ds := runServerWithMockedDS(t)

	var appliedQueries []*fleet.Query
	ds.ApplyQueriesFunc = func(ctx context.Context, authorID uint, queries []*fleet.Query) error {
		appliedQueries = queries
		return nil
	}
	ds.NewActivityFunc = func(ctx context.Context, user *fleet.User, activityType string, details *map[string]interface{}) error {
		return nil
	}

	name := writeTmpYml(t, `---
apiVersion: v1
kind: query
spec:
  description: Retrieves the list of application scheme/protocol-based IPC handlers.
  name: app_schemes
  query: select * from app_schemes;
`)

	assert.Equal(t, "[+] applied 1 queries\n", runAppForTest(t, []string{"apply", "-f", name}))
	assert.True(t, ds.ApplyQueriesFuncInvoked)
	require.Len(t, appliedQueries, 1)
	assert.Equal(t, "app_schemes", appliedQueries[0].Name)
	assert.Equal(t, "select * from app_schemes;", appliedQueries[0].Query)
}
