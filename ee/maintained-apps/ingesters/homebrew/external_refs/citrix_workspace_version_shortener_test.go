package externalrefs

import (
	"testing"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
	"github.com/stretchr/testify/assert"
)

func TestCitrixWorkspaceVersionShortener(t *testing.T) {
	t.Run("successful version found", func(t *testing.T) {
		app := &maintained_apps.FMAManifestApp{
			UniqueIdentifier: "citrix-workspace/darwin",
			Version:          "25.08.10.31",
		}
		result, err := CitrixWorkspaceVersionShortener(app)
		assert.NoError(t, err)
		assert.Equal(t, "25.08.10", result.Version)
	})

	t.Run("unexpected version format - too few segments", func(t *testing.T) {
		app := &maintained_apps.FMAManifestApp{
			UniqueIdentifier: "citrix-workspace/darwin",
			Version:          "25.08.10",
		}
		result, err := CitrixWorkspaceVersionShortener(app)
		assert.Error(t, err)
		assert.Equal(t, "Expected Citrix Workspace version to match XX.XX.XX.XX but found '25.08.10'", err.Error())
		assert.Equal(t, "25.08.10", result.Version)
	})

	t.Run("unexpected version format - too many segments", func(t *testing.T) {
		app := &maintained_apps.FMAManifestApp{
			UniqueIdentifier: "citrix-workspace/darwin",
			Version:          "25.08.10.31.5",
		}
		result, err := CitrixWorkspaceVersionShortener(app)
		assert.Error(t, err)
		assert.Equal(t, "Expected Citrix Workspace version to match XX.XX.XX.XX but found '25.08.10.31.5'", err.Error())
		assert.Equal(t, "25.08.10.31.5", result.Version)
	})
}
