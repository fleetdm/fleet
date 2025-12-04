package externalrefs

import (
	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
)

// CiscoJabberVersionTransformer sets the version to "15.2.0" which matches what osquery reports.
// Homebrew reports a build number (e.g., "20251027035315") instead of the app version (e.g., "15.2.0").
func CiscoJabberVersionTransformer(app *maintained_apps.FMAManifestApp) (*maintained_apps.FMAManifestApp, error) {
	app.Version = "15.2.0"
	return app, nil
}
