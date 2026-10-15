package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260925160000(t *testing.T) {
	db := applyUpToPrev(t)

	execNoErr(t, db, `INSERT INTO hosts (osquery_host_id, node_key, hostname, uuid) VALUES ('patch-host', 'patch-key', 'patch-host', 'patch-uuid')`)
	var hostID uint
	require.NoError(t, db.Get(&hostID, `SELECT id FROM hosts WHERE osquery_host_id = 'patch-host'`))
	execNoErr(t, db, `INSERT INTO notifications_end_user (uuid, host_id, status, kind, payload, expires_at) VALUES ('notification-uuid', ?, 'pending', 'patch', '{}', NOW(6))`, hostID)
	titleID := execNoErrLastID(t, db, `INSERT INTO software_titles (name, source) VALUES ('Patch App', 'apps')`)
	execNoErr(t, db, `INSERT INTO patch_notification_apps (notification_uuid, software_title_id) VALUES ('notification-uuid', ?)`, titleID)

	applyNext(t, db)

	// read the app added before the migration, the column should default to 0
	var updatedInInventory bool
	require.NoError(t, db.Get(&updatedInInventory, `SELECT updated_in_inventory FROM patch_notification_apps WHERE notification_uuid = 'notification-uuid'`))
	require.False(t, updatedInInventory)
}
