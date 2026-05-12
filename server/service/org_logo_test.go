package service

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"testing"

	activity_api "github.com/fleetdm/fleet/v4/server/activity/api"
	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/require"
)

func TestOrgLogoAuth(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}
	ds.SaveAppConfigFunc = func(ctx context.Context, conf *fleet.AppConfig) error {
		return nil
	}

	testCases := []struct {
		name            string
		user            *fleet.User
		shouldFailWrite bool // PUT and DELETE
	}{
		{
			"global admin",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)},
			false,
		},
		{
			"global maintainer",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleMaintainer)},
			true,
		},
		{
			"global observer",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleObserver)},
			true,
		},
		{
			"global observer+",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleObserverPlus)},
			true,
		},
		{
			// Global gitops can write app_config (per the rego policy),
			// which is what fleetctl gitops uses to upload custom org
			// logos via the new org_logo_path_*_mode keys.
			"global gitops",
			&fleet.User{GlobalRole: ptr.String(fleet.RoleGitOps)},
			false,
		},
		{
			"team admin",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleAdmin}}},
			true,
		},
		{
			"team maintainer",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleMaintainer}}},
			true,
		},
		{
			"team observer",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserver}}},
			true,
		},
		{
			"team observer+",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserverPlus}}},
			true,
		},
		{
			"team gitops",
			&fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleGitOps}}},
			true,
		},
		{
			"user without roles",
			&fleet.User{ID: 777},
			true,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			authedCtx := viewer.NewContext(ctx, viewer.Viewer{User: tt.user})

			err := svc.UploadOrgLogo(authedCtx, fleet.OrgLogoModeLight, bytes.NewReader([]byte{}))
			checkOrgLogoAuth(t, tt.shouldFailWrite, err)

			err = svc.DeleteOrgLogo(authedCtx, fleet.OrgLogoModeLight)
			checkOrgLogoAuth(t, tt.shouldFailWrite, err)

			// GET is public — never an authz failure regardless of viewer.
			_, _, err = svc.GetOrgLogo(authedCtx, fleet.OrgLogoModeLight)
			checkOrgLogoAuth(t, false, err)
		})
	}

	// GET should also work without any viewer in the context (login page
	// case). It may still fail downstream because no store is wired, but
	// that's not an authz failure.
	t.Run("public GET without viewer", func(t *testing.T) {
		_, _, err := svc.GetOrgLogo(ctx, fleet.OrgLogoModeLight)
		checkOrgLogoAuth(t, false, err)
	})
}

// TestDeleteOrgLogoExternalURL covers the case where the configured logo URL
// points at an external host (e.g. https://placehold.co/100) — the legacy
// behavior errored because nothing exists in the store to delete, but the
// user still needs the URL field cleared.
func TestDeleteOrgLogoExternalURL(t *testing.T) {
	t.Run("dark mode only, external URL", func(t *testing.T) {
		ds := new(mock.Store)
		opts := &TestServerOpts{}
		svc, ctx := newTestService(t, ds, nil, nil, opts)
		ctx = viewer.NewContext(ctx, viewer.Viewer{
			User: &fleet.User{ID: 1, GlobalRole: ptr.String(fleet.RoleAdmin)},
		})

		dsAppConfig := &fleet.AppConfig{
			OrgInfo: fleet.OrgInfo{
				OrgLogoURL:         "https://placehold.co/100",
				OrgLogoURLDarkMode: "https://placehold.co/100",
			},
		}
		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			return dsAppConfig, nil
		}
		var saved *fleet.AppConfig
		ds.SaveAppConfigFunc = func(ctx context.Context, conf *fleet.AppConfig) error {
			saved = conf
			return nil
		}

		var activityFired bool
		opts.ActivityMock.NewActivityFunc = func(_ context.Context, _ *activity_api.User, act activity_api.ActivityDetails) error {
			if _, ok := act.(fleet.ActivityTypeDeletedOrgLogo); ok {
				activityFired = true
			}
			return nil
		}

		require.NoError(t, svc.DeleteOrgLogo(ctx, fleet.OrgLogoModeDark))
		require.True(t, activityFired)
		require.NotNil(t, saved)
		require.Empty(t, saved.OrgInfo.OrgLogoURL)
		require.Empty(t, saved.OrgInfo.OrgLogoURLDarkMode)
	})

	t.Run("mode=all with mixed Fleet-hosted and external URLs", func(t *testing.T) {
		ds := new(mock.Store)
		opts := &TestServerOpts{}
		svc, ctx := newTestService(t, ds, nil, nil, opts)
		ctx = viewer.NewContext(ctx, viewer.Viewer{
			User: &fleet.User{ID: 1, GlobalRole: ptr.String(fleet.RoleAdmin)},
		})

		// Light mode has a Fleet-hosted URL (would normally have a blob in
		// the store — we leave the store empty so Exists returns false,
		// which is treated the same as "already deleted" by the existing
		// idempotency path). Dark mode is external.
		dsAppConfig := &fleet.AppConfig{
			OrgInfo: fleet.OrgInfo{
				OrgLogoURLLightMode:       "/api/latest/fleet/logo?mode=light&v=123",
				OrgLogoURLLightBackground: "/api/latest/fleet/logo?mode=light&v=123",
				OrgLogoURL:                "https://placehold.co/100",
				OrgLogoURLDarkMode:        "https://placehold.co/100",
			},
		}
		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			return dsAppConfig, nil
		}
		var saved *fleet.AppConfig
		ds.SaveAppConfigFunc = func(ctx context.Context, conf *fleet.AppConfig) error {
			saved = conf
			return nil
		}
		opts.ActivityMock.NewActivityFunc = func(_ context.Context, _ *activity_api.User, _ activity_api.ActivityDetails) error {
			return nil
		}

		require.NoError(t, svc.DeleteOrgLogo(ctx, fleet.OrgLogoModeAll))
		require.NotNil(t, saved)
		require.Empty(t, saved.OrgInfo.OrgLogoURLLightMode)
		require.Empty(t, saved.OrgInfo.OrgLogoURLLightBackground)
		require.Empty(t, saved.OrgInfo.OrgLogoURLDarkMode)
		require.Empty(t, saved.OrgInfo.OrgLogoURL)
	})

	t.Run("URL already empty but blob remains (gitops PATCH-then-delete flow)", func(t *testing.T) {
		// In the GitOps clearing path, doGitOpsOrgLogos PATCHes the URL to ""
		// first, then calls DeleteOrgLogo to drop the blob. DeleteOrgLogo
		// must still find and remove the blob even though the URL is
		// already empty.
		ds := new(mock.Store)
		opts := &TestServerOpts{}
		svc, ctx := newTestService(t, ds, nil, nil, opts)
		ctx = viewer.NewContext(ctx, viewer.Viewer{
			User: &fleet.User{ID: 1, GlobalRole: ptr.String(fleet.RoleAdmin)},
		})

		dsAppConfig := &fleet.AppConfig{}
		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			return dsAppConfig, nil
		}
		ds.SaveAppConfigFunc = func(ctx context.Context, conf *fleet.AppConfig) error {
			*dsAppConfig = *conf
			return nil
		}
		opts.ActivityMock.NewActivityFunc = func(_ context.Context, _ *activity_api.User, _ activity_api.ActivityDetails) error {
			return nil
		}

		// Plant a blob in the store for dark mode via UploadOrgLogo so the
		// real filesystem store wired by newTestService has something to
		// delete. Then zero out the URL fields to simulate the
		// post-PATCH state GitOps lands in before invoking DeleteOrgLogo.
		pngImg := image.NewRGBA(image.Rect(0, 0, 1, 1))
		pngImg.Set(0, 0, color.RGBA{R: 0, G: 128, B: 0, A: 255})
		var pngBuf bytes.Buffer
		require.NoError(t, png.Encode(&pngBuf, pngImg))
		require.NoError(t, svc.UploadOrgLogo(ctx, fleet.OrgLogoModeDark, bytes.NewReader(pngBuf.Bytes())))
		dsAppConfig.OrgInfo.OrgLogoURL = ""
		dsAppConfig.OrgInfo.OrgLogoURLDarkMode = ""

		require.NoError(t, svc.DeleteOrgLogo(ctx, fleet.OrgLogoModeDark))
	})

	t.Run("nothing to clear returns BadRequest", func(t *testing.T) {
		ds := new(mock.Store)
		opts := &TestServerOpts{}
		svc, ctx := newTestService(t, ds, nil, nil, opts)
		ctx = viewer.NewContext(ctx, viewer.Viewer{
			User: &fleet.User{ID: 1, GlobalRole: ptr.String(fleet.RoleAdmin)},
		})

		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			return &fleet.AppConfig{}, nil
		}
		ds.SaveAppConfigFunc = func(ctx context.Context, conf *fleet.AppConfig) error {
			return nil
		}

		err := svc.DeleteOrgLogo(ctx, fleet.OrgLogoModeDark)
		require.Error(t, err)
		var br *fleet.BadRequestError
		require.ErrorAs(t, err, &br)
	})

	t.Run("legacy deprecated field only", func(t *testing.T) {
		// Mirrors the issue's repro: an external URL is written directly to
		// the DB under the deprecated org_logo_url field (so the new
		// mode-aware field is empty until NormalizeLogoFields runs). DELETE
		// must still clear it.
		ds := new(mock.Store)
		opts := &TestServerOpts{}
		svc, ctx := newTestService(t, ds, nil, nil, opts)
		ctx = viewer.NewContext(ctx, viewer.Viewer{
			User: &fleet.User{ID: 1, GlobalRole: ptr.String(fleet.RoleAdmin)},
		})

		dsAppConfig := &fleet.AppConfig{
			OrgInfo: fleet.OrgInfo{
				OrgLogoURL: "https://placehold.co/100",
			},
		}
		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			return dsAppConfig, nil
		}
		var saved *fleet.AppConfig
		ds.SaveAppConfigFunc = func(ctx context.Context, conf *fleet.AppConfig) error {
			saved = conf
			return nil
		}
		opts.ActivityMock.NewActivityFunc = func(_ context.Context, _ *activity_api.User, _ activity_api.ActivityDetails) error {
			return nil
		}

		require.NoError(t, svc.DeleteOrgLogo(ctx, fleet.OrgLogoModeDark))
		require.NotNil(t, saved)
		require.Empty(t, saved.OrgInfo.OrgLogoURL)
		require.Empty(t, saved.OrgInfo.OrgLogoURLDarkMode)
	})
}

func checkOrgLogoAuth(t *testing.T, shouldFail bool, err error) {
	t.Helper()
	var forbidden *authz.Forbidden
	if shouldFail {
		require.Error(t, err)
		require.ErrorAs(t, err, &forbidden, "expected authz Forbidden, got %T: %v", err, err)
		return
	}
	if err != nil {
		require.NotErrorAs(t, err, &forbidden,
			"expected non-authz error, got authz Forbidden: %v", err)
	}
}
