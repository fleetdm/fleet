package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260609143000(t *testing.T) {
	db := applyUpToPrev(t)

	applyNext(t, db)

	// The table exists and accepts a retained profile row.
	_, err := db.Exec(`INSERT INTO mdm_windows_configuration_profiles_pending_delete
		(profile_uuid, team_id, name, syncml, created_at)
		VALUES (?, ?, ?, ?, NOW(6))`,
		"w-pending-1", 0, "Test Profile", []byte("<Replace></Replace>"))
	require.NoError(t, err)

	// profile_uuid is the primary key: re-inserting the same UUID conflicts.
	_, err = db.Exec(`INSERT INTO mdm_windows_configuration_profiles_pending_delete
		(profile_uuid, team_id, name, syncml, created_at)
		VALUES (?, ?, ?, ?, NOW(6))`,
		"w-pending-1", 0, "Test Profile", []byte("<Replace></Replace>"))
	require.Error(t, err)

	// The created_at index supports the GC range delete.
	res, err := db.Exec(`DELETE FROM mdm_windows_configuration_profiles_pending_delete WHERE created_at < NOW(6) + INTERVAL 1 DAY`)
	require.NoError(t, err)
	affected, err := res.RowsAffected()
	require.NoError(t, err)
	require.Equal(t, int64(1), affected)
}
