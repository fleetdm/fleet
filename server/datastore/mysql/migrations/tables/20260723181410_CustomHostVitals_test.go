package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260723181410(t *testing.T) {
	db := applyUpToPrev(t)

	// Seed a host so we can attach a per-host value.
	res, err := db.Exec(`
		INSERT INTO hosts (osquery_host_id, node_key, hostname, uuid)
		VALUES (?, ?, ?, ?)`,
		"host-1-osquery-id", "host-1-node-key", "host-1", "host-1-uuid",
	)
	require.NoError(t, err)
	hostIDInt, err := res.LastInsertId()
	require.NoError(t, err)
	hostID := uint(hostIDInt) //nolint:gosec

	// Apply current migration.
	applyNext(t, db)

	// Insert a definition; the name is unique.
	res, err = db.Exec(`INSERT INTO custom_host_vitals (name) VALUES ('Asset tag')`)
	require.NoError(t, err)
	vitalIDInt, err := res.LastInsertId()
	require.NoError(t, err)
	vitalID := uint(vitalIDInt) //nolint:gosec

	_, err = db.Exec(`INSERT INTO custom_host_vitals (name) VALUES ('Asset tag')`)
	require.Error(t, err, "duplicate name should be rejected")

	// value is NOT NULL.
	_, err = db.Exec(`INSERT INTO host_custom_host_vitals (host_id, custom_host_vital_id, value) VALUES (?, ?, NULL)`, hostID, vitalID)
	require.Error(t, err, "NULL value should be rejected")

	// Insert a per-host value; (host_id, custom_host_vital_id) is unique.
	_, err = db.Exec(`INSERT INTO host_custom_host_vitals (host_id, custom_host_vital_id, value) VALUES (?, ?, 'engineering')`, hostID, vitalID)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO host_custom_host_vitals (host_id, custom_host_vital_id, value) VALUES (?, ?, 'other')`, hostID, vitalID)
	require.Error(t, err, "duplicate (host_id, custom_host_vital_id) should be rejected")

	// Deleting the definition cascades to the per-host value.
	_, err = db.Exec(`DELETE FROM custom_host_vitals WHERE id = ?`, vitalID)
	require.NoError(t, err)
	var count int
	err = db.QueryRow(`SELECT COUNT(*) FROM host_custom_host_vitals WHERE custom_host_vital_id = ?`, vitalID).Scan(&count)
	require.NoError(t, err)
	require.Zero(t, count)
}
