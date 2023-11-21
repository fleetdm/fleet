package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20230411102858(t *testing.T) {
	db := applyUpToPrev(t)
	applyNext(t, db)

	_, err := db.Exec(`
          INSERT INTO nano_commands (command_uuid, request_type, command)
          VALUES ('command-uuid', 'foo', '<?xml')
	`)
	require.NoError(t, err)

	insertStmt := "INSERT INTO host_mdm_apple_bootstrap_packages (host_uuid, command_uuid) VALUES (?, ?)"
	_, err = db.Exec(insertStmt, "host-uuid", "command-uuid")
	require.NoError(t, err)

	_, err = db.Exec(insertStmt, "host-uuid-2", "command-uuid")
	require.NoError(t, err)

	_, err = db.Exec(insertStmt, "host-uuid-3", "not-exists")
	require.ErrorContains(t, err, "Error 1452")

	_, err = db.Exec(insertStmt, "host-uuid", "command-uuid")
	require.ErrorContains(t, err, "Error 1062")

	var count int
	err = db.Get(&count, `SELECT COUNT(*) FROM host_mdm_apple_bootstrap_packages`)
	require.NoError(t, err)
	require.Equal(t, 2, count)

	// deleting from nano_commands cascades the deletion of this too
	_, err = db.Exec("DELETE FROM nano_commands WHERE command_uuid = ?", "command-uuid")
	require.NoError(t, err)
	err = db.Get(&count, `SELECT COUNT(*) FROM host_mdm_apple_bootstrap_packages`)
	require.NoError(t, err)
	require.Zero(t, count)
}
