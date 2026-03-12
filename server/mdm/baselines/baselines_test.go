package baselines

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestListBaselines(t *testing.T) {
	baselines, err := ListBaselines()
	require.NoError(t, err)
	require.NotEmpty(t, baselines, "expected at least one embedded baseline")

	// Verify the NVIDIA baseline is present
	var found bool
	for _, b := range baselines {
		if b.ID == "nvidia-security-baseline" {
			found = true
			require.Equal(t, "NVIDIA Windows Security Baseline", b.Name)
			require.Equal(t, "1.0.0", b.Version)
			require.Equal(t, "windows", b.Platform)
			require.NotEmpty(t, b.Description)
			require.NotEmpty(t, b.Categories)
			break
		}
	}
	require.True(t, found, "expected nvidia-security-baseline in embedded baselines")
}

func TestGetBaseline(t *testing.T) {
	t.Run("existing baseline", func(t *testing.T) {
		manifest, err := GetBaseline("nvidia-security-baseline")
		require.NoError(t, err)
		require.NotNil(t, manifest)
		require.Equal(t, "nvidia-security-baseline", manifest.ID)
		require.Equal(t, "windows", manifest.Platform)
	})

	t.Run("nonexistent baseline", func(t *testing.T) {
		_, err := GetBaseline("does-not-exist")
		require.Error(t, err)
		require.Contains(t, err.Error(), "not found")
	})
}

func TestBaselineCategories(t *testing.T) {
	manifest, err := GetBaseline("nvidia-security-baseline")
	require.NoError(t, err)

	// Verify expected categories exist
	categoryNames := make(map[string]bool)
	for _, c := range manifest.Categories {
		categoryNames[c.Name] = true
	}

	expectedCategories := []string{
		"Windows Firewall",
		"Windows Defender",
		"Credential Guard",
		"Attack Surface Reduction",
		"Audit Policies",
		"Network Security",
		"Account Lockout",
		"User Rights",
		"Browser Security",
		"Service Configuration",
	}

	for _, name := range expectedCategories {
		require.True(t, categoryNames[name], "expected category %q", name)
	}
}

func TestBaselineCategoryContents(t *testing.T) {
	manifest, err := GetBaseline("nvidia-security-baseline")
	require.NoError(t, err)

	for _, c := range manifest.Categories {
		t.Run(c.Name, func(t *testing.T) {
			// Every category must have at least one policy
			require.NotEmpty(t, c.Policies, "category %q must have at least one policy", c.Name)

			// Categories with profiles must reference .xml files
			for _, p := range c.Profiles {
				require.Contains(t, p, ".xml", "profile path must be .xml: %s", p)
			}

			// Categories with policies must reference .yaml files
			for _, p := range c.Policies {
				require.Contains(t, p, ".yaml", "policy path must be .yaml: %s", p)
			}

			// Categories with scripts must reference .ps1 files
			for _, s := range c.Scripts {
				require.Contains(t, s, ".ps1", "script path must be .ps1: %s", s)
			}
		})
	}
}

func TestAllReferencedFilesExist(t *testing.T) {
	manifest, err := GetBaseline("nvidia-security-baseline")
	require.NoError(t, err)

	for _, c := range manifest.Categories {
		for _, p := range c.Profiles {
			content, err := GetProfileContent(manifest.ID, p)
			require.NoError(t, err, "profile %q in category %q not found", p, c.Name)
			require.NotEmpty(t, content, "profile %q is empty", p)
		}
		for _, p := range c.Policies {
			content, err := GetPolicyContent(manifest.ID, p)
			require.NoError(t, err, "policy %q in category %q not found", p, c.Name)
			require.NotEmpty(t, content, "policy %q is empty", p)
		}
		for _, s := range c.Scripts {
			content, err := GetScriptContent(manifest.ID, s)
			require.NoError(t, err, "script %q in category %q not found", s, c.Name)
			require.NotEmpty(t, content, "script %q is empty", s)
		}
	}
}

func TestGetProfileContentInvalid(t *testing.T) {
	_, err := GetProfileContent("nvidia-security-baseline", "profiles/nonexistent.xml")
	require.Error(t, err)
}

func TestGetProfileContentValid(t *testing.T) {
	content, err := GetProfileContent("nvidia-security-baseline", "profiles/firewall.xml")
	require.NoError(t, err)
	require.Contains(t, string(content), "Firewall")
	require.Contains(t, string(content), "LocURI")
}

func TestAllProfilesPassFleetValidation(t *testing.T) {
	manifest, err := GetBaseline("nvidia-security-baseline")
	require.NoError(t, err)

	for _, c := range manifest.Categories {
		for _, p := range c.Profiles {
			t.Run(c.Name+"/"+p, func(t *testing.T) {
				content, err := GetProfileContent(manifest.ID, p)
				require.NoError(t, err)

				// Build a profile name using the baseline naming convention.
				base := filepath.Base(p)
				profileName := "[NVIDIA Baseline] " + c.Name + " - " + strings.TrimSuffix(base, filepath.Ext(base))

				profile := &fleet.MDMWindowsConfigProfile{
					Name:   profileName,
					SyncML: content,
				}
				err = profile.ValidateUserProvided(false)
				require.NoError(t, err, "profile %s failed Fleet validation", p)
			})
		}
	}
}

func TestServiceConfigurationCategoryHasNoProfiles(t *testing.T) {
	manifest, err := GetBaseline("nvidia-security-baseline")
	require.NoError(t, err)

	for _, c := range manifest.Categories {
		if c.Name == "Service Configuration" {
			require.Empty(t, c.Profiles, "Service Configuration should use scripts, not profiles")
			require.NotEmpty(t, c.Scripts, "Service Configuration must have remediation scripts")
			return
		}
	}
	t.Fatal("Service Configuration category not found")
}
