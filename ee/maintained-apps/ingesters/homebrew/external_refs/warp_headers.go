package externalrefs

import (
	"strings"

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

func WarpVersionTransformer(app *maintained_apps.FMAManifestApp) (*maintained_apps.FMAManifestApp, error) {
	// Warp's homebrew cask version includes ".stable_" but the actual app bundle doesn't
	// Example: 0.2025.12.17.17.17.stable_02 -> 0.2025.12.17.17.17.02
	app.Version = strings.Replace(app.Version, ".stable_", ".", 1)
	return app, nil
}
