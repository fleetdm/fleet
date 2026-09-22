package externalrefs

import (
	"testing"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDatadogAgentVersionFixer(t *testing.T) {
	t.Run("corrects winget version to the real MSI ProductVersion", func(t *testing.T) {
		app := &maintained_apps.FMAManifestApp{
			UniqueIdentifier: "Datadog Agent",
			Slug:             "datadog-agent/windows",
			Version:          "7.81.2.1",
		}
		result, err := DatadogAgentVersionFixer(app)
		require.NoError(t, err)
		assert.Equal(t, "7.81.2.0", result.Version)
	})

	t.Run("errors when the winget-reported version no longer matches", func(t *testing.T) {
		app := &maintained_apps.FMAManifestApp{
			UniqueIdentifier: "Datadog Agent",
			Slug:             "datadog-agent/windows",
			Version:          "7.82.0.0",
		}
		result, err := DatadogAgentVersionFixer(app)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "expected Datadog Agent winget version to be '7.81.2.1' but found '7.82.0.0'")
		assert.Equal(t, "7.82.0.0", result.Version) // Version unchanged on error
	})
}
