package spec

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/file"
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

func createTempFile(t *testing.T, pattern, contents string) (filePath string, baseDir string) {
	tmpFile, err := os.CreateTemp(t.TempDir(), pattern)
	require.NoError(t, err)
	_, err = tmpFile.WriteString(contents)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())
	return tmpFile.Name(), filepath.Dir(tmpFile.Name())
}

func createNamedFileOnTempDir(t *testing.T, name string, contents string) (filePath string, baseDir string) {
	tmpFilePath := filepath.Join(t.TempDir(), name)
	tmpFile, err := os.Create(tmpFilePath)
	require.NoError(t, err)
	_, err = tmpFile.WriteString(contents)
	require.NoError(t, err)
	require.NoError(t, tmpFile.Close())
	return tmpFile.Name(), filepath.Dir(tmpFile.Name())
}

func gitOpsFromString(t *testing.T, s string) (*GitOps, error) {
	path, basePath := createTempFile(t, "", s)
	return GitOpsFromFile(path, basePath, nil, nopLogf)
}

func nopLogf(_ string, _ ...interface{}) {
}

func TestValidGitOpsYaml(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		environment map[string]string
		filePath    string
		isTeam      bool
	}{
		"global_config_no_paths": {
			environment: map[string]string{
				"FLEET_SECRET_FLEET_SECRET_": "fleet_secret",
				"FLEET_SECRET_NAME":          "secret_name",
				"FLEET_SECRET_length":        "10",
				"FLEET_SECRET_BANANA":        "bread",
			},
			filePath: "testdata/global_config_no_paths.yml",
		},
		"global_config_with_paths": {
			environment: map[string]string{
				"LINUX_OS":                      "linux",
				"DISTRIBUTED_DENYLIST_DURATION": "0",
				"ORG_NAME":                      "Fleet Device Management",
				"FLEET_SECRET_FLEET_SECRET_":    "fleet_secret",
				"FLEET_SECRET_NAME":             "secret_name",
				"FLEET_SECRET_length":           "10",
				"FLEET_SECRET_BANANA":           "bread",
			},
			filePath: "testdata/global_config.yml",
		},
		"team_config_no_paths": {
			environment: map[string]string{
				"FLEET_SECRET_FLEET_SECRET_": "fleet_secret",
				"FLEET_SECRET_NAME":          "secret_name",
				"FLEET_SECRET_length":        "10",
				"FLEET_SECRET_BANANA":        "bread",
			},
			filePath: "testdata/team_config_no_paths.yml",
			isTeam:   true,
		},
		"team_config_with_paths": {
			environment: map[string]string{
				"POLICY":                          "policy",
				"LINUX_OS":                        "linux",
				"DISTRIBUTED_DENYLIST_DURATION":   "0",
				"ENABLE_FAILING_POLICIES_WEBHOOK": "true",
				"FLEET_SECRET_FLEET_SECRET_":      "fleet_secret",
				"FLEET_SECRET_NAME":               "secret_name",
				"FLEET_SECRET_length":             "10",
				"FLEET_SECRET_BANANA":             "bread",
			},
			filePath: "testdata/team_config.yml",
			isTeam:   true,
		},
	}

	for name, test := range tests {
		test := test
		name := name
		t.Run(
			name, func(t *testing.T) {
				if len(test.environment) > 0 {
					for k, v := range test.environment {
						os.Setenv(k, v)
					}
					t.Cleanup(func() {
						for k := range test.environment {
							os.Unsetenv(k)
						}
					})
				}

				var appConfig *fleet.EnrichedAppConfig
				if test.isTeam {
					appConfig = &fleet.EnrichedAppConfig{}
					appConfig.License = &fleet.LicenseInfo{
						Tier: fleet.TierPremium,
					}
				}

				gitops, err := GitOpsFromFile(test.filePath, "./testdata", appConfig, nopLogf)
				require.NoError(t, err)

				if test.isTeam {
					// Check team settings
					assert.Equal(t, "Team1", *gitops.TeamName)
					assert.Contains(t, gitops.TeamSettings, "webhook_settings")
					webhookSettings, ok := gitops.TeamSettings["webhook_settings"].(map[string]interface{})
					assert.True(t, ok, "webhook_settings not found")
					assert.Contains(t, webhookSettings, "failing_policies_webhook")
					failingPoliciesWebhook, ok := webhookSettings["failing_policies_webhook"].(map[string]interface{})
					assert.True(t, ok, "webhook_settings not found")
					assert.Contains(t, failingPoliciesWebhook, "enable_failing_policies_webhook")
					enableFailingPoliciesWebhook, ok := failingPoliciesWebhook["enable_failing_policies_webhook"].(bool)
					assert.True(t, ok)
					assert.True(t, enableFailingPoliciesWebhook)
					assert.Contains(t, gitops.TeamSettings, "host_expiry_settings")
					assert.Contains(t, gitops.TeamSettings, "features")
					assert.Contains(t, gitops.TeamSettings, "secrets")
					secrets, ok := gitops.TeamSettings["secrets"]
					assert.True(t, ok, "secrets not found")
					require.Len(t, secrets.([]*fleet.EnrollSecret), 2)
					assert.Equal(t, "SampleSecret123", secrets.([]*fleet.EnrollSecret)[0].Secret)
					assert.Equal(t, "ABC", secrets.([]*fleet.EnrollSecret)[1].Secret)
					require.Len(t, gitops.Software.Packages, 2)
					for _, pkg := range gitops.Software.Packages {
						if strings.Contains(pkg.URL, "MicrosoftTeams") {
							assert.Equal(t, "testdata/lib/uninstall.sh", pkg.UninstallScript.Path)
						} else {
							assert.Empty(t, pkg.UninstallScript.Path)
						}
					}
				} else {
					// Check org settings
					serverSettings, ok := gitops.OrgSettings["server_settings"]
					assert.True(t, ok, "server_settings not found")
					assert.Equal(t, "https://fleet.example.com", serverSettings.(map[string]interface{})["server_url"])
					assert.EqualValues(t, 2000, serverSettings.(map[string]interface{})["query_report_cap"])
					assert.Contains(t, gitops.OrgSettings, "org_info")
					orgInfo, ok := gitops.OrgSettings["org_info"].(map[string]interface{})
					assert.True(t, ok)
					assert.Equal(t, "Fleet Device Management", orgInfo["org_name"])
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
				_, ok = gitops.Controls.IOSUpdates.(map[string]interface{})
				assert.True(t, ok, "ios_updates not found")
				_, ok = gitops.Controls.IPadOSUpdates.(map[string]interface{})
				assert.True(t, ok, "ipados_updates not found")
				_, ok = gitops.Controls.WindowsEnabledAndConfigured.(bool)
				assert.True(t, ok, "windows_enabled_and_configured not found")
				_, ok = gitops.Controls.WindowsMigrationEnabled.(bool)
				assert.True(t, ok, "windows_migration_enabled not found")
				_, ok = gitops.Controls.WindowsUpdates.(map[string]interface{})
				assert.True(t, ok, "windows_updates not found")
				require.Len(t, gitops.FleetSecrets, 4)
				assert.Equal(t, "fleet_secret", gitops.FleetSecrets["FLEET_SECRET_FLEET_SECRET_"])
				assert.Equal(t, "secret_name", gitops.FleetSecrets["FLEET_SECRET_NAME"])
				assert.Equal(t, "10", gitops.FleetSecrets["FLEET_SECRET_length"])
				assert.Equal(t, "bread", gitops.FleetSecrets["FLEET_SECRET_BANANA"])

				// Check agent options
				assert.NotNil(t, gitops.AgentOptions)
				assert.Contains(t, string(*gitops.AgentOptions), "\"distributed_denylist_duration\":0")

				// Check queries
				require.Len(t, gitops.Queries, 3)
				assert.Equal(t, "Scheduled query stats", gitops.Queries[0].Name)
				assert.Equal(t, "orbit_info", gitops.Queries[1].Name)
				assert.Equal(t, "darwin,linux,windows", gitops.Queries[1].Platform)
				assert.Equal(t, "osquery_info", gitops.Queries[2].Name)

				// Check software
				if test.isTeam {
					require.Len(t, gitops.Software.Packages, 2)
					require.Equal(t, "https://statics.teams.cdn.office.net/production-osx/enterprise/webview2/lkg/MicrosoftTeams.pkg", gitops.Software.Packages[0].URL)
					require.False(t, gitops.Software.Packages[0].SelfService)
					require.Equal(t, "https://ftp.mozilla.org/pub/firefox/releases/129.0.2/mac/en-US/Firefox%20129.0.2.pkg", gitops.Software.Packages[1].URL)
					require.True(t, gitops.Software.Packages[1].SelfService)

					require.Len(t, gitops.Software.AppStoreApps, 1)
					require.Equal(t, gitops.Software.AppStoreApps[0].AppStoreID, "123456")
					require.False(t, gitops.Software.AppStoreApps[0].SelfService)
				}

				// Check policies
				expectedPoliciesCount := 5
				if test.isTeam {
					expectedPoliciesCount = 9
				}
				require.Len(t, gitops.Policies, expectedPoliciesCount)
				assert.Equal(t, "😊 Failing policy", gitops.Policies[0].Name)
				assert.Equal(t, "Passing policy", gitops.Policies[1].Name)
				assert.Equal(t, "No root logins (macOS, Linux)", gitops.Policies[2].Name)
				assert.Equal(t, "🔥 Failing policy", gitops.Policies[3].Name)
				assert.Equal(t, "linux", gitops.Policies[3].Platform)
				assert.Equal(t, "😊😊 Failing policy", gitops.Policies[4].Name)
				if test.isTeam {
					assert.Equal(t, "Microsoft Teams on macOS installed and up to date", gitops.Policies[5].Name)
					assert.NotNil(t, gitops.Policies[5].InstallSoftware)
					assert.Equal(t, "./microsoft-teams.pkg.software.yml", gitops.Policies[5].InstallSoftware.PackagePath)

					assert.Equal(t, "Slack on macOS is installed", gitops.Policies[6].Name)
					assert.NotNil(t, gitops.Policies[6].InstallSoftware)
					assert.Equal(t, "123456", gitops.Policies[6].InstallSoftware.AppStoreID)

					assert.Equal(t, "Script run policy", gitops.Policies[7].Name)
					assert.NotNil(t, gitops.Policies[7].RunScript)
					assert.Equal(t, "./lib/collect-fleetd-logs.sh", gitops.Policies[7].RunScript.Path)

					assert.Equal(t, "🔥 Failing policy with script", gitops.Policies[8].Name)
					assert.NotNil(t, gitops.Policies[8].RunScript)
					// . or .. depending on whether with paths or without
					assert.Contains(t, gitops.Policies[8].RunScript.Path, "./lib/collect-fleetd-logs.sh")
				}
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
	_, err := gitOpsFromString(t, config)
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
	_, err := gitOpsFromString(t, config)
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
	_, err := gitOpsFromString(t, config)
	assert.ErrorContains(t, err, "query name must be in ASCII")
}

func TestUnicodeTeamName(t *testing.T) {
	t.Parallel()
	config := getTeamConfig([]string{"name"})
	config += `name: 😊 TeamName`
	_, err := gitOpsFromString(t, config)
	assert.NoError(t, err)
}

func TestVarExpansion(t *testing.T) {
	os.Setenv("MACOS_OS", "darwin")
	os.Setenv("LINUX_OS", "linux")
	os.Setenv("EMPTY_VAR", "")
	t.Cleanup(func() {
		os.Unsetenv("MACOS_OS")
		os.Unsetenv("LINUX_OS")
		os.Unsetenv("EMPTY_VAR")
	})
	config := getGlobalConfig([]string{"queries"})
	config += `
queries:
- name: orbit_info \$NOT_EXPANDED \\\$ALSO_NOT_EXPANDED
  query: "SELECT * from orbit_info; -- double quotes are escaped by YAML after Fleet's escaping of backslashes \\\\\$NOT_EXPANDED"
  interval: 0
  platform: $MACOS_OS,${LINUX_OS},windows$EMPTY_VAR
  min_osquery_version: all
  observer_can_run: false
  automations_enabled: true
  logging: snapshot
  description: 'single quotes are not escaped by YAML \\\$NOT_EXPANDED'
`
	gitOps, err := gitOpsFromString(t, config)
	require.NoError(t, err)
	require.Len(t, gitOps.Queries, 1)
	require.Equal(t, "darwin,linux,windows", gitOps.Queries[0].Platform)
	require.Equal(t, `orbit_info $NOT_EXPANDED \$ALSO_NOT_EXPANDED`, gitOps.Queries[0].Name)
	require.Equal(t, `single quotes are not escaped by YAML \$NOT_EXPANDED`, gitOps.Queries[0].Description)
	require.Equal(t, `SELECT * from orbit_info; -- double quotes are escaped by YAML after Fleet's escaping of backslashes \$NOT_EXPANDED`, gitOps.Queries[0].Query)

	config = getGlobalConfig([]string{"queries"})
	config += `
queries:
- name: orbit_info $NOT_DEFINED
  query: SELECT * from orbit_info;
  interval: 0
  platform: darwin,linux,windows
  min_osquery_version: all
  observer_can_run: false
  automations_enabled: true
  logging: snapshot
`
	_, err = gitOpsFromString(t, config)
	require.Error(t, err)
	require.Contains(t, err.Error(), "environment variable \"NOT_DEFINED\" not set")
}

func TestMixingGlobalAndTeamConfig(t *testing.T) {
	t.Parallel()

	// Mixing org_settings and team name
	config := getGlobalConfig(nil)
	config += "name: TeamName\n"
	_, err := gitOpsFromString(t, config)
	assert.ErrorContains(t, err, "'org_settings' cannot be used with 'name', 'team_settings'")

	// Mixing org_settings and team_settings
	config = getGlobalConfig(nil)
	config += "team_settings:\n  secrets: []\n"
	_, err = gitOpsFromString(t, config)
	assert.ErrorContains(t, err, "'org_settings' cannot be used with 'name', 'team_settings'")

	// Mixing org_settings and team name and team_settings
	config = getGlobalConfig(nil)
	config += "name: TeamName\n"
	config += "team_settings:\n  secrets: []\n"
	_, err = gitOpsFromString(t, config)
	assert.ErrorContains(t, err, "'org_settings' cannot be used with 'name', 'team_settings'")
}

func TestInvalidGitOpsYaml(t *testing.T) {
	t.Parallel()

	// Bad YAML
	_, err := gitOpsFromString(t, "bad:\nbad")
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
					_, err = gitOpsFromString(t, config)
					assert.ErrorContains(t, err, "unknown top-level field")

					// Invalid team name
					config = getConfig([]string{"name"})
					config += "name: [2]\n"
					_, err = gitOpsFromString(t, config)
					assert.ErrorContains(t, err, "failed to unmarshal name")

					// Missing team name
					config = getConfig([]string{"name"})
					config += "name:\n"
					_, err = gitOpsFromString(t, config)
					assert.ErrorContains(t, err, "'name' is required")

					// Invalid team_settings
					config = getConfig([]string{"team_settings"})
					config += "team_settings:\n  path: [2]\n"
					_, err = gitOpsFromString(t, config)
					assert.ErrorContains(t, err, "failed to unmarshal team_settings")

					// Invalid team_settings in a separate file
					tmpFile, err := os.CreateTemp(t.TempDir(), "*team_settings.yml")
					require.NoError(t, err)
					_, err = tmpFile.WriteString("[2]")
					require.NoError(t, err)
					config = getConfig([]string{"team_settings"})
					config += fmt.Sprintf("%s:\n  path: %s\n", "team_settings", tmpFile.Name())
					_, err = gitOpsFromString(t, config)
					assert.ErrorContains(t, err, "failed to unmarshal team settings file")

					// Invalid secrets 1
					config = getConfig([]string{"team_settings"})
					config += "team_settings:\n  secrets: bad\n"
					_, err = gitOpsFromString(t, config)
					assert.ErrorContains(t, err, "must be a list of secret items")

					// Invalid secrets 2
					config = getConfig([]string{"team_settings"})
					config += "team_settings:\n  secrets: [2]\n"
					_, err = gitOpsFromString(t, config)
					assert.ErrorContains(t, err, "must have a 'secret' key")

					// Missing team_settings.
					config = getConfig([]string{"team_settings"})
					_, err = gitOpsFromString(t, config)
					assert.ErrorContains(t, err, "'team_settings' is required when 'name' is provided")

					// team_settings set on a "no-team.yml".
					config = getConfig([]string{"name"})
					config += "name: No team\n"
					noTeamPath1, noTeamBasePath1 := createNamedFileOnTempDir(t, "no-team.yml", config)
					_, err = GitOpsFromFile(noTeamPath1, noTeamBasePath1, nil, nopLogf)
					assert.ErrorContains(t, err, fmt.Sprintf("cannot set 'team_settings' on 'No team' file: %q", noTeamPath1))

					// 'No team' file with invalid name.
					config = getConfig([]string{"name", "team_settings"})
					config += "name: No team\n"
					noTeamPath2, noTeamBasePath2 := createNamedFileOnTempDir(t, "foobar.yml", config)
					_, err = GitOpsFromFile(noTeamPath2, noTeamBasePath2, nil, nopLogf)
					assert.ErrorContains(t, err, fmt.Sprintf("file %q for 'No team' must be named 'no-team.yml'", noTeamPath2))

					// Missing secrets
					config = getConfig([]string{"team_settings"})
					config += "team_settings:\n"
					_, err = gitOpsFromString(t, config)
					assert.ErrorContains(t, err, "'team_settings.secrets' is required")
				} else {
					// 'software' is not allowed in global config
					config := getConfig(nil)
					config += "software:\n  packages:\n    - url: https://example.com\n"
					path1, basePath1 := createTempFile(t, "", config)
					appConfig := fleet.EnrichedAppConfig{}
					appConfig.License = &fleet.LicenseInfo{
						Tier: fleet.TierPremium,
					}
					_, err = GitOpsFromFile(path1, basePath1, &appConfig, nopLogf)
					assert.ErrorContains(t, err, "'software' cannot be set on global file")

					// Invalid org_settings
					config = getConfig([]string{"org_settings"})
					config += "org_settings:\n  path: [2]\n"
					_, err = gitOpsFromString(t, config)
					assert.ErrorContains(t, err, "failed to unmarshal org_settings")

					// Invalid org_settings in a separate file
					tmpFile, err := os.CreateTemp(t.TempDir(), "*org_settings.yml")
					require.NoError(t, err)
					_, err = tmpFile.WriteString("[2]")
					require.NoError(t, err)
					config = getConfig([]string{"org_settings"})
					config += fmt.Sprintf("%s:\n  path: %s\n", "org_settings", tmpFile.Name())
					_, err = gitOpsFromString(t, config)
					assert.ErrorContains(t, err, "failed to unmarshal org settings file")

					// Invalid secrets 1
					config = getConfig([]string{"org_settings"})
					config += "org_settings:\n  secrets: bad\n"
					_, err = gitOpsFromString(t, config)
					assert.ErrorContains(t, err, "must be a list of secret items")

					// Invalid secrets 2
					config = getConfig([]string{"org_settings"})
					config += "org_settings:\n  secrets: [2]\n"
					_, err = gitOpsFromString(t, config)
					assert.ErrorContains(t, err, "must have a 'secret' key")

					// Missing secrets
					config = getConfig([]string{"org_settings"})
					config += "org_settings:\n"
					_, err = gitOpsFromString(t, config)
					assert.ErrorContains(t, err, "'org_settings.secrets' is required")
				}

				// Invalid agent_options
				config := getConfig([]string{"agent_options"})
				config += "agent_options:\n  path: [2]\n"
				_, err = gitOpsFromString(t, config)
				assert.ErrorContains(t, err, "failed to unmarshal agent_options")

				// Invalid agent_options in a separate file
				tmpFile, err := os.CreateTemp(t.TempDir(), "*agent_options.yml")
				require.NoError(t, err)
				_, err = tmpFile.WriteString("[2]")
				require.NoError(t, err)
				config = getConfig([]string{"agent_options"})
				config += fmt.Sprintf("%s:\n  path: %s\n", "agent_options", tmpFile.Name())
				_, err = gitOpsFromString(t, config)
				assert.ErrorContains(t, err, "failed to unmarshal agent options file")

				// Invalid controls
				config = getConfig([]string{"controls"})
				config += "controls:\n  path: [2]\n"
				_, err = gitOpsFromString(t, config)
				assert.ErrorContains(t, err, "failed to unmarshal controls")

				// Invalid controls in a separate file
				tmpFile, err = os.CreateTemp(t.TempDir(), "*controls.yml")
				require.NoError(t, err)
				_, err = tmpFile.WriteString("[2]")
				require.NoError(t, err)
				config = getConfig([]string{"controls"})
				config += fmt.Sprintf("%s:\n  path: %s\n", "controls", tmpFile.Name())
				_, err = gitOpsFromString(t, config)
				assert.ErrorContains(t, err, "failed to unmarshal controls file")

				// Invalid policies
				config = getConfig([]string{"policies"})
				config += "policies:\n  path: [2]\n"
				_, err = gitOpsFromString(t, config)
				assert.ErrorContains(t, err, "failed to unmarshal policies")

				// Invalid policies in a separate file
				tmpFile, err = os.CreateTemp(t.TempDir(), "*policies.yml")
				require.NoError(t, err)
				_, err = tmpFile.WriteString("[2]")
				require.NoError(t, err)
				config = getConfig([]string{"policies"})
				config += fmt.Sprintf("%s:\n  - path: %s\n", "policies", tmpFile.Name())
				_, err = gitOpsFromString(t, config)
				assert.ErrorContains(t, err, "failed to unmarshal policies file")

				// Policy name missing
				config = getConfig([]string{"policies"})
				config += "policies:\n  - query: SELECT 1;\n"
				_, err = gitOpsFromString(t, config)
				assert.ErrorContains(t, err, "name is required")

				// Policy query missing
				config = getConfig([]string{"policies"})
				config += "policies:\n  - name: Test Policy\n"
				_, err = gitOpsFromString(t, config)
				assert.ErrorContains(t, err, "query is required")

				// Invalid queries
				config = getConfig([]string{"queries"})
				config += "queries:\n  path: [2]\n"
				_, err = gitOpsFromString(t, config)
				assert.ErrorContains(t, err, "failed to unmarshal queries")

				// Invalid policies in a separate file
				tmpFile, err = os.CreateTemp(t.TempDir(), "*queries.yml")
				require.NoError(t, err)
				_, err = tmpFile.WriteString("[2]")
				require.NoError(t, err)
				config = getConfig([]string{"queries"})
				config += fmt.Sprintf("%s:\n  - path: %s\n", "queries", tmpFile.Name())
				_, err = gitOpsFromString(t, config)
				assert.ErrorContains(t, err, "failed to unmarshal queries file")

				// Query name missing
				config = getConfig([]string{"queries"})
				config += "queries:\n  - query: SELECT 1;\n"
				_, err = gitOpsFromString(t, config)
				assert.ErrorContains(t, err, "name is required")

				// Query SQL query missing
				config = getConfig([]string{"queries"})
				config += "queries:\n  - name: Test Query\n"
				_, err = gitOpsFromString(t, config)
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
				_, err := gitOpsFromString(t, config)
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
	gitops, err := gitOpsFromString(t, config)
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
				tmpDir := t.TempDir()
				tmpFile, err := os.CreateTemp(tmpDir, "*good.yml")
				require.NoError(t, err)
				_, err = tmpFile.WriteString(test.goodConfig)
				require.NoError(t, err)
				config := getConfig([]string{name})
				if test.isArray {
					config += fmt.Sprintf("%s:\n  - path: %s\n", name, tmpFile.Name())
				} else {
					config += fmt.Sprintf("%s:\n  path: %s\n", name, tmpFile.Name())
				}
				_, err = gitOpsFromString(t, config)
				assert.NoError(t, err)

				// Test a relative top level path
				config = getConfig([]string{name})
				mainTmpFile, err := os.CreateTemp(tmpDir, "*main.yml")
				require.NoError(t, err)
				dir, file := filepath.Split(tmpFile.Name())
				if test.isArray {
					config += fmt.Sprintf("%s:\n  - path: ./%s\n", name, file)
				} else {
					config += fmt.Sprintf("%s:\n  path: ./%s\n", name, file)
				}
				err = os.WriteFile(mainTmpFile.Name(), []byte(config), 0o644)
				require.NoError(t, err)

				_, err = GitOpsFromFile(mainTmpFile.Name(), dir, nil, nopLogf)
				assert.NoError(t, err)

				// Test a bad path
				config = getConfig([]string{name})
				if test.isArray {
					config += fmt.Sprintf("%s:\n  - path: ./%s\n", name, "doesNotExist.yml")
				} else {
					config += fmt.Sprintf("%s:\n  path: ./%s\n", name, "doesNotExist.yml")
				}
				err = os.WriteFile(mainTmpFile.Name(), []byte(config), 0o644)
				require.NoError(t, err)

				_, err = GitOpsFromFile(mainTmpFile.Name(), dir, nil, nopLogf)
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
				_, err = gitOpsFromString(t, config)
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
				err = os.WriteFile(mainTmpFile.Name(), []byte(config), 0o644)
				require.NoError(t, err)
				_, err = GitOpsFromFile(mainTmpFile.Name(), dir, nil, nopLogf)
				assert.ErrorContains(t, err, "nested paths are not supported")
			},
		)
	}
}

func TestGitOpsGlobalPolicyWithInstallSoftware(t *testing.T) {
	t.Parallel()
	config := getGlobalConfig([]string{"policies"})
	config += `
policies:
- name: Some policy
  query: SELECT 1;
  install_software:
    package_path: ./some_path.yml
`
	_, err := gitOpsFromString(t, config)
	assert.ErrorContains(t, err, "install_software can only be set on team policies")
}

func TestGitOpsGlobalPolicyWithRunScript(t *testing.T) {
	t.Parallel()
	config := getGlobalConfig([]string{"policies"})
	config += `
policies:
- name: Some policy
  query: SELECT 1;
  run_script:
    path: ./some_path.sh
`
	_, err := gitOpsFromString(t, config)
	assert.ErrorContains(t, err, "run_script can only be set on team policies")
}

func TestGitOpsTeamPolicyWithInvalidInstallSoftware(t *testing.T) {
	t.Parallel()
	config := getTeamConfig([]string{"policies"})
	config += `
policies:
- name: Some policy
  query: SELECT 1;
  install_software:
    package_path: ./some_path.yml
`
	_, err := gitOpsFromString(t, config)
	assert.ErrorContains(t, err, "failed to read install_software.package_path file")

	config = getTeamConfig([]string{"policies"})
	config += `
policies:
- name: Some policy
  query: SELECT 1;
  install_software:
    package_path:
`
	_, err = gitOpsFromString(t, config)
	assert.ErrorContains(t, err, "must include either a package path or app store app ID")

	config = getTeamConfig([]string{"policies"})
	config += `
policies:
- name: Some policy
  query: SELECT 1;
  install_software:
    package_path: ./some_path.yml
    app_store_id: "123456"
`
	_, err = gitOpsFromString(t, config)
	assert.ErrorContains(t, err, "must have only one of package_path or app_store_id")

	// Software has a URL that's too big
	tooBigURL := fmt.Sprintf("https://ftp.mozilla.org/%s", strings.Repeat("a", 4000-23))
	config = getTeamConfig([]string{"software"})
	config += fmt.Sprintf(`
software:
  packages:
    - url: %s
`, tooBigURL)
	appConfig := fleet.EnrichedAppConfig{}
	appConfig.License = &fleet.LicenseInfo{
		Tier: fleet.TierPremium,
	}
	path, basePath := createTempFile(t, "", config)
	_, err = GitOpsFromFile(path, basePath, &appConfig, nopLogf)
	assert.ErrorContains(t, err, fmt.Sprintf("software URL \"%s\" is too long, must be 4000 characters or less", tooBigURL))

	// Policy references a VPP app not present on the team
	config = getTeamConfig([]string{"policies"})
	config += `
policies:
- name: Some policy
  query: SELECT 1;
  install_software:
    app_store_id: "123456"
`
	_, err = gitOpsFromString(t, config)
	assert.ErrorContains(t, err, "not found on team")

	// Policy references a software installer not present in the team.
	config = getTeamConfig([]string{"policies"})
	config += `
policies:
  - path: ./team_install_software.policies.yml
software:
  packages:
    - url: https://ftp.mozilla.org/pub/firefox/releases/129.0.2/mac/en-US/Firefox%20129.0.2.pkg
      self_service: true

`
	path, basePath = createTempFile(t, "", config)
	err = file.Copy(
		filepath.Join("testdata", "team_install_software.policies.yml"),
		filepath.Join(basePath, "team_install_software.policies.yml"),
		0o755,
	)
	require.NoError(t, err)
	err = file.Copy(
		filepath.Join("testdata", "microsoft-teams.pkg.software.yml"),
		filepath.Join(basePath, "microsoft-teams.pkg.software.yml"),
		0o755,
	)
	require.NoError(t, err)
	_, err = GitOpsFromFile(path, basePath, &appConfig, nopLogf)
	assert.ErrorContains(t, err,
		"install_software.package_path URL https://statics.teams.cdn.office.net/production-osx/enterprise/webview2/lkg/MicrosoftTeams.pkg not found on team",
	)

	// Policy references a software installer file that has an invalid yaml.
	config = getTeamConfig([]string{"policies"})
	config += `
policies:
  - path: ./team_install_software.policies.yml
software:
  packages:
    - url: https://ftp.mozilla.org/pub/firefox/releases/129.0.2/mac/en-US/Firefox%20129.0.2.pkg
      self_service: true
`
	path, basePath = createTempFile(t, "", config)
	err = file.Copy(
		filepath.Join("testdata", "team_install_software.policies.yml"),
		filepath.Join(basePath, "team_install_software.policies.yml"),
		0o755,
	)
	require.NoError(t, err)
	err = os.WriteFile( // nolint:gosec
		filepath.Join(basePath, "microsoft-teams.pkg.software.yml"),
		[]byte("invalid yaml"),
		0o755,
	)
	require.NoError(t, err)
	appConfig = fleet.EnrichedAppConfig{}
	appConfig.License = &fleet.LicenseInfo{
		Tier: fleet.TierPremium,
	}
	_, err = GitOpsFromFile(path, basePath, &appConfig, nopLogf)
	assert.ErrorContains(t, err, "failed to unmarshal install_software.package_path file")
}

func TestGitOpsWithStrayScriptEntryWithNoPath(t *testing.T) {
	t.Parallel()
	config := getTeamConfig([]string{"controls"})
	config += `
controls:
  scripts:
    -
`
	_, err := gitOpsFromString(t, config)
	assert.ErrorContains(t, err, `check for a stray "-"`)
}

func TestGitOpsTeamPolicyWithInvalidRunScript(t *testing.T) {
	t.Parallel()
	config := getTeamConfig([]string{"policies"})
	config += `
policies:
- name: Some policy
  query: SELECT 1;
  run_script:
    path: ./some_path.sh
`
	_, err := gitOpsFromString(t, config)
	assert.ErrorContains(t, err, "script file does not exist")

	config = getTeamConfig([]string{"policies"})
	config += `
policies:
- name: Some policy
  query: SELECT 1;
  run_script:
    path:
`
	_, err = gitOpsFromString(t, config)
	assert.ErrorContains(t, err, "empty run_script path")

	// Policy references a script not present in the team.
	config = getTeamConfig([]string{"policies"})
	config += `
policies:
  - path: ./policies/script-policy.yml
software:
controls:
  scripts:
    - path: ./policies/policies2.yml

`
	path, basePath := createTempFile(t, "", config)
	err = file.Copy(
		filepath.Join("testdata", "policies", "script-policy.yml"),
		filepath.Join(basePath, "policies", "script-policy.yml"),
		0o755,
	)
	require.NoError(t, err)
	err = file.Copy(
		filepath.Join("testdata", "lib", "collect-fleetd-logs.sh"),
		filepath.Join(basePath, "lib", "collect-fleetd-logs.sh"),
		0o755,
	)
	require.NoError(t, err)
	appConfig := fleet.EnrichedAppConfig{}
	appConfig.License = &fleet.LicenseInfo{
		Tier: fleet.TierPremium,
	}
	_, err = GitOpsFromFile(path, basePath, &appConfig, nopLogf)
	assert.ErrorContains(t, err,
		"was not defined in controls for TeamName",
	)
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

func TestIllegalFleetSecret(t *testing.T) {
	t.Parallel()
	config := getGlobalConfig([]string{"policies"})
	config += `
policies:
  - name: $FLEET_SECRET_POLICY
    platform: linux
    query: SELECT 1 FROM osquery_info WHERE start_time < 0;
  - name: My policy
    platform: windows
    query: SELECT 1;
`
	_, err := gitOpsFromString(t, config)
	assert.ErrorContains(t, err, "variables with \"FLEET_SECRET_\" prefix are only allowed")
}
