package maintained_apps

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/fleetdm/fleet/v4/server/dev_mode"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestAppsJSON returns a valid apps.json payload for testing.
func newTestAppsJSON() []byte {
	data, _ := json.Marshal(AppsList{
		Version: 2,
		Apps: []appListing{
			{Name: "Test App", Slug: "test-app", Platform: "darwin", UniqueIdentifier: "com.test.app"},
		},
	})
	return data
}

func TestFetchAppsListPrimarySuccess(t *testing.T) {
	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/apps.json", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(newTestAppsJSON())
	}))
	t.Cleanup(primary.Close)

	fallback := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("fallback should not be called when primary succeeds")
	}))
	t.Cleanup(fallback.Close)

	dev_mode.SetOverride("FLEET_DEV_MAINTAINED_APPS_BASE_URL", primary.URL, t)
	dev_mode.SetOverride("FLEET_DEV_MAINTAINED_APPS_FALLBACK_BASE_URL", fallback.URL, t)

	apps, err := FetchAppsList(t.Context())
	require.NoError(t, err)
	require.Len(t, apps.Apps, 1)
	assert.Equal(t, "test-app", apps.Apps[0].Slug)
}

func TestFetchAppsListFallbackOnPrimaryFailure(t *testing.T) {
	var fallbackWasCalled atomic.Bool
	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("primary is down"))
	}))
	t.Cleanup(primary.Close)

	fallback := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/apps.json", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(newTestAppsJSON())
		fallbackWasCalled.Store(true)
	}))
	t.Cleanup(fallback.Close)

	dev_mode.SetOverride("FLEET_DEV_MAINTAINED_APPS_BASE_URL", primary.URL, t)
	dev_mode.SetOverride("FLEET_DEV_MAINTAINED_APPS_FALLBACK_BASE_URL", fallback.URL, t)

	apps, err := FetchAppsList(t.Context())
	require.NoError(t, err)
	require.Len(t, apps.Apps, 1)
	assert.Equal(t, "test-app", apps.Apps[0].Slug)
	assert.True(t, fallbackWasCalled.Load())
}

func TestFetchAppsListFallbackOnPrimary404(t *testing.T) {
	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(primary.Close)

	fallback := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(newTestAppsJSON())
	}))
	t.Cleanup(fallback.Close)

	dev_mode.SetOverride("FLEET_DEV_MAINTAINED_APPS_BASE_URL", primary.URL, t)
	dev_mode.SetOverride("FLEET_DEV_MAINTAINED_APPS_FALLBACK_BASE_URL", fallback.URL, t)

	apps, err := FetchAppsList(t.Context())
	require.NoError(t, err)
	require.Len(t, apps.Apps, 1)
}

func TestFetchAppsListBothFail(t *testing.T) {
	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("primary down"))
	}))
	t.Cleanup(primary.Close)

	fallback := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("fallback down"))
	}))
	t.Cleanup(fallback.Close)

	dev_mode.SetOverride("FLEET_DEV_MAINTAINED_APPS_BASE_URL", primary.URL, t)
	dev_mode.SetOverride("FLEET_DEV_MAINTAINED_APPS_FALLBACK_BASE_URL", fallback.URL, t)

	_, err := FetchAppsList(t.Context())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "primary")
	assert.Contains(t, err.Error(), "fallback")
}

func TestFetchAppsListFallbackOnPrimaryNetworkError(t *testing.T) {
	// Start and immediately close the primary to simulate a connection-refused error.
	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	primary.Close()

	fallback := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(newTestAppsJSON())
	}))
	t.Cleanup(fallback.Close)

	dev_mode.SetOverride("FLEET_DEV_MAINTAINED_APPS_BASE_URL", primary.URL, t)
	dev_mode.SetOverride("FLEET_DEV_MAINTAINED_APPS_FALLBACK_BASE_URL", fallback.URL, t)

	apps, err := FetchAppsList(t.Context())
	require.NoError(t, err)
	require.Len(t, apps.Apps, 1)
}

func TestFetchAppsListOnlyFallbackFails(t *testing.T) {
	// Primary works; fallback is broken. Nothing should break.
	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(newTestAppsJSON())
	}))
	t.Cleanup(primary.Close)

	// Closed server as broken fallback.
	fallback := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	fallback.Close()

	dev_mode.SetOverride("FLEET_DEV_MAINTAINED_APPS_BASE_URL", primary.URL, t)
	dev_mode.SetOverride("FLEET_DEV_MAINTAINED_APPS_FALLBACK_BASE_URL", fallback.URL, t)

	apps, err := FetchAppsList(t.Context())
	require.NoError(t, err)
	require.Len(t, apps.Apps, 1)
}

func TestFetchManifestDataFallbackUsedForSlugPath(t *testing.T) {
	var primaryHits, fallbackHits atomic.Int32

	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		primaryHits.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(primary.Close)

	fallback := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fallbackHits.Add(1)
		assert.Equal(t, "/test-app.json", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"test": true}`))
	}))
	t.Cleanup(fallback.Close)

	dev_mode.SetOverride("FLEET_DEV_MAINTAINED_APPS_BASE_URL", primary.URL, t)
	dev_mode.SetOverride("FLEET_DEV_MAINTAINED_APPS_FALLBACK_BASE_URL", fallback.URL, t)

	body, err := fetchManifestFile(t.Context(), "/test-app.json")
	require.NoError(t, err)
	assert.Contains(t, string(body), `"test"`)
	assert.Equal(t, int32(1), primaryHits.Load(), "primary should have been tried once")
	assert.Equal(t, int32(1), fallbackHits.Load(), "fallback should have been tried once")
}

func TestFetchManifestDataPrimarySucceedsSkipsFallback(t *testing.T) {
	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok": true}`))
	}))
	t.Cleanup(primary.Close)

	var fallbackHits atomic.Int32
	fallback := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fallbackHits.Add(1)
	}))
	t.Cleanup(fallback.Close)

	dev_mode.SetOverride("FLEET_DEV_MAINTAINED_APPS_BASE_URL", primary.URL, t)
	dev_mode.SetOverride("FLEET_DEV_MAINTAINED_APPS_FALLBACK_BASE_URL", fallback.URL, t)

	body, err := fetchManifestFile(t.Context(), "/something.json")
	require.NoError(t, err)
	assert.Contains(t, string(body), `"ok"`)
	assert.Equal(t, int32(0), fallbackHits.Load(), "fallback must not be contacted when primary succeeds")
}

func TestResolveBaseURLsDefaults(t *testing.T) {
	dev_mode.SetOverride("FLEET_DEV_MAINTAINED_APPS_BASE_URL", "", t)
	dev_mode.SetOverride("FLEET_DEV_MAINTAINED_APPS_FALLBACK_BASE_URL", "", t)
	primary, fallback := resolveBaseURLs()
	assert.Equal(t, fmaOutputsBase, primary)
	assert.Equal(t, fmaOutputsFallbackBase, fallback)
}

func TestResolveBaseURLsWithOverrides(t *testing.T) {
	dev_mode.SetOverride("FLEET_DEV_MAINTAINED_APPS_BASE_URL", "http://custom-primary", t)
	dev_mode.SetOverride("FLEET_DEV_MAINTAINED_APPS_FALLBACK_BASE_URL", "http://custom-fallback", t)

	primary, fallback := resolveBaseURLs()
	assert.Equal(t, "http://custom-primary", primary)
	assert.Equal(t, "http://custom-fallback", fallback)
}

func TestDoFetchSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/test.json", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	t.Cleanup(srv.Close)

	body, err := doFetch(t.Context(), srv.URL, "/test.json")
	require.NoError(t, err)
	assert.Equal(t, "ok", string(body))
}

func TestDoFetchNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)

	_, err := doFetch(t.Context(), srv.URL, "/missing.json")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "404")
}

func TestDoFetchServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("bad gateway"))
	}))
	t.Cleanup(srv.Close)

	_, err := doFetch(t.Context(), srv.URL, "/broken.json")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "502")
}

func TestDoFetchNetworkError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close() // immediately close to force connection errors

	_, err := doFetch(t.Context(), srv.URL, "/anything.json")
	require.Error(t, err)
}

func TestDoFetchTruncatesLargeBodyInErrorMessage(t *testing.T) {
	largeBody := make([]byte, 1024)
	for i := range largeBody {
		largeBody[i] = 'x'
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write(largeBody)
	}))
	t.Cleanup(srv.Close)

	_, err := doFetch(t.Context(), srv.URL, "/big.json")
	require.Error(t, err)
	// The error message should contain at most 512 bytes of body.
	assert.LessOrEqual(t, len(err.Error()), 600) // 512 body + status prefix
}

func TestFetchAppsListFallbackOverrideViaEnvVar(t *testing.T) {
	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	t.Cleanup(primary.Close)

	var fallbackWasCalled atomic.Bool
	customFallback := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(newTestAppsJSON())
		fallbackWasCalled.Store(true)
	}))
	t.Cleanup(customFallback.Close)

	dev_mode.SetOverride("FLEET_DEV_MAINTAINED_APPS_BASE_URL", primary.URL, t)
	dev_mode.SetOverride("FLEET_DEV_MAINTAINED_APPS_FALLBACK_BASE_URL", customFallback.URL, t)

	apps, err := FetchAppsList(t.Context())
	require.NoError(t, err)
	require.Len(t, apps.Apps, 1)
	assert.Equal(t, "test-app", apps.Apps[0].Slug)
	assert.True(t, fallbackWasCalled.Load())
}
