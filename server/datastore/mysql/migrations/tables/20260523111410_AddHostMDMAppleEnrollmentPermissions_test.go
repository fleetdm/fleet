package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260523111410(t *testing.T) {
	db := applyUpToPrev(t)

	// Seed hosts: a manually-enrolled Mac (should backfill), an ADE Mac
	// (installed_from_dep=1, should NOT), an unenrolled Mac, and a Windows
	// host enrolled with installed_from_dep=0 (also should NOT — the table
	// is Apple-specific).
	execNoErr(t, db, `INSERT INTO hosts (osquery_host_id, node_key, hostname, uuid, platform) VALUES (?, ?, ?, ?, ?)`,
		"oh-manual", "nk-manual", "manual.local", "uuid-manual", "darwin")
	execNoErr(t, db, `INSERT INTO hosts (osquery_host_id, node_key, hostname, uuid, platform) VALUES (?, ?, ?, ?, ?)`,
		"oh-dep", "nk-dep", "dep.local", "uuid-dep", "darwin")
	execNoErr(t, db, `INSERT INTO hosts (osquery_host_id, node_key, hostname, uuid, platform) VALUES (?, ?, ?, ?, ?)`,
		"oh-unenrolled", "nk-unenrolled", "unenrolled.local", "uuid-unenrolled", "darwin")
	execNoErr(t, db, `INSERT INTO hosts (osquery_host_id, node_key, hostname, uuid, platform) VALUES (?, ?, ?, ?, ?)`,
		"oh-win", "nk-win", "win.local", "uuid-win", "windows")

	var manualID, depID, unenrolledID, winID uint
	require.NoError(t, db.Get(&manualID, `SELECT id FROM hosts WHERE uuid = 'uuid-manual'`))
	require.NoError(t, db.Get(&depID, `SELECT id FROM hosts WHERE uuid = 'uuid-dep'`))
	require.NoError(t, db.Get(&unenrolledID, `SELECT id FROM hosts WHERE uuid = 'uuid-unenrolled'`))
	require.NoError(t, db.Get(&winID, `SELECT id FROM hosts WHERE uuid = 'uuid-win'`))

	// Enroll manually (installed_from_dep=0).
	execNoErr(t, db, `INSERT INTO host_mdm (host_id, enrolled, server_url, installed_from_dep, is_personal_enrollment) VALUES (?, 1, 'https://fleet.local', 0, 0)`, manualID)
	// Enroll via DEP (installed_from_dep=1) — should not be backfilled.
	execNoErr(t, db, `INSERT INTO host_mdm (host_id, enrolled, server_url, installed_from_dep, is_personal_enrollment) VALUES (?, 1, 'https://fleet.local', 1, 0)`, depID)
	// Not enrolled — should not be backfilled.
	execNoErr(t, db, `INSERT INTO host_mdm (host_id, enrolled, server_url, installed_from_dep, is_personal_enrollment) VALUES (?, 0, '', 0, 0)`, unenrolledID)
	// Windows manual enrollment (installed_from_dep=0 because not AAD/Autopilot)
	// — should NOT land in this Apple-specific table.
	execNoErr(t, db, `INSERT INTO host_mdm (host_id, enrolled, server_url, installed_from_dep, is_personal_enrollment) VALUES (?, 1, 'https://fleet.local', 0, 0)`, winID)

	applyNext(t, db)

	// Manual-enrolled Mac must have an 8191 row.
	var rights int
	require.NoError(t, db.Get(&rights, `SELECT access_rights FROM host_mdm_apple_enrollment_permissions WHERE host_id = ?`, manualID))
	require.Equal(t, 8191, rights)

	// DEP-enrolled host must NOT have a row.
	var count int
	require.NoError(t, db.Get(&count, `SELECT COUNT(*) FROM host_mdm_apple_enrollment_permissions WHERE host_id = ?`, depID))
	require.Equal(t, 0, count)

	// Unenrolled host must NOT have a row.
	require.NoError(t, db.Get(&count, `SELECT COUNT(*) FROM host_mdm_apple_enrollment_permissions WHERE host_id = ?`, unenrolledID))
	require.Equal(t, 0, count)

	// Windows host must NOT have a row (Apple-specific table).
	require.NoError(t, db.Get(&count, `SELECT COUNT(*) FROM host_mdm_apple_enrollment_permissions WHERE host_id = ?`, winID))
	require.Equal(t, 0, count)

	// Upsert via INSERT ... ON DUPLICATE KEY UPDATE must update access_rights.
	execNoErr(t, db, `
		INSERT INTO host_mdm_apple_enrollment_permissions (host_id, access_rights)
		VALUES (?, 7167)
		ON DUPLICATE KEY UPDATE access_rights = VALUES(access_rights)`, manualID)
	require.NoError(t, db.Get(&rights, `SELECT access_rights FROM host_mdm_apple_enrollment_permissions WHERE host_id = ?`, manualID))
	require.Equal(t, 7167, rights)
}
