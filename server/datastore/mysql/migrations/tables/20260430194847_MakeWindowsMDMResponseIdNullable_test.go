package tables

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUp_20260430194847(t *testing.T) {
	db := applyUpToPrev(t)

	// Set up FK-chain: enrollment -> response -> command_result.
	hostUUID := uuid.NewString()
	_, err := db.Exec(`INSERT INTO mdm_windows_enrollments
		(mdm_device_id, mdm_hardware_id, device_state, device_type, device_name,
		 enroll_type, enroll_user_id, enroll_proto_version, enroll_client_version, not_in_oobe, host_uuid)
		VALUES (?, 'hw1', 'MDMDeviceEnrolledEnrolled', 'CIMClient_Windows', 'test',
		        'Full', 'user1', '7.0', '10', 0, ?)`,
		"device-1", hostUUID)
	require.NoError(t, err)

	var enrollmentID int64
	err = db.Get(&enrollmentID, `SELECT id FROM mdm_windows_enrollments WHERE mdm_device_id = ?`, "device-1")
	require.NoError(t, err)

	cmdUUID := uuid.NewString()
	_, err = db.Exec(`INSERT INTO windows_mdm_commands (command_uuid, raw_command, target_loc_uri) VALUES (?, '<Exec/>', './test')`, cmdUUID)
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO windows_mdm_responses (enrollment_id, raw_response) VALUES (?, '<SyncML/>')`, enrollmentID)
	require.NoError(t, err)

	var responseID int64
	err = db.Get(&responseID, `SELECT id FROM windows_mdm_responses WHERE enrollment_id = ?`, enrollmentID)
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO windows_mdm_command_results
		(enrollment_id, command_uuid, raw_result, response_id, status_code)
		VALUES (?, ?, '', ?, '200')`, enrollmentID, cmdUUID, responseID)
	require.NoError(t, err)

	// Before migration: NULL response_id should fail.
	cmdUUID2 := uuid.NewString()
	_, err = db.Exec(`INSERT INTO windows_mdm_commands (command_uuid, raw_command, target_loc_uri) VALUES (?, '<Get/>', './test2')`, cmdUUID2)
	require.NoError(t, err)

	_, err = db.Exec(`INSERT INTO windows_mdm_command_results
		(enrollment_id, command_uuid, raw_result, response_id, status_code)
		VALUES (?, ?, '', NULL, '200')`, enrollmentID, cmdUUID2)
	require.Error(t, err, "NULL response_id should fail before migration")

	// Apply migration.
	applyNext(t, db)

	// Existing row should still have its response_id.
	var gotResponseID *int64
	err = db.Get(&gotResponseID, `SELECT response_id FROM windows_mdm_command_results WHERE command_uuid = ?`, cmdUUID)
	require.NoError(t, err)
	require.NotNil(t, gotResponseID)
	assert.Equal(t, responseID, *gotResponseID)

	// After migration: NULL response_id should succeed.
	_, err = db.Exec(`INSERT INTO windows_mdm_command_results
		(enrollment_id, command_uuid, raw_result, response_id, status_code)
		VALUES (?, ?, '', NULL, '200')`, enrollmentID, cmdUUID2)
	require.NoError(t, err, "NULL response_id should succeed after migration")

	var nullResponseID *int64
	err = db.Get(&nullResponseID, `SELECT response_id FROM windows_mdm_command_results WHERE command_uuid = ?`, cmdUUID2)
	require.NoError(t, err)
	assert.Nil(t, nullResponseID)
}
