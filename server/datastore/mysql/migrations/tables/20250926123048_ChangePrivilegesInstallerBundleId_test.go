package tables

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20250926123048_NoPrivileges(t *testing.T) {
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
		WHERE bundle_identifier = 'corp.sap.privileges'
	`)
	require.NoError(t, err)
	require.Equal(t, 0, count, "expected 'corp.sap.privileges' not to be inserted into software_titles")

	err = db.Get(&count, `
		SELECT count(1)
		FROM software_installers
		WHERE title_id = ?
	`, titleId)
	require.NoError(t, err)
	require.Equal(t, 1, count, "expected existing software installer to remain unchanged")
}

func TestUp_20250926123048_NoIndexedPrivileges(t *testing.T) {
	db := applyUpToPrev(t)

	userId := execNoErrLastID(t, db, `INSERT INTO users (name, email, password, salt) VALUES (?, ?, ?, ?)`, "Alice", "alice@example.com", "password", "salt")
	installScriptID := execNoErrLastID(t, db, `INSERT INTO script_contents (md5_checksum, contents) VALUES (?, ?)`, "a", "echo 'install script'")
	uninstallScriptID := execNoErrLastID(t, db, `INSERT INTO script_contents (md5_checksum, contents) VALUES (?, ?)`, "b", "echo 'uninstall script'")

	titleId := execNoErrLastID(t, db, `
		INSERT INTO software_titles (name, source, bundle_identifier) VALUES
		('Privileges', 'apps', 'corp.sap.privileges.pkg')
	`)

	_ = execNoErrLastID(t, db, `
		INSERT INTO software_installers (title_id, filename, version, platform, install_script_content_id, storage_id, self_service, user_id, user_name, user_email, url, package_ids, extension, uninstall_script_content_id, updated_at, fleet_maintained_app_id, install_during_setup, upgrade_code) VALUES
		(?, 'Privileges_2.0.0.pkg', '2.0.0', 'darwin', ?, 'e18bde3e9c86ff5161e193976c68b29fded2fe91a058ec0c336827166d962989', 1, ?, 'Alice', 'alice@example.com', '', 'corp.sap.privileges.pkg', 'pkg', ?, NOW(), NULL, 0, '')
	`, titleId, installScriptID, userId, uninstallScriptID)

	var count int
	err := db.Get(&count, `
		SELECT count(1) FROM
		software_titles
		WHERE bundle_identifier = 'corp.sap.privileges'
	`)
	require.NoError(t, err)
	require.Equal(t, 0, count, "did not expect 'corp.sap.privileges' to be present into software_titles")

	applyNext(t, db)

	titleRows, err := db.Query(`
		SELECT id, bundle_identifier
		FROM software_titles
		WHERE bundle_identifier IN ('corp.sap.privileges.pkg', 'corp.sap.privileges')
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
	require.Contains(t, bundleIdToTitleId, "corp.sap.privileges", "expected 'corp.sap.privileges' to be inserted into software_titles")
	require.NotContains(t, bundleIdToTitleId, "corp.sap.privileges.pkg", "expected 'corp.sap.privileges.pkg' to be deleted from software_titles")

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
	require.Equal(t, bundleIdToTitleId["corp.sap.privileges"], softwareInstallerTitleIds[0], "expected existing software installer to point to correct software title")
}

func TestUp_20250926123048_IndexedPrivileges(t *testing.T) {
	db := applyUpToPrev(t)

	userId := execNoErrLastID(t, db, `INSERT INTO users (name, email, password, salt) VALUES (?, ?, ?, ?)`, "Alice", "alice@example.com", "password", "salt")
	installScriptID := execNoErrLastID(t, db, `INSERT INTO script_contents (md5_checksum, contents) VALUES (?, ?)`, "a", "echo 'install script'")
	uninstallScriptID := execNoErrLastID(t, db, `INSERT INTO script_contents (md5_checksum, contents) VALUES (?, ?)`, "b", "echo 'uninstall script'")

	incorrectTitleId := execNoErrLastID(t, db, `
		INSERT INTO software_titles (name, source, bundle_identifier) VALUES
		('Privileges', 'apps', 'corp.sap.privileges.pkg')
	`)
	correctTitleId := execNoErrLastID(t, db, `
		INSERT INTO software_titles (name, source, bundle_identifier) VALUES
		('Privileges', 'apps', 'corp.sap.privileges')
	`)

	_ = execNoErrLastID(t, db, `
		INSERT INTO software_installers (title_id, filename, version, platform, install_script_content_id, storage_id, self_service, user_id, user_name, user_email, url, package_ids, extension, uninstall_script_content_id, updated_at, fleet_maintained_app_id, install_during_setup, upgrade_code) VALUES
		(?, 'Privileges_2.0.0.pkg', '2.0.0', 'darwin', ?, 'e18bde3e9c86ff5161e193976c68b29fded2fe91a058ec0c336827166d962989', 1, ?, 'Alice', 'alice@example.com', '', 'corp.sap.privileges.pkg', 'pkg', ?, NOW(), NULL, 0, '')
	`, incorrectTitleId, installScriptID, userId, uninstallScriptID)

	applyNext(t, db)

	titleRows, err := db.Query(`
		SELECT id, bundle_identifier
		FROM software_titles
		WHERE bundle_identifier IN ('corp.sap.privileges.pkg', 'corp.sap.privileges')
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
	require.Contains(t, bundleIdToTitleId, "corp.sap.privileges", "expected 'corp.sap.privileges' to be in software_titles")
	require.NotContains(t, bundleIdToTitleId, "corp.sap.privileges.pkg", "expected 'corp.sap.privileges.pkg' to be deleted from software_titles")

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
