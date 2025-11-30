package externalrefs

import (
	"regexp"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
)

// CamtasiaVersionTransformer transforms versions like "26.0.2" to "2026.0.2" to match
// what osquery reports. Camtasia uses year-based versioning where "26" represents "2026".
func CamtasiaVersionTransformer(app *maintained_apps.FMAManifestApp) (*maintained_apps.FMAManifestApp, error) {
	// Check if version starts with a 2-digit number followed by a dot (e.g., "26.0.2")
	matched, _ := regexp.MatchString(`^\d{2}\.`, app.Version)
	if matched {
		// Prepend "20" to the version (e.g., "26.0.2" -> "2026.0.2")
		app.Version = "20" + app.Version
	}
	return app, nil
}

