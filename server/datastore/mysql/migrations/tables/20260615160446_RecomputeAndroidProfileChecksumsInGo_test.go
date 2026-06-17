package tables

import (
	"crypto/md5" // nolint:gosec // used only to hash for efficient comparisons
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260615160446(t *testing.T) {
	db := applyUpToPrev(t)

	rawJSON := `{"b":1,"a":2,"openNetworkConfiguration":{"Type":"WiFi"}}`
	profileUUID := "g_recompute-1"
	oldChecksum := []byte("OLDoldOLDoldOLD1") // 16 bytes; simulates the old md5(normalized) basis

	// Seed a profile + host profile in the "old basis" state, where the host's copy
	// already matches the desired checksum (an unchanged, delivered profile).
	_, err := db.Exec(`INSERT INTO mdm_android_configuration_profiles (profile_uuid, team_id, name, raw_json, checksum) VALUES (?, 0, 'P', ?, ?)`,
		profileUUID, rawJSON, oldChecksum)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO host_mdm_android_profiles (host_uuid, status, operation_type, profile_uuid, profile_name, checksum) VALUES (?, 'verified', 'install', ?, 'P', ?)`,
		"host-1", profileUUID, oldChecksum)
	require.NoError(t, err)

	// Orphan host row (no matching profile) — must be left untouched.
	orphanChecksum := []byte("ORPHANorphan0001")
	_, err = db.Exec(`INSERT INTO host_mdm_android_profiles (host_uuid, status, operation_type, profile_uuid, profile_name, checksum) VALUES (?, 'pending', 'install', ?, 'Missing', ?)`,
		"host-2", "g_missing", orphanChecksum)
	require.NoError(t, err)

	applyNext(t, db)

	// Expected = md5 of the canonical form of the stored raw_json (what the runtime
	// write path also produces for the same content).
	var storedRaw []byte
	require.NoError(t, db.QueryRow(`SELECT raw_json FROM mdm_android_configuration_profiles WHERE profile_uuid = ?`, profileUUID).Scan(&storedRaw))
	canonical, err := canonicalizeAndroidProfileJSON20260615160446(storedRaw)
	require.NoError(t, err)
	expected := md5.Sum(canonical) // nolint:gosec

	var profChecksum, hostChecksum, gotOrphan []byte
	require.NoError(t, db.QueryRow(`SELECT checksum FROM mdm_android_configuration_profiles WHERE profile_uuid = ?`, profileUUID).Scan(&profChecksum))
	require.NoError(t, db.QueryRow(`SELECT checksum FROM host_mdm_android_profiles WHERE host_uuid = ?`, "host-1").Scan(&hostChecksum))
	require.NoError(t, db.QueryRow(`SELECT checksum FROM host_mdm_android_profiles WHERE host_uuid = ?`, "host-2").Scan(&gotOrphan))

	require.Equal(t, expected[:], profChecksum, "desired checksum recomputed to canonical md5")
	require.Equal(t, expected[:], hostChecksum, "host checksum re-pointed to desired (no re-delivery)")
	require.NotEqual(t, oldChecksum, profChecksum, "checksum basis changed from the old value")
	require.Equal(t, orphanChecksum, gotOrphan, "orphan host checksum left untouched")
}
