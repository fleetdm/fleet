package tables

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestUp_20230313135301(t *testing.T) {
	db := applyUpToPrev(t)

	stmt := `
INSERT INTO
	mdm_apple_configuration_profiles (team_id, identifier, name, mobileconfig)
VALUES (?, ?, ?, ?)`

	r, err := db.Exec(stmt, 0, "TestPayloadIdentifier", "TestPayloadName", `<?xml version="1.0"`)
	require.NoError(t, err)
	profileID, _ := r.LastInsertId()

	stmt = `
INSERT INTO host_mdm_apple_profiles
	(profile_id, profile_identifier, host_uuid, command_uuid, status, operation_type, detail)
VALUES
	(?, 'com.foo.bar', ?, 'command-uuid', ?, ?, ?)`

	execNoErr(t, db, stmt, profileID, "ABC", "pending", "install", "")
	execNoErr(t, db, stmt, profileID, "DEF", "failed", "remove", "error message")

	// Apply current migration.
	applyNext(t, db)

	// Okay if we don't provide name
	execNoErr(t, db, stmt, profileID, "GHI", "pending", "install", "")

	// Insert with name
	stmt = `
INSERT INTO host_mdm_apple_profiles
	(profile_id, profile_identifier, host_uuid, command_uuid, status, operation_type, detail, profile_name)
VALUES
	(?, 'com.foo.bar', ?, 'command-uuid', ?, ?, ?, ?)`

	execNoErr(t, db, stmt, profileID, "JKL", "pending", "install", "", "TestPayloadName")

	var rows []fleet.HostMDMAppleProfile
	err = db.SelectContext(context.Background(), &rows, `
SELECT
	profile_id,
	profile_identifier AS identifier,
	host_uuid,
	command_uuid,
	status,
	operation_type,
	detail,
	profile_name AS name
FROM
	host_mdm_apple_profiles`)

	require.NoError(t, err)
	require.Len(t, rows, 4)

	for _, r := range rows {
		if r.HostUUID == "JKL" {
			require.Equal(t, "TestPayloadName", r.Name)
		} else {
			require.Equal(t, "", r.Name)
		}
	}
}
