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
