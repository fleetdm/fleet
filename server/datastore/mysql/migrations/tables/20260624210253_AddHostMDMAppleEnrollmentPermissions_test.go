package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260624210253(t *testing.T) {
	db := applyUpToPrev(t)

	// Seed hosts: manually-enrolled Mac (should backfill), DEP-enrolled Mac
	// (should NOT), unenrolled Mac, Windows manual-enrolled (should NOT —
	// table is Apple-specific), manually-enrolled Mac with blank uuid (should NOT).
	execNoErr(t, db, `INSERT INTO hosts (osquery_host_id, node_key, hostname, uuid, platform) VALUES (?, ?, ?, ?, ?)`,
		"oh-manual", "nk-manual", "manual.local", "uuid-manual", "darwin")
	execNoErr(t, db, `INSERT INTO hosts (osquery_host_id, node_key, hostname, uuid, platform) VALUES (?, ?, ?, ?, ?)`,
		"oh-dep", "nk-dep", "dep.local", "uuid-dep", "darwin")
	execNoErr(t, db, `INSERT INTO hosts (osquery_host_id, node_key, hostname, uuid, platform) VALUES (?, ?, ?, ?, ?)`,
		"oh-unenrolled", "nk-unenrolled", "unenrolled.local", "uuid-unenrolled", "darwin")
	execNoErr(t, db, `INSERT INTO hosts (osquery_host_id, node_key, hostname, uuid, platform) VALUES (?, ?, ?, ?, ?)`,
		"oh-win", "nk-win", "win.local", "uuid-win", "windows")
	execNoErr(t, db, `INSERT INTO hosts (osquery_host_id, node_key, hostname, uuid, platform) VALUES (?, ?, ?, ?, ?)`,
		"oh-blank", "nk-blank", "blank.local", "", "darwin")

	var manualID, depID, unenrolledID, winID, blankID uint
	require.NoError(t, db.Get(&manualID, `SELECT id FROM hosts WHERE uuid = 'uuid-manual'`))
	require.NoError(t, db.Get(&depID, `SELECT id FROM hosts WHERE uuid = 'uuid-dep'`))
	require.NoError(t, db.Get(&unenrolledID, `SELECT id FROM hosts WHERE uuid = 'uuid-unenrolled'`))
	require.NoError(t, db.Get(&winID, `SELECT id FROM hosts WHERE uuid = 'uuid-win'`))
	require.NoError(t, db.Get(&blankID, `SELECT id FROM hosts WHERE osquery_host_id = 'oh-blank'`))

	execNoErr(t, db, `INSERT INTO host_mdm (host_id, enrolled, server_url, installed_from_dep, is_personal_enrollment) VALUES (?, 1, 'https://fleet.local', 0, 0)`, manualID)
	execNoErr(t, db, `INSERT INTO host_mdm (host_id, enrolled, server_url, installed_from_dep, is_personal_enrollment) VALUES (?, 1, 'https://fleet.local', 1, 0)`, depID)
	execNoErr(t, db, `INSERT INTO host_mdm (host_id, enrolled, server_url, installed_from_dep, is_personal_enrollment) VALUES (?, 0, '', 0, 0)`, unenrolledID)
	execNoErr(t, db, `INSERT INTO host_mdm (host_id, enrolled, server_url, installed_from_dep, is_personal_enrollment) VALUES (?, 1, 'https://fleet.local', 0, 0)`, winID)
	execNoErr(t, db, `INSERT INTO host_mdm (host_id, enrolled, server_url, installed_from_dep, is_personal_enrollment) VALUES (?, 1, 'https://fleet.local', 0, 0)`, blankID)

	applyNext(t, db)

	// Manual-enrolled Mac must have an 8191 row.
	var rights int
	require.NoError(t, db.Get(&rights, `SELECT access_rights FROM host_mdm_apple_enrollment_permissions WHERE host_uuid = ?`, "uuid-manual"))
	require.Equal(t, 8191, rights)

	var count int
	require.NoError(t, db.Get(&count, `SELECT COUNT(*) FROM host_mdm_apple_enrollment_permissions WHERE host_uuid = ?`, "uuid-dep"))
	require.Equal(t, 0, count)

	require.NoError(t, db.Get(&count, `SELECT COUNT(*) FROM host_mdm_apple_enrollment_permissions WHERE host_uuid = ?`, "uuid-unenrolled"))
	require.Equal(t, 0, count)

	require.NoError(t, db.Get(&count, `SELECT COUNT(*) FROM host_mdm_apple_enrollment_permissions WHERE host_uuid = ?`, "uuid-win"))
	require.Equal(t, 0, count)

	require.NoError(t, db.Get(&count, `SELECT COUNT(*) FROM host_mdm_apple_enrollment_permissions WHERE host_uuid = ?`, ""))
	require.Equal(t, 0, count)

	// Upsert must update access_rights on duplicate.
	execNoErr(t, db, `
		INSERT INTO host_mdm_apple_enrollment_permissions (host_uuid, access_rights)
		VALUES (?, 8179)
		ON DUPLICATE KEY UPDATE access_rights = VALUES(access_rights)`, "uuid-manual")
	require.NoError(t, db.Get(&rights, `SELECT access_rights FROM host_mdm_apple_enrollment_permissions WHERE host_uuid = ?`, "uuid-manual"))
	require.Equal(t, 8179, rights)
}
