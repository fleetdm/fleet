package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20261123143955(t *testing.T) {
	db := applyUpToPrev(t)

	const fv = "com.fleetdm.fleet.mdm.filevault"
	// status is passed as a *string so we can exercise the NULL-as-pending path.
	insertProfile := func(hostUUID, profileUUID, identifier, opType string, status *string) {
		execNoErr(t, db, `
			INSERT INTO host_mdm_apple_profiles
				(host_uuid, profile_uuid, profile_identifier, command_uuid, checksum, operation_type, status)
			VALUES (?, ?, ?, ?, UNHEX(MD5(?)), ?, ?)`,
			hostUUID, profileUUID, identifier, "cmd-"+hostUUID+profileUUID, profileUUID, opType, status)
	}
	insertDecl := func(hostUUID, declUUID, name, opType string, status *string) {
		execNoErr(t, db, `
			INSERT INTO host_mdm_apple_declarations
				(host_uuid, declaration_uuid, declaration_identifier, declaration_name, token, operation_type, status)
			VALUES (?, ?, ?, ?, UNHEX(MD5(?)), ?, ?)`,
			hostUUID, declUUID, "id-"+declUUID, name, hostUUID+declUUID, opType, status)
	}
	s := func(v string) *string { return &v }

	// a failed profile wins over everything else; the FileVault profile is tracked separately
	insertProfile("host-failed", "p1", "com.a", "install", s("failed"))
	insertProfile("host-failed", "p2", "com.b", "install", s("verified"))
	insertProfile("host-failed", "p3", fv, "install", s("verified"))
	// a NULL status (equivalent to pending) wins over a verified profile
	insertProfile("host-pending", "p1", "com.a", "install", nil)
	insertProfile("host-pending", "p2", "com.b", "install", s("verified"))
	insertProfile("host-pending", "p3", fv, "install", s("verifying"))
	// an install verifying wins over an install verified
	insertProfile("host-verifying", "p1", "com.a", "install", s("verifying"))
	insertProfile("host-verifying", "p2", "com.b", "install", s("verified"))
	insertProfile("host-verifying", "p3", fv, "install", nil)
	insertProfile("host-verified", "p1", "com.a", "install", s("verified"))
	// only the FileVault profile, so the other-profiles bucket is empty
	insertProfile("host-fv-only", "p3", fv, "install", s("failed"))
	// remove operations only count when failed or pending
	insertProfile("host-remove-verifying", "p1", "com.a", "remove", s("verifying"))

	insertDecl("host-failed", "d1", "Decl A", "install", s("failed"))
	insertDecl("host-failed", "d2", "Decl B", "install", s("verified"))
	insertDecl("host-pending", "d1", "Decl A", "install", s("pending"))
	insertDecl("host-verified", "d1", "Decl A", "install", s("verified"))
	// reserved OS update declarations are excluded
	insertDecl("host-reserved-only", "d1", "Fleet macOS OS Updates", "install", s("failed"))
	insertDecl("host-remove-verified", "d1", "Decl A", "remove", s("verified"))

	applyNext(t, db)

	type profStatus struct{ Status, FileVault string }
	gotProf := map[string]profStatus{}
	rows, err := db.Query(`SELECT host_uuid, status, filevault_status FROM host_mdm_apple_profiles_status`)
	require.NoError(t, err)
	for rows.Next() {
		var host string
		var ps profStatus
		require.NoError(t, rows.Scan(&host, &ps.Status, &ps.FileVault))
		gotProf[host] = ps
	}
	require.NoError(t, rows.Err())
	defer rows.Close()
	require.Equal(t, map[string]profStatus{
		"host-failed":           {"failed", "verified"},
		"host-pending":          {"pending", "verifying"},
		"host-verifying":        {"verifying", "pending"},
		"host-verified":         {"verified", ""},
		"host-fv-only":          {"", "failed"},
		"host-remove-verifying": {"", ""},
	}, gotProf)

	gotDecl := map[string]string{}
	rows, err = db.Query(`SELECT host_uuid, status FROM host_mdm_apple_declarations_status`)
	require.NoError(t, err)
	for rows.Next() {
		var host, status string
		require.NoError(t, rows.Scan(&host, &status))
		gotDecl[host] = status
	}
	require.NoError(t, rows.Err())
	defer rows.Close()
	require.Equal(t, map[string]string{
		"host-failed":          "failed",
		"host-pending":         "pending",
		"host-verified":        "verified",
		"host-reserved-only":   "",
		"host-remove-verified": "",
	}, gotDecl)
}
