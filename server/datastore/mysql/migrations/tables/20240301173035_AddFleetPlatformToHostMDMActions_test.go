package tables

import (
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestUp_20240301173035(t *testing.T) {
	db := applyUpToPrev(t)

	// create an existing host_mdm_actions row
	_, err := db.Exec("INSERT INTO host_mdm_actions (host_id, lock_ref) VALUES (1, 'a')")
	require.NoError(t, err)

	applyNext(t, db)

	var hostActions []struct {
		HostID        uint    `db:"host_id"`
		LockRef       *string `db:"lock_ref"`
		FleetPlatform string  `db:"fleet_platform"`
	}

	// fleet platform is left empty for pre-existing rows
	err = sqlx.Select(db, &hostActions, `SELECT host_id, lock_ref, fleet_platform FROM host_mdm_actions`)
	require.NoError(t, err)
	require.Len(t, hostActions, 1)
	require.Equal(t, uint(1), hostActions[0].HostID)
	require.NotNil(t, hostActions[0].LockRef)
	require.Equal(t, "a", *hostActions[0].LockRef)
	require.Empty(t, hostActions[0].FleetPlatform)
}
