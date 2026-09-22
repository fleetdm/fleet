package mysql

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestAppleMDMCleanups(t *testing.T) {
	ds := CreateMySQLDS(t)
	ctx := t.Context()

	t.Run("bare datastore cleanup state is a no-op", func(t *testing.T) {
		state, err := ds.GetMDMAppleCommandCleanupState(ctx)
		require.NoError(t, err)
		require.Nil(t, state)
		require.NoError(t, ds.SetMDMAppleCommandCleanupState(ctx, &fleet.MDMAppleCommandCleanupState{
			Retention: map[string]fleet.MDMAppleCommandCleanupCursor{"short:Acknowledged": {ID: "x", CommandUUID: "y"}},
		}))
		state, err = ds.GetMDMAppleCommandCleanupState(ctx)
		require.NoError(t, err)
		require.Nil(t, state, "the bare datastore never persists cleanup state")
	})
}
