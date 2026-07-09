package tables

import (
	"crypto/sha256"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUp_20260707153947(t *testing.T) {
	db := applyUpToPrev(t)

	// ---- Set up test data using the pre-migration schema (MD5/BINARY(16)) ----

	// Apple config profile
	appleProfileUUID := "a_test-apple-1"
	mobileconfig := []byte(`<?xml version="1.0"?><plist><dict><key>test</key></dict></plist>`)
	_, err := db.Exec(`
		INSERT INTO mdm_apple_configuration_profiles
			(profile_uuid, team_id, identifier, name, scope, mobileconfig, checksum, uploaded_at)
		VALUES (?, 0, 'com.test', 'TestApple', 'system', ?, UNHEX(SHA2(?, 256)), CURRENT_TIMESTAMP(6))`,
		appleProfileUUID, mobileconfig, mobileconfig)
	require.NoError(t, err)

	// Host Apple profile with matching checksum (current version)
	_, err = db.Exec(`
		INSERT INTO host_mdm_apple_profiles
			(host_uuid, profile_uuid, profile_identifier, profile_name, status, operation_type, checksum)
		VALUES ('host-1', ?, 'com.test', 'TestApple', 'verified', 'install',
			(SELECT checksum FROM mdm_apple_configuration_profiles WHERE profile_uuid = ?))`,
		appleProfileUUID, appleProfileUUID)
	require.NoError(t, err)

	// Policy
	_, err = db.Exec(`
		INSERT INTO policies (name, query, description, checksum)
		VALUES ('test-policy', 'SELECT 1', 'test', UNHEX(SHA2(CONCAT_WS(CHAR(0), '', 'test-policy'), 256)))`)
	require.NoError(t, err)

	// Software
	_, err = db.Exec(`
		INSERT INTO software (name, version, source, checksum)
		VALUES ('test-sw', '1.0', 'test', UNHEX(SHA2(CONCAT_WS(CHAR(0), '1.0', 'test', '', '', '', '', '', '', 'test-sw'), 256)))`)
	require.NoError(t, err)

	// ---- Apply migration ----
	applyNext(t, db)

	// ---- Verify checksums are 32 bytes (SHA2-256) ----

	// Apple profile checksum
	var appleChecksum []byte
	err = db.QueryRow(`SELECT checksum FROM mdm_apple_configuration_profiles WHERE profile_uuid = ?`, appleProfileUUID).Scan(&appleChecksum)
	require.NoError(t, err)
	assert.Len(t, appleChecksum, 32, "Apple profile checksum should be 32 bytes (SHA2-256)")

	expectedApple := sha256.Sum256(mobileconfig)
	assert.Equal(t, fmt.Sprintf("%x", expectedApple[:]), fmt.Sprintf("%x", appleChecksum))

	// Host Apple profile should have matching checksum
	var hostAppleChecksum []byte
	err = db.QueryRow(`SELECT checksum FROM host_mdm_apple_profiles WHERE host_uuid = 'host-1'`).Scan(&hostAppleChecksum)
	require.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("%x", expectedApple[:]), fmt.Sprintf("%x", hostAppleChecksum),
		"Host profile checksum should match recalculated profile checksum")

	// Policy checksum
	var policyChecksum []byte
	err = db.QueryRow(`SELECT checksum FROM policies WHERE name = 'test-policy'`).Scan(&policyChecksum)
	require.NoError(t, err)
	assert.Len(t, policyChecksum, 32, "Policy checksum should be 32 bytes (SHA2-256)")

	// Software checksum
	var swChecksum []byte
	err = db.QueryRow(`SELECT checksum FROM software WHERE name = 'test-sw'`).Scan(&swChecksum)
	require.NoError(t, err)
	assert.Len(t, swChecksum, 32, "Software checksum should be 32 bytes (SHA2-256)")
}
