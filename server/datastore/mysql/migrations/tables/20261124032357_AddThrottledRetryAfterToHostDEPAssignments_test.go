package tables

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUp_20261124032357(t *testing.T) {
	db := applyUpToPrev(t)

	hostID := execNoErrLastID(t, db,
		`INSERT INTO hosts (osquery_host_id, node_key, hostname, uuid, platform, hardware_serial) VALUES ('h', 'h', 'h', 'h', 'darwin', 'serial')`)
	execNoErr(t, db, `INSERT INTO host_dep_assignments (host_id, hardware_serial, assign_profile_response, response_updated_at)
		VALUES (?, 'serial', 'THROTTLED', NOW())`, hostID)

	applyNext(t, db)

	var retryAfter *time.Time
	require.NoError(t, db.Get(&retryAfter, `SELECT throttled_retry_after FROM host_dep_assignments WHERE host_id = ?`, hostID))
	require.Nil(t, retryAfter, "rows throttled before the migration keep the fixed cooldown")

	execNoErr(t, db, `UPDATE host_dep_assignments SET throttled_retry_after = DATE_ADD(NOW(), INTERVAL 1 HOUR) WHERE host_id = ?`, hostID)
	require.NoError(t, db.Get(&retryAfter, `SELECT throttled_retry_after FROM host_dep_assignments WHERE host_id = ?`, hostID))
	require.NotNil(t, retryAfter)
}
