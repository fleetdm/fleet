package externalrefs

import (
	"strings"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
)

// UPDFInstallerHost swaps the cask's download host: download.updf.com serves a
// Cloudflare challenge to non-browser clients, while the vendor's
// download.superace.com serves the same path without one.
func UPDFInstallerHost(app *maintained_apps.FMAManifestApp) (*maintained_apps.FMAManifestApp, error) {
	app.InstallerURL = strings.Replace(app.InstallerURL, "//download.updf.com/", "//download.superace.com/", 1)
	app.SHA256 = "no_check"

	return app, nil
}
