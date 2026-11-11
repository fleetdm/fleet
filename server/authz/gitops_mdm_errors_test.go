package authz

import (
	"errors"
	"testing"

	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/require"
)

func TestGitOpsMDMCommandErrors(t *testing.T) {
	authzErr := errors.New("authz failed")
	fleetGitOps := &fleet.User{Teams: []fleet.UserTeam{{ID: 1, Role: fleet.RoleGitOps}}}
	cases := []struct {
		name            string
		user            *fleet.User
		teamID          *uint
		wantHostMessage bool // GitOpsMDMCommandError
		wantListMessage bool // GitOpsHostListError
	}{
		{"global gitops, no fleet", test.UserGitOps, nil, true, true},
		{"global gitops, fleet", test.UserGitOps, new(uint(1)), true, true},
		{"fleet gitops, own fleet", fleetGitOps, new(uint(1)), true, true},
		{"fleet gitops, other fleet", fleetGitOps, new(uint(2)), false, true},
		{"fleet gitops, no fleet", fleetGitOps, nil, false, true},
		{"fleet gitops on another fleet, maintainer on this one", &fleet.User{Teams: []fleet.UserTeam{
			{ID: 1, Role: fleet.RoleMaintainer}, {ID: 2, Role: fleet.RoleGitOps},
		}}, new(uint(1)), false, true},
		{"global maintainer", test.UserMaintainer, nil, false, false},
		{"fleet observer", test.UserTeamObserverTeam1, new(uint(1)), false, false},
		{"no viewer", nil, nil, false, false},
	}
	requireMessage := func(t *testing.T, wantMessage bool, err error) {
		t.Helper()
		if !wantMessage {
			require.Equal(t, authzErr, err)
			return
		}
		var forbidden *Forbidden
		require.ErrorAs(t, err, &forbidden)
		require.Equal(t, fleet.GitOpsMDMCommandErrorMessage, err.Error())
		require.Equal(t, authzErr.Error(), forbidden.Internal())
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctx := t.Context()
			if c.user != nil {
				ctx = viewer.NewContext(ctx, viewer.Viewer{User: c.user})
			}
			requireMessage(t, c.wantHostMessage, GitOpsMDMCommandError(ctx, c.teamID, authzErr))
			requireMessage(t, c.wantListMessage, GitOpsHostListError(ctx, authzErr))
		})
	}
}
