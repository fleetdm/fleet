package externalrefs

import (
	"fmt"
	"strings"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
)

// TelegramVersionShortener transforms the Telegram Desktop version from the
// winget manifest format to the version its Inno Setup installer registers in
// Windows Programs.
//
// Winget reports: "7.3.0"
// Installer registers: "7.3" (AppVersion is tdesktop's AppVersionStrSmall,
// which omits a zero patch part; 7.3.1 registers "7.3.1")
//
// osquery's version_compare sorts "7.3" below "7.3.0", so without this the
// patch policy fails on hosts that already have the current version.
func TelegramVersionShortener(app *maintained_apps.FMAManifestApp) (*maintained_apps.FMAManifestApp, error) {
	parts := strings.Split(app.Version, ".")
	switch len(parts) {
	case 2:
		return app, nil
	case 3:
		if parts[2] == "0" {
			app.Version = strings.Join(parts[:2], ".")
		}
		return app, nil
	default:
		return app, fmt.Errorf("expected Telegram version to have 2 or 3 parts but found %d parts in '%s'", len(parts), app.Version)
	}
}
