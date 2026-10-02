package tables

import (
	"database/sql"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20261001120357(t *testing.T) {
	db := applyUpToPrev(t)

	titleID := execNoErrLastID(t, db, `INSERT INTO software_titles (name, source) VALUES ('Live', 'apps')`)
	scriptID := execNoErrLastID(t, db, `INSERT INTO script_contents (contents, md5_checksum) VALUES ('#!/bin/sh', UNHEX(MD5('live')))`)
	liveInstallerID := execNoErrLastID(t, db, `
		INSERT INTO software_installers
			(global_or_team_id, title_id, filename, extension, version, platform,
			 install_script_content_id, uninstall_script_content_id, storage_id, package_ids, patch_query)
		VALUES (0, ?, 'live.pkg', 'pkg', '1.0', 'darwin', ?, ?, 'live', '', '')`, titleID, scriptID, scriptID)

	queue := func(execID, activityType string, activated bool, installerID any) {
		uaID := execNoErrLastID(t, db, `INSERT INTO upcoming_activities (host_id, activity_type, execution_id, payload, activated_at)
			VALUES (1, ?, ?, '{}', IF(?, NOW(6), NULL))`, activityType, execID, activated)
		if activityType != "script" {
			execNoErr(t, db, `INSERT INTO software_install_upcoming_activities (upcoming_activity_id, software_installer_id) VALUES (?, ?)`,
				uaID, installerID)
		}
	}

	// orphans
	execNoErr(t, db, `INSERT INTO host_software_installs (execution_id, host_id) VALUES ('orphan-install', 1)`)
	queue("orphan-install", "software_install", true, nil)
	execNoErr(t, db, `INSERT INTO host_software_installs (execution_id, host_id, uninstall) VALUES ('orphan-uninstall', 1, 1)`)
	execNoErr(t, db, `INSERT INTO host_script_results (host_id, execution_id, output) VALUES (1, 'orphan-uninstall', '')`)
	queue("orphan-uninstall", "software_uninstall", true, nil)
	queue("orphan-queued", "software_install", false, nil)

	// untouched
	untouched := map[string]string{
		"completed":    `INSERT INTO host_software_installs (execution_id, host_id, install_script_exit_code) VALUES ('completed', 1, 0)`,
		"canceled":     `INSERT INTO host_software_installs (execution_id, host_id, canceled) VALUES ('canceled', 1, 1)`,
		"removed":      `INSERT INTO host_software_installs (execution_id, host_id, removed) VALUES ('removed', 1, 1)`,
		"host-deleted": `INSERT INTO host_software_installs (execution_id, host_id, host_deleted_at) VALUES ('host-deleted', 1, NOW())`,
		"no-pre-query": `INSERT INTO host_software_installs (execution_id, host_id, pre_install_query_output) VALUES ('no-pre-query', 1, '')`,
	}
	for _, stmt := range untouched {
		execNoErr(t, db, stmt)
	}
	execNoErr(t, db, `INSERT INTO host_software_installs (execution_id, host_id, software_installer_id) VALUES ('live', 1, ?)`, liveInstallerID)
	queue("live", "software_install", true, liveInstallerID)
	queue("script", "script", false, nil)

	type row struct {
		Status          sql.NullString `db:"status"`
		InstallExit     sql.NullInt64  `db:"install_script_exit_code"`
		UninstallExit   sql.NullInt64  `db:"uninstall_script_exit_code"`
		InstallOutput   sql.NullString `db:"install_script_output"`
		UninstallOutput sql.NullString `db:"uninstall_script_output"`
	}
	getRow := func(execID string) row {
		var r row
		require.NoError(t, db.Get(&r, `SELECT status, install_script_exit_code, uninstall_script_exit_code, install_script_output, uninstall_script_output
			FROM host_software_installs WHERE execution_id = ?`, execID))
		return r
	}
	before := map[string]row{}
	for execID := range untouched {
		before[execID] = getRow(execID)
	}
	before["live"] = getRow("live")

	// cross a batch boundary on both tables
	const bulk = 1001
	var values []string
	for i := range bulk {
		values = append(values, fmt.Sprintf("('bulk-hsi-%d', 1)", i))
	}
	execNoErr(t, db, `INSERT INTO host_software_installs (execution_id, host_id) VALUES `+strings.Join(values, ","))
	values = values[:0]
	for i := range bulk {
		values = append(values, fmt.Sprintf("(1, 'software_install', 'bulk-ua-%d', '{}')", i))
	}
	execNoErr(t, db, `INSERT INTO upcoming_activities (host_id, activity_type, execution_id, payload) VALUES `+strings.Join(values, ","))
	execNoErr(t, db, `INSERT INTO software_install_upcoming_activities (upcoming_activity_id)
		SELECT id FROM upcoming_activities WHERE execution_id LIKE 'bulk-ua-%'`)

	applyNext(t, db)

	install := getRow("orphan-install")
	require.Equal(t, "failed_install", install.Status.String)
	require.EqualValues(t, -4, install.InstallExit.Int64)
	require.Equal(t, "Installer no longer exists on the server.", install.InstallOutput.String)

	uninstall := getRow("orphan-uninstall")
	require.Equal(t, "failed_uninstall", uninstall.Status.String)
	require.EqualValues(t, -4, uninstall.UninstallExit.Int64)
	require.Equal(t, "Installer no longer exists on the server.", uninstall.UninstallOutput.String)
	require.False(t, uninstall.InstallExit.Valid)

	count := func(query string, args ...any) int {
		var n int
		require.NoError(t, db.Get(&n, query, args...))
		return n
	}
	require.Zero(t, count(`SELECT COUNT(*) FROM host_script_results WHERE execution_id = 'orphan-uninstall'`))
	for _, execID := range []string{"orphan-install", "orphan-uninstall", "orphan-queued"} {
		require.Zero(t, count(`SELECT COUNT(*) FROM upcoming_activities WHERE execution_id = ?`, execID), execID)
	}

	for execID, want := range before {
		require.Equal(t, want, getRow(execID), execID)
	}
	require.Equal(t, 1, count(`SELECT COUNT(*) FROM upcoming_activities WHERE execution_id = 'live'`))
	require.Equal(t, 1, count(`SELECT COUNT(*) FROM upcoming_activities WHERE execution_id = 'script'`))

	require.Equal(t, bulk, count(`SELECT COUNT(*) FROM host_software_installs WHERE execution_id LIKE 'bulk-hsi-%' AND status = 'failed_install'`))
	require.Zero(t, count(`SELECT COUNT(*) FROM upcoming_activities WHERE execution_id LIKE 'bulk-ua-%'`))

	tx, err := db.Begin()
	require.NoError(t, err)
	defer tx.Rollback() //nolint:errcheck
	n, err := countOrphanedSoftwareInstalls(tx)
	require.NoError(t, err)
	require.Zero(t, n)
	n, err = countOrphanedInstallQueueRows(tx)
	require.NoError(t, err)
	require.Zero(t, n)
}
