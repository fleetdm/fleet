package tables

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUp_20261015212550(t *testing.T) {
	db := applyUpToPrev(t)

	hostID := execNoErrLastID(t, db, `INSERT INTO hosts (osquery_host_id, node_key, hostname, uuid) VALUES ('host-1', 'host-1', 'host-1', 'host-1')`)
	execNoErr(t, db, `INSERT INTO vpp_apps (adam_id, platform) VALUES ('ios-app', 'ios')`)
	versionID := execNoErrLastID(t, db, `INSERT INTO vpp_apps_teams (adam_id, platform, global_or_team_id, name) VALUES ('ios-app', 'ios', 0, 'Default version')`)
	createdAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	updatedAt := time.Date(2026, 1, 3, 4, 5, 6, 0, time.UTC)
	execNoErr(t, db, `
		INSERT INTO host_vpp_software_installs (host_id, adam_id, platform, command_uuid, vpp_app_team_id, created_at, updated_at)
		VALUES (?, 'ios-app', 'ios', 'cmd-1', ?, ?, ?)`, hostID, versionID, createdAt, updatedAt)

	applyNext(t, db)

	// read the indexes, the new index should be added and the single-column index should be dropped
	require.Equal(t, []string{"vpp_app_team_id", "host_id", "canceled", "removed", "id"},
		indexColumns(t, db, "host_vpp_software_installs", "idx_host_vpp_software_installs_version_host_id"))
	require.Empty(t, indexColumns(t, db, "host_vpp_software_installs", "fk_host_vpp_software_installs_vpp_app_team_id"))

	// read the install, its timestamps should be unchanged
	var install struct {
		CreatedAt time.Time `db:"created_at"`
		UpdatedAt time.Time `db:"updated_at"`
	}
	require.NoError(t, db.Get(&install, `SELECT created_at, updated_at FROM host_vpp_software_installs WHERE command_uuid = 'cmd-1'`))
	require.True(t, createdAt.Equal(install.CreatedAt), "created_at changed: %s", install.CreatedAt)
	require.True(t, updatedAt.Equal(install.UpdatedAt), "updated_at changed: %s", install.UpdatedAt)

	// run the migration again, the indexes should be unchanged
	tx, err := db.Begin()
	require.NoError(t, err)
	require.NoError(t, Up_20261015212550(tx))
	require.NoError(t, tx.Commit())
	require.Equal(t, []string{"vpp_app_team_id", "host_id", "canceled", "removed", "id"},
		indexColumns(t, db, "host_vpp_software_installs", "idx_host_vpp_software_installs_version_host_id"))
	require.Empty(t, indexColumns(t, db, "host_vpp_software_installs", "fk_host_vpp_software_installs_vpp_app_team_id"))

	// insert an install of a version that doesn't exist, the foreign key should reject it
	_, err = db.Exec(`
		INSERT INTO host_vpp_software_installs (host_id, adam_id, platform, command_uuid, vpp_app_team_id)
		VALUES (?, 'ios-app', 'ios', 'cmd-2', ?)`, hostID, versionID+1000)
	require.Error(t, err)

	// delete the version, the foreign key should clear the install's version
	execNoErr(t, db, `DELETE FROM vpp_apps_teams WHERE id = ?`, versionID)
	var installVersionID *uint
	require.NoError(t, db.Get(&installVersionID, `SELECT vpp_app_team_id FROM host_vpp_software_installs WHERE command_uuid = 'cmd-1'`))
	require.Nil(t, installVersionID)
}
