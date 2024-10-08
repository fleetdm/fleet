package maintainedapps

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/log"
	"github.com/stretchr/testify/require"
)

// IngestMaintainedApps ingests the maintained apps from the testdata
// directory, to fill the library of maintained apps with valid data for tests.
// It returns the expected results of the ingestion as a slice of
// fleet.MaintainedApps with only a few fields filled - the result of
// unmarshaling the testdata/expected_apps.json file.
func IngestMaintainedApps(t *testing.T, ds fleet.Datastore) []fleet.MaintainedApp {
	_, filename, _, _ := runtime.Caller(0)
	base := filepath.Dir(filename)
	testdataDir := filepath.Join(base, "testdata")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := path.Base(r.URL.Path)
		b, err := os.ReadFile(filepath.Join(testdataDir, token))
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
	os.Setenv("FLEET_DEV_BREW_API_URL", srv.URL)
	defer os.Unsetenv("FLEET_DEV_BREW_API_URL")

	err := Refresh(context.Background(), ds, log.NewNopLogger())
	require.NoError(t, err)

	var expected []fleet.MaintainedApp
	b, err := os.ReadFile(filepath.Join(testdataDir, "expected_apps.json"))
	require.NoError(t, err)
	err = json.Unmarshal(b, &expected)
	require.NoError(t, err)
	return expected
}

// ExpectedAppTokens returns the list of app tokens (unique identifier) that are
// expected to be in the maintained apps library after ingestion. The tokens are
// taken from the apps.json list.
func ExpectedAppTokens(t *testing.T) []string {
	var apps []maintainedApp
	err := json.Unmarshal(appsJSON, &apps)
	require.NoError(t, err)

	tokens := make([]string, len(apps))
	for i, app := range apps {
		tokens[i] = app.Identifier
	}
	return tokens
}
