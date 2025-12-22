package homebrew

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleethttp"
	"github.com/go-kit/log"
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

		case "ok", "install_script_path", "uninstall_script_path", "uninstall_script_path_with_pre", "uninstall_script_path_with_post":
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
				logger:  log.NewNopLogger(),
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

		})
	}
}
