package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20241017163402(t *testing.T) {
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
	    (1, 'Foo.app', 'apps', ''), (2, 'WillBeDeleted.app', 'apps', '');

	  INSERT INTO software_installers
	    (id, title_id, filename, version, platform, install_script_content_id, storage_id, package_ids, uninstall_script_content_id, uploaded_at)
	  VALUES
	    (1, 1, 'foo-installer.pkg', '1.1', 'darwin', 1, 'storage-id', '', 1, NOW() + INTERVAL 5 SECOND),
	    (2, 2, 'to-delete-installer.pkg', '1.2', 'darwin', 1, 'storage-id', '', 1, '2024-09-30 00:00:00');
	`
	_, err := db.Exec(dataStmts)
	require.NoError(t, err)

	hsiStmt := `
	INSERT INTO host_software_installs (
		host_id,
		execution_id,
		software_installer_id,
		install_script_exit_code,
		updated_at,
		uninstall
	) VALUES (?, ?, ?, ?, '2024-10-01 00:00:00', ?)`
	hsi1 := execNoErrLastID(t, db, hsiStmt, hostID1, "execution-id1", 1, 0, 0)
	hsi2 := execNoErrLastID(t, db, hsiStmt, hostID1, "execution-id2", 2, 0, 0)
	hsiUn := execNoErrLastID(t, db, hsiStmt, hostID1, "execution-id3", 2, 0, 1)

	execNoErr(t, db, `DELETE FROM software_titles WHERE id = 2`) // sets title ID to null for installer 2

	// Apply current migration.
	applyNext(t, db)

	result := struct {
		Filename    string `db:"installer_filename"`
		Version     string `db:"version"`
		InstallerID *uint  `db:"software_installer_id"`
		TitleID     *uint  `db:"software_title_id"`
		TitleName   string `db:"software_title_name"`
		UpdatedAt   string `db:"updated_at"`
	}{}

	err = db.Get(&result, "SELECT installer_filename, version, software_installer_id, software_title_id, software_title_name, updated_at FROM host_software_installs WHERE id = ?", hsi1)
	require.NoError(t, err)
	require.Equal(t, "foo-installer.pkg", result.Filename)
	require.Equal(t, "unknown", result.Version)
	require.Equal(t, uint(1), *result.InstallerID)
	require.Equal(t, uint(1), *result.TitleID)
	require.Equal(t, "Foo.app", result.TitleName)
	require.Equal(t, "2024-10-01T00:00:00Z", result.UpdatedAt)

	err = db.Get(&result, "SELECT installer_filename, version, software_installer_id, software_title_id, software_title_name, updated_at FROM host_software_installs WHERE id = ?", hsi2)
	require.NoError(t, err)
	require.Equal(t, "to-delete-installer.pkg", result.Filename)
	require.Equal(t, "1.2", result.Version)
	require.Equal(t, uint(2), *result.InstallerID)
	require.Nil(t, result.TitleID)
	require.Equal(t, "[deleted title]", result.TitleName)
	require.Equal(t, "2024-10-01T00:00:00Z", result.UpdatedAt)

	// we know less about uninstalls as we may be able to uninstall a version that was installed earlier
	err = db.Get(&result, "SELECT installer_filename, version, software_installer_id, software_title_id, software_title_name, updated_at FROM host_software_installs WHERE id = ?", hsiUn)
	require.NoError(t, err)
	require.Equal(t, "", result.Filename)
	require.Equal(t, "unknown", result.Version)
	require.Equal(t, uint(2), *result.InstallerID)
	require.Nil(t, result.TitleID)
	require.Equal(t, "[deleted title]", result.TitleName)
	require.Equal(t, "2024-10-01T00:00:00Z", result.UpdatedAt)

	execNoErr(t, db, `DELETE FROM software_installers WHERE id = 2`) // sets installer ID to null for install 2

	err = db.Get(&result, "SELECT installer_filename, version, software_installer_id, software_title_id, software_title_name, updated_at FROM host_software_installs WHERE id = ?", hsi2)
	require.NoError(t, err)
	require.Equal(t, "to-delete-installer.pkg", result.Filename)
	require.Equal(t, "1.2", result.Version)
	require.Nil(t, result.InstallerID)
	require.Equal(t, "2024-10-01T00:00:00Z", result.UpdatedAt)

	// test activity hydration manual query
	execNoErr(t, db, `INSERT INTO activities (activity_type, details) VALUES 
		("installed_software", '{"install_uuid": "execution-id1", "software_title": "Foo", "software_package": "foo.pkg"}'),
		("installed_software", '{"install_uuid": "execution-id2", "software_title": "A Real Title"}'),
		("uninstalled_software", '{"execution_id": "execution-id3", "software_title": "Ignore Me"}')`)

	execNoErr(t, db, `UPDATE host_software_installs i
JOIN activities a ON a.activity_type = 'installed_software'
	AND i.execution_id = a.details->>"$.install_uuid"
SET i.software_title_name = COALESCE(a.details->>"$.software_title", i.software_title_name),
	i.installer_filename = COALESCE(a.details->>"$.software_package", i.installer_filename),
	i.updated_at = i.updated_at`)

	err = db.Get(&result, "SELECT installer_filename, version, software_installer_id, software_title_id, software_title_name, updated_at FROM host_software_installs WHERE id = ?", hsi1)
	require.NoError(t, err)
	require.Equal(t, "foo.pkg", result.Filename)
	require.Equal(t, "Foo", result.TitleName)
	require.Equal(t, "2024-10-01T00:00:00Z", result.UpdatedAt)

	err = db.Get(&result, "SELECT installer_filename, version, software_installer_id, software_title_id, software_title_name, updated_at FROM host_software_installs WHERE id = ?", hsi2)
	require.NoError(t, err)
	require.Equal(t, "to-delete-installer.pkg", result.Filename)
	require.Equal(t, "A Real Title", result.TitleName)
	require.Equal(t, "2024-10-01T00:00:00Z", result.UpdatedAt)

	// uninstall should not have been modified
	err = db.Get(&result, "SELECT installer_filename, version, software_installer_id, software_title_id, software_title_name, updated_at FROM host_software_installs WHERE id = ?", hsiUn)
	require.NoError(t, err)
	require.Equal(t, "", result.Filename)
	require.Equal(t, "[deleted title]", result.TitleName)
	require.Equal(t, "2024-10-01T00:00:00Z", result.UpdatedAt)
}
