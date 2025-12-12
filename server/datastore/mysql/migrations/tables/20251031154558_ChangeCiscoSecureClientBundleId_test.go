package tables

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20251031154558_NoSecureClient(t *testing.T) {
	db := applyUpToPrev(t)

	userId := execNoErrLastID(t, db, `INSERT INTO users (name, email, password, salt) VALUES (?, ?, ?, ?)`, "Alice", "alice@example.com", "password", "salt")
	installScriptID := execNoErrLastID(t, db, `INSERT INTO script_contents (md5_checksum, contents) VALUES (?, ?)`, "a", "echo 'install script'")
	uninstallScriptID := execNoErrLastID(t, db, `INSERT INTO script_contents (md5_checksum, contents) VALUES (?, ?)`, "b", "echo 'uninstall script'")

	titleId := execNoErrLastID(t, db, `
		INSERT INTO software_titles (name, source, bundle_identifier) VALUES
		('Some App', 'apps', 'com.some.app')
	`)

	_ = execNoErrLastID(t, db, `
		INSERT INTO software_installers (title_id, filename, version, platform, install_script_content_id, storage_id, user_id, user_name, user_email, url, package_ids, extension, uninstall_script_content_id, updated_at, fleet_maintained_app_id, install_during_setup, upgrade_code) VALUES
		((SELECT id FROM software_titles WHERE bundle_identifier = 'com.some.app'), 'some_app_installer.pkg', '1.0.0', 'darwin', ?, 'dummysha256', ?, 'Alice', 'alice@example.com', '', 'com.some.app.pkg', 'pkg', ?, NOW(), NULL, 0, '')
	`, installScriptID, userId, uninstallScriptID)

	applyNext(t, db)

	var count int
	err := db.Get(&count, `
		SELECT count(1) FROM
		software_titles
		WHERE bundle_identifier = 'com.cisco.secureclient.gui'
	`)
	require.NoError(t, err)
	require.Equal(t, 0, count, "expected 'com.cisco.secureclient.gui' not to be inserted into software_titles")

	err = db.Get(&count, `
		SELECT count(1)
		FROM software_installers
		WHERE title_id = ?
	`, titleId)
	require.NoError(t, err)
	require.Equal(t, 1, count, "expected existing software installer to remain unchanged")
}

func TestUp_20251031154558_NoIndexedSecureClient(t *testing.T) {
	db := applyUpToPrev(t)

	userId := execNoErrLastID(t, db, `INSERT INTO users (name, email, password, salt) VALUES (?, ?, ?, ?)`, "Alice", "alice@example.com", "password", "salt")
	installScriptID := execNoErrLastID(t, db, `INSERT INTO script_contents (md5_checksum, contents) VALUES (?, ?)`, "a", "echo 'install script'")
	uninstallScriptID := execNoErrLastID(t, db, `INSERT INTO script_contents (md5_checksum, contents) VALUES (?, ?)`, "b", "echo 'uninstall script'")

	titleId := execNoErrLastID(t, db, `
		INSERT INTO software_titles (name, source, bundle_identifier) VALUES
		('Cisco Secure Client', 'apps', 'com.cisco.pkg.anyconnect.vpn')
	`)

	_ = execNoErrLastID(t, db, `
		INSERT INTO software_installers (title_id, filename, version, platform, install_script_content_id, storage_id, self_service, user_id, user_name, user_email, url, package_ids, extension, uninstall_script_content_id, updated_at, fleet_maintained_app_id, install_during_setup, upgrade_code) VALUES
		(?, 'cisco-secure-client-macos-5.1.3.62-core-vpn-webdeploy-k9.pkg', '5.1.3.62', 'darwin', ?, '56c973787a9c8b38a5c81591c0d9b891683d41c1ba9c6b742966006e861dc414', 1, ?, 'Alice', 'alice@example.com', '', 'com.cisco.pkg.anyconnect.vpn', 'pkg', ?, NOW(), NULL, 0, '')
	`, titleId, installScriptID, userId, uninstallScriptID)

	var count int
	err := db.Get(&count, `
		SELECT count(1) FROM
		software_titles
		WHERE bundle_identifier = 'com.cisco.secureclient.gui'
	`)
	require.NoError(t, err)
	require.Equal(t, 0, count, "did not expect 'com.cisco.secureclient.gui' to be present into software_titles")

	applyNext(t, db)

	titleRows, err := db.Query(`
		SELECT id, bundle_identifier
		FROM software_titles
		WHERE bundle_identifier IN ('com.cisco.pkg.anyconnect.vpn', 'com.cisco.secureclient.gui')
	`)
	require.NoError(t, err)
	defer titleRows.Close()

	bundleIdToTitleId := map[string]string{}
	for titleRows.Next() {
		var id int
		var bundleId string
		if err := titleRows.Scan(&id, &bundleId); err != nil {
			require.NoError(t, err)
		}
		bundleIdToTitleId[bundleId] = fmt.Sprintf("%d", id)
	}
	require.NoError(t, titleRows.Err())
	require.Contains(t, bundleIdToTitleId, "com.cisco.secureclient.gui", "expected 'com.cisco.secureclient.gui' to be inserted into software_titles")
	require.NotContains(t, bundleIdToTitleId, "com.cisco.pkg.anyconnect.vpn", "expected 'com.cisco.pkg.anyconnect.vpn' to be deleted from software_titles")

	installerRows, err := db.Query(`
		SELECT title_id
		FROM software_installers
	`)
	require.NoError(t, err)
	defer installerRows.Close()
	var softwareInstallerTitleIds []string
	for installerRows.Next() {
		var titleId string
		if err := installerRows.Scan(&titleId); err != nil {
			require.NoError(t, err)
		}
		softwareInstallerTitleIds = append(softwareInstallerTitleIds, titleId)
	}
	require.NoError(t, installerRows.Err())
	require.Len(t, softwareInstallerTitleIds, 1)
	require.Equal(t, bundleIdToTitleId["com.cisco.secureclient.gui"], softwareInstallerTitleIds[0], "expected existing software installer to point to correct software title")
}

func TestUp_20251031154558_IndexedSecureClient(t *testing.T) {
	db := applyUpToPrev(t)

	userId := execNoErrLastID(t, db, `INSERT INTO users (name, email, password, salt) VALUES (?, ?, ?, ?)`, "Alice", "alice@example.com", "password", "salt")
	installScriptID := execNoErrLastID(t, db, `INSERT INTO script_contents (md5_checksum, contents) VALUES (?, ?)`, "a", "echo 'install script'")
	uninstallScriptID := execNoErrLastID(t, db, `INSERT INTO script_contents (md5_checksum, contents) VALUES (?, ?)`, "b", "echo 'uninstall script'")

	incorrectTitleId := execNoErrLastID(t, db, `
		INSERT INTO software_titles (name, source, bundle_identifier) VALUES
		('Cisco Secure Client', 'apps', 'com.cisco.pkg.anyconnect.vpn')
	`)
	correctTitleId := execNoErrLastID(t, db, `
		INSERT INTO software_titles (name, source, bundle_identifier) VALUES
		('Cisco Secure Client', 'apps', 'com.cisco.secureclient.gui')
	`)

	_ = execNoErrLastID(t, db, `
		INSERT INTO software_installers (title_id, filename, version, platform, install_script_content_id, storage_id, self_service, user_id, user_name, user_email, url, package_ids, extension, uninstall_script_content_id, updated_at, fleet_maintained_app_id, install_during_setup, upgrade_code) VALUES
		(?, 'cisco-secure-client-macos-5.1.3.62-core-vpn-webdeploy-k9.pkg', '5.1.3.62', 'darwin', ?, '56c973787a9c8b38a5c81591c0d9b891683d41c1ba9c6b742966006e861dc414', 1, ?, 'Alice', 'alice@example.com', '', 'com.cisco.pkg.anyconnect.vpn', 'pkg', ?, NOW(), NULL, 0, '')
	`, incorrectTitleId, installScriptID, userId, uninstallScriptID)

	applyNext(t, db)

	titleRows, err := db.Query(`
		SELECT id, bundle_identifier
		FROM software_titles
		WHERE bundle_identifier IN ('com.cisco.pkg.anyconnect.vpn', 'com.cisco.secureclient.gui')
	`)
	require.NoError(t, err)
	defer titleRows.Close()

	bundleIdToTitleId := map[string]string{}
	for titleRows.Next() {
		var id int
		var bundleId string
		if err := titleRows.Scan(&id, &bundleId); err != nil {
			require.NoError(t, err)
		}
		bundleIdToTitleId[bundleId] = fmt.Sprintf("%d", id)
	}
	require.NoError(t, titleRows.Err())
	require.Contains(t, bundleIdToTitleId, "com.cisco.secureclient.gui", "expected 'com.cisco.secureclient.gui' to be in software_titles")
	require.NotContains(t, bundleIdToTitleId, "com.cisco.pkg.anyconnect.vpn", "expected 'com.cisco.pkg.anyconnect.vpn' to be deleted from software_titles")

	installerRows, err := db.Query(`
		SELECT title_id
		FROM software_installers
	`)
	require.NoError(t, err)
	defer installerRows.Close()
	var softwareInstallerTitleIds []string
	for installerRows.Next() {
		var titleId string
		if err := installerRows.Scan(&titleId); err != nil {
			require.NoError(t, err)
		}
		softwareInstallerTitleIds = append(softwareInstallerTitleIds, titleId)
	}
	require.NoError(t, installerRows.Err())
	require.Len(t, softwareInstallerTitleIds, 1)
	require.Equal(t, fmt.Sprintf("%d", correctTitleId), softwareInstallerTitleIds[0], "expected existing software installer to point to correct software title")
}
