package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260807151355(t *testing.T) {
	db := applyUpToPrev(t)

	// Seed a command so the migration is exercised against a non-empty table.
	execNoErr(t, db, `
		INSERT INTO mdm_android_commands (command_uuid, host_uuid, operation_name, command_type, status)
		VALUES ('cmd-uuid-1', 'host-uuid-1', 'enterprises/e1/devices/d1/operations/op1', 'LOCK', 'pending')
	`)

	applyNext(t, db)

	rows, err := db.Query(
		`SELECT column_name FROM information_schema.statistics
		 WHERE table_schema = DATABASE() AND table_name = 'mdm_android_commands'
		   AND index_name = 'idx_mdm_android_commands_status_created_at'
		 ORDER BY seq_in_index`,
	)
	require.NoError(t, err)
	defer rows.Close()

	var columns []string
	for rows.Next() {
		var columnName string
		require.NoError(t, rows.Scan(&columnName))
		columns = append(columns, columnName)
	}
	require.NoError(t, rows.Err())
	require.Equal(t, []string{"status", "created_at"}, columns)

	// The seeded row survives the ALTER and is still readable through the new index's predicate.
	var count int
	require.NoError(t, db.QueryRow(
		`SELECT COUNT(*) FROM mdm_android_commands WHERE status = 'pending' AND created_at < NOW(6)`,
	).Scan(&count))
	require.Equal(t, 1, count)
}
