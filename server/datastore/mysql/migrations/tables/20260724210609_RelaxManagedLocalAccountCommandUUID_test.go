package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260724210609(t *testing.T) {
	db := applyUpToPrev(t)

	const hostUUID = "windows-oobe-host"

	// Before the migration command_uuid is NOT NULL, so a Windows-shaped row (no MDM command)
	// cannot be inserted.
	_, err := db.Exec(`
		INSERT INTO host_managed_local_account_passwords (host_uuid, encrypted_password, command_uuid, status)
		VALUES (?, ?, NULL, 'verified')`, hostUUID, []byte("enc"))
	require.Error(t, err)

	applyNext(t, db)

	// After the migration the same Windows-shaped row with a NULL command_uuid inserts cleanly.
	_, err = db.Exec(`
		INSERT INTO host_managed_local_account_passwords (host_uuid, encrypted_password, command_uuid, status)
		VALUES (?, ?, NULL, 'verified')`, hostUUID, []byte("enc"))
	require.NoError(t, err)

	var commandUUID *string
	require.NoError(t, db.QueryRow(
		`SELECT command_uuid FROM host_managed_local_account_passwords WHERE host_uuid = ?`, hostUUID,
	).Scan(&commandUUID))
	require.Nil(t, commandUUID)

	// A macOS-shaped row with a populated command_uuid still inserts, confirming the column
	// stays usable for the existing path.
	_, err = db.Exec(`
		INSERT INTO host_managed_local_account_passwords (host_uuid, encrypted_password, command_uuid, status)
		VALUES (?, ?, ?, 'verified')`, "macos-host", []byte("enc"), "some-command-uuid")
	require.NoError(t, err)
}
