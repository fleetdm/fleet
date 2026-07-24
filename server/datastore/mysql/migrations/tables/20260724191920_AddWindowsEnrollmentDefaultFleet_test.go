package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260724191920(t *testing.T) {
	db := applyUpToPrev(t)

	// Insert a team and an unlinked Windows enrollment to verify the migration is safe with existing data.
	teamID := execNoErrLastID(t, db, `INSERT INTO teams (name) VALUES ('Workstations')`)
	execNoErr(t, db, `
		INSERT INTO mdm_windows_enrollments
			(mdm_device_id, mdm_hardware_id, device_state, device_type, device_name,
			 enroll_type, enroll_user_id, enroll_proto_version, enroll_client_version, not_in_oobe)
		VALUES ('device-1', 'hw-1', 'enrolled', 'CIMClient_Windows', 'DESKTOP-TEST',
			 'Full', 'user@example.com', '5.0', '10.0.19045', 0)`)

	// Apply current migration.
	applyNext(t, db)

	// New enrollment column exists, defaults to NULL, and is writable.
	var serial *string
	require.NoError(t, db.Get(&serial, `SELECT hardware_serial FROM mdm_windows_enrollments WHERE mdm_device_id = 'device-1'`))
	require.Nil(t, serial)
	execNoErr(t, db, `UPDATE mdm_windows_enrollments SET hardware_serial = 'SER123' WHERE mdm_device_id = 'device-1'`)

	// Config table accepts the single row and nulls team_id when the team is deleted.
	execNoErr(t, db, `INSERT INTO windows_enrollment_config (id, team_id) VALUES (1, ?)`, teamID)
	execNoErr(t, db, `DELETE FROM teams WHERE id = ?`, teamID)
	var gotTeamID *uint
	require.NoError(t, db.Get(&gotTeamID, `SELECT team_id FROM windows_enrollment_config WHERE id = 1`))
	require.Nil(t, gotTeamID)
}
