package tables

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUp_20240801115359(t *testing.T) {
	db := applyUpToPrev(t)

	// Create user
	u1 := execNoErrLastID(t, db, `INSERT INTO users (name, email, password, salt) VALUES (?, ?, ?, ?)`, "u1", "u1@b.c", "1234", "salt")
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
		osVersion, buildVersion, platformLike, codeName, cpuType, cpuSubtype, cpuBrand, hwVendor, hwModel, hwVersion, hwSerial, computerName, nil)
	hostIDMissing := hostID + 1

	// Create VPP app
	adamID := "a"
	execNoErr(
		t, db, `INSERT INTO vpp_apps (adam_id, platform) VALUES (?,?)`, adamID, hostPlatform,
	)
	// Create another VPP app for a different platform
	execNoErr(
		t, db, `INSERT INTO vpp_apps (adam_id, platform) VALUES (?,?)`, adamID, "unused",
	)

	// create an install on a known host
	hvsi1 := execNoErrLastID(t, db, `INSERT INTO host_vpp_software_installs (host_id, adam_id, command_uuid, user_id) VALUES (?,?,?,?)`, hostID, adamID, "command_uuid", u1)
	// create an install on a missing host
	hvsi2 := execNoErrLastID(t, db, `INSERT INTO host_vpp_software_installs (host_id, adam_id, command_uuid, user_id) VALUES (?,?,?,?)`, hostIDMissing, adamID, "command_uuid2", u1)

	// Apply current migration.
	applyNext(t, db)

	// Check that the platform column was updated.
	var platformResp string
	require.NoError(t, db.Get(&platformResp, `SELECT platform FROM host_vpp_software_installs WHERE id = ?`, hvsi1))
	assert.Equal(t, hostPlatform, platformResp)
	require.NoError(t, db.Get(&platformResp, `SELECT platform FROM host_vpp_software_installs WHERE id = ?`, hvsi2))
	assert.Equal(t, hostPlatform, platformResp)
}
