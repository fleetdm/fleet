package winoffice_test

import (
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/pkg/nettest"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/winoffice"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegrationCheckVersion(t *testing.T) {
	nettest.Run(t)

	client := fleethttp.NewClient(fleethttp.WithTimeout(60 * time.Second))

	bulletin, err := winoffice.FetchBulletin(client)
	require.NoError(t, err)
	require.NotEmpty(t, bulletin.Versions)
	require.NotEmpty(t, bulletin.BuildPrefixes)

	// Get the newest version's build prefix (highest version number)
	var newestVersion string
	for _, version := range bulletin.BuildPrefixes {
		if version > newestVersion {
			newestVersion = version
		}
	}
	var testPrefix string
	for prefix, version := range bulletin.BuildPrefixes {
		if version == newestVersion {
			testPrefix = prefix
			break
		}
	}

	t.Logf("Bulletin: %d versions", len(bulletin.Versions))

	t.Run("old version is vulnerable", func(t *testing.T) {
		vulns := winoffice.CheckVersion("16.0."+testPrefix+".10000", bulletin)
		assert.NotEmpty(t, vulns, "old build should have vulnerabilities")
		t.Logf("Version 16.0.%s.10000: %d CVEs", testPrefix, len(vulns))
	})

	t.Run("latest version is not vulnerable", func(t *testing.T) {
		vulns := winoffice.CheckVersion("16.0."+testPrefix+".99999", bulletin)
		assert.Empty(t, vulns, "latest build should have no vulnerabilities")
	})

	t.Run("unknown version returns no vulnerabilities", func(t *testing.T) {
		vulns := winoffice.CheckVersion("16.0.99999.99999", bulletin)
		assert.Empty(t, vulns, "unknown version should return empty")
	})

	t.Run("invalid version returns no vulnerabilities", func(t *testing.T) {
		vulns := winoffice.CheckVersion("invalid", bulletin)
		assert.Empty(t, vulns, "invalid version should return empty")
	})
}
