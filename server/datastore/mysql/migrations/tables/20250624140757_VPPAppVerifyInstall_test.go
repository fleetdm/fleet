package tables

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestUp_20250624140757(t *testing.T) {
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
	hostPlatform := "darwin"
	osqueryVer := "5.9.1"
	osVersion := "macOS 14.5"
	buildVersion := "10.0.19042.1234"
	platformLike := "darwin"
	codeName := "20H2"
	cpuType := "x86_64"
	cpuSubtype := "x86_64"
	cpuBrand := "Intel"
	hwVendor := "Apple Inc."
	hwModel := "Mac14,3"
	hwVersion := "1.0"
	hwSerial := "ABCDEFGHIJ"
	computerName := "DESKTOP-TEST"

	hostID := execNoErrLastID(t, db, insertHostStmt, hostName, hostUUID, hostPlatform, osqueryVer,
		osVersion, buildVersion, platformLike, codeName, cpuType, cpuSubtype, cpuBrand, hwVendor, hwModel, hwVersion, hwSerial, computerName, nil)

	// Create VPP app
	adamID := "a"
	execNoErr(
		t, db, `INSERT INTO vpp_apps (adam_id, platform) VALUES (?,?)`, adamID, hostPlatform,
	)

	installedUUID := uuid.NewString()

	execNoErr(t, db, `INSERT INTO nano_devices (id, authenticate) VALUES (?, ?)`, hostUUID, "auth")
	execNoErr(t, db, `
	INSERT INTO nano_enrollments (id, device_id, type, topic, push_magic, token_hex, last_seen_at)
	VALUES (?, ?, ?, ?, ?, ?, ?)`, hostUUID, hostUUID, "device", "topic", "magic", "hex", time.Now())
	execNoErr(t, db, `INSERT INTO nano_commands (command_uuid, request_type, command) VALUES (?, ?, ?)`,
		installedUUID, "InstallApplication", "<?xml")
	execNoErr(t, db, `INSERT INTO nano_enrollment_queue (id, command_uuid) VALUES (?, ?)`,
		hostUUID, installedUUID)

	execNoErr(t, db, `INSERT INTO nano_command_results (id, command_uuid, status, result) VALUES (?, ?, ?, ?)`,
		hostUUID, installedUUID, "Acknowledged", "<?xml")

	// create an install on a known host
	hvsi1 := execNoErrLastID(t, db, `INSERT INTO host_vpp_software_installs (host_id, adam_id, command_uuid, user_id, platform) VALUES (?,?,?,?,?)`, hostID, adamID, installedUUID, u1, "darwin")

	// Apply current migration.
	applyNext(t, db)

	var verifiedTime *time.Time
	require.NoError(t, db.Get(&verifiedTime, `SELECT verification_at FROM host_vpp_software_installs WHERE id = ?`, hvsi1))
	require.NotNil(t, verifiedTime)
	require.NotZero(t, *verifiedTime)
}
