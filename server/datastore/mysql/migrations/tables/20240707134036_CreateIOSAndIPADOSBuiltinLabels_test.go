package tables

import (
	"fmt"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20240707134036(t *testing.T) {
	db := applyUpToPrev(t)

	// Insert existing hosts before migration.
	hostID := 1
	newHost := func(platform, uuid string) uint {
		id := fmt.Sprintf("%d", hostID)
		hostID++
		return uint(execNoErrLastID(t, db, //nolint:gosec // dismiss G115
			`INSERT INTO hosts (osquery_host_id, node_key, uuid, platform) VALUES (?, ?, ?, ?);`,
			id, id, uuid, platform,
		))
	}
	iOSID := newHost("ios", "iOS_UUID")
	iPadOSID := newHost("ipados", "iPadOS_UUID")
	newHost("darwin", "macOS_UUID")

	// Insert existing profiles and host profiles before migration.
	stmt := `
INSERT INTO
	mdm_apple_configuration_profiles (team_id, identifier, name, mobileconfig, checksum, profile_uuid)
VALUES (?, ?, ?, ?, '', ?)`

	_, err := db.Exec(stmt, 0, "profileID0", "TestPayloadName0", `<?xml version="1.0"`, "profileID0")
	require.NoError(t, err)
	_, err = db.Exec(stmt, 0, "profileID1", "TestPayloadName1", `<?xml version="1.0"`, "profileID1")
	require.NoError(t, err)

	stmt = `
INSERT INTO host_mdm_apple_profiles
	(profile_uuid, profile_identifier, host_uuid, command_uuid, status, operation_type, detail, checksum)
VALUES
	(?, 'com.foo.bar', ?, 'command-uuid', ?, ?, 'detail', '')`

	execNoErr(t, db, stmt, "profileID0", "iOS_UUID", "verifying", "install")
	execNoErr(t, db, stmt, "profileID0", "iPadOS_UUID", "verifying", "install")
	execNoErr(t, db, stmt, "profileID0", "macOS_UUID", "verifying", "install")
	execNoErr(t, db, stmt, "profileID1", "iOS_UUID", "pending", "install")
	execNoErr(t, db, stmt, "profileID1", "iPadOS_UUID", "pending", "install")

	// Apply current migration.
	applyNext(t, db)

	var labelIDs []uint
	err = db.Select(&labelIDs, `SELECT id FROM labels WHERE name = 'iOS' OR name = 'iPadOS';`)
	require.NoError(t, err)
	require.Len(t, labelIDs, 2)
	iOSLabelID := labelIDs[0]
	iPadOSLabelID := labelIDs[1]

	type hostAndLabel struct {
		HostID  uint `db:"host_id"`
		LabelID uint `db:"label_id"`
	}
	var labelMemberships []hostAndLabel
	err = db.Select(&labelMemberships, `SELECT host_id, label_id FROM label_membership;`)
	require.NoError(t, err)
	require.Len(t, labelMemberships, 2)
	sort.Slice(labelMemberships, func(i, j int) bool {
		return labelMemberships[i].HostID < labelMemberships[j].HostID
	})
	require.Equal(t, iOSID, labelMemberships[0].HostID)
	require.Equal(t, iOSLabelID, labelMemberships[0].LabelID)
	require.Equal(t, iPadOSID, labelMemberships[1].HostID)
	require.Equal(t, iPadOSLabelID, labelMemberships[1].LabelID)

	type hostPlusProfile struct {
		HostUUID    string `db:"host_uuid"`
		ProfileUUID string `db:"profile_uuid"`
		Status      string `db:"status"`
	}
	var hostProfiles []hostPlusProfile
	err = db.Select(&hostProfiles, `SELECT host_uuid, profile_uuid, status FROM host_mdm_apple_profiles;`)
	require.NoError(t, err)
	require.Len(t, hostProfiles, 5)
	for _, hostProfile := range hostProfiles {
		switch {
		case hostProfile.HostUUID == "iOS_UUID" && hostProfile.ProfileUUID == "profileID0":
			require.Equal(t, "verified", hostProfile.Status) // should now be verified
		case hostProfile.HostUUID == "iPadOS_UUID" && hostProfile.ProfileUUID == "profileID0":
			require.Equal(t, "verified", hostProfile.Status) // should now be verified
		case hostProfile.HostUUID == "macOS_UUID" && hostProfile.ProfileUUID == "profileID0":
			require.Equal(t, "verifying", hostProfile.Status) // should remain unchanged
		case hostProfile.HostUUID == "iOS_UUID" && hostProfile.ProfileUUID == "profileID1":
			require.Equal(t, "pending", hostProfile.Status) // should remain unchanged because it's pending
		case hostProfile.HostUUID == "iPadOS_UUID" && hostProfile.ProfileUUID == "profileID1":
			require.Equal(t, "pending", hostProfile.Status) // should remain unchanged because it's pending
		}
	}
}
