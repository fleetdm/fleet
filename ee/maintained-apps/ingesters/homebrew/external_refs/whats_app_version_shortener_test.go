package externalrefs

import (
	"testing"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
	"github.com/stretchr/testify/assert"
)

func TestWhatsAppVersionShortener(t *testing.T) {
	t.Run("successful version found", func(t *testing.T) {
		app := &maintained_apps.FMAManifestApp{
			UniqueIdentifier: "whatsapp/darwin",
			Version:          "2.25.16.81",
		}
		result, err := WhatsAppVersionShortener(app)
		assert.NoError(t, err)
		assert.Equal(t, "25.16.81", result.Version)
	})

	t.Run("new version scheme without 2. prefix", func(t *testing.T) {
		app := &maintained_apps.FMAManifestApp{
			UniqueIdentifier: "whatsapp/darwin",
			Version:          "26.26.12",
		}
		result, err := WhatsAppVersionShortener(app)
		assert.NoError(t, err)
		assert.Equal(t, "26.26.12", result.Version)
	})
}
