package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260720211014(t *testing.T) {
	db := applyUpToPrev(t)

	// insertProfile adds one host_mdm_windows_profiles row. status is passed as a *string so we can exercise the NULL-as-pending
	// path.
	insertProfile := func(hostUUID, profileUUID, profileName string, opType string, status *string) {
		_, err := db.Exec(`
			INSERT INTO host_mdm_windows_profiles
				(host_uuid, profile_uuid, profile_name, command_uuid, operation_type, status)
			VALUES (?, ?, ?, ?, ?, ?)`,
			hostUUID, profileUUID, profileName, "cmd-"+profileUUID, opType, status)
		require.NoError(t, err)
	}
	statusPtr := func(s string) *string { return &s }

	// host-failed: a failed non-reserved profile wins over everything else.
	insertProfile("host-failed", "p1", "Profile A", "install", statusPtr("failed"))
	insertProfile("host-failed", "p2", "Profile B", "install", statusPtr("verified"))

	// host-pending: a NULL status (equivalent to pending) wins over a verified profile.
	insertProfile("host-pending", "p1", "Profile A", "install", nil)
	insertProfile("host-pending", "p2", "Profile B", "install", statusPtr("verified"))

	// host-verifying: an install verifying wins over an install verified.
	insertProfile("host-verifying", "p1", "Profile A", "install", statusPtr("verifying"))
	insertProfile("host-verifying", "p2", "Profile B", "install", statusPtr("verified"))

	// host-verified: only install verified profiles.
	insertProfile("host-verified", "p1", "Profile A", "install", statusPtr("verified"))
	insertProfile("host-verified", "p2", "Profile B", "install", statusPtr("verified"))

	// host-reserved-only: the only profile is the reserved "Windows OS Updates" one, so it is excluded and the host resolves to the
	// empty status (not counted by the summary).
	insertProfile("host-reserved-only", "p1", "Windows OS Updates", "install", statusPtr("failed"))

	// host-remove-verifying: a remove verifying does not count as verifying (install-only), so with no other non-reserved status the
	// host resolves to the empty status.
	insertProfile("host-remove-verifying", "p1", "Profile A", "remove", statusPtr("verifying"))

	applyNext(t, db)

	want := map[string]string{
		"host-failed":           "failed",
		"host-pending":          "pending",
		"host-verifying":        "verifying",
		"host-verified":         "verified",
		"host-reserved-only":    "",
		"host-remove-verifying": "",
	}

	rows, err := db.Query(`SELECT host_uuid, status FROM host_mdm_windows_profiles_status`)
	require.NoError(t, err)
	defer rows.Close()

	got := map[string]string{}
	for rows.Next() {
		var hostUUID, status string
		require.NoError(t, rows.Scan(&hostUUID, &status))
		got[hostUUID] = status
	}
	require.NoError(t, rows.Err())

	require.Equal(t, want, got)
}
