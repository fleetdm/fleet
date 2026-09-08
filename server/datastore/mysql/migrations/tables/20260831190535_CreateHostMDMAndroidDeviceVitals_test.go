package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260831190535(t *testing.T) {
	db := applyUpToPrev(t)

	// Seed a host so we can attach vitals to it.
	_, err := db.Exec(`
		INSERT INTO hosts (osquery_host_id, node_key, hostname, uuid)
		VALUES (?, ?, ?, ?)`,
		"android-host-osquery-id", "android-host-node-key", "android-host", "android-host-uuid",
	)
	require.NoError(t, err)

	applyNext(t, db)

	_, err = db.Exec(`
		INSERT INTO host_mdm_android_device_vitals (
			host_uuid, adb_enabled, passcode_protected, play_protect_enabled,
			encryption_type, manufacturer, security_update_version, device_kernel_version,
			bootloader_version, system_update_status, security_posture, api_level,
			security_posture_details, telephony_infos
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"android-host-uuid", true, true, false,
		"ACTIVE", "Google", "2026-05-01", "6.1.75-android14",
		"slider-1.4-12345678", "UP_TO_DATE", "SECURE", 36,
		`[{"security_risk": "UNKNOWN_OS", "advice": ["Update the OS"]}]`,
		`[{"phone_number": "+15555550100", "carrier_name": "Acme Mobile"}]`,
	)
	require.NoError(t, err, "insert with all column types should succeed")

	// Vitals arrive asynchronously, so every column but the PK must be nullable.
	_, err = db.Exec(`INSERT INTO host_mdm_android_device_vitals (host_uuid) VALUES (?)`, "other-host-uuid")
	require.NoError(t, err, "a row with only a host_uuid should be accepted")

	// Re-inserting for the same host_uuid should violate the PK.
	_, err = db.Exec(`INSERT INTO host_mdm_android_device_vitals (host_uuid) VALUES (?)`, "android-host-uuid")
	require.Error(t, err, "duplicate host_uuid should be rejected")

	var (
		manufacturer string
		apiLevel     int
		phoneNumber  string
	)
	err = db.QueryRow(`
		SELECT manufacturer, api_level, telephony_infos->>'$[0].phone_number'
		FROM host_mdm_android_device_vitals WHERE host_uuid = ?`,
		"android-host-uuid").Scan(&manufacturer, &apiLevel, &phoneNumber)
	require.NoError(t, err)
	require.Equal(t, "Google", manufacturer)
	require.Equal(t, 36, apiLevel)
	require.Equal(t, "+15555550100", phoneNumber)
}
