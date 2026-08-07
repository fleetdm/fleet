package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260604221206(t *testing.T) {
	db := applyUpToPrev(t)

	// Create a team and a host on that team.
	_, err := db.Exec(`INSERT INTO teams (name) VALUES ('test-team')`)
	require.NoError(t, err)
	var teamID int64
	require.NoError(t, db.Get(&teamID, `SELECT id FROM teams WHERE name = 'test-team'`))

	res, err := db.Exec(`INSERT INTO hosts (hostname, uuid, platform, team_id, osquery_host_id, node_key,
		detail_updated_at, label_updated_at, policy_updated_at)
		VALUES ('android1', 'uuid1', 'android', ?, 'oq1', 'nk1', '2026-01-01', '2026-01-01', '2026-01-01')`, teamID)
	require.NoError(t, err)
	hostID, err := res.LastInsertId()
	require.NoError(t, err)

	// Create a host with no team.
	res2, err := db.Exec(`INSERT INTO hosts (hostname, uuid, platform, team_id, osquery_host_id, node_key,
		detail_updated_at, label_updated_at, policy_updated_at)
		VALUES ('android2', 'uuid2', 'android', NULL, 'oq2', 'nk2', '2026-01-01', '2026-01-01', '2026-01-01')`)
	require.NoError(t, err)
	hostID2, err := res2.LastInsertId()
	require.NoError(t, err)

	// Insert android_devices rows.
	_, err = db.Exec(`INSERT INTO android_devices (host_id, device_id, enterprise_specific_id) VALUES (?, 'd1', 'esid1')`, hostID)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO android_devices (host_id, device_id, enterprise_specific_id) VALUES (?, 'd2', 'esid2')`, hostID2)
	require.NoError(t, err)

	// Apply migration.
	applyNext(t, db)

	// Verify team_id was backfilled from hosts.team_id.
	var gotTeamID *int64
	require.NoError(t, db.Get(&gotTeamID, `SELECT team_id FROM android_devices WHERE device_id = 'd1'`))
	require.NotNil(t, gotTeamID)
	require.Equal(t, teamID, *gotTeamID)

	// Verify NULL team_id for host with no team.
	var gotTeamID2 *int64
	require.NoError(t, db.Get(&gotTeamID2, `SELECT team_id FROM android_devices WHERE device_id = 'd2'`))
	require.Nil(t, gotTeamID2)

	// Verify ON DELETE SET NULL: deleting the team sets team_id to NULL.
	_, err = db.Exec(`DELETE FROM teams WHERE id = ?`, teamID)
	require.NoError(t, err)
	require.NoError(t, db.Get(&gotTeamID, `SELECT team_id FROM android_devices WHERE device_id = 'd1'`))
	require.Nil(t, gotTeamID)
}
