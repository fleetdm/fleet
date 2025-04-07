package tables

import (
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUp_20240905200000(t *testing.T) {
	db := applyUpToPrev(t)

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

	dataStmts := `
	  INSERT INTO script_contents (id, md5_checksum, contents) VALUES
	    (1, 'checksum', 'script content');

	  INSERT INTO software_titles (id, name, source, browser) VALUES
	    (1, 'Foo.app', 'apps', ''),
	    (2, 'Go', 'deb_packages', ''),
	    (3, 'Microsoft Teams.exe', 'programs', '');

	  INSERT INTO software_installers
	    (id, title_id, filename, version, platform, install_script_content_id, storage_id)
	  VALUES
	    (1, 1, 'foo-installer.pkg', '1.1', 'darwin', 1, 'storage-id'),
	    (2, 2, 'go-installer.deb', '2.2', 'linux', 1, 'storage-id'),
	    (3, 3, 'teams-installer.exe', '3.3', 'windows', 1, 'storage-id');
	`
	_, err := db.Exec(dataStmts)
	require.NoError(t, err)

	tx, err := db.Begin()
	require.NoError(t, err)
	txx := sqlx.Tx{Tx: tx, Mapper: reflectx.NewMapperFunc("db", sqlx.NameMapper)}
	scriptID, err := getOrInsertScript(txx, placeholderUninstallScript)
	require.NoError(t, err)
	err = tx.Commit()
	require.NoError(t, err)

	hsiStmt := `
	INSERT INTO host_software_installs (
		host_id,
		execution_id,
		software_installer_id,
		install_script_exit_code
	) VALUES (?, ?, ?, ?)`
	hsi1 := execNoErrLastID(t, db, hsiStmt, hostID1, "execution-id1", 1, 0)

	// Apply current migration.
	applyNext(t, db)

	var scriptIDs []int64
	err = db.Select(&scriptIDs, "SELECT uninstall_script_content_id FROM software_installers WHERE id IN (1, 2)")
	require.NoError(t, err)
	require.ElementsMatch(t, []int64{scriptID, scriptID}, scriptIDs)

	var windowsScript string
	err = db.Get(&windowsScript, `
		SELECT contents FROM script_contents sc
		INNER JOIN software_installers si ON sc.id = si.uninstall_script_content_id
		WHERE si.id = 3`)
	require.NoError(t, err)
	assert.Equal(t, placeholderUninstallScriptWindows, windowsScript)

	var extension string
	err = db.Get(&extension, `SELECT extension FROM software_installers si WHERE si.id = 3 AND updated_at = uploaded_at`)
	require.NoError(t, err)
	assert.Equal(t, "exe", extension)

	var status string
	err = db.Get(&status, "SELECT status FROM host_software_installs WHERE id = ?", hsi1)
	require.NoError(t, err)
	assert.Equal(t, "installed", status)

}
