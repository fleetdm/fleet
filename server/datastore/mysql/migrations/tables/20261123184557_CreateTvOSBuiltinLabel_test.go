package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20261123184557(t *testing.T) {
	db := applyUpToPrev(t)

	applyNext(t, db)

	var label struct {
		Description         string `db:"description"`
		Platform            string `db:"platform"`
		LabelType           uint   `db:"label_type"`
		LabelMembershipType uint   `db:"label_membership_type"`
	}
	require.NoError(t, db.Get(&label, `
		SELECT description, platform, label_type, label_membership_type
		FROM labels WHERE name = 'tvOS'`))

	require.Equal(t, "All tvOS hosts", label.Description)
	// Builtin labels must carry no platform; see Up_20251015103800.
	require.Empty(t, label.Platform)
	require.EqualValues(t, 1, label.LabelType)           // builtin
	require.EqualValues(t, 1, label.LabelMembershipType) // manual
}

func TestUp_20261123184557_RenamesConflictingLabel(t *testing.T) {
	db := applyUpToPrev(t)

	insertLabel := func(name string, teamID *int64) int64 {
		return execNoErrLastID(t, db, `
			INSERT INTO labels (name, description, query, platform, label_type, label_membership_type, team_id)
			VALUES (?, '', 'SELECT 1', '', 0, 0, ?)`, name, teamID)
	}
	teamID := execNoErrLastID(t, db, `INSERT INTO teams (name) VALUES ('Conference rooms')`)

	// "TVOS" matches the built-in name under the column's collation.
	conflictingID := insertLabel("TVOS", nil)
	insertLabel("tvOS 2", nil)
	fleetLabelID := insertLabel("tvOS", &teamID)

	execNoErr(t, db, `
		INSERT INTO mdm_apple_configuration_profiles (profile_uuid, team_id, identifier, name, mobileconfig, checksum)
		VALUES ('a1', 0, 'com.example.tv', 'tv', '<plist></plist>', '')`)
	execNoErr(t, db, `
		INSERT INTO mdm_configuration_profile_labels (apple_profile_uuid, label_name, label_id)
		VALUES ('a1', 'TVOS', ?)`, conflictingID)
	// The same profile still holds the name of a deleted label.
	execNoErr(t, db, `
		INSERT INTO mdm_configuration_profile_labels (apple_profile_uuid, label_name, label_id)
		VALUES ('a1', 'tvOS 3', NULL)`)

	applyNext(t, db)

	var name string
	require.NoError(t, db.Get(&name, `SELECT name FROM labels WHERE id = ?`, conflictingID))
	require.Equal(t, "tvOS 4", name)
	require.NoError(t, db.Get(&name, `SELECT label_name FROM mdm_configuration_profile_labels WHERE label_id = ?`, conflictingID))
	require.Equal(t, "tvOS 4", name)

	// A fleet's label can't share a name with an all-fleets label, so it's renamed too.
	require.NoError(t, db.Get(&name, `SELECT name FROM labels WHERE id = ?`, fleetLabelID))
	require.Equal(t, "tvOS 5", name)

	var labelTypes []uint
	require.NoError(t, db.Select(&labelTypes, `SELECT label_type FROM labels WHERE name = 'tvOS'`))
	require.Equal(t, []uint{1}, labelTypes) // only the built-in label keeps the name
}
