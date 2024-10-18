package maintainedapps

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/go-kit/log"
	"github.com/stretchr/testify/require"
)

func TestIngestValidations(t *testing.T) {
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

		case "ok":
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
	ds := new(mock.Store)
	ds.UpsertMaintainedAppFunc = func(ctx context.Context, app *fleet.MaintainedApp) (*fleet.MaintainedApp, error) {
		return nil, nil
	}

	cases := []struct {
		appToken     string
		wantErr      string
		upsertCalled bool
	}{
		{"fail", "brew API returned status 500", false},
		{"notfound", "", false},
		{"noname", "missing name for cask noname", false},
		{"emptyname", "missing name for cask emptyname", false},
		{"notoken", "missing token for cask notoken", false},
		{"noversion", "missing version for cask noversion", false},
		{"nourl", "missing URL for cask nourl", false},
		{"invalidurl", "parse URL for cask invalidurl", false},
		{"ok", "", true},
		{"multi:ok", "", true},
		{"multi:fail", "brew API returned status 500", true},
	}
	for _, c := range cases {
		t.Run(c.appToken, func(t *testing.T) {
			i := ingester{baseURL: srv.URL, ds: ds, logger: log.NewNopLogger()}

			var apps []maintainedApp
			var ignoreDSCheck bool
			if strings.HasPrefix(c.appToken, "multi:") {
				token := strings.TrimPrefix(c.appToken, "multi:")
				if token == "fail" {
					// do not check the DS call, as it may or may not have happened depending
					// on the concurrent execution
					ignoreDSCheck = true
					// send 3 ok, one fail
					apps = []maintainedApp{
						{Identifier: "ok", BundleIdentifier: "abc", InstallerFormat: "pkg"},
						{Identifier: "fail", BundleIdentifier: "def", InstallerFormat: "pkg"},
						{Identifier: "ok", BundleIdentifier: "ghi", InstallerFormat: "pkg"},
						{Identifier: "ok", BundleIdentifier: "klm", InstallerFormat: "pkg"},
					}
				} else {
					// send 4 apps with ok
					apps = []maintainedApp{
						{Identifier: token, BundleIdentifier: "abc", InstallerFormat: "pkg"},
						{Identifier: token, BundleIdentifier: "def", InstallerFormat: "pkg"},
						{Identifier: token, BundleIdentifier: "ghi", InstallerFormat: "pkg"},
						{Identifier: token, BundleIdentifier: "klm", InstallerFormat: "pkg"},
					}
				}
			} else {
				apps = []maintainedApp{
					{Identifier: c.appToken, BundleIdentifier: "abc", InstallerFormat: "pkg"},
				}
			}

			err := i.ingest(ctx, apps)
			if c.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, c.wantErr)
			}

			if !ignoreDSCheck {
				require.Equal(t, c.upsertCalled, ds.UpsertMaintainedAppFuncInvoked)
			}
			ds.UpsertMaintainedAppFuncInvoked = false
		})
	}
}

func TestExtensionForBundleIdentifier(t *testing.T) {
	testCases := []struct {
		name       string
		identifier string
		expected   string
		expectErr  bool
	}{
		{
			name:       "Valid identifier with zip format",
			identifier: "com.1password.1password",
			expected:   "zip",
			expectErr:  false,
		},
		{
			name:       "Valid identifier with dmg format",
			identifier: "com.adobe.Reader",
			expected:   "dmg",
			expectErr:  false,
		},
		{
			name:       "Valid identifier with pkg format",
			identifier: "com.box.desktop",
			expected:   "pkg",
			expectErr:  false,
		},
		{
			name:       "Non-existent identifier",
			identifier: "com.nonexistent.app",
			expected:   "",
			expectErr:  false,
		},
		{
			name:       "Empty identifier",
			identifier: "",
			expected:   "",
			expectErr:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			extension, err := ExtensionForBundleIdentifier(tc.identifier)

			if tc.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			require.Equal(t, tc.expected, extension)
		})
	}
}
