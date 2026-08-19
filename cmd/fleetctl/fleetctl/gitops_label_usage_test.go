package fleetctl

import (
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/pkg/spec"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeProfileConfig(t *testing.T, profiles []fleet.MDMProfileSpec, osName string) *spec.GitOps {
	t.Helper()
	config := &spec.GitOps{}
	switch osName {
	case "macos":
		config.Controls.MacOSSettings = &fleet.MacOSSettings{CustomSettings: profiles}
	case "windows":
		config.Controls.WindowsSettings = &fleet.WindowsSettings{CustomSettings: optjson.SetSlice(profiles)}
	default:
		t.Fatalf("unknown os: %s", osName)
	}
	return config
}

func TestGetLabelUsageProfilePathShortened(t *testing.T) {
	// Simulate what happens when spec parsing resolves a relative path to absolute.
	absPath := "/home/runner/work/Detroit-GitOps-Workshop/Detroit-GitOps-Workshop/lib/macos/configuration-profiles/disable-bluetooth-file-sharing.mobileconfig"
	profiles := []fleet.MDMProfileSpec{{Path: absPath, LabelsIncludeAll: []string{"nonexistent-label"}}}

	for _, osName := range []string{"macos", "windows"} {
		t.Run(osName, func(t *testing.T) {
			usage, err := getLabelUsage(makeProfileConfig(t, profiles, osName))
			require.NoError(t, err)

			// The label "nonexistent-label" should be in the usage map.
			entries, ok := usage["nonexistent-label"]
			require.True(t, ok, "expected label to be in usage map")
			require.Len(t, entries, 1)

			// The Name should be the base filename, not the full absolute path.
			assert.Equal(t, "disable-bluetooth-file-sharing.mobileconfig", entries[0].Name,
				"profile path should be shortened to just the filename")

			// The Type should be "configuration profile", not "MDM Profile".
			assert.Equal(t, "configuration profile", entries[0].Type,
				"type should say 'configuration profile' not 'MDM Profile'")
		})
	}
}

func TestGetLabelUsageMultipleLabelKeysError(t *testing.T) {
	absPath := "/absolute/path/to/profile.mobileconfig"
	profiles := []fleet.MDMProfileSpec{{
		Path:             absPath,
		LabelsIncludeAll: []string{"label-a"},
		LabelsIncludeAny: []string{"label-b"},
	}}

	// Two include modes together is still invalid.
	for _, osName := range []string{"macos", "windows"} {
		t.Run(osName, func(t *testing.T) {
			_, err := getLabelUsage(makeProfileConfig(t, profiles, osName))
			require.Error(t, err)

			// Error should use "configuration profile" and the short filename.
			assert.Contains(t, err.Error(), "configuration profile")
			assert.Contains(t, err.Error(), "profile.mobileconfig")
			assert.NotContains(t, err.Error(), "/absolute/path/to/")
		})
	}
}

func TestGetLabelUsageIncludeExcludeOverlapError(t *testing.T) {
	absPath := "/absolute/path/to/profile.mobileconfig"

	// A label appearing in both include and exclude is rejected before any apply.
	cases := []fleet.MDMProfileSpec{
		{Path: absPath, LabelsIncludeAll: []string{"shared-label"}, LabelsExcludeAny: []string{"shared-label"}},
		{Path: absPath, LabelsIncludeAny: []string{"shared-label"}, LabelsExcludeAny: []string{"shared-label"}},
		{Path: absPath, LabelsIncludeAll: []string{"ok-label", "shared-label"}, LabelsExcludeAny: []string{"shared-label", "other-label"}},
	}
	for _, profileSpec := range cases {
		for _, osName := range []string{"macos", "windows"} {
			t.Run(osName, func(t *testing.T) {
				_, err := getLabelUsage(makeProfileConfig(t, []fleet.MDMProfileSpec{profileSpec}, osName))
				require.Error(t, err)
				assert.Contains(t, err.Error(), "shared-label")
				assert.Contains(t, err.Error(), "profile.mobileconfig")
			})
		}
	}
}

func TestGetLabelUsageIncludeAndExcludeAllowed(t *testing.T) {
	absPath := "/absolute/path/to/profile.mobileconfig"

	// include + exclude combination is valid for configuration profiles.
	cases := []fleet.MDMProfileSpec{
		{Path: absPath, LabelsIncludeAll: []string{"include-label"}, LabelsExcludeAny: []string{"exclude-label"}},
		{Path: absPath, LabelsIncludeAny: []string{"include-label"}, LabelsExcludeAny: []string{"exclude-label"}},
		{Path: absPath, LabelsExcludeAny: []string{"exclude-label"}},
	}
	for _, profileSpec := range cases {
		for _, osName := range []string{"macos", "windows"} {
			t.Run(osName, func(t *testing.T) {
				usage, err := getLabelUsage(makeProfileConfig(t, []fleet.MDMProfileSpec{profileSpec}, osName))
				require.NoError(t, err)
				allLabels := append(profileSpec.LabelsIncludeAll, append(profileSpec.LabelsIncludeAny, profileSpec.LabelsExcludeAny...)...) //nolint:gocritic
				for _, label := range allLabels {
					assert.Contains(t, usage, label)
				}
			})
		}
	}
}
