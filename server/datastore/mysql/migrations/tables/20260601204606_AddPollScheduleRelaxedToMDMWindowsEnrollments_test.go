package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260601204606(t *testing.T) {
	db := applyUpToPrev(t)

	// applyNext runs the ALTER plus the has_pending_commands backfill UPDATE; this exercises that SQL.
	applyNext(t, db)

	// Both columns exist and default to 0 (the aggressive poll / no-pending-commands defaults). The backfill's
	// EXISTS logic is the same as the recompute and is covered by testMDMWindowsGetHostConfigState.
	for _, col := range []string{"poll_schedule_relaxed", "has_pending_commands"} {
		_, err := db.Exec(`SELECT ` + col + ` FROM mdm_windows_enrollments LIMIT 0`)
		require.NoError(t, err, "column %s should exist", col)

		var def string
		require.NoError(t, db.Get(&def,
			`SELECT COLUMN_DEFAULT FROM INFORMATION_SCHEMA.COLUMNS
			 WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'mdm_windows_enrollments' AND COLUMN_NAME = ?`, col))
		require.Equal(t, "0", def, "column %s should default to 0", col)
	}
}
