package authz

import (
	"errors"
	"testing"

	authz_ctx "github.com/fleetdm/fleet/v4/server/contexts/authz"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/require"
)

func TestAuthorizeOrNotFound(t *testing.T) {
	notFoundErr := errors.New("not found sentinel")
	teamHost := &fleet.Host{TeamID: new(uint(1))}

	t.Run("write allowed", func(t *testing.T) {
		ctx := test.UserContext(t.Context(), &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleAdmin}}})
		err := auth.AuthorizeOrNotFound(ctx, teamHost, fleet.ActionWrite, notFoundErr)
		require.NoError(t, err)
	})

	t.Run("write denied but read allowed returns the write error, not masked", func(t *testing.T) {
		// A team observer can read the host but can't write it: this is not
		// an existence oracle (the caller already knows the host exists), so
		// the real Forbidden should surface, not notFoundErr.
		ctx := test.UserContext(t.Context(), &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserver}}})
		err := auth.AuthorizeOrNotFound(ctx, teamHost, fleet.ActionWrite, notFoundErr)
		require.Error(t, err)
		require.NotErrorIs(t, err, notFoundErr)
		var forbidden *Forbidden
		require.ErrorAs(t, err, &forbidden)
	})

	t.Run("write denied and read denied masks as notFoundErr", func(t *testing.T) {
		// A caller with no relationship to the host's team can't read or
		// write it: masking as notFoundErr prevents them from learning the
		// host exists on some other team via a distinguishable Forbidden.
		ctx := test.UserContext(t.Context(), &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 2}, Role: fleet.RoleObserver}}})
		err := auth.AuthorizeOrNotFound(ctx, teamHost, fleet.ActionWrite, notFoundErr)
		require.Error(t, err)
		require.ErrorIs(t, err, notFoundErr)
	})

	t.Run("nil notFoundErr never fails open", func(t *testing.T) {
		// A caller misusing this helper by passing a nil notFoundErr must
		// never get nil (success) back for a caller who can neither read nor
		// write the resource: that would silently bypass authorization.
		ctx := test.UserContext(t.Context(), &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 2}, Role: fleet.RoleObserver}}})
		err := auth.AuthorizeOrNotFound(ctx, teamHost, fleet.ActionWrite, nil)
		require.Error(t, err)
		var forbidden *Forbidden
		require.ErrorAs(t, err, &forbidden)
	})
}

func TestCanWriteSecretVariables(t *testing.T) {
	teamUser := func(role string) *fleet.User {
		ut := fleet.UserTeam{Role: role}
		ut.Team.ID = 1
		return &fleet.User{Teams: []fleet.UserTeam{ut}}
	}

	for _, c := range []struct {
		name string
		user *fleet.User
		want bool
	}{
		{"global admin", &fleet.User{GlobalRole: new(fleet.RoleAdmin)}, true},
		{"global maintainer", &fleet.User{GlobalRole: new(fleet.RoleMaintainer)}, true},
		{"global gitops", &fleet.User{GlobalRole: new(fleet.RoleGitOps)}, true},
		{"global observer", &fleet.User{GlobalRole: new(fleet.RoleObserver)}, false},
		{"team admin", teamUser(fleet.RoleAdmin), false},
		{"team maintainer", teamUser(fleet.RoleMaintainer), false},
		{"team gitops", teamUser(fleet.RoleGitOps), false},
		{"team observer", teamUser(fleet.RoleObserver), false},
	} {
		t.Run(c.name, func(t *testing.T) {
			ctx := test.UserContext(t.Context(), c.user)
			require.Equal(t, c.want, auth.CanWriteSecretVariables(ctx))
		})
	}

	t.Run("marks the authorization context as checked", func(t *testing.T) {
		// The probe goes through Authorize, which sets the checked flag, so
		// callers must run it after their own Authorize call.
		ctx := authz_ctx.NewContext(t.Context(), &authz_ctx.AuthorizationContext{})
		ctx = test.UserContext(ctx, teamUser(fleet.RoleMaintainer))
		authctx, ok := authz_ctx.FromContext(ctx)
		require.True(t, ok)
		require.False(t, authctx.Checked())

		require.False(t, auth.CanWriteSecretVariables(ctx))
		require.True(t, authctx.Checked())
	})
}
