package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260727083533(t *testing.T) {
	db := applyUpToPrev(t)

	// Apply current migration.
	applyNext(t, db)

	// apple_software_update_assets: insert a row, uniqueness on
	// (class, product_version, build).
	res, err := db.Exec(`
		INSERT INTO apple_software_update_assets
			(class, product_version, build, posting_date, expiration_date, supported_devices)
		VALUES
			('macos', '15.1', '24B83', '2026-01-01', NULL, '["J123AP"]')`)
	require.NoError(t, err)
	assetIDInt, err := res.LastInsertId()
	require.NoError(t, err)
	assetID := uint(assetIDInt) //nolint:gosec
	require.NotZero(t, assetID)

	_, err = db.Exec(`
		INSERT INTO apple_software_update_assets
			(class, product_version, build, supported_devices)
		VALUES
			('macos', '15.1', '24B83', '["J123AP"]')`)
	require.Error(t, err, "duplicate (class, product_version, build) should be rejected")

	// A different build for the same class/version is allowed.
	_, err = db.Exec(`
		INSERT INTO apple_software_update_assets
			(class, product_version, build, supported_devices)
		VALUES
			('macos', '15.1', '24B84', '["J123AP"]')`)
	require.NoError(t, err)

	// supported_devices is NOT NULL.
	_, err = db.Exec(`
		INSERT INTO apple_software_update_assets
			(class, product_version, build, supported_devices)
		VALUES
			('ios', '18.1', '', NULL)`)
	require.Error(t, err, "NULL supported_devices should be rejected")

	// host_mdm_apple_os_updates: keyed by host_uuid, defaults apply.
	_, err = db.Exec(`
		INSERT INTO host_mdm_apple_os_updates (host_uuid)
		VALUES ('host-1-uuid')`)
	require.NoError(t, err)

	var targetOSVersion, softwareUpdateDeviceID string
	err = db.QueryRow(`
		SELECT target_os_version, software_update_device_id
		FROM host_mdm_apple_os_updates WHERE host_uuid = 'host-1-uuid'`,
	).Scan(&targetOSVersion, &softwareUpdateDeviceID)
	require.NoError(t, err)
	require.Empty(t, targetOSVersion)
	require.Empty(t, softwareUpdateDeviceID)

	_, err = db.Exec(`
		INSERT INTO host_mdm_apple_os_updates (host_uuid)
		VALUES ('host-1-uuid')`)
	require.Error(t, err, "duplicate host_uuid should be rejected")
}
