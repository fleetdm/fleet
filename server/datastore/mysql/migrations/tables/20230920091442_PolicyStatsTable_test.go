package tables

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20230920091442(t *testing.T) {
	db := applyUpToPrev(t)

	const (
		insertUsersStmt = `INSERT INTO users (id, name, password, salt, email) VALUES (?, ?, ?, ?, ?)`

		insertPolicyStmt = `INSERT INTO policies (
			team_id, name, query, description, author_id, platforms, critical
		) VALUES (?, ?, ?, ?, ?, ?, ?)`

		insertTeamStmt = `INSERT INTO teams (name) VALUES (?)`

		deletePolicyStmt = `DELETE FROM policies WHERE id = ?`

		deleteTeamStmt = `DELETE FROM teams WHERE id = ?`

		loadPolicyStatsStmt = `SELECT
			id, policy_id, inherited_team_id, passing_host_count, failing_host_count
		FROM policy_stats WHERE id = ?`
	)

	// Apply current migration.
	applyNext(t, db)

	// Create a user
	_, err := db.Exec(insertUsersStmt, 1, "user1", "password999", "salt999", "foo")
	require.NoError(t, err)

	// Create a team
	res, err := db.Exec(insertTeamStmt, "team1")
	require.NoError(t, err)
	teamID, _ := res.LastInsertId()

	// Create a global policy
	res, err = db.Exec(insertPolicyStmt, nil, "global-policy", "SELECT 1;", "Global policy description", 1, "all", false)
	require.NoError(t, err)
	globalPolicyStatID, _ := res.LastInsertId()

	// Insert a policy_stats entry for the global policy (globally)
	_, err = db.Exec(`INSERT INTO policy_stats (policy_id, inherited_team_id, passing_host_count, failing_host_count) VALUES (?, NULL, ?, ?)`, globalPolicyStatID, 100, 10)
	require.NoError(t, err)

	// Insert a policy_stats entry for the team inheriting the global policy
	res, err = db.Exec(`INSERT INTO policy_stats (policy_id, inherited_team_id, passing_host_count, failing_host_count) VALUES (?, ?, ?, ?)`, globalPolicyStatID, teamID, 50, 5)
	require.NoError(t, err)
	inheritedPolicyStatID, _ := res.LastInsertId()

	// Verify the entries in the policy_stats table
	var id int64
	var policyID, inheritedTeamID sql.NullInt64
	var passingCount, failingCount int

	// Verify global policy stats (global level)
	err = db.QueryRow(loadPolicyStatsStmt, 1).Scan(&id, &policyID, &inheritedTeamID, &passingCount, &failingCount)
	require.NoError(t, err)
	require.Equal(t, globalPolicyStatID, policyID.Int64)
	require.False(t, inheritedTeamID.Valid)

	// Verify global policy stats (team level)
	err = db.QueryRow(loadPolicyStatsStmt, 2).Scan(&id, &policyID, &inheritedTeamID, &passingCount, &failingCount)
	require.NoError(t, err)
	require.Equal(t, globalPolicyStatID, policyID.Int64)
	require.True(t, inheritedTeamID.Valid)
	require.Equal(t, teamID, inheritedTeamID.Int64)

	// Delete the team and check that its policy_stats entry is also deleted
	_, err = db.Exec(deleteTeamStmt, teamID)
	require.NoError(t, err)

	err = db.QueryRow(loadPolicyStatsStmt, inheritedPolicyStatID).Scan(&id, &policyID, &inheritedTeamID, &passingCount, &failingCount)
	require.Error(t, err)
	require.ErrorIs(t, err, sql.ErrNoRows)

	// Verify global policy stats still exist (global level)
	err = db.QueryRow(loadPolicyStatsStmt, globalPolicyStatID).Scan(&id, &policyID, &inheritedTeamID, &passingCount, &failingCount)
	require.NoError(t, err)

	// Delete the global policy and check that its policy_stats entry is also deleted
	_, err = db.Exec(deletePolicyStmt, globalPolicyStatID)
	require.NoError(t, err)

	err = db.QueryRow(loadPolicyStatsStmt, globalPolicyStatID).Scan(&id, &policyID, &inheritedTeamID, &passingCount, &failingCount)
	require.Error(t, err)
	require.ErrorIs(t, err, sql.ErrNoRows)
}
