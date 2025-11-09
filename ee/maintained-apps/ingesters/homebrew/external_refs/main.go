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
	"omnissa-horizon-client/darwin": {OmnissaHorizonVersionShortener},
	"cisco-jabber/darwin":           {CiscoJabberVersionTransformer},
}

func ChromePKGInstaller(app *maintained_apps.FMAManifestApp) (*maintained_apps.FMAManifestApp, error) {
	app.Version = "latest"
	app.InstallerURL = "https://dl.google.com/dl/chrome/mac/universal/stable/gcem/GoogleChrome.pkg"

	return app, nil
}

// CiscoJabberVersionTransformer sets the version to "latest" so that the validation
// extracts the actual app version from the PKG file, which matches what osquery reports.
// Homebrew reports a build number (e.g., "20251027035315") instead of the app version (e.g., "15.1.2").
func CiscoJabberVersionTransformer(app *maintained_apps.FMAManifestApp) (*maintained_apps.FMAManifestApp, error) {
	app.Version = "latest"
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
