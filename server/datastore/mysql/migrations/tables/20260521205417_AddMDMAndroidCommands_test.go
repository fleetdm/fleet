package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260521205417(t *testing.T) {
	db := applyUpToPrev(t)

	applyNext(t, db)

	// Insert a representative row to confirm the schema accepts every column we plan to write
	// at runtime (Lock command, pending status, full AMAPI operation name, JSON payload).
	execNoErr(t, db, `
		INSERT INTO mdm_android_commands
			(command_uuid, host_uuid, operation_name, command_type, status, request_payload)
		VALUES
			(?, ?, ?, 'LOCK', 'pending', JSON_OBJECT('type', 'LOCK', 'duration', '315360000s'))`,
		"00000000-0000-0000-0000-000000000001",
		"NWXZ-4L5T-V6UN-SCHUL-JOEA-RAVB-Z",
		"enterprises/LC01aeejlw/devices/33d68ef3111852c0/operations/1779311936147")

	// Insert a second row with an error so we exercise the nullable error_code / error_message
	// columns (which Pub/Sub will populate when AMAPI rejects a command).
	execNoErr(t, db, `
		INSERT INTO mdm_android_commands
			(command_uuid, host_uuid, operation_name, command_type, status, error_code, error_message)
		VALUES
			(?, ?, ?, 'WIPE', 'error', 'UNSUPPORTED', 'device does not support WIPE')`,
		"00000000-0000-0000-0000-000000000002",
		"1d5c65c684b5fd5c47a337db8fd54afabc922f509daf7508ceb8e16e088f9c75",
		"enterprises/LC01aeejlw/devices/39138c6262b1b063/operations/1779392334766")

	// Confirm both rows are readable.
	var rows []struct {
		CommandUUID string  `db:"command_uuid"`
		HostUUID    string  `db:"host_uuid"`
		CommandType string  `db:"command_type"`
		Status      string  `db:"status"`
		ErrorCode   *string `db:"error_code"`
	}
	require.NoError(t, db.Select(&rows,
		`SELECT command_uuid, host_uuid, command_type, status, error_code
		   FROM mdm_android_commands ORDER BY command_uuid`))
	require.Len(t, rows, 2)
	require.Equal(t, "LOCK", rows[0].CommandType)
	require.Equal(t, "pending", rows[0].Status)
	require.Nil(t, rows[0].ErrorCode)
	require.Equal(t, "WIPE", rows[1].CommandType)
	require.Equal(t, "error", rows[1].Status)
	require.NotNil(t, rows[1].ErrorCode)
	require.Equal(t, "UNSUPPORTED", *rows[1].ErrorCode)

	// Confirm indexes exist (so production lookups by host_uuid + operation_name are fast) and
	// that operation_name is unique (so Pub/Sub COMMAND lookups can't return ambiguous rows).
	var indexes []struct {
		IndexName string `db:"INDEX_NAME"`
		NonUnique int    `db:"NON_UNIQUE"`
	}
	require.NoError(t, db.Select(&indexes, `
		SELECT INDEX_NAME, MIN(NON_UNIQUE) AS NON_UNIQUE FROM information_schema.STATISTICS
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'mdm_android_commands'
		GROUP BY INDEX_NAME ORDER BY INDEX_NAME`))
	indexUnique := make(map[string]bool, len(indexes))
	for _, idx := range indexes {
		indexUnique[idx.IndexName] = idx.NonUnique == 0
	}
	require.Contains(t, indexUnique, "idx_mdm_android_commands_host_uuid")
	require.Contains(t, indexUnique, "idx_mdm_android_commands_operation_name")
	require.True(t, indexUnique["idx_mdm_android_commands_operation_name"], "operation_name index must be UNIQUE")

	// Attempting to insert a duplicate operation_name must fail.
	_, err := db.Exec(`
		INSERT INTO mdm_android_commands
			(command_uuid, host_uuid, operation_name, command_type, status)
		VALUES
			(?, ?, ?, 'LOCK', 'pending')`,
		"00000000-0000-0000-0000-000000000003",
		"NWXZ-4L5T-V6UN-SCHUL-JOEA-RAVB-Z",
		"enterprises/LC01aeejlw/devices/33d68ef3111852c0/operations/1779311936147")
	require.Error(t, err)
	require.Contains(t, err.Error(), "Duplicate entry")
}
