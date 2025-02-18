package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20240730215453(t *testing.T) {
	db := applyUpToPrev(t)

	stmt := "INSERT INTO `software_titles_host_counts` (`software_title_id`, `hosts_count`, `team_id`) VALUES (?, ?, ?)"

	s1t1 := 10
	s2t1 := 1
	s1t2 := 15
	s2t2 := 3
	s1g := 40
	s2g := 0 // edge case where global count is incorrectly 0

	// insert software 1 team 1 counts
	_, err := db.Exec(stmt, 1, s1t1, 1)
	require.NoError(t, err)

	// insert software 2 team 1 counts
	_, err = db.Exec(stmt, 2, s2t1, 1)
	require.NoError(t, err)

	// insert software 1 team 2 counts
	_, err = db.Exec(stmt, 1, s1t2, 2)
	require.NoError(t, err)

	// insert software 2 team 2 counts
	_, err = db.Exec(stmt, 2, s2t2, 2)
	require.NoError(t, err)

	// insert software 1 global counts (team_id = 0)
	_, err = db.Exec(stmt, 1, s1g, 0)
	require.NoError(t, err)

	// insert software 2 global counts (team_id = 0)
	_, err = db.Exec(stmt, 2, s2g, 0)
	require.NoError(t, err)

	applyNext(t, db)

	// Ensure the data is still there
	var result struct {
		SoftwareID  uint `db:"software_title_id"`
		HostsCount  int  `db:"hosts_count"`
		TeamID      uint `db:"team_id"`
		GlobalStats bool `db:"global_stats"`
	}
	assertHostCount := func(softwareID, hostsCount int, teamID uint, globalStats bool) {
		t.Helper()
		res := db.QueryRow("SELECT `software_title_id`, `hosts_count`, `team_id`, `global_stats` FROM `software_titles_host_counts` WHERE `software_title_id` = ? AND `team_id` = ? AND global_stats = ?", softwareID, teamID, globalStats)
		err = res.Scan(&result.SoftwareID, &result.HostsCount, &result.TeamID, &result.GlobalStats)
		require.NoError(t, err)
		require.EqualValues(t, softwareID, result.SoftwareID)
		require.Equal(t, hostsCount, result.HostsCount)
		require.Equal(t, teamID, result.TeamID)
		require.Equal(t, globalStats, result.GlobalStats)
	}

	// software 1 team 1
	assertHostCount(1, s1t1, 1, false)
	// software 1 team 2
	assertHostCount(1, s1t2, 2, false)
	// software 1 global
	assertHostCount(1, s1g, 0, true)
	// software 1 no team
	assertHostCount(1, s1g-s1t1-s1t2, 0, false)

	// software 2 team 1
	assertHostCount(2, s2t1, 1, false)
	// software 2 team 2
	assertHostCount(2, s2t2, 2, false)
	// software 2 global
	assertHostCount(2, s2g, 0, true)
	// software 2 no team
	assertHostCount(2, 0, 0, false) // edge case where global count should not be negative
}
