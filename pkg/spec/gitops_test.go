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
  secrets:`,
}

func TestValidGitOpsYaml(t *testing.T) {
	t.Parallel()
	dat, err := os.ReadFile("test_data/global_config.yml")
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

	// Check agent options

	// Check queries

	// Check policies

}

func TestInvalidGitOpsYaml(t *testing.T) {
	t.Parallel()
	_, err := GitOpsFromBytes([]byte("bad:\nbad"), "")
	assert.ErrorContains(t, err, "failed to unmarshal")
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
			goodConfig: topLevelOptions["org_settings"],
		},
		"controls": {
			isArray:    false,
			goodConfig: topLevelOptions["controls"],
		},
		"queries": {
			isArray:    true,
			goodConfig: topLevelOptions["queries"],
		},
		"policies": {
			isArray:    true,
			goodConfig: topLevelOptions["policies"],
		},
		"agent_options": {
			isArray:    false,
			goodConfig: topLevelOptions["agent_options"],
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

				// Test a relative top level path
				config = getBaseConfig([]string{name})
				dir, file := filepath.Split(tmpFile.Name())
				if test.isArray {
					config += fmt.Sprintf("%s:\n  - path: ./%s\n", name, file)
				} else {
					config += fmt.Sprintf("%s:\n  path: ./%s\n", name, file)
				}
				_, err = GitOpsFromBytes([]byte(config), dir)

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

// All good settings -- check them, including secrets

// Duplicate policies and queries

// Non-Ascii query names
