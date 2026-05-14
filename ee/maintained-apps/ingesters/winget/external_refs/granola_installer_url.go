package externalrefs

import (
	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
)

// GranolaWindowsInstallerURL overrides the installer URL for Granola on Windows
// to always use Granola's "download-latest-windows" endpoint instead of the
// versioned URL surfaced by the winget manifest.
//
// Version is kept from winget (not set to "latest") so patch policies still work.
func GranolaWindowsInstallerURL(app *maintained_apps.FMAManifestApp) (*maintained_apps.FMAManifestApp, error) {
	app.InstallerURL = "https://api.granola.ai/v1/download-latest-windows"
	// The URL is not pinned to a version, so we can't validate the hash.
	app.SHA256 = "no_check"

	return app, nil
}
