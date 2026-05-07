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
		var cask BrewCask

		appToken := strings.TrimSuffix(path.Base(r.URL.Path), ".json")
		switch appToken {
		case "fail":
			w.WriteHeader(http.StatusInternalServerError)
			return

		case "notfound":
			w.WriteHeader(http.StatusNotFound)
			return

		case "noname":
			cask = BrewCask{
				Token:   appToken,
				Name:    nil,
				URL:     "https://example.com",
				Version: "1.0",
			}

		case "emptyname":
			cask = BrewCask{
				Token:   appToken,
				Name:    []string{""},
				URL:     "https://example.com",
				Version: "1.0",
			}

		case "notoken":
			cask = BrewCask{
				Token:   "",
				Name:    []string{appToken},
				URL:     "https://example.com",
				Version: "1.0",
			}

		case "noversion":
			cask = BrewCask{
				Token:   appToken,
				Name:    []string{appToken},
				URL:     "https://example.com",
				Version: "",
			}

		case "nourl":
			cask = BrewCask{
				Token:   appToken,
				Name:    []string{appToken},
				URL:     "",
				Version: "1.0",
			}

		case "invalidurl":
			cask = BrewCask{
				Token:   appToken,
				Name:    []string{appToken},
				URL:     "https://\x00\x01\x02",
				Version: "1.0",
			}

		case "ok", "install_script_path", "uninstall_script_path", "uninstall_script_path_with_pre", "uninstall_script_path_with_post", "patch_policy_path":
			cask = BrewCask{
				Token:   appToken,
				Name:    []string{appToken},
				URL:     "https://example.com",
				Version: "1.0",
			}

		default:
			w.WriteHeader(http.StatusBadRequest)
			t.Fatalf("unexpected app token %s", appToken)
		}

		err := json.NewEncoder(w).Encode(cask)
		require.NoError(t, err)
	}))
	t.Cleanup(srv.Close)

	ctx := context.Background()

	cases := []struct {
		wantErr  string
		inputApp InputApp
	}{
		{"brew API returned status 500", InputApp{Token: "fail", UniqueIdentifier: "abc", InstallerFormat: "pkg"}},
		{"app not found in brew API", InputApp{Token: "notfound", UniqueIdentifier: "abc", InstallerFormat: "pkg"}},
		{"missing name for cask noname", InputApp{Token: "noname", UniqueIdentifier: "abc", InstallerFormat: "pkg"}},
		{"missing name for cask emptyname", InputApp{Token: "emptyname", UniqueIdentifier: "abc", InstallerFormat: "pkg"}},
		{"missing token for cask notoken", InputApp{Token: "notoken", UniqueIdentifier: "abc", InstallerFormat: "pkg"}},
		{"missing version for cask noversion", InputApp{Token: "noversion", UniqueIdentifier: "abc", InstallerFormat: "pkg"}},
		{"missing URL for cask nourl", InputApp{Token: "nourl", UniqueIdentifier: "abc", InstallerFormat: "pkg"}},
		{"parse URL for cask invalidurl", InputApp{Token: "invalidurl", UniqueIdentifier: "abc", InstallerFormat: "pkg"}},
		{"", InputApp{Token: "ok", UniqueIdentifier: "abc", InstallerFormat: "pkg"}},
		{"", InputApp{Token: "install_script_path", UniqueIdentifier: "abc", InstallerFormat: "pkg", InstallScriptPath: path.Join(tempDir, "install_script.sh")}},
		{"", InputApp{Token: "uninstall_script_path", UniqueIdentifier: "abc", InstallerFormat: "pkg", UninstallScriptPath: path.Join(tempDir, "uninstall_script.sh")}},
		{"cannot provide pre-uninstall scripts if uninstall script is provided", InputApp{Token: "uninstall_script_path_with_pre", UniqueIdentifier: "abc", InstallerFormat: "pkg", UninstallScriptPath: path.Join(tempDir, "uninstall_script.sh"), PreUninstallScripts: []string{"foo", "bar"}}},
		{"cannot provide post-uninstall scripts if uninstall script is provided", InputApp{Token: "uninstall_script_path_with_post", UniqueIdentifier: "abc", InstallerFormat: "pkg", UninstallScriptPath: path.Join(tempDir, "uninstall_script.sh"), PostUninstallScripts: []string{"foo", "bar"}}},
	}
	for _, c := range cases {
		t.Run(c.inputApp.Token, func(t *testing.T) {
			i := &BrewIngester{
				Logger:  slog.New(slog.DiscardHandler),
				Client:  fleethttp.NewClient(fleethttp.WithTimeout(10 * time.Second)),
				BaseURL: srv.URL + "/",
			}

			out, err := i.IngestOne(ctx, c.inputApp)
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

			require.Equal(t,
				fmt.Sprintf("SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM apps WHERE bundle_identifier = '%s' AND version_compare(bundle_short_version, '%s') < 0);", c.inputApp.UniqueIdentifier, out.Version),
				out.Queries.Patched,
			)

		})
	}
}

// TestIngestCaskPath verifies that when an input app sets cask_path, the
// ingester reads cask JSON from that local file and makes no HTTP call.
// This is the path used for casks committed into inputs/homebrew/custom-tap/.
func TestIngestCaskPath(t *testing.T) {
	tempDir := t.TempDir()

	caskJSON, err := json.Marshal(BrewCask{
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
	i := &BrewIngester{
		Logger:  slog.New(slog.DiscardHandler),
		Client:  fleethttp.NewClient(fleethttp.WithTimeout(10 * time.Second)),
		BaseURL: srv.URL + "/",
	}

	out, err := i.IngestOne(ctx, InputApp{
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
	_, err = i.IngestOne(ctx, InputApp{
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
	mismatchJSON, err := json.Marshal(BrewCask{
		Token:   "some-other-cask",
		Name:    []string{"Some Other Cask"},
		URL:     "https://example.com/other/installer.pkg",
		Version: "1.0.0",
		SHA256:  "cafebabe",
	})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(mismatchPath, mismatchJSON, 0o644))

	_, err = i.IngestOne(ctx, InputApp{
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
	emptyNameJSON, err := json.Marshal(BrewCask{
		Token:   "local-cask",
		Name:    []string{},
		URL:     "https://example.com/local/installer.pkg",
		Version: "9.9.9",
		SHA256:  "deadbeef",
	})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(emptyNamePath, emptyNameJSON, 0o644))

	_, err = i.IngestOne(ctx, InputApp{
		Token:            "local-cask",
		UniqueIdentifier: "com.example.localcask",
		InstallerFormat:  "pkg",
		Name:             "Local Cask",
		CaskPath:         emptyNamePath,
	})
	require.ErrorContains(t, err, "empty name")
	require.Equal(t, 0, httpHits)
}
