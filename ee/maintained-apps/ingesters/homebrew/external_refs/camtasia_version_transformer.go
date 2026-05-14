package externalrefs

import (
	"regexp"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
)

// camtasiaYearPrefixed matches versions Homebrew already reports with the full
// 4-digit year prefix (e.g., "2026.1.0"). Used to keep this transformer
// idempotent if Homebrew's version format includes the year.
var camtasiaYearPrefixed = regexp.MustCompile(`^20\d{2}\.`)

// CamtasiaVersionTransformer prepends "20" to the version to match what osquery reports.
// Older Homebrew releases used a 2-digit year (e.g., "26.0.2") while osquery reports
// "2026.0.2" (year-based versioning). Recent Homebrew releases include the full year
// (e.g., "2026.1.0"); in that case this transformer is a no-op so the version doesn't
// become "202026.1.0".
func CamtasiaVersionTransformer(app *maintained_apps.FMAManifestApp) (*maintained_apps.FMAManifestApp, error) {
	if camtasiaYearPrefixed.MatchString(app.Version) {
		return app, nil
	}
	app.Version = "20" + app.Version
	return app, nil
}
