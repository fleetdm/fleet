package externalrefs

import (
	"fmt"
	"regexp"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
)

var hoppscotchVersionRe = regexp.MustCompile(`^(\d+\.\d+\.\d+)(-\d+)?$`)

// HoppscotchVersionShortener drops the release revision from the winget version,
// which the MSI's ProductVersion doesn't carry.
//
// Winget reports: "26.11.0-0" (the release tag)
// Installer registers: "26.11.0"
//
// osquery's version_compare sorts "26.11.0" below "26.11.0-0", so without this
// the patch policy fails on hosts that already have the current version.
func HoppscotchVersionShortener(app *maintained_apps.FMAManifestApp) (*maintained_apps.FMAManifestApp, error) {
	m := hoppscotchVersionRe.FindStringSubmatch(app.Version)
	if m == nil {
		return app, fmt.Errorf("expected Hoppscotch version like '26.11.0-0' but found '%s'", app.Version)
	}
	app.Version = m[1]
	return app, nil
}
