package spec

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/file"
	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/ghodss/yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// yamlToRawJSON converts a YAML string into the map[string]json.RawMessage that parse functions expect.
func yamlToRawJSON(t *testing.T, yamlStr string) map[string]json.RawMessage {
	t.Helper()
	j, err := yaml.YAMLToJSON([]byte(yamlStr))
	require.NoError(t, err)
	var top map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(j, &top))
	return top
}

var topLevelOptions = map[string]string{
	"controls":      "controls:",
	"reports":       "reports:",
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
	"reports":       "reports:",
	"policies":      "policies:",
	"agent_options": "agent_options:",
	"name":          "name: TeamName",
	"settings": `
settings:
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

func premiumAppConfig() *fleet.EnrichedAppConfig {
	ac := &fleet.EnrichedAppConfig{}
	ac.License = &fleet.LicenseInfo{Tier: fleet.TierPremium}
	return ac
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
				"FLEET_SECRET_LENGTH":        "10",
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
				"FLEET_SECRET_LENGTH":           "10",
				"FLEET_SECRET_BANANA":           "bread",
			},
			filePath: "testdata/global_config.yml",
		},
		"team_config_no_paths": {
			environment: map[string]string{
				"FLEET_SECRET_FLEET_SECRET_": "fleet_secret",
				"FLEET_SECRET_NAME":          "secret_name",
				"FLEET_SECRET_LENGTH":        "10",
				"FLEET_SECRET_BANANA":        "bread",
				"FLEET_SECRET_CLEMENTINE":    "not-an-orange",
				"FLEET_SECRET_DURIAN":        "fruity", // not used
				"FLEET_SECRET_EGGPLANT":      "parmesan",
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
				"FLEET_SECRET_LENGTH":             "10",
				"FLEET_SECRET_BANANA":             "bread",
				"FLEET_SECRET_CLEMENTINE":         "not-an-orange",
				"FLEET_SECRET_DURIAN":             "fruity", // not used
				"FLEET_SECRET_EGGPLANT":           "parmesan",
			},
			filePath: "testdata/team_config.yml",
			isTeam:   true,
		},
		"team_config_with_paths_and_only_sha256": {
			environment: map[string]string{
				"POLICY":                          "policy",
				"LINUX_OS":                        "linux",
				"DISTRIBUTED_DENYLIST_DURATION":   "0",
				"ENABLE_FAILING_POLICIES_WEBHOOK": "true",
				"FLEET_SECRET_FLEET_SECRET_":      "fleet_secret",
				"FLEET_SECRET_NAME":               "secret_name",
				"FLEET_SECRET_LENGTH":             "10",
				"FLEET_SECRET_BANANA":             "bread",
				"FLEET_SECRET_CLEMENTINE":         "not-an-orange",
				"FLEET_SECRET_DURIAN":             "fruity", // not used
				"FLEET_SECRET_EGGPLANT":           "parmesan",
			},
			filePath: "testdata/team_config_only_sha256.yml",
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
					require.Len(t, gitops.FleetSecrets, 6)
					for _, pkg := range gitops.Software.Packages {
						if strings.Contains(pkg.URL, "MicrosoftTeams") {
							assert.Equal(t, "testdata/lib/uninstall.sh", pkg.UninstallScript.Path)
							assert.Contains(t, pkg.LabelsIncludeAny, "a")
							assert.Contains(t, pkg.Categories, "Communication")
							assert.Empty(t, pkg.LabelsExcludeAny)
							assert.Empty(t, pkg.LabelsIncludeAll)
						} else {
							assert.Empty(t, pkg.UninstallScript.Path)
							assert.Contains(t, pkg.LabelsExcludeAny, "a")
							assert.Empty(t, pkg.LabelsIncludeAny)
							assert.Empty(t, pkg.LabelsIncludeAll)
						}
					}
					require.Len(t, gitops.Software.FleetMaintainedApps, 2)
					for _, fma := range gitops.Software.FleetMaintainedApps {
						switch fma.Slug {
						case "slack/darwin":
							require.ElementsMatch(t, fma.Categories, []string{"Productivity", "Communication"})
							require.Equal(t, "4.47.65", fma.Version)
							require.Empty(t, fma.PreInstallQuery)
							require.Empty(t, fma.PostInstallScript)
							require.Empty(t, fma.InstallScript)
							require.Empty(t, fma.UninstallScript)
						case "box-drive/windows":
							require.ElementsMatch(t, fma.Categories, []string{"Productivity", "Developer tools"})
							require.Empty(t, fma.Version)
							require.NotEmpty(t, fma.PreInstallQuery)
							require.NotEmpty(t, fma.PostInstallScript)
							require.NotEmpty(t, fma.InstallScript)
							require.NotEmpty(t, fma.UninstallScript)
						default:
							assert.FailNow(t, "unexpected slug found in gitops file", "slug: %s", fma.Slug)
						}
					}
				} else {
					// Check org settings
					serverSettings, ok := gitops.OrgSettings["server_settings"]
					assert.True(t, ok, "server_settings not found")
					assert.Equal(t, "https://fleet.example.com", serverSettings.(map[string]any)["server_url"])
					assert.EqualValues(t, 2000, serverSettings.(map[string]any)["report_cap"])
					assert.Contains(t, gitops.OrgSettings, "org_info")
					orgInfo, ok := gitops.OrgSettings["org_info"].(map[string]any)
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
					require.Len(t, gitops.FleetSecrets, 4)

					// Check labels
					require.Len(t, gitops.Labels, 2)
					assert.Equal(t, "Global label numero uno", gitops.Labels[0].Name)
					assert.Equal(t, "Global label numero dos", gitops.Labels[1].Name)
					assert.Equal(t, "SELECT 1 FROM osquery_info", gitops.Labels[0].Query)
					require.Len(t, gitops.Labels[1].Hosts, 2)
					assert.Equal(t, "host1", gitops.Labels[1].Hosts[0])
					assert.Equal(t, "2", gitops.Labels[1].Hosts[1])

				}

				// Check controls
				_, ok := gitops.Controls.MacOSSettings.(fleet.MacOSSettings)
				assert.True(t, ok, "macos_settings not found")
				_, ok = gitops.Controls.WindowsSettings.(fleet.WindowsSettings)
				assert.True(t, ok, "windows_settings not found")
				_, ok = gitops.Controls.EnableDiskEncryption.(bool)
				assert.True(t, ok, "enable_disk_encryption not found")
				_, ok = gitops.Controls.EnableRecoveryLockPassword.(bool)
				assert.True(t, ok, "enable_recovery_lock_password not found")
				_, ok = gitops.Controls.MacOSMigration.(map[string]interface{})
				assert.True(t, ok, "macos_migration not found")
				assert.NotNil(t, gitops.Controls.MacOSSetup, "macos_setup not found")
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
				_, ok = gitops.Controls.EnableTurnOnWindowsMDMManually.(bool)
				assert.True(t, ok, "enable_turn_on_windows_mdm_manually not found")
				_, ok = gitops.Controls.WindowsEntraTenantIDs.([]any)
				assert.True(t, ok, "windows_entra_tenant_ids not found")
				_, ok = gitops.Controls.WindowsUpdates.(map[string]interface{})
				assert.True(t, ok, "windows_updates not found")
				_, ok = gitops.Controls.AppleRequireHardwareAttestation.(bool)
				assert.True(t, ok, "apple_require_hardware_attestation not found")
				assert.Equal(t, "fleet_secret", gitops.FleetSecrets["FLEET_SECRET_FLEET_SECRET_"])
				assert.Equal(t, "secret_name", gitops.FleetSecrets["FLEET_SECRET_NAME"])
				assert.Equal(t, "10", gitops.FleetSecrets["FLEET_SECRET_LENGTH"])
				assert.Equal(t, "bread", gitops.FleetSecrets["FLEET_SECRET_BANANA"])

				// Check agent options
				assert.NotNil(t, gitops.AgentOptions)
				assert.Contains(t, string(*gitops.AgentOptions), "\"distributed_denylist_duration\":0")

				// Check reports
				require.Len(t, gitops.Queries, 3)
				assert.Equal(t, "Scheduled query stats", gitops.Queries[0].Name)
				assert.Equal(t, "orbit_info", gitops.Queries[1].Name)
				assert.Equal(t, "darwin,linux,windows", gitops.Queries[1].Platform)
				assert.Equal(t, "osquery_info", gitops.Queries[2].Name)

				// Check software
				if test.isTeam {
					require.Len(t, gitops.Software.Packages, 2)
					if name == "team_config_with_paths_and_only_sha256" {
						require.Empty(t, gitops.Software.Packages[0].URL)
						require.True(t, gitops.Software.Packages[0].InstallDuringSetup.Value)
						require.True(t, gitops.Software.Packages[1].InstallDuringSetup.Value)
					} else {
						require.Equal(t, "https://statics.teams.cdn.office.net/production-osx/enterprise/webview2/lkg/MicrosoftTeams.pkg", gitops.Software.Packages[0].URL)
					}
					require.Equal(t, "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", gitops.Software.Packages[0].SHA256)
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

					if name == "team_config_with_paths_and_only_sha256" {
						assert.Equal(t, "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", gitops.Policies[5].InstallSoftware.Other.HashSHA256)
					} else {
						assert.Equal(t, "./microsoft-teams.pkg.software.yml", gitops.Policies[5].InstallSoftware.Other.PackagePath)
						assert.Equal(t, "https://statics.teams.cdn.office.net/production-osx/enterprise/webview2/lkg/MicrosoftTeams.pkg", gitops.Policies[5].InstallSoftwareURL)
					}

					assert.Equal(t, "Slack on macOS is installed", gitops.Policies[6].Name)
					assert.NotNil(t, gitops.Policies[6].InstallSoftware)
					assert.Equal(t, "123456", gitops.Policies[6].InstallSoftware.Other.AppStoreID)

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

func TestManualLabelEmptyHostList(t *testing.T) {
	t.Parallel()
	config := getGlobalConfig([]string{})
	config += `
labels:
  - name: TestLabel
    description: Label for testing
    hosts: []
    label_membership_type: manual`

	gitops, err := gitOpsFromString(t, config)
	require.NoError(t, err)
	require.NotNil(t, gitops.Labels[0].Hosts)
	assert.Empty(t, gitops.Labels[0].Hosts)
}

func TestManualLabelNullHostsKey(t *testing.T) {
	t.Parallel()
	config := getGlobalConfig([]string{})
	config += `
labels:
  - name: TestLabel
    description: Label for testing
    hosts:
    label_membership_type: manual`

	gitops, err := gitOpsFromString(t, config)
	require.NoError(t, err)
	// hosts key present with null value should produce a non-nil empty slice
	// (meaning "clear all hosts"), distinct from a nil slice (key omitted,
	// meaning "preserve existing membership").
	require.NotNil(t, gitops.Labels[0].Hosts)
	assert.Empty(t, gitops.Labels[0].Hosts)
}

func TestDuplicateQueryNames(t *testing.T) {
	t.Parallel()
	config := getGlobalConfig([]string{"reports"})
	config += `
reports:
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
	assert.ErrorContains(t, err, "duplicate report names")
}

func TestUnicodeQueryNames(t *testing.T) {
	t.Parallel()
	config := getGlobalConfig([]string{"reports"})
	config += `
reports:
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
	assert.ErrorContains(t, err, "`name` must be in ASCII")
}

func TestUnicodeTeamName(t *testing.T) {
	t.Parallel()
	config := getTeamConfig([]string{"name"})
	config += `name: 😊 TeamName`
	_, err := gitOpsFromString(t, config)
	assert.NoError(t, err)
}

func TestWhitespaceOnlyTeamName(t *testing.T) {
	t.Parallel()
	config := getTeamConfig([]string{"name"})
	config += `name: "     "`
	_, err := gitOpsFromString(t, config)
	require.Error(t, err)
	require.Contains(t, err.Error(), "team 'name' is required")
}

func TestPaddedTeamNameIsTrimmed(t *testing.T) {
	t.Parallel()
	config := getTeamConfig([]string{"name"})
	config += `name: "  Team Name  "`
	gitOps, err := gitOpsFromString(t, config)
	require.NoError(t, err)
	require.NotNil(t, gitOps.TeamName)
	require.Equal(t, "Team Name", *gitOps.TeamName)
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
	config := getGlobalConfig([]string{"reports"})
	config += `
reports:
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

	config = getGlobalConfig([]string{"reports"})
	config += `
reports:
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
	assert.ErrorContains(t, err, "'org_settings' cannot be used with 'name', 'settings'")

	// Mixing org_settings and settings (formerly settings)
	config = getGlobalConfig(nil)
	config += "settings:\n  secrets: []\n"
	_, err = gitOpsFromString(t, config)
	assert.ErrorContains(t, err, "'org_settings' cannot be used with 'name', 'settings'")

	// Mixing org_settings and team name and settings
	config = getGlobalConfig(nil)
	config += "name: TeamName\n"
	config += "settings:\n  secrets: []\n"
	_, err = gitOpsFromString(t, config)
	assert.ErrorContains(t, err, "'org_settings' cannot be used with 'name', 'settings'")
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
					assert.ErrorContains(t, err, "expected type string but got array")

					// Missing team name
					config = getConfig([]string{"name"})
					config += "name:\n"
					_, err = gitOpsFromString(t, config)
					assert.ErrorContains(t, err, "'name' is required")

					// Invalid settings
					config = getConfig([]string{"settings"})
					config += "settings:\n  path: [2]\n"
					_, err = gitOpsFromString(t, config)
					assert.ErrorContains(t, err, "expected type string but got array")

					// Invalid settings in a separate file
					tmpFile, err := os.CreateTemp(t.TempDir(), "*settings.yml")
					require.NoError(t, err)
					_, err = tmpFile.WriteString("[2]")
					require.NoError(t, err)
					config = getConfig([]string{"settings"})
					config += fmt.Sprintf("%s:\n  path: %s\n", "settings", tmpFile.Name())
					_, err = gitOpsFromString(t, config)
					assert.ErrorContains(t, err, "expected type fleet.BaseItem but got array")

					// Invalid secrets 1
					config = getConfig([]string{"settings"})
					config += "settings:\n  secrets: bad\n"
					_, err = gitOpsFromString(t, config)
					assert.ErrorContains(t, err, "must be a list of secret items")

					// Invalid secrets 2
					config = getConfig([]string{"settings"})
					config += "settings:\n  secrets: [2]\n"
					_, err = gitOpsFromString(t, config)
					assert.ErrorContains(t, err, "must have a 'secret' key")

					// Missing settings is now allowed (defaults to null, clearing team settings).
					config = getConfig([]string{"settings"})
					_, err = gitOpsFromString(t, config)
					assert.NoError(t, err)

					// settings is now allowed on "no-team.yml" for webhook settings
					config = getConfig([]string{"name", "settings"}) // Exclude settings with secrets
					config += "name: No team\n"
					noTeamPath1, noTeamBasePath1 := createNamedFileOnTempDir(t, "no-team.yml", config)
					gitops, err := GitOpsFromFile(noTeamPath1, noTeamBasePath1, nil, nopLogf)
					assert.NoError(t, err)
					assert.NotNil(t, gitops)

					// No team with valid webhook_settings should work
					config = getConfig([]string{"name", "settings"})
					config += "name: No team\nsettings:\n  webhook_settings:\n    failing_policies_webhook:\n      enable_failing_policies_webhook: true\n"
					noTeamPath2, noTeamBasePath2 := createNamedFileOnTempDir(t, "no-team.yml", config)
					gitops, err = GitOpsFromFile(noTeamPath2, noTeamBasePath2, nil, nopLogf)
					assert.NoError(t, err)
					assert.NotNil(t, gitops)

					// No team with invalid settings option should fail
					config = getConfig([]string{"name", "settings"})
					config += "name: No team\nsettings:\n  features:\n    enable_host_users: false\n"
					noTeamPath3, noTeamBasePath3 := createNamedFileOnTempDir(t, "no-team.yml", config)
					_, err = GitOpsFromFile(noTeamPath3, noTeamBasePath3, nil, nopLogf)
					assert.ErrorContains(t, err, "unsupported settings option 'features' in no-team.yml - only 'webhook_settings' is allowed")

					// No team with multiple settings options (one valid, one invalid) should fail
					config = getConfig([]string{"name", "settings"})
					config += "name: No team\nsettings:\n  webhook_settings:\n    failing_policies_webhook:\n      enable_failing_policies_webhook: true\n  secrets:\n    - secret: test\n"
					noTeamPath4, noTeamBasePath4 := createNamedFileOnTempDir(t, "no-team.yml", config)
					_, err = GitOpsFromFile(noTeamPath4, noTeamBasePath4, nil, nopLogf)
					assert.ErrorContains(t, err, "unsupported settings option 'secrets' in no-team.yml - only 'webhook_settings' is allowed")

					// No team with host_status_webhook in webhook_settings should fail
					config = getConfig([]string{"name", "settings"})
					config += "name: No team\nsettings:\n  webhook_settings:\n    host_status_webhook:\n      enable_host_status_webhook: true\n    failing_policies_webhook:\n      enable_failing_policies_webhook: true\n"
					noTeamPath5a, noTeamBasePath5a := createNamedFileOnTempDir(t, "no-team.yml", config)
					_, err = GitOpsFromFile(noTeamPath5a, noTeamBasePath5a, nil, nopLogf)
					assert.ErrorContains(t, err, "unsupported webhook_settings option 'host_status_webhook' in no-team.yml - only 'failing_policies_webhook' is allowed")

					// No team with vulnerabilities_webhook in webhook_settings should fail
					config = getConfig([]string{"name", "settings"})
					config += "name: No team\nsettings:\n  webhook_settings:\n    vulnerabilities_webhook:\n      enable_vulnerabilities_webhook: true\n"
					noTeamPath5b, noTeamBasePath5b := createNamedFileOnTempDir(t, "no-team.yml", config)
					_, err = GitOpsFromFile(noTeamPath5b, noTeamBasePath5b, nil, nopLogf)
					assert.ErrorContains(t, err, "unsupported webhook_settings option 'vulnerabilities_webhook' in no-team.yml - only 'failing_policies_webhook' is allowed")

					// 'No team' file with invalid name.
					config = getConfig([]string{"name", "settings"})
					config += "name: No team\n"
					noTeamPath6, noTeamBasePath6 := createNamedFileOnTempDir(t, "foobar.yml", config)
					_, err = GitOpsFromFile(noTeamPath6, noTeamBasePath6, nil, nopLogf)
					assert.ErrorContains(t, err, fmt.Sprintf("file `%s` for No Team must be named `no-team.yml`", noTeamPath6))

					// no-team.yml with a non-"No Team" name should fail.
					config = getConfig([]string{"name", "settings"})
					config += "name: SomeOtherTeam\nsettings:\n  secrets:\n"
					noTeamPath7, noTeamBasePath7 := createNamedFileOnTempDir(t, "no-team.yml", config)
					_, err = GitOpsFromFile(noTeamPath7, noTeamBasePath7, nil, nopLogf)
					assert.ErrorContains(t, err, fmt.Sprintf("file %q must have team name 'No Team'", noTeamPath7))

					// unassigned.yml with a non-"Unassigned" name should fail.
					config = getConfig([]string{"name", "settings"})
					config += "name: SomeOtherTeam\nsettings:\n  secrets:\n"
					unassignedPathBadName, unassignedBasePathBadName := createNamedFileOnTempDir(t, "unassigned.yml", config)
					_, err = GitOpsFromFile(unassignedPathBadName, unassignedBasePathBadName, nil, nopLogf)
					assert.ErrorContains(t, err, fmt.Sprintf("file %q must have team name 'Unassigned'", unassignedPathBadName))

					// no-team.yml with "Unassigned" name should fail (wrong name for this file).
					config = getConfig([]string{"name", "settings"})
					config += "name: Unassigned\n"
					noTeamPath8, noTeamBasePath8 := createNamedFileOnTempDir(t, "no-team.yml", config)
					_, err = GitOpsFromFile(noTeamPath8, noTeamBasePath8, nil, nopLogf)
					assert.ErrorContains(t, err, fmt.Sprintf("file %q must have team name 'No Team'", noTeamPath8))

					// unassigned.yml with "No team" name should fail (wrong name for this file).
					config = getConfig([]string{"name", "settings"})
					config += "name: No team\n"
					unassignedPathNoTeam, unassignedBasePathNoTeam := createNamedFileOnTempDir(t, "unassigned.yml", config)
					_, err = GitOpsFromFile(unassignedPathNoTeam, unassignedBasePathNoTeam, nil, nopLogf)
					assert.ErrorContains(t, err, fmt.Sprintf("file %q must have team name 'Unassigned'", unassignedPathNoTeam))

					// 'Unassigned' team in unassigned.yml should work and coerce to "No team" internally.
					config = getConfig([]string{"name", "settings"})
					config += "name: Unassigned\n"
					unassignedPath1, unassignedBasePath1 := createNamedFileOnTempDir(t, "unassigned.yml", config)
					gitops, err = GitOpsFromFile(unassignedPath1, unassignedBasePath1, nil, nopLogf)
					assert.NoError(t, err)
					assert.NotNil(t, gitops)
					assert.True(t, gitops.IsNoTeam(), "unassigned.yml should be treated as no-team after coercion")
					assert.Equal(t, "No team", *gitops.TeamName)

					// 'Unassigned' team with wrong filename should fail.
					config = getConfig([]string{"name", "settings"})
					config += "name: Unassigned\n"
					unassignedPath2, unassignedBasePath2 := createNamedFileOnTempDir(t, "foobar.yml", config)
					_, err = GitOpsFromFile(unassignedPath2, unassignedBasePath2, nil, nopLogf)
					assert.ErrorContains(t, err, fmt.Sprintf("file `%s` for unassigned hosts must be named `unassigned.yml`", unassignedPath2))

					// 'Unassigned' (case-insensitive) in unassigned.yml should work.
					config = getConfig([]string{"name", "settings"})
					config += "name: unassigned\n"
					unassignedPath3, unassignedBasePath3 := createNamedFileOnTempDir(t, "unassigned.yml", config)
					gitops, err = GitOpsFromFile(unassignedPath3, unassignedBasePath3, nil, nopLogf)
					assert.NoError(t, err)
					assert.NotNil(t, gitops)
					assert.True(t, gitops.IsNoTeam())

					// 'Unassigned' with webhook settings in unassigned.yml should work.
					config = getConfig([]string{"name", "settings"})
					config += "name: Unassigned\nsettings:\n  webhook_settings:\n    failing_policies_webhook:\n      enable_failing_policies_webhook: true\n"
					unassignedPath4, unassignedBasePath4 := createNamedFileOnTempDir(t, "unassigned.yml", config)
					gitops, err = GitOpsFromFile(unassignedPath4, unassignedBasePath4, nil, nopLogf)
					assert.NoError(t, err)
					assert.NotNil(t, gitops)

					// 'Unassigned' with invalid settings option should fail with unassigned.yml in message.
					config = getConfig([]string{"name", "settings"})
					config += "name: Unassigned\nsettings:\n  features:\n    enable_host_users: false\n"
					unassignedPath5, unassignedBasePath5 := createNamedFileOnTempDir(t, "unassigned.yml", config)
					_, err = GitOpsFromFile(unassignedPath5, unassignedBasePath5, nil, nopLogf)
					assert.ErrorContains(t, err, "unsupported settings option 'features' in unassigned.yml")

					// Missing secrets -- should be a no-op (existing secrets preserved)
					config = getConfig([]string{"settings"})
					config += "settings:\n"
					result, err := gitOpsFromString(t, config)
					assert.NoError(t, err)
					_, hasSecrets := result.TeamSettings["secrets"]
					assert.False(t, hasSecrets, "secrets should not be set when omitted from config")
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
					assert.ErrorContains(t, err, "expected type string but got array")

					// Invalid org_settings in a separate file
					tmpFile, err := os.CreateTemp(t.TempDir(), "*org_settings.yml")
					require.NoError(t, err)
					_, err = tmpFile.WriteString("[2]")
					require.NoError(t, err)
					config = getConfig([]string{"org_settings"})
					config += fmt.Sprintf("%s:\n  path: %s\n", "org_settings", tmpFile.Name())
					_, err = gitOpsFromString(t, config)
					assert.ErrorContains(t, err, "expected type fleet.BaseItem but got array")

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

					// Invalid secrets 3 (using wrong type in one key)
					config = getConfig([]string{"org_settings"})
					config += "org_settings:\n  secrets: \n    - secret: some secret\n    - secret: 123\n"
					_, err = gitOpsFromString(t, config)
					assert.ErrorContains(t, err, "each item in 'secrets' must have a 'secret' key")

					// Missing secrets -- should be a no-op (existing secrets preserved)
					config = getConfig([]string{"org_settings"})
					config += "org_settings:\n"
					result, err := gitOpsFromString(t, config)
					assert.NoError(t, err)
					_, hasSecrets := result.OrgSettings["secrets"]
					assert.False(t, hasSecrets, "secrets should not be set when omitted from config")

					// Empty secrets (valid, will remove all secrets)
					config = getConfig([]string{"org_settings"})
					config += "org_settings:\n  secrets: \n"
					_, err = gitOpsFromString(t, config)
					assert.NoError(t, err)

					// Bad label spec (float instead of string in hosts)
					config = getConfig([]string{"labels"})
					config += "labels:\n  - name: TestLabel\n    description: Label for testing\n    hosts:\n    - 2.5\n    label_membership_type: manual\n"
					_, err = gitOpsFromString(t, config)
					assert.ErrorContains(t, err, "hosts must be strings or integers, got float 2.5")
				}

				// Invalid agent_options
				config := getConfig([]string{"agent_options"})
				config += "agent_options:\n  path: [2]\n"
				_, err = gitOpsFromString(t, config)
				assert.ErrorContains(t, err, "expected type string but got array")

				// Invalid agent_options in a separate file
				tmpFile, err := os.CreateTemp(t.TempDir(), "*agent_options.yml")
				require.NoError(t, err)
				_, err = tmpFile.WriteString("[2]")
				require.NoError(t, err)
				config = getConfig([]string{"agent_options"})
				config += fmt.Sprintf("%s:\n  path: %s\n", "agent_options", tmpFile.Name())
				_, err = gitOpsFromString(t, config)
				assert.ErrorContains(t, err, "expected type fleet.BaseItem but got array")

				// Invalid controls
				config = getConfig([]string{"controls"})
				config += "controls:\n  path: [2]\n"
				_, err = gitOpsFromString(t, config)
				assert.ErrorContains(t, err, "expected type string but got array")

				// Invalid controls in a separate file
				tmpFile, err = os.CreateTemp(t.TempDir(), "*controls.yml")
				require.NoError(t, err)
				_, err = tmpFile.WriteString("[2]")
				require.NoError(t, err)
				config = getConfig([]string{"controls"})
				config += fmt.Sprintf("%s:\n  path: %s\n", "controls", tmpFile.Name())
				_, err = gitOpsFromString(t, config)
				assert.ErrorContains(t, err, "expected type spec.GitOpsControls but got array")

				// Invalid policies
				config = getConfig([]string{"policies"})
				config += "policies:\n  path: [2]\n"
				_, err = gitOpsFromString(t, config)
				assert.ErrorContains(t, err, "expected type []spec.Policy but got object")

				// Invalid policies in a separate file
				tmpFile, err = os.CreateTemp(t.TempDir(), "*policies.yml")
				require.NoError(t, err)
				_, err = tmpFile.WriteString("[2]")
				require.NoError(t, err)
				config = getConfig([]string{"policies"})
				config += fmt.Sprintf("%s:\n  - path: %s\n", "policies", tmpFile.Name())
				_, err = gitOpsFromString(t, config)
				assert.ErrorContains(t, err, "expected type spec.Policy but got number")

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

				// Invalid reports
				config = getConfig([]string{"reports"})
				config += "reports:\n  path: [2]\n"
				_, err = gitOpsFromString(t, config)
				assert.ErrorContains(t, err, "expected type []spec.Query but got object")

				// Invalid reports in a separate file
				tmpFile, err = os.CreateTemp(t.TempDir(), "*reports.yml")
				require.NoError(t, err)
				_, err = tmpFile.WriteString("[2]")
				require.NoError(t, err)
				config = getConfig([]string{"reports"})
				config += fmt.Sprintf("%s:\n  - path: %s\n", "reports", tmpFile.Name())
				_, err = gitOpsFromString(t, config)
				assert.ErrorContains(t, err, "expected type spec.Query but got number")

				// Report name missing
				config = getConfig([]string{"reports"})
				config += "reports:\n  - query: SELECT 1;\n"
				_, err = gitOpsFromString(t, config)
				assert.ErrorContains(t, err, "`name` is required")

				// Report SQL missing
				config = getConfig([]string{"reports"})
				config += "reports:\n  - name: Test Query\n"
				_, err = gitOpsFromString(t, config)
				assert.ErrorContains(t, err, "`query` is required")
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
		// Top-level keys besides "name" and "org_settings" are now optional.
		// A file must have either "name" (team) or "org_settings" (global).
		"missing_all_global": {
			optsToExclude: []string{"controls", "reports", "policies", "agent_options", "org_settings"},
		},
		"missing_reports": {
			optsToExclude: []string{"reports"},
			shouldPass:    true,
		},
		"missing_policies": {
			optsToExclude: []string{"policies"},
			shouldPass:    true,
		},
		"missing_agent_options": {
			optsToExclude: []string{"agent_options"},
			shouldPass:    true,
		},
		"missing_org_settings": {
			optsToExclude: []string{"org_settings"},
		},
		"missing_name": {
			optsToExclude: []string{"name"},
			isTeam:        true,
		},
		"missing_settings": {
			optsToExclude: []string{"settings"},
			shouldPass:    true,
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

	config := getGlobalConfig([]string{"reports", "policies"})
	config += "reports: null\npolicies: ~\n"
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
		"settings": {
			isArray:    false,
			isTeam:     true,
			goodConfig: "secrets: []\n",
		},
		"controls": {
			isArray:    false,
			goodConfig: "windows_enabled_and_configured: true\n",
		},
		"reports": {
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
				tmpFileBad, err = os.CreateTemp(filepath.Dir(mainTmpFile.Name()), "*bad.yml")
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
	assert.ErrorContains(t, err, "install_software must include either a package_path, an app_store_id, a hash_sha256 or a fleet_maintained_app_slug")

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
	assert.ErrorContains(t, err, "must have only one of package_path, app_store_id, hash_sha256 or fleet_maintained_app_slug")

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

	// Software URL isn't a valid URL
	config = getTeamConfig([]string{"software"})
	invalidURL := "1.2.3://"
	config += fmt.Sprintf(`
software:
  packages:
    - url: %s
`, invalidURL)

	path, basePath = createTempFile(t, "", config)
	_, err = GitOpsFromFile(path, basePath, &appConfig, nopLogf)
	assert.ErrorContains(t, err, fmt.Sprintf("%s is not a valid URL", invalidURL))

	// Software URL refers to a .exe but doesn't have (un)install scripts specified
	config = getTeamConfig([]string{"software"})
	exeURL := "https://download-installer.cdn.mozilla.net/pub/firefox/releases/136.0.4/win64/en-US/Firefox%20Setup%20136.0.4.exe?foo=bar"
	config += fmt.Sprintf(`
software:
  packages:
    - url: %s
`, exeURL)

	path, basePath = createTempFile(t, "", config)
	_, err = GitOpsFromFile(path, basePath, &appConfig, nopLogf)
	assert.ErrorContains(t, err, fmt.Sprintf("software URL %s refers to an .exe package, which requires both install_script and uninstall_script", exeURL))

	// Software URL refers to a .tar.gz but doesn't have (un)install scripts specified (URL doesn't exist as Firefox is all .tar.xz)
	config = getTeamConfig([]string{"software"})
	tgzURL := "https://download-installer.cdn.mozilla.net/pub/firefox/releases/137.0.2/linux-x86_64/en-US/firefox-137.0.2.tar.gz?foo=baz"
	config += fmt.Sprintf(`
software:
  packages:
    - url: %s
`, tgzURL)

	path, basePath = createTempFile(t, "", config)
	_, err = GitOpsFromFile(path, basePath, &appConfig, nopLogf)
	assert.ErrorContains(t, err, fmt.Sprintf("software URL %s refers to a .tar.gz archive, which requires both install_script and uninstall_script", tgzURL))

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
	assert.ErrorContains(t, err, "file \"./microsoft-teams.pkg.software.yml\" does not contain a valid software package definition")

	// Policy references a software installer file that has multiple pieces of software specified
	config = getTeamConfig([]string{"policies"})
	config += `
policies:
  - path: ./multipkg.policies.yml
software:
  packages:
    - path: ./multiple-packages.yml
`
	path, basePath = createTempFile(t, "", config)
	err = file.Copy(
		filepath.Join("testdata", "multipkg.policies.yml"),
		filepath.Join(basePath, "multipkg.policies.yml"),
		0o755,
	)
	require.NoError(t, err)
	err = file.Copy(
		filepath.Join("testdata", "software", "multiple-packages.yml"),
		filepath.Join(basePath, "multiple-packages.yml"),
		0o755,
	)
	require.NoError(t, err)
	appConfig = fleet.EnrichedAppConfig{}
	appConfig.License = &fleet.LicenseInfo{
		Tier: fleet.TierPremium,
	}
	_, err = GitOpsFromFile(path, basePath, &appConfig, nopLogf)
	assert.ErrorContains(t, err, "contains multiple packages, so cannot be used as a target for policy automation")
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

func TestSoftwarePackagesUnmarshalMulti(t *testing.T) {
	t.Parallel()
	config := getTeamConfig([]string{"software"})
	config += `
software:
  packages:
    - path: software/single-package.yml
    - path: software/multiple-packages.yml
`

	path, basePath := createTempFile(t, "", config)

	for _, f := range []string{"single-package.yml", "multiple-packages.yml"} {
		err := file.Copy(
			filepath.Join("testdata", "software", f),
			filepath.Join(basePath, "software", f),
			os.FileMode(0o755),
		)
		require.NoError(t, err)
	}

	appConfig := fleet.EnrichedAppConfig{}
	appConfig.License = &fleet.LicenseInfo{
		Tier: fleet.TierPremium,
	}
	_, err := GitOpsFromFile(path, basePath, &appConfig, nopLogf)
	require.NoError(t, err)
}

func TestSoftwarePackagesPathWithInline(t *testing.T) {
	t.Parallel()
	config := getTeamConfig([]string{"software"})
	config += `
software:
  packages:
    - path: software/single-package.yml
      icon:
        path: ./foo/bar.png
`

	path, basePath := createTempFile(t, "", config)

	err := file.Copy(
		filepath.Join("testdata", "software", "single-package.yml"),
		filepath.Join(basePath, "software", "single-package.yml"),
		os.FileMode(0o755),
	)
	require.NoError(t, err)

	appConfig := fleet.EnrichedAppConfig{}
	appConfig.License = &fleet.LicenseInfo{
		Tier: fleet.TierPremium,
	}
	_, err = GitOpsFromFile(path, basePath, &appConfig, nopLogf)
	assert.ErrorContains(t, err, "the software package defined in software/single-package.yml must not have icons, scripts, queries, URL, or hash specified at the team level")
}

func TestScriptOnlyPackagesPathWithInline(t *testing.T) {
	t.Parallel()
	config := getTeamConfig([]string{"software"})
	config += `
software:
  packages:
    - path: software/script-only.sh
      icon:
        path: ./foo/bar.png
`

	path, basePath := createTempFile(t, "", config)

	err := file.Copy(
		filepath.Join("testdata", "software", "script-only.sh"),
		filepath.Join(basePath, "software", "script-only.sh"),
		os.FileMode(0o755),
	)
	require.NoError(t, err)

	appConfig := fleet.EnrichedAppConfig{}
	appConfig.License = &fleet.LicenseInfo{
		Tier: fleet.TierPremium,
	}
	gitops, err := GitOpsFromFile(path, basePath, &appConfig, nopLogf)
	require.NoError(t, err)
	require.Len(t, gitops.Software.Packages, 1)
	assert.Equal(t, filepath.Join(basePath, "foo", "bar.png"), gitops.Software.Packages[0].Icon.Path)
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

func TestInvalidSoftwareInstallerHash(t *testing.T) {
	appConfig := &fleet.EnrichedAppConfig{}
	appConfig.License = &fleet.LicenseInfo{
		Tier: fleet.TierPremium,
	}
	_, err := GitOpsFromFile("testdata/team_config_invalid_sha.yml", "./testdata", appConfig, nopLogf)
	assert.ErrorContains(t, err, "must be a valid lower-case hex-encoded (64-character) SHA-256 hash value")
}

func TestSoftwareDisplayNameValidation(t *testing.T) {
	t.Parallel()
	appConfig := &fleet.EnrichedAppConfig{}
	appConfig.License = &fleet.LicenseInfo{
		Tier: fleet.TierPremium,
	}

	// Create a string with 256 'a' characters (exceeds 255 limit)
	longDisplayName := strings.Repeat("a", 256)

	t.Run("package_display_name_too_long", func(t *testing.T) {
		config := getTeamConfig([]string{"name", "software"})
		// Use hash instead of URL to avoid script validation before display_name validation
		config += `name: Test Team
software:
  packages:
    - hash_sha256: "abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234"
      display_name: "` + longDisplayName + `"
`
		path, basePath := createTempFile(t, "", config)
		_, err := GitOpsFromFile(path, basePath, appConfig, nopLogf)
		assert.ErrorContains(t, err, "display_name is too long (max 255 characters)")
	})

	t.Run("app_store_display_name_too_long", func(t *testing.T) {
		config := getTeamConfig([]string{"name", "software"})
		config += `name: Test Team
software:
  app_store_apps:
    - app_store_id: "12345"
      display_name: "` + longDisplayName + `"
`
		path, basePath := createTempFile(t, "", config)
		_, err := GitOpsFromFile(path, basePath, appConfig, nopLogf)
		assert.ErrorContains(t, err, "display_name is too long (max 255 characters)")
	})

	t.Run("valid_display_name", func(t *testing.T) {
		config := getTeamConfig([]string{"name", "software"})
		// Use hash instead of URL to avoid network calls, and no scripts required
		config += `name: Test Team
software:
  packages:
    - hash_sha256: "abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234abcd1234"
      display_name: "Custom Package Name"
  app_store_apps:
    - app_store_id: "12345"
      display_name: "Custom VPP App Name"
`
		path, basePath := createTempFile(t, "", config)
		result, err := GitOpsFromFile(path, basePath, appConfig, nopLogf)
		require.NoError(t, err)
		require.Len(t, result.Software.Packages, 1)
		assert.Equal(t, "Custom Package Name", result.Software.Packages[0].DisplayName)
		require.Len(t, result.Software.AppStoreApps, 1)
		assert.Equal(t, "Custom VPP App Name", result.Software.AppStoreApps[0].DisplayName)
	})
}

func TestWebhookPolicyIDsValidation(t *testing.T) {
	t.Parallel()

	appConfig := &fleet.EnrichedAppConfig{}
	appConfig.License = &fleet.LicenseInfo{
		Tier: fleet.TierPremium,
	}

	t.Run("no_team_invalid_policy_ids_as_number", func(t *testing.T) {
		config := getTeamConfig([]string{"name", "settings"})
		config += `name: No team
settings:
  webhook_settings:
    failing_policies_webhook:
      enable_failing_policies_webhook: true
      destination_url: https://webhook.site/test
      policy_ids: 567
      host_batch_size: 0
software:
  packages: []
policies: []
`
		noTeamPath, noTeamBasePath := createNamedFileOnTempDir(t, "no-team.yml", config)
		_, err := GitOpsFromFile(noTeamPath, noTeamBasePath, appConfig, nopLogf)
		assert.ErrorContains(t, err, "policy_ids' must be an array")
	})

	t.Run("no_team_invalid_policy_ids_as_string", func(t *testing.T) {
		config := getTeamConfig([]string{"name", "settings"})
		config += `name: No team
settings:
  webhook_settings:
    failing_policies_webhook:
      enable_failing_policies_webhook: true
      destination_url: https://webhook.site/test
      policy_ids: "567"
      host_batch_size: 0
software:
  packages: []
policies: []
`
		noTeamPath, noTeamBasePath := createNamedFileOnTempDir(t, "no-team.yml", config)
		_, err := GitOpsFromFile(noTeamPath, noTeamBasePath, appConfig, nopLogf)
		assert.ErrorContains(t, err, "policy_ids' must be an array")
	})

	t.Run("no_team_valid_policy_ids_as_array", func(t *testing.T) {
		config := getTeamConfig([]string{"name", "settings"})
		config += `name: No team
settings:
  webhook_settings:
    failing_policies_webhook:
      enable_failing_policies_webhook: true
      destination_url: https://webhook.site/test
      policy_ids: [567, 890]
      host_batch_size: 0
software:
  packages: []
policies: []
`
		noTeamPath, noTeamBasePath := createNamedFileOnTempDir(t, "no-team.yml", config)
		gitops, err := GitOpsFromFile(noTeamPath, noTeamBasePath, appConfig, nopLogf)
		assert.NoError(t, err)
		assert.NotNil(t, gitops)
		assert.True(t, gitops.IsNoTeam())
	})

	t.Run("no_team_valid_policy_ids_as_empty_array", func(t *testing.T) {
		config := getTeamConfig([]string{"name", "settings"})
		config += `name: No team
settings:
  webhook_settings:
    failing_policies_webhook:
      enable_failing_policies_webhook: true
      destination_url: https://webhook.site/test
      policy_ids: []
      host_batch_size: 0
software:
  packages: []
policies: []
`
		noTeamPath, noTeamBasePath := createNamedFileOnTempDir(t, "no-team.yml", config)
		gitops, err := GitOpsFromFile(noTeamPath, noTeamBasePath, appConfig, nopLogf)
		assert.NoError(t, err)
		assert.NotNil(t, gitops)
	})

	t.Run("no_team_valid_policy_ids_as_yaml_list", func(t *testing.T) {
		config := getTeamConfig([]string{"name", "settings"})
		config += `name: No team
settings:
  webhook_settings:
    failing_policies_webhook:
      enable_failing_policies_webhook: true
      destination_url: https://webhook.site/test
      policy_ids:
        - 567
        - 890
      host_batch_size: 0
software:
  packages: []
policies: []
`
		noTeamPath, noTeamBasePath := createNamedFileOnTempDir(t, "no-team.yml", config)
		gitops, err := GitOpsFromFile(noTeamPath, noTeamBasePath, appConfig, nopLogf)
		assert.NoError(t, err)
		assert.NotNil(t, gitops)
	})

	t.Run("regular_team_invalid_policy_ids_as_number", func(t *testing.T) {
		config := getTeamConfig([]string{"settings"})
		config += `settings:
  secrets:
    - secret: test123
  webhook_settings:
    failing_policies_webhook:
      enable_failing_policies_webhook: true
      destination_url: https://webhook.site/test
      policy_ids: 567
      host_batch_size: 0
`
		_, err := gitOpsFromString(t, config)
		assert.ErrorContains(t, err, "policy_ids' must be an array")
	})

	t.Run("regular_team_valid_policy_ids_as_array", func(t *testing.T) {
		config := getTeamConfig([]string{"settings"})
		config += `settings:
  secrets:
    - secret: test123
  webhook_settings:
    failing_policies_webhook:
      enable_failing_policies_webhook: true
      destination_url: https://webhook.site/test
      policy_ids: [567, 890]
      host_batch_size: 0
`
		gitops, err := gitOpsFromString(t, config)
		assert.NoError(t, err)
		assert.NotNil(t, gitops)
		assert.NotNil(t, gitops.TeamSettings["webhook_settings"])
	})
}

func TestContainsGlobMeta(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  bool
	}{
		{"./scripts/foo.sh", false},
		{"./scripts/*.sh", true},
		{"./scripts/**/*.sh", true},
		{"./scripts/[abc].sh", true},
		{"./scripts/{a,b}.sh", true},
		{"./scripts/foo?.sh", true},
		{"", false},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, containsGlobMeta(tt.input), "containsGlobMeta(%q)", tt.input)
	}
}

func TestExpandBaseItems(t *testing.T) {
	t.Parallel()

	// requireErrorContains is a helper that asserts at least one error contains substr.
	requireErrorContains := func(t *testing.T, errs []error, substr string) {
		t.Helper()
		require.NotEmpty(t, errs, "expected errors but got none")
		var found bool
		for _, err := range errs {
			if strings.Contains(err.Error(), substr) {
				found = true
				break
			}
		}
		assert.True(t, found, "expected an error containing %q, got: %v", substr, errs)
	}

	t.Run("basic_glob", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "a.yml"), []byte(""), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "b.yml"), []byte(""), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "c.txt"), []byte(""), 0o644))

		items := []fleet.BaseItem{{Paths: ptr.String("*.yml")}} //nolint:modernize
		result, errs := expandBaseItems(items, dir, "test", GlobExpandOptions{})
		require.Empty(t, errs)
		require.Len(t, result, 2)
		assert.Equal(t, filepath.Join(dir, "a.yml"), *result[0].Path)
		assert.Equal(t, filepath.Join(dir, "b.yml"), *result[1].Path)
		assert.Nil(t, result[0].Paths)
		assert.Nil(t, result[1].Paths)
	})

	t.Run("recursive_glob", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		subdir := filepath.Join(dir, "sub")
		require.NoError(t, os.MkdirAll(subdir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "top.yml"), []byte(""), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(subdir, "nested.yml"), []byte(""), 0o644))

		items := []fleet.BaseItem{{Paths: ptr.String("**/*.yml")}} //nolint:modernize
		result, errs := expandBaseItems(items, dir, "test", GlobExpandOptions{})
		require.Empty(t, errs)
		require.Len(t, result, 2)
		assert.Equal(t, filepath.Join(subdir, "nested.yml"), *result[0].Path)
		assert.Equal(t, filepath.Join(dir, "top.yml"), *result[1].Path)
	})

	t.Run("mixed_path_and_paths", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "single.yml"), []byte(""), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "glob1.yaml"), []byte(""), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "glob2.yaml"), []byte(""), 0o644))

		items := []fleet.BaseItem{
			{Path: ptr.String("single.yml")}, //nolint:modernize
			{Paths: ptr.String("*.yaml")},    //nolint:modernize
		}
		result, errs := expandBaseItems(items, dir, "test", GlobExpandOptions{})
		require.Empty(t, errs)
		require.Len(t, result, 3)
		assert.Equal(t, filepath.Join(dir, "single.yml"), *result[0].Path)
		assert.Equal(t, filepath.Join(dir, "glob1.yaml"), *result[1].Path)
		assert.Equal(t, filepath.Join(dir, "glob2.yaml"), *result[2].Path)
	})

	t.Run("paths_without_glob_error", func(t *testing.T) {
		t.Parallel()
		items := []fleet.BaseItem{{Paths: ptr.String("foo.yml")}} //nolint:modernize
		_, errs := expandBaseItems(items, "/tmp", "test", GlobExpandOptions{})
		requireErrorContains(t, errs, `does not contain glob characters`)
	})

	t.Run("path_with_glob_error", func(t *testing.T) {
		t.Parallel()
		items := []fleet.BaseItem{{Path: ptr.String("*.yml")}} //nolint:modernize
		_, errs := expandBaseItems(items, "/tmp", "test", GlobExpandOptions{})
		requireErrorContains(t, errs, `contains glob characters`)
	})

	t.Run("both_path_and_paths_error", func(t *testing.T) {
		t.Parallel()
		items := []fleet.BaseItem{{Path: ptr.String("foo.yml"), Paths: ptr.String("*.yml")}} //nolint:modernize
		_, errs := expandBaseItems(items, "/tmp", "test", GlobExpandOptions{})
		requireErrorContains(t, errs, `cannot have both "path" and "paths"`)
	})

	t.Run("inline_items_passed_through", func(t *testing.T) {
		t.Parallel()
		items := []fleet.BaseItem{{}}
		result, errs := expandBaseItems(items, "/tmp", "test", GlobExpandOptions{})
		require.Empty(t, errs)
		require.Len(t, result, 1)
		assert.Nil(t, result[0].Path)
		assert.Nil(t, result[0].Paths)
	})

	t.Run("require_file_reference_error", func(t *testing.T) {
		t.Parallel()
		items := []fleet.BaseItem{{}}
		_, errs := expandBaseItems(items, "/tmp", "test", GlobExpandOptions{
			RequireFileReference: true,
		})
		requireErrorContains(t, errs, `no "path" or "paths" field`)
	})

	t.Run("no_matches_warning", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		var warnings []string
		logFn := func(format string, args ...any) {
			warnings = append(warnings, fmt.Sprintf(format, args...))
		}
		items := []fleet.BaseItem{{Paths: ptr.String("*.yml")}} //nolint:modernize
		result, errs := expandBaseItems(items, dir, "test", GlobExpandOptions{LogFn: logFn})
		require.Empty(t, errs)
		assert.Empty(t, result)
		require.Len(t, warnings, 1)
		assert.Contains(t, warnings[0], "matched no test")
	})

	t.Run("duplicate_basenames_error", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		sub1 := filepath.Join(dir, "sub1")
		sub2 := filepath.Join(dir, "sub2")
		require.NoError(t, os.MkdirAll(sub1, 0o755))
		require.NoError(t, os.MkdirAll(sub2, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(sub1, "dup.yml"), []byte(""), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(sub2, "dup.yml"), []byte(""), 0o644))

		items := []fleet.BaseItem{{Paths: ptr.String("**/*.yml")}} //nolint:modernize
		_, errs := expandBaseItems(items, dir, "test", GlobExpandOptions{
			RequireUniqueBasenames: true,
		})
		requireErrorContains(t, errs, "duplicate test basename")
	})

	t.Run("duplicate_basenames_across_items_error", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		sub := filepath.Join(dir, "sub")
		require.NoError(t, os.MkdirAll(sub, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "item.yml"), []byte(""), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(sub, "item.yml"), []byte(""), 0o644))

		items := []fleet.BaseItem{
			{Path: ptr.String("item.yml")},   //nolint:modernize
			{Paths: ptr.String("sub/*.yml")}, //nolint:modernize
		}
		_, errs := expandBaseItems(items, dir, "test", GlobExpandOptions{
			RequireUniqueBasenames: true,
		})
		requireErrorContains(t, errs, `duplicate test basename "item.yml"`)
		requireErrorContains(t, errs, `sub/*.yml`)
	})

	t.Run("allowed_extensions_filter", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "good.sh"), []byte("#!/bin/bash"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "bad.txt"), []byte("text"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "bad.py"), []byte("python"), 0o644))

		var warnings []string
		logFn := func(format string, args ...any) {
			warnings = append(warnings, fmt.Sprintf(format, args...))
		}

		items := []fleet.BaseItem{{Paths: ptr.String("*")}} //nolint:modernize
		result, errs := expandBaseItems(items, dir, "test", GlobExpandOptions{
			AllowedExtensions: map[string]bool{".sh": true},
			LogFn:             logFn,
		})
		require.Empty(t, errs)
		require.Len(t, result, 1)
		assert.Equal(t, filepath.Join(dir, "good.sh"), *result[0].Path)
		assert.Len(t, warnings, 2)
	})

	t.Run("results_sorted", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "z.yml"), []byte(""), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "a.yml"), []byte(""), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "m.yml"), []byte(""), 0o644))

		items := []fleet.BaseItem{{Paths: ptr.String("*.yml")}} //nolint:modernize
		result, errs := expandBaseItems(items, dir, "test", GlobExpandOptions{})
		require.Empty(t, errs)
		require.Len(t, result, 3)
		assert.Equal(t, filepath.Join(dir, "a.yml"), *result[0].Path)
		assert.Equal(t, filepath.Join(dir, "m.yml"), *result[1].Path)
		assert.Equal(t, filepath.Join(dir, "z.yml"), *result[2].Path)
	})

	t.Run("multiple_errors_collected", func(t *testing.T) {
		t.Parallel()
		items := []fleet.BaseItem{{Path: ptr.String("*.yml")}, {Paths: ptr.String("noglob.yml")}} //nolint:modernize
		_, errs := expandBaseItems(items, "", "test", GlobExpandOptions{})
		require.Len(t, errs, 2)
		assert.Contains(t, errs[0].Error(), `contains glob characters`)
		assert.Contains(t, errs[1].Error(), `does not contain glob characters`)
	})
}

func TestResolveScriptPaths(t *testing.T) {
	t.Parallel()

	t.Run("path_resolves", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "script.sh"), []byte("#!/bin/bash"), 0o644))

		items := []fleet.BaseItem{{Path: ptr.String("script.sh")}} //nolint:modernize
		result, errs := resolveScriptPaths(items, dir, nopLogf)
		require.Empty(t, errs)
		require.Len(t, result, 1)
		assert.Equal(t, filepath.Join(dir, "script.sh"), *result[0].Path)
	})

	t.Run("glob_expands", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "a.sh"), []byte("#!/bin/bash"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "b.sh"), []byte("#!/bin/bash"), 0o644))

		items := []fleet.BaseItem{{Paths: ptr.String("*.sh")}} //nolint:modernize
		result, errs := resolveScriptPaths(items, dir, nopLogf)
		require.Empty(t, errs)
		require.Len(t, result, 2)
	})

	t.Run("glob_filters_non_script_extensions", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "a.sh"), []byte("#!/bin/bash"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "b.ps1"), []byte("Write-Host"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "c.py"), []byte("#!/usr/bin/env python3"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "d.txt"), []byte("not a script"), 0o644))

		items := []fleet.BaseItem{{Paths: ptr.String("*")}} //nolint:modernize
		result, errs := resolveScriptPaths(items, dir, nopLogf)
		require.Empty(t, errs)
		require.Len(t, result, 3)
		got := make([]string, 0, len(result))
		for _, r := range result {
			got = append(got, filepath.Base(*r.Path))
		}
		assert.Equal(t, []string{"a.sh", "b.ps1", "c.py"}, got)
	})

	t.Run("inline_not_allowed", func(t *testing.T) {
		t.Parallel()
		items := []fleet.BaseItem{{}}
		_, errs := resolveScriptPaths(items, "/tmp", nopLogf)
		require.NotEmpty(t, errs)
		assert.Contains(t, errs[0].Error(), `no "path" or "paths" field`)
	})
}

func TestParseLabelsGlob(t *testing.T) {
	t.Parallel()

	t.Run("inline_and_path", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()

		// Write a label file referenced by path.
		labelFile := filepath.Join(dir, "labels", "from-file.yml")
		require.NoError(t, os.MkdirAll(filepath.Dir(labelFile), 0o755))
		require.NoError(t, os.WriteFile(labelFile, []byte("- name: FileLabel\n  label_membership_type: manual\n"), 0o644))

		top := yamlToRawJSON(t, `
labels:
  - name: InlineLabel
    label_membership_type: manual
  - path: labels/from-file.yml
`)
		result := &GitOps{}
		multiErr := parseLabels(top, result, dir, nopLogf, "test.yml", nil)
		require.Nil(t, multiErr.ErrorOrNil())
		require.Len(t, result.Labels, 2)
		assert.Equal(t, "InlineLabel", result.Labels[0].Name)
		assert.Equal(t, "FileLabel", result.Labels[1].Name)
	})

	t.Run("glob_expands", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()

		labelsDir := filepath.Join(dir, "labels")
		require.NoError(t, os.MkdirAll(labelsDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(labelsDir, "a.yml"), []byte("- name: LabelA\n  label_membership_type: manual\n"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(labelsDir, "b.yml"), []byte("- name: LabelB\n  label_membership_type: manual\n"), 0o644))

		top := yamlToRawJSON(t, `
labels:
  - paths: "labels/*.yml"
`)
		result := &GitOps{}
		multiErr := parseLabels(top, result, dir, nopLogf, "test.yml", nil)
		require.Nil(t, multiErr.ErrorOrNil())
		require.Len(t, result.Labels, 2)
		assert.Equal(t, "LabelA", result.Labels[0].Name)
		assert.Equal(t, "LabelB", result.Labels[1].Name)
	})
}

func TestParsePoliciesGlob(t *testing.T) {
	t.Parallel()

	t.Run("inline_and_path", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()

		policyFile := filepath.Join(dir, "policies", "from-file.yml")
		require.NoError(t, os.MkdirAll(filepath.Dir(policyFile), 0o755))
		require.NoError(t, os.WriteFile(policyFile, []byte("- name: FilePolicy\n  query: SELECT 1;\n"), 0o644))

		top := yamlToRawJSON(t, `
policies:
  - name: InlinePolicy
    query: SELECT 1;
  - path: policies/from-file.yml
`)
		result := &GitOps{}
		multiErr := parsePolicies(top, result, dir, nopLogf, "test.yml", nil)
		require.Nil(t, multiErr.ErrorOrNil())
		require.Len(t, result.Policies, 2)
		assert.Equal(t, "InlinePolicy", result.Policies[0].Name)
		assert.Equal(t, "FilePolicy", result.Policies[1].Name)
	})

	t.Run("glob_expands", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()

		policiesDir := filepath.Join(dir, "policies")
		require.NoError(t, os.MkdirAll(policiesDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(policiesDir, "a.yml"), []byte("- name: PolicyA\n  query: SELECT 1;\n"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(policiesDir, "b.yml"), []byte("- name: PolicyB\n  query: SELECT 1;\n"), 0o644))

		top := yamlToRawJSON(t, `
policies:
  - paths: "policies/*.yml"
`)
		result := &GitOps{}
		multiErr := parsePolicies(top, result, dir, nopLogf, "test.yml", nil)
		require.Nil(t, multiErr.ErrorOrNil())
		require.Len(t, result.Policies, 2)
		assert.Equal(t, "PolicyA", result.Policies[0].Name)
		assert.Equal(t, "PolicyB", result.Policies[1].Name)
	})
}

func TestParseReportsGlob(t *testing.T) {
	t.Parallel()

	t.Run("inline_and_path", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()

		reportFile := filepath.Join(dir, "reports", "from-file.yml")
		require.NoError(t, os.MkdirAll(filepath.Dir(reportFile), 0o755))
		require.NoError(t, os.WriteFile(reportFile, []byte("- name: FileReport\n  query: SELECT 1;\n"), 0o644))

		top := yamlToRawJSON(t, `
reports:
  - name: InlineReport
    query: SELECT 1;
  - path: reports/from-file.yml
`)
		teamName := "TestTeam"
		result := &GitOps{TeamName: &teamName}
		multiErr := parseReports(top, result, dir, nopLogf, "test.yml", nil)
		require.Nil(t, multiErr.ErrorOrNil())
		require.Len(t, result.Queries, 2)
		assert.Equal(t, "InlineReport", result.Queries[0].Name)
		assert.Equal(t, "FileReport", result.Queries[1].Name)
	})

	t.Run("glob_expands", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()

		reportsDir := filepath.Join(dir, "reports")
		require.NoError(t, os.MkdirAll(reportsDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(reportsDir, "a.yml"), []byte("- name: ReportA\n  query: SELECT 1;\n"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(reportsDir, "b.yml"), []byte("- name: ReportB\n  query: SELECT 1;\n"), 0o644))

		top := yamlToRawJSON(t, `
reports:
  - paths: "reports/*.yml"
`)
		teamName := "TestTeam"
		result := &GitOps{TeamName: &teamName}
		multiErr := parseReports(top, result, dir, nopLogf, "test.yml", nil)
		require.Nil(t, multiErr.ErrorOrNil())
		require.Len(t, result.Queries, 2)
		assert.Equal(t, "ReportA", result.Queries[0].Name)
		assert.Equal(t, "ReportB", result.Queries[1].Name)
	})
}

func TestGitOpsGlobScripts(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	scriptsDir := filepath.Join(dir, "scripts")
	require.NoError(t, os.MkdirAll(scriptsDir, 0o755))
	scriptsSubDir := filepath.Join(scriptsDir, "sub")
	require.NoError(t, os.MkdirAll(scriptsSubDir, 0o755))

	// Create script files
	require.NoError(t, os.WriteFile(filepath.Join(scriptsDir, "alpha.sh"), []byte("#!/bin/bash\necho alpha"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(scriptsDir, "beta.sh"), []byte("#!/bin/bash\necho beta"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(scriptsDir, "gamma.ps1"), []byte("Write-Host gamma"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(scriptsSubDir, "delta.sh"), []byte("nada"), 0o644))

	// Write a gitops YAML file that uses paths: glob
	config := getGlobalConfig([]string{"controls"})
	config += `controls:
  scripts:
    - paths: scripts/*.sh
    - path: scripts/gamma.ps1
`
	yamlPath := filepath.Join(dir, "gitops.yml")
	require.NoError(t, os.WriteFile(yamlPath, []byte(config), 0o644))

	result, err := GitOpsFromFile(yamlPath, dir, nil, nopLogf)
	require.NoError(t, err)
	require.Len(t, result.Controls.Scripts, 3)

	// Glob results come first (sorted), then the explicit path
	assert.Equal(t, filepath.Join(scriptsDir, "alpha.sh"), *result.Controls.Scripts[0].Path)
	assert.Equal(t, filepath.Join(scriptsDir, "beta.sh"), *result.Controls.Scripts[1].Path)
	assert.Equal(t, filepath.Join(scriptsDir, "gamma.ps1"), *result.Controls.Scripts[2].Path)
}

func TestGitOpsGlobProfiles(t *testing.T) {
	t.Parallel()

	t.Run("macos_profiles", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		profilesDir := filepath.Join(dir, "profiles")
		require.NoError(t, os.MkdirAll(profilesDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(profilesDir, "alpha.mobileconfig"), []byte("<plist></plist>"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(profilesDir, "beta.json"), []byte("{}"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(profilesDir, "gamma.mobileconfig"), []byte("<plist></plist>"), 0o644))

		config := getGlobalConfig([]string{"controls"})
		config += `controls:
  apple_settings:
    configuration_profiles:
      - paths: profiles/*.mobileconfig
      - path: profiles/beta.json
`
		yamlPath := filepath.Join(dir, "gitops.yml")
		require.NoError(t, os.WriteFile(yamlPath, []byte(config), 0o644))

		result, err := GitOpsFromFile(yamlPath, dir, nil, nopLogf)
		require.NoError(t, err)
		macSettings, ok := result.Controls.MacOSSettings.(fleet.MacOSSettings)
		require.True(t, ok)
		require.Len(t, macSettings.CustomSettings, 3)

		// Glob results come first (sorted), then the explicit path
		assert.Contains(t, macSettings.CustomSettings[0].Path, "alpha.mobileconfig")
		assert.Contains(t, macSettings.CustomSettings[1].Path, "gamma.mobileconfig")
		assert.Contains(t, macSettings.CustomSettings[2].Path, "beta.json")
	})

	t.Run("windows_profiles", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		profilesDir := filepath.Join(dir, "profiles")
		require.NoError(t, os.MkdirAll(profilesDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(profilesDir, "alpha.xml"), []byte("<xml/>"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(profilesDir, "beta.xml"), []byte("<xml/>"), 0o644))

		config := getGlobalConfig([]string{"controls"})
		config += `controls:
  windows_settings:
    configuration_profiles:
      - paths: profiles/*.xml
`
		yamlPath := filepath.Join(dir, "gitops.yml")
		require.NoError(t, os.WriteFile(yamlPath, []byte(config), 0o644))

		result, err := GitOpsFromFile(yamlPath, dir, nil, nopLogf)
		require.NoError(t, err)
		winSettings, ok := result.Controls.WindowsSettings.(fleet.WindowsSettings)
		require.True(t, ok)
		require.True(t, winSettings.CustomSettings.Valid)
		require.Len(t, winSettings.CustomSettings.Value, 2)

		assert.Contains(t, winSettings.CustomSettings.Value[0].Path, "alpha.xml")
		assert.Contains(t, winSettings.CustomSettings.Value[1].Path, "beta.xml")
	})

	t.Run("android_profiles", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		profilesDir := filepath.Join(dir, "profiles")
		require.NoError(t, os.MkdirAll(profilesDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(profilesDir, "skip.xml"), []byte("<xml/>"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(profilesDir, "beta.json"), []byte("{}"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(profilesDir, "skip.txt"), []byte("nope"), 0o644))

		config := getGlobalConfig([]string{"controls"})
		config += `controls:
  android_settings:
    configuration_profiles:
      - paths: profiles/*
`
		yamlPath := filepath.Join(dir, "gitops.yml")
		require.NoError(t, os.WriteFile(yamlPath, []byte(config), 0o644))

		result, err := GitOpsFromFile(yamlPath, dir, nil, nopLogf)
		require.NoError(t, err)
		androidSettings, ok := result.Controls.AndroidSettings.(fleet.AndroidSettings)
		require.True(t, ok)
		require.True(t, androidSettings.CustomSettings.Valid)
		require.Len(t, androidSettings.CustomSettings.Value, 1)

		// Sorted alphabetically by path
		assert.Contains(t, androidSettings.CustomSettings.Value[0].Path, "beta.json")
	})

	t.Run("macos_profiles_with_labels", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		profilesDir := filepath.Join(dir, "profiles")
		require.NoError(t, os.MkdirAll(profilesDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(profilesDir, "a.mobileconfig"), []byte("<plist></plist>"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(profilesDir, "b.mobileconfig"), []byte("<plist></plist>"), 0o644))

		config := getGlobalConfig([]string{"controls"})
		config += `controls:
  apple_settings:
    configuration_profiles:
      - paths: profiles/*.mobileconfig
        labels_include_all:
          - MyLabel
`
		yamlPath := filepath.Join(dir, "gitops.yml")
		require.NoError(t, os.WriteFile(yamlPath, []byte(config), 0o644))

		result, err := GitOpsFromFile(yamlPath, dir, nil, nopLogf)
		require.NoError(t, err)
		macSettings, ok := result.Controls.MacOSSettings.(fleet.MacOSSettings)
		require.True(t, ok)
		require.Len(t, macSettings.CustomSettings, 2)
		for _, p := range macSettings.CustomSettings {
			assert.Equal(t, []string{"MyLabel"}, p.LabelsIncludeAll)
		}
	})
}

func TestUnknownKeyDetection(t *testing.T) {
	t.Parallel()

	t.Run("unknown key in controls", func(t *testing.T) {
		t.Parallel()
		config := `
name: TeamName
settings:
  secrets:
agent_options:
controls:
  macos_updates:
    minimum_version: "14.0"
    deadline: "2024-01-01"
  unknown_control_field: true
reports:
policies:
software:
`
		path, basePath := createTempFile(t, "", config)
		_, err := GitOpsFromFile(path, basePath, premiumAppConfig(), nopLogf)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown_control_field")
	})

	t.Run("unknown key in controls macos_updates (any-field)", func(t *testing.T) {
		t.Parallel()
		config := `
name: TeamName
settings:
  secrets:
agent_options:
controls:
  macos_updates:
    minimum_version: "14.0"
    deadlinee: "2024-01-01"
reports:
policies:
software:
`
		path, basePath := createTempFile(t, "", config)
		_, err := GitOpsFromFile(path, basePath, premiumAppConfig(), nopLogf)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "deadlinee")
		assert.Contains(t, err.Error(), `did you mean "deadline"?`)
		assert.Contains(t, err.Error(), "controls.macos_updates")
	})

	t.Run("unknown key in query entry", func(t *testing.T) {
		t.Parallel()
		config := `
name: TeamName
settings:
  secrets:
agent_options:
controls:
reports:
  - name: test_query
    query: SELECT 1;
    unknown_query_field: true
policies:
software:
`
		path, basePath := createTempFile(t, "", config)
		_, err := GitOpsFromFile(path, basePath, premiumAppConfig(), nopLogf)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown_query_field")
	})

	t.Run("unknown key in policy entry", func(t *testing.T) {
		t.Parallel()
		config := `
name: TeamName
settings:
  secrets:
agent_options:
controls:
reports:
policies:
  - name: test_policy
    query: SELECT 1;
    unknown_policy_field: true
software:
`
		path, basePath := createTempFile(t, "", config)
		_, err := GitOpsFromFile(path, basePath, premiumAppConfig(), nopLogf)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown_policy_field")
	})

	t.Run("unknown key in label entry", func(t *testing.T) {
		t.Parallel()
		config := `
name: TeamName
settings:
  secrets:
agent_options:
controls:
labels:
  - name: test_label
    query: SELECT 1
    label_membership_type: dynamic
    unknown_label_field: true
reports:
policies:
software:
`
		path, basePath := createTempFile(t, "", config)
		_, err := GitOpsFromFile(path, basePath, premiumAppConfig(), nopLogf)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown_label_field")
	})

	t.Run("unknown key in software section", func(t *testing.T) {
		t.Parallel()
		config := `
name: TeamName
settings:
  secrets:
agent_options:
controls:
reports:
policies:
software:
  unknown_software_field: true
`
		path, basePath := createTempFile(t, "", config)
		_, err := GitOpsFromFile(path, basePath, premiumAppConfig(), nopLogf)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown_software_field")
	})

	t.Run("multiple unknown keys reported at once", func(t *testing.T) {
		t.Parallel()
		config := `
name: TeamName
settings:
  secrets:
agent_options:
controls:
  bad_control_key: true
reports:
  - name: test_query
    query: SELECT 1;
    bad_query_key: true
policies:
  - name: test_policy
    query: SELECT 1;
    bad_policy_key: true
software:
`
		path, basePath := createTempFile(t, "", config)
		_, err := GitOpsFromFile(path, basePath, premiumAppConfig(), nopLogf)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "bad_control_key")
		assert.Contains(t, err.Error(), "bad_query_key")
		assert.Contains(t, err.Error(), "bad_policy_key")
	})

	t.Run("multiple unknown keys within a single section", func(t *testing.T) {
		t.Parallel()
		config := `
name: TeamName
settings:
  secrets:
agent_options:
controls:
  macos_updates:
    minimum_version: "14.0"
    deadlinee: "2024-01-01"
    update_new_hostss: true
  bad_control_key: true
reports:
policies:
software:
`
		path, basePath := createTempFile(t, "", config)
		_, err := GitOpsFromFile(path, basePath, premiumAppConfig(), nopLogf)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "deadlinee")
		assert.Contains(t, err.Error(), "update_new_hostss")
		assert.Contains(t, err.Error(), "bad_control_key")
	})

	t.Run("valid config no unknown key errors", func(t *testing.T) {
		t.Parallel()
		config := `
name: TeamName
settings:
  secrets:
agent_options:
controls:
  macos_updates:
    minimum_version: "14.0"
    deadline: "2024-01-01"
reports:
  - name: test_query
    query: SELECT 1;
    interval: 3600
policies:
  - name: test_policy
    query: SELECT 1;
    description: A test policy
software:
`
		path, basePath := createTempFile(t, "", config)
		_, err := GitOpsFromFile(path, basePath, premiumAppConfig(), nopLogf)
		require.NoError(t, err)
	})

	t.Run("allow-unknown-keys option logs instead of erroring", func(t *testing.T) {
		t.Parallel()
		config := `
name: TeamName
settings:
  secrets:
agent_options:
controls:
  unknown_control_field: true
reports:
policies:
software:
`
		path, basePath := createTempFile(t, "", config)
		var logMessages []string
		sawExpectErrorMsg := false
		logFn := func(format string, a ...any) {
			msg := fmt.Sprintf(format, a...)
			if strings.Contains(msg, "unknown_control_field") {
				sawExpectErrorMsg = true
			}
			logMessages = append(logMessages, msg)
		}
		_, err := GitOpsFromFile(path, basePath, premiumAppConfig(), logFn, GitOpsOptions{AllowUnknownKeys: true})
		require.NoError(t, err)
		// Should have logged a warning about the unknown key
		require.NotEmpty(t, logMessages)
		assert.True(t, sawExpectErrorMsg, "expected warning about unknown_control_field in log messages: %v", logMessages)
	})

	t.Run("unknown key in controls on no-team path", func(t *testing.T) {
		t.Parallel()
		config := `
name: No team
controls:
  unknown_control_field: true
policies:
`
		path, basePath := createNamedFileOnTempDir(t, "no-team.yml", config)
		_, err := GitOpsFromFile(path, basePath, nil, nopLogf)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown_control_field")
	})

	t.Run("unknown key in software package via path", func(t *testing.T) {
		t.Parallel()
		config := getTeamConfig([]string{"software"})
		config += `
software:
  packages:
    - path: pkg.yml
`
		path, basePath := createTempFile(t, "", config)
		pkgYAML := `
url: https://example.com/pkg.pkg
hash_sha256: aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
unknown_pkg_field: bad
`
		require.NoError(t, os.WriteFile(filepath.Join(basePath, "pkg.yml"), []byte(pkgYAML), 0o644))
		_, err := GitOpsFromFile(path, basePath, premiumAppConfig(), nopLogf)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown_pkg_field")
	})

	t.Run("unknown key in software package array via path", func(t *testing.T) {
		t.Parallel()
		config := getTeamConfig([]string{"software"})
		config += `
software:
  packages:
    - path: pkgs.yml
`
		path, basePath := createTempFile(t, "", config)
		pkgYAML := `
- url: https://example.com/pkg.pkg
  hash_sha256: aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
  unknown_array_field: bad
`
		require.NoError(t, os.WriteFile(filepath.Join(basePath, "pkgs.yml"), []byte(pkgYAML), 0o644))
		_, err := GitOpsFromFile(path, basePath, premiumAppConfig(), nopLogf)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown_array_field")
	})

	t.Run("unknown key in org_settings", func(t *testing.T) {
		t.Parallel()
		config := `
org_settings:
  server_settings:
    server_url: https://fleet.example.com
  org_info:
    contact_url: https://example.com/contact
    org_name: Test Org
  unknown_org_field: true
  secrets:
controls:
agent_options:
reports:
policies:
`
		path, basePath := createTempFile(t, "", config)
		_, err := GitOpsFromFile(path, basePath, nil, nopLogf)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown_org_field")
	})

	t.Run("unknown nested key in org_settings", func(t *testing.T) {
		t.Parallel()
		config := `
org_settings:
  server_settings:
    server_url: https://fleet.example.com
    unknown_server_field: true
  org_info:
    contact_url: https://example.com/contact
    org_name: Test Org
  secrets:
controls:
agent_options:
reports:
policies:
`
		path, basePath := createTempFile(t, "", config)
		_, err := GitOpsFromFile(path, basePath, nil, nopLogf)
		require.Error(t, err)
		assert.Contains(t, err.Error(), `unknown key "org_settings.server_settings.unknown_server_field"`)
	})

	t.Run("unknown key in org_settings with typo suggestion", func(t *testing.T) {
		t.Parallel()
		config := `
org_settings:
  server_settigns:
    server_url: https://fleet.example.com
  org_info:
    contact_url: https://example.com/contact
    org_name: Test Org
  secrets:
controls:
agent_options:
reports:
policies:
`
		path, basePath := createTempFile(t, "", config)
		_, err := GitOpsFromFile(path, basePath, nil, nopLogf)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "org_settings.server_settigns")
		assert.Contains(t, err.Error(), `did you mean "server_settings"?`)
	})

	t.Run("unknown key in fleet settings", func(t *testing.T) {
		t.Parallel()
		config := `
name: FleetName
settings:
  secrets:
  unknown_fleet_field: true
agent_options:
controls:
reports:
policies:
software:
`
		path, basePath := createTempFile(t, "", config)
		_, err := GitOpsFromFile(path, basePath, premiumAppConfig(), nopLogf)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown_fleet_field")
	})

	t.Run("unknown nested key in fleet settings webhook_settings", func(t *testing.T) {
		t.Parallel()
		config := `
name: FleetName
settings:
  secrets:
  webhook_settings:
    unknown_webhook_field: true
agent_options:
controls:
reports:
policies:
software:
`
		path, basePath := createTempFile(t, "", config)
		_, err := GitOpsFromFile(path, basePath, premiumAppConfig(), nopLogf)
		require.Error(t, err)
		assert.Contains(t, err.Error(), `unknown key "settings.webhook_settings.unknown_webhook_field"`)
	})

	t.Run("unknown key in org_settings via path", func(t *testing.T) {
		t.Parallel()
		config := `
org_settings:
  path: org_settings.yml
controls:
agent_options:
reports:
policies:
`
		path, basePath := createTempFile(t, "", config)
		orgSettingsYAML := `
server_settings:
  server_url: https://fleet.example.com
org_info:
  contact_url: https://example.com/contact
  org_name: Test Org
unknown_org_path_field: true
secrets:
`
		require.NoError(t, os.WriteFile(filepath.Join(basePath, "org_settings.yml"), []byte(orgSettingsYAML), 0o644))
		_, err := GitOpsFromFile(path, basePath, nil, nopLogf)
		require.Error(t, err)
		assert.Contains(t, err.Error(), `unknown key "org_settings.unknown_org_path_field" in "org_settings.yml"`)
	})

	t.Run("unknown key in fleet settings via path", func(t *testing.T) {
		t.Parallel()
		config := `
name: FleetName
settings:
  path: fleet_settings.yml
agent_options:
controls:
reports:
policies:
software:
`
		path, basePath := createTempFile(t, "", config)
		fleetSettingsYAML := `
secrets:
unknown_fleet_path_field: true
`
		require.NoError(t, os.WriteFile(filepath.Join(basePath, "fleet_settings.yml"), []byte(fleetSettingsYAML), 0o644))
		_, err := GitOpsFromFile(path, basePath, premiumAppConfig(), nopLogf)
		require.Error(t, err)
		assert.Contains(t, err.Error(), `unknown key "settings.unknown_fleet_path_field" in "fleet_settings.yml"`)
	})

	t.Run("unknown key in policy install_software package_path", func(t *testing.T) {
		t.Parallel()
		config := getTeamConfig([]string{"policies", "software"})
		config += `
software:
  packages:
    - url: https://example.com/pkg.pkg
      hash_sha256: aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
policies:
  - name: Test policy
    query: SELECT 1;
    install_software:
      package_path: pkg.yml
`
		path, basePath := createTempFile(t, "", config)
		pkgYAML := `
url: https://example.com/pkg.pkg
hash_sha256: aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
unknown_policy_pkg_field: bad
`
		require.NoError(t, os.WriteFile(filepath.Join(basePath, "pkg.yml"), []byte(pkgYAML), 0o644))
		_, err := GitOpsFromFile(path, basePath, premiumAppConfig(), nopLogf)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown_policy_pkg_field")
	})
}

// TestControlsNewKeyNames verifies that the new multi-platform key names
// (apple_settings, setup_experience, configuration_profiles, apple_setup_assistant,
// macos_bootstrap_package, apple_enable_release_device_manually, macos_script, macos_manual_agent_install)
// are accepted in controls parsing and produce the same result as the old names.
func TestControlsNewKeyNames(t *testing.T) {
	t.Parallel()

	// Test with inline controls using new key names
	t.Run("inline_new_names", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		profileDir := filepath.Join(dir, "lib")
		require.NoError(t, os.Mkdir(profileDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(profileDir, "macos-password.mobileconfig"), []byte("<plist></plist>"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(profileDir, "windows-screenlock.xml"), []byte("<xml/>"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(profileDir, "collect-fleetd-logs.sh"), []byte("#!/bin/bash"), 0o644))

		config := `
controls:
  apple_settings:
    configuration_profiles:
      - path: ./lib/macos-password.mobileconfig
  windows_settings:
    configuration_profiles:
      - path: ./lib/windows-screenlock.xml
  scripts:
    - path: ./lib/collect-fleetd-logs.sh
  enable_disk_encryption: true
  setup_experience:
    macos_bootstrap_package: null
    enable_end_user_authentication: false
    apple_setup_assistant: null
    apple_enable_release_device_manually: null
    macos_manual_agent_install: null
  macos_updates:
    deadline: null
    minimum_version: null
  windows_enabled_and_configured: true
reports:
policies:
agent_options:
org_settings:
  server_settings:
    server_url: https://fleet.example.com
  org_info:
    contact_url: https://example.com/contact
    org_logo_url: ""
    org_logo_url_light_background: ""
    org_name: Test Org
  secrets:
`
		yamlPath := filepath.Join(dir, "gitops.yml")
		require.NoError(t, os.WriteFile(yamlPath, []byte(config), 0o644))

		gitops, err := GitOpsFromFile(yamlPath, dir, nil, nopLogf)
		require.NoError(t, err)

		// Verify controls parsed correctly with new key names
		macSettings, ok := gitops.Controls.MacOSSettings.(fleet.MacOSSettings)
		require.True(t, ok, "macos_settings (via apple_settings) not parsed")
		require.Len(t, macSettings.CustomSettings, 1)

		winSettings, ok := gitops.Controls.WindowsSettings.(fleet.WindowsSettings)
		require.True(t, ok, "windows_settings not parsed")
		require.True(t, winSettings.CustomSettings.Valid)
		require.Len(t, winSettings.CustomSettings.Value, 1)

		require.NotNil(t, gitops.Controls.MacOSSetup, "macos_setup (via setup_experience) not parsed")

		diskEnc, ok := gitops.Controls.EnableDiskEncryption.(bool)
		require.True(t, ok)
		require.True(t, diskEnc)
	})

	// Test with external controls file using new key names
	t.Run("external_file_new_names", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		profileDir := filepath.Join(dir, "lib")
		require.NoError(t, os.Mkdir(profileDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(profileDir, "macos-password.mobileconfig"), []byte("<plist></plist>"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(profileDir, "windows-screenlock.xml"), []byte("<xml/>"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(profileDir, "collect-fleetd-logs.sh"), []byte("#!/bin/bash"), 0o644))

		controlsYAML := `
apple_settings:
  configuration_profiles:
    - path: ./lib/macos-password.mobileconfig
windows_settings:
  configuration_profiles:
    - path: ./lib/windows-screenlock.xml
scripts:
  - path: ./lib/collect-fleetd-logs.sh
enable_disk_encryption: true
setup_experience:
  macos_bootstrap_package: null
  enable_end_user_authentication: false
  apple_setup_assistant: null
  apple_enable_release_device_manually: null
  macos_manual_agent_install: null
macos_updates:
  deadline: null
  minimum_version: null
windows_enabled_and_configured: true
`
		controlsPath := filepath.Join(dir, "controls.yml")
		require.NoError(t, os.WriteFile(controlsPath, []byte(controlsYAML), 0o644))

		config := `
controls:
  path: ./controls.yml
reports:
policies:
agent_options:
org_settings:
  server_settings:
    server_url: https://fleet.example.com
  org_info:
    contact_url: https://example.com/contact
    org_logo_url: ""
    org_logo_url_light_background: ""
    org_name: Test Org
  secrets:
`
		yamlPath := filepath.Join(dir, "gitops.yml")
		require.NoError(t, os.WriteFile(yamlPath, []byte(config), 0o644))

		gitops, err := GitOpsFromFile(yamlPath, dir, nil, nopLogf)
		require.NoError(t, err)

		// Verify controls parsed correctly from external file with new key names
		macSettings, ok := gitops.Controls.MacOSSettings.(fleet.MacOSSettings)
		require.True(t, ok, "macos_settings (via apple_settings in external file) not parsed")
		require.Len(t, macSettings.CustomSettings, 1)

		winSettings, ok := gitops.Controls.WindowsSettings.(fleet.WindowsSettings)
		require.True(t, ok, "windows_settings not parsed")
		require.True(t, winSettings.CustomSettings.Valid)
		require.Len(t, winSettings.CustomSettings.Value, 1)

		require.NotNil(t, gitops.Controls.MacOSSetup, "macos_setup (via setup_experience in external file) not parsed")
	})

	// Test that duplicate settings with old and new key names produce an error
	t.Run("duplicate_old_and_new_keys_error_apple_settings", func(t *testing.T) {
		dir := t.TempDir()
		profileDir := filepath.Join(dir, "lib")
		require.NoError(t, os.Mkdir(profileDir, 0o755))
		config := `
reports:
policies:
agent_options:
org_settings:
  server_settings:
  org_info:
  secrets:
controls:
  apple_settings:
    configuration_profiles:
      - path: ./lib/macos-password.mobileconfig
  macos_settings:
`
		yamlPath := filepath.Join(dir, "gitops.yml")
		require.NoError(t, os.WriteFile(yamlPath, []byte(config), 0o644))

		_, err := GitOpsFromFile(yamlPath, dir, nil, nopLogf)
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot specify both")
		require.Contains(t, err.Error(), "apple_settings")
		require.Contains(t, err.Error(), "'controls.macos_settings' (deprecated)")
	})

	t.Run("duplicate_old_and_new_keys_error_apple_custom_settings", func(t *testing.T) {
		dir := t.TempDir()
		profileDir := filepath.Join(dir, "lib")
		require.NoError(t, os.Mkdir(profileDir, 0o755))
		config := `
reports:
policies:
agent_options:
org_settings:
  server_settings:
  org_info:
  secrets:
controls:
  apple_settings:
    configuration_profiles:
      - path: ./lib/macos-password.mobileconfig
    custom_settings:
      - path: ./lib/macos-password.mobileconfig
`
		yamlPath := filepath.Join(dir, "gitops.yml")
		require.NoError(t, os.WriteFile(yamlPath, []byte(config), 0o644))

		_, err := GitOpsFromFile(yamlPath, dir, nil, nopLogf)
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot specify both")
		require.Contains(t, err.Error(), "configuration_profiles")
		require.Contains(t, err.Error(), "'controls.apple_settings.custom_settings' (deprecated)")
	})

	t.Run("duplicate_old_and_new_keys_error_windows_custom_settings", func(t *testing.T) {
		dir := t.TempDir()
		profileDir := filepath.Join(dir, "lib")
		require.NoError(t, os.Mkdir(profileDir, 0o755))
		config := `
reports:
policies:
agent_options:
org_settings:
  server_settings:
  org_info:
  secrets:
controls:
  windows_settings:
    configuration_profiles:
      - path: ./lib/foo
    custom_settings:
      - path: ./lib/bar
`
		yamlPath := filepath.Join(dir, "gitops.yml")
		require.NoError(t, os.WriteFile(yamlPath, []byte(config), 0o644))

		_, err := GitOpsFromFile(yamlPath, dir, nil, nopLogf)
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot specify both")
		require.Contains(t, err.Error(), "configuration_profiles")
		require.Contains(t, err.Error(), "'controls.windows_settings.custom_settings' (deprecated)")
	})

	t.Run("duplicate_old_and_new_keys_error_android_custom_settings", func(t *testing.T) {
		dir := t.TempDir()
		profileDir := filepath.Join(dir, "lib")
		require.NoError(t, os.Mkdir(profileDir, 0o755))
		config := `
reports:
policies:
agent_options:
org_settings:
  server_settings:
  org_info:
  secrets:
controls:
  android_settings:
    configuration_profiles:
      - path: ./lib/foo
    custom_settings:
      - path: ./lib/bar
`
		yamlPath := filepath.Join(dir, "gitops.yml")
		require.NoError(t, os.WriteFile(yamlPath, []byte(config), 0o644))

		_, err := GitOpsFromFile(yamlPath, dir, nil, nopLogf)
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot specify both")
		require.Contains(t, err.Error(), "configuration_profiles")
		require.Contains(t, err.Error(), "'controls.android_settings.custom_settings' (deprecated)")
	})

	t.Run("duplicate_old_and_new_keys_error_setup_experience", func(t *testing.T) {
		dir := t.TempDir()
		profileDir := filepath.Join(dir, "lib")
		require.NoError(t, os.Mkdir(profileDir, 0o755))
		config := `
reports:
policies:
agent_options:
org_settings:
  server_settings:
  org_info:
  secrets:
controls:
  setup_experience:
  macos_setup:
`
		yamlPath := filepath.Join(dir, "gitops.yml")
		require.NoError(t, os.WriteFile(yamlPath, []byte(config), 0o644))

		_, err := GitOpsFromFile(yamlPath, dir, nil, nopLogf)
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot specify both")
		require.Contains(t, err.Error(), "setup_experience")
		require.Contains(t, err.Error(), "'controls.macos_setup' (deprecated)")
	})

	t.Run("duplicate_old_and_new_keys_error_bootstrap_package", func(t *testing.T) {
		dir := t.TempDir()
		config := `
reports:
policies:
agent_options:
org_settings:
  server_settings:
  org_info:
  secrets:
controls:
  setup_experience:
    bootstrap_package: ""
    macos_bootstrap_package: ""
`
		yamlPath := filepath.Join(dir, "gitops.yml")
		require.NoError(t, os.WriteFile(yamlPath, []byte(config), 0o644))

		_, err := GitOpsFromFile(yamlPath, dir, nil, nopLogf)
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot specify both")
		require.Contains(t, err.Error(), "macos_bootstrap_package")
		require.Contains(t, err.Error(), "'controls.setup_experience.bootstrap_package' (deprecated)")
	})

	t.Run("duplicate_old_and_new_keys_error_enable_release_device_manually", func(t *testing.T) {
		dir := t.TempDir()
		config := `
reports:
policies:
agent_options:
org_settings:
  server_settings:
  org_info:
  secrets:
controls:
  setup_experience:
    enable_release_device_manually: false
    apple_enable_release_device_manually: false
`
		yamlPath := filepath.Join(dir, "gitops.yml")
		require.NoError(t, os.WriteFile(yamlPath, []byte(config), 0o644))

		_, err := GitOpsFromFile(yamlPath, dir, nil, nopLogf)
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot specify both")
		require.Contains(t, err.Error(), "apple_enable_release_device_manually")
		require.Contains(t, err.Error(), "'controls.setup_experience.enable_release_device_manually' (deprecated)")
	})

	t.Run("duplicate_old_and_new_keys_error_script", func(t *testing.T) {
		dir := t.TempDir()
		config := `
reports:
policies:
agent_options:
org_settings:
  server_settings:
  org_info:
  secrets:
controls:
  setup_experience:
    script: null
    macos_script: null
`
		yamlPath := filepath.Join(dir, "gitops.yml")
		require.NoError(t, os.WriteFile(yamlPath, []byte(config), 0o644))

		_, err := GitOpsFromFile(yamlPath, dir, nil, nopLogf)
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot specify both")
		require.Contains(t, err.Error(), "macos_script")
		require.Contains(t, err.Error(), "'controls.setup_experience.script' (deprecated)")
	})

	t.Run("duplicate_old_and_new_keys_error_manual_agent_install", func(t *testing.T) {
		dir := t.TempDir()
		config := `
reports:
policies:
agent_options:
org_settings:
  server_settings:
  org_info:
  secrets:
controls:
  setup_experience:
    manual_agent_install: false
    macos_manual_agent_install: false
`
		yamlPath := filepath.Join(dir, "gitops.yml")
		require.NoError(t, os.WriteFile(yamlPath, []byte(config), 0o644))

		_, err := GitOpsFromFile(yamlPath, dir, nil, nopLogf)
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot specify both")
		require.Contains(t, err.Error(), "macos_manual_agent_install")
		require.Contains(t, err.Error(), "'controls.setup_experience.manual_agent_install' (deprecated)")
	})

	t.Run("duplicate_keys_external_file", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		profileDir := filepath.Join(dir, "lib")
		require.NoError(t, os.Mkdir(profileDir, 0o755))

		controlsYAML := `
apple_settings:
macos_settings:
`
		controlsPath := filepath.Join(dir, "controls.yml")
		require.NoError(t, os.WriteFile(controlsPath, []byte(controlsYAML), 0o644))

		config := `
controls:
  path: ./controls.yml
reports:
policies:
agent_options:
org_settings:
  secrets:
`
		yamlPath := filepath.Join(dir, "gitops.yml")
		require.NoError(t, os.WriteFile(yamlPath, []byte(config), 0o644))

		_, err := GitOpsFromFile(yamlPath, dir, nil, nopLogf)
		require.Error(t, err)
		require.Contains(t, err.Error(), "cannot specify both")
		require.Contains(t, err.Error(), "apple_settings")
		require.Contains(t, err.Error(), "`macos_settings` (deprecated)")
	})
}

func TestSoftwarePackagesScriptPath(t *testing.T) {
	t.Parallel()
	appConfig := &fleet.EnrichedAppConfig{}
	appConfig.License = &fleet.LicenseInfo{
		Tier: fleet.TierPremium,
	}

	t.Run("valid_sh_script_path", func(t *testing.T) {
		config := getTeamConfig([]string{"software"})
		config += `
software:
  packages:
    - path: software/install-app.sh
      categories:
        - Utilities
      self_service: true
`
		path, basePath := createTempFile(t, "", config)

		err := file.Copy(
			filepath.Join("testdata", "software", "install-app.sh"),
			filepath.Join(basePath, "software", "install-app.sh"),
			os.FileMode(0o755),
		)
		require.NoError(t, err)

		result, err := GitOpsFromFile(path, basePath, appConfig, nopLogf)
		require.NoError(t, err)
		require.Len(t, result.Software.Packages, 1)
		assert.True(t, strings.HasSuffix(result.Software.Packages[0].InstallScript.Path, "install-app.sh"))
		assert.Equal(t, []string{"Utilities"}, result.Software.Packages[0].Categories)
		assert.True(t, result.Software.Packages[0].SelfService)
		assert.Empty(t, result.Software.Packages[0].URL)
		assert.Empty(t, result.Software.Packages[0].SHA256)
	})

	t.Run("sh_script_with_shell_variables", func(t *testing.T) {
		config := getTeamConfig([]string{"software"})
		config += `
software:
  packages:
    - path: software/shell-vars.sh
      self_service: true
`
		path, basePath := createTempFile(t, "", config)

		// Create a script that uses standard shell variables (not set in CI env).
		_, expectedVarUnset := os.LookupEnv("SOMETHING_UNSET") // Make sure at least one is not set in the CI environment
		require.False(t, expectedVarUnset, "SOMETHING_UNSET should not be set in the test environment")
		scriptContent := []byte("#!/bin/bash\necho \"EUID=$EUID\"\necho \"USER=$USER\"\necho \"HOME=$HOME\"\necho \"SOMETHING_UNSET=$SOMETHING_UNSET\"\n")
		require.NoError(t, os.MkdirAll(filepath.Join(basePath, "software"), 0o755))                                  // nolint:gosec
		require.NoError(t, os.WriteFile(filepath.Join(basePath, "software", "shell-vars.sh"), scriptContent, 0o755)) // nolint:gosec

		// Shell variables must not be expanded by the gitops parser.
		result, err := GitOpsFromFile(path, basePath, appConfig, nopLogf)
		require.NoError(t, err)
		require.Len(t, result.Software.Packages, 1)
		assert.True(t, strings.HasSuffix(result.Software.Packages[0].InstallScript.Path, "shell-vars.sh"))
	})

	t.Run("valid_ps1_script_path", func(t *testing.T) {
		config := getTeamConfig([]string{"software"})
		config += `
software:
  packages:
    - path: software/install-app.ps1
      self_service: false
`
		path, basePath := createTempFile(t, "", config)

		// Copy the test script file
		err := file.Copy(
			filepath.Join("testdata", "software", "install-app.ps1"),
			filepath.Join(basePath, "software", "install-app.ps1"),
			os.FileMode(0o755),
		)
		require.NoError(t, err)

		result, err := GitOpsFromFile(path, basePath, appConfig, nopLogf)
		require.NoError(t, err)
		require.Len(t, result.Software.Packages, 1)
		assert.True(t, strings.HasSuffix(result.Software.Packages[0].InstallScript.Path, "install-app.ps1"))
	})

	t.Run("invalid_extension_error", func(t *testing.T) {
		config := getTeamConfig([]string{"software"})
		config += `
software:
  packages:
    - path: software/install-app.txt
`
		path, basePath := createTempFile(t, "", config)

		// Create a .txt file
		err := os.MkdirAll(filepath.Join(basePath, "software"), 0o755)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(basePath, "software", "install-app.txt"), []byte("test"), 0o644)
		require.NoError(t, err)

		_, err = GitOpsFromFile(path, basePath, appConfig, nopLogf)
		assert.ErrorContains(t, err, "unsupported extension")
		assert.ErrorContains(t, err, "only .yml, .yaml, .sh, or .ps1 files are supported")
	})

	t.Run("script_with_team_options", func(t *testing.T) {
		config := getTeamConfig([]string{"software"})
		config += `
software:
  packages:
    - path: software/install-app.sh
      categories:
        - Browsers
        - Productivity
      self_service: true
      setup_experience: true
      labels_include_any:
        - include_label
`
		path, basePath := createTempFile(t, "", config)

		err := file.Copy(
			filepath.Join("testdata", "software", "install-app.sh"),
			filepath.Join(basePath, "software", "install-app.sh"),
			os.FileMode(0o755),
		)
		require.NoError(t, err)

		result, err := GitOpsFromFile(path, basePath, appConfig, nopLogf)
		require.NoError(t, err)
		require.Len(t, result.Software.Packages, 1)
		pkg := result.Software.Packages[0]
		assert.Equal(t, []string{"Browsers", "Productivity"}, pkg.Categories)
		assert.True(t, pkg.SelfService)
		assert.True(t, pkg.InstallDuringSetup.Value)
		assert.Equal(t, []string{"include_label"}, pkg.LabelsIncludeAny)
	})

	t.Run("mixed_yaml_and_script_paths", func(t *testing.T) {
		config := getTeamConfig([]string{"software"})
		config += `
software:
  packages:
    - path: software/single-package.yml
    - path: software/install-app.sh
      self_service: true
`
		path, basePath := createTempFile(t, "", config)

		err := file.Copy(
			filepath.Join("testdata", "software", "single-package.yml"),
			filepath.Join(basePath, "software", "single-package.yml"),
			os.FileMode(0o755),
		)
		require.NoError(t, err)
		err = file.Copy(
			filepath.Join("testdata", "software", "install-app.sh"),
			filepath.Join(basePath, "software", "install-app.sh"),
			os.FileMode(0o755),
		)
		require.NoError(t, err)

		result, err := GitOpsFromFile(path, basePath, appConfig, nopLogf)
		require.NoError(t, err)
		require.Len(t, result.Software.Packages, 2)
		assert.NotEmpty(t, result.Software.Packages[0].SHA256)
		assert.True(t, strings.HasSuffix(result.Software.Packages[1].InstallScript.Path, "install-app.sh"))
		assert.True(t, result.Software.Packages[1].SelfService)
	})
}

func TestParsePolicyInstallSoftware(t *testing.T) {
	t.Parallel()

	teamName := "test-team"

	t.Run("wrapErrs prefixes errors", func(t *testing.T) {
		t.Parallel()

		var installSoftware optjson.BoolOr[*PolicyInstallSoftware]
		installSoftware.Other = &PolicyInstallSoftware{}

		policy := &Policy{
			GitOpsPolicySpec: GitOpsPolicySpec{
				PolicySpec:      fleet.PolicySpec{Name: "my policy"},
				InstallSoftware: installSoftware, // no package_path, app_store_id, or hash_sha256
			},
		}
		errs := parsePolicyInstallSoftware(".", &teamName, policy, nil, nil, nil)
		require.Len(t, errs, 1)
		assert.Equal(t, errs[0].Error(), `failed to parse policy install_software "my policy": install_software must include either a package_path, an app_store_id, a hash_sha256 or a fleet_maintained_app_slug`)
	})

	t.Run("unknown key in package_path file", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		sha := "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"
		content := fmt.Sprintf("hash_sha256: %s\nbad_field: oops\n", sha)
		path := filepath.Join(dir, "pkg.yml")
		require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

		var installSoftware optjson.BoolOr[*PolicyInstallSoftware]
		installSoftware.Other = &PolicyInstallSoftware{PackagePath: path}

		policy := &Policy{
			GitOpsPolicySpec: GitOpsPolicySpec{
				PolicySpec:      fleet.PolicySpec{Name: "typo policy"},
				InstallSoftware: installSoftware,
			},
		}
		packages := []*fleet.SoftwarePackageSpec{{SHA256: sha}}
		errs := parsePolicyInstallSoftware(".", &teamName, policy, packages, nil, nil)
		require.Len(t, errs, 1)
		var unknownErr *ParseUnknownKeyError
		require.ErrorAs(t, errs[0], &unknownErr)
		assert.Equal(t, "bad_field", unknownErr.Field)
	})
	t.Run("fleet_maintained_app_slug valid", func(t *testing.T) {
		t.Parallel()

		var installSoftware optjson.BoolOr[*PolicyInstallSoftware]
		installSoftware.Other = &PolicyInstallSoftware{FleetMaintainedAppSlug: "zoom/darwin"}

		policy := &Policy{
			GitOpsPolicySpec: GitOpsPolicySpec{
				PolicySpec:      fleet.PolicySpec{Name: "fma policy"},
				InstallSoftware: installSoftware,
			},
		}
		fmasBySlug := map[string]struct{}{"zoom/darwin": {}}
		errs := parsePolicyInstallSoftware(".", &teamName, policy, nil, nil, fmasBySlug)
		require.Nil(t, errs)
		assert.Equal(t, "zoom/darwin", policy.FleetMaintainedAppSlug)
	})

	t.Run("fleet_maintained_app_slug not in FMAs", func(t *testing.T) {
		t.Parallel()

		var installSoftware optjson.BoolOr[*PolicyInstallSoftware]
		installSoftware.Other = &PolicyInstallSoftware{FleetMaintainedAppSlug: "notreal/darwin"}

		policy := &Policy{
			GitOpsPolicySpec: GitOpsPolicySpec{
				PolicySpec:      fleet.PolicySpec{Name: "bad fma policy"},
				InstallSoftware: installSoftware,
			},
		}
		fmasBySlug := map[string]struct{}{"zoom/darwin": {}}
		errs := parsePolicyInstallSoftware(".", &teamName, policy, nil, nil, fmasBySlug)
		require.Len(t, errs, 1)
		assert.Contains(t, errs[0].Error(), `fleet_maintained_app_slug "notreal/darwin" not found`)
	})

	t.Run("fleet_maintained_app_slug with other fields errors", func(t *testing.T) {
		t.Parallel()

		var installSoftware optjson.BoolOr[*PolicyInstallSoftware]
		installSoftware.Other = &PolicyInstallSoftware{
			FleetMaintainedAppSlug: "zoom/darwin",
			HashSHA256:             "abc123",
		}

		policy := &Policy{
			GitOpsPolicySpec: GitOpsPolicySpec{
				PolicySpec:      fleet.PolicySpec{Name: "conflicting policy"},
				InstallSoftware: installSoftware,
			},
		}
		errs := parsePolicyInstallSoftware(".", &teamName, policy, nil, nil, nil)
		require.Len(t, errs, 1)
		assert.Contains(t, errs[0].Error(), "install_software must have only one of")
	})

	t.Run("fleet_maintained_app_slug on global policy errors", func(t *testing.T) {
		t.Parallel()

		var installSoftware optjson.BoolOr[*PolicyInstallSoftware]
		installSoftware.Other = &PolicyInstallSoftware{FleetMaintainedAppSlug: "zoom/darwin"}

		policy := &Policy{
			GitOpsPolicySpec: GitOpsPolicySpec{
				PolicySpec:      fleet.PolicySpec{Name: "global fma policy"},
				InstallSoftware: installSoftware,
			},
		}
		errs := parsePolicyInstallSoftware(".", nil, policy, nil, nil, nil)
		require.Len(t, errs, 1)
		assert.Contains(t, errs[0].Error(), "install_software can only be set on team policies")
	})
}

func TestGitOpsPresenceTracking(t *testing.T) {
	t.Run("labels present", func(t *testing.T) {
		gitops, err := gitOpsFromString(t, `
org_settings:
  server_settings:
    server_url: https://example.com
  org_info:
    org_name: Test
labels:
  - name: test-label
    query: SELECT 1
`)
		require.NoError(t, err)
		assert.True(t, gitops.LabelsPresent)
	})

	t.Run("labels absent", func(t *testing.T) {
		gitops, err := gitOpsFromString(t, `
org_settings:
  server_settings:
    server_url: https://example.com
  org_info:
    org_name: Test
`)
		require.NoError(t, err)
		assert.False(t, gitops.LabelsPresent)
		assert.Nil(t, gitops.Labels, "absent labels should be nil")
	})

	t.Run("labels present but empty", func(t *testing.T) {
		gitops, err := gitOpsFromString(t, `
org_settings:
  server_settings:
    server_url: https://example.com
  org_info:
    org_name: Test
labels:
`)
		require.NoError(t, err)
		assert.True(t, gitops.LabelsPresent)
		assert.Nil(t, gitops.Labels, "empty labels section should result in nil")
	})

	t.Run("secrets present", func(t *testing.T) {
		gitops, err := gitOpsFromString(t, `
org_settings:
  server_settings:
    server_url: https://example.com
  org_info:
    org_name: Test
  secrets:
    - secret: mysecret
`)
		require.NoError(t, err)
		assert.True(t, gitops.SecretsPresent)
	})

	t.Run("secrets absent", func(t *testing.T) {
		gitops, err := gitOpsFromString(t, `
org_settings:
  server_settings:
    server_url: https://example.com
  org_info:
    org_name: Test
`)
		require.NoError(t, err)
		assert.False(t, gitops.SecretsPresent)
	})

	t.Run("software present on team", func(t *testing.T) {
		premiumConfig := &fleet.EnrichedAppConfig{}
		premiumConfig.License = &fleet.LicenseInfo{Tier: fleet.TierPremium}

		path, basePath := createTempFile(t, "", `
name: TestTeam
software:
  packages:
    - url: https://example.com/pkg.deb
`)
		gitops, err := GitOpsFromFile(path, basePath, premiumConfig, nopLogf)
		require.NoError(t, err)
		assert.True(t, gitops.SoftwarePresent)
	})

	t.Run("software absent on team", func(t *testing.T) {
		premiumConfig := &fleet.EnrichedAppConfig{}
		premiumConfig.License = &fleet.LicenseInfo{Tier: fleet.TierPremium}

		path, basePath := createTempFile(t, "", `
name: TestTeam
`)
		gitops, err := GitOpsFromFile(path, basePath, premiumConfig, nopLogf)
		require.NoError(t, err)
		assert.False(t, gitops.SoftwarePresent)
	})
}
