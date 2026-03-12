package service

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/mdm/baselines"
	"github.com/stretchr/testify/require"
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
