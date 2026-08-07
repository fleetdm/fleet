package externalrefs

import (
	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
)

func VivaldiDMGInstaller(app *maintained_apps.FMAManifestApp) (*maintained_apps.FMAManifestApp, error) {
	// Override installer URL to use Vivaldi's direct DMG instead of Homebrew's tar.xz
	// The tar.xz format is not supported by the Fleet ingester's install script generator.
	// Version is kept from Homebrew (not set to "latest")
	app.InstallerURL = "https://downloads.vivaldi.com/stable/Vivaldi." + app.Version + ".universal.dmg"
	// Set SHA256 to "no_check" since we're using a different installer URL than Homebrew
	app.SHA256 = "no_check"

	return app, nil
}
