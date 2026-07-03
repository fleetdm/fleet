package tables

import (
	"crypto/md5" //nolint:gosec // checksum for comparison, not security
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260703114904(t *testing.T) {
	db := applyUpToPrev(t)

	// Seed an in-flight deletion in the old pending-delete table; it must carry over to the new table.
	syncML := []byte(`<Replace><Item><Target><LocURI>./Device/Foo</LocURI></Target></Item></Replace>`)
	execNoErr(t, db, `INSERT INTO mdm_windows_configuration_profiles_pending_delete (profile_uuid, team_id, name, syncml) VALUES (?, ?, ?, ?)`,
		"w-deleted", 0, "deleted-prof", syncML)

	applyNext(t, db)

	// The old table is gone.
	var exists int
	require.NoError(t, db.Get(&exists,
		`SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = 'mdm_windows_configuration_profiles_pending_delete'`))
	require.Equal(t, 0, exists)

	// The retained content carried over, keyed by md5(syncml).
	wantChecksum := md5.Sum(syncML) //nolint:gosec // checksum for comparison, not security
	var got struct {
		ProfileUUID string `db:"profile_uuid"`
		Checksum    []byte `db:"checksum"`
		SyncML      []byte `db:"syncml"`
	}
	require.NoError(t, db.Get(&got,
		`SELECT profile_uuid, checksum, syncml FROM mdm_windows_configuration_profiles_prior_content WHERE profile_uuid = ?`, "w-deleted"))
	require.Equal(t, "w-deleted", got.ProfileUUID)
	require.Equal(t, wantChecksum[:], got.Checksum)
	require.Equal(t, syncML, got.SyncML)

	// (profile_uuid, checksum) is the primary key: the same profile with a different version checksum is a distinct row.
	otherChecksum := make([]byte, 16)
	otherChecksum[0] = 0x02
	execNoErr(t, db, `INSERT INTO mdm_windows_configuration_profiles_prior_content (profile_uuid, checksum, syncml) VALUES (?, ?, ?)`,
		"w-deleted", otherChecksum, []byte("<other/>"))
	var count int
	require.NoError(t, db.Get(&count, `SELECT COUNT(*) FROM mdm_windows_configuration_profiles_prior_content WHERE profile_uuid = ?`, "w-deleted"))
	require.Equal(t, 2, count)

	// Re-inserting the same (profile_uuid, checksum) violates the primary key (the code path uses INSERT ... ON DUPLICATE KEY UPDATE).
	_, err := db.Exec(`INSERT INTO mdm_windows_configuration_profiles_prior_content (profile_uuid, checksum, syncml) VALUES (?, ?, ?)`,
		"w-deleted", otherChecksum, []byte("<other/>"))
	require.Error(t, err)
}
