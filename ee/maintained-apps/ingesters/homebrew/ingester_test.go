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
	"github.com/stretchr/testify/assert"
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

		case "ok", "install_script_path", "uninstall_script_path", "uninstall_script_path_with_pre", "uninstall_script_path_with_post", "patch_policy_path":
			cask = brewCask{
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
		{"", inputApp{Token: "install_script_path", UniqueIdentifier: "abc", InstallerFormat: "pkg", InstallScriptPath: path.Join(tempDir, "install_script.sh")}},
		{"", inputApp{Token: "uninstall_script_path", UniqueIdentifier: "abc", InstallerFormat: "pkg", UninstallScriptPath: path.Join(tempDir, "uninstall_script.sh")}},
		{"cannot provide pre-uninstall scripts if uninstall script is provided", inputApp{Token: "uninstall_script_path_with_pre", UniqueIdentifier: "abc", InstallerFormat: "pkg", UninstallScriptPath: path.Join(tempDir, "uninstall_script.sh"), PreUninstallScripts: []string{"foo", "bar"}}},
		{"cannot provide post-uninstall scripts if uninstall script is provided", inputApp{Token: "uninstall_script_path_with_post", UniqueIdentifier: "abc", InstallerFormat: "pkg", UninstallScriptPath: path.Join(tempDir, "uninstall_script.sh"), PostUninstallScripts: []string{"foo", "bar"}}},
	}
	for _, c := range cases {
		t.Run(c.inputApp.Token, func(t *testing.T) {
			i := &brewIngester{
				logger:  slog.New(slog.DiscardHandler),
				client:  fleethttp.NewClient(fleethttp.WithTimeout(10 * time.Second)),
				baseURL: srv.URL + "/",
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

			require.Equal(t,
				fmt.Sprintf("SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM apps WHERE bundle_identifier = '%s' AND version_compare(bundle_short_version, '%s') < 0);", c.inputApp.UniqueIdentifier, out.Version),
				out.Queries.Patched,
			)

		})
	}
}

// TestIngestCustomAPIBaseURL verifies that when an input app sets
// api_base_url, the ingester fetches cask metadata from that host instead of
// the ingester's default base URL. This supports ingesting casks from
// third-party taps that publish a brew-API-compatible JSON endpoint.
func TestIngestCustomAPIBaseURL(t *testing.T) {
	var defaultHits, overrideHits int

	defaultSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defaultHits++
		// Should never be called for an app with api_base_url set.
		appToken := strings.TrimSuffix(path.Base(r.URL.Path), ".json")
		cask := brewCask{
			Token:   appToken,
			Name:    []string{appToken},
			URL:     "https://example.com/default",
			Version: "1.0",
		}
		// Use assert (not require) inside handlers: require's FailNow only
		// exits the handler goroutine, not the test. testifylint go-require.
		assert.NoError(t, json.NewEncoder(w).Encode(cask))
	}))
	t.Cleanup(defaultSrv.Close)

	overrideSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		overrideHits++
		// Ensure the path layout matches what the ingester constructs:
		// "<baseURL>/cask/<token>.json"
		assert.True(t, strings.HasPrefix(r.URL.Path, "/cask/"), "unexpected path: %s", r.URL.Path)
		appToken := strings.TrimSuffix(path.Base(r.URL.Path), ".json")
		cask := brewCask{
			Token:   appToken,
			Name:    []string{appToken},
			URL:     "https://example.com/override",
			Version: "2.0",
		}
		assert.NoError(t, json.NewEncoder(w).Encode(cask))
	}))
	t.Cleanup(overrideSrv.Close)

	ctx := context.Background()
	i := &brewIngester{
		logger:  slog.New(slog.DiscardHandler),
		client:  fleethttp.NewClient(fleethttp.WithTimeout(10 * time.Second)),
		baseURL: defaultSrv.URL + "/",
	}

	// App without api_base_url uses the default.
	out, err := i.ingestOne(ctx, inputApp{
		Token:            "vanilla",
		UniqueIdentifier: "com.example.vanilla",
		InstallerFormat:  "pkg",
		Name:             "Vanilla",
	})
	require.NoError(t, err)
	require.Equal(t, "https://example.com/default", out.InstallerURL)
	require.Equal(t, "1.0", out.Version)
	require.Equal(t, 1, defaultHits)
	require.Equal(t, 0, overrideHits)

	// App with api_base_url routes to the override host. Intentionally omit the
	// trailing slash to exercise normalization.
	out, err = i.ingestOne(ctx, inputApp{
		Token:            "tapped",
		UniqueIdentifier: "com.example.tapped",
		InstallerFormat:  "pkg",
		Name:             "Tapped",
		APIBaseURL:       overrideSrv.URL,
	})
	require.NoError(t, err)
	require.Equal(t, "https://example.com/override", out.InstallerURL)
	require.Equal(t, "2.0", out.Version)
	require.Equal(t, 1, defaultHits, "default server should not have been hit again")
	require.Equal(t, 1, overrideHits)

	// Override with a trailing slash should also work.
	out, err = i.ingestOne(ctx, inputApp{
		Token:            "tapped-slash",
		UniqueIdentifier: "com.example.tappedslash",
		InstallerFormat:  "pkg",
		Name:             "TappedSlash",
		APIBaseURL:       overrideSrv.URL + "/",
	})
	require.NoError(t, err)
	require.Equal(t, "https://example.com/override", out.InstallerURL)
	require.Equal(t, 1, defaultHits)
	require.Equal(t, 2, overrideHits)
}
