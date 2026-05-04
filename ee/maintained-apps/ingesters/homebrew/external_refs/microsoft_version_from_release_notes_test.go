package externalrefs

import (
	"testing"
	"time"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
	macoffice "github.com/fleetdm/fleet/v4/server/vulnerabilities/macoffice"
	"github.com/tj/assert"
)

func TestMicrosoftVersionFromReleaseNotes(t *testing.T) {
	releaseNotesCache = macoffice.ReleaseNotes{
		{
			Date:            time.Date(2025, 6, 27, 0, 0, 0, 0, time.UTC),
			Version:         "Version 16.98.3 (Build 25062733)",
			SecurityUpdates: nil,
		},
		{
			Date:            time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC),
			Version:         "Version 16.98.1 (Build 25061520)",
			SecurityUpdates: nil,
		},
		{
			Date:    time.Date(2025, 6, 8, 0, 0, 0, 0, time.UTC),
			Version: "Version 16.98 (Build 25060824)",
			SecurityUpdates: []macoffice.SecurityUpdate{
				{Product: macoffice.Word, Vulnerability: "CVE-2025-0008"},
				{Product: macoffice.Excel, Vulnerability: "CVE-2025-0009"},
				{Product: macoffice.PowerPoint, Vulnerability: "CVE-2025-0010"},
			},
		},
	}
	cacheOnce.Do(func() {})

	t.Run("successful version lookup", func(t *testing.T) {
		app := &maintained_apps.FMAManifestApp{
			UniqueIdentifier: "microsoft-word/darwin",
			Version:          "16.98.25062733",
		}
		result, err := MicrosoftVersionFromReleaseNotes(app)
		assert.NoError(t, err)
		assert.Equal(t, "16.98.3", result.Version)
	})

	t.Run("version not found falls back to base version", func(t *testing.T) {
		// Build number "25999999" is not in the release notes cache above.
		// The function should fall back to the base major.minor version ("16.50")
		// rather than leaving the raw Homebrew build string, which would cause a
		// perpetual "update available" loop against osquery's short version reporting.
		app := &maintained_apps.FMAManifestApp{
			UniqueIdentifier: "microsoft-excel/darwin",
			Version:          "16.50.25999999",
		}
		result, err := MicrosoftVersionFromReleaseNotes(app)
		assert.NoError(t, err)
		assert.Equal(t, "16.50", result.Version)
	})

	t.Run("version with fewer than 3 segments is returned unchanged", func(t *testing.T) {
		// Versions with fewer than 3 segments have no build number to look up;
		// the function should return the version unchanged.
		app := &maintained_apps.FMAManifestApp{
			UniqueIdentifier: "microsoft-onenote/darwin",
			Version:          "16.106",
		}
		result, err := MicrosoftVersionFromReleaseNotes(app)
		assert.NoError(t, err)
		assert.Equal(t, "16.106", result.Version)
	})
}
