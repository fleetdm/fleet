package tables

import (
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUp_20260610172952(t *testing.T) {
	db := applyUpToPrev(t)

	columnExists := func(t *testing.T, db *sqlx.DB, table, column string) bool {
		t.Helper()
		var count int
		require.NoError(t, db.Get(&count, `
			SELECT COUNT(*) FROM information_schema.columns
			WHERE table_schema = DATABASE() AND table_name = ? AND column_name = ?`,
			table, column,
		))
		return count > 0
	}

	insertProfile := func(t *testing.T, mobileconfig string) string {
		t.Helper()
		profUUID := "a" + uuid.NewString()
		_, err := db.Exec(`
			INSERT INTO mdm_apple_configuration_profiles
				(profile_uuid, team_id, identifier, name, mobileconfig, checksum, uploaded_at)
			VALUES (?, 0, ?, ?, ?, ?, NOW())`,
			profUUID, "id-"+profUUID, "name-"+profUUID, mobileconfig, []byte("0123456789abcdef"))
		require.NoError(t, err)
		return profUUID
	}

	insertHostProfile := func(t *testing.T, hostUUID, profUUID string) {
		t.Helper()
		_, err := db.Exec(`
			INSERT INTO host_mdm_apple_profiles
				(host_uuid, profile_uuid, profile_identifier, command_uuid, checksum, operation_type, status)
			VALUES (?, ?, ?, ?, ?, 'install', 'verified')`,
			hostUUID, profUUID, "id-"+profUUID, "cmd-"+profUUID, []byte("0123456789abcdef"))
		require.NoError(t, err)
	}

	acmeProf := insertProfile(t, `<plist><dict><key>PayloadType</key><string>com.apple.security.acme</string></dict></plist>`)
	scepProf := insertProfile(t, `<plist><dict><key>PayloadType</key><string>com.apple.security.scep</string></dict></plist>`)
	insertHostProfile(t, "host-acme", acmeProf)
	insertHostProfile(t, "host-scep", scepProf)

	require.False(t, columnExists(t, db, "host_mdm_apple_profiles", "has_acme_payload"))

	applyNext(t, db)

	require.True(t, columnExists(t, db, "host_mdm_apple_profiles", "has_acme_payload"))

	flag := func(hostUUID, profUUID string) bool {
		var v bool
		require.NoError(t, db.Get(&v, `
			SELECT has_acme_payload FROM host_mdm_apple_profiles WHERE host_uuid = ? AND profile_uuid = ?`,
			hostUUID, profUUID))
		return v
	}
	assert.True(t, flag("host-acme", acmeProf), "ACME profile row should be backfilled to 1")
	assert.False(t, flag("host-scep", scepProf), "non-ACME profile row should stay 0")
}
