package tables

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestUp_20230317173844(t *testing.T) {
	db := applyUpToPrev(t)

	insertStmt := `
INSERT INTO host_mdm_apple_profiles
	(profile_id, profile_identifier, host_uuid, command_uuid, status, operation_type, detail)
VALUES
	(?, 'com.foo.bar', ?, 'command-uuid', ?, ?, ?)`

	execNoErr(t, db, insertStmt, 1, "ABC", "pending", "install", "")
	execNoErr(t, db, insertStmt, 1, "DEF", "failed", "remove", "MDMClientError (89): Profile with identifier 'com.foo.bar' not found.")
	execNoErr(t, db, insertStmt, 1, "GHI", "pending", "remove", "")
	execNoErr(t, db, insertStmt, 1, "JKL", "failed", "remove", "MDMClientError (96): Cannot replace profile 'p2' because it was not installed by the MDM server.")

	selectStmt := `
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
	host_mdm_apple_profiles`

	var rows []fleet.HostMDMAppleProfile
	require.NoError(t, db.SelectContext(context.Background(), &rows, selectStmt))
	require.Len(t, rows, 4)

	// Apply current migration.
	applyNext(t, db)

	rows = nil
	require.NoError(t, db.SelectContext(context.Background(), &rows, selectStmt))
	require.Len(t, rows, 3)

	for _, r := range rows {
		require.NotContains(t, r.Detail, "MDMClientError (89)")
	}
}
