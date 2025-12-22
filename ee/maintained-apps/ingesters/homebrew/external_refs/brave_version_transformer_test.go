package externalrefs

import (
	"testing"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
	"github.com/stretchr/testify/assert"
)

func TestBraveVersionTransformer(t *testing.T) {
	t.Run("Version does not end with .0", func(t *testing.T) {
		app := &maintained_apps.FMAManifestApp{
			UniqueIdentifier: "brave-browser/darwin",
			Version:          "1.79.123",
		}
		result, err := BraveVersionTransformer(app)
		assert.Error(t, err)
		assert.Equal(t, "Expected Brave version to end with '.0' but found '1.79.123'", err.Error())
		assert.Equal(t, "1.79.123", result.Version)
	})

	t.Run("version has less than three parts after dropping .0", func(t *testing.T) {
		app := &maintained_apps.FMAManifestApp{
			UniqueIdentifier: "brave-browser/darwin",
			Version:          "1.79.0",
		}
		result, err := BraveVersionTransformer(app)
		assert.Error(t, err)
		assert.Equal(t, "Expected Brave version to have four parts but found '1.79.0'", err.Error())
		assert.Equal(t, "1.79.0", result.Version)
	})

	t.Run("version has more than three parts after dropping .0", func(t *testing.T) {
		app := &maintained_apps.FMAManifestApp{
			UniqueIdentifier: "brave-browser/darwin",
			Version:          "1.1.1.79.0",
		}
		result, err := BraveVersionTransformer(app)
		assert.Error(t, err)
		assert.Equal(t, "Expected Brave version to have four parts but found '1.1.1.79.0'", err.Error())
		assert.Equal(t, "1.1.1.79.0", result.Version)
	})

	t.Run("cannot parse second part as integer", func(t *testing.T) {
		app := &maintained_apps.FMAManifestApp{
			UniqueIdentifier: "brave-browser/darwin",
			Version:          "1.a79.123.0",
		}
		result, err := BraveVersionTransformer(app)
		assert.Error(t, err)
		assert.Equal(t, "Failed to parse 'a79' of Brave version '1.a79.123.0': strconv.Atoi: parsing \"a79\": invalid syntax", err.Error())
		assert.Equal(t, "1.a79.123.0", result.Version)
	})

	t.Run("Valid Brave Version", func(t *testing.T) {
		app := &maintained_apps.FMAManifestApp{
			UniqueIdentifier: "brave-browser/darwin",
			Version:          "1.79.123.0",
		}
		result, err := BraveVersionTransformer(app)
		assert.NoError(t, err)
		assert.Equal(t, "137.1.79.123", result.Version)
	})
}
