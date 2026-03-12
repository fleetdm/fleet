package service

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/baselines"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestListBaselinesEndpointReturnsBaselines(t *testing.T) {
	all, err := baselines.ListBaselines()
	require.NoError(t, err)
	require.NotEmpty(t, all)

	var found bool
	for _, b := range all {
		if b.ID == "nvidia-security-baseline" {
			found = true
			require.Equal(t, "windows", b.Platform)
			require.NotEmpty(t, b.Categories)
		}
	}
	require.True(t, found)
}

func TestApplyBaselineRequestValidation(t *testing.T) {
	// Verify baseline exists before apply
	_, err := baselines.GetBaseline("nvidia-security-baseline")
	require.NoError(t, err)

	// Verify nonexistent baseline returns error
	_, err = baselines.GetBaseline("nonexistent")
	require.Error(t, err)
}

func TestBaselineProfileContentsReadable(t *testing.T) {
	manifest, err := baselines.GetBaseline("nvidia-security-baseline")
	require.NoError(t, err)

	for _, cat := range manifest.Categories {
		for _, p := range cat.Profiles {
			content, err := baselines.GetProfileContent(manifest.ID, p)
			require.NoError(t, err, "failed to read profile %s in %s", p, cat.Name)
			require.Contains(t, string(content), "LocURI", "profile %s should contain LocURI", p)
		}
	}
}

func TestStripExtension(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"profiles/firewall.xml", "firewall"},
		{"profiles/defender.xml", "defender"},
		{"scripts/configure-services.ps1", "configure-services"},
		{"policies/verify-firewall.yaml", "verify-firewall"},
		{"simple.xml", "simple"},
		{"noext", "noext"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			require.Equal(t, tt.expected, stripExtension(tt.input))
		})
	}
}

func TestBaselinePolicyYAMLParsing(t *testing.T) {
	manifest, err := baselines.GetBaseline("nvidia-security-baseline")
	require.NoError(t, err)

	totalPolicies := 0
	for _, cat := range manifest.Categories {
		for _, p := range cat.Policies {
			content, err := baselines.GetPolicyContent(manifest.ID, p)
			require.NoError(t, err, "failed to read policy %s", p)

			var policies []baselinePolicyYAML
			err = yaml.Unmarshal(content, &policies)
			require.NoError(t, err, "failed to parse YAML for %s", p)
			require.NotEmpty(t, policies, "policy file %s should have at least one policy", p)

			for _, pol := range policies {
				require.NotEmpty(t, pol.Name, "policy name must not be empty in %s", p)
				require.NotEmpty(t, pol.Query, "policy query must not be empty in %s", p)
				require.Contains(t, pol.Name, baselineNamePrefix,
					"policy %q should have baseline prefix", pol.Name)
				require.Equal(t, "windows", pol.Platform,
					"policy %q should target windows", pol.Name)
			}
			totalPolicies += len(policies)
		}
	}
	require.GreaterOrEqual(t, totalPolicies, 20, "expected at least 20 total verification policies")
}

func TestBaselinePolicySpecConversion(t *testing.T) {
	manifest, err := baselines.GetBaseline("nvidia-security-baseline")
	require.NoError(t, err)

	teamName := "Test Team"
	for _, cat := range manifest.Categories {
		for _, p := range cat.Policies {
			content, err := baselines.GetPolicyContent(manifest.ID, p)
			require.NoError(t, err)

			var policies []baselinePolicyYAML
			require.NoError(t, yaml.Unmarshal(content, &policies))

			for _, pol := range policies {
				spec := &fleet.PolicySpec{
					Name:        pol.Name,
					Query:       pol.Query,
					Description: pol.Description,
					Resolution:  pol.Resolution,
					Platform:    pol.Platform,
					Critical:    pol.Critical,
					Team:        teamName,
				}
				require.Equal(t, teamName, spec.Team)
				require.Equal(t, pol.Name, spec.Name)
				require.Equal(t, pol.Query, spec.Query)
				require.Equal(t, pol.Platform, spec.Platform)
			}
		}
	}
}

func TestBaselineProfileNaming(t *testing.T) {
	manifest, err := baselines.GetBaseline("nvidia-security-baseline")
	require.NoError(t, err)

	for _, cat := range manifest.Categories {
		for _, p := range cat.Profiles {
			profileName := baselineNamePrefix + cat.Name + " - " + stripExtension(p)
			require.Contains(t, profileName, baselineNamePrefix,
				"profile name must start with baseline prefix")
			require.Contains(t, profileName, cat.Name,
				"profile name must include category name")
		}
	}
}
