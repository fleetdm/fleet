package externalrefs

import (
	"strings"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
)

func WhatsAppVersionShortener(app *maintained_apps.FMAManifestApp) (*maintained_apps.FMAManifestApp, error) {
	homebrewVersion := app.Version

	// Legacy WhatsApp versions used a "2." prefix added by Homebrew that
	// needed to be stripped (e.g. "2.25.16.81" -> "25.16.81").
	// Newer versions (e.g. "26.26.12") no longer carry this prefix.
	if strings.HasPrefix(homebrewVersion, "2.") {
		app.Version = strings.TrimPrefix(homebrewVersion, "2.")
	}

	return app, nil
}
