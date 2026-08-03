package authz

import (
	"errors"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/require"
)

func TestAuthorizeOrNotFound(t *testing.T) {
	notFoundErr := errors.New("not found sentinel")
	teamHost := &fleet.Host{TeamID: new(uint(1))}

	t.Run("write allowed", func(t *testing.T) {
		ctx := test.UserContext(t.Context(), &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleAdmin}}})
		err := auth.AuthorizeOrNotFound(ctx, teamHost, fleet.ActionWrite, teamHost, notFoundErr)
		require.NoError(t, err)
	})

	t.Run("write denied but read allowed returns the write error, not masked", func(t *testing.T) {
		// A team observer can read the host but can't write it: this is not
		// an existence oracle (the caller already knows the host exists), so
		// the real Forbidden should surface, not notFoundErr.
		ctx := test.UserContext(t.Context(), &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 1}, Role: fleet.RoleObserver}}})
		err := auth.AuthorizeOrNotFound(ctx, teamHost, fleet.ActionWrite, teamHost, notFoundErr)
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
		err := auth.AuthorizeOrNotFound(ctx, teamHost, fleet.ActionWrite, teamHost, notFoundErr)
		require.Error(t, err)
		require.ErrorIs(t, err, notFoundErr)
	})

	t.Run("nil notFoundErr never fails open", func(t *testing.T) {
		// A caller misusing this helper by passing a nil notFoundErr must
		// never get nil (success) back for a caller who can neither read nor
		// write the resource: that would silently bypass authorization.
		ctx := test.UserContext(t.Context(), &fleet.User{Teams: []fleet.UserTeam{{Team: fleet.Team{ID: 2}, Role: fleet.RoleObserver}}})
		err := auth.AuthorizeOrNotFound(ctx, teamHost, fleet.ActionWrite, teamHost, nil)
		require.Error(t, err)
		var forbidden *Forbidden
		require.ErrorAs(t, err, &forbidden)
	})
}
