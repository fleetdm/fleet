package tables

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUp_20260727083533(t *testing.T) {
	db := applyUpToPrev(t)

	// Apply current migration.
	applyNext(t, db)

	// apple_software_update_assets: insert a row, uniqueness on
	// (class, product_version, build).
	assetID := execNoErrLastID(t, db, `
		INSERT INTO apple_software_update_assets
			(class, product_version, build, posting_date, expiration_date, supported_devices)
		VALUES
			('macos', '15.1', '24B83', '2026-01-01', NULL, '["J123AP"]')`)
	require.NotZero(t, assetID)

	// expiration_date is nullable; first_seen_at and updated_at are populated by
	// their defaults.
	var (
		gotExpirationDate *time.Time
		gotFirstSeenAt    time.Time
		gotUpdatedAt      time.Time
	)
	err := db.QueryRow(`
		SELECT expiration_date, first_seen_at, updated_at
		FROM apple_software_update_assets WHERE id = ?`, assetID,
	).Scan(&gotExpirationDate, &gotFirstSeenAt, &gotUpdatedAt)
	require.NoError(t, err)
	require.Nil(t, gotExpirationDate)
	require.False(t, gotFirstSeenAt.IsZero())
	require.False(t, gotUpdatedAt.IsZero())

	_, err = db.Exec(`
		INSERT INTO apple_software_update_assets
			(class, product_version, build, supported_devices)
		VALUES
			('macos', '15.1', '24B83', '["J123AP"]')`)
	require.Error(t, err, "duplicate (class, product_version, build) should be rejected")

	// An upsert on the same (class, product_version, build) — the shape the GDMF
	// refresh uses — keeps first_seen_at from the original insert while
	// updated_at advances. posting_date is changed so the row is a real update:
	// MySQL leaves updated_at alone when no column value actually changes.
	execNoErr(t, db, `
		INSERT INTO apple_software_update_assets
			(class, product_version, build, posting_date, supported_devices)
		VALUES
			('macos', '15.1', '24B83', '2026-01-02', '["J123AP","J456AP"]')
		ON DUPLICATE KEY UPDATE
			posting_date = VALUES(posting_date),
			supported_devices = VALUES(supported_devices)`)

	var (
		gotFirstSeenAtAfterUpsert time.Time
		gotUpdatedAtAfterUpsert   time.Time
	)
	err = db.QueryRow(`
		SELECT first_seen_at, updated_at
		FROM apple_software_update_assets WHERE id = ?`, assetID,
	).Scan(&gotFirstSeenAtAfterUpsert, &gotUpdatedAtAfterUpsert)
	require.NoError(t, err)
	require.True(t, gotFirstSeenAt.Equal(gotFirstSeenAtAfterUpsert),
		"first_seen_at must not change on upsert")
	require.True(t, gotUpdatedAtAfterUpsert.After(gotUpdatedAt),
		"updated_at must advance on upsert")

	// A different build for the same class/version is allowed.
	execNoErr(t, db, `
		INSERT INTO apple_software_update_assets
			(class, product_version, build, supported_devices)
		VALUES
			('macos', '15.1', '24B84', '["J123AP"]')`)

	// supported_devices is NOT NULL.
	_, err = db.Exec(`
		INSERT INTO apple_software_update_assets
			(class, product_version, build, supported_devices)
		VALUES
			('ios', '18.1', '', NULL)`)
	require.Error(t, err, "NULL supported_devices should be rejected")

	// Invalid class tvos is rejected by the ENUM.
	_, err = db.Exec(`
		INSERT INTO apple_software_update_assets
			(class, product_version, build, supported_devices)
		VALUES
			('tvos', '18.1', '22J1', '["J123AP"]')`)
	require.Error(t, err, "class outside the enum should be rejected")

	// build defaults to '' when omitted.
	iosAssetID := execNoErrLastID(t, db, `
		INSERT INTO apple_software_update_assets
			(class, product_version, supported_devices)
		VALUES
			('ios', '18.1', '["iPhone16,1"]')`)

	var gotBuild string
	err = db.QueryRow(`
		SELECT build FROM apple_software_update_assets WHERE id = ?`, iosAssetID,
	).Scan(&gotBuild)
	require.NoError(t, err)
	require.Empty(t, gotBuild)

	// host_mdm_apple_os_updates: keyed by host_uuid, defaults apply.
	execNoErr(t, db, `
		INSERT INTO host_mdm_apple_os_updates (host_uuid)
		VALUES ('host-1-uuid')`)

	// Scanning the two string columns into non-pointers also asserts they
	// default to '' rather than NULL; target_deadline and resolved_at are
	// nullable until the target is resolved for the host.
	var (
		gotTargetOSVersion        string
		gotSoftwareUpdateDeviceID string
		gotTargetDeadline         *time.Time
		gotResolvedAt             *time.Time
		gotCreatedAt              time.Time
	)
	err = db.QueryRow(`
		SELECT target_os_version, software_update_device_id, target_deadline,
			resolved_at, created_at
		FROM host_mdm_apple_os_updates WHERE host_uuid = 'host-1-uuid'`,
	).Scan(&gotTargetOSVersion, &gotSoftwareUpdateDeviceID, &gotTargetDeadline, &gotResolvedAt, &gotCreatedAt)
	require.NoError(t, err)
	require.Empty(t, gotTargetOSVersion)
	require.Empty(t, gotSoftwareUpdateDeviceID)
	require.Nil(t, gotTargetDeadline)
	require.Nil(t, gotResolvedAt)
	require.False(t, gotCreatedAt.IsZero())

	_, err = db.Exec(`
		INSERT INTO host_mdm_apple_os_updates (host_uuid)
		VALUES ('host-1-uuid')`)
	require.Error(t, err, "duplicate host_uuid should be rejected")
}
