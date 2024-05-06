package spec

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var topLevelOptions = map[string]string{
	"controls":      "controls:",
	"queries":       "queries:",
	"policies":      "policies:",
	"agent_options": "agent_options:",
	"org_settings": `
org_settings:
  server_settings:
    server_url: https://fleet.example.com
  org_info:
    contact_url: https://example.com/contact
    org_logo_url: ""
    org_logo_url_light_background: ""
    org_name: Test Org
  secrets:
`,
}

var teamLevelOptions = map[string]string{
	"controls":      "controls:",
	"queries":       "queries:",
	"policies":      "policies:",
	"agent_options": "agent_options:",
	"name":          "name: TeamName",
	"team_settings": `
team_settings:
  secrets:
`,
}

func TestValidGitOpsYaml(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		filePath string
		isTeam   bool
	}{
		"global_config_no_paths": {
			filePath: "testdata/global_config_no_paths.yml",
		},
		"global_config_with_paths": {
			filePath: "testdata/global_config.yml",
		},
		"team_config_no_paths": {
			filePath: "testdata/team_config_no_paths.yml",
			isTeam:   true,
		},
		"team_config_with_paths": {
			filePath: "testdata/team_config.yml",
			isTeam:   true,
		},
	}

	for name, test := range tests {
		test := test
		name := name
		t.Run(
			name, func(t *testing.T) {
				t.Parallel()
				dat, err := os.ReadFile(test.filePath)
				require.NoError(t, err)
				gitops, err := GitOpsFromBytes(dat, "./testdata")
				require.NoError(t, err)

				if test.isTeam {
					// Check team settings
					assert.Equal(t, "Team1", *gitops.TeamName)
					assert.Contains(t, gitops.TeamSettings, "webhook_settings")
					assert.Contains(t, gitops.TeamSettings, "host_expiry_settings")
					assert.Contains(t, gitops.TeamSettings, "features")
					assert.Contains(t, gitops.TeamSettings, "secrets")
					secrets, ok := gitops.TeamSettings["secrets"]
					assert.True(t, ok, "secrets not found")
					require.Len(t, secrets.([]*fleet.EnrollSecret), 2)
					assert.Equal(t, "SampleSecret123", secrets.([]*fleet.EnrollSecret)[0].Secret)
					assert.Equal(t, "ABC", secrets.([]*fleet.EnrollSecret)[1].Secret)
				} else {
					// Check org settings
					serverSettings, ok := gitops.OrgSettings["server_settings"]
					assert.True(t, ok, "server_settings not found")
					assert.Equal(t, "https://fleet.example.com", serverSettings.(map[string]interface{})["server_url"])
					assert.Contains(t, gitops.OrgSettings, "org_info")
					assert.Contains(t, gitops.OrgSettings, "smtp_settings")
					assert.Contains(t, gitops.OrgSettings, "sso_settings")
					assert.Contains(t, gitops.OrgSettings, "integrations")
					assert.Contains(t, gitops.OrgSettings, "mdm")
					assert.Contains(t, gitops.OrgSettings, "webhook_settings")
					assert.Contains(t, gitops.OrgSettings, "fleet_desktop")
					assert.Contains(t, gitops.OrgSettings, "host_expiry_settings")
					assert.Contains(t, gitops.OrgSettings, "activity_expiry_settings")
					assert.Contains(t, gitops.OrgSettings, "features")
					assert.Contains(t, gitops.OrgSettings, "vulnerability_settings")
					assert.Contains(t, gitops.OrgSettings, "secrets")
					secrets, ok := gitops.OrgSettings["secrets"]
					assert.True(t, ok, "secrets not found")
					require.Len(t, secrets.([]*fleet.EnrollSecret), 2)
					assert.Equal(t, "SampleSecret123", secrets.([]*fleet.EnrollSecret)[0].Secret)
					assert.Equal(t, "ABC", secrets.([]*fleet.EnrollSecret)[1].Secret)
					activityExpirySettings, ok := gitops.OrgSettings["activity_expiry_settings"].(map[string]interface{})
					require.True(t, ok)
					activityExpiryEnabled, ok := activityExpirySettings["activity_expiry_enabled"].(bool)
					require.True(t, ok)
					require.True(t, activityExpiryEnabled)
					activityExpiryWindow, ok := activityExpirySettings["activity_expiry_window"].(float64)
					require.True(t, ok)
					require.Equal(t, 30, int(activityExpiryWindow))
				}

				// Check controls
				_, ok := gitops.Controls.MacOSSettings.(map[string]interface{})
				assert.True(t, ok, "macos_settings not found")
				_, ok = gitops.Controls.WindowsSettings.(map[string]interface{})
				assert.True(t, ok, "windows_settings not found")
				_, ok = gitops.Controls.EnableDiskEncryption.(bool)
				assert.True(t, ok, "enable_disk_encryption not found")
				_, ok = gitops.Controls.MacOSMigration.(map[string]interface{})
				assert.True(t, ok, "macos_migration not found")
				_, ok = gitops.Controls.MacOSSetup.(map[string]interface{})
				assert.True(t, ok, "macos_setup not found")
				_, ok = gitops.Controls.MacOSUpdates.(map[string]interface{})
				assert.True(t, ok, "macos_updates not found")
				_, ok = gitops.Controls.WindowsEnabledAndConfigured.(bool)
				assert.True(t, ok, "windows_enabled_and_configured not found")
				_, ok = gitops.Controls.WindowsUpdates.(map[string]interface{})
				assert.True(t, ok, "windows_updates not found")

				// Check agent options
				assert.NotNil(t, gitops.AgentOptions)
				assert.Contains(t, string(*gitops.AgentOptions), "distributed_denylist_duration")

				// Check queries
				require.Len(t, gitops.Queries, 3)
				assert.Equal(t, "Scheduled query stats", gitops.Queries[0].Name)
				assert.Equal(t, "orbit_info", gitops.Queries[1].Name)
				assert.Equal(t, "osquery_info", gitops.Queries[2].Name)

				// Check policies
				require.Len(t, gitops.Policies, 5)
				assert.Equal(t, "😊 Failing policy", gitops.Policies[0].Name)
				assert.Equal(t, "Passing policy", gitops.Policies[1].Name)
				assert.Equal(t, "No root logins (macOS, Linux)", gitops.Policies[2].Name)
				assert.Equal(t, "🔥 Failing policy", gitops.Policies[3].Name)
				assert.Equal(t, "😊😊 Failing policy", gitops.Policies[4].Name)
			},
		)
	}
}

func TestDuplicatePolicyNames(t *testing.T) {
	t.Parallel()
	config := getGlobalConfig([]string{"policies"})
	config += `
policies:
  - name: My policy
    platform: linux
    query: SELECT 1 FROM osquery_info WHERE start_time < 0;
  - name: My policy
    platform: windows
    query: SELECT 1;
`
	_, err := GitOpsFromBytes([]byte(config), "")
	assert.ErrorContains(t, err, "duplicate policy names")
}

func TestDuplicateQueryNames(t *testing.T) {
	t.Parallel()
	config := getGlobalConfig([]string{"queries"})
	config += `
queries:
- name: orbit_info
  query: SELECT * from orbit_info;
  interval: 0
  platform: darwin,linux,windows
  min_osquery_version: all
  observer_can_run: false
  automations_enabled: true
  logging: snapshot
- name: orbit_info
  query: SELECT 1;
  interval: 300
  platform: windows
  min_osquery_version: all
  observer_can_run: false
  automations_enabled: true
  logging: snapshot
`
	_, err := GitOpsFromBytes([]byte(config), "")
	assert.ErrorContains(t, err, "duplicate query names")
}

func TestUnicodeQueryNames(t *testing.T) {
	t.Parallel()
	config := getGlobalConfig([]string{"queries"})
	config += `
queries:
- name: 😊 orbit_info
  query: SELECT * from orbit_info;
  interval: 0
  platform: darwin,linux,windows
  min_osquery_version: all
  observer_can_run: false
  automations_enabled: true
  logging: snapshot
`
	_, err := GitOpsFromBytes([]byte(config), "")
	assert.ErrorContains(t, err, "query name must be in ASCII")
}

func TestUnicodeTeamName(t *testing.T) {
	t.Parallel()
	config := getTeamConfig([]string{"name"})
	config += `name: 😊 TeamName`
	_, err := GitOpsFromBytes([]byte(config), "")
	assert.NoError(t, err)
}

func TestMixingGlobalAndTeamConfig(t *testing.T) {
	t.Parallel()

	// Mixing org_settings and team name
	config := getGlobalConfig(nil)
	config += "name: TeamName\n"
	_, err := GitOpsFromBytes([]byte(config), "")
	assert.ErrorContains(t, err, "'org_settings' cannot be used with 'name' or 'team_settings'")

	// Mixing org_settings and team_settings
	config = getGlobalConfig(nil)
	config += "team_settings:\n  secrets: []\n"
	_, err = GitOpsFromBytes([]byte(config), "")
	assert.ErrorContains(t, err, "'org_settings' cannot be used with 'name' or 'team_settings'")

	// Mixing org_settings and team name and team_settings
	config = getGlobalConfig(nil)
	config += "name: TeamName\n"
	config += "team_settings:\n  secrets: []\n"
	_, err = GitOpsFromBytes([]byte(config), "")
	assert.ErrorContains(t, err, "'org_settings' cannot be used with 'name' or 'team_settings'")
}

func TestInvalidGitOpsYaml(t *testing.T) {
	t.Parallel()

	// Bad YAML
	_, err := GitOpsFromBytes([]byte("bad:\nbad"), "")
	assert.ErrorContains(t, err, "failed to unmarshal")

	for _, name := range []string{"global", "team"} {
		t.Run(
			name, func(t *testing.T) {
				isTeam := name == "team"
				getConfig := getGlobalConfig
				if isTeam {
					getConfig = getTeamConfig
				}

				if isTeam {
					// Invalid top level key
					config := getConfig(nil)
					config += "unknown_key:\n"
					_, err = GitOpsFromBytes([]byte(config), "")
					assert.ErrorContains(t, err, "unknown top-level field")

					// Invalid team name
					config = getConfig([]string{"name"})
					config += "name: [2]\n"
					_, err = GitOpsFromBytes([]byte(config), "")
					assert.ErrorContains(t, err, "failed to unmarshal name")

					// Missing team name
					config = getConfig([]string{"name"})
					config += "name:\n"
					_, err = GitOpsFromBytes([]byte(config), "")
					assert.ErrorContains(t, err, "'name' is required")

					// Invalid team_settings
					config = getConfig([]string{"team_settings"})
					config += "team_settings:\n  path: [2]\n"
					_, err = GitOpsFromBytes([]byte(config), "")
					assert.ErrorContains(t, err, "failed to unmarshal team_settings")

					// Invalid team_settings in a separate file
					tmpFile, err := os.CreateTemp(t.TempDir(), "*team_settings.yml")
					require.NoError(t, err)
					_, err = tmpFile.WriteString("[2]")
					require.NoError(t, err)
					config = getConfig([]string{"team_settings"})
					config += fmt.Sprintf("%s:\n  path: %s\n", "team_settings", tmpFile.Name())
					_, err = GitOpsFromBytes([]byte(config), "")
					assert.ErrorContains(t, err, "failed to unmarshal team settings file")

					// Invalid secrets 1
					config = getConfig([]string{"team_settings"})
					config += "team_settings:\n  secrets: bad\n"
					_, err = GitOpsFromBytes([]byte(config), "")
					assert.ErrorContains(t, err, "must be a list of secret items")

					// Invalid secrets 2
					config = getConfig([]string{"team_settings"})
					config += "team_settings:\n  secrets: [2]\n"
					_, err = GitOpsFromBytes([]byte(config), "")
					assert.ErrorContains(t, err, "must have a 'secret' key")

					// Missing secrets
					config = getConfig([]string{"team_settings"})
					config += "team_settings:\n"
					_, err = GitOpsFromBytes([]byte(config), "")
					assert.ErrorContains(t, err, "'team_settings.secrets' is required")
				} else {
					// Invalid org_settings
					config := getConfig([]string{"org_settings"})
					config += "org_settings:\n  path: [2]\n"
					_, err = GitOpsFromBytes([]byte(config), "")
					assert.ErrorContains(t, err, "failed to unmarshal org_settings")

					// Invalid org_settings in a separate file
					tmpFile, err := os.CreateTemp(t.TempDir(), "*org_settings.yml")
					require.NoError(t, err)
					_, err = tmpFile.WriteString("[2]")
					require.NoError(t, err)
					config = getConfig([]string{"org_settings"})
					config += fmt.Sprintf("%s:\n  path: %s\n", "org_settings", tmpFile.Name())
					_, err = GitOpsFromBytes([]byte(config), "")
					assert.ErrorContains(t, err, "failed to unmarshal org settings file")

					// Invalid secrets 1
					config = getConfig([]string{"org_settings"})
					config += "org_settings:\n  secrets: bad\n"
					_, err = GitOpsFromBytes([]byte(config), "")
					assert.ErrorContains(t, err, "must be a list of secret items")

					// Invalid secrets 2
					config = getConfig([]string{"org_settings"})
					config += "org_settings:\n  secrets: [2]\n"
					_, err = GitOpsFromBytes([]byte(config), "")
					assert.ErrorContains(t, err, "must have a 'secret' key")

					// Missing secrets
					config = getConfig([]string{"org_settings"})
					config += "org_settings:\n"
					_, err = GitOpsFromBytes([]byte(config), "")
					assert.ErrorContains(t, err, "'org_settings.secrets' is required")
				}

				// Invalid agent_options
				config := getConfig([]string{"agent_options"})
				config += "agent_options:\n  path: [2]\n"
				_, err = GitOpsFromBytes([]byte(config), "")
				assert.ErrorContains(t, err, "failed to unmarshal agent_options")

				// Invalid agent_options in a separate file
				tmpFile, err := os.CreateTemp(t.TempDir(), "*agent_options.yml")
				require.NoError(t, err)
				_, err = tmpFile.WriteString("[2]")
				require.NoError(t, err)
				config = getConfig([]string{"agent_options"})
				config += fmt.Sprintf("%s:\n  path: %s\n", "agent_options", tmpFile.Name())
				_, err = GitOpsFromBytes([]byte(config), "")
				assert.ErrorContains(t, err, "failed to unmarshal agent options file")

				// Invalid controls
				config = getConfig([]string{"controls"})
				config += "controls:\n  path: [2]\n"
				_, err = GitOpsFromBytes([]byte(config), "")
				assert.ErrorContains(t, err, "failed to unmarshal controls")

				// Invalid controls in a separate file
				tmpFile, err = os.CreateTemp(t.TempDir(), "*controls.yml")
				require.NoError(t, err)
				_, err = tmpFile.WriteString("[2]")
				require.NoError(t, err)
				config = getConfig([]string{"controls"})
				config += fmt.Sprintf("%s:\n  path: %s\n", "controls", tmpFile.Name())
				_, err = GitOpsFromBytes([]byte(config), "")
				assert.ErrorContains(t, err, "failed to unmarshal controls file")

				// Invalid policies
				config = getConfig([]string{"policies"})
				config += "policies:\n  path: [2]\n"
				_, err = GitOpsFromBytes([]byte(config), "")
				assert.ErrorContains(t, err, "failed to unmarshal policies")

				// Invalid policies in a separate file
				tmpFile, err = os.CreateTemp(t.TempDir(), "*policies.yml")
				require.NoError(t, err)
				_, err = tmpFile.WriteString("[2]")
				require.NoError(t, err)
				config = getConfig([]string{"policies"})
				config += fmt.Sprintf("%s:\n  - path: %s\n", "policies", tmpFile.Name())
				_, err = GitOpsFromBytes([]byte(config), "")
				assert.ErrorContains(t, err, "failed to unmarshal policies file")

				// Policy name missing
				config = getConfig([]string{"policies"})
				config += "policies:\n  - query: SELECT 1;\n"
				_, err = GitOpsFromBytes([]byte(config), "")
				assert.ErrorContains(t, err, "name is required")

				// Policy query missing
				config = getConfig([]string{"policies"})
				config += "policies:\n  - name: Test Policy\n"
				_, err = GitOpsFromBytes([]byte(config), "")
				assert.ErrorContains(t, err, "query is required")

				// Invalid queries
				config = getConfig([]string{"queries"})
				config += "queries:\n  path: [2]\n"
				_, err = GitOpsFromBytes([]byte(config), "")
				assert.ErrorContains(t, err, "failed to unmarshal queries")

				// Invalid policies in a separate file
				tmpFile, err = os.CreateTemp(t.TempDir(), "*queries.yml")
				require.NoError(t, err)
				_, err = tmpFile.WriteString("[2]")
				require.NoError(t, err)
				config = getConfig([]string{"queries"})
				config += fmt.Sprintf("%s:\n  - path: %s\n", "queries", tmpFile.Name())
				_, err = GitOpsFromBytes([]byte(config), "")
				assert.ErrorContains(t, err, "failed to unmarshal queries file")

				// Query name missing
				config = getConfig([]string{"queries"})
				config += "queries:\n  - query: SELECT 1;\n"
				_, err = GitOpsFromBytes([]byte(config), "")
				assert.ErrorContains(t, err, "name is required")

				// Query SQL query missing
				config = getConfig([]string{"queries"})
				config += "queries:\n  - name: Test Query\n"
				_, err = GitOpsFromBytes([]byte(config), "")
				assert.ErrorContains(t, err, "query is required")
			},
		)
	}
}

func TestTopLevelGitOpsValidation(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		optsToExclude []string
		shouldPass    bool
		isTeam        bool
	}{
		"all_present_global": {
			optsToExclude: []string{},
			shouldPass:    true,
		},
		"all_present_team": {
			optsToExclude: []string{},
			shouldPass:    true,
			isTeam:        true,
		},
		"missing_all": {
			optsToExclude: []string{"controls", "queries", "policies", "agent_options", "org_settings"},
		},
		"missing_controls": {
			optsToExclude: []string{"controls"},
		},
		"missing_queries": {
			optsToExclude: []string{"queries"},
		},
		"missing_policies": {
			optsToExclude: []string{"policies"},
		},
		"missing_agent_options": {
			optsToExclude: []string{"agent_options"},
		},
		"missing_org_settings": {
			optsToExclude: []string{"org_settings"},
		},
		"missing_name": {
			optsToExclude: []string{"name"},
			isTeam:        true,
		},
		"missing_team_settings": {
			optsToExclude: []string{"team_settings"},
			isTeam:        true,
		},
	}
	for name, test := range tests {
		t.Run(
			name, func(t *testing.T) {
				var config string
				if test.isTeam {
					config = getTeamConfig(test.optsToExclude)
				} else {
					config = getGlobalConfig(test.optsToExclude)
				}
				_, err := GitOpsFromBytes([]byte(config), "")
				if test.shouldPass {
					assert.NoError(t, err)
				} else {
					assert.ErrorContains(t, err, "is required")
				}
			},
		)
	}
}

func TestGitOpsNullArrays(t *testing.T) {
	t.Parallel()

	config := getGlobalConfig([]string{"queries", "policies"})
	config += "queries: null\npolicies: ~\n"
	gitops, err := GitOpsFromBytes([]byte(config), "")
	assert.NoError(t, err)
	assert.Nil(t, gitops.Queries)
	assert.Nil(t, gitops.Policies)
}

func TestGitOpsPaths(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		isArray    bool
		isTeam     bool
		goodConfig string
	}{
		"org_settings": {
			isArray:    false,
			goodConfig: "secrets: []\n",
		},
		"team_settings": {
			isArray:    false,
			isTeam:     true,
			goodConfig: "secrets: []\n",
		},
		"controls": {
			isArray:    false,
			goodConfig: "windows_enabled_and_configured: true\n",
		},
		"queries": {
			isArray:    true,
			goodConfig: "[]",
		},
		"policies": {
			isArray:    true,
			goodConfig: "[]",
		},
		"agent_options": {
			isArray:    false,
			goodConfig: "name: value\n",
		},
	}

	for name, test := range tests {
		test := test
		name := name
		t.Run(
			name, func(t *testing.T) {
				t.Parallel()

				getConfig := getGlobalConfig
				if test.isTeam {
					getConfig = getTeamConfig
				}

				// Test an absolute top level path
				tmpFile, err := os.CreateTemp(t.TempDir(), "*good.yml")
				require.NoError(t, err)
				_, err = tmpFile.WriteString(test.goodConfig)
				require.NoError(t, err)
				config := getConfig([]string{name})
				if test.isArray {
					config += fmt.Sprintf("%s:\n  - path: %s\n", name, tmpFile.Name())
				} else {
					config += fmt.Sprintf("%s:\n  path: %s\n", name, tmpFile.Name())
				}
				_, err = GitOpsFromBytes([]byte(config), "")
				assert.NoError(t, err)

				// Test a relative top level path
				config = getConfig([]string{name})
				dir, file := filepath.Split(tmpFile.Name())
				if test.isArray {
					config += fmt.Sprintf("%s:\n  - path: ./%s\n", name, file)
				} else {
					config += fmt.Sprintf("%s:\n  path: ./%s\n", name, file)
				}
				_, err = GitOpsFromBytes([]byte(config), dir)
				assert.NoError(t, err)

				// Test a bad path
				config = getConfig([]string{name})
				if test.isArray {
					config += fmt.Sprintf("%s:\n  - path: ./%s\n", name, "doesNotExist.yml")
				} else {
					config += fmt.Sprintf("%s:\n  path: ./%s\n", name, "doesNotExist.yml")
				}
				_, err = GitOpsFromBytes([]byte(config), dir)
				assert.ErrorContains(t, err, "no such file or directory")

				// Test a bad file -- cannot be unmarshalled
				tmpFileBad, err := os.CreateTemp(t.TempDir(), "*invalid.yml")
				require.NoError(t, err)
				_, err = tmpFileBad.WriteString("bad:\nbad")
				require.NoError(t, err)
				config = getConfig([]string{name})
				if test.isArray {
					config += fmt.Sprintf("%s:\n  - path: %s\n", name, tmpFileBad.Name())
				} else {
					config += fmt.Sprintf("%s:\n  path: %s\n", name, tmpFileBad.Name())
				}
				_, err = GitOpsFromBytes([]byte(config), "")
				assert.ErrorContains(t, err, "failed to unmarshal")

				// Test a nested path -- bad
				tmpFileBad, err = os.CreateTemp(t.TempDir(), "*bad.yml")
				require.NoError(t, err)
				if test.isArray {
					_, err = tmpFileBad.WriteString(fmt.Sprintf("- path: %s\n", tmpFile.Name()))
				} else {
					_, err = tmpFileBad.WriteString(fmt.Sprintf("path: %s\n", tmpFile.Name()))
				}
				require.NoError(t, err)
				config = getConfig([]string{name})
				dir, file = filepath.Split(tmpFileBad.Name())
				if test.isArray {
					config += fmt.Sprintf("%s:\n  - path: ./%s\n", name, file)
				} else {
					config += fmt.Sprintf("%s:\n  path: ./%s\n", name, file)
				}
				_, err = GitOpsFromBytes([]byte(config), dir)
				assert.ErrorContains(t, err, "nested paths are not supported")
			},
		)
	}
}

func getGlobalConfig(optsToExclude []string) string {
	return getBaseConfig(topLevelOptions, optsToExclude)
}

func getTeamConfig(optsToExclude []string) string {
	return getBaseConfig(teamLevelOptions, optsToExclude)
}

func getBaseConfig(options map[string]string, optsToExclude []string) string {
	var config string
	for key, value := range options {
		if !slices.Contains(optsToExclude, key) {
			config += value + "\n"
		}
	}
	return config
}
