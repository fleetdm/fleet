package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260603101320(t *testing.T) {
	db := applyUpToPrev(t)

	applyNext(t, db)

	t.Run("mdm_configuration_profile_labels blocks label deletion", func(t *testing.T) {
		labelID := execNoErrLastID(t, db, `INSERT INTO labels (name, query, label_type) VALUES ('test-label-cfg', 'SELECT 1', 0)`)
		profileUUID := "test-profile-uuid-cfg-restrict"
		execNoErr(t, db, `INSERT INTO mdm_apple_configuration_profiles (profile_uuid, team_id, identifier, name, mobileconfig, checksum, uploaded_at) VALUES (?, 0, 'com.example.test', 'Test Profile', '<plist/>', '', NOW())`, profileUUID)
		execNoErr(t, db, `INSERT INTO mdm_configuration_profile_labels (apple_profile_uuid, label_name, label_id) VALUES (?, 'test-label-cfg', ?)`, profileUUID, labelID)

		_, err := db.Exec(`DELETE FROM labels WHERE id = ?`, labelID)
		require.Error(t, err, "expected FK restriction on mdm_configuration_profile_labels to prevent label deletion")
	})

	t.Run("mdm_declaration_labels blocks label deletion", func(t *testing.T) {
		labelID := execNoErrLastID(t, db, `INSERT INTO labels (name, query, label_type) VALUES ('test-label-decl', 'SELECT 1', 0)`)
		declUUID := "test-decl-uuid-restrict"
		execNoErr(t, db, `INSERT INTO mdm_apple_declarations (declaration_uuid, team_id, identifier, name, raw_json) VALUES (?, 0, 'com.example.decl', 'Test Declaration', '{}')`, declUUID)
		execNoErr(t, db, `INSERT INTO mdm_declaration_labels (apple_declaration_uuid, label_name, label_id) VALUES (?, 'test-label-decl', ?)`, declUUID, labelID)

		_, err := db.Exec(`DELETE FROM labels WHERE id = ?`, labelID)
		require.Error(t, err, "expected FK restriction on mdm_declaration_labels to prevent label deletion")
	})
}
