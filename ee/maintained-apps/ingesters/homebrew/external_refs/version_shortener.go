package externalrefs

import (
	"errors"
	"fmt"
	"strings"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
)

// makeVersionShortener returns a manifest enricher that truncates the
// Homebrew version to the first keepSegments dot-separated segments.
// This aligns the manifest version with what macOS reports as
// bundle_short_version, preventing false "Update available" status.
func makeVersionShortener(keepSegments int) func(*maintained_apps.FMAManifestApp) (*maintained_apps.FMAManifestApp, error) {
	return func(app *maintained_apps.FMAManifestApp) (*maintained_apps.FMAManifestApp, error) {
		if app.Version == "" {
			return app, fmt.Errorf("empty version for app %s", app.Slug)
		}
		parts := strings.Split(app.Version, ".")
		if len(parts) <= keepSegments {
			return app, nil
		}
		app.Version = strings.Join(parts[:keepSegments], ".")
		return app, nil
	}
}

// https://github.com/fleetdm/fleet/issues/42673
var (
	AndroidStudioVersionShortener       = makeVersionShortener(2) // "2025.3.2.6" → "2025.3"
	MicrosoftAutoUpdateVersionShortener = makeVersionShortener(2) // "4.82.26020434" → "4.82"
	OperaVersionShortener               = makeVersionShortener(2) // "129.0.5823.28" → "129.0"
	TwingateVersionShortener            = makeVersionShortener(2) // "2026.29.22575" → "2026.29"
	CitrixWorkspaceVersionShortener     = makeVersionShortener(3) // "25.11.1.42" → "25.11.1"
	ElgatoStreamDeckVersionShortener    = makeVersionShortener(3) // "7.3.1.22604" → "7.3.1"
	FileMakerProVersionShortener        = makeVersionShortener(3) // "22.0.5.500" → "22.0.5"
	RoyalTSXVersionShortener            = makeVersionShortener(3) // "6.4.2.1000" → "6.4.2"
	GrammarlyDesktopVersionShortener    = makeVersionShortener(3) // "1.160.0.0" → "1.160.0"
	AnkaVersionShortener                = makeVersionShortener(3) // "3.8.6.212" → "3.8.6"
)

// SublimeVersionTransformer prepends "Build " to match what macOS reports as
// bundle_short_version for Sublime Text and Sublime Merge (e.g. "4200" → "Build 4200").
// Without this, osquery's version_compare treats the "Build " prefix as making
// the host version always greater, breaking patch policy detection.
func SublimeVersionTransformer(app *maintained_apps.FMAManifestApp) (*maintained_apps.FMAManifestApp, error) {
	if app.Version == "" {
		return app, fmt.Errorf("empty version for Sublime app %s", app.Slug)
	}
	if strings.HasPrefix(app.Version, "Build ") {
		return app, nil
	}
	app.Version = "Build " + app.Version
	return app, nil
}

// MySQLWorkbenchVersionTransformer appends ".CE" to match what macOS reports as
// bundle_short_version for MySQL Workbench Community Edition (e.g. "8.0.46" → "8.0.46.CE").
func MySQLWorkbenchVersionTransformer(app *maintained_apps.FMAManifestApp) (*maintained_apps.FMAManifestApp, error) {
	if app.Version == "" {
		return app, errors.New("empty version for MySQL Workbench")
	}
	app.Version += ".CE"
	return app, nil
}

// LensVersionTransformer appends "-latest" to match what macOS reports as
// bundle_short_version for Lens (e.g. "2026.3.251250" → "2026.3.251250-latest").
func LensVersionTransformer(app *maintained_apps.FMAManifestApp) (*maintained_apps.FMAManifestApp, error) {
	if app.Version == "" {
		return app, errors.New("empty version for Lens")
	}
	app.Version += "-latest"
	return app, nil
}
