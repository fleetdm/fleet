package externalrefs

import (
	"strings"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
)

func GoogleDriveVersionShortener(app *maintained_apps.FMAManifestApp) (*maintained_apps.FMAManifestApp, error) {
	homebrewVersion := app.Version

	// If the google drive version has three or more places, remove all but two
	// https://github.com/fleetdm/fleet/issues/40751
	if strings.Count(homebrewVersion, ".") > 1 {
		split := strings.Split(homebrewVersion, ".")
		shortVersion := strings.Join(split[0:2], ".")
		app.Version = shortVersion
	}

	return app, nil
}
