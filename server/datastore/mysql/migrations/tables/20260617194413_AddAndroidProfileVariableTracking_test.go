package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260617194413(t *testing.T) {
	db := applyUpToPrev(t)

	// Apply current migration.
	applyNext(t, db)

	// Verify the android_profile_uuid column exists and the unique key works.
	// First insert a fleet variable and an android profile to reference.
	_, err := db.Exec(`INSERT INTO fleet_variables (name) VALUES ('HOST_UUID') ON DUPLICATE KEY UPDATE id=id`)
	require.NoError(t, err)

	var fleetVarID uint
	err = db.QueryRow(`SELECT id FROM fleet_variables WHERE name = 'HOST_UUID'`).Scan(&fleetVarID)
	require.NoError(t, err)

	// Create a team and an android profile for the FK.
	_, err = db.Exec(`INSERT INTO teams (name) VALUES ('test_team_android_var')`)
	require.NoError(t, err)

	var teamID uint
	err = db.QueryRow(`SELECT id FROM teams WHERE name = 'test_team_android_var'`).Scan(&teamID)
	require.NoError(t, err)

	profUUID := "g-test-profile-uuid"
	_, err = db.Exec(`INSERT INTO mdm_android_configuration_profiles (profile_uuid, team_id, name, raw_json) VALUES (?, ?, 'test', '{}')`, profUUID, teamID)
	require.NoError(t, err)

	// Insert a variable association for the android profile.
	_, err = db.Exec(`INSERT INTO mdm_configuration_profile_variables (android_profile_uuid, fleet_variable_id) VALUES (?, ?)`, profUUID, fleetVarID)
	require.NoError(t, err)

	// Verify the unique key prevents duplicates.
	_, err = db.Exec(`INSERT INTO mdm_configuration_profile_variables (android_profile_uuid, fleet_variable_id) VALUES (?, ?)`, profUUID, fleetVarID)
	require.Error(t, err)

	// Use a real Windows profile so the FK doesn't reject before the CHECK evaluates.
	winProfUUID := "w-test-profile-uuid"
	_, err = db.Exec(`INSERT INTO mdm_windows_configuration_profiles (profile_uuid, team_id, name, syncml) VALUES (?, ?, 'wintest', '<SyncML/>')`, winProfUUID, teamID)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO mdm_configuration_profile_variables (android_profile_uuid, windows_profile_uuid, fleet_variable_id) VALUES (?, ?, ?)`, profUUID, winProfUUID, fleetVarID)
	require.Error(t, err)

	// Verify cascade delete works.
	_, err = db.Exec(`DELETE FROM mdm_android_configuration_profiles WHERE profile_uuid = ?`, profUUID)
	require.NoError(t, err)

	var count int
	err = db.QueryRow(`SELECT COUNT(*) FROM mdm_configuration_profile_variables WHERE android_profile_uuid = ?`, profUUID).Scan(&count)
	require.NoError(t, err)
	require.Zero(t, count)
}
