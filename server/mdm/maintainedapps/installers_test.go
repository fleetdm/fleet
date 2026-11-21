package maintained_apps

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/fleetdm/fleet/v4/server/version"
	"github.com/stretchr/testify/require"
)

func TestInstallerFilenameExtraction(t *testing.T) {
	// Mock server to serve the "installers"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/redirect":
			w.Header().Set("Location", "/redirected%20package.exe")
			w.WriteHeader(302)
			_, _ = w.Write([]byte("redirecting"))
		case "/redirected%20package.exe":
			_, _ = w.Write([]byte("redirected fallback"))
		case "/compliant":
			w.Header().Set("Content-Disposition", `attachment; filename="compliant.msi"`)
			_, _ = w.Write([]byte("compliant"))
		case "/not_compliant":
			w.Header().Set("Content-Disposition", `attachment; filename=not_compliant.pkg`)
			_, _ = w.Write([]byte("not_compliant"))
		}
	}))
	defer srv.Close()

	// follow redirect and fall back to URL, after sanitization, when we don't have a content-disposition header
	client := fleethttp.NewClient(fleethttp.WithTimeout(time.Second))
	_, filename, err := DownloadInstaller(context.Background(), srv.URL+"/redirect", client)
	require.NoError(t, err)
	require.Equal(t, "redirected package.exe", filename)

	// handle properly formatted content-disposition header
	_, filename, err = DownloadInstaller(context.Background(), srv.URL+"/compliant", client)
	require.NoError(t, err)
	require.Equal(t, "compliant.msi", filename)

	// handle non-compliant content-disposition header
	_, filename, err = DownloadInstaller(context.Background(), srv.URL+"/not_compliant", client)
	require.NoError(t, err)
	require.Equal(t, "not_compliant.pkg", filename)
}

func TestDownloadInstallerUserAgent(t *testing.T) {
	fleetUA := "fleet/" + version.Version().Version
	goTokenUA := fleetUA + " Go-http-client/1.1"

	// Each case mimics one vendor CDN's User-Agent rule.
	cases := []struct {
		name    string
		allow   func(ua string) bool
		wantErr string
		wantUAs []string
	}{
		{
			// Akamai (downloads.tableau.com)
			name:    "requires an HTTP library token",
			allow:   func(ua string) bool { return strings.Contains(ua, "Go-http-client/") },
			wantUAs: []string{goTokenUA},
		},
		{
			// Cloudflare-fronted hosts (monotypeapp.monotype.com)
			name:    "rejects Go's default User-Agent",
			allow:   func(ua string) bool { return !strings.HasPrefix(ua, "Go-http-client") },
			wantUAs: []string{goTokenUA},
		},
		{
			// www.crestron.com
			name:    "rejects the Go token anywhere",
			allow:   func(ua string) bool { return !strings.Contains(strings.ToLower(ua), "go-http-client") },
			wantUAs: []string{goTokenUA, fleetUA},
		},
		{
			name:    "forbidden for every User-Agent",
			allow:   func(string) bool { return false },
			wantErr: "received response status code 403",
			wantUAs: []string{goTokenUA, fleetUA},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var mu sync.Mutex
			var gotUAs []string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				gotUAs = append(gotUAs, r.UserAgent())
				mu.Unlock()
				if !c.allow(r.UserAgent()) {
					w.WriteHeader(http.StatusForbidden)
					return
				}
				_, _ = w.Write([]byte("ok"))
			}))
			defer srv.Close()

			client := fleethttp.NewClient(fleethttp.WithTimeout(time.Second))
			tfr, _, err := DownloadInstaller(t.Context(), srv.URL+"/installer.msi", client)
			if c.wantErr != "" {
				require.ErrorContains(t, err, c.wantErr)
			} else {
				require.NoError(t, err)
				require.NoError(t, tfr.Close())
			}
			mu.Lock()
			defer mu.Unlock()
			require.Equal(t, c.wantUAs, gotUAs)
		})
	}
}
