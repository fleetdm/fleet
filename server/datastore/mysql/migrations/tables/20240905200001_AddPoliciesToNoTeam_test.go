package tables

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20240905200001(t *testing.T) {
	db := applyUpToPrev(t)

	team1ID := uint(execNoErrLastID(t, db, `INSERT INTO teams (name) VALUES ('team1');`)) //nolint:gosec // dismiss G115
	globalPolicy0 := uint(execNoErrLastID(t, db,                                          //nolint:gosec // dismiss G115
		`INSERT INTO policies (name, query, description, checksum) VALUES
		('globalPolicy0', 'SELECT 0', 'Description', 'checksum');`,
	))
	policy1Team1 := uint(execNoErrLastID(t, db, //nolint:gosec // dismiss G115
		`INSERT INTO policies (name, query, description, team_id, checksum)
		VALUES ('policy1Team1', 'SELECT 1', 'Description', ?, 'checksum2');`,
		team1ID,
	))

	// Insert policy stats for a global policy.
	execNoErr(t, db,
		`INSERT INTO policy_stats
			(policy_id, inherited_team_id, passing_host_count, failing_host_count)
		VALUES 
			(?, ?, 1, 2), (?, ?, 3, 4);`,
		globalPolicy0,
		0,
		globalPolicy0,
		policy1Team1,
	)
	// Insert policy stats for a team policy.
	execNoErr(t, db,
		`INSERT INTO policy_stats (policy_id, inherited_team_id, passing_host_count, failing_host_count)
		VALUES (?, ?, 5, 6);`,
		policy1Team1,
		0,
	)

	applyNext(t, db)

	// Check the policy_stats for global have been migrated correctly.
	var results []struct {
		PolicyID            uint   `db:"policy_id"`
		InheritedTeamID     *uint  `db:"inherited_team_id"`
		InheritedTeamIDChar string `db:"inherited_team_id_char"`
		PassingHostCount    uint   `db:"passing_host_count"`
		FailingHostCount    uint   `db:"failing_host_count"`
	}
	err := db.Select(&results,
		`SELECT policy_id, inherited_team_id, inherited_team_id_char, passing_host_count, failing_host_count
		FROM policy_stats ORDER BY policy_id ASC;`,
	)
	require.NoError(t, err)
	require.Len(t, results, 3)

	require.Equal(t, globalPolicy0, results[0].PolicyID)
	require.Nil(t, results[0].InheritedTeamID)
	require.Equal(t, "global", results[0].InheritedTeamIDChar)
	require.Equal(t, uint(1), results[0].PassingHostCount)
	require.Equal(t, uint(2), results[0].FailingHostCount)

	require.Equal(t, globalPolicy0, results[1].PolicyID)
	require.NotNil(t, results[1].InheritedTeamID)
	require.Equal(t, policy1Team1, *results[1].InheritedTeamID)
	require.Equal(t, strconv.FormatUint(uint64(policy1Team1), 10), results[1].InheritedTeamIDChar)
	require.Equal(t, uint(3), results[1].PassingHostCount)
	require.Equal(t, uint(4), results[1].FailingHostCount)

	require.Equal(t, policy1Team1, results[2].PolicyID)
	require.Nil(t, results[2].InheritedTeamID)
	require.Equal(t, "global", results[2].InheritedTeamIDChar)
	require.Equal(t, uint(5), results[2].PassingHostCount)
	require.Equal(t, uint(6), results[2].FailingHostCount)

	// The team can be deleted, and the policy won't be automatically deleted.
	execNoErr(t, db,
		`DELETE FROM teams;`,
	)
	var ok bool
	err = db.Get(&ok, `SELECT 1 FROM policies WHERE id = ?;`, policy1Team1)
	require.NoError(t, err)
}
