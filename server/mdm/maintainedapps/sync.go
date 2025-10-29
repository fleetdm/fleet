package maintained_apps

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	ma "github.com/fleetdm/fleet/v4/ee/maintained-apps"
	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
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
	httpClient := fleethttp.NewClient(fleethttp.WithTimeout(10 * time.Second))
	baseURL := fmaOutputsBase
	if baseFromEnvVar := os.Getenv("FLEET_DEV_MAINTAINED_APPS_BASE_URL"); baseFromEnvVar != "" {
		baseURL = baseFromEnvVar
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/apps.json", baseURL), nil)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "create http request")
	}

	res, err := httpClient.Do(req)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "execute http request")
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "read http response body")
	}

	switch res.StatusCode {
	case http.StatusOK:
		// success, go on
	case http.StatusNotFound:
		return ctxerr.New(ctx, "maintained apps list not found")
	default:
		if len(body) > 512 {
			body = body[:512]
		}
		return ctxerr.Errorf(ctx, "apps list returned HTTP status %d: %s", res.StatusCode, string(body))
	}

	var appsList AppsList
	if err := json.Unmarshal(body, &appsList); err != nil {
		return ctxerr.Wrap(ctx, err, "unmarshal apps list")
	}
	if appsList.Version != 2 {
		return ctxerr.New(ctx, "apps list is an incompatible version")
	}

	var gotApps []string

	for _, app := range appsList.Apps {
		gotApps = append(gotApps, app.Slug)

		if app.UniqueIdentifier == "" {
			app.UniqueIdentifier = app.Name
		}

		if _, err = ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{
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

// Hydrate pulls information from app-level FMA manifests info an FMA skeleton pulled from the database
func Hydrate(ctx context.Context, app *fleet.MaintainedApp) (*fleet.MaintainedApp, error) {
	httpClient := fleethttp.NewClient(fleethttp.WithTimeout(10 * time.Second))
	baseURL := fmaOutputsBase
	if baseFromEnvVar := os.Getenv("FLEET_DEV_MAINTAINED_APPS_BASE_URL"); baseFromEnvVar != "" {
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

	return app, nil
}
