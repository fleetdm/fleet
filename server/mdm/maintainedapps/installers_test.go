package maintained_apps

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
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
	// Mimics both CDN rules: Cloudflare-fronted hosts that reject Go's default
	// User-Agent, and Akamai hosts that reject any User-Agent lacking a known
	// HTTP library token.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ua := r.UserAgent()
		if strings.HasPrefix(ua, "Go-http-client") || !strings.Contains(ua, "Go-http-client/") {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	client := fleethttp.NewClient(fleethttp.WithTimeout(time.Second))
	_, _, err := DownloadInstaller(context.Background(), srv.URL+"/installer.pkg", client)
	require.NoError(t, err)
	require.Equal(t, "fleet/"+version.Version().Version+" Go-http-client/1.1", installerUserAgent())
}

func TestDownloadInstallerReferer(t *testing.T) {
	var srvURL string
	var referers []string
	dmg := []byte{0x78, 0x01, 0x73, 0x0d, 0x62, 0x62, 0x60}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/downloads/installer.dmg":
			referers = append(referers, r.Referer())
			// Mimics jamovi.org: downloads only work when linked from the vendor's site.
			if r.Referer() != srvURL+"/" {
				http.Redirect(w, r, "/404.html", http.StatusFound)
				return
			}
			http.Redirect(w, r, "/cdn/installer.dmg", http.StatusFound)
		case "/cdn/installer.dmg":
			if r.Referer() != srvURL+"/" {
				http.Redirect(w, r, "/404.html", http.StatusFound)
				return
			}
			_, _ = w.Write(dmg)
		case "/sf/installer.dmg":
			referers = append(referers, r.Referer())
			// Mimics SourceForge: a Referer gets the HTML download page, not the file.
			if r.Referer() != "" {
				_, _ = w.Write([]byte("<!DOCTYPE html><html><body>Your download will start shortly</body></html>"))
				return
			}
			_, _ = w.Write(dmg)
		case "/404.html":
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte("<!DOCTYPE html><html><body>Not found</body></html>"))
		}
	}))
	defer srv.Close()
	srvURL = srv.URL

	client := fleethttp.NewClient(fleethttp.WithTimeout(time.Second))
	tfr, filename, err := DownloadInstaller(t.Context(), srv.URL+"/downloads/installer.dmg", client)
	require.NoError(t, err)
	defer tfr.Close()
	require.Equal(t, "installer.dmg", filename)
	b, err := io.ReadAll(tfr)
	require.NoError(t, err)
	require.Equal(t, dmg, b)
	require.Equal(t, []string{"", srv.URL + "/"}, referers)

	referers = nil
	tfr2, _, err := DownloadInstaller(t.Context(), srv.URL+"/sf/installer.dmg", client)
	require.NoError(t, err)
	defer tfr2.Close()
	require.Equal(t, []string{""}, referers)
}

func TestDownloadInstallerHTMLPage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/moved.dmg":
			http.Redirect(w, r, "/404.html", http.StatusFound)
		case "/404.html":
			_, _ = w.Write([]byte("\n  <!doctype html>\n<html><head><title>Page not found</title></head></html>"))
		case "/installer.pkg":
			_, _ = w.Write([]byte("xar!\x00\x1c\x00\x01"))
		case "/empty.zip":
		}
	}))
	defer srv.Close()

	client := fleethttp.NewClient(fleethttp.WithTimeout(time.Second))
	_, _, err := DownloadInstaller(t.Context(), srv.URL+"/moved.dmg", client)
	require.ErrorContains(t, err, "returned a web page instead of an installer")

	tfr, _, err := DownloadInstaller(t.Context(), srv.URL+"/installer.pkg", client)
	require.NoError(t, err)
	defer tfr.Close()
	// The sniff must leave the reader at the start of the file.
	b, err := io.ReadAll(tfr)
	require.NoError(t, err)
	require.Equal(t, "xar!\x00\x1c\x00\x01", string(b))

	tfr2, _, err := DownloadInstaller(t.Context(), srv.URL+"/empty.zip", client)
	require.NoError(t, err)
	tfr2.Close()
}
