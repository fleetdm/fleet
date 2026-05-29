package tables

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUp_20260529151118(t *testing.T) {
	db := applyUpToPrev(t)

	// Create a host to satisfy the FK.
	hostID := execNoErrLastID(t, db,
		`INSERT INTO hosts (osquery_host_id, node_key, uuid, hostname, hardware_serial, platform) VALUES (?, ?, ?, ?, ?, ?)`,
		"oh1", "nk1", "uuid-1", "host-1", "ABC123", "darwin",
	)

	applyNext(t, db)

	// Insert a row with explicit values.
	_, err := db.Exec(`
		INSERT INTO host_google_cloud_identity_clientstates
			(host_id, workspace_email, partner_suffix, device_user_resource, last_compliant, last_managed, last_score_reason, last_etag)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		hostID, "user@example.com", "fleet", "devices/abc/deviceUsers/xyz", true, true, "all good", "etag123",
	)
	require.NoError(t, err)

	// Insert before resolution: device_user_resource is NULL.
	_, err = db.Exec(`
		INSERT INTO host_google_cloud_identity_clientstates
			(host_id, workspace_email, partner_suffix)
		VALUES (?, ?, ?)`,
		hostID, "other@example.com", "fleet",
	)
	require.NoError(t, err)

	// Unique key (host_id, workspace_email, partner_suffix) must reject duplicates.
	_, err = db.Exec(`
		INSERT INTO host_google_cloud_identity_clientstates
			(host_id, workspace_email, partner_suffix)
		VALUES (?, ?, ?)`,
		hostID, "user@example.com", "fleet",
	)
	require.Error(t, err, "duplicate (host_id, workspace_email, partner_suffix) should violate unique key")

	// Different suffix for same host+email succeeds (per-team suffix override).
	_, err = db.Exec(`
		INSERT INTO host_google_cloud_identity_clientstates
			(host_id, workspace_email, partner_suffix)
		VALUES (?, ?, ?)`,
		hostID, "user@example.com", "fleet-engineering",
	)
	require.NoError(t, err)

	// Verify row count.
	var count int
	err = db.QueryRow(`SELECT COUNT(*) FROM host_google_cloud_identity_clientstates WHERE host_id = ?`, hostID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 3, count)

	// ON DELETE CASCADE: deleting the host removes the rows.
	_, err = db.Exec(`DELETE FROM hosts WHERE id = ?`, hostID)
	require.NoError(t, err)

	err = db.QueryRow(`SELECT COUNT(*) FROM host_google_cloud_identity_clientstates WHERE host_id = ?`, hostID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count, "rows should cascade-delete with host")
}
