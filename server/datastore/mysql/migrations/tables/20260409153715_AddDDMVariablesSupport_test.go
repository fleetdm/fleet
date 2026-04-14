package tables

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUp_20260409153715(t *testing.T) {
	db := applyUpToPrev(t)

	// Create prerequisite data: an Apple config profile, a Windows config
	// profile, an Apple declaration, and a fleet variable.
	varID := execNoErrLastID(t, db, `INSERT INTO fleet_variables (name) VALUES ('test_var')`)

	appleProfileUUID := "apple-profile-uuid-001"
	execNoErr(t, db, `
		INSERT INTO mdm_apple_configuration_profiles (profile_uuid, team_id, identifier, name, mobileconfig, checksum)
		VALUES (?, 0, 'com.test.profile', 'Test Profile', '<plist></plist>', '')`,
		appleProfileUUID,
	)

	windowsProfileUUID := "windows-profile-uuid-001"
	execNoErr(t, db, `
		INSERT INTO mdm_windows_configuration_profiles (profile_uuid, team_id, name, syncml)
		VALUES (?, 0, 'Test Windows Profile', '<SyncML></SyncML>')`,
		windowsProfileUUID,
	)

	declUUID := "decl-uuid-001"
	execNoErr(t, db, `
		INSERT INTO mdm_apple_declarations (declaration_uuid, team_id, identifier, name, raw_json)
		VALUES (?, 0, 'com.test.decl', 'Test Declaration', '{}')`,
		declUUID,
	)

	// Insert existing rows into mdm_configuration_profile_variables (pre-migration).
	execNoErr(t, db, `
		INSERT INTO mdm_configuration_profile_variables (apple_profile_uuid, fleet_variable_id) VALUES (?, ?)`,
		appleProfileUUID, varID,
	)
	execNoErr(t, db, `
		INSERT INTO mdm_configuration_profile_variables (windows_profile_uuid, fleet_variable_id) VALUES (?, ?)`,
		windowsProfileUUID, varID,
	)

	// Insert a host declaration row to verify variables_updated_at column is added.
	hostUUID := "host-uuid-001"
	execNoErr(t, db, `INSERT INTO mdm_delivery_status (status) VALUES ('pending') ON DUPLICATE KEY UPDATE status=status`)
	execNoErr(t, db, `INSERT INTO mdm_operation_types (operation_type) VALUES ('install') ON DUPLICATE KEY UPDATE operation_type=operation_type`)
	execNoErr(t, db, `
		INSERT INTO host_mdm_apple_declarations (host_uuid, declaration_uuid, declaration_identifier, token, status, operation_type)
		VALUES (?, ?, 'com.test.decl', UNHEX(MD5('token')), 'pending', 'install')`,
		hostUUID, declUUID,
	)

	// Apply current migration.
	applyNext(t, db)

	// Verify existing rows survived the migration.
	var count int
	err := db.Get(&count, `SELECT COUNT(*) FROM mdm_configuration_profile_variables`)
	require.NoError(t, err)
	require.Equal(t, 2, count)

	// Verify variables_updated_at column exists and defaults to NULL.
	var variablesUpdatedAt *time.Time
	err = db.Get(&variablesUpdatedAt, `
		SELECT variables_updated_at FROM host_mdm_apple_declarations WHERE host_uuid = ?`, hostUUID)
	require.NoError(t, err)
	require.Nil(t, variablesUpdatedAt)

	// Verify we can insert a row with apple_declaration_uuid.
	execNoErr(t, db, `
		INSERT INTO mdm_configuration_profile_variables (apple_declaration_uuid, fleet_variable_id) VALUES (?, ?)`,
		declUUID, varID,
	)

	// Verify check constraint: inserting with no UUID fails.
	_, err = db.Exec(`INSERT INTO mdm_configuration_profile_variables (fleet_variable_id) VALUES (?)`, varID)
	require.Error(t, err, "expected check constraint violation when no UUID is set")

	// Verify check constraint: inserting with two UUIDs fails.
	_, err = db.Exec(`
		INSERT INTO mdm_configuration_profile_variables (apple_profile_uuid, apple_declaration_uuid, fleet_variable_id) VALUES (?, ?, ?)`,
		appleProfileUUID, declUUID, varID,
	)
	require.Error(t, err, "expected check constraint violation when multiple UUIDs are set")
}
