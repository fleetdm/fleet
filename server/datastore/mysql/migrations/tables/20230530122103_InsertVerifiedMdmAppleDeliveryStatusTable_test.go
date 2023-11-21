package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20230530122103(t *testing.T) {
	db := applyUpToPrev(t)

	var statuses []string
	err := db.Select(&statuses, "SELECT status FROM mdm_apple_delivery_status")
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"failed", "verifying", "pending"}, statuses)

	// Insert some data.
	stmt := `
INSERT INTO host_mdm_apple_profiles (
	profile_id,
	profile_identifier,
	profile_name,
	host_uuid,
	status,
	operation_type,
	command_uuid,
	checksum
)
VALUES (?,?,?,?,?,?,?,?)`

	_, err = db.Exec(stmt, 1, "com.example.test", "Test Profile", "huuid1", "verifying", "install", "cuuid1", "csum1")
	require.NoError(t, err)

	_, err = db.Exec(stmt, 2, "com.example.test", "Test Profile 2", "huuid1", "pending", "install", "cuuid2", "csum2")
	require.NoError(t, err)

	_, err = db.Exec(stmt, 3, "com.example.test", "Test Profile 3", "huuid2", "failed", "install", "cuuid3", "csum3")
	require.NoError(t, err)

	// Apply current migration.
	applyNext(t, db)

	// Check that the data was updated.
	statuses = []string{}
	err = db.Select(&statuses, "SELECT status FROM mdm_apple_delivery_status")
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"failed", "verifying", "pending", "verified"}, statuses)

	// Check that existing data was unchanged.
	var status string
	err = db.QueryRow("SELECT status FROM host_mdm_apple_profiles WHERE profile_id = 1").Scan(&status)
	require.NoError(t, err)
	require.Equal(t, "verifying", status)

	err = db.QueryRow("SELECT status FROM host_mdm_apple_profiles WHERE profile_id = 2").Scan(&status)
	require.NoError(t, err)
	require.Equal(t, "pending", status)

	err = db.QueryRow("SELECT status FROM host_mdm_apple_profiles WHERE profile_id = 3").Scan(&status)
	require.NoError(t, err)
	require.Equal(t, "failed", status)

	// Insert some data with the new verified status.
	_, err = db.Exec(stmt, 4, "com.example.test", "Test Profile 4", "huuid4", "verified", "install", "cuuid4", "csum4")
	require.NoError(t, err)
}
