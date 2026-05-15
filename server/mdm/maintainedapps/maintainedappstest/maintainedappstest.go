// Package maintainedappstest provides test helpers that drive the maintained
// apps sync flow against an in-memory HTTP server serving the repo's
// `ee/maintained-apps/outputs` testdata.
//
// It imports the "testing" package and must therefore only ever be imported
// from test code; importing it from production code would pull "testing"
// into the resulting binary.
package maintainedappstest

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/fleetdm/fleet/v4/server/dev_mode"
	"github.com/fleetdm/fleet/v4/server/fleet"
	maintained_apps "github.com/fleetdm/fleet/v4/server/mdm/maintainedapps"
	"github.com/stretchr/testify/require"
)

// outputsDir returns the absolute path to the repo's
// ee/maintained-apps/outputs directory. It walks up from this file's path,
// which is reliable as long as this file lives at
// server/mdm/maintainedapps/maintainedappstest/maintainedappstest.go.
func outputsDir() string {
	_, filename, _, _ := runtime.Caller(0)
	// Walk up: maintainedappstest -> maintainedapps -> mdm -> server -> repo root.
	base := filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(filename)))))
	return filepath.Join(base, "ee/maintained-apps/outputs")
}

// SyncApps ingests the maintained apps from the apps list manifest to fill
// the library of maintained apps with valid data for tests. It returns the
// results of the ingestion as a slice of fleet.MaintainedApps.
func SyncApps(t *testing.T, ds fleet.Datastore) []fleet.MaintainedApp {
	dir := outputsDir()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := os.ReadFile(filepath.Join(dir, r.URL.Path))
		if err != nil {
			if os.IsNotExist(err) {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(err.Error())) //nolint:gosec // test helper, error from local testdata read
			return
		}
		_, _ = w.Write(b) //nolint:gosec // test helper, serving local testdata
	}))
	defer srv.Close()

	// not using t.Setenv because we want the env var to be unset on return of
	// this call
	dev_mode.SetOverride("FLEET_DEV_MAINTAINED_APPS_BASE_URL", srv.URL)
	defer dev_mode.ClearOverride("FLEET_DEV_MAINTAINED_APPS_BASE_URL")
	dev_mode.SetOverride("FLEET_DEV_MAINTAINED_APPS_FALLBACK_BASE_URL", srv.URL)
	defer dev_mode.ClearOverride("FLEET_DEV_MAINTAINED_APPS_FALLBACK_BASE_URL")

	err := maintained_apps.SyncAppsList(context.Background(), ds)
	require.NoError(t, err)

	apps, _, err := ds.ListAvailableFleetMaintainedApps(context.Background(), nil, fleet.ListOptions{
		OrderKey: "slug",
	})
	require.NoError(t, err)
	return apps
}

// ExpectedAppSlugs returns the list of app slugs (unique identifier) that
// are expected to be in the maintained apps library after ingestion. The
// slugs are taken from the apps.json list.
func ExpectedAppSlugs(t *testing.T) []string {
	b, err := os.ReadFile(filepath.Join(outputsDir(), "apps.json"))
	require.NoError(t, err)

	var appsList maintained_apps.AppsList
	err = json.Unmarshal(b, &appsList)
	require.NoError(t, err)

	slugs := make([]string, len(appsList.Apps))
	for i, app := range appsList.Apps {
		slugs[i] = app.Slug
	}
	return slugs
}

// SyncAndRemoveApps exercises the maintained-apps sync flow by:
//  1. Seeding the database from the real apps.json.
//  2. Re-syncing with the first app removed from the manifest and asserting
//     that app is dropped from the database.
//  3. Re-syncing with an empty manifest and asserting all apps are removed.
func SyncAndRemoveApps(t *testing.T, ds fleet.Datastore) {
	b, err := os.ReadFile(filepath.Join(outputsDir(), "apps.json"))
	require.NoError(t, err)
	var appsFile maintained_apps.AppsList
	require.NoError(t, json.Unmarshal(b, &appsFile))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, err := json.Marshal(&appsFile)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(err.Error()))
			return
		}
		_, _ = w.Write(data)
	}))
	defer srv.Close()

	// not using t.Setenv because we want the env var to be unset on return of
	// this call
	dev_mode.SetOverride("FLEET_DEV_MAINTAINED_APPS_BASE_URL", srv.URL)
	defer dev_mode.ClearOverride("FLEET_DEV_MAINTAINED_APPS_BASE_URL")

	err = maintained_apps.SyncAppsList(context.Background(), ds)
	require.NoError(t, err)

	originalApps, _, err := ds.ListAvailableFleetMaintainedApps(context.Background(), nil, fleet.ListOptions{})
	require.NoError(t, err)

	require.Len(t, originalApps, len(appsFile.Apps))

	// Modify the apps list to simulate removing an app from upstream
	removedApp := appsFile.Apps[0]
	appsFile.Apps = appsFile.Apps[1:]

	err = maintained_apps.SyncAppsList(context.Background(), ds)
	require.NoError(t, err)

	modifiedApps, _, err := ds.ListAvailableFleetMaintainedApps(context.Background(), nil, fleet.ListOptions{})
	require.NoError(t, err)

	require.Len(t, modifiedApps, len(appsFile.Apps))
	require.Len(t, modifiedApps, len(originalApps)-1)
	for _, a := range modifiedApps {
		require.NotEqual(t, removedApp.Slug, a.Slug)
	}

	// Remove all apps from upstream. We use a zero-length slice of the
	// existing (unexported) element type instead of a literal because
	// AppsList.Apps's element type is unexported.
	appsFile.Apps = appsFile.Apps[:0]

	err = maintained_apps.SyncAppsList(context.Background(), ds)
	require.NoError(t, err)

	modifiedApps, _, err = ds.ListAvailableFleetMaintainedApps(context.Background(), nil, fleet.ListOptions{})
	require.ErrorIs(t, err, &fleet.NoMaintainedAppsInDatabaseError{})
	require.Empty(t, modifiedApps)
}
