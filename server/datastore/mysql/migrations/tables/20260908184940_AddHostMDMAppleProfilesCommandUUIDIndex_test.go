package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260908184940(t *testing.T) {
	db := applyUpToPrev(t)

	execNoErr(t, db, `
		INSERT INTO host_mdm_apple_profiles
			(host_uuid, profile_uuid, profile_identifier, command_uuid, checksum, operation_type, status)
		VALUES
			('host-1', 'prof-1', 'com.example.one', 'cmd-current', UNHEX(MD5('a')), 'install', 'verified'),
			('host-1', 'prof-2', 'com.example.two', 'cmd-other', UNHEX(MD5('b')), 'install', 'pending')`)

	applyNext(t, db)

	require.Equal(t, []string{"command_uuid"}, indexColumns(t, db, "host_mdm_apple_profiles", "idx_hmap_command_uuid"))

	// Seeded rows survive the ALTER and the guard probe resolves by command UUID.
	var count int
	require.NoError(t, db.Get(&count, `SELECT COUNT(*) FROM host_mdm_apple_profiles WHERE command_uuid = ?`, "cmd-current"))
	require.Equal(t, 1, count)
	require.NoError(t, db.Get(&count, `SELECT COUNT(*) FROM host_mdm_apple_profiles WHERE command_uuid = ?`, "cmd-superseded"))
	require.Equal(t, 0, count)
}

func TestUp_20260908184940_AlreadyApplied(t *testing.T) {
	db := applyUpToPrev(t)
	execNoErr(t, db, `ALTER TABLE host_mdm_apple_profiles ADD INDEX idx_hmap_command_uuid (command_uuid)`)

	applyNext(t, db)

	require.Equal(t, []string{"command_uuid"}, indexColumns(t, db, "host_mdm_apple_profiles", "idx_hmap_command_uuid"))
}
