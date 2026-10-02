package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260904202219(t *testing.T) {
	db := applyUpToPrev(t)

	const staleDetail = `"cameraDisabled" setting couldn't apply to a host.`

	insertProfile := func(hostUUID, status, detail string) {
		execNoErr(t, db, `
			INSERT INTO host_mdm_android_profiles
				(host_uuid, profile_uuid, profile_name, status, operation_type, detail)
			VALUES (?, 'a1234567-89ab-cdef-0123-456789abcdef', 'Camera policy', ?, 'install', ?)`,
			hostUUID, status, detail)
	}

	insertProfile("verified-with-stale-detail", "verified", staleDetail)
	insertProfile("verified-already-clean", "verified", "")
	insertProfile("failed-with-real-detail", "failed", staleDetail)
	insertProfile("pending-with-detail", "pending", staleDetail)
	insertProfile("verifying-with-detail", "verifying", staleDetail)

	type profileRow struct {
		HostUUID  string `db:"host_uuid"`
		Detail    string `db:"detail"`
		UpdatedAt string `db:"updated_at"`
	}
	snapshot := func() map[string]profileRow {
		var rows []profileRow
		err := db.Select(&rows, `SELECT host_uuid, detail, updated_at FROM host_mdm_android_profiles`)
		require.NoError(t, err)
		byHost := make(map[string]profileRow, len(rows))
		for _, row := range rows {
			byHost[row.HostUUID] = row
		}
		return byHost
	}
	before := snapshot()

	// Apply current migration.
	applyNext(t, db)

	after := snapshot()

	// The stale detail on a verified profile is what this migration exists to clear.
	require.Empty(t, after["verified-with-stale-detail"].Detail)
	require.Empty(t, after["verified-already-clean"].Detail)

	// A non-verified profile's detail is its current error message, so it has to
	// survive. Clearing these would turn a cosmetic bug into a real one.
	require.Equal(t, staleDetail, after["failed-with-real-detail"].Detail)
	require.Equal(t, staleDetail, after["pending-with-detail"].Detail)
	require.Equal(t, staleDetail, after["verifying-with-detail"].Detail)

	// updated_at is ON UPDATE CURRENT_TIMESTAMP and the migration assigns it to
	// itself, so clearing the detail doesn't make the profile look freshly
	// reported.
	require.Equal(t,
		before["verified-with-stale-detail"].UpdatedAt,
		after["verified-with-stale-detail"].UpdatedAt,
	)
}
