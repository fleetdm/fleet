package tables

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUp_20260814183816(t *testing.T) {
	db := applyUpToPrev(t)

	cmdUUID := uuid.New().String()

	// Insert a command row before migration (no raw_command / raw_result columns yet)
	_, err := db.Exec(`
		INSERT INTO mdm_android_commands (command_uuid, host_uuid, operation_name, command_type)
		VALUES (?, ?, ?, ?)`,
		cmdUUID, "host-uuid-1", "enterprises/LC00/devices/d1/operations/op1", "lock")
	require.NoError(t, err)

	// Apply migration
	applyNext(t, db)

	// Verify existing row has NULL for new columns
	var rawCmd, rawResult *string
	err = db.QueryRow(`SELECT raw_command, raw_result FROM mdm_android_commands WHERE command_uuid = ?`, cmdUUID).Scan(&rawCmd, &rawResult)
	require.NoError(t, err)
	assert.Nil(t, rawCmd)
	assert.Nil(t, rawResult)

	// Verify we can insert a new row with the new columns populated
	cmdUUID2 := uuid.New().String()
	rawJSON := `{"type":"REBOOT"}`
	resultJSON := `{"name":"enterprises/LC00/devices/d1/operations/op2","done":true}`

	_, err = db.Exec(`
		INSERT INTO mdm_android_commands (command_uuid, host_uuid, operation_name, command_type, raw_command, raw_result)
		VALUES (?, ?, ?, ?, ?, ?)`,
		cmdUUID2, "host-uuid-2", "enterprises/LC00/devices/d2/operations/op2", "reboot", rawJSON, resultJSON)
	require.NoError(t, err)

	var storedCmd, storedResult string
	err = db.QueryRow(`SELECT raw_command, raw_result FROM mdm_android_commands WHERE command_uuid = ?`, cmdUUID2).Scan(&storedCmd, &storedResult)
	require.NoError(t, err)
	assert.JSONEq(t, rawJSON, storedCmd)
	assert.JSONEq(t, resultJSON, storedResult)
}
