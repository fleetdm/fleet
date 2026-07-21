package homebrew

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/stretchr/testify/require"
)

func TestIngestValidations(t *testing.T) {
	tempDir := t.TempDir()

	testInstallScriptContents := "this is a test install script"
	require.NoError(t, os.WriteFile(path.Join(tempDir, "install_script.sh"), []byte(testInstallScriptContents), 0644))

	testUninstallScriptContents := "this is a test uninstall script"
	require.NoError(t, os.WriteFile(path.Join(tempDir, "uninstall_script.sh"), []byte(testUninstallScriptContents), 0644))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var cask brewCask

		appToken := strings.TrimSuffix(path.Base(r.URL.Path), ".json")
		switch appToken {
		case "fail":
			w.WriteHeader(http.StatusInternalServerError)
			return

		case "notfound":
			w.WriteHeader(http.StatusNotFound)
			return

		case "noname":
			cask = brewCask{
				Token:   appToken,
				Name:    nil,
				URL:     "https://example.com",
				Version: "1.0",
			}

		case "emptyname":
			cask = brewCask{
				Token:   appToken,
				Name:    []string{""},
				URL:     "https://example.com",
				Version: "1.0",
			}

		case "notoken":
			cask = brewCask{
				Token:   "",
				Name:    []string{appToken},
				URL:     "https://example.com",
				Version: "1.0",
			}

		case "noversion":
			cask = brewCask{
				Token:   appToken,
				Name:    []string{appToken},
				URL:     "https://example.com",
				Version: "",
			}

		case "nourl":
			cask = brewCask{
				Token:   appToken,
				Name:    []string{appToken},
				URL:     "",
				Version: "1.0",
			}

		case "invalidurl":
			cask = brewCask{
				Token:   appToken,
				Name:    []string{appToken},
				URL:     "https://\x00\x01\x02",
				Version: "1.0",
			}

		case "ok", "docker-desktop", "swiftdialog", "install_script_path", "uninstall_script_path", "uninstall_script_path_with_pre", "uninstall_script_path_with_post", "patch_policy_path", "open-query":
			cask = brewCask{
				Token:   appToken,
				Name:    []string{appToken},
				URL:     "https://example.com",
				Version: "1.0",
			}

		case "firefox@developer-edition":
			cask = brewCask{
				Token:   appToken,
				Name:    []string{"Mozilla Firefox Developer Edition"},
				URL:     "https://example.com",
				Version: "153.0b13",
			}

		case "firefox@nightly":
			cask = brewCask{
				Token:   appToken,
				Name:    []string{"Mozilla Firefox Nightly"},
				URL:     "https://example.com",
				Version: "154.0a1,2026-07-17-09-27-13",
			}

		default:
			w.WriteHeader(http.StatusBadRequest)
			t.Fatalf("unexpected app token %s", appToken)
		}

		err := json.NewEncoder(w).Encode(cask)
		require.NoError(t, err)
	}))
	t.Cleanup(srv.Close)

	// buildhub stub: DevEd 153.0b13 build id -> CFBundleVersion "15326.7.15".
	buildhubSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"hits":{"hits":[{"_source":{"build":{"id":"20260715125817"}}}]}}`))
	}))
	t.Cleanup(buildhubSrv.Close)

	ctx := context.Background()

	cases := []struct {
		wantErr  string
		inputApp inputApp
	}{
		{"brew API returned status 500", inputApp{Token: "fail", UniqueIdentifier: "abc", InstallerFormat: "pkg"}},
		{"app not found in brew API", inputApp{Token: "notfound", UniqueIdentifier: "abc", InstallerFormat: "pkg"}},
		{"missing name for cask noname", inputApp{Token: "noname", UniqueIdentifier: "abc", InstallerFormat: "pkg"}},
		{"missing name for cask emptyname", inputApp{Token: "emptyname", UniqueIdentifier: "abc", InstallerFormat: "pkg"}},
		{"missing token for cask notoken", inputApp{Token: "notoken", UniqueIdentifier: "abc", InstallerFormat: "pkg"}},
		{"missing version for cask noversion", inputApp{Token: "noversion", UniqueIdentifier: "abc", InstallerFormat: "pkg"}},
		{"missing URL for cask nourl", inputApp{Token: "nourl", UniqueIdentifier: "abc", InstallerFormat: "pkg"}},
		{"parse URL for cask invalidurl", inputApp{Token: "invalidurl", UniqueIdentifier: "abc", InstallerFormat: "pkg"}},
		{"", inputApp{Token: "ok", UniqueIdentifier: "abc", InstallerFormat: "pkg"}},
		{"", inputApp{Token: "docker-desktop", UniqueIdentifier: "com.electron.dockerdesktop", InstallerFormat: "dmg", Name: "Docker Desktop", Slug: "docker-desktop/darwin"}},
		{"", inputApp{Token: "firefox@developer-edition", UniqueIdentifier: "org.mozilla.firefoxdeveloperedition", InstallerFormat: "dmg", Name: "Mozilla Firefox Developer Edition", Slug: "firefox@developer-edition/darwin"}},
		{"", inputApp{Token: "firefox@nightly", UniqueIdentifier: "org.mozilla.nightly", InstallerFormat: "dmg", Name: "Mozilla Firefox Nightly", Slug: "firefox@nightly/darwin"}},
		{"", inputApp{Token: "swiftdialog", UniqueIdentifier: "au.csiro.dialog", InstallerFormat: "pkg", Name: "swiftDialog", Slug: "swiftdialog/darwin"}},
		{"", inputApp{Token: "install_script_path", UniqueIdentifier: "abc", InstallerFormat: "pkg", InstallScriptPath: path.Join(tempDir, "install_script.sh")}},
		{"", inputApp{Token: "uninstall_script_path", UniqueIdentifier: "abc", InstallerFormat: "pkg", UninstallScriptPath: path.Join(tempDir, "uninstall_script.sh")}},
		{"", inputApp{Token: "open-query", UniqueIdentifier: "com.example.app", InstallerFormat: "pkg", Name: "Example App"}},
		{"cannot provide pre-uninstall scripts if uninstall script is provided", inputApp{Token: "uninstall_script_path_with_pre", UniqueIdentifier: "abc", InstallerFormat: "pkg", UninstallScriptPath: path.Join(tempDir, "uninstall_script.sh"), PreUninstallScripts: []string{"foo", "bar"}}},
		{"cannot provide post-uninstall scripts if uninstall script is provided", inputApp{Token: "uninstall_script_path_with_post", UniqueIdentifier: "abc", InstallerFormat: "pkg", UninstallScriptPath: path.Join(tempDir, "uninstall_script.sh"), PostUninstallScripts: []string{"foo", "bar"}}},
	}
	for _, c := range cases {
		t.Run(c.inputApp.Token, func(t *testing.T) {
			i := &brewIngester{
				logger:           slog.New(slog.DiscardHandler),
				client:           fleethttp.NewClient(fleethttp.WithTimeout(10 * time.Second)),
				baseURL:          srv.URL + "/",
				buildhubURL:      buildhubSrv.URL,
				retryInterval:    time.Millisecond,
				retryMaxAttempts: 3,
			}

			out, err := i.ingestOne(ctx, c.inputApp)
			if c.wantErr != "" {
				require.ErrorContains(t, err, c.wantErr)
				return
			}

			require.NoError(t, err)

			if c.inputApp.InstallScriptPath != "" {
				require.Equal(t, testInstallScriptContents, out.InstallScript)
			}

			if c.inputApp.UninstallScriptPath != "" {
				require.Equal(t, testUninstallScriptContents, out.UninstallScript)
			}

			switch c.inputApp.Token {
			case "docker-desktop":
				require.Equal(t, "SELECT 1 FROM apps WHERE bundle_identifier = 'com.electron.dockerdesktop';", out.Queries.Exists)
				require.Equal(t,
					"SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM apps WHERE bundle_identifier = 'com.electron.dockerdesktop' AND path NOT LIKE '%.back' AND version_compare(bundle_short_version, '1.0') < 0);",
					out.Queries.Patched,
				)
			case "firefox@developer-edition":
				// Patched query compares the buildhub-resolved CFBundleVersion.
				require.Equal(t, "153.0b13", out.Version)
				require.Equal(t,
					"SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM apps WHERE bundle_identifier = 'org.mozilla.firefoxdeveloperedition' AND version_compare(bundle_version, '15326.7.15') < 0);",
					out.Queries.Patched,
				)
			case "firefox@nightly":
				// Patched query compares the CFBundleVersion derived from the cask
				// version's build timestamp.
				require.Equal(t, "154.0a1", out.Version)
				require.Equal(t,
					"SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM apps WHERE bundle_identifier = 'org.mozilla.nightly' AND version_compare(bundle_version, '15426.7.17') < 0);",
					out.Queries.Patched,
				)
			case "swiftdialog":
				require.Equal(t, "SELECT 1 FROM apps WHERE bundle_identifier = 'au.csiro.dialog' AND path != '/opt/orbit/bin/swiftDialog/macos/stable/Dialog.app';", out.Queries.Exists)
				require.Equal(t,
					"SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM apps WHERE bundle_identifier = 'au.csiro.dialog' AND path != '/opt/orbit/bin/swiftDialog/macos/stable/Dialog.app' AND version_compare(bundle_short_version, '1.0') < 0);",
					out.Queries.Patched,
				)
			default:
				require.Equal(t,
					fmt.Sprintf("SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM apps WHERE bundle_identifier = '%s' AND version_compare(bundle_short_version, '%s') < 0);", c.inputApp.UniqueIdentifier, out.Version),
					out.Queries.Patched,
				)
			}

			// The managed "is app open" query matches a running process inside the app bundle.
			require.Equal(t,
				fmt.Sprintf("SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM apps a JOIN processes p ON p.path LIKE concat(a.path, '/%%') WHERE a.bundle_identifier = '%s');", out.UniqueIdentifier),
				out.Queries.Open,
			)
		})
	}
}

// TestIngestRetriesTransientErrors verifies that transient brew API failures
// (e.g. the 503s GitHub Pages intermittently returns for formulae.brew.sh) are
// retried instead of aborting the whole ingestion run, while permanent failures
// still return after exhausting attempts.
func TestIngestRetriesTransientErrors(t *testing.T) {
	ctx := context.Background()

	t.Run("recovers after transient 503s", func(t *testing.T) {
		var hits int
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hits++
			// Fail the first two attempts, then succeed.
			if hits < 3 {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
			_ = json.NewEncoder(w).Encode(brewCask{
				Token:   "ok",
				Name:    []string{"ok"},
				URL:     "https://example.com",
				Version: "1.0",
			})
		}))
		t.Cleanup(srv.Close)

		i := &brewIngester{
			logger:           slog.New(slog.DiscardHandler),
			client:           fleethttp.NewClient(fleethttp.WithTimeout(10 * time.Second)),
			baseURL:          srv.URL + "/",
			retryInterval:    time.Millisecond,
			retryMaxAttempts: 5,
		}

		out, err := i.ingestOne(ctx, inputApp{Token: "ok", UniqueIdentifier: "abc", InstallerFormat: "pkg"})
		require.NoError(t, err)
		require.Equal(t, "1.0", out.Version)
		require.Equal(t, 3, hits, "should have retried until success")
	})

	t.Run("gives up after exhausting attempts", func(t *testing.T) {
		var hits int
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			hits++
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		t.Cleanup(srv.Close)

		i := &brewIngester{
			logger:           slog.New(slog.DiscardHandler),
			client:           fleethttp.NewClient(fleethttp.WithTimeout(10 * time.Second)),
			baseURL:          srv.URL + "/",
			retryInterval:    time.Millisecond,
			retryMaxAttempts: 3,
		}

		_, err := i.ingestOne(ctx, inputApp{Token: "fail", UniqueIdentifier: "abc", InstallerFormat: "pkg"})
		require.ErrorContains(t, err, "brew API returned status 503")
		require.Equal(t, 3, hits, "should have attempted exactly retryMaxAttempts times")
	})

	t.Run("does not retry 404", func(t *testing.T) {
		var hits int
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			hits++
			w.WriteHeader(http.StatusNotFound)
		}))
		t.Cleanup(srv.Close)

		i := &brewIngester{
			logger:           slog.New(slog.DiscardHandler),
			client:           fleethttp.NewClient(fleethttp.WithTimeout(10 * time.Second)),
			baseURL:          srv.URL + "/",
			retryInterval:    time.Millisecond,
			retryMaxAttempts: 5,
		}

		_, err := i.ingestOne(ctx, inputApp{Token: "notfound", UniqueIdentifier: "abc", InstallerFormat: "pkg"})
		require.ErrorContains(t, err, "app not found in brew API")
		require.Equal(t, 1, hits, "404 is permanent and must not be retried")
	})
}

// TestIngestCaskPath verifies that when an input app sets cask_path, the
// ingester reads cask JSON from that local file and makes no HTTP call.
// This is the path used for casks committed into inputs/homebrew/custom-tap/.
func TestIngestCaskPath(t *testing.T) {
	tempDir := t.TempDir()

	caskJSON, err := json.Marshal(brewCask{
		Token:   "local-cask",
		Name:    []string{"Local Cask"},
		URL:     "https://example.com/local/installer.pkg",
		Version: "9.9.9",
		SHA256:  "deadbeef",
	})
	require.NoError(t, err)

	caskPath := path.Join(tempDir, "local-cask.json")
	require.NoError(t, os.WriteFile(caskPath, caskJSON, 0o644))

	// Server that should never be called when cask_path is set; any hit is a
	// bug because it means the ingester fell back to HTTP.
	var httpHits int
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		httpHits++
	}))
	t.Cleanup(srv.Close)

	ctx := context.Background()
	i := &brewIngester{
		logger:  slog.New(slog.DiscardHandler),
		client:  fleethttp.NewClient(fleethttp.WithTimeout(10 * time.Second)),
		baseURL: srv.URL + "/",
	}

	out, err := i.ingestOne(ctx, inputApp{
		Token:            "local-cask",
		UniqueIdentifier: "com.example.localcask",
		InstallerFormat:  "pkg",
		Name:             "Local Cask",
		CaskPath:         caskPath,
	})
	require.NoError(t, err)
	require.Equal(t, "https://example.com/local/installer.pkg", out.InstallerURL)
	require.Equal(t, "9.9.9", out.Version)
	require.Equal(t, "deadbeef", out.SHA256)
	require.Equal(t, 0, httpHits, "cask_path path must not make an HTTP call")

	// Missing file yields an actionable error.
	_, err = i.ingestOne(ctx, inputApp{
		Token:            "missing",
		UniqueIdentifier: "com.example.missing",
		InstallerFormat:  "pkg",
		Name:             "Missing",
		CaskPath:         path.Join(tempDir, "does-not-exist.json"),
	})
	require.ErrorContains(t, err, "reading local cask JSON file")
	require.Equal(t, 0, httpHits)

	// Token mismatch between input and cask file is rejected so a misconfigured
	// cask_path can't silently ingest the wrong app.
	mismatchPath := path.Join(tempDir, "mismatch.json")
	mismatchJSON, err := json.Marshal(brewCask{
		Token:   "some-other-cask",
		Name:    []string{"Some Other Cask"},
		URL:     "https://example.com/other/installer.pkg",
		Version: "1.0.0",
		SHA256:  "cafebabe",
	})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(mismatchPath, mismatchJSON, 0o644))

	_, err = i.ingestOne(ctx, inputApp{
		Token:            "local-cask",
		UniqueIdentifier: "com.example.localcask",
		InstallerFormat:  "pkg",
		Name:             "Local Cask",
		CaskPath:         mismatchPath,
	})
	require.ErrorContains(t, err, "does not match input token")
	require.Equal(t, 0, httpHits)

	// Cask file with an empty name is rejected.
	emptyNamePath := path.Join(tempDir, "empty-name.json")
	emptyNameJSON, err := json.Marshal(brewCask{
		Token:   "local-cask",
		Name:    []string{},
		URL:     "https://example.com/local/installer.pkg",
		Version: "9.9.9",
		SHA256:  "deadbeef",
	})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(emptyNamePath, emptyNameJSON, 0o644))

	_, err = i.ingestOne(ctx, inputApp{
		Token:            "local-cask",
		UniqueIdentifier: "com.example.localcask",
		InstallerFormat:  "pkg",
		Name:             "Local Cask",
		CaskPath:         emptyNamePath,
	})
	require.ErrorContains(t, err, "empty name")
	require.Equal(t, 0, httpHits)
}

func TestFirefoxBetaBaseVersion(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"153.0b13", "153.0"},
		{"154.0b1", "154.0"},
		{"153.0.1b2", "153.0.1"},
		{"153.0", "153.0"},
		{"152.0.6", "152.0.6"},
		{"154.0a1", "154.0a1"},
		{"", ""},
	}
	for _, c := range cases {
		require.Equal(t, c.want, firefoxBetaBaseVersion(c.in), "input %q", c.in)
	}
}

func TestFirefoxMacBundleVersion(t *testing.T) {
	cases := []struct {
		version   string
		buildDate string
		want      string
		wantErr   bool
	}{
		{"153.0b13", "20260715125817", "15326.7.15", false},
		{"154.0a1", "20260717", "15426.7.17", false},
		{"153.0.1b2", "20261201000000", "15326.12.1", false},
		{"153.0b13", "2026071", "", true},        // build date too short
		{"153.0b13", "2026x715", "", true},       // build date not numeric
		{"153.0b13", "20261315000000", "", true}, // month out of range
		{"153.0b13", "20260732000000", "", true}, // day out of range
		{"153.0b13", "20260231000000", "", true}, // impossible calendar date
		{"x.0b13", "20260715125817", "", true},   // non-numeric major
		{"", "20260715125817", "", true},
	}
	for _, c := range cases {
		got, err := firefoxMacBundleVersion(c.version, c.buildDate)
		if c.wantErr {
			require.Error(t, err, "version %q buildDate %q", c.version, c.buildDate)
			continue
		}
		require.NoError(t, err, "version %q buildDate %q", c.version, c.buildDate)
		require.Equal(t, c.want, got, "version %q buildDate %q", c.version, c.buildDate)
	}
}

func TestFirefoxNightlyMacBundleVersion(t *testing.T) {
	cases := []struct {
		caskVersion string
		want        string
		wantErr     bool
	}{
		{"154.0a1,2026-07-17-09-27-13", "15426.7.17", false},
		{"154.0a1,2026-07-17", "15426.7.17", false},
		{"154.0a1", "", true},                     // no build timestamp
		{"154.0a1,not-a-date", "", true},          // malformed timestamp
		{"154.0a1,2026-13-17-09-27-13", "", true}, // month out of range
	}
	for _, c := range cases {
		got, err := firefoxNightlyMacBundleVersion(c.caskVersion)
		if c.wantErr {
			require.Error(t, err, "caskVersion %q", c.caskVersion)
			continue
		}
		require.NoError(t, err, "caskVersion %q", c.caskVersion)
		require.Equal(t, c.want, got, "caskVersion %q", c.caskVersion)
	}
}

// TestFirefoxDevEditionBuildhubFallback verifies that a buildhub failure falls
// back to a base-version patch comparison instead of failing ingestion.
func TestFirefoxDevEditionBuildhubFallback(t *testing.T) {
	brewSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := json.NewEncoder(w).Encode(brewCask{
			Token:   "firefox@developer-edition",
			Name:    []string{"Mozilla Firefox Developer Edition"},
			URL:     "https://example.com",
			Version: "153.0b13",
		})
		if err != nil {
			t.Errorf("encoding fixture: %v", err)
		}
	}))
	t.Cleanup(brewSrv.Close)

	cases := []struct {
		name    string
		handler http.HandlerFunc
	}{
		{"buildhub has no matching build", func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"hits":{"hits":[]}}`))
		}},
		{"buildhub is unavailable", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			buildhubSrv := httptest.NewServer(c.handler)
			t.Cleanup(buildhubSrv.Close)

			i := &brewIngester{
				logger:           slog.New(slog.DiscardHandler),
				client:           fleethttp.NewClient(fleethttp.WithTimeout(10 * time.Second)),
				baseURL:          brewSrv.URL + "/",
				buildhubURL:      buildhubSrv.URL,
				retryInterval:    time.Millisecond,
				retryMaxAttempts: 2,
			}

			out, err := i.ingestOne(context.Background(), inputApp{
				Token:            "firefox@developer-edition",
				UniqueIdentifier: "org.mozilla.firefoxdeveloperedition",
				InstallerFormat:  "dmg",
				Name:             "Mozilla Firefox Developer Edition",
				Slug:             "firefox@developer-edition/darwin",
			})
			require.NoError(t, err)
			require.Equal(t, "153.0b13", out.Version)
			require.Equal(t,
				"SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM apps WHERE bundle_identifier = 'org.mozilla.firefoxdeveloperedition' AND version_compare(bundle_short_version, '153.0') < 0);",
				out.Queries.Patched,
			)
		})
	}
}
