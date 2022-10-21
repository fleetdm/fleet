package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
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
	_, ds := runServerWithMockedDS(t, &service.TestServerOpts{License: license})

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

	ds.NewActivityFunc = func(ctx context.Context, user *fleet.User, activityType string, details *map[string]interface{}) error {
		return nil
	}

	filename := writeTmpYml(t, `
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
        views:
          foo: bar
    name: team1
    secrets:
      - secret: AAA
`)

	newAgentOpts := json.RawMessage(`{"config":{"views":{"foo":"bar"}}}`)
	require.Equal(t, "[+] applied 2 teams\n", runAppForTest(t, []string{"apply", "-f", filename}))
	assert.JSONEq(t, string(agentOpts), string(*teamsByName["team2"].Config.AgentOptions))
	assert.JSONEq(t, string(newAgentOpts), string(*teamsByName["team1"].Config.AgentOptions))
	assert.Equal(t, []*fleet.EnrollSecret{{Secret: "AAA"}}, enrolledSecretsCalled[uint(42)])
	assert.True(t, ds.ApplyEnrollSecretsFuncInvoked)
	ds.ApplyEnrollSecretsFuncInvoked = false

	filename = writeTmpYml(t, `
apiVersion: v1
kind: team
spec:
  team:
    name: team1
`)

	require.Equal(t, "[+] applied 1 teams\n", runAppForTest(t, []string{"apply", "-f", filename}))
	assert.Equal(t, []*fleet.EnrollSecret{{Secret: "AAA"}}, enrolledSecretsCalled[uint(42)])
	assert.False(t, ds.ApplyEnrollSecretsFuncInvoked)

	filename = writeTmpYml(t, `
apiVersion: v1
kind: team
spec:
  team:
    agent_options:
      config:
        views:
          foo: qux
    name: team1
    secrets:
      - secret: BBB
`)

	newAgentOpts = json.RawMessage(`{"config":{"views":{"foo":"qux"}}}`)
	require.Equal(t, "[+] applied 1 teams\n", runAppForTest(t, []string{"apply", "-f", filename}))
	assert.JSONEq(t, string(newAgentOpts), string(*teamsByName["team1"].Config.AgentOptions))
	assert.Equal(t, []*fleet.EnrollSecret{{Secret: "BBB"}}, enrolledSecretsCalled[uint(42)])
	assert.True(t, ds.ApplyEnrollSecretsFuncInvoked)
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
		return &fleet.AppConfig{OrgInfo: fleet.OrgInfo{OrgName: "Fleet"}, ServerSettings: fleet.ServerSettings{ServerURL: "https://example.org"}}, nil
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
  features:
    enable_host_users: false
    enable_software_inventory: false
`)

	assert.Equal(t, "[+] applied fleet config\n", runAppForTest(t, []string{"apply", "-f", name}))
	require.NotNil(t, savedAppConfig)
	assert.False(t, savedAppConfig.Features.EnableHostUsers)
	assert.False(t, savedAppConfig.Features.EnableSoftwareInventory)

	name = writeTmpYml(t, `---
apiVersion: v1
kind: config
spec:
  features:
    enable_host_users: true
    enable_software_inventory: true
`)

	assert.Equal(t, "[+] applied fleet config\n", runAppForTest(t, []string{"apply", "-f", name}))
	require.NotNil(t, savedAppConfig)
	assert.True(t, savedAppConfig.Features.EnableHostUsers)
	assert.True(t, savedAppConfig.Features.EnableSoftwareInventory)
}

func TestApplyAppConfigDryRunIssue(t *testing.T) {
	// reproduces the bug fixed by https://github.com/fleetdm/fleet/pull/8194
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

	ds.NewActivityFunc = func(ctx context.Context, user *fleet.User, activityType string, details *map[string]interface{}) error {
		return nil
	}

	var currentAppConfig = &fleet.AppConfig{
		OrgInfo: fleet.OrgInfo{OrgName: "Fleet"}, ServerSettings: fleet.ServerSettings{ServerURL: "https://example.org"},
	}
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return currentAppConfig, nil
	}

	ds.SaveAppConfigFunc = func(ctx context.Context, config *fleet.AppConfig) error {
		currentAppConfig = config
		return nil
	}

	// first, set the default app config's agent options as set after fleetctl setup
	name := writeTmpYml(t, `---
apiVersion: v1
kind: config
spec:
  agent_options:
    config:
      decorators:
        load:
        - SELECT uuid AS host_uuid FROM system_info;
        - SELECT hostname AS hostname FROM system_info;
      options:
        disable_distributed: false
        distributed_interval: 10
        distributed_plugin: tls
        distributed_tls_max_attempts: 3
        logger_tls_endpoint: /api/osquery/log
        logger_tls_period: 10
        pack_delimiter: /
    overrides: {}
`)

	assert.Equal(t, "[+] applied fleet config\n", runAppForTest(t, []string{"apply", "-f", name}))

	// then, dry-run a valid app config's agent options, which made the original
	// app config's agent options invalid JSON (when it shouldn't have modified
	// it at all - the issue was in the cached_mysql datastore, it did not clone
	// the app config properly).
	name = writeTmpYml(t, `---
apiVersion: v1
kind: config
spec:
  agent_options:
    overrides:
      platforms:
        darwin:
          auto_table_construction:
            tcc_system_entries:
              query: "SELECT service, client, allowed, prompt_count, last_modified FROM access"
              path: "/Library/Application Support/com.apple.TCC/TCC.db"
              columns:
                - "service"
                - "client"
                - "allowed"
                - "prompt_count"
                - "last_modified"
`)

	assert.Equal(t, "[+] would've applied fleet config\n", runAppForTest(t, []string{"apply", "--dry-run", "-f", name}))

	// the saved app config was left unchanged, still equal to the original agent
	// options
	got := runAppForTest(t, []string{"get", "config"})
	assert.Contains(t, got, `agent_options:
    config:
      decorators:
        load:
        - SELECT uuid AS host_uuid FROM system_info;
        - SELECT hostname AS hostname FROM system_info;
      options:
        disable_distributed: false
        distributed_interval: 10
        distributed_plugin: tls
        distributed_tls_max_attempts: 3
        logger_tls_endpoint: /api/osquery/log
        logger_tls_period: 10
        pack_delimiter: /
    overrides: {}`)
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
  features:
    enabled_software_inventory: false # typo, correct config is enable_software_inventory
`)

	runAppCheckErr(t, []string{"apply", "-f", name},
		"applying fleet config: PATCH /api/latest/fleet/config received status 400 Bad Request: unsupported key provided: \"enabled_software_inventory\"",
	)
	require.Nil(t, savedAppConfig)
}

func TestApplyAppConfigDeprecatedFields(t *testing.T) {
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
		return &fleet.AppConfig{OrgInfo: fleet.OrgInfo{OrgName: "Fleet"}, ServerSettings: fleet.ServerSettings{ServerURL: "https://example.org"}}, nil
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
	assert.False(t, savedAppConfig.Features.EnableHostUsers)
	assert.False(t, savedAppConfig.Features.EnableSoftwareInventory)

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
	assert.True(t, savedAppConfig.Features.EnableHostUsers)
	assert.True(t, savedAppConfig.Features.EnableSoftwareInventory)
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
		return nil, errors.New("unexpected team name!")
	}
	ds.NewActivityFunc = func(ctx context.Context, user *fleet.User, activityType string, details *map[string]interface{}) error {
		return nil
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

	interval := writeTmpYml(t, `---
apiVersion: v1
kind: pack
spec:
  name: test_bad_interval
  queries:
    - query: good_interval
      name: good_interval
      interval: 7200
    - query: bad_interval
      name: bad_interval
      interval: 604801
`)

	expectedErrMsg := "applying packs: POST /api/latest/fleet/spec/packs received status 400 Bad request: pack payload verification: pack scheduled query interval must be an integer greater than 1 and less than 604800"

	_, err := runAppNoChecks([]string{"apply", "-f", interval})
	assert.Error(t, err)
	require.Equal(t, expectedErrMsg, err.Error())
}

func TestApplyQueries(t *testing.T) {
	_, ds := runServerWithMockedDS(t)

	var appliedQueries []*fleet.Query
	ds.QueryByNameFunc = func(ctx context.Context, name string, opts ...fleet.OptionalArg) (*fleet.Query, error) {
		return nil, sql.ErrNoRows
	}
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

func TestCanApplyIntervalsInNanoseconds(t *testing.T) {
	_, ds := runServerWithMockedDS(t)

	// Stubs
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
		return &fleet.AppConfig{OrgInfo: fleet.OrgInfo{OrgName: "Fleet"}, ServerSettings: fleet.ServerSettings{ServerURL: "https://example.org"}}, nil
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
  webhook_settings:
    interval: 30000000000
`)

	assert.Equal(t, "[+] applied fleet config\n", runAppForTest(t, []string{"apply", "-f", name}))
	require.Equal(t, savedAppConfig.WebhookSettings.Interval.Duration, 30*time.Second)
}

func TestCanApplyIntervalsUsingDurations(t *testing.T) {
	_, ds := runServerWithMockedDS(t)

	// Stubs
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
		return &fleet.AppConfig{OrgInfo: fleet.OrgInfo{OrgName: "Fleet"}, ServerSettings: fleet.ServerSettings{ServerURL: "https://example.org"}}, nil
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
  webhook_settings:
    interval: 30s
`)

	assert.Equal(t, "[+] applied fleet config\n", runAppForTest(t, []string{"apply", "-f", name}))
	require.Equal(t, savedAppConfig.WebhookSettings.Interval.Duration, 30*time.Second)
}

func TestApplySpecs(t *testing.T) {
	setupDS := func(ds *mock.Store) {
		// labels
		ds.ApplyLabelSpecsFunc = func(ctx context.Context, specs []*fleet.LabelSpec) error {
			return nil
		}

		// teams - team ID 1 already exists
		teamsByName := map[string]*fleet.Team{
			"team1": {
				ID:          1,
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

		i := 1 // new teams will start at 2
		ds.NewTeamFunc = func(ctx context.Context, team *fleet.Team) (*fleet.Team, error) {
			i++
			team.ID = uint(i)
			teamsByName[team.Name] = team
			return team, nil
		}

		agentOpts := json.RawMessage(`{"config":{}}`)
		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			return &fleet.AppConfig{AgentOptions: &agentOpts}, nil
		}

		ds.SaveTeamFunc = func(ctx context.Context, team *fleet.Team) (*fleet.Team, error) {
			teamsByName[team.Name] = team
			return team, nil
		}

		ds.ApplyEnrollSecretsFunc = func(ctx context.Context, teamID *uint, secrets []*fleet.EnrollSecret) error {
			return nil
		}

		// activities
		ds.NewActivityFunc = func(ctx context.Context, user *fleet.User, activityType string, details *map[string]interface{}) error {
			return nil
		}

		// app config
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
			return &fleet.AppConfig{OrgInfo: fleet.OrgInfo{OrgName: "Fleet"}, ServerSettings: fleet.ServerSettings{ServerURL: "https://example.org"}}, nil
		}

		ds.SaveAppConfigFunc = func(ctx context.Context, config *fleet.AppConfig) error {
			return nil
		}
	}

	cases := []struct {
		desc       string
		flags      []string
		spec       string
		wantOutput string
		wantErr    string
	}{
		{
			desc: "empty team spec",
			spec: `
apiVersion: v1
kind: team
spec:
`,
			wantOutput: "[+] applied 1 teams",
		},
		{
			desc: "empty team name",
			spec: `
apiVersion: v1
kind: team
spec:
  team:
    name: ""
`,
			wantErr: `422 Validation Failed: name may not be empty`,
		},
		{
			desc: "invalid agent options for existing team",
			spec: `
apiVersion: v1
kind: team
spec:
  team:
    name: team1
    agent_options:
      config:
        blah: nope
`,
			wantErr: `400 Bad Request: unsupported key provided: "blah"`,
		},
		{
			desc: "invalid top-level key for team",
			spec: `
apiVersion: v1
kind: team
spec:
  team:
    name: team1
    blah: nope
`,
			wantOutput: `[+] applied 1 teams`, // TODO(mna): currently, only agent options is validated, other unknown keys are ignored
		},
		{
			desc: "invalid agent options for new team",
			spec: `
apiVersion: v1
kind: team
spec:
  team:
    name: teamNEW
    agent_options:
      config:
        blah: nope
`,
			wantErr: `400 Bad Request: unsupported key provided: "blah"`,
		},
		{
			desc: "invalid agent options dry-run",
			spec: `
apiVersion: v1
kind: team
spec:
  team:
    name: teamNEW
    agent_options:
      config:
        blah: nope
`,
			flags:   []string{"--dry-run"},
			wantErr: `400 Bad Request: unsupported key provided: "blah"`,
		},
		{
			desc: "invalid agent options force",
			spec: `
apiVersion: v1
kind: team
spec:
  team:
    name: teamNEW
    agent_options:
      config:
        blah: nope
`,
			flags:      []string{"--force"},
			wantOutput: `[+] applied 1 teams`,
		},
		{
			desc: "invalid agent options field type",
			spec: `
apiVersion: v1
kind: team
spec:
  team:
    name: teamNEW
    agent_options:
      config:
        options:
          aws_debug: 123
`,
			flags:   []string{"--dry-run"},
			wantErr: `400 Bad Request: invalid value type at 'options.aws_debug': expected bool but got number`,
		},
		{
			desc: "invalid team agent options command-line flag",
			spec: `
apiVersion: v1
kind: team
spec:
  team:
    name: teamNEW
    agent_options:
      command_line_flags:
        no_such_flag: 123
`,
			wantErr: `400 Bad Request: unsupported key provided: "no_such_flag"`,
		},
		{
			desc: "valid team agent options command-line flag",
			spec: `
apiVersion: v1
kind: team
spec:
  team:
    name: teamNEW
    agent_options:
      command_line_flags:
        enable_tables: "abc"
`,
			wantOutput: `[+] applied 1 teams`,
		},
		{
			desc: "invalid agent options field type in overrides",
			spec: `
apiVersion: v1
kind: team
spec:
  team:
    name: teamNEW
    agent_options:
      config:
        options:
          aws_debug: true
      overrides:
        platforms:
          darwin:
            options:
              aws_debug: 123
`,
			wantErr: `400 Bad Request: invalid value type at 'options.aws_debug': expected bool but got number`,
		},
		{
			desc: "empty config",
			spec: `
apiVersion: v1
kind: config
spec:
`,
			wantOutput: ``, // no output for empty config
		},
		{
			desc: "config with blank required org name",
			spec: `
apiVersion: v1
kind: config
spec:
  org_info:
    org_name: ""
`,
			wantErr: `422 Validation Failed: organization name must be present`,
		},
		{
			desc: "config with blank required server url",
			spec: `
apiVersion: v1
kind: config
spec:
  server_settings:
    server_url: ""
`,
			wantErr: `422 Validation Failed: Fleet server URL must be present`,
		},
		{
			desc: "config with unknown key",
			spec: `
apiVersion: v1
kind: config
spec:
  server_settings:
    foo: bar
`,
			wantErr: `400 Bad Request: unsupported key provided: "foo"`,
		},
		{
			desc: "config with invalid key type",
			spec: `
apiVersion: v1
kind: config
spec:
  server_settings:
    server_url: 123
`,
			wantErr: `400 Bad request: json: cannot unmarshal number into Go struct field ServerSettings.server_settings.server_url of type string`,
		},
		{
			desc: "config with invalid agent options in dry-run",
			spec: `
apiVersion: v1
kind: config
spec:
  agent_options:
    foo: bar
`,
			flags:   []string{"--dry-run"},
			wantErr: `400 Bad Request: unsupported key provided: "foo"`,
		},
		{
			desc: "config with invalid agent options data type in dry-run",
			spec: `
apiVersion: v1
kind: config
spec:
  agent_options:
    config:
      options:
        aws_debug: 123
`,
			flags:   []string{"--dry-run"},
			wantErr: `400 Bad Request: invalid value type at 'options.aws_debug': expected bool but got number`,
		},
		{
			desc: "config with invalid agent options data type with force",
			spec: `
apiVersion: v1
kind: config
spec:
  agent_options:
    config:
      options:
        aws_debug: 123
`,
			flags:      []string{"--force"},
			wantOutput: `[+] applied fleet config`,
		},
		{
			desc: "config with invalid agent options command-line flags",
			spec: `
apiVersion: v1
kind: config
spec:
  agent_options:
    command_line_flags:
      enable_tables: "foo"
      no_such_flag: false
`,
			wantErr: `400 Bad Request: unsupported key provided: "no_such_flag"`,
		},
		{
			desc: "config with invalid value for agent options command-line flags",
			spec: `
apiVersion: v1
kind: config
spec:
  agent_options:
    command_line_flags:
      enable_tables: 123
`,
			wantErr: `400 Bad Request: invalid value type at 'enable_tables': expected string but got number`,
		},
		{
			desc: "config with valid agent options command-line flags",
			spec: `
apiVersion: v1
kind: config
spec:
  agent_options:
    command_line_flags:
      enable_tables: "abc"
`,
			wantOutput: `[+] applied fleet config`,
		},
		{
			desc: "dry-run set with unsupported spec",
			spec: `
apiVersion: v1
kind: label
spec:
  name: label1
  query: SELECT 1
`,
			flags:      []string{"--dry-run"},
			wantOutput: `[!] ignoring labels, dry run mode only supported for 'config' and 'team' specs`,
		},
		{
			desc: "dry-run set with various specs, appconfig warning for legacy",
			spec: `
apiVersion: v1
kind: team
spec:
  team:
    name: teamNEW
---
apiVersion: v1
kind: label
spec:
  name: label1
  query: SELECT 1
---
apiVersion: v1
kind: config
spec:
  host_settings:
    enable_software_inventory: true
`,
			flags:      []string{"--dry-run"},
			wantErr:    `400 Bad request: warning: deprecated settings were used in the configuration: [host_settings]`,
			wantOutput: `[!] ignoring labels, dry run mode only supported for 'config' and 'team' spec`,
		},
		{
			desc: "dry-run set with various specs, no errors",
			spec: `
apiVersion: v1
kind: team
spec:
  team:
    name: teamNEW
---
apiVersion: v1
kind: label
spec:
  name: label1
  query: SELECT 1
---
apiVersion: v1
kind: config
spec:
  features:
    enable_software_inventory: true
`,
			flags: []string{"--dry-run"},
			wantOutput: `[!] ignoring labels, dry run mode only supported for 'config' and 'team' specs
[+] would've applied fleet config
[+] would've applied 1 teams`,
		},
		{
			desc: "missing required sso entity_id",
			spec: `
apiVersion: v1
kind: config
spec:
  sso_settings:
    enable_sso: true
    entity_id: ""
    issuer_uri: "http://localhost:8080/simplesaml/saml2/idp/SSOService.php"
    idp_name: "SimpleSAML"
    metadata_url: "http://localhost:9080/simplesaml/saml2/idp/metadata.php"
`,
			wantErr: `422 Validation Failed: required`,
		},
		{
			desc: "missing required sso idp_name",
			spec: `
apiVersion: v1
kind: config
spec:
  sso_settings:
    enable_sso: true
    entity_id: "https://localhost:8080"
    issuer_uri: "http://localhost:8080/simplesaml/saml2/idp/SSOService.php"
    idp_name: ""
    metadata_url: "http://localhost:9080/simplesaml/saml2/idp/metadata.php"
`,
			wantErr: `422 Validation Failed: required`,
		},
		{
			desc: "missing required failing policies destination_url",
			spec: `
apiVersion: v1
kind: config
spec:
  webhook_settings:
    failing_policies_webhook:
      enable_failing_policies_webhook: true
      destination_url: ""
      policy_ids:
        - 1
      host_batch_size: 1000
    interval: 1h
`,
			wantErr: `422 Validation Failed: destination_url is required to enable the failing policies webhook`,
		},
		{
			desc: "missing required vulnerabilities destination_url",
			spec: `
apiVersion: v1
kind: config
spec:
  webhook_settings:
    vulnerabilities_webhook:
      enable_vulnerabilities_webhook: true
      destination_url: ""
      host_batch_size: 1000
    interval: 1h
`,
			wantErr: `422 Validation Failed: destination_url is required to enable the vulnerabilities webhook`,
		},
		{
			desc: "missing required host status destination_url",
			spec: `
apiVersion: v1
kind: config
spec:
  webhook_settings:
    host_status_webhook:
      enable_host_status_webhook: true
      destination_url: ""
      days_count: 10
      host_percentage: 10
    interval: 1h
`,
			wantErr: `422 Validation Failed: destination_url is required to enable the host status webhook`,
		},
		{
			desc: "missing required host status days_count",
			spec: `
apiVersion: v1
kind: config
spec:
  webhook_settings:
    host_status_webhook:
      enable_host_status_webhook: true
      destination_url: "http://some/url"
      days_count: 0
      host_percentage: 10
    interval: 1h
`,
			wantErr: `422 Validation Failed: days_count must be > 0 to enable the host status webhook`,
		},
		{
			desc: "missing required host status host_percentage",
			spec: `
apiVersion: v1
kind: config
spec:
  webhook_settings:
    host_status_webhook:
      enable_host_status_webhook: true
      destination_url: "http://some/url"
      days_count: 10
      host_percentage: -1
    interval: 1h
`,
			wantErr: `422 Validation Failed: host_percentage must be > 0 to enable the host status webhook`,
		},
	}
	// NOTE: Integrations required fields are not tested (Jira/Zendesk) because
	// they require a complex setup to mock the client that would communicate
	// with the external API. However, we make a test API call when enabling an
	// integration, ensuring that any missing configuration field results in an
	// error. Same for smtp_settings (a test email is sent when enabling).

	license := &fleet.LicenseInfo{Tier: fleet.TierPremium, Expiration: time.Now().Add(24 * time.Hour)}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			_, ds := runServerWithMockedDS(t, &service.TestServerOpts{License: license})
			setupDS(ds)
			filename := writeTmpYml(t, c.spec)

			var got string
			if c.wantErr == "" {
				got = runAppForTest(t, append([]string{"apply", "-f", filename}, c.flags...))
			} else {
				buf, err := runAppNoChecks(append([]string{"apply", "-f", filename}, c.flags...))
				require.ErrorContains(t, err, c.wantErr)
				got = buf.String()
			}
			if c.wantOutput == "" {
				require.Empty(t, got)
			} else {
				require.Contains(t, got, c.wantOutput)
			}
		})
	}
}
