package tables

import (
	"bytes"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestUp_20230911163618(t *testing.T) {
	db := applyUpToPrev(t)
	insertStmt := `
INSERT INTO  host_mdm_apple_profiles (
	profile_id,
	profile_identifier,
	host_uuid,
	status,
	operation_type,
	detail,
	command_uuid,
	profile_name,
	checksum)
VALUES
	(?, ?, ?, ?, ?, ?, ?, ?, ?)`

	args := []interface{}{
		1,
		"test-identifier",
		"test-host-uuid",
		fleet.MDMDeliveryVerified,
		fleet.MDMOperationTypeInstall,
		"test-detail",
		"test-command-uuid",
		"test-profile-name",
		[]byte("test-checksum"),
	}
	execNoErr(t, db, insertStmt, args...)

	applyNext(t, db)

	// retrieve the stored value
	var hmap struct {
		ProfileID         uint                     `db:"profile_id"`
		ProfileIdentifier string                   `db:"profile_identifier"`
		HostUUID          string                   `db:"host_uuid"`
		Status            *fleet.MDMDeliveryStatus `db:"status"`
		OperationType     fleet.MDMOperationType   `db:"operation_type"`
		Detail            string                   `db:"detail"`
		CommandUUID       string                   `db:"command_uuid"`
		ProfileName       string                   `db:"profile_name"`
		Checksum          []byte                   `db:"checksum"`
		Retries           uint                     `db:"retries"`
	}

	selectStmt := "SELECT * FROM host_mdm_apple_profiles WHERE host_uuid = ?"
	require.NoError(t, db.Get(&hmap, selectStmt, "test-host-uuid"))
	require.Equal(t, uint(1), hmap.ProfileID)
	require.Equal(t, "test-identifier", hmap.ProfileIdentifier)
	require.Equal(t, "test-host-uuid", hmap.HostUUID)
	require.Equal(t, fleet.MDMDeliveryVerified, *hmap.Status)
	require.Equal(t, fleet.MDMOperationTypeInstall, hmap.OperationType)
	require.Equal(t, "test-detail", hmap.Detail)
	require.Equal(t, "test-command-uuid", hmap.CommandUUID)
	require.Equal(t, "test-profile-name", hmap.ProfileName)
	require.True(t, bytes.HasPrefix(hmap.Checksum, []byte("test-checksum")))
	require.Equal(t, uint(0), hmap.Retries)

	insertStmt = `
INSERT INTO  host_mdm_apple_profiles (
	profile_id,
	profile_identifier,
	host_uuid,
	status,
	operation_type,
	detail,
	command_uuid,
	profile_name,
	checksum,
	retries)
VALUES
	(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	args = []interface{}{
		1,
		"test-identifier",
		"test-host-uuid-2",
		fleet.MDMDeliveryVerified,
		fleet.MDMOperationTypeInstall,
		"test-detail",
		"test-command-uuid-2",
		"test-profile-name",
		[]byte("test-checksum"),
		1,
	}
	execNoErr(t, db, insertStmt, args...)

	require.NoError(t, db.Get(&hmap, selectStmt, "test-host-uuid-2"))
	require.Equal(t, uint(1), hmap.ProfileID)
	require.Equal(t, "test-identifier", hmap.ProfileIdentifier)
	require.Equal(t, "test-host-uuid-2", hmap.HostUUID)
	require.Equal(t, fleet.MDMDeliveryVerified, *hmap.Status)
	require.Equal(t, fleet.MDMOperationTypeInstall, hmap.OperationType)
	require.Equal(t, "test-detail", hmap.Detail)
	require.Equal(t, "test-command-uuid-2", hmap.CommandUUID)
	require.Equal(t, "test-profile-name", hmap.ProfileName)
	require.True(t, bytes.HasPrefix(hmap.Checksum, []byte("test-checksum")))
	require.Equal(t, uint(1), hmap.Retries)
}
