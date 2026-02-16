package maintained_apps

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/stretchr/testify/require"
)

func TestDownloadInstallerWithHeaders(t *testing.T) {
	// Mock server that checks for User-Agent header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userAgent := r.Header.Get("User-Agent")

		switch r.URL.Path {
		case "/with-header-required":
			// Simulate Warp behavior: return HTML without header, binary with header
			if userAgent == "Homebrew" {
				w.Header().Set("Content-Disposition", `attachment; filename="app.dmg"`)
				_, _ = w.Write([]byte("binary content"))
			} else {
				w.Header().Set("Content-Type", "text/html")
				_, _ = w.Write([]byte("<html>Download page</html>"))
			}
		case "/no-header-needed":
			// Normal app that works without special headers
			w.Header().Set("Content-Disposition", `attachment; filename="normal.pkg"`)
			_, _ = w.Write([]byte("normal app"))
		}
	}))
	defer srv.Close()

	client := fleethttp.NewClient(fleethttp.WithTimeout(time.Second))

	// Test 1: Existing FMA with nil headers (backwards compatibility)
	t.Run("nil headers work", func(t *testing.T) {
		_, filename, err := DownloadInstaller(context.Background(), srv.URL+"/no-header-needed", nil, client)
		require.NoError(t, err)
		require.Equal(t, "normal.pkg", filename)
	})

	// Test 2: Existing FMA with empty headers map (backwards compatibility)
	t.Run("empty headers map works", func(t *testing.T) {
		emptyHeaders := make(map[string]string)
		_, filename, err := DownloadInstaller(context.Background(), srv.URL+"/no-header-needed", emptyHeaders, client)
		require.NoError(t, err)
		require.Equal(t, "normal.pkg", filename)
	})

	// Test 3: New FMA with custom headers (Warp use case)
	t.Run("custom headers work", func(t *testing.T) {
		headers := map[string]string{
			"User-Agent": "Homebrew",
		}
		_, filename, err := DownloadInstaller(context.Background(), srv.URL+"/with-header-required", headers, client)
		require.NoError(t, err)
		require.Equal(t, "app.dmg", filename)
	})

	// Test 4: Verify headers don't break normal downloads
	t.Run("headers don't break normal apps", func(t *testing.T) {
		// Send headers even to apps that don't need them
		headers := map[string]string{
			"User-Agent": "Homebrew",
		}
		_, filename, err := DownloadInstaller(context.Background(), srv.URL+"/no-header-needed", headers, client)
		require.NoError(t, err)
		require.Equal(t, "normal.pkg", filename)
	})
}
