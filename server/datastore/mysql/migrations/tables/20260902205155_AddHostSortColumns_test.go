package tables

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUp_20260902205155(t *testing.T) {
	db := applyUpToPrev(t)

	// Insert a team.
	_, err := db.Exec(`INSERT INTO teams (name, description) VALUES ('Alpha', '')`)
	require.NoError(t, err)

	now := time.Now().UTC().Truncate(time.Second)
	yesterday := now.Add(-24 * time.Hour)

	// Insert a host with a team.
	res, err := db.Exec(
		`INSERT INTO hosts (osquery_host_id, node_key, hostname, uuid, platform, team_id, created_at)
		 VALUES ('osq-1', 'nk-1', 'host-1', 'uuid-1', 'darwin', 1, ?)`, yesterday,
	)
	require.NoError(t, err)
	hostWithTeamID, _ := res.LastInsertId()

	// Insert a host without a team (NULL team_id).
	res, err = db.Exec(
		`INSERT INTO hosts (osquery_host_id, node_key, hostname, uuid, platform, team_id, created_at)
		 VALUES ('osq-2', 'nk-2', 'host-2', 'uuid-2', 'windows', NULL, ?)`, yesterday,
	)
	require.NoError(t, err)
	hostNoTeamID, _ := res.LastInsertId()

	// Populate adjacency tables for host 1.
	_, err = db.Exec(`INSERT INTO host_seen_times (host_id, seen_time) VALUES (?, ?)`, hostWithTeamID, now)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO host_updates (host_id, software_updated_at) VALUES (?, ?)`, hostWithTeamID, now)
	require.NoError(t, err)
	_, err = db.Exec(
		`INSERT INTO host_disks (host_id, gigs_disk_space_available, percent_disk_space_available, gigs_total_disk_space)
		 VALUES (?, 42.50, 17.00, 250.00)`, hostWithTeamID,
	)
	require.NoError(t, err)
	_, err = db.Exec(
		`INSERT INTO host_issues (host_id, failing_policies_count, critical_vulnerabilities_count, total_issues_count)
		 VALUES (?, 3, 1, 4)`, hostWithTeamID,
	)
	require.NoError(t, err)

	// Host 2 has NO adjacency rows to exercise the COALESCE/NULL fallback path.

	// Apply the migration under test.
	applyNext(t, db)

	// Verify host 1: values populated from adjacency tables.
	var (
		sortSeenTime            time.Time
		sortSoftwareUpdatedAt   time.Time
		sortGigsAvail           float64
		sortGigsTotal           float64
		sortPercentAvail        float64
		sortTotalIssues         int
		sortTeamName            *string
	)
	err = db.QueryRow(
		`SELECT sort_seen_time, sort_software_updated_at,
		        sort_gigs_disk_space_available, sort_gigs_total_disk_space,
		        sort_percent_disk_space_available, sort_total_issues_count,
		        sort_team_name
		 FROM hosts WHERE id = ?`, hostWithTeamID,
	).Scan(&sortSeenTime, &sortSoftwareUpdatedAt, &sortGigsAvail, &sortGigsTotal, &sortPercentAvail, &sortTotalIssues, &sortTeamName)
	require.NoError(t, err)

	require.WithinDuration(t, now, sortSeenTime, time.Second)
	require.WithinDuration(t, now, sortSoftwareUpdatedAt, time.Second)
	require.InDelta(t, 42.50, sortGigsAvail, 0.01)
	require.InDelta(t, 250.00, sortGigsTotal, 0.01)
	require.InDelta(t, 17.00, sortPercentAvail, 0.01)
	require.Equal(t, 4, sortTotalIssues)
	require.NotNil(t, sortTeamName)
	require.Equal(t, "Alpha", *sortTeamName)

	// Verify host 2: fallback values (no adjacency rows).
	var (
		sortSeenTime2            time.Time
		sortSoftwareUpdatedAt2   time.Time
		sortGigsAvail2           float64
		sortGigsTotal2           float64
		sortPercentAvail2        float64
		sortTotalIssues2         int
		sortTeamName2            *string
	)
	err = db.QueryRow(
		`SELECT sort_seen_time, sort_software_updated_at,
		        sort_gigs_disk_space_available, sort_gigs_total_disk_space,
		        sort_percent_disk_space_available, sort_total_issues_count,
		        sort_team_name
		 FROM hosts WHERE id = ?`, hostNoTeamID,
	).Scan(&sortSeenTime2, &sortSoftwareUpdatedAt2, &sortGigsAvail2, &sortGigsTotal2, &sortPercentAvail2, &sortTotalIssues2, &sortTeamName2)
	require.NoError(t, err)

	// seen_time and software_updated_at fall back to created_at.
	require.WithinDuration(t, yesterday, sortSeenTime2, time.Second)
	require.WithinDuration(t, yesterday, sortSoftwareUpdatedAt2, time.Second)
	// Disk and issues fall back to 0.
	require.InDelta(t, 0, sortGigsAvail2, 0.01)
	require.InDelta(t, 0, sortGigsTotal2, 0.01)
	require.InDelta(t, 0, sortPercentAvail2, 0.01)
	require.Equal(t, 0, sortTotalIssues2)
	// No team = NULL team_name.
	require.Nil(t, sortTeamName2)
}
