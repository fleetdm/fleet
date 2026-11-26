package spec

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGitOpsEmptySoftwarePackageListItem(t *testing.T) {
	t.Parallel()

	config := getTeamConfig([]string{"software"})
	config += "\nsoftware:\n  packages:\n    - path: app.yml\n"
	path, basePath := createTempFile(t, "", config)
	require.NoError(t, os.WriteFile(filepath.Join(basePath, "app.yml"),
		[]byte("- url: https://example.com/app.pkg\n- # url: https://example.com/old.pkg\n"), 0o600))

	// Software is only parsed with a premium license.
	appConfig := fleet.EnrichedAppConfig{License: &fleet.LicenseInfo{Tier: fleet.TierPremium}}

	var err error
	require.NotPanics(t, func() {
		_, err = GitOpsFromFile(path, basePath, &appConfig, nopLogf)
	})

	assert.ErrorContains(t, err, `software package file "app.yml": list item 2 is empty`)
}
