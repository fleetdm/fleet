package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260624210311(t *testing.T) {
	db := applyUpToPrev(t)

	execNoErr(t, db, `INSERT INTO hosts (osquery_host_id, node_key, hostname, uuid, platform) VALUES (?, ?, ?, ?, ?)`,
		"oh-personal", "nk-personal", "personal.local", "uuid-personal", "ios")
	execNoErr(t, db, `INSERT INTO hosts (osquery_host_id, node_key, hostname, uuid, platform) VALUES (?, ?, ?, ?, ?)`,
		"oh-manual", "nk-manual", "manual.local", "uuid-manual", "darwin")

	var personalID, manualID uint
	require.NoError(t, db.Get(&personalID, `SELECT id FROM hosts WHERE uuid = 'uuid-personal'`))
	require.NoError(t, db.Get(&manualID, `SELECT id FROM hosts WHERE uuid = 'uuid-manual'`))

	execNoErr(t, db, `INSERT INTO host_mdm (host_id, enrolled, server_url, installed_from_dep, is_personal_enrollment) VALUES (?, 1, 'https://fleet.local', 0, 1)`, personalID)
	execNoErr(t, db, `INSERT INTO host_mdm (host_id, enrolled, server_url, installed_from_dep, is_personal_enrollment) VALUES (?, 1, 'https://fleet.local', 0, 0)`, manualID)

	applyNext(t, db)

	var status string
	require.NoError(t, db.Get(&status, `SELECT enrollment_status FROM host_mdm WHERE host_id = ?`, personalID))
	require.Equal(t, "On (manual - personal)", status)

	require.NoError(t, db.Get(&status, `SELECT enrollment_status FROM host_mdm WHERE host_id = ?`, manualID))
	require.Equal(t, "On (manual)", status)
}
