package tables

import (
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestUp_20250102205257(t *testing.T) {
	db := applyUpToPrev(t)

	// Create user
	u1 := execNoErrLastID(t, db, `INSERT INTO users (name, email, password, salt) VALUES (?, ?, ?, ?)`, "u1", "u1@b.c", "1234", "salt")

	// insert a team
	teamID := execNoErrLastID(t, db, `INSERT INTO teams (name) VALUES ("Foo")`)

	// Create host
	insertHostStmt := `
		INSERT INTO hosts (
			hostname, uuid, platform, osquery_version, os_version, build, platform_like, code_name,
			cpu_type, cpu_subtype, cpu_brand, hardware_vendor, hardware_model, hardware_version,
			hardware_serial, computer_name, team_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	hostName := "Dummy Hostname"
	hostUUID := "12345678-1234-1234-1234-123456789012"
	hostPlatform := "ios"
	osqueryVer := "5.9.1"
	osVersion := "Windows 10"
	buildVersion := "10.0.19042.1234"
	platformLike := "apple"
	codeName := "20H2"
	cpuType := "x86_64"
	cpuSubtype := "x86_64"
	cpuBrand := "Intel"
	hwVendor := "Dell Inc."
	hwModel := "OptiPlex 7090"
	hwVersion := "1.0"
	hwSerial := "ABCDEFGHIJ"
	computerName := "DESKTOP-TEST"

	hostID := execNoErrLastID(t, db, insertHostStmt, hostName, hostUUID, hostPlatform, osqueryVer,
		osVersion, buildVersion, platformLike, codeName, cpuType, cpuSubtype, cpuBrand, hwVendor, hwModel, hwVersion, hwSerial, computerName, teamID)

	// Create VPP app, token, and associated team
	adamID := "a"
	execNoErr(
		t, db, `INSERT INTO vpp_apps (adam_id, platform) VALUES (?,?)`, adamID, hostPlatform,
	)
	vppTokenID := execNoErrLastID(t, db, `
	INSERT INTO vpp_tokens (
		organization_name,
		location,
		renew_at,
		token
	) VALUES
		(?, ?, ?, ?)
	`,
		"org1", "loc1", "2030-01-01 10:10:10", "blob1",
	)

	vppAppsTeamsID := execNoErrLastID(
		t,
		db,
		`INSERT INTO vpp_apps_teams (adam_id, platform, global_or_team_id, team_id, vpp_token_id) VALUES (?,?,?,?,?)`,
		adamID,
		fleet.IOSPlatform,
		teamID,
		teamID,
		vppTokenID,
	)

	// Apply current migration.
	applyNext(t, db)

	// create a policy associated with a VPP apps teams record
	policyID := execNoErrLastID(t, db, `INSERT INTO policies (name, query, description, team_id, vpp_apps_teams_id, checksum)
		VALUES ('test_policy', "SELECT 1", "", ?, ?, "a123b123")`, teamID, vppAppsTeamsID)

	// create a VPP install with the policy ID
	hvsi1 := execNoErrLastID(t, db, `INSERT INTO host_vpp_software_installs (host_id, adam_id, platform, command_uuid, user_id, policy_id) VALUES (?,?,?,?,?, ?)`, hostID, adamID, hostPlatform, "command_uuid", u1, policyID)

	// attempt to delete the VPP app; should error
	_, err := db.Exec(`DELETE FROM vpp_apps_teams WHERE id = ?`, vppAppsTeamsID)
	require.Error(t, err)

	// delete the policy
	execNoErr(t, db, `DELETE FROM policies WHERE id = ?`, policyID)

	// confirm that the policy ID on the existing install is null
	var retrievedPolicyID *uint
	require.NoError(t, db.Get(&retrievedPolicyID, `SELECT policy_id FROM host_vpp_software_installs WHERE id = ?`, hvsi1))
	require.Nil(t, retrievedPolicyID)

	// attempt to delete the VPP app; should succeed
	execNoErr(t, db, `DELETE FROM vpp_apps_teams WHERE id = ?`, vppAppsTeamsID)
}
