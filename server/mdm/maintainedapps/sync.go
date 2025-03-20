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

type appsList struct {
	Version uint         `json:"version"`
	Apps    []appListing `json:"apps"`
}

const fmaOutputsBase = "https://raw.githubusercontent.com/fleetdm/fleet/refs/heads/main/ee/maintained-apps/outputs"

// Refresh fetches the latest information about maintained apps from the brew
// API and updates the Fleet database with the new information.
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

	var appsList appsList
	if err := json.Unmarshal(body, &appsList); err != nil {
		return ctxerr.Wrap(ctx, err, "unmarshal apps list")
	}
	if appsList.Version != 2 {
		return ctxerr.Errorf(ctx, "apps list is an incompatible version")
	}

	for _, app := range appsList.Apps {
		if _, err = ds.UpsertMaintainedApp(ctx, &fleet.MaintainedApp{
			Name:             app.Name,
			Slug:             app.Slug,
			Platform:         app.Platform,
			UniqueIdentifier: app.UniqueIdentifier,
		}); err != nil {
			return ctxerr.Wrap(ctx, err, "upsert maintained app")
		}
	}

	return nil
}
