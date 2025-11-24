package externalrefs

import (
	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
)

// OneDriveVersionTransformer sets the version to "latest" so that the validation
// extracts the actual app version from the installed app, which matches what osquery reports.
// OneDrive auto-updates, so the installer URL version may not match the installed version.
func OneDriveVersionTransformer(app *maintained_apps.FMAManifestApp) (*maintained_apps.FMAManifestApp, error) {
	app.Version = "latest"
	return app, nil
}

