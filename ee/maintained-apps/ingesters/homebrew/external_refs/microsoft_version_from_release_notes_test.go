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

	t.Run("version not found", func(t *testing.T) {
		app := &maintained_apps.FMAManifestApp{
			UniqueIdentifier: "microsoft-excel/darwin",
			Version:          "16.50.25999999",
		}
		result, err := MicrosoftVersionFromReleaseNotes(app)
		assert.Error(t, err)
		assert.Equal(t, "16.50.25999999", result.Version)
	})
}
