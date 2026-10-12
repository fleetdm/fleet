package tables

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUp_20260902124249(t *testing.T) {
	db := applyUpToPrev(t)

	// A live host with a known password, a soft-deleted one (unenrolled, but the device may
	// still be holding the lock), and one that never got a password set.
	execNoErr(t, db, `
		INSERT INTO host_recovery_key_passwords
			(host_uuid, encrypted_password, pending_encrypted_password, status, operation_type, set_command_uuid, deleted)
		VALUES
			('live-uuid',      'live-cipher',    'pending-cipher', 'verified', 'install', 'set-cmd-live',    0),
			('unenrolled-uuid','deleted-cipher', NULL,             'verified', 'install', 'set-cmd-deleted', 1),
			('no-password-uuid', NULL,           'pending-only',   'pending',  'install', NULL,              0)
	`)

	applyNext(t, db)

	type archived struct {
		HostUUID          string         `db:"host_uuid"`
		EncryptedPassword string         `db:"encrypted_password"`
		SetCommandUUID    sql.NullString `db:"set_command_uuid"`
	}

	rows, err := db.Query(`
		SELECT host_uuid, encrypted_password, set_command_uuid
		FROM host_recovery_key_password_archive ORDER BY host_uuid`)
	require.NoError(t, err)
	defer rows.Close()

	var got []archived
	for rows.Next() {
		var a archived
		require.NoError(t, rows.Scan(&a.HostUUID, &a.EncryptedPassword, &a.SetCommandUUID))
		got = append(got, a)
	}
	require.NoError(t, rows.Err())

	// The unenrolled host is seeded too — that is the case the archive exists for. The host
	// with no password has nothing to seed, and pending passwords are not seeded: they were
	// never confirmed and are re-generated on the next attempt anyway.
	require.Len(t, got, 2)
	assert.Equal(t, "live-uuid", got[0].HostUUID)
	assert.Equal(t, "live-cipher", got[0].EncryptedPassword)
	assert.Equal(t, "set-cmd-live", got[0].SetCommandUUID.String)
	assert.Equal(t, "unenrolled-uuid", got[1].HostUUID)
	assert.Equal(t, "deleted-cipher", got[1].EncryptedPassword)
	assert.Equal(t, "set-cmd-deleted", got[1].SetCommandUUID.String)

	// A host can accumulate several candidates, so nothing may constrain host_uuid to one row.
	execNoErr(t, db, `
		INSERT INTO host_recovery_key_password_archive (host_uuid, encrypted_password, set_command_uuid)
		VALUES ('live-uuid', 'second-cipher', 'set-cmd-live-2')
	`)
	var count int
	require.NoError(t, db.QueryRow(
		`SELECT COUNT(*) FROM host_recovery_key_password_archive WHERE host_uuid = 'live-uuid'`).Scan(&count))
	assert.Equal(t, 2, count)
}
