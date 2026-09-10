package tables

import (
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func seedNanoResultForCleanupIndexTest(t *testing.T, db *sqlx.DB, enrollmentID, cmdUUID, status string) {
	const plist = `<?xml version="1.0"?><plist/>`
	execNoErr(t, db, `INSERT INTO nano_commands (command_uuid, request_type, command, name) VALUES (?, 'DeviceInformation', ?, '')`, cmdUUID, plist)
	execNoErr(t, db, `INSERT INTO nano_enrollment_queue (id, command_uuid) VALUES (?, ?)`, enrollmentID, cmdUUID)
	execNoErr(t, db, `INSERT INTO nano_command_results (id, command_uuid, status, result) VALUES (?, ?, ?, ?)`, enrollmentID, cmdUUID, status, plist)
}

func TestUp_20260908184938(t *testing.T) {
	db := applyUpToPrev(t)

	execNoErr(t, db, `INSERT INTO nano_devices (id, authenticate) VALUES ('device-1', 'auth')`)
	execNoErr(t, db, `
		INSERT INTO nano_enrollments (id, device_id, type, topic, push_magic, token_hex, enabled)
		VALUES ('device-1', 'device-1', 'Device', 'com.apple.mgmt.test', 'magic', 'abcdef', 1)`)
	seedNanoResultForCleanupIndexTest(t, db, "device-1", "REFETCH-DEVICE-1", "Acknowledged")
	seedNanoResultForCleanupIndexTest(t, db, "device-1", "REFETCH-APPS-1", "Error")
	seedNanoResultForCleanupIndexTest(t, db, "device-1", "cmd-not-now", "NotNow")

	applyNext(t, db)

	require.Equal(t, []string{"status", "updated_at"}, indexColumns(t, db, "nano_command_results", "idx_ncr_status_updated_at"))
	require.Empty(t, indexColumns(t, db, "nano_command_results", "status"))

	// Seeded rows survive the ALTER and are readable through the sweep's keyset scan.
	var uuids []string
	require.NoError(t, db.Select(&uuids, `
		SELECT command_uuid FROM nano_command_results
		WHERE status = 'Acknowledged'
		  AND updated_at < NOW() + INTERVAL 1 DAY
		  AND (updated_at, id, command_uuid) > ('1970-01-01', '', '')
		ORDER BY updated_at, id, command_uuid`))
	require.Equal(t, []string{"REFETCH-DEVICE-1"}, uuids)
}

func TestUp_20260908184938_AlreadyApplied(t *testing.T) {
	db := applyUpToPrev(t)
	execNoErr(t, db, "ALTER TABLE nano_command_results ADD INDEX idx_ncr_status_updated_at (status, updated_at), DROP INDEX `status`")

	applyNext(t, db)

	require.Equal(t, []string{"status", "updated_at"}, indexColumns(t, db, "nano_command_results", "idx_ncr_status_updated_at"))
	require.Empty(t, indexColumns(t, db, "nano_command_results", "status"))
}

func TestUp_20260908184938_OldIndexStillPresent(t *testing.T) {
	db := applyUpToPrev(t)
	execNoErr(t, db, `ALTER TABLE nano_command_results ADD INDEX idx_ncr_status_updated_at (status, updated_at)`)

	applyNext(t, db)

	require.Equal(t, []string{"status", "updated_at"}, indexColumns(t, db, "nano_command_results", "idx_ncr_status_updated_at"))
	require.Empty(t, indexColumns(t, db, "nano_command_results", "status"))
}
