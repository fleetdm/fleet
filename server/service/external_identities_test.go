package service

import (
	"context"
	"log/slog"
	"testing"
	"time"

	activity_api "github.com/fleetdm/fleet/v4/server/activity/api"
	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxdb"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/throttled/throttled/v2/store/memstore"
)

func TestSetAPIOnlyUserExternalIdentities(t *testing.T) {
	ds := new(mock.Store)
	cfg := config.TestConfig()
	cfg.Auth.JWTIssuers = []config.JWTIssuerConfig{{Name: "github"}, {Name: "clarion"}}
	opts := &TestServerOpts{}
	svc, ctx := newTestServiceWithConfig(t, ds, cfg, nil, nil, opts)

	var activities []activity_api.ActivityDetails
	opts.ActivityMock.NewActivityFunc = func(_ context.Context, _ *activity_api.User, a activity_api.ActivityDetails) error {
		activities = append(activities, a)
		return nil
	}

	admin := &fleet.User{ID: 1, GlobalRole: new(fleet.RoleAdmin)}
	maintainer := &fleet.User{ID: 2, GlobalRole: new(fleet.RoleMaintainer)}
	apiUser := &fleet.User{ID: 10, Name: "bot", Email: "bot@example.com", APIOnly: true, GlobalRole: new(fleet.RoleObserver)}
	human := &fleet.User{ID: 11, GlobalRole: new(fleet.RoleObserver)}
	apiAdmin := &fleet.User{ID: 12, APIOnly: true, GlobalRole: new(fleet.RoleAdmin)}
	ds.UserByIDFunc = func(_ context.Context, id uint) (*fleet.User, error) {
		for _, u := range []*fleet.User{admin, maintainer, apiUser, human, apiAdmin} {
			if u.ID == id {
				return u, nil
			}
		}
		return nil, &notFoundError{}
	}
	var added, removed []fleet.ExternalIdentity
	ds.SetUserExternalIdentitiesFunc = func(_ context.Context, _ uint, _ []fleet.ExternalIdentity) ([]fleet.ExternalIdentity, []fleet.ExternalIdentity, error) {
		return added, removed, nil
	}
	ds.ListUserExternalIdentitiesFunc = func(_ context.Context, _ uint) ([]fleet.ExternalIdentity, error) {
		return []fleet.ExternalIdentity{}, nil
	}

	as := func(u *fleet.User) context.Context { return viewer.NewContext(ctx, viewer.Viewer{User: u}) }
	gh := func(sub string) fleet.ExternalIdentity { return fleet.ExternalIdentity{Issuer: "github", Subject: sub} }

	t.Run("non-admin is forbidden", func(t *testing.T) {
		_, err := svc.SetAPIOnlyUserExternalIdentities(as(maintainer), apiUser.ID, []fleet.ExternalIdentity{gh("a")})
		var forbidden *authz.Forbidden
		require.ErrorAs(t, err, &forbidden)
		_, _, err = svc.ListAPIOnlyUserExternalIdentities(as(maintainer), apiUser.ID)
		require.ErrorAs(t, err, &forbidden)
		assert.False(t, ds.SetUserExternalIdentitiesFuncInvoked)
	})

	invalid := map[string]struct {
		target uint
		ids    []fleet.ExternalIdentity
	}{
		"non-API-only target": {human.ID, []fleet.ExternalIdentity{gh("a")}},
		"self":                {apiAdmin.ID, []fleet.ExternalIdentity{gh("a")}},
		"unknown issuer":      {apiUser.ID, []fleet.ExternalIdentity{{Issuer: "nope", Subject: "a"}}},
		"empty subject":       {apiUser.ID, []fleet.ExternalIdentity{gh("")}},
		"subject too long":    {apiUser.ID, []fleet.ExternalIdentity{gh(string(make([]byte, 256)))}},
		"duplicate in body":   {apiUser.ID, []fleet.ExternalIdentity{gh("a"), gh("a")}},
		"trailing whitespace": {apiUser.ID, []fleet.ExternalIdentity{gh("a ")}},
		"leading whitespace":  {apiUser.ID, []fleet.ExternalIdentity{gh("\ta")}},
	}
	for name, c := range invalid {
		t.Run(name, func(t *testing.T) {
			actor := admin
			if c.target == apiAdmin.ID {
				actor = apiAdmin
			}
			_, err := svc.SetAPIOnlyUserExternalIdentities(as(actor), c.target, c.ids)
			var invErr *fleet.InvalidArgumentError
			require.ErrorAs(t, err, &invErr)
			assert.False(t, ds.SetUserExternalIdentitiesFuncInvoked)
		})
	}

	t.Run("activity per added and removed identity", func(t *testing.T) {
		activities = nil
		added, removed = []fleet.ExternalIdentity{gh("new")}, []fleet.ExternalIdentity{gh("old1"), gh("old2")}
		_, err := svc.SetAPIOnlyUserExternalIdentities(as(admin), apiUser.ID, []fleet.ExternalIdentity{gh("new")})
		require.NoError(t, err)
		require.Len(t, activities, 3)
		assert.Equal(t, fleet.ActivityTypeAddedUserExternalIdentity{
			UserID: apiUser.ID, UserName: "bot", UserEmail: "bot@example.com", Issuer: "github", Subject: "new",
		}, activities[0])
		assert.Equal(t, "deleted_user_external_identity", activities[1].ActivityName())
		assert.Equal(t, "deleted_user_external_identity", activities[2].ActivityName())
	})

	t.Run("response is read from the primary", func(t *testing.T) {
		var readPrimary bool
		ds.ListUserExternalIdentitiesFunc = func(ctx context.Context, _ uint) ([]fleet.ExternalIdentity, error) {
			readPrimary = ctxdb.IsPrimaryRequired(ctx)
			return []fleet.ExternalIdentity{gh("new")}, nil
		}
		t.Cleanup(func() {
			ds.ListUserExternalIdentitiesFunc = func(_ context.Context, _ uint) ([]fleet.ExternalIdentity, error) {
				return []fleet.ExternalIdentity{}, nil
			}
		})
		ids, err := svc.SetAPIOnlyUserExternalIdentities(as(admin), apiUser.ID, []fleet.ExternalIdentity{gh("new")})
		require.NoError(t, err)
		assert.Equal(t, []fleet.ExternalIdentity{gh("new")}, ids)
		assert.True(t, readPrimary, "a replica could still return the bindings from before the write")
	})

	t.Run("no-op emits no activities", func(t *testing.T) {
		activities = nil
		added, removed = nil, nil
		_, err := svc.SetAPIOnlyUserExternalIdentities(as(admin), apiUser.ID, []fleet.ExternalIdentity{gh("new")})
		require.NoError(t, err)
		assert.Empty(t, activities)
	})
}

// A viewer authenticated by an external JWT carries a user but no session.
func TestSessionlessViewer(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)
	bot := &fleet.User{ID: 7, APIOnly: true, GlobalRole: new(fleet.RoleObserver)}
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: bot})

	t.Run("/me returns the user", func(t *testing.T) {
		u, err := svc.AuthenticatedUser(ctx)
		require.NoError(t, err)
		require.Equal(t, bot.ID, u.ID)
	})

	t.Run("logout explains there is no session", func(t *testing.T) {
		err := svc.DestroySession(ctx)
		var invErr *fleet.InvalidArgumentError
		require.ErrorAs(t, err, &invErr)
		assert.Contains(t, err.Error(), "external token")
		assert.False(t, ds.SessionByIDFuncInvoked)
	})

	t.Run("forced password reset is refused without panicking", func(t *testing.T) {
		forced := *bot
		forced.AdminForcedPasswordReset = true
		ctx := viewer.NewContext(ctx, viewer.Viewer{User: &forced})
		require.NotPanics(t, func() {
			_, err := svc.PerformRequiredPasswordReset(ctx, "new-Passw0rd!!")
			var permErr *fleet.PermissionError
			require.ErrorAs(t, err, &permErr)
		})
	})
}

func TestNewExternalAuthenticatorRateDefault(t *testing.T) {
	cfg := config.TestConfig()
	cfg.Auth.JWTIssuers = []config.JWTIssuerConfig{{
		Name: "t", Issuer: "http://127.0.0.1:1", Audience: "fleet", SubjectClaim: "sub",
		AllowedAlgorithms: []string{"RS256"}, MaxTokenLifetime: time.Minute,
	}}
	store, err := memstore.New(10)
	require.NoError(t, err)
	for _, rate := range []int{0, -1} {
		cfg.Auth.JWTRateLimitPerMinute = rate
		require.NotPanics(t, func() {
			a, err := NewExternalAuthenticator(t.Context(), cfg, new(mock.Store), store, slog.New(slog.DiscardHandler))
			require.NoError(t, err)
			require.NotNil(t, a)
		})
	}
	cfg.Auth.JWTIssuers = nil
	a, err := NewExternalAuthenticator(t.Context(), cfg, new(mock.Store), store, slog.New(slog.DiscardHandler))
	require.NoError(t, err)
	require.Nil(t, a)
}
