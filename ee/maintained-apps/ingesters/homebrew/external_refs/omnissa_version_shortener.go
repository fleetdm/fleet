package externalrefs

import (
	"fmt"
	"strings"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
)

func OmnissaHorizonVersionShortener(app *maintained_apps.FMAManifestApp) (*maintained_apps.FMAManifestApp, error) {
	homebrewVersion := app.Version

	parts := strings.Split(homebrewVersion, "-")
	if len(parts) == 3 {
		app.Version = parts[1]
	} else {
		return app, fmt.Errorf("Expected Omnissa Horizon Client version to match XXXX-0.00.0-XXXXXXXXXXX but found '%s'", homebrewVersion)
	}

	return app, nil
}
