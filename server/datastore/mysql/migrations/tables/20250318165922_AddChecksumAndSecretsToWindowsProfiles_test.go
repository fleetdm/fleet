package tables

import (
	"crypto/md5" // nolint:gosec // used only to hash for efficient comparisons
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUp_20250318165922(t *testing.T) {
	db := applyUpToPrev(t)

	syncml := `<Replace></Replace>`
	_, err := db.Exec(`
	INSERT INTO
		mdm_windows_configuration_profiles (name, syncml, profile_uuid)
		VALUES (?, ?, ?)`, "name", syncml, "w1")
	require.NoError(t, err)

	_, err = db.Exec(`
	INSERT INTO
		host_mdm_windows_profiles (host_uuid, status, operation_type, profile_uuid, command_uuid)
		VALUES (?, ?, ?, ?, ?)`, "uuid", "verifying", "install", "w1", "c1")
	require.NoError(t, err)

	// apply migration
	applyNext(t, db)

	// The checksum column is added as a plain BINARY(16). It is no longer a STORED
	// generated column (MySQL 9.6/9.7 removed the MD5() function), so the migration
	// does not auto-compute it; the application computes and writes it in Go at the
	// next profile write. The pre-existing row therefore has a NULL checksum.
	var checksum []byte
	err = db.QueryRow(`SELECT checksum FROM mdm_windows_configuration_profiles WHERE profile_uuid = ?`, "w1").Scan(&checksum)
	require.NoError(t, err)
	require.Nil(t, checksum)

	// The column must now be plain (writable) — this would fail on a generated column.
	want := md5.Sum([]byte(syncml)) // nolint:gosec
	_, err = db.Exec(`UPDATE mdm_windows_configuration_profiles SET checksum = ? WHERE profile_uuid = ?`, want[:], "w1")
	require.NoError(t, err)
	err = db.QueryRow(`SELECT checksum FROM mdm_windows_configuration_profiles WHERE profile_uuid = ?`, "w1").Scan(&checksum)
	require.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("%x", want), fmt.Sprintf("%x", checksum))
}
