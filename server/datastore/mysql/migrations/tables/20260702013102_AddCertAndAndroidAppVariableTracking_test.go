package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260702013102(t *testing.T) {
	db := applyUpToPrev(t)

	// Apply current migration.
	applyNext(t, db)

	// Ensure a fleet variable exists.
	_, err := db.Exec(`INSERT INTO fleet_variables (name) VALUES ('FLEET_VAR_HOST_UUID') ON DUPLICATE KEY UPDATE id=id`)
	require.NoError(t, err)

	var fleetVarID uint
	err = db.QueryRow(`SELECT id FROM fleet_variables WHERE name = 'FLEET_VAR_HOST_UUID'`).Scan(&fleetVarID)
	require.NoError(t, err)

	// Create a team.
	_, err = db.Exec(`INSERT INTO teams (name) VALUES ('test_team_var_tracking')`)
	require.NoError(t, err)

	var teamID uint
	err = db.QueryRow(`SELECT id FROM teams WHERE name = 'test_team_var_tracking'`).Scan(&teamID)
	require.NoError(t, err)

	// --- Certificate template variable tracking ---

	_, err = db.Exec(`INSERT INTO certificate_authorities (name, type, url) VALUES ('test_ca', 'custom_scep_proxy', 'https://ca.example.com')`)
	require.NoError(t, err)

	var caID uint
	err = db.QueryRow(`SELECT id FROM certificate_authorities WHERE name = 'test_ca'`).Scan(&caID)
	require.NoError(t, err)

	res, err := db.Exec(`INSERT INTO certificate_templates (team_id, certificate_authority_id, name, subject_name) VALUES (?, ?, 'test_cert', 'CN=$FLEET_VAR_HOST_UUID')`, teamID, caID)
	require.NoError(t, err)
	certTemplateIDInt, err := res.LastInsertId()
	require.NoError(t, err)
	certTemplateID := uint(certTemplateIDInt) //nolint:gosec

	// Insert and verify uniqueness.
	_, err = db.Exec(`INSERT INTO mdm_configuration_profile_variables (certificate_template_id, fleet_variable_id) VALUES (?, ?)`, certTemplateID, fleetVarID)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO mdm_configuration_profile_variables (certificate_template_id, fleet_variable_id) VALUES (?, ?)`, certTemplateID, fleetVarID)
	require.Error(t, err)

	// CHECK: cannot set both certificate_template_id and another column.
	winProfUUID := "w-test-var-tracking"
	_, err = db.Exec(`INSERT INTO mdm_windows_configuration_profiles (profile_uuid, team_id, name, syncml) VALUES (?, ?, 'wintest', '<SyncML/>')`, winProfUUID, teamID)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO mdm_configuration_profile_variables (certificate_template_id, windows_profile_uuid, fleet_variable_id) VALUES (?, ?, ?)`, certTemplateID, winProfUUID, fleetVarID)
	require.Error(t, err)

	// Cascade delete.
	_, err = db.Exec(`DELETE FROM certificate_templates WHERE id = ?`, certTemplateID)
	require.NoError(t, err)
	var count int
	err = db.QueryRow(`SELECT COUNT(*) FROM mdm_configuration_profile_variables WHERE certificate_template_id = ?`, certTemplateID).Scan(&count)
	require.NoError(t, err)
	require.Zero(t, count)

	// --- Android app configuration variable tracking ---

	res, err = db.Exec(`INSERT INTO android_app_configurations (application_id, team_id, global_or_team_id, configuration) VALUES ('com.example.app', ?, ?, '{"managedConfiguration":{"key":"$FLEET_VAR_HOST_UUID"}}')`, teamID, teamID)
	require.NoError(t, err)
	appConfigIDInt, err := res.LastInsertId()
	require.NoError(t, err)
	appConfigID := uint(appConfigIDInt) //nolint:gosec

	// Insert and verify uniqueness.
	_, err = db.Exec(`INSERT INTO mdm_configuration_profile_variables (android_app_configuration_id, fleet_variable_id) VALUES (?, ?)`, appConfigID, fleetVarID)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO mdm_configuration_profile_variables (android_app_configuration_id, fleet_variable_id) VALUES (?, ?)`, appConfigID, fleetVarID)
	require.Error(t, err)

	// CHECK: cannot set both android_app_configuration_id and another column.
	_, err = db.Exec(`INSERT INTO mdm_configuration_profile_variables (android_app_configuration_id, windows_profile_uuid, fleet_variable_id) VALUES (?, ?, ?)`, appConfigID, winProfUUID, fleetVarID)
	require.Error(t, err)

	// Cascade delete.
	_, err = db.Exec(`DELETE FROM android_app_configurations WHERE id = ?`, appConfigID)
	require.NoError(t, err)
	err = db.QueryRow(`SELECT COUNT(*) FROM mdm_configuration_profile_variables WHERE android_app_configuration_id = ?`, appConfigID).Scan(&count)
	require.NoError(t, err)
	require.Zero(t, count)
}
