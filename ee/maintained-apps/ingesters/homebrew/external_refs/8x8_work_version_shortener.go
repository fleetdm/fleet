package externalrefs

import (
	"strings"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
)

func EightXEightWorkVersionShortener(app *maintained_apps.FMAManifestApp) (*maintained_apps.FMAManifestApp, error) {
	homebrewVersion := app.Version

	// Strip everything after the first '-' (e.g., "8.28.2-3" -> "8.28.2")
	parts := strings.Split(homebrewVersion, "-")
	if len(parts) > 1 {
		app.Version = parts[0]
	}
	// If no '-' is found, keep the version as-is

	return app, nil
}
