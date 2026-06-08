package tables

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUp_20260409153717(t *testing.T) {
	db := applyUpToPrev(t)

	// Apply current migration.
	applyNext(t, db)

	// INSERT with NULL status (pending).
	_, err := db.Exec(`
		INSERT INTO host_managed_local_account_passwords
			(host_uuid, encrypted_password, command_uuid, status)
		VALUES (?, ?, ?, NULL)`,
		"host-uuid-1", []byte("encrypted-pw-1"), "cmd-uuid-1",
	)
	require.NoError(t, err)

	// INSERT with valid status.
	_, err = db.Exec(`
		INSERT INTO host_managed_local_account_passwords
			(host_uuid, encrypted_password, command_uuid, status)
		VALUES (?, ?, ?, ?)`,
		"host-uuid-2", []byte("encrypted-pw-2"), "cmd-uuid-2", "verified",
	)
	require.NoError(t, err)

	// FK constraint rejects invalid status.
	_, err = db.Exec(`
		INSERT INTO host_managed_local_account_passwords
			(host_uuid, encrypted_password, command_uuid, status)
		VALUES (?, ?, ?, ?)`,
		"host-uuid-3", []byte("encrypted-pw-3"), "cmd-uuid-3", "bogus_status",
	)
	require.Error(t, err)

	// Upsert via ON DUPLICATE KEY UPDATE.
	_, err = db.Exec(`
		INSERT INTO host_managed_local_account_passwords
			(host_uuid, encrypted_password, command_uuid, status)
		VALUES (?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			encrypted_password = VALUES(encrypted_password),
			command_uuid = VALUES(command_uuid),
			status = VALUES(status)`,
		"host-uuid-1", []byte("new-encrypted-pw"), "cmd-uuid-new", "verified",
	)
	require.NoError(t, err)

	// Verify the upsert updated the row.
	var (
		encPw   []byte
		cmdUUID string
		status  *string
	)
	err = db.QueryRow(`
		SELECT encrypted_password, command_uuid, status
		FROM host_managed_local_account_passwords
		WHERE host_uuid = ?`, "host-uuid-1",
	).Scan(&encPw, &cmdUUID, &status)
	require.NoError(t, err)
	assert.Equal(t, []byte("new-encrypted-pw"), encPw)
	assert.Equal(t, "cmd-uuid-new", cmdUUID)
	require.NotNil(t, status)
	assert.Equal(t, "verified", *status)

	// Timestamps auto-populate.
	var createdAt, updatedAt time.Time
	err = db.QueryRow(`
		SELECT created_at, updated_at
		FROM host_managed_local_account_passwords
		WHERE host_uuid = ?`, "host-uuid-2",
	).Scan(&createdAt, &updatedAt)
	require.NoError(t, err)
	assert.False(t, createdAt.IsZero())
	assert.False(t, updatedAt.IsZero())
}
