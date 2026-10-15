package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20261015210759(t *testing.T) {
	db := applyUpToPrev(t)

	titleID := execNoErrLastID(t, db, `INSERT INTO software_titles (name, source) VALUES ('demo', 'apps')`)
	execNoErr(t, db, `INSERT INTO script_contents (id, md5_checksum, contents) VALUES (1, 'demo-checksum', 'demo')`)
	installerID := execNoErrLastID(t, db, `
INSERT INTO software_installers (title_id, global_or_team_id, filename, version, platform, install_script_content_id, uninstall_script_content_id, storage_id, package_ids, patch_query)
VALUES (?, 0, 'demo.pkg', '1.0', 'darwin', 1, 1, 'storage-demo', '', '')`, titleID)
	scriptID := execNoErrLastID(t, db, `INSERT INTO scripts (global_or_team_id, name, script_content_id) VALUES (0, 'demo.sh', 1)`)
	execNoErr(t, db, `INSERT INTO vpp_apps (adam_id, platform) VALUES ('1234567890', 'darwin')`)
	vppAppTeamID := execNoErrLastID(t, db, `INSERT INTO vpp_apps_teams (adam_id, global_or_team_id, platform, name) VALUES ('1234567890', 0, 'darwin', 'Demo')`)
	policyID := execNoErrLastID(t, db, `INSERT INTO policies (name, query, description, checksum) VALUES ('demo', 'SELECT 1', '', 'demo-checksum')`)

	// insert two successful and one failed policy install on host 1 and one on host 2, plus manual, uninstall, pending, canceled and deleted-host runs the backfill should skip
	const insertInstall = `INSERT INTO host_software_installs
		(host_id, execution_id, software_installer_id, policy_id, uninstall, install_script_exit_code, uninstall_script_exit_code, canceled, host_deleted_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
	execNoErr(t, db, insertInstall, 1, "install-ok-1", installerID, policyID, 0, 0, nil, 0, nil)
	execNoErr(t, db, insertInstall, 1, "install-ok-2", installerID, policyID, 0, 0, nil, 0, nil)
	execNoErr(t, db, insertInstall, 1, "install-fail", installerID, policyID, 0, 1, nil, 0, nil)
	execNoErr(t, db, insertInstall, 2, "install-ok-host-2", installerID, policyID, 0, 0, nil, 0, nil)
	execNoErr(t, db, insertInstall, 1, "install-manual", installerID, nil, 0, 0, nil, 0, nil)
	execNoErr(t, db, insertInstall, 1, "uninstall", installerID, nil, 1, nil, 0, 0, nil)
	execNoErr(t, db, insertInstall, 1, "install-pending", installerID, policyID, 0, nil, nil, 0, nil)
	execNoErr(t, db, insertInstall, 1, "install-canceled", installerID, policyID, 0, 0, nil, 1, nil)
	execNoErr(t, db, insertInstall, 1, "install-deleted-host", installerID, policyID, 0, 0, nil, 0, "2026-01-01 00:00:00")

	// insert one successful and two failed policy script runs on host 1, plus manual, pending and canceled runs the backfill should skip
	const insertScriptRun = `INSERT INTO host_script_results (host_id, execution_id, output, script_id, policy_id, exit_code, canceled)
		VALUES (1, ?, '', ?, ?, ?, ?)`
	execNoErr(t, db, insertScriptRun, "script-ok", scriptID, policyID, 0, 0)
	execNoErr(t, db, insertScriptRun, "script-fail", scriptID, policyID, 1, 0)
	execNoErr(t, db, insertScriptRun, "script-timeout", scriptID, policyID, -1, 0)
	execNoErr(t, db, insertScriptRun, "script-manual", scriptID, nil, 0, 0)
	execNoErr(t, db, insertScriptRun, "script-pending", scriptID, policyID, nil, 0)
	execNoErr(t, db, insertScriptRun, "script-canceled", scriptID, policyID, 0, 1)

	// insert one verified and one failed policy VPP install on host 1, plus manual, pending and canceled installs the backfill should skip
	const insertVPPInstall = `INSERT INTO host_vpp_software_installs
		(host_id, adam_id, platform, command_uuid, vpp_app_team_id, policy_id, verification_at, verification_failed_at, canceled)
		VALUES (1, '1234567890', 'darwin', ?, ?, ?, ?, ?, ?)`
	execNoErr(t, db, insertVPPInstall, "vpp-ok", vppAppTeamID, policyID, "2026-01-01 00:00:00", nil, 0)
	execNoErr(t, db, insertVPPInstall, "vpp-fail", vppAppTeamID, policyID, nil, "2026-01-01 00:00:00", 0)
	execNoErr(t, db, insertVPPInstall, "vpp-manual", vppAppTeamID, nil, "2026-01-01 00:00:00", nil, 0)
	execNoErr(t, db, insertVPPInstall, "vpp-pending", vppAppTeamID, policyID, nil, nil, 0)
	execNoErr(t, db, insertVPPInstall, "vpp-canceled", vppAppTeamID, policyID, "2026-01-01 00:00:00", nil, 1)

	applyNext(t, db)

	type runCount struct {
		PolicyID     uint   `db:"policy_id"`
		ActivityType string `db:"activity_type"`
		HostID       uint   `db:"host_id"`
		SuccessCount uint   `db:"success_count"`
		FailCount    uint   `db:"fail_count"`
	}

	// read the backfilled counts, only finished policy runs should be counted, per host and activity type
	var counts []runCount
	require.NoError(t, db.Select(&counts, `SELECT * FROM host_policy_automation_run_counts ORDER BY activity_type, host_id`))
	expectedCounts := []runCount{
		{PolicyID: uint(policyID), ActivityType: "installed_app_store_app", HostID: 1, SuccessCount: 1, FailCount: 1}, //nolint:gosec // dismiss G115
		{PolicyID: uint(policyID), ActivityType: "installed_software", HostID: 1, SuccessCount: 2, FailCount: 1},      //nolint:gosec // dismiss G115
		{PolicyID: uint(policyID), ActivityType: "installed_software", HostID: 2, SuccessCount: 1},                    //nolint:gosec // dismiss G115
		{PolicyID: uint(policyID), ActivityType: "ran_script", HostID: 1, SuccessCount: 1, FailCount: 2},              //nolint:gosec // dismiss G115
	}
	require.Equal(t, expectedCounts, counts)

	// delete the policy, its count rows should be deleted with it
	execNoErr(t, db, `DELETE FROM policies WHERE id = ?`, policyID)
	var rows int
	require.NoError(t, db.Get(&rows, `SELECT COUNT(*) FROM host_policy_automation_run_counts`))
	require.Zero(t, rows)
}
