package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	ma "github.com/fleetdm/fleet/v4/ee/maintained-apps"
	"github.com/fleetdm/fleet/v4/server/authz"
	authz_ctx "github.com/fleetdm/fleet/v4/server/contexts/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/require"
)

func TestListMaintainedAppsAuth(t *testing.T) {
	t.Parallel()
	ds := new(mock.Store)
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}
	ds.ListAvailableFleetMaintainedAppsFunc = func(ctx context.Context, teamID *uint, opt fleet.ListOptions) ([]fleet.MaintainedApp, *fleet.PaginationMetadata, error) {
		return []fleet.MaintainedApp{}, &fleet.PaginationMetadata{}, nil
	}
	authorizer, err := authz.NewAuthorizer()
	require.NoError(t, err)
	svc := &Service{authz: authorizer, ds: ds}

	testCases := []struct {
		name                        string
		user                        *fleet.User
		shouldFailWithNoTeam        bool
		shouldFailWithMatchingTeam  bool
		shouldFailWithDifferentTeam bool
	}{
		{
			"global admin",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
			false,
			false,
			false,
		},
		{
			"global maintainer",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleMaintainer)},
			false,
			false,
			false,
		},
		{
			"global observer",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleObserver)},
			true,
			true,
			true,
		},
		{
			"team admin",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleAdmin}}},
			false,
			false,
			true,
		},
		{
			"team maintainer",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleMaintainer}}},
			false,
			false,
			true,
		},
		{
			"team observer",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserver}}},
			true,
			true,
			true,
		},
	}

	var forbiddenError *authz.Forbidden
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := viewer.NewContext(context.Background(), viewer.Viewer{User: tt.user})

			_, _, err := svc.ListFleetMaintainedApps(ctx, nil, fleet.ListOptions{})
			if tt.shouldFailWithNoTeam {
				require.Error(t, err)
				require.ErrorAs(t, err, &forbiddenError)
			} else {
				require.NoError(t, err)
			}

			_, _, err = svc.ListFleetMaintainedApps(ctx, ptr.Uint(1), fleet.ListOptions{})
			if tt.shouldFailWithMatchingTeam {
				require.Error(t, err)
				require.ErrorAs(t, err, &forbiddenError)
			} else {
				require.NoError(t, err)
			}

			_, _, err = svc.ListFleetMaintainedApps(ctx, ptr.Uint(2), fleet.ListOptions{})
			if tt.shouldFailWithDifferentTeam {
				require.Error(t, err)
				require.ErrorAs(t, err, &forbiddenError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestGetMaintainedAppAuth(t *testing.T) {
	ds := new(mock.Store)
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}
	ds.GetMaintainedAppByIDFunc = func(ctx context.Context, appID uint, teamID *uint) (*fleet.MaintainedApp, error) {
		return &fleet.MaintainedApp{Slug: "1password/darwin"}, nil
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slug := strings.TrimPrefix(strings.TrimSuffix(r.URL.Path, ".json"), "/")
		var manifest ma.FMAManifestFile
		switch slug {
		case "fail":
			w.WriteHeader(http.StatusInternalServerError)
			return

		case "notfound":
			w.WriteHeader(http.StatusNotFound)
			return

		case "1password/darwin":
			var versions []*ma.FMAManifestApp
			versions = append(versions, &ma.FMAManifestApp{
				Version: "1",
				Queries: ma.FMAQueries{
					Exists: "SELECT 1 FROM osquery_info;",
				},
				InstallerURL:       "https://google.com",
				InstallScriptRef:   "foobaz",
				UninstallScriptRef: "foobaz",
				SHA256:             "deadbeef",
			})

			manifest = ma.FMAManifestFile{
				Versions: versions,
				Refs: map[string]string{
					"foobaz": "Hello World!",
				},
			}

		default:
			w.WriteHeader(http.StatusBadRequest)
			t.Fatalf("unexpected app token %s", slug)
		}

		err := json.NewEncoder(w).Encode(manifest)
		require.NoError(t, err)
	}))
	t.Cleanup(srv.Close)

	authorizer, err := authz.NewAuthorizer()
	require.NoError(t, err)
	svc := &Service{authz: authorizer, ds: ds}

	testCases := []struct {
		name                        string
		user                        *fleet.User
		shouldFailWithNoTeam        bool
		shouldFailWithMatchingTeam  bool
		shouldFailWithDifferentTeam bool
	}{
		{
			"global admin",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
			false,
			false,
			false,
		},
		{
			"global maintainer",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleMaintainer)},
			false,
			false,
			false,
		},
		{
			"global observer",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleObserver)},
			true,
			true,
			true,
		},
		{
			"team admin",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleAdmin}}},
			false,
			false,
			true,
		},
		{
			"team maintainer",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleMaintainer}}},
			false,
			false,
			true,
		},
		{
			"team observer",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserver}}},
			true,
			true,
			true,
		},
	}

	var forbiddenError *authz.Forbidden
	require.NoError(t, os.Setenv("FLEET_DEV_MAINTAINED_APPS_BASE_URL", srv.URL))
	defer os.Unsetenv("FLEET_DEV_MAINTAINED_APPS_BASE_URL")
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			ctx := viewer.NewContext(context.Background(), viewer.Viewer{User: tt.user})
			_, err := svc.GetFleetMaintainedApp(ctx, 123, nil)

			if tt.shouldFailWithNoTeam {
				require.Error(t, err)
				require.ErrorAs(t, err, &forbiddenError)
			} else {
				require.NoError(t, err)
			}

			_, err = svc.GetFleetMaintainedApp(ctx, 1, ptr.Uint(1))
			if tt.shouldFailWithMatchingTeam {
				require.Error(t, err)
				require.ErrorAs(t, err, &forbiddenError)
			} else {
				require.NoError(t, err)
			}

			_, err = svc.GetFleetMaintainedApp(ctx, 1, ptr.Uint(2))
			if tt.shouldFailWithDifferentTeam {
				require.Error(t, err)
				require.ErrorAs(t, err, &forbiddenError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestAddFleetMaintainedApp(t *testing.T) {
	installerBytes := []byte("abc")

	// this is the hash we expect to get in the DB
	h := sha256.New()
	_, err := h.Write(installerBytes)
	require.NoError(t, err)
	spoofedSHA := hex.EncodeToString(h.Sum(nil))

	ds := new(mock.Store)
	ds.ValidateEmbeddedSecretsFunc = func(ctx context.Context, documents []string) error {
		return nil
	}
	ds.GetMaintainedAppByIDFunc = func(ctx context.Context, appID uint, teamID *uint) (*fleet.MaintainedApp, error) {
		return &fleet.MaintainedApp{
			ID:               1,
			Name:             "Internet Exploder",
			Slug:             "iexplode/windows",
			Platform:         "windows",
			TitleID:          nil,
			UniqueIdentifier: "Internet Exploder",
		}, nil
	}
	ds.GetSoftwareCategoryIDsFunc = func(ctx context.Context, names []string) ([]uint, error) {
		return []uint{}, nil
	}

	// Mock server to serve the "installer"
	installerServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(installerBytes)
	}))
	defer installerServer.Close()
	ds.MatchOrCreateSoftwareInstallerFunc = func(ctx context.Context, payload *fleet.UploadSoftwareInstallerPayload) (uint, uint, error) {
		require.Equal(t, spoofedSHA, payload.StorageID)
		require.Empty(t, payload.BundleIdentifier)
		require.Equal(t, "Internet Exploder", payload.Title)
		require.Equal(t, "programs", payload.Source)
		require.Equal(t, "Hello World!", payload.InstallScript)
		require.Equal(t, "Hello World!", payload.UninstallScript)
		require.Equal(t, installerServer.URL+"/iexplode.exe", payload.URL)

		// Can't easily inject a proper fleet.service so we bail early before NewActivity gets called and panics
		return 0, 0, errors.New("forced error to short-circuit storage and activity creation")
	}

	// Mock server to serve the manifest
	manifestServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var versions []*ma.FMAManifestApp
		versions = append(versions, &ma.FMAManifestApp{
			Version: "6.0",
			Queries: ma.FMAQueries{
				Exists: "SELECT 1 FROM osquery_info;",
			},
			InstallerURL:       installerServer.URL + "/iexplode.exe",
			InstallScriptRef:   "foobaz",
			UninstallScriptRef: "foobaz",
			SHA256:             noCheckHash,
		})

		manifest := ma.FMAManifestFile{
			Versions: versions,
			Refs: map[string]string{
				"foobaz": "Hello World!",
			},
		}

		err := json.NewEncoder(w).Encode(manifest)
		require.NoError(t, err)
	}))

	t.Cleanup(manifestServer.Close)
	os.Setenv("FLEET_DEV_MAINTAINED_APPS_BASE_URL", manifestServer.URL)
	defer os.Unsetenv("FLEET_DEV_MAINTAINED_APPS_BASE_URL")

	svc := newTestService(t, ds)

	authCtx := authz_ctx.AuthorizationContext{}
	ctx := authz_ctx.NewContext(context.Background(), &authCtx)
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}})
	_, err = svc.AddFleetMaintainedApp(ctx, nil, 1, "", "", "", "", false, false, nil, nil)
	require.ErrorContains(t, err, "forced error to short-circuit storage and activity creation")

	require.True(t, ds.MatchOrCreateSoftwareInstallerFuncInvoked)
}

func TestExtractMaintainedAppVersionWhenLatest(t *testing.T) {
	installerBytes, err := os.ReadFile(filepath.Join("testdata", "dummy_installer.pkg"))
	require.NoError(t, err)

	// this is the hash we expect to get in the DB
	h := sha256.New()
	_, err = h.Write(installerBytes)
	require.NoError(t, err)
	spoofedSHA := hex.EncodeToString(h.Sum(nil))

	ds := new(mock.Store)
	ds.ValidateEmbeddedSecretsFunc = func(ctx context.Context, documents []string) error {
		return nil
	}
	ds.GetMaintainedAppByIDFunc = func(ctx context.Context, appID uint, teamID *uint) (*fleet.MaintainedApp, error) {
		return &fleet.MaintainedApp{
			ID:               1,
			Name:             "Dummy",
			Slug:             "dummy/darwin",
			Platform:         "darwin",
			TitleID:          nil,
			UniqueIdentifier: "com.example.dummy",
		}, nil
	}
	ds.GetSoftwareCategoryIDsFunc = func(ctx context.Context, names []string) ([]uint, error) {
		return []uint{}, nil
	}

	// Mock server to serve the dummy package
	installerServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(installerBytes)
	}))
	defer installerServer.Close()
	ds.MatchOrCreateSoftwareInstallerFunc = func(ctx context.Context, payload *fleet.UploadSoftwareInstallerPayload) (uint, uint, error) {
		require.Equal(t, spoofedSHA, payload.StorageID)
		require.Equal(t, "1.0.0", payload.Version)

		// Can't easily inject a proper fleet.service so we bail early before NewActivity gets called and panics
		return 0, 0, errors.New("forced error to short-circuit storage and activity creation")
	}

	// Mock server to serve the manifest
	manifestServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var versions []*ma.FMAManifestApp
		versions = append(versions, &ma.FMAManifestApp{
			Version: "latest",
			Queries: ma.FMAQueries{
				Exists: "SELECT 1 FROM osquery_info;",
			},
			InstallerURL:       installerServer.URL + "/dummy.pkg",
			InstallScriptRef:   "foobaz",
			UninstallScriptRef: "foobaz",
			SHA256:             noCheckHash,
		})

		manifest := ma.FMAManifestFile{
			Versions: versions,
			Refs: map[string]string{
				"foobaz": "Hello World!",
			},
		}

		err := json.NewEncoder(w).Encode(manifest)
		require.NoError(t, err)
	}))

	t.Cleanup(manifestServer.Close)
	os.Setenv("FLEET_DEV_MAINTAINED_APPS_BASE_URL", manifestServer.URL)
	defer os.Unsetenv("FLEET_DEV_MAINTAINED_APPS_BASE_URL")

	svc := newTestService(t, ds)

	authCtx := authz_ctx.AuthorizationContext{}
	ctx := authz_ctx.NewContext(context.Background(), &authCtx)
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}})
	_, err = svc.AddFleetMaintainedApp(ctx, nil, 1, "", "", "", "", false, false, nil, nil)
	require.ErrorContains(t, err, "forced error to short-circuit storage and activity creation")

	require.True(t, ds.MatchOrCreateSoftwareInstallerFuncInvoked)
}
