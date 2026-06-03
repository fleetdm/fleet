package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260528213326(t *testing.T) {
	db := applyUpToPrev(t)

	rawJSON := `{"openNetworkConfiguration":{"Type":"WiFi"}}`
	profileUUID := "g_test-profile-1"

	// Insert a config profile before the checksum column exists.
	_, err := db.Exec(`
		INSERT INTO mdm_android_configuration_profiles (profile_uuid, team_id, name, raw_json)
		VALUES (?, 0, 'TestProfile', ?)`, profileUUID, rawJSON)
	require.NoError(t, err)

	// Apply migration.
	applyNext(t, db)

	// The checksum column is added as a plain BINARY(16) with no backfill (the
	// original generated column auto-populated released deployments; a fresh install
	// replays this against an empty table). So a row that predates the column has a
	// NULL checksum until the application writes it (or a later migration recomputes
	// it).
	var checksum []byte
	require.NoError(t, db.QueryRow(`SELECT checksum FROM mdm_android_configuration_profiles WHERE profile_uuid = ?`, profileUUID).Scan(&checksum))
	require.Nil(t, checksum)

	// The column is plain (writable) — this would fail on a generated column.
	_, err = db.Exec(`UPDATE mdm_android_configuration_profiles SET checksum = ? WHERE profile_uuid = ?`, []byte("0123456789abcdef"), profileUUID)
	require.NoError(t, err)
}
