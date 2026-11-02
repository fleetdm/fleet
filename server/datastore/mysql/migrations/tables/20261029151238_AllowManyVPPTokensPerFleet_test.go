package tables

import (
	"testing"

	"github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
)

func TestUp_20261029151238(t *testing.T) {
	db := applyUpToPrev(t)

	token1 := execNoErrLastID(t, db, `INSERT INTO vpp_tokens (organization_name, location, renew_at, token) VALUES ('org1', 'loc1', NOW(), 'tok1')`)
	token2 := execNoErrLastID(t, db, `INSERT INTO vpp_tokens (organization_name, location, renew_at, token) VALUES ('org2', 'loc2', NOW(), 'tok2')`)
	teamID := execNoErrLastID(t, db, `INSERT INTO teams (name) VALUES ('team1')`)
	execNoErr(t, db, `INSERT INTO vpp_token_teams (vpp_token_id, team_id, null_team_type) VALUES (?, ?, 'none')`, token1, teamID)

	applyNext(t, db)

	require.Equal(t, []string{"team_id", "vpp_token_id"}, indexColumns(t, db, "vpp_token_teams", "idx_vpp_token_teams_team_id_vpp_token_id"))
	require.Empty(t, indexColumns(t, db, "vpp_token_teams", "idx_vpp_token_teams_team_id"))

	var count int
	require.NoError(t, db.Get(&count, `SELECT COUNT(*) FROM vpp_token_teams WHERE vpp_token_id = ? AND team_id = ?`, token1, teamID))
	require.Equal(t, 1, count)

	execNoErr(t, db, `INSERT INTO vpp_token_teams (vpp_token_id, team_id, null_team_type) VALUES (?, ?, 'none')`, token2, teamID)

	_, err := db.Exec(`INSERT INTO vpp_token_teams (vpp_token_id, team_id, null_team_type) VALUES (?, ?, 'none')`, token1, teamID)
	var mysqlErr *mysql.MySQLError
	require.ErrorAs(t, err, &mysqlErr)
	require.NotNil(t, mysqlErr)
	require.EqualValues(t, 1062, mysqlErr.Number)

	// The team_id foreign key still cascades.
	execNoErr(t, db, `DELETE FROM teams WHERE id = ?`, teamID)
	require.NoError(t, db.Get(&count, `SELECT COUNT(*) FROM vpp_token_teams`))
	require.Zero(t, count)
}

func TestUp_20261029151238_AlreadyApplied(t *testing.T) {
	db := applyUpToPrev(t)
	execNoErr(t, db, `
		ALTER TABLE vpp_token_teams
		DROP INDEX idx_vpp_token_teams_team_id,
		ADD UNIQUE KEY idx_vpp_token_teams_team_id_vpp_token_id (team_id, vpp_token_id)`)

	applyNext(t, db)

	require.Equal(t, []string{"team_id", "vpp_token_id"}, indexColumns(t, db, "vpp_token_teams", "idx_vpp_token_teams_team_id_vpp_token_id"))
	require.Empty(t, indexColumns(t, db, "vpp_token_teams", "idx_vpp_token_teams_team_id"))
}
