package externalrefs

import (
	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
)

// CamtasiaVersionTransformer prepends "20" to the version to match what osquery reports.
// Homebrew reports "26.0.2" but osquery reports "2026.0.2" (year-based versioning).
// This ensures the output always contains the real version that osquery reports.
func CamtasiaVersionTransformer(app *maintained_apps.FMAManifestApp) (*maintained_apps.FMAManifestApp, error) {
	app.Version = "20" + app.Version
	return app, nil
}
