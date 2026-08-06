package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260729115013(t *testing.T) {
	db := applyUpToPrev(t)

	// The stale activation references table is present before the migration.
	var staleTableCount int
	err := db.Get(&staleTableCount, `
		SELECT COUNT(*) FROM information_schema.tables
		WHERE table_schema = DATABASE() AND table_name = 'mdm_apple_declaration_activation_references'`)
	require.NoError(t, err)
	require.Equal(t, 1, staleTableCount)

	// Seed a declaration and a profile variable row bound to it. Replacing the
	// check constraint revalidates every existing row, so this guards against
	// the migration failing (or dropping data) on a non-empty table.
	const declUUID = "db7e0a0e6-0000-0000-0000-000000000001"
	execNoErr(t, db, `
		INSERT INTO mdm_apple_declarations (declaration_uuid, team_id, identifier, name, raw_json, uploaded_at)
		VALUES (?, 0, 'com.fleet.test.config', 'Test config', '{"Type":"com.apple.configuration.passcode.settings"}', NOW(6))`,
		declUUID)

	var fleetVarID uint
	require.NoError(t, db.Get(&fleetVarID, `SELECT id FROM fleet_variables ORDER BY id LIMIT 1`))
	execNoErr(t, db, `
		INSERT INTO mdm_configuration_profile_variables (apple_declaration_uuid, fleet_variable_id)
		VALUES (?, ?)`, declUUID, fleetVarID)

	applyNext(t, db)

	// Stale table is gone, and the pre-existing variable row survived the
	// check constraint swap.
	err = db.Get(&staleTableCount, `
		SELECT COUNT(*) FROM information_schema.tables
		WHERE table_schema = DATABASE() AND table_name = 'mdm_apple_declaration_activation_references'`)
	require.NoError(t, err)
	require.Equal(t, 0, staleTableCount)

	var existingVars int
	require.NoError(t, db.Get(&existingVars, `
		SELECT COUNT(*) FROM mdm_configuration_profile_variables WHERE apple_declaration_uuid = ?`, declUUID))
	require.Equal(t, 1, existingVars)

	// An activation can be attached to the declaration, and its token is
	// generated from the stored JSON.
	const actUUID = "ab7e0a0e6-0000-0000-0000-000000000001"
	execNoErr(t, db, `
		INSERT INTO mdm_apple_ddm_activations
			(activation_uuid, team_id, identifier, raw_json, declaration_uuid, configuration_identifier, uploaded_at)
		VALUES (?, 0, 'com.fleet.test.config.activation', ?, ?, 'com.fleet.test.config', NOW(6))`,
		actUUID,
		`{"Type":"com.apple.activation.simple","Payload":{"StandardConfigurations":["com.fleet.test.config"]}}`,
		declUUID)

	var token []byte
	require.NoError(t, db.Get(&token, `SELECT token FROM mdm_apple_ddm_activations WHERE activation_uuid = ?`, actUUID))
	require.Len(t, token, 16)

	// Only one activation per declaration.
	_, err = db.Exec(`
		INSERT INTO mdm_apple_ddm_activations
			(activation_uuid, team_id, identifier, raw_json, declaration_uuid, configuration_identifier)
		VALUES ('a-dupe', 0, 'com.fleet.test.other.activation', '{}', ?, 'com.fleet.test.other')`, declUUID)
	require.Error(t, err)

	// An activation must reference a declaration that exists.
	_, err = db.Exec(`
		INSERT INTO mdm_apple_ddm_activations
			(activation_uuid, team_id, identifier, raw_json, declaration_uuid, configuration_identifier)
		VALUES ('a-orphan', 0, 'com.fleet.test.orphan.activation', '{}', 'd-does-not-exist', 'com.fleet.test.orphan')`)
	require.Error(t, err)

	// Fleet variables can be associated with an activation. This is the case
	// the check constraint would reject if it hadn't been replaced.
	execNoErr(t, db, `
		INSERT INTO mdm_configuration_profile_variables (apple_ddm_activation_uuid, fleet_variable_id)
		VALUES (?, ?)`, actUUID, fleetVarID)

	// The constraint still requires exactly one owner: neither two nor zero.
	_, err = db.Exec(`
		INSERT INTO mdm_configuration_profile_variables (apple_ddm_activation_uuid, apple_declaration_uuid, fleet_variable_id)
		VALUES (?, ?, ?)`, actUUID, declUUID, fleetVarID)
	require.Error(t, err)

	_, err = db.Exec(`INSERT INTO mdm_configuration_profile_variables (fleet_variable_id) VALUES (?)`, fleetVarID)
	require.Error(t, err)

	// host_mdm_apple_declarations tracks when the activation last changed.
	execNoErr(t, db, `
		INSERT INTO host_mdm_apple_declarations
			(host_uuid, declaration_uuid, declaration_identifier, declaration_name, token, activation_updated_at)
		VALUES ('host-uuid-1', ?, 'com.fleet.test.config', 'Test config', UNHEX(MD5('t')), NOW(6))`, declUUID)

	var activationUpdatedAt *string
	require.NoError(t, db.Get(&activationUpdatedAt, `
		SELECT activation_updated_at FROM host_mdm_apple_declarations WHERE host_uuid = 'host-uuid-1'`))
	require.NotNil(t, activationUpdatedAt)

	// Deleting the declaration cascades to the activation and, through it, to
	// the activation's variable rows.
	execNoErr(t, db, `DELETE FROM mdm_apple_declarations WHERE declaration_uuid = ?`, declUUID)

	var remainingActivations int
	require.NoError(t, db.Get(&remainingActivations, `SELECT COUNT(*) FROM mdm_apple_ddm_activations`))
	require.Equal(t, 0, remainingActivations)

	var remainingVars int
	require.NoError(t, db.Get(&remainingVars, `
		SELECT COUNT(*) FROM mdm_configuration_profile_variables WHERE apple_ddm_activation_uuid = ?`, actUUID))
	require.Equal(t, 0, remainingVars)
}
