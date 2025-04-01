package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	ma "github.com/fleetdm/fleet/v4/ee/maintained-apps"
	"github.com/fleetdm/fleet/v4/server/authz"
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
