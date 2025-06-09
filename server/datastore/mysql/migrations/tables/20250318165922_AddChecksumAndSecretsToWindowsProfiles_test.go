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

	_, err = db.Exec(`
	INSERT INTO
		host_mdm_windows_profiles (host_uuid, status, operation_type, profile_uuid, command_uuid)
		VALUES (?, ?, ?, ?, ?)`, "uuid", "verifying", "install", "missing", "c2")
	require.NoError(t, err)

	// apply migration
	applyNext(t, db)

	var checksum []byte
	err = db.QueryRow(`SELECT checksum FROM mdm_windows_configuration_profiles WHERE profile_uuid = ?`, "w1").Scan(&checksum)
	require.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("%x", md5.Sum([]byte(syncml))), // nolint:gosec // used only to hash for efficient comparisons
		fmt.Sprintf("%x", checksum)) // nolint:gosec // used only to hash for efficient comparisons

	err = db.QueryRow(`SELECT checksum FROM host_mdm_windows_profiles WHERE profile_uuid = ?`, "w1").Scan(&checksum)
	require.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("%x", md5.Sum([]byte(syncml))), // nolint:gosec // used only to hash for efficient comparisons
		fmt.Sprintf("%x", checksum)) // nolint:gosec // used only to hash for efficient comparisons
}
