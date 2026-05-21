package externalrefs

import (
	"strings"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
)

func WarpDirectInstaller(app *maintained_apps.FMAManifestApp) (*maintained_apps.FMAManifestApp, error) {
	// Override the Homebrew URL (which requires a User-Agent: Homebrew header) with the direct CDN
	// URL. The Homebrew endpoint redirects to this URL, so the DMG is identical.
	// Example: version "0.2026.02.25.08.24.stable_01" ->
	//   https://releases.warp.dev/stable/v0.2026.02.25.08.24.stable_01/Warp.dmg
	app.InstallerURL = "https://releases.warp.dev/stable/v" + app.Version + "/Warp.dmg"
	app.SHA256 = "no_check"

	// Strip ".stable_" from the version string — the app bundle version omits it.
	// Example: "0.2026.02.25.08.24.stable_01" -> "0.2026.02.25.08.24.01"
	app.Version = strings.Replace(app.Version, ".stable_", ".", 1)

	return app, nil
}
