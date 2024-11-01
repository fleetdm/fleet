package tables

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20240131083822(t *testing.T) {
	db := applyUpToPrev(t)

	// Create a team
	const insertTeamStmt = `INSERT INTO teams (name) VALUES (?)`
	teamID := execNoErrLastID(t, db, insertTeamStmt, "team1")

	// add some policies entries
	const insertStmt = `INSERT INTO policies
		(team_id, name, query, description)
	VALUES
		(?, ?, ?, ?)`

	policy1 := execNoErrLastID(t, db, "INSERT INTO policies (name, query, description) VALUES (?,?,?)", "policy1", "", "")
	policy2 := execNoErrLastID(t, db, insertStmt, teamID, "policy2", "", "")
	policy3 := execNoErrLastID(t, db, insertStmt, teamID, "policy3", "", "")

	// Apply current migration.
	applyNext(t, db)

	var policyCheck []struct {
		ID       int64  `db:"id"`
		Name     string `db:"name"`
		Checksum string `db:"checksum"`
	}
	err := db.SelectContext(context.Background(), &policyCheck, `SELECT id, name, HEX(checksum) AS checksum FROM policies ORDER BY id`)
	require.NoError(t, err)
	wantIDs := []int64{policy1, policy2, policy3}
	require.Len(t, policyCheck, len(wantIDs))

	gotIDs := make([]int64, len(wantIDs))
	for i, pc := range policyCheck {
		if pc.ID == policy1 { //nolint:gocritic // ignore ifelseChain
			require.Equal(t, pc.Name, "policy1")
		} else if pc.ID == policy2 {
			require.Equal(t, pc.Name, "policy2")
		} else {
			require.Equal(t, pc.Name, "policy3")
		}
		gotIDs[i] = pc.ID
		require.NotEmpty(t, pc.Checksum)
		require.Len(t, pc.Checksum, 32)
	}
	require.Equal(t, wantIDs, gotIDs)

	// Now insert a policy with the same name but different team_id
	const insertStmtWithChecksum = `INSERT INTO policies
		(team_id, name, query, description, checksum)
	VALUES
		(?, ?, ?, ?, ?)`
	_ = execNoErrLastID(t, db, insertStmtWithChecksum, "1", "policy1", "", "", "checksum")

}
