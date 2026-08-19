package externalrefs

import (
	"regexp"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
)

// yearPrefixedCamtasiaVersion matches bundle_short_version values osquery reports
// for recent Camtasia releases (e.g. 2026.1.0). Homebrew historically reported
// two-digit year segments (26.1.0); newer cask metadata uses the full year
// (2026.1.0). Only prepend "20" when the version is not already year-prefixed.
var yearPrefixedCamtasiaVersion = regexp.MustCompile(`^20\d{2}\.`)

// CamtasiaVersionTransformer prepends "20" to the version to match what osquery reports.
// Homebrew reports "26.0.2" but osquery reports "2026.0.2" (year-based versioning).
// When Homebrew already reports "2026.1.0", leave the version unchanged.
func CamtasiaVersionTransformer(app *maintained_apps.FMAManifestApp) (*maintained_apps.FMAManifestApp, error) {
	if yearPrefixedCamtasiaVersion.MatchString(app.Version) {
		return app, nil
	}
	app.Version = "20" + app.Version
	return app, nil
}
