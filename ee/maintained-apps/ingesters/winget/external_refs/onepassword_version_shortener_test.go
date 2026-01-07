package externalrefs

import (
	"testing"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
	"github.com/stretchr/testify/assert"
)

func TestOnePasswordVersionShortener(t *testing.T) {
	t.Run("successful version transformation from 4 parts to 3 parts", func(t *testing.T) {
		app := &maintained_apps.FMAManifestApp{
			UniqueIdentifier: "1Password",
			Slug:             "1password/windows",
			Version:          "8.11.18.36",
		}
		result, err := OnePasswordVersionShortener(app)
		assert.NoError(t, err)
		assert.Equal(t, "8.11.18", result.Version)
	})

	t.Run("handles different 4-part versions", func(t *testing.T) {
		app := &maintained_apps.FMAManifestApp{
			UniqueIdentifier: "1Password",
			Slug:             "1password/windows",
			Version:          "9.0.1.123",
		}
		result, err := OnePasswordVersionShortener(app)
		assert.NoError(t, err)
		assert.Equal(t, "9.0.1", result.Version)
	})

	t.Run("error when version has only 3 parts", func(t *testing.T) {
		app := &maintained_apps.FMAManifestApp{
			UniqueIdentifier: "1Password",
			Slug:             "1password/windows",
			Version:          "8.11.18",
		}
		result, err := OnePasswordVersionShortener(app)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "expected 1Password version to have 4 parts but found 3 parts")
		assert.Equal(t, "8.11.18", result.Version) // Version unchanged on error
	})

	t.Run("error when version has 2 parts", func(t *testing.T) {
		app := &maintained_apps.FMAManifestApp{
			UniqueIdentifier: "1Password",
			Slug:             "1password/windows",
			Version:          "8.11",
		}
		result, err := OnePasswordVersionShortener(app)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "expected 1Password version to have 4 parts but found 2 parts")
		assert.Equal(t, "8.11", result.Version)
	})

	t.Run("error when version has 5 parts", func(t *testing.T) {
		app := &maintained_apps.FMAManifestApp{
			UniqueIdentifier: "1Password",
			Slug:             "1password/windows",
			Version:          "8.11.18.36.1",
		}
		result, err := OnePasswordVersionShortener(app)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "expected 1Password version to have 4 parts but found 5 parts")
		assert.Equal(t, "8.11.18.36.1", result.Version)
	})
}
