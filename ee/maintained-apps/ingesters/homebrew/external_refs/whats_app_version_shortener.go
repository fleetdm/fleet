package externalrefs

import (
	"fmt"
	"strings"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
)

func WhatsAppVersionShortener(app *maintained_apps.FMAManifestApp) (*maintained_apps.FMAManifestApp, error) {
	homebrewVersion := app.Version

	if strings.HasPrefix(homebrewVersion, "2.") {
		app.Version = strings.TrimPrefix(homebrewVersion, "2.")
	} else {
		return app, fmt.Errorf("Expected WhatsApp version to start with '2.' but found '%s'", homebrewVersion)
	}

	return app, nil
}
