package spec

import (
	"fmt"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"slices"
	"testing"
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

func TestValidGitOpsYaml(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		filePath string
	}{
		"global_config_no_paths": {
			filePath: "test_data/global_config_no_paths.yml",
		},
		"global_config_with_paths": {
			filePath: "test_data/global_config.yml",
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
				gitops, err := GitOpsFromBytes(dat, "./test_data")
				require.NoError(t, err)

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
				assert.Contains(t, gitops.OrgSettings, "features")
				assert.Contains(t, gitops.OrgSettings, "vulnerability_settings")
				assert.Contains(t, gitops.OrgSettings, "secrets")
				secrets, ok := gitops.OrgSettings["secrets"]
				assert.True(t, ok, "secrets not found")
				require.Len(t, secrets.([]*fleet.EnrollSecret), 2)
				assert.Equal(t, "SampleSecret123", secrets.([]*fleet.EnrollSecret)[0].Secret)
				assert.Equal(t, "ABC", secrets.([]*fleet.EnrollSecret)[1].Secret)

				// Check controls
				_, ok = gitops.Controls.MacOSSettings.(map[string]interface{})
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
	config := getBaseConfig([]string{"policies"})
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
	config := getBaseConfig([]string{"queries"})
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
	config := getBaseConfig([]string{"queries"})
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

func TestInvalidGitOpsYaml(t *testing.T) {
	t.Parallel()
	_, err := GitOpsFromBytes([]byte("bad:\nbad"), "")
	assert.ErrorContains(t, err, "failed to unmarshal")

	// Invalid org_settings
	config := getBaseConfig([]string{"org_settings"})
	config += "org_settings:\n  path: [2]\n"
	_, err = GitOpsFromBytes([]byte(config), "")
	assert.ErrorContains(t, err, "failed to unmarshal org_settings")

	// Invalid org_settings in a separate file
	tmpFile, err := os.CreateTemp(t.TempDir(), "*org_settings.yml")
	require.NoError(t, err)
	_, err = tmpFile.WriteString("[2]")
	require.NoError(t, err)
	config = getBaseConfig([]string{"org_settings"})
	config += fmt.Sprintf("%s:\n  path: %s\n", "org_settings", tmpFile.Name())
	_, err = GitOpsFromBytes([]byte(config), "")
	assert.ErrorContains(t, err, "failed to unmarshal org settings file")

	// Invalid secrets 1
	config = getBaseConfig([]string{"org_settings"})
	config += "org_settings:\n  secrets: bad\n"
	_, err = GitOpsFromBytes([]byte(config), "")
	assert.ErrorContains(t, err, "must be a list of secret items")

	// Invalid secrets 2
	config = getBaseConfig([]string{"org_settings"})
	config += "org_settings:\n  secrets: [2]\n"
	_, err = GitOpsFromBytes([]byte(config), "")
	assert.ErrorContains(t, err, "must have a 'secret' key")

	// Invalid agent_options
	config = getBaseConfig([]string{"agent_options"})
	config += "agent_options:\n  path: [2]\n"
	_, err = GitOpsFromBytes([]byte(config), "")
	assert.ErrorContains(t, err, "failed to unmarshal agent_options")

	// Invalid org_settings in a separate file
	tmpFile, err = os.CreateTemp(t.TempDir(), "*agent_options.yml")
	require.NoError(t, err)
	_, err = tmpFile.WriteString("[2]")
	require.NoError(t, err)
	config = getBaseConfig([]string{"agent_options"})
	config += fmt.Sprintf("%s:\n  path: %s\n", "agent_options", tmpFile.Name())
	_, err = GitOpsFromBytes([]byte(config), "")
	assert.ErrorContains(t, err, "failed to unmarshal agent options file")

	// Invalid controls
	config = getBaseConfig([]string{"controls"})
	config += "controls:\n  path: [2]\n"
	_, err = GitOpsFromBytes([]byte(config), "")
	assert.ErrorContains(t, err, "failed to unmarshal controls")

	// Invalid controls in a separate file
	tmpFile, err = os.CreateTemp(t.TempDir(), "*controls.yml")
	require.NoError(t, err)
	_, err = tmpFile.WriteString("[2]")
	require.NoError(t, err)
	config = getBaseConfig([]string{"controls"})
	config += fmt.Sprintf("%s:\n  path: %s\n", "controls", tmpFile.Name())
	_, err = GitOpsFromBytes([]byte(config), "")
	assert.ErrorContains(t, err, "failed to unmarshal controls file")

	// Invalid policies
	config = getBaseConfig([]string{"policies"})
	config += "policies:\n  path: [2]\n"
	_, err = GitOpsFromBytes([]byte(config), "")
	assert.ErrorContains(t, err, "failed to unmarshal policies")

	// Invalid policies in a separate file
	tmpFile, err = os.CreateTemp(t.TempDir(), "*policies.yml")
	require.NoError(t, err)
	_, err = tmpFile.WriteString("[2]")
	require.NoError(t, err)
	config = getBaseConfig([]string{"policies"})
	config += fmt.Sprintf("%s:\n  - path: %s\n", "policies", tmpFile.Name())
	_, err = GitOpsFromBytes([]byte(config), "")
	assert.ErrorContains(t, err, "failed to unmarshal policies file")

	// Invalid queries
	config = getBaseConfig([]string{"queries"})
	config += "queries:\n  path: [2]\n"
	_, err = GitOpsFromBytes([]byte(config), "")
	assert.ErrorContains(t, err, "failed to unmarshal queries")

	// Invalid policies in a separate file
	tmpFile, err = os.CreateTemp(t.TempDir(), "*queries.yml")
	require.NoError(t, err)
	_, err = tmpFile.WriteString("[2]")
	require.NoError(t, err)
	config = getBaseConfig([]string{"queries"})
	config += fmt.Sprintf("%s:\n  - path: %s\n", "queries", tmpFile.Name())
	_, err = GitOpsFromBytes([]byte(config), "")
	assert.ErrorContains(t, err, "failed to unmarshal queries file")

}

func TestTopLevelGitOpsValidation(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		optsToExclude []string
		shouldPass    bool
	}{
		"all_present_global": {
			optsToExclude: []string{},
			shouldPass:    true,
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
	}
	for name, test := range tests {
		t.Run(
			name, func(t *testing.T) {
				config := getBaseConfig(test.optsToExclude)
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

func TestGitOpsPaths(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		isArray    bool
		goodConfig string
	}{
		"org_settings": {
			isArray:    false,
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

				// Test an absolute top level path
				tmpFile, err := os.CreateTemp(t.TempDir(), "*good.yml")
				require.NoError(t, err)
				_, err = tmpFile.WriteString(test.goodConfig)
				require.NoError(t, err)
				config := getBaseConfig([]string{name})
				if test.isArray {
					config += fmt.Sprintf("%s:\n  - path: %s\n", name, tmpFile.Name())
				} else {
					config += fmt.Sprintf("%s:\n  path: %s\n", name, tmpFile.Name())
				}
				_, err = GitOpsFromBytes([]byte(config), "")
				assert.NoError(t, err)

				// Test a relative top level path
				config = getBaseConfig([]string{name})
				dir, file := filepath.Split(tmpFile.Name())
				if test.isArray {
					config += fmt.Sprintf("%s:\n  - path: ./%s\n", name, file)
				} else {
					config += fmt.Sprintf("%s:\n  path: ./%s\n", name, file)
				}
				_, err = GitOpsFromBytes([]byte(config), dir)
				assert.NoError(t, err)

				// Test a bad path
				config = getBaseConfig([]string{name})
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
				config = getBaseConfig([]string{name})
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
				config = getBaseConfig([]string{name})
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

func getBaseConfig(optsToExclude []string) string {
	var config string
	for key, value := range topLevelOptions {
		if !slices.Contains(optsToExclude, key) {
			config += value + "\n"
		}
	}
	return config
}
