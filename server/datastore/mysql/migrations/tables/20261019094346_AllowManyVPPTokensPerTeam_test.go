package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20261019094346(t *testing.T) {
	db := applyUpToPrev(t)

	teamID := execNoErrLastID(t, db, `INSERT INTO teams (name) VALUES ('Mobile')`)
	insertToken := func(location string) int64 {
		return execNoErrLastID(t, db, `INSERT INTO vpp_tokens (organization_name, location, renew_at, token) VALUES ('Acme', ?, NOW(), 'x')`, location)
	}
	usToken, jpToken := insertToken("Acme US"), insertToken("Acme Japan")
	execNoErr(t, db, `INSERT INTO vpp_token_teams (vpp_token_id, team_id, null_team_type) VALUES (?, ?, 'none')`, usToken, teamID)

	_, err := db.Exec(`INSERT INTO vpp_token_teams (vpp_token_id, team_id, null_team_type) VALUES (?, ?, 'none')`, jpToken, teamID)
	require.Error(t, err)

	applyNext(t, db)

	var count int
	require.NoError(t, db.Get(&count, `SELECT COUNT(*) FROM vpp_token_teams WHERE team_id = ? AND vpp_token_id = ?`, teamID, usToken))
	require.Equal(t, 1, count)

	execNoErr(t, db, `INSERT INTO vpp_token_teams (vpp_token_id, team_id, null_team_type) VALUES (?, ?, 'none')`, jpToken, teamID)

	_, err = db.Exec(`INSERT INTO vpp_token_teams (vpp_token_id, team_id, null_team_type) VALUES (?, ?, 'none')`, jpToken, teamID)
	require.Error(t, err)
}
