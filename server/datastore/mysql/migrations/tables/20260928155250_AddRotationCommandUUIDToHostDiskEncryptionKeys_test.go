package tables

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUp_20260928155250(t *testing.T) {
	db := applyUpToPrev(t)

	updatedAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	execNoErr(t, db, `
		INSERT INTO host_disk_encryption_keys (host_id, base64_encrypted, decryptable, updated_at)
		VALUES (1, 'enc', 1, ?)`, updatedAt)

	applyNext(t, db)

	var row struct {
		RotationCommandUUID sql.NullString `db:"rotation_command_uuid"`
		UpdatedAt           time.Time      `db:"updated_at"`
	}
	require.NoError(t, db.Get(&row, `SELECT rotation_command_uuid, updated_at FROM host_disk_encryption_keys WHERE host_id = 1`))
	require.False(t, row.RotationCommandUUID.Valid)
	require.True(t, updatedAt.Equal(row.UpdatedAt), "updated_at changed: %s", row.UpdatedAt)

	require.Equal(t, []string{"rotation_command_uuid"},
		indexColumns(t, db, "host_disk_encryption_keys", "idx_hdek_rotation_command_uuid"))

	execNoErr(t, db, `UPDATE host_disk_encryption_keys SET rotation_command_uuid = 'cmd-1' WHERE host_id = 1`)
	var hostID uint
	require.NoError(t, db.Get(&hostID, `SELECT host_id FROM host_disk_encryption_keys WHERE rotation_command_uuid = ?`, "cmd-1"))
	require.EqualValues(t, 1, hostID)

	tx, err := db.Begin()
	require.NoError(t, err)
	require.NoError(t, Up_20260928155250(tx))
	require.NoError(t, tx.Commit())
}
