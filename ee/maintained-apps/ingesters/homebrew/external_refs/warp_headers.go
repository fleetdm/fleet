package externalrefs

import (
	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
)

func WarpHomebrewHeaders(app *maintained_apps.FMAManifestApp) (*maintained_apps.FMAManifestApp, error) {
	// Add Homebrew User-Agent header required by Warp's download endpoint
	// Warp's server returns HTML without this header instead of the DMG binary
	if app.Headers == nil {
		app.Headers = make(map[string]string)
	}
	app.Headers["User-Agent"] = "Homebrew"
	return app, nil
}
