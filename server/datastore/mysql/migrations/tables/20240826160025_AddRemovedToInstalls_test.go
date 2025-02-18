package tables

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUp_20240826160025(t *testing.T) {
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

	hostID1 := execNoErrLastID(t, db, insertHostStmt, hostName, hostUUID, hostPlatform, osqueryVer,
		osVersion, buildVersion, platformLike, codeName, cpuType, cpuSubtype, cpuBrand, hwVendor, hwModel, hwVersion, hwSerial,
		computerName, nil)
	hostID2 := execNoErrLastID(t, db, insertHostStmt, hostName, hostUUID, hostPlatform, osqueryVer,
		osVersion, buildVersion, platformLike, codeName, cpuType, cpuSubtype, cpuBrand, hwVendor, hwModel, hwVersion, hwSerial,
		computerName, nil)

	// Insert data into software_titles
	title1 := execNoErrLastID(t, db, "INSERT INTO software_titles (name, source, browser) VALUES (?, ?, ?)", "sw1", "src1", "")
	title2 := execNoErrLastID(t, db, "INSERT INTO software_titles (name, source, browser) VALUES (?, ?, ?)", "sw2", "src2", "")
	title3 := execNoErrLastID(t, db, "INSERT INTO software_titles (name, source, browser) VALUES (?, ?, ?)", "sw3", "src3", "")
	title4 := execNoErrLastID(t, db, "INSERT INTO software_titles (name, source, browser) VALUES (?, ?, ?)", "sw4", "src4", "")

	// Insert software
	const insertStmt = `INSERT INTO software
		(name, version, source, browser, checksum, title_id)
	VALUES
		(?, ?, ?, ?, ?, ?)`
	execNoErr(t, db, insertStmt, "sw1", "1.0", "src1", "", "1", title1)
	sw2 := execNoErrLastID(t, db, insertStmt, "sw2", "2.0", "src2", "", "2", title2)
	sw3 := execNoErrLastID(t, db, insertStmt, "sw3", "3.0", "src3", "", "3", title3)
	// sw4 is not in software table

	// Insert host_software
	execNoErr(t, db, "INSERT INTO host_software (host_id, software_id) VALUES (?, ?)", hostID2, sw2)
	execNoErr(t, db, "INSERT INTO host_software (host_id, software_id) VALUES (?, ?)", hostID2, sw3)

	// Create package apps
	// 1 app will remain because it is not in the software table (sw1)
	// 1 app will remain because it is still installed
	// 1 app will be removed on host 1 but remain on host 2
	execNoErr(t, db, `INSERT INTO script_contents (id, md5_checksum, contents) VALUES (1, 'checksum', 'script content')`)
	siStmt := `INSERT INTO software_installers
	    (title_id, filename, version, platform, install_script_content_id, storage_id)
	  VALUES
		(?,?,?,?,?,?)`
	si1 := execNoErrLastID(t, db, siStmt, title1, "sw1-installer.pkg", "1.2", hostPlatform, 1, "storage-id1")
	si2 := execNoErrLastID(t, db, siStmt, title2, "sw2-installer.pkg", "2.2", hostPlatform, 1, "storage-id2")
	si3 := execNoErrLastID(t, db, siStmt, title3, "sw3-installer.pkg", "3.2", hostPlatform, 1, "storage-id3")
	si4 := execNoErrLastID(t, db, siStmt, title4, "sw3-installer.pkg", "4.2", hostPlatform, 1, "storage-id4")

	hsiStmt := `
	INSERT INTO host_software_installs (
		host_id,
		execution_id,
		software_installer_id,
		install_script_exit_code,
		post_install_script_exit_code
	) VALUES (?, ?, ?, ?, ?)`
	hsi1 := execNoErrLastID(t, db, hsiStmt, hostID1, "execution-id1", si1, 0, nil)       // will be removed
	hsi2_1 := execNoErrLastID(t, db, hsiStmt, hostID2, "execution-id2_1", si2, nil, nil) // remains because it is still being installed
	hsi2_2 := execNoErrLastID(t, db, hsiStmt, hostID2, "execution-id2_2", si2, 0, nil)
	hsi3_1 := execNoErrLastID(t, db, hsiStmt, hostID1, "execution-id3_1", si3, 0, 0) // will be removed because it is not in host_software
	hsi3_2 := execNoErrLastID(t, db, hsiStmt, hostID2, "execution-id3_2", si3, 0, 0)
	hsi4 := execNoErrLastID(t, db, hsiStmt, hostID1, "execution-id4", si4, 0, 0) // remains because it is not in software table

	// Create VPP apps -- 1 VPP app will be removed, 1 will remain
	adamID1 := "removed"
	execNoErr(
		t, db, `INSERT INTO vpp_apps (adam_id, platform, title_id) VALUES (?,?,?)`, adamID1, hostPlatform, title1,
	)
	adamID2 := "kept"
	execNoErr(
		t, db, `INSERT INTO vpp_apps (adam_id, platform, title_id) VALUES (?,?,?)`, adamID2, hostPlatform, title2,
	)

	// create VPP installs
	hvsi1 := execNoErrLastID(t, db,
		`INSERT INTO host_vpp_software_installs (host_id, adam_id, platform, command_uuid, user_id) VALUES (?,?,?,?,?)`,
		hostID1, adamID1, hostPlatform, "command_uuid", u1)
	hvsi2 := execNoErrLastID(t, db,
		`INSERT INTO host_vpp_software_installs (host_id, adam_id, platform, command_uuid, user_id) VALUES (?,?,?,?,?)`,
		hostID2, adamID2, hostPlatform, "command_uuid2", u1)

	time.Sleep(1 * time.Second) // because we are not using max timestamp precision
	execNoErr(t, db, `UPDATE hosts SET detail_updated_at = NOW()`)

	// Apply current migration.
	applyNext(t, db)
	var removed bool

	// Check packages
	require.NoError(t, db.Get(&removed, `SELECT removed from host_software_installs WHERE id = ?`, hsi1))
	assert.True(t, removed)
	require.NoError(t, db.Get(&removed, `SELECT removed from host_software_installs WHERE id = ?`, hsi2_1))
	assert.False(t, removed)
	require.NoError(t, db.Get(&removed, `SELECT removed from host_software_installs WHERE id = ?`, hsi2_2))
	assert.False(t, removed)
	require.NoError(t, db.Get(&removed, `SELECT removed from host_software_installs WHERE id = ?`, hsi3_1))
	assert.True(t, removed)
	require.NoError(t, db.Get(&removed, `SELECT removed from host_software_installs WHERE id = ?`, hsi3_2))
	assert.False(t, removed)
	require.NoError(t, db.Get(&removed, `SELECT removed from host_software_installs WHERE id = ?`, hsi4))
	assert.False(t, removed)

	// Check VPP
	require.NoError(t, db.Get(&removed, `SELECT removed from host_vpp_software_installs WHERE id = ?`, hvsi1))
	assert.True(t, removed)
	require.NoError(t, db.Get(&removed, `SELECT removed from host_vpp_software_installs WHERE id = ?`, hvsi2))
	assert.False(t, removed)

}
