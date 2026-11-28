package winget

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"testing"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
	"github.com/stretchr/testify/require"
)

func TestValidateSource(t *testing.T) {
	cases := []struct {
		name    string
		input   inputApp
		wantErr string
	}{
		{name: "winget", input: inputApp{PackageIdentifier: "Foo.Bar"}},
		{name: "vendor", input: inputApp{LatestInstallerURL: "https://example.com/latest", ProgramPublisher: "Foo", InstallerType: "exe"}},
		{name: "neither", input: inputApp{}, wantErr: "missing package identifier"},
		{name: "both", input: inputApp{PackageIdentifier: "Foo.Bar", LatestInstallerURL: "https://example.com/latest"}, wantErr: "can't both be set"},
		{name: "vendor over http", input: inputApp{LatestInstallerURL: "http://example.com/latest", ProgramPublisher: "Foo", InstallerType: "exe"}, wantErr: "must be an https URL"},
		{name: "vendor without publisher", input: inputApp{LatestInstallerURL: "https://example.com/latest", InstallerType: "exe"}, wantErr: "requires program_publisher"},
		{name: "vendor msi", input: inputApp{LatestInstallerURL: "https://example.com/latest", ProgramPublisher: "Foo", InstallerType: "msi"}},
		{name: "vendor msix", input: inputApp{LatestInstallerURL: "https://example.com/latest", ProgramPublisher: "Foo", InstallerType: "msix"}, wantErr: "supports installer_type msi or exe"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateSource(tc.input)
			if tc.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tc.wantErr)
		})
	}
}

func TestIngestOneVendor(t *testing.T) {
	ctx := t.Context()
	exe, err := os.ReadFile("../../../../pkg/file/testdata/software-installers/hello-world-installer.exe")
	require.NoError(t, err)
	exeSum := sha256.Sum256(exe)

	tempDir := t.TempDir()
	installScript := filepath.Join(tempDir, "install.ps1")
	uninstallScript := filepath.Join(tempDir, "uninstall.ps1")
	require.NoError(t, os.WriteFile(installScript, []byte("install"), 0o644))
	require.NoError(t, os.WriteFile(uninstallScript, []byte("uninstall"), 0o644))

	var latestStatus atomic.Int32
	var downloads atomic.Int32
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/latest":
			if s := int(latestStatus.Load()); s != 0 {
				w.WriteHeader(s)
				return
			}
			http.Redirect(w, r, "/hop", http.StatusFound)
		case "/hop":
			http.Redirect(w, r, "/files/Foo-1-0-0.exe", http.StatusMovedPermanently)
		case "/http":
			http.Redirect(w, r, "http://example.com/Foo.exe", http.StatusFound)
		case "/loop":
			http.Redirect(w, r, "/loop", http.StatusFound)
		case "/latest-unavailable":
			http.Redirect(w, r, "/files/unavailable.exe", http.StatusFound)
		case "/files/unavailable.exe":
			if r.Method == http.MethodGet {
				w.WriteHeader(http.StatusServiceUnavailable)
			}
		case "/latest-truncated":
			http.Redirect(w, r, "/files/truncated.exe", http.StatusFound)
		case "/files/truncated.exe":
			w.Header().Set("Content-Length", strconv.Itoa(len(exe)))
			if r.Method == http.MethodGet {
				_, _ = w.Write(exe[:100])
			}
		case "/files/Foo-1-0-0.exe":
			if r.Method == http.MethodGet {
				downloads.Add(1)
			}
			_, _ = w.Write(exe)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)

	outputDir := t.TempDir()
	i := &wingetIngester{
		logger:       slog.New(slog.DiscardHandler),
		vendorClient: srv.Client(),
		outputPath:   outputDir,
	}
	input := inputApp{
		Name:                "Foo",
		Slug:                "foo/windows",
		LatestInstallerURL:  srv.URL + "/latest",
		UniqueIdentifier:    "Foo",
		FuzzyMatchName:      fuzzyMatch{Enabled: true},
		ProgramPublisher:    "Foo, Inc",
		InstallerArch:       "x64",
		InstallerType:       "exe",
		InstallerScope:      "machine",
		InstallScriptPath:   installScript,
		UninstallScriptPath: uninstallScript,
	}
	installerURL := srv.URL + "/files/Foo-1-0-0.exe"

	t.Run("downloads the installer the latest URL redirects to", func(t *testing.T) {
		out, err := i.ingestOne(ctx, input)
		require.NoError(t, err)
		require.Equal(t, "1.0.0", out.Version)
		require.Equal(t, installerURL, out.InstallerURL)
		require.Equal(t, hex.EncodeToString(exeSum[:]), out.SHA256)
		require.Equal(t, "SELECT 1 FROM programs WHERE name LIKE 'Foo %' AND publisher = 'Foo, Inc';", out.Queries.Exists)
		require.Contains(t, out.Queries.Patched, "version_compare(version, '1.0.0') < 0")
		require.EqualValues(t, 1, downloads.Load())
	})

	t.Run("reuses the current output when the installer URL is unchanged", func(t *testing.T) {
		writeOutput := func(installerURL string) {
			b, err := json.Marshal(maintained_apps.FMAManifestFile{Versions: []*maintained_apps.FMAManifestApp{{
				Version: "0.9.0", InstallerURL: installerURL, SHA256: "cached",
			}}})
			require.NoError(t, err)
			require.NoError(t, os.MkdirAll(filepath.Join(outputDir, "foo"), 0o755))
			require.NoError(t, os.WriteFile(filepath.Join(outputDir, "foo", "windows.json"), b, 0o644))
		}
		t.Cleanup(func() { _ = os.RemoveAll(filepath.Join(outputDir, "foo")) })

		downloads.Store(0)
		writeOutput(installerURL)
		out, err := i.ingestOne(ctx, input)
		require.NoError(t, err)
		require.Equal(t, "0.9.0", out.Version)
		require.Equal(t, "cached", out.SHA256)
		require.Zero(t, downloads.Load())

		writeOutput(srv.URL + "/files/Foo-0-9-0.exe")
		out, err = i.ingestOne(ctx, input)
		require.NoError(t, err)
		require.Equal(t, "1.0.0", out.Version)
		require.EqualValues(t, 1, downloads.Load())
	})

	t.Run("a latest URL that doesn't redirect is rejected", func(t *testing.T) {
		in := input
		in.LatestInstallerURL = installerURL
		_, err := i.ingestOne(ctx, in)
		require.ErrorContains(t, err, "didn't redirect to a versioned installer")
	})

	t.Run("a redirect to plain HTTP is rejected", func(t *testing.T) {
		in := input
		in.LatestInstallerURL = srv.URL + "/http"
		_, err := i.ingestOne(ctx, in)
		require.ErrorContains(t, err, "non-HTTPS")
	})

	t.Run("a redirect loop is rejected", func(t *testing.T) {
		in := input
		in.LatestInstallerURL = srv.URL + "/loop"
		_, err := i.ingestOne(ctx, in)
		require.ErrorContains(t, err, "redirected more than")
	})

	t.Run("a failed or truncated download is transient", func(t *testing.T) {
		for _, latest := range []string{"/latest-unavailable", "/latest-truncated"} {
			in := input
			in.LatestInstallerURL = srv.URL + latest
			_, err := i.ingestOne(ctx, in)
			require.Error(t, err, latest)
			require.True(t, isTransientVendorError(err), "%s: %v", latest, err)
		}
	})

	t.Run("server errors are transient", func(t *testing.T) {
		latestStatus.Store(http.StatusServiceUnavailable)
		t.Cleanup(func() { latestStatus.Store(0) })
		_, err := i.ingestOne(ctx, input)
		require.Error(t, err)
		require.True(t, isTransientVendorError(err))
	})

	t.Run("a blocked request fails the run", func(t *testing.T) {
		latestStatus.Store(http.StatusForbidden)
		t.Cleanup(func() { latestStatus.Store(0) })
		_, err := i.ingestOne(ctx, input)
		require.ErrorContains(t, err, "returned 403")
		require.False(t, isTransientVendorError(err))
	})
}

func TestIngestOneVendorMSI(t *testing.T) {
	ctx := t.Context()
	msi, err := os.ReadFile("../../../../server/service/testdata/software-installers/fleet-osquery.msi")
	require.NoError(t, err)
	exe, err := os.ReadFile("../../../../pkg/file/testdata/software-installers/hello-world-installer.exe")
	require.NoError(t, err)

	var downloads atomic.Int32
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/latest":
			http.Redirect(w, r, "/files/foo-1.msi", http.StatusFound)
		case "/latest-exe":
			http.Redirect(w, r, "/files/foo-1.exe", http.StatusFound)
		case "/files/foo-1.msi":
			if r.Method == http.MethodGet {
				downloads.Add(1)
			}
			_, _ = w.Write(msi)
		case "/files/foo-1.exe":
			_, _ = w.Write(exe)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)

	outputDir := t.TempDir()
	i := &wingetIngester{logger: slog.New(slog.DiscardHandler), vendorClient: srv.Client(), outputPath: outputDir}
	input := inputApp{
		Name:               "Foo",
		Slug:               "foo/windows",
		LatestInstallerURL: srv.URL + "/latest",
		UniqueIdentifier:   "Foo",
		ProgramPublisher:   "Foo, Inc",
		InstallerArch:      "x64",
		InstallerType:      "msi",
		InstallerScope:     "machine",
	}

	out, err := i.ingestOne(ctx, input)
	require.NoError(t, err)
	require.NotEmpty(t, out.Version)
	require.NotEmpty(t, out.UpgradeCode)
	require.Contains(t, out.UninstallScript, out.UpgradeCode, "the generated uninstall script goes by upgrade code")
	require.Contains(t, out.Queries.Patched, "version_compare(version, '"+out.Version+"') < 0")
	require.EqualValues(t, 1, downloads.Load())

	t.Run("reuses the current output, upgrade code included", func(t *testing.T) {
		b, err := json.Marshal(maintained_apps.FMAManifestFile{Versions: []*maintained_apps.FMAManifestApp{out}})
		require.NoError(t, err)
		require.NoError(t, os.MkdirAll(filepath.Join(outputDir, "foo"), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(outputDir, "foo", "windows.json"), b, 0o644))

		cached, err := i.ingestOne(ctx, input)
		require.NoError(t, err)
		require.EqualValues(t, 1, downloads.Load())
		require.Equal(t, out.Version, cached.Version)
		require.Equal(t, out.SHA256, cached.SHA256)
		require.Equal(t, out.UpgradeCode, cached.UpgradeCode)
		require.Equal(t, out.UninstallScript, cached.UninstallScript)
	})

	t.Run("a file of the wrong type is rejected", func(t *testing.T) {
		in := input
		in.LatestInstallerURL = srv.URL + "/latest-exe"
		_, err := i.ingestOne(ctx, in)
		require.ErrorContains(t, err, "is an exe file, not msi")
	})
}
