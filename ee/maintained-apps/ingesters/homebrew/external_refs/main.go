package externalrefs

import (
	"fmt"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
)

var Funcs = map[string][]func(*maintained_apps.FMAManifestApp) (*maintained_apps.FMAManifestApp, error){
	"microsoft-word/darwin":         {MicrosoftVersionFromReleaseNotes},
	"microsoft-excel/darwin":        {MicrosoftVersionFromReleaseNotes},
	"brave-browser/darwin":          {BraveVersionTransformer},
	"whatsapp/darwin":               {WhatsAppVersionShortener},
	"google-chrome/darwin":          {ChromePKGInstaller},
	"1password/darwin":              {OnePasswordPKGInstaller},
	"zoom/darwin":                   {ZoomPKGInstaller},
	"slack/darwin":                  {SlackPKGInstaller},
	"omnissa-horizon-client/darwin": {OmnissaHorizonVersionShortener},
	"8x8-work/darwin":               {EightXEightWorkVersionShortener},
	"cisco-jabber/darwin":           {CiscoJabberVersionTransformer},
	"parallels/darwin":              {ParallelsVersionShortener},
	"github/darwin":                 {GitHubDesktopVersionShortener},
	"camtasia/darwin":               {CamtasiaVersionTransformer},
	"warp/darwin":                   {WarpHomebrewHeaders, WarpVersionTransformer},
}

func ChromePKGInstaller(app *maintained_apps.FMAManifestApp) (*maintained_apps.FMAManifestApp, error) {
	// Override installer URL to use Google's PKG installer instead of Homebrew's DMG
	// Version is kept from Homebrew (not set to "latest")
	app.InstallerURL = "https://dl.google.com/dl/chrome/mac/universal/stable/gcem/GoogleChrome.pkg"
	// Set SHA256 to "no_check" since we're using a different installer URL than Homebrew
	app.SHA256 = "no_check"

	return app, nil
}

func OnePasswordPKGInstaller(app *maintained_apps.FMAManifestApp) (*maintained_apps.FMAManifestApp, error) {
	// Override installer URL to use 1Password's Universal PKG installer instead of Homebrew's DMG
	// Version is kept from Homebrew (not set to "latest")
	app.InstallerURL = "https://downloads.1password.com/mac/1Password.pkg"
	// Set SHA256 to "no_check" since we're using a different installer URL than Homebrew
	app.SHA256 = "no_check"

	return app, nil
}

func SlackPKGInstaller(app *maintained_apps.FMAManifestApp) (*maintained_apps.FMAManifestApp, error) {
	// Override installer URL to use Slack's Universal PKG installer instead of Homebrew's DMG
	// Version is kept from Homebrew (not set to "latest")
	app.InstallerURL = "https://slack.com/api/desktop.latestRelease?redirect=1&variant=pkg&arch=universal"
	// Set SHA256 to "no_check" since we're using a different installer URL than Homebrew
	app.SHA256 = "no_check"

	return app, nil
}

func ZoomPKGInstaller(app *maintained_apps.FMAManifestApp) (*maintained_apps.FMAManifestApp, error) {
	// Override installer URL to use Zoom's Universal PKG installer instead of Homebrew's DMG
	// Version is kept from Homebrew (not set to "latest")
	app.InstallerURL = "https://zoom.us/client/latest/ZoomInstallerIT.pkg"
	// Set SHA256 to "no_check" since we're using a different installer URL than Homebrew
	app.SHA256 = "no_check"

	return app, nil
}

func EnrichManifest(app *maintained_apps.FMAManifestApp) {
	// Enrich the app manifest with additional metadata
	if enrichers, ok := Funcs[app.Slug]; ok {
		for _, enricher := range enrichers {
			var err error
			app, err = enricher(app)
			if err != nil {
				fmt.Printf("Error enriching app %s: %v\n", app.UniqueIdentifier, err)
			}
		}
	}
}
