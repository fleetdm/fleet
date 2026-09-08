package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260902083032(t *testing.T) {
	db := applyUpToPrev(t)

	// Seed an enrollment so the migration is exercised against a non-empty
	// table (device row first: device_id has a foreign key).
	execNoErr(t, db, `
		INSERT INTO nano_devices (id, authenticate)
		VALUES ('device-uuid-1', 'auth-payload')
	`)
	execNoErr(t, db, `
		INSERT INTO nano_enrollments (id, device_id, type, topic, push_magic, token_hex, enabled, last_seen_at)
		VALUES ('device-uuid-1', 'device-uuid-1', 'Device', 'com.apple.mgmt.test', 'magic-1', 'abcdef', 1, NOW())
	`)

	applyNext(t, db)

	rows, err := db.Query(
		`SELECT column_name FROM information_schema.statistics
		 WHERE table_schema = DATABASE() AND table_name = 'nano_enrollments'
		   AND index_name = 'idx_nano_enrollments_enabled'
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
	require.Equal(t, []string{"enabled"}, columns)

	// The seeded row survives the ALTER and is readable through the sweep's predicate.
	var count int
	require.NoError(t, db.QueryRow(
		`SELECT COUNT(*) FROM nano_enrollments WHERE enabled = 1 AND id > ''`,
	).Scan(&count))
	require.Equal(t, 1, count)
}
