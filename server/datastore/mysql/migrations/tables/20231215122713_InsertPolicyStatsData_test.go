package tables

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/require"
)

func TestUp_20231215122713(t *testing.T) {
	db := applyUpToPrev(t)

	// assert no data in table
	assertRowCount(t, db, "policy_stats", 0)

	// insert 2 teams
	_, err := db.Exec(`INSERT INTO teams (name) VALUES ('team1'), ('team2')`)
	require.NoError(t, err)

	// insert 3 hosts
	h1 := insertHost(t, db, nil)
	h2 := insertHost(t, db, ptr.Uint(1))
	h3 := insertHost(t, db, ptr.Uint(1))

	// insert 1 global policy
	_, err = db.Exec(`INSERT INTO policies (name, description, query, platforms, critical) VALUES ('policy1', 'policy1', 'select 1', 'mac', 1)`)
	require.NoError(t, err)

	// insert 1 team policy
	_, err = db.Exec(`INSERT INTO policies (name, description, query, platforms, critical, team_id) VALUES ('policy2', 'policy2', 'select 1', 'mac', 1, 1)`)
	require.NoError(t, err)

	// insert policy_membership rows
	_, err = db.Exec(`INSERT INTO policy_membership (policy_id, host_id, passes) VALUES (1, ?, 1)`, h1)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO policy_membership (policy_id, host_id, passes) VALUES (1, ?, 1), (2, ?, 0)`, h2, h2)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO policy_membership (policy_id, host_id, passes) VALUES (1, ?, 0), (2, ?, 1)`, h3, h3)
	require.NoError(t, err)

	// Apply current migration.
	applyNext(t, db)

	// 1 global policy stat
	// 2 inherited team stats
	// 1 team1 policy stat
	// 0 team2 policy stats
	assertRowCount(t, db, "policy_stats", 4)

	// assert global policy stat
	type PolicyStat struct {
		PolicyID         int64 `db:"policy_id"`
		InheritedTeamID  int64 `db:"inherited_team_id"`
		PassingHostCount int64 `db:"passing_host_count"`
		FailingHostCount int64 `db:"failing_host_count"`
	}

	var policyStat PolicyStat

	// assert global policy stat
	err = db.Get(&policyStat, `SELECT policy_id, inherited_team_id, passing_host_count, failing_host_count FROM policy_stats WHERE policy_id = 1 AND inherited_team_id = 0`)
	require.NoError(t, err)
	require.Equal(t, PolicyStat{
		PolicyID:         1,
		InheritedTeamID:  0,
		PassingHostCount: 2,
		FailingHostCount: 1,
	}, policyStat)

	// assert inherited team1 stats
	err = db.Get(&policyStat, `SELECT policy_id, inherited_team_id, passing_host_count, failing_host_count FROM policy_stats WHERE policy_id = 1 AND inherited_team_id = 1`)
	require.NoError(t, err)
	require.Equal(t, PolicyStat{
		PolicyID:         1,
		InheritedTeamID:  1,
		PassingHostCount: 1,
		FailingHostCount: 1,
	}, policyStat)

	// assert inherited team2 stats
	err = db.Get(&policyStat, `SELECT policy_id, inherited_team_id, passing_host_count, failing_host_count FROM policy_stats WHERE policy_id = 1 AND inherited_team_id = 2`)
	require.NoError(t, err)
	require.Equal(t, PolicyStat{
		PolicyID:         1,
		InheritedTeamID:  2,
		PassingHostCount: 0,
		FailingHostCount: 0,
	}, policyStat)

	// assert team1 policy stat
	err = db.Get(&policyStat, `SELECT policy_id, inherited_team_id, passing_host_count, failing_host_count FROM policy_stats WHERE policy_id = 2 AND inherited_team_id = 0`)
	require.NoError(t, err)
	require.Equal(t, PolicyStat{
		PolicyID:         2,
		InheritedTeamID:  0,
		PassingHostCount: 1,
		FailingHostCount: 1,
	}, policyStat)
}
