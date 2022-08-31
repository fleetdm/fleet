package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20220524102918(t *testing.T) {
	db := applyUpToPrev(t)

	res, err := db.Exec(`
    INSERT INTO teams (name)
    VALUES ('test_team')
  `)
	require.NoError(t, err)
	teamID, err := res.LastInsertId()
	require.NoError(t, err)

	_, err = db.Exec(`
    INSERT INTO policies (name, query, description, team_id)
    VALUES ('test_policy', "", "", ?)
  `, teamID)
	require.NoError(t, err)
	policyID, err := res.LastInsertId()
	require.NoError(t, err)

	res, err = db.Exec(`
    INSERT INTO hosts (osquery_host_id, team_id)
    VALUES (1, ?)
  `, teamID)
	require.NoError(t, err)
	host1ID, err := res.LastInsertId()
	require.NoError(t, err)

	res, err = db.Exec(`
    INSERT INTO hosts (osquery_host_id, team_id)
    VALUES (2, ?)
  `, nil)
	require.NoError(t, err)
	host2ID, err := res.LastInsertId()
	require.NoError(t, err)

	_, err = db.Exec(`
    INSERT INTO policy_membership (host_id, policy_id)
    VALUES (?, ?)
  `, host1ID, policyID)
	require.NoError(t, err)

	_, err = db.Exec(`
    INSERT INTO policy_membership (host_id, policy_id)
    VALUES (?, ?)
  `, host2ID, policyID)
	require.NoError(t, err)

	var count int
	const countStmt = `SELECT COUNT(*) FROM policy_membership`
	err = db.Get(&count, countStmt)
	require.NoError(t, err)
	require.Equal(t, 2, count)

	// Apply current migration.
	applyNext(t, db)

	err = db.Get(&count, countStmt)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	var id int64
	err = db.Get(&id, `SELECT host_id FROM policy_membership`)
	require.NoError(t, err)
	require.Equal(t, id, host1ID)
}
