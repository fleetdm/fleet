package externalrefs

import (
	"testing"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
	"github.com/stretchr/testify/assert"
)

func TestOmnissaVersionShortener(t *testing.T) {
	t.Run("successful version found", func(t *testing.T) {
		app := &maintained_apps.FMAManifestApp{
			UniqueIdentifier: "omnissa-horizon-client/darwin",
			Version:          "2506-8.16.0-16536825094",
		}
		result, err := OmnissaHorizonVersionShortener(app)
		assert.NoError(t, err)
		assert.Equal(t, "8.16.0", result.Version)
	})

	t.Run("unexpected version format", func(t *testing.T) {
		app := &maintained_apps.FMAManifestApp{
			UniqueIdentifier: "omnissa-horizon-client/darwin",
			Version:          "8.16.0-wrong",
		}
		result, err := OmnissaHorizonVersionShortener(app)
		assert.Error(t, err)
		assert.Equal(t, "Expected Omnissa Horizon Client version to match XXXX-0.00.0-XXXXXXXXXXX but found '8.16.0-wrong'", err.Error())
		assert.Equal(t, "8.16.0-wrong", result.Version)
	})

	t.Run("correct version format unchanged", func(t *testing.T) {
		app := &maintained_apps.FMAManifestApp{
			UniqueIdentifier: "omnissa-horizon-client/darwin",
			Version:          "8.16.0",
		}
		result, err := OmnissaHorizonVersionShortener(app)
		assert.Error(t, err)
		assert.Equal(t, "Expected Omnissa Horizon Client version to match XXXX-0.00.0-XXXXXXXXXXX but found '8.16.0'", err.Error())
		assert.Equal(t, "8.16.0", result.Version)
	})
}
