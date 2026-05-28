package tables

import (
	"crypto/md5" // nolint:gosec // used only to hash for efficient comparisons
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUp_20260528211618(t *testing.T) {
	db := applyUpToPrev(t)

	rawJSON := `{"openNetworkConfiguration":{"Type":"WiFi"}}`
	profileUUID := "g_test-profile-1"

	// Insert a config profile
	_, err := db.Exec(`
		INSERT INTO mdm_android_configuration_profiles (profile_uuid, team_id, name, raw_json)
		VALUES (?, 0, 'TestProfile', ?)`, profileUUID, rawJSON)
	require.NoError(t, err)

	// Insert a host profile referencing the config profile
	_, err = db.Exec(`
		INSERT INTO host_mdm_android_profiles (host_uuid, status, operation_type, profile_uuid, profile_name)
		VALUES (?, 'verified', 'install', ?, 'TestProfile')`, "host-uuid-1", profileUUID)
	require.NoError(t, err)

	// Insert a host profile with a missing config profile (orphan)
	_, err = db.Exec(`
		INSERT INTO host_mdm_android_profiles (host_uuid, status, operation_type, profile_uuid, profile_name)
		VALUES (?, 'pending', 'install', ?, 'MissingProfile')`, "host-uuid-2", "g_missing")
	require.NoError(t, err)

	// Apply migration
	applyNext(t, db)

	// Verify config profile checksum matches MD5 of raw_json
	var checksum []byte
	err = db.QueryRow(`SELECT checksum FROM mdm_android_configuration_profiles WHERE profile_uuid = ?`, profileUUID).Scan(&checksum)
	require.NoError(t, err)
	// MySQL's CAST(json AS CHAR) normalizes JSON, so compute expected checksum from what MySQL stores
	var storedJSON string
	err = db.QueryRow(`SELECT CAST(raw_json AS CHAR) FROM mdm_android_configuration_profiles WHERE profile_uuid = ?`, profileUUID).Scan(&storedJSON)
	require.NoError(t, err)
	expectedChecksum := md5.Sum([]byte(storedJSON)) // nolint:gosec // used only to hash for efficient comparisons
	assert.Equal(t, fmt.Sprintf("%x", expectedChecksum), fmt.Sprintf("%x", checksum))

	// Verify host profile checksum was backfilled from config profile
	err = db.QueryRow(`SELECT checksum FROM host_mdm_android_profiles WHERE host_uuid = ?`, "host-uuid-1").Scan(&checksum)
	require.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("%x", expectedChecksum), fmt.Sprintf("%x", checksum))

	// Verify orphan host profile got default checksum (COALESCE returns 0,
	// which MySQL stores as 0x30 followed by zeros in BINARY(16))
	var orphanChecksum []byte
	err = db.QueryRow(`SELECT checksum FROM host_mdm_android_profiles WHERE host_uuid = ?`, "host-uuid-2").Scan(&orphanChecksum)
	require.NoError(t, err)
	// Orphan checksum should NOT equal the real profile's checksum
	assert.NotEqual(t, fmt.Sprintf("%x", expectedChecksum), fmt.Sprintf("%x", orphanChecksum))
}
