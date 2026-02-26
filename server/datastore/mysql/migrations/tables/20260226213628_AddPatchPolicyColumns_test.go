package tables

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260226213628(t *testing.T) {
	db := applyUpToPrev(t)

	teamID := execNoErrLastID(t, db, `INSERT INTO teams (name) VALUES (?)`, "Test Team")

	execNoErr(t, db, `INSERT INTO policies (team_id, name, query, description, checksum) 
		VALUES 
			(?, 'policy-1', 'SELECT 1;', '', 'checksum1'),
			(?, 'policy-2', 'SELECT 1;', '', 'checksum2'),
			(?, 'policy-3', 'SELECT 1;', '', 'checksum3'),
			(?, 'policy-4', 'SELECT 1;', '', 'checksum4')
	`, teamID, teamID, teamID, teamID)

	// Apply current migration.
	applyNext(t, db)

	var policyCheck []struct {
		ID      int64  `db:"id"`
		Name    string `db:"name"`
		Type    string `db:"type"`
		TitleID *uint  `db:"patch_software_title_id"`
	}
	err := db.SelectContext(context.Background(), &policyCheck, `SELECT id, name, type, patch_software_title_id FROM policies`)
	require.NoError(t, err)
	require.Len(t, policyCheck, 4)
	// check that type was set to 'dynamic' on all previous policies
	for _, policy := range policyCheck {
		require.Equal(t, "dynamic", policy.Type)
		require.Nil(t, policy.TitleID)
	}

	// check software title foreign key
	title1 := execNoErrLastID(t, db, "INSERT INTO software_titles (name, source, extension_for) VALUES (?, ?, ?)", "sw1", "src1", "")
	policy5 := execNoErrLastID(t, db, `INSERT INTO policies (team_id, name, query, description, checksum, type, patch_software_title_id) 
		VALUES (?, 'policy-5', 'SELECT 1;', '', 'checksum5', 'patch', ?)`, teamID, title1)

	// check uniqueness index
	_, err = db.Exec(`INSERT INTO policies (team_id, name, query, description, checksum, type, patch_software_title_id) 
		VALUES (?, 'policy-6', 'SELECT 1;', '', 'checksum6', 'patch', ?)`, teamID, title1)
	require.ErrorContains(t, err, "Duplicate entry")

	// check on delete cascade
	execNoErr(t, db, `DELETE FROM software_titles WHERE id = ?`, title1)
	var found int
	err = db.GetContext(context.Background(), &found, `SELECT 1 FROM policies WHERE id = ?`, policy5)
	require.ErrorContains(t, err, "sql: no rows in result set")

}
