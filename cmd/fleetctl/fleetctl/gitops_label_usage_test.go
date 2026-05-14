package fleetctl

import (
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/spec"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetLabelUsageProfilePathShortened(t *testing.T) {
	// Simulate what happens when spec parsing resolves a relative path to absolute.
	absPath := "/home/runner/work/Detroit-GitOps-Workshop/Detroit-GitOps-Workshop/lib/macos/configuration-profiles/disable-bluetooth-file-sharing.mobileconfig"

	config := &spec.GitOps{
		Controls: spec.GitOpsControls{
			MacOSSettings: &fleet.MacOSSettings{
				CustomSettings: []fleet.MDMProfileSpec{
					{
						Path:             absPath,
						LabelsIncludeAll: []string{"nonexistent-label"},
					},
				},
			},
		},
	}

	usage, err := getLabelUsage(config)
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
}

func TestGetLabelUsageMultipleLabelKeysError(t *testing.T) {
	absPath := "/absolute/path/to/profile.mobileconfig"

	config := &spec.GitOps{
		Controls: spec.GitOpsControls{
			MacOSSettings: &fleet.MacOSSettings{
				CustomSettings: []fleet.MDMProfileSpec{
					{
						Path:             absPath,
						LabelsIncludeAll: []string{"label-a"},
						LabelsIncludeAny: []string{"label-b"},
					},
				},
			},
		},
	}

	_, err := getLabelUsage(config)
	require.Error(t, err)

	// Error should use "configuration profile" and the short filename.
	assert.Contains(t, err.Error(), "configuration profile")
	assert.Contains(t, err.Error(), "profile.mobileconfig")
	assert.NotContains(t, err.Error(), "/absolute/path/to/")
}
