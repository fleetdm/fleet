package externalrefs

import (
	"regexp"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
)

var postgresAppBundleIDRe = regexp.MustCompile(`com\.postgresapp\.Postgres\d+`)

// PostgresAppBundleIDs fixes the uninstall script's bundle IDs. The cask derives
// them from the major version (com.postgresapp.Postgres3 for 3.x), but Postgres.app
// 3.x still ships com.postgresapp.Postgres2 and com.postgresapp.Postgres2LoginHelper.
func PostgresAppBundleIDs(app *maintained_apps.FMAManifestApp) (*maintained_apps.FMAManifestApp, error) {
	app.UninstallScript = postgresAppBundleIDRe.ReplaceAllString(app.UninstallScript, app.UniqueIdentifier)
	app.UninstallScriptRef = maintained_apps.GetScriptRef(app.UninstallScript)
	return app, nil
}
