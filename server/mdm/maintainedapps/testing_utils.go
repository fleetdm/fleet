package maintained_apps

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/log"
	"github.com/stretchr/testify/require"
)

// SyncApps ingests the maintained apps from the apps list manifest
// to fill the library of maintained apps with valid data for tests.
// It returns the results of the ingestion as a slice of
// fleet.MaintainedApps.
func SyncApps(t *testing.T, ds fleet.Datastore) []fleet.MaintainedApp {
	_, filename, _, _ := runtime.Caller(0)
	base := filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(filename))))
	outputsDir := filepath.Join(base, "ee/maintained-apps/outputs")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := os.ReadFile(filepath.Join(outputsDir, r.URL.Path))
		if err != nil {
			if os.IsNotExist(err) {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(err.Error()))
			return
		}
		_, _ = w.Write(b)
	}))
	defer srv.Close()

	// not using t.Setenv because we want the env var to be unset on return of
	// this call
	os.Setenv("FLEET_DEV_MAINTAINED_APPS_BASE_URL", srv.URL)
	defer os.Unsetenv("FLEET_DEV_MAINTAINED_APPS_BASE_URL")

	err := Refresh(context.Background(), ds, log.NewNopLogger())
	require.NoError(t, err)

	apps, _, err := ds.ListAvailableFleetMaintainedApps(context.Background(), nil, fleet.ListOptions{})
	require.NoError(t, err)
	return apps
}

// ExpectedAppSlugs returns the list of app slugs (unique identifier) that are
// expected to be in the maintained apps library after ingestion. The slugs are
// taken from the apps.json list.
func ExpectedAppSlugs(t *testing.T) []string {
	_, filename, _, _ := runtime.Caller(0)
	base := filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(filename))))
	outputsDir := filepath.Join(base, "ee/maintained-apps/outputs")
	b, err := os.ReadFile(filepath.Join(outputsDir, "apps.json"))
	require.NoError(t, err)

	var appsList AppsList
	err = json.Unmarshal(b, &appsList)
	require.NoError(t, err)

	slugs := make([]string, len(appsList.Apps))
	for i, app := range appsList.Apps {
		slugs[i] = app.Slug
	}
	return slugs
}

func SyncAndRemoveApps(t *testing.T, ds fleet.Datastore) {
	_, filename, _, _ := runtime.Caller(0)
	base := filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(filename))))
	outputsDir := filepath.Join(base, "ee/maintained-apps/outputs")

	b, err := os.ReadFile(filepath.Join(outputsDir, "apps.json"))
	require.NoError(t, err)
	var appsFile AppsList
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
	os.Setenv("FLEET_DEV_MAINTAINED_APPS_BASE_URL", srv.URL)
	defer os.Unsetenv("FLEET_DEV_MAINTAINED_APPS_BASE_URL")

	err = Refresh(context.Background(), ds, log.NewNopLogger())
	require.NoError(t, err)

	originalApps, _, err := ds.ListAvailableFleetMaintainedApps(context.Background(), nil, fleet.ListOptions{})
	require.NoError(t, err)

	require.Equal(t, len(appsFile.Apps), len(originalApps))

	// Modify the apps list to simulate removing an app from upstream
	removedApp := appsFile.Apps[0]
	appsFile.Apps = appsFile.Apps[1:]

	err = Refresh(context.Background(), ds, log.NewNopLogger())
	require.NoError(t, err)

	modifiedApps, _, err := ds.ListAvailableFleetMaintainedApps(context.Background(), nil, fleet.ListOptions{})
	require.NoError(t, err)

	require.Equal(t, len(appsFile.Apps), len(modifiedApps))
	require.Equal(t, len(originalApps)-1, len(modifiedApps))
	for _, a := range modifiedApps {
		require.NotEqual(t, removedApp.Slug, a.Slug)
	}

	// remove all apps from upstream.
	appsFile.Apps = []appListing{}

	err = Refresh(context.Background(), ds, log.NewNopLogger())
	require.NoError(t, err)

	modifiedApps, _, err = ds.ListAvailableFleetMaintainedApps(context.Background(), nil, fleet.ListOptions{})
	require.ErrorIs(t, err, &fleet.NoMaintainedAppsInDatabaseError{})
	require.Empty(t, modifiedApps)
}
