package maintained_apps

import (
	"context"
	"encoding/json"
	"fmt"
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

// IngestMaintainedApps ingests the maintained apps from the apps list manifest
// to fill the library of maintained apps with valid data for tests.
// It returns the results of the ingestion as a slice of
// fleet.MaintainedApps.
func IngestMaintainedApps(t *testing.T, ds fleet.Datastore) []fleet.MaintainedApp {
	_, filename, _, _ := runtime.Caller(0)
	base := filepath.Dir(filepath.Dir(filepath.Dir(filepath.Dir(filename))))
	outputsDir := filepath.Join(base, "ee/maintained-apps/outputs")
	fmt.Println(outputsDir)

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
	outputsDir := filepath.Join(base, "ee/maintained-appsList/outputs")
	b, err := os.ReadFile(filepath.Join(outputsDir, "appsList.json"))
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
