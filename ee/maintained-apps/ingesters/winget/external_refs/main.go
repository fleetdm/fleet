package externalrefs

import (
	"fmt"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
)

// Funcs is a registry of enrichment functions keyed by app slug.
// Each slug can have multiple enricher functions that run sequentially.
var Funcs = map[string][]func(*maintained_apps.FMAManifestApp) (*maintained_apps.FMAManifestApp, error){
	"1password/windows":                {OnePasswordVersionShortener},
	"anyburn/windows":                  {AnyBurnVersionShortener},
	"aomei-backupper-standard/windows": {AOMEIBackupperVersionShortener},
	"datadog-agent/windows":            {DatadogAgentVersionNormalizer},
	"firefox@esr/windows":              {FirefoxESRVersionShortener},
	"geany/windows":                    {GeanyVersionShortener},
	"megasync/windows":                 {MEGAsyncVersionShortener},
	"sublime-text/windows":             {SublimeTextFileVersion},
	"telegram/windows":                 {TelegramVersionShortener},
}

// EnrichManifest applies all registered enrichment functions for the given app.
// Enrichers are looked up by app.Slug and run sequentially. An enricher error
// stops ingestion: enrichers set the version the patch policy compares, so
// skipping one would ship a policy that passes outdated hosts.
func EnrichManifest(app *maintained_apps.FMAManifestApp) error {
	for _, enricher := range Funcs[app.Slug] {
		var err error
		app, err = enricher(app)
		if err != nil {
			return fmt.Errorf("enriching app %s: %w", app.Slug, err)
		}
	}
	return nil
}
