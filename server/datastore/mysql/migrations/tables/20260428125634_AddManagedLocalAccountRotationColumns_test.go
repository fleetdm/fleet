package tables

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUp_20260428125634(t *testing.T) {
	db := applyUpToPrev(t)

	// Insert a row before the migration to verify the ALTER works on existing data.
	_, err := db.Exec(`
		INSERT INTO host_managed_local_account_passwords
			(host_uuid, encrypted_password, command_uuid, status)
		VALUES (?, ?, ?, ?)`,
		"host-uuid-existing", []byte("enc-pw"), "cmd-uuid-existing", "verified",
	)
	require.NoError(t, err)

	applyNext(t, db)

	// New columns default to NULL / 0 on pre-existing row.
	var (
		accountUUID              *string
		autoRotateAt             *time.Time
		pendingEncryptedPassword []byte
		pendingCommandUUID       *string
		initiatedByFleet         bool
	)
	err = db.QueryRow(`
		SELECT account_uuid, auto_rotate_at, pending_encrypted_password, pending_command_uuid, initiated_by_fleet
		FROM host_managed_local_account_passwords
		WHERE host_uuid = ?`, "host-uuid-existing",
	).Scan(&accountUUID, &autoRotateAt, &pendingEncryptedPassword, &pendingCommandUUID, &initiatedByFleet)
	require.NoError(t, err)
	assert.Nil(t, accountUUID)
	assert.Nil(t, autoRotateAt)
	assert.Nil(t, pendingEncryptedPassword)
	assert.Nil(t, pendingCommandUUID)
	assert.False(t, initiatedByFleet)

	// Insert a row exercising all new columns.
	rotateAt := time.Now().Add(time.Hour).UTC().Truncate(time.Microsecond)
	_, err = db.Exec(`
		INSERT INTO host_managed_local_account_passwords
			(host_uuid, encrypted_password, command_uuid, status,
			 account_uuid, auto_rotate_at, pending_encrypted_password, pending_command_uuid, initiated_by_fleet)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"host-uuid-new", []byte("enc-pw-2"), "cmd-uuid-new", "pending",
		"AAAAAAAA-BBBB-CCCC-DDDD-EEEEEEEEEEEE", rotateAt, []byte("pending-enc-pw"), "pending-cmd-uuid", 1,
	)
	require.NoError(t, err)

	err = db.QueryRow(`
		SELECT account_uuid, auto_rotate_at, pending_encrypted_password, pending_command_uuid, initiated_by_fleet
		FROM host_managed_local_account_passwords
		WHERE host_uuid = ?`, "host-uuid-new",
	).Scan(&accountUUID, &autoRotateAt, &pendingEncryptedPassword, &pendingCommandUUID, &initiatedByFleet)
	require.NoError(t, err)
	require.NotNil(t, accountUUID)
	assert.Equal(t, "AAAAAAAA-BBBB-CCCC-DDDD-EEEEEEEEEEEE", *accountUUID)
	require.NotNil(t, autoRotateAt)
	assert.WithinDuration(t, rotateAt, *autoRotateAt, time.Second)
	assert.Equal(t, []byte("pending-enc-pw"), pendingEncryptedPassword)
	require.NotNil(t, pendingCommandUUID)
	assert.Equal(t, "pending-cmd-uuid", *pendingCommandUUID)
	assert.True(t, initiatedByFleet)

	// UPDATE using WHERE account_uuid IS NULL only touches the untouched row.
	res, err := db.Exec(`
		UPDATE host_managed_local_account_passwords
		SET account_uuid = ?
		WHERE host_uuid = ? AND account_uuid IS NULL`,
		"11111111-2222-3333-4444-555555555555", "host-uuid-existing",
	)
	require.NoError(t, err)
	affected, err := res.RowsAffected()
	require.NoError(t, err)
	assert.Equal(t, int64(1), affected)

	// Re-running the same UPDATE is a no-op.
	res, err = db.Exec(`
		UPDATE host_managed_local_account_passwords
		SET account_uuid = ?
		WHERE host_uuid = ? AND account_uuid IS NULL`,
		"11111111-2222-3333-4444-555555555555", "host-uuid-existing",
	)
	require.NoError(t, err)
	affected, err = res.RowsAffected()
	require.NoError(t, err)
	assert.Equal(t, int64(0), affected)

	// Index on auto_rotate_at exists.
	var idxCount int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM information_schema.statistics
		WHERE table_schema = DATABASE()
		  AND table_name = 'host_managed_local_account_passwords'
		  AND index_name = 'idx_hmlap_auto_rotate_at'`,
	).Scan(&idxCount)
	require.NoError(t, err)
	assert.Equal(t, 1, idxCount)

	// Query using the index works.
	rows, err := db.Query(`
		SELECT host_uuid FROM host_managed_local_account_passwords
		WHERE auto_rotate_at IS NOT NULL AND auto_rotate_at <= ?`,
		time.Now().Add(2*time.Hour),
	)
	require.NoError(t, err)
	defer rows.Close()
	var hostUUIDs []string
	for rows.Next() {
		var u string
		require.NoError(t, rows.Scan(&u))
		hostUUIDs = append(hostUUIDs, u)
	}
	require.NoError(t, rows.Err())
	assert.Equal(t, []string{"host-uuid-new"}, hostUUIDs)
}
