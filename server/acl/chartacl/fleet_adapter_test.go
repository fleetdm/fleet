package chartacl

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestViewerScopeNoViewerInContextErrors(t *testing.T) {
	a := NewFleetViewerAdapter()
	_, _, err := a.ViewerScope(t.Context())
	require.Error(t, err, "missing viewer should fail closed, not silently return global=false")
}

func TestViewerScopeGlobalUser(t *testing.T) {
	a := NewFleetViewerAdapter()

	role := fleet.RoleAdmin
	ctx := viewer.NewContext(t.Context(), viewer.Viewer{
		User: &fleet.User{GlobalRole: &role},
	})

	isGlobal, teamIDs, err := a.ViewerScope(ctx)
	require.NoError(t, err)
	assert.True(t, isGlobal)
	assert.Nil(t, teamIDs, "global user's team list is not used by the caller")
}

func TestViewerScopeGlobalUserEmptyRoleStringIsNotGlobal(t *testing.T) {
	// Defensive: a User with a non-nil but empty GlobalRole string should not
	// be treated as global. This matches the User.HasAnyGlobalRole semantics.
	a := NewFleetViewerAdapter()

	empty := ""
	ctx := viewer.NewContext(t.Context(), viewer.Viewer{
		User: &fleet.User{GlobalRole: &empty},
	})

	isGlobal, _, err := a.ViewerScope(ctx)
	require.NoError(t, err)
	assert.False(t, isGlobal)
}

func TestViewerScopeTeamUser(t *testing.T) {
	a := NewFleetViewerAdapter()

	ctx := viewer.NewContext(t.Context(), viewer.Viewer{
		User: &fleet.User{
			Teams: []fleet.UserTeam{
				{Team: fleet.Team{ID: 3}, Role: fleet.RoleObserver},
				{Team: fleet.Team{ID: 7}, Role: fleet.RoleMaintainer},
			},
		},
	})

	isGlobal, teamIDs, err := a.ViewerScope(ctx)
	require.NoError(t, err)
	assert.False(t, isGlobal)
	assert.Equal(t, []uint{3, 7}, teamIDs)
}

func TestViewerScopeTeamUserNoTeams(t *testing.T) {
	// Authenticated user with neither a global role nor any team memberships.
	// The chart service treats this as "team-scoped with zero teams" and
	// returns empty data — the adapter's job is just to report the scope
	// faithfully.
	a := NewFleetViewerAdapter()

	ctx := viewer.NewContext(t.Context(), viewer.Viewer{
		User: &fleet.User{},
	})

	isGlobal, teamIDs, err := a.ViewerScope(ctx)
	require.NoError(t, err)
	assert.False(t, isGlobal)
	assert.Empty(t, teamIDs)
}
