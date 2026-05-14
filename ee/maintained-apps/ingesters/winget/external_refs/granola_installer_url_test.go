package externalrefs

import (
	"testing"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
	"github.com/stretchr/testify/assert"
)

func TestGranolaWindowsInstallerURL(t *testing.T) {
	t.Run("overrides installer URL and sha256, preserves version", func(t *testing.T) {
		app := &maintained_apps.FMAManifestApp{
			UniqueIdentifier: "Granola",
			Slug:             "granola/windows",
			Version:          "7.205.1",
			InstallerURL:     "https://api.granola.ai/v1/check-for-update/Granola-7.205.1-win-x64.exe",
			SHA256:           "c1b34c35fe83cc5852a2bdc9f079029368ed19be49ebffff2a7c965523ebe9cd",
		}
		result, err := GranolaWindowsInstallerURL(app)
		assert.NoError(t, err)
		assert.Equal(t, "https://api.granola.ai/v1/download-latest-windows", result.InstallerURL)
		assert.Equal(t, "no_check", result.SHA256)
		assert.Equal(t, "7.205.1", result.Version)
	})
}
