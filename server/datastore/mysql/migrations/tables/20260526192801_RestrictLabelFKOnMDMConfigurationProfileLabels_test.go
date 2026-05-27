package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260526192801(t *testing.T) {
	db := applyUpToPrev(t)

	// Set up a label, an Apple config profile, and a declaration, each linked to the label
	labelID := execNoErrLastID(t, db, `INSERT INTO labels (name, query, label_type) VALUES ('test-label', 'SELECT 1', 0)`)

	profileUUID := "test-profile-uuid-cfg-restrict"
	execNoErr(t, db, `INSERT INTO mdm_apple_configuration_profiles (profile_uuid, team_id, identifier, name, mobileconfig, checksum, uploaded_at) VALUES (?, 0, 'com.example.test', 'Test Profile', '<plist/>', '', NOW())`, profileUUID)
	execNoErr(t, db, `INSERT INTO mdm_configuration_profile_labels (apple_profile_uuid, label_name, label_id) VALUES (?, 'test-label', ?)`, profileUUID, labelID)

	declUUID := "test-decl-uuid-restrict"
	execNoErr(t, db, `INSERT INTO mdm_apple_declarations (declaration_uuid, team_id, identifier, name, raw_json) VALUES (?, 0, 'com.example.decl', 'Test Declaration', '{}')`, declUUID)
	execNoErr(t, db, `INSERT INTO mdm_declaration_labels (apple_declaration_uuid, label_name, label_id) VALUES (?, 'test-label', ?)`, declUUID, labelID)

	applyNext(t, db)

	// After migration, deleting the label should be blocked by RESTRICT on both tables
	_, err := db.Exec(`DELETE FROM labels WHERE id = ?`, labelID)
	require.Error(t, err, "expected FK restriction to prevent label deletion")
}
