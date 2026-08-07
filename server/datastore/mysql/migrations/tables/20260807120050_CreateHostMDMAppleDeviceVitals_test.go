package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260807120050(t *testing.T) {
	db := applyUpToPrev(t)

	// Seed a host so we can attach vitals to it.
	_, err := db.Exec(`
		INSERT INTO hosts (osquery_host_id, node_key, hostname, uuid)
		VALUES (?, ?, ?, ?)`,
		"host-1-osquery-id", "host-1-node-key", "host-1", "host-1-uuid",
	)
	require.NoError(t, err)

	// Apply current migration.
	applyNext(t, db)

	_, err = db.Exec(`
		INSERT INTO host_mdm_apple_device_vitals (
			host_uuid, udid, model_number, battery_level, cellular_technology,
			app_analytics_enabled, last_cloud_backup_date,
			accessibility_settings, organization_info, mdm_options, device_properties_attestation
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"host-1-uuid", "00008030-ABCDEF", "MK1A3LL/A", 0.85, 1,
		true, "2026-07-01 00:00:00",
		`{"voice_over_enabled": true}`, `{"organization_name": "Acme"}`, `{"bootstrap_token_allowed": true}`, `["ZGVy"]`,
	)
	require.NoError(t, err, "insert with all column types should succeed")

	// Re-inserting for the same host_uuid should violate the PK.
	_, err = db.Exec(`INSERT INTO host_mdm_apple_device_vitals (host_uuid) VALUES (?)`, "host-1-uuid")
	require.Error(t, err, "duplicate host_uuid should be rejected")

	// service_subscriptions: multiple rows per host_uuid, unique by (host_uuid, slot).
	_, err = db.Exec(`
		INSERT INTO host_mdm_apple_service_subscriptions (host_uuid, slot, iccid, is_data_preferred)
		VALUES (?, ?, ?, ?)`,
		"host-1-uuid", "slot-1", "8901410321111111111", true,
	)
	require.NoError(t, err)
	_, err = db.Exec(`
		INSERT INTO host_mdm_apple_service_subscriptions (host_uuid, slot, iccid, is_data_preferred)
		VALUES (?, ?, ?, ?)`,
		"host-1-uuid", "slot-2", "8901410321222222222", false,
	)
	require.NoError(t, err, "a second subscription slot for the same host should be allowed")

	_, err = db.Exec(`
		INSERT INTO host_mdm_apple_service_subscriptions (host_uuid, slot) VALUES (?, ?)`,
		"host-1-uuid", "slot-1",
	)
	require.Error(t, err, "duplicate (host_uuid, slot) should be rejected")

	var count int
	err = db.QueryRow(`SELECT COUNT(*) FROM host_mdm_apple_service_subscriptions WHERE host_uuid = ?`, "host-1-uuid").Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 2, count)
}
