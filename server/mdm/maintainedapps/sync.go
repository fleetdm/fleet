package maintained_apps

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	ma "github.com/fleetdm/fleet/v4/ee/maintained-apps"
	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/dev_mode"
	"github.com/fleetdm/fleet/v4/server/fleet"
	kitlog "github.com/go-kit/log"
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

const fmaOutputsBase = "https://raw.githubusercontent.com/fleetdm/fleet/refs/heads/main/ee/maintained-apps/outputs"

// Refresh fetches the latest information about maintained apps from FMA's
// apps list on GitHub and updates the Fleet database with the new information.
func Refresh(ctx context.Context, ds fleet.Datastore, logger kitlog.Logger) error {
	appsList, err := FetchAppsList(ctx)
	if err != nil {
		return err
	}

	return upsertMaintainedApps(ctx, appsList, ds)
}

func FetchAppsList(ctx context.Context) (*AppsList, error) {
	httpClient := fleethttp.NewClient(fleethttp.WithTimeout(10 * time.Second))
	baseURL := fmaOutputsBase
	if baseFromEnvVar := dev_mode.Env("FLEET_DEV_MAINTAINED_APPS_BASE_URL"); baseFromEnvVar != "" {
		baseURL = baseFromEnvVar
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/apps.json", baseURL), nil)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "create http request")
	}

	res, err := httpClient.Do(req)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "execute http request")
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "read http response body")
	}

	switch res.StatusCode {
	case http.StatusOK:
		// success, go on
	case http.StatusNotFound:
		return nil, ctxerr.New(ctx, "maintained apps list not found")
	default:
		if len(body) > 512 {
			body = body[:512]
		}
		return nil, ctxerr.Errorf(ctx, "apps list returned HTTP status %d: %s", res.StatusCode, string(body))
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
// loads the metadata from the local cache, returning an error if the version is
// not cached. If no version is specified, it fetches the latest from the remote manifest.
func Hydrate(ctx context.Context, app *fleet.MaintainedApp, version string, teamID *uint, cache FMAInstallerCache) (*fleet.MaintainedApp, error) {
	if version != "" && cache == nil {
		return nil, ctxerr.New(ctx, "no fma version cache provided")
	}

	// If a specific version is requested and we have a cache, try the cache first.
	if version != "" && cache != nil {
		cached, err := cache.GetCachedFMAInstallerMetadata(ctx, teamID, app.ID, version)
		if err != nil {
			if fleet.IsNotFound(err) {
				// Version not found in cache - return the same error as BatchSetSoftwareInstallers
				return nil, ctxerr.Wrap(ctx, &fleet.BadRequestError{
					Message: fmt.Sprintf(
						"Couldn't edit %q: specified version is not available. Available versions are listed in the Fleet UI under Actions > Edit software.",
						app.Name,
					),
				})
			}
			return nil, ctxerr.Wrap(ctx, err, "get cached FMA installer metadata")
		}

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
		return app, nil
	}

	httpClient := fleethttp.NewClient(fleethttp.WithTimeout(10 * time.Second))
	baseURL := fmaOutputsBase
	if baseFromEnvVar := dev_mode.Env("FLEET_DEV_MAINTAINED_APPS_BASE_URL"); baseFromEnvVar != "" {
		baseURL = baseFromEnvVar
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/%s.json", baseURL, app.Slug), nil)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "create http request")
	}

	res, err := httpClient.Do(req)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "execute http request")
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "read http response body")
	}

	switch res.StatusCode {
	case http.StatusOK:
		// success, go on
	case http.StatusNotFound:
		return nil, ctxerr.New(ctx, "app not found in Fleet manifests")
	default:
		if len(body) > 512 {
			body = body[:512]
		}
		return nil, ctxerr.Errorf(ctx, "manifest retrieval returned HTTP status %d: %s", res.StatusCode, string(body))
	}

	var manifest ma.FMAManifestFile
	if err := json.Unmarshal(body, &manifest); err != nil {
		return nil, ctxerr.Wrapf(ctx, err, "unmarshal FMA manifest for %s", app.Slug)
	}
	manifest.Versions[0].Slug = app.Slug

	app.Version = manifest.Versions[0].Version
	app.Platform = manifest.Versions[0].Platform()
	app.InstallerURL = manifest.Versions[0].InstallerURL
	app.SHA256 = manifest.Versions[0].SHA256
	app.InstallScript = manifest.Refs[manifest.Versions[0].InstallScriptRef]
	app.UninstallScript = manifest.Refs[manifest.Versions[0].UninstallScriptRef]
	app.AutomaticInstallQuery = manifest.Versions[0].Queries.Exists
	app.Categories = manifest.Versions[0].DefaultCategories
	app.UpgradeCode = manifest.Versions[0].UpgradeCode

	return app, nil
}
