package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260723181404(t *testing.T) {
	db := applyUpToPrev(t)

	// Apply current migration.
	applyNext(t, db)

	// Insert a row into mdm_apple_declaration_assets and check for unique constraints
	_, err := db.ExecContext(t.Context(), "INSERT INTO mdm_apple_declaration_assets (asset_uuid, team_id, identifier, name, raw_json) VALUES ('uuid1', 1, 'identifier1', 'name1', '{}')")
	require.NoError(t, err)

	// Attempt to insert a duplicate row with the same team_id and identifier
	_, err = db.ExecContext(t.Context(), "INSERT INTO mdm_apple_declaration_assets (asset_uuid, team_id, identifier, name, raw_json) VALUES ('uuid2', 1, 'identifier1', 'name2', '{}')")
	require.Error(t, err)

	// Attempt to insert a duplicate row with the same team_id and name
	_, err = db.ExecContext(t.Context(), "INSERT INTO mdm_apple_declaration_assets (asset_uuid, team_id, identifier, name, raw_json) VALUES ('uuid3', 1, 'identifier3', 'name1', '{}')")
	require.Error(t, err)

	// Same identifier and name but different team_id should succeed
	_, err = db.ExecContext(t.Context(), "INSERT INTO mdm_apple_declaration_assets (asset_uuid, team_id, identifier, name, raw_json) VALUES ('uuid4', 2, 'identifier1', 'name1', '{}')")
	require.NoError(t, err)

	// Insert declaration
	_, err = db.ExecContext(t.Context(), "INSERT INTO mdm_apple_declarations (declaration_uuid, identifier, name, team_id, raw_json) VALUES ('decl_uuid1', 'identifier1', 'name1', 1, '{}')")
	require.NoError(t, err)

	// Insert a reference to the asset
	_, err = db.ExecContext(t.Context(), "INSERT INTO mdm_apple_declaration_asset_references (declaration_uuid, asset_uuid) VALUES ('decl_uuid1', 'uuid1')")
	require.NoError(t, err)

	// Verify that mdm_apple_declaration_asset_references table has the correct foreign key constraints
	_, err = db.ExecContext(t.Context(), "INSERT INTO mdm_apple_declaration_asset_references (declaration_uuid, asset_uuid) VALUES ('decl_uuid2', 'uuid1')")
	require.Error(t, err) // Should fail because 'decl_uuid2' does not exist in mdm_apple_declarations
	_, err = db.ExecContext(t.Context(), "INSERT INTO mdm_apple_declaration_asset_references (declaration_uuid, asset_uuid) VALUES ('decl_uuid1', 'uuid-none')")
	require.Error(t, err) // Should fail because 'uuid-none' does not exist in mdm_apple_declaration_assets

	// Verify deleting asset is not allowed
	_, err = db.ExecContext(t.Context(), "DELETE FROM mdm_apple_declaration_assets WHERE asset_uuid = 'uuid1'")
	require.Error(t, err) // Should fail due to foreign key constraint

	// Verify deleting declaration cascades to references
	_, err = db.ExecContext(t.Context(), "DELETE FROM mdm_apple_declarations WHERE declaration_uuid = 'decl_uuid1'")
	require.NoError(t, err)

	// Verify that the reference has been deleted
	var count int
	err = db.QueryRowContext(t.Context(), "SELECT COUNT(*) FROM mdm_apple_declaration_asset_references WHERE declaration_uuid = 'decl_uuid1'").Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 0, count)
}
