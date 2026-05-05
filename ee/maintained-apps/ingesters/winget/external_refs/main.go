package externalrefs

import (
	"fmt"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
)

// Funcs is a registry of enrichment functions keyed by app slug.
// Each slug can have multiple enricher functions that run sequentially.
var Funcs = map[string][]func(*maintained_apps.FMAManifestApp) (*maintained_apps.FMAManifestApp, error){
	"1password/windows": {OnePasswordVersionShortener},
}

// EnrichManifest applies all registered enrichment functions for the given app.
// Enrichers are looked up by app.Slug and run sequentially.
// Errors are logged but do not stop the enrichment pipeline.
func EnrichManifest(app *maintained_apps.FMAManifestApp) {
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
