package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	ma "github.com/fleetdm/fleet/v4/ee/maintained-apps"
	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/fleetdm/fleet/v4/server/mock"
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
pkgids="com.foo"
they are "com.foo", right $MY_SECRET?
quotes for "com.foo"
blah"com.foo"withConcat
quotes and braces for "com.foo"
"com.foo"`
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
  "com.foo"
  "com.bar"
)
they are (
  "com.foo"
  "com.bar"
), right $MY_SECRET?
quotes for (
  "com.foo"
  "com.bar"
)
blah(
  "com.foo"
  "com.bar"
)withConcat
quotes and braces for (
  "com.foo"
  "com.bar"
)
(
  "com.foo"
  "com.bar"
)`
	assert.Equal(t, expected, payload.UninstallScript)

	payload.UninstallScript = "$UPGRADE_CODE"
	require.Error(t, preProcessUninstallScript(&payload))

	payload.UpgradeCode = "foo"
	require.NoError(t, preProcessUninstallScript(&payload))
	assert.Equal(t, `"foo"`, payload.UninstallScript)
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

func TestInstallSoftwareTitle(t *testing.T) {
	t.Parallel()
	ds := new(mock.Store)
	svc := newTestService(t, ds)

	ds.GetInHouseAppMetadataByTeamAndTitleIDFunc = func(ctx context.Context, teamID *uint, titleID uint) (*fleet.SoftwareInstaller, error) {
		return nil, nil
	}

	ctx := viewer.NewContext(context.Background(), viewer.Viewer{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}})

	host := &fleet.Host{
		UUID:         "personal-ios",
		OrbitNodeKey: ptr.String("orbit_key"),
		Platform:     "ios",
		TeamID:       ptr.Uint(1),
	}

	ds.HostFunc = func(ctx context.Context, id uint) (*fleet.Host, error) {
		return host, nil
	}

	ds.GetNanoMDMEnrollmentFunc = func(ctx context.Context, id string) (*fleet.NanoEnrollment, error) {
		return &fleet.NanoEnrollment{
			Type: mdm.EnrollType(mdm.UserEnrollmentDevice).String(),
		}, nil
	}

	require.ErrorContains(t, svc.InstallSoftwareTitle(ctx, 1, 10), fleet.InstallSoftwarePersonalAppleDeviceErrMsg)
}

func TestSoftwareInstallerPayloadFromSlug(t *testing.T) {
	t.Parallel()
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
	os.Setenv("FLEET_DEV_MAINTAINED_APPS_BASE_URL", manifestServer.URL)
	defer os.Unsetenv("FLEET_DEV_MAINTAINED_APPS_BASE_URL")

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
			}, nil
		}

		return nil, notFoundError{}
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
            <string>https://example.com/api/latest/fleet/software/titles/1/in_house_app?team_id=0</string>
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
		require.Equal(t, "scripts", payload.Source)
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
		require.Equal(t, "scripts", payload.Source)
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
