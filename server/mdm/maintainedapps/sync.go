package maintained_apps

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	ma "github.com/fleetdm/fleet/v4/ee/maintained-apps"
	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/dev_mode"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

type appListing struct {
	Name             string `json:"name"`
	Slug             string `json:"slug"`
	Platform         string `json:"platform"`
	UniqueIdentifier string `json:"unique_identifier"`
}

type AppsList struct {
	Version uint         `json:"version"`
	Apps    []appListing `json:"apps"`
}

const fmaOutputsBase = "https://maintained-apps.fleetdm.com/manifests"
const fmaOutputsFallbackBase = "https://raw.githubusercontent.com/fleetdm/fleet/refs/heads/main/ee/maintained-apps/outputs"

// resolveBaseURLs returns the primary and fallback base URLs for FMA manifests,
// taking into account any dev-mode env var overrides.
func resolveBaseURLs() (primary, fallback string) {
	primary = fmaOutputsBase
	if baseFromEnvVar := dev_mode.Env("FLEET_DEV_MAINTAINED_APPS_BASE_URL"); baseFromEnvVar != "" {
		primary = baseFromEnvVar
	}

	fallback = fmaOutputsFallbackBase
	if fallbackFromEnvVar := dev_mode.Env("FLEET_DEV_MAINTAINED_APPS_FALLBACK_BASE_URL"); fallbackFromEnvVar != "" {
		fallback = fallbackFromEnvVar
	}

	return primary, fallback
}

// fetchManifestFile fetches a manifest file from the primary FMA CDN, falling back to the
// fallback CDN if the primary fails.
func fetchManifestFile(ctx context.Context, path string) ([]byte, error) {
	primaryBase, fallbackBase := resolveBaseURLs()

	body, primaryErr := doFetch(ctx, primaryBase, path)
	if primaryErr == nil {
		return body, nil
	}

	// Primary failed; try fallback.
	body, fallbackErr := doFetch(ctx, fallbackBase, path)
	if fallbackErr == nil {
		return body, nil
	}

	return nil, ctxerr.Errorf(ctx, "fetching FMA manifest file %q: primary (%s) failed: %v; fallback (%s) also failed: %v",
		path, primaryBase, primaryErr, fallbackBase, fallbackErr)
}

// doFetch performs a single HTTP GET for baseURL+path and returns the response
// body on success (HTTP 200). Any other outcome is returned as an error.
func doFetch(ctx context.Context, baseURL, path string) ([]byte, error) {
	httpClient := fleethttp.NewClient(fleethttp.WithTimeout(10 * time.Second))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s%s", baseURL, path), nil)
	if err != nil {
		return nil, fmt.Errorf("create http request: %w", err)
	}

	res, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute http request: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("read http response body: %w", err)
	}

	switch res.StatusCode {
	case http.StatusOK:
		return body, nil
	case http.StatusNotFound:
		return nil, errors.New("not found (HTTP 404)")
	default:
		if len(body) > 512 {
			body = body[:512]
		}
		return nil, fmt.Errorf("HTTP status %d: %s", res.StatusCode, string(body))
	}
}

// SyncAppsList fetches the latest FMA apps list and updates the apps list copy cached in the DB
func SyncAppsList(ctx context.Context, ds fleet.Datastore) error {
	appsList, err := FetchAppsList(ctx)
	if err != nil {
		return err
	}

	return upsertMaintainedApps(ctx, appsList, ds)
}

func FetchAppsList(ctx context.Context) (*AppsList, error) {
	body, err := fetchManifestFile(ctx, "/apps.json")
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "fetch apps list")
	}

	var appsList AppsList
	if err := json.Unmarshal(body, &appsList); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "unmarshal apps list")
	}
	if appsList.Version != 2 {
		return nil, ctxerr.New(ctx, "apps list is an incompatible version")
	}
	return &appsList, nil
}

func upsertMaintainedApps(ctx context.Context, appsList *AppsList, ds fleet.Datastore) error {
	var gotApps []string

	for _, app := range appsList.Apps {
		gotApps = append(gotApps, app.Slug)

		if app.UniqueIdentifier == "" {
			app.UniqueIdentifier = app.Name
		}

		if _, err := ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{
			Name:             app.Name,
			Slug:             app.Slug,
			Platform:         app.Platform,
			UniqueIdentifier: app.UniqueIdentifier,
		}); err != nil {
			return ctxerr.Wrap(ctx, err, "upsert maintained app")
		}
	}

	// remove apps that were removed upstream
	if err := ds.ClearRemovedFleetMaintainedApps(ctx, gotApps); err != nil {
		return ctxerr.Wrap(ctx, err, "clear removed maintained apps during refresh")
	}

	// Normalize software names to canonical FMA names. Runs after the loop so it
	// sees every app at once, which matters for FMAs sharing a bundle identifier.
	if err := ds.ReconcileMaintainedAppSoftwareNames(ctx); err != nil {
		return ctxerr.Wrap(ctx, err, "reconcile maintained app software names during refresh")
	}

	return nil
}

// FMAInstallerCache is an optional interface for looking up cached FMA installer
// metadata from the database. When provided to Hydrate with a target version,
// it allows skipping the remote manifest fetch if the version is already cached.
type FMAInstallerCache interface {
	GetCachedFMAInstallerMetadata(ctx context.Context, teamID *uint, fmaID uint, version string) (*fleet.MaintainedApp, error)
}

// Hydrate pulls information from app-level FMA manifests into an FMA skeleton
// pulled from the database. If version is non-empty and cache is provided, it
// loads the metadata from the local cache. On a cache miss it falls back to the
// remote manifest, so a published-but-not-yet-cached version (e.g. a freshly
// released latest an admin is pinning to in a single GitOps apply) can still be
// hydrated and downloaded; if the requested version isn't currently published
// either, it returns a "version not available" error. If no version is specified,
// it fetches the latest from the remote manifest.
func Hydrate(ctx context.Context, app *fleet.MaintainedApp, version string, teamID *uint, cache FMAInstallerCache) (*fleet.MaintainedApp, error) {
	if version != "" && cache == nil {
		return nil, ctxerr.New(ctx, "no fma version cache provided")
	}

	// If a specific version is requested and we have a cache, try the cache first.
	if version != "" && cache != nil {
		cached, err := cache.GetCachedFMAInstallerMetadata(ctx, teamID, app.ID, version)
		if err != nil && !fleet.IsNotFound(err) {
			return nil, ctxerr.Wrap(ctx, err, "get cached FMA installer metadata")
		}
		if err == nil {
			// Copy installer-level fields from cache onto the app,
			// preserving the app-level fields (ID, Name, Slug, etc.)
			// that were already loaded from the database.
			app.Version = cached.Version
			app.Platform = cached.Platform
			app.InstallerURL = cached.InstallerURL
			app.SHA256 = cached.SHA256
			app.InstallScript = cached.InstallScript
			app.UninstallScript = cached.UninstallScript
			app.AutomaticInstallQuery = cached.AutomaticInstallQuery
			app.Categories = cached.Categories
			app.UpgradeCode = cached.UpgradeCode
			app.PatchQuery = cached.PatchQuery
			return app, nil
		}
		// Cache miss: fall through to the remote manifest so a not-yet-cached
		// version an admin is pinning to can still be hydrated and downloaded. The
		// requested version must match a currently-published manifest version
		// (selected below), otherwise it's reported as not available.
	}

	body, err := fetchManifestFile(ctx, fmt.Sprintf("/%s.json", app.Slug))
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "fetch app manifest")
	}

	var manifest ma.FMAManifestFile
	if err := json.Unmarshal(body, &manifest); err != nil {
		return nil, ctxerr.Wrapf(ctx, err, "unmarshal FMA manifest for %s", app.Slug)
	}

	// Hydrate from the requested version when pinning to a not-yet-cached version,
	// otherwise the latest. A malformed manifest with no versions falls through to
	// the not-available error below rather than panicking.
	var selected *ma.FMAManifestApp
	if version == "" {
		if len(manifest.Versions) > 0 {
			selected = manifest.Versions[0]
		}
	} else {
		for _, v := range manifest.Versions {
			if v.Version == version {
				selected = v
				break
			}
		}
	}
	if selected == nil {
		// Manifests expose only currently-published versions, so a version that's
		// neither cached nor published (e.g. evicted or older) can't be fetched.
		return nil, ctxerr.Wrap(ctx, &fleet.BadRequestError{
			Message: fmt.Sprintf(
				"Couldn't edit %q: specified version is not available. Available versions are listed in the Fleet UI under Actions > Edit software.",
				app.Name,
			),
		})
	}
	selected.Slug = app.Slug

	app.Version = selected.Version
	app.Platform = selected.Platform()
	app.InstallerURL = selected.InstallerURL
	app.SHA256 = selected.SHA256
	app.InstallScript = manifest.Refs[selected.InstallScriptRef]
	app.UninstallScript = manifest.Refs[selected.UninstallScriptRef]
	app.AutomaticInstallQuery = selected.Queries.Exists
	app.Categories = selected.DefaultCategories
	app.UpgradeCode = selected.UpgradeCode
	app.PatchQuery = selected.Queries.Patched
	app.AppOpenQuery = selected.Queries.Open

	return app, nil
}
