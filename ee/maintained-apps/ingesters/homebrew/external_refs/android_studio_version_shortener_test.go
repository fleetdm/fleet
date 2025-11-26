package externalrefs

import (
	"testing"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
	"github.com/stretchr/testify/require"
)

func TestAndroidStudioVersionShortener(t *testing.T) {
	t.Run("Success with 4-part version", func(t *testing.T) {
		app := &maintained_apps.FMAManifestApp{
			Version: "2025.2.1.8",
		}
		result, err := AndroidStudioVersionShortener(app)
		require.NoError(t, err)
		require.Equal(t, "2025.2", result.Version)
	})

	t.Run("Success with 3-part version", func(t *testing.T) {
		app := &maintained_apps.FMAManifestApp{
			Version: "2025.2.1",
		}
		result, err := AndroidStudioVersionShortener(app)
		require.NoError(t, err)
		require.Equal(t, "2025.2", result.Version)
	})

	t.Run("Error on invalid version format", func(t *testing.T) {
		app := &maintained_apps.FMAManifestApp{
			Version: "1",
		}
		result, err := AndroidStudioVersionShortener(app)
		require.Error(t, err)
		require.Equal(t, app, result)
	})
}
