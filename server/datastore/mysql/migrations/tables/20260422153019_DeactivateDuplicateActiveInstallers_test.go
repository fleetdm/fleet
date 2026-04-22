package tables

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUp_20260422153019(t *testing.T) {
	db := applyUpToPrev(t)

	// Set up prerequisite data: titles, script contents, and installers.
	titleA := execNoErrLastID(t, db, `INSERT INTO software_titles (name, source, browser) VALUES ('AppA', 'deb_packages', '')`)
	titleB := execNoErrLastID(t, db, `INSERT INTO software_titles (name, source, browser) VALUES ('AppB', 'programs', '')`)
	titleFMA := execNoErrLastID(t, db, `INSERT INTO software_titles (name, source, browser) VALUES ('FMA App', 'programs', '')`)

	scriptID := execNoErrLastID(t, db, `INSERT INTO script_contents (md5_checksum, contents) VALUES ('md5abc', 'echo hello')`)

	// -- Case 1: titleA has two active non-FMA installers on global (global_or_team_id=0), different versions.
	installerA1 := execNoErrLastID(t, db, `
		INSERT INTO software_installers
			(team_id, global_or_team_id, title_id, storage_id, filename, extension, version,
			 install_script_content_id, uninstall_script_content_id, platform, package_ids, is_active)
		VALUES (NULL, 0, ?, 'storageA1', 'appA-1.0.deb', 'deb', '1.0', ?, ?, 'linux', '', 1)`,
		titleA, scriptID, scriptID)

	installerA2 := execNoErrLastID(t, db, `
		INSERT INTO software_installers
			(team_id, global_or_team_id, title_id, storage_id, filename, extension, version,
			 install_script_content_id, uninstall_script_content_id, platform, package_ids, is_active)
		VALUES (NULL, 0, ?, 'storageA2', 'appA-2.0.deb', 'deb', '2.0', ?, ?, 'linux', '', 1)`,
		titleA, scriptID, scriptID)

	// -- Case 2: titleB has a single active installer — should NOT be touched.
	installerB := execNoErrLastID(t, db, `
		INSERT INTO software_installers
			(team_id, global_or_team_id, title_id, storage_id, filename, extension, version,
			 install_script_content_id, uninstall_script_content_id, platform, package_ids, is_active)
		VALUES (NULL, 0, ?, 'storageB1', 'appB-1.0.msi', 'msi', '1.0', ?, ?, 'windows', '', 1)`,
		titleB, scriptID, scriptID)

	// -- Case 3: titleFMA has two active FMA installers — should NOT be touched by the migration.
	fmaID := execNoErrLastID(t, db, `INSERT INTO fleet_maintained_apps (name, version, platform, installer_url, sha256, bundle_identifier, install_script_content_id, uninstall_script_content_id)
		VALUES ('FMA App', '1.0', 'linux', 'https://example.com', 'sha256abc', 'com.example.fma', ?, ?)`, scriptID, scriptID)

	installerFMA1 := execNoErrLastID(t, db, `
		INSERT INTO software_installers
			(team_id, global_or_team_id, title_id, storage_id, filename, extension, version,
			 install_script_content_id, uninstall_script_content_id, platform, package_ids,
			 is_active, fleet_maintained_app_id)
		VALUES (NULL, 0, ?, 'storageFMA1', 'fma-1.0.deb', 'deb', '1.0', ?, ?, 'linux', '', 1, ?)`,
		titleFMA, scriptID, scriptID, fmaID)

	installerFMA2 := execNoErrLastID(t, db, `
		INSERT INTO software_installers
			(team_id, global_or_team_id, title_id, storage_id, filename, extension, version,
			 install_script_content_id, uninstall_script_content_id, platform, package_ids,
			 is_active, fleet_maintained_app_id)
		VALUES (NULL, 0, ?, 'storageFMA2', 'fma-2.0.deb', 'deb', '2.0', ?, ?, 'linux', '', 1, ?)`,
		titleFMA, scriptID, scriptID, fmaID)

	// -- Create a policy pointing to the OLD installer (A1); migration should re-point to A2.
	execNoErr(t, db, `INSERT INTO policies (name, query, description, team_id, software_installer_id, checksum)
		VALUES ('auto-install A', 'SELECT 1', 'test policy', NULL, ?, '')`, installerA1)

	// Apply the migration.
	applyNext(t, db)

	// Verify Case 1: only the newest (A2) is active, A1 is deactivated.
	var isActiveA1, isActiveA2 int
	err := db.Get(&isActiveA1, `SELECT is_active FROM software_installers WHERE id = ?`, installerA1)
	require.NoError(t, err)
	assert.Equal(t, 0, isActiveA1, "old installer A1 should be deactivated")

	err = db.Get(&isActiveA2, `SELECT is_active FROM software_installers WHERE id = ?`, installerA2)
	require.NoError(t, err)
	assert.Equal(t, 1, isActiveA2, "newest installer A2 should remain active")

	// Verify the policy was re-pointed to A2.
	var policyInstallerID int64
	err = db.Get(&policyInstallerID, `SELECT software_installer_id FROM policies WHERE name = 'auto-install A'`)
	require.NoError(t, err)
	assert.Equal(t, installerA2, policyInstallerID, "policy should be re-pointed to newest installer")

	// Verify Case 2: single installer B is unchanged.
	var isActiveB int
	err = db.Get(&isActiveB, `SELECT is_active FROM software_installers WHERE id = ?`, installerB)
	require.NoError(t, err)
	assert.Equal(t, 1, isActiveB, "single installer B should remain active")

	// Verify Case 3: FMA installers are untouched.
	var isActiveFMA1, isActiveFMA2 int
	err = db.Get(&isActiveFMA1, `SELECT is_active FROM software_installers WHERE id = ?`, installerFMA1)
	require.NoError(t, err)
	assert.Equal(t, 1, isActiveFMA1, "FMA installer 1 should remain active")

	err = db.Get(&isActiveFMA2, `SELECT is_active FROM software_installers WHERE id = ?`, installerFMA2)
	require.NoError(t, err)
	assert.Equal(t, 1, isActiveFMA2, "FMA installer 2 should remain active")
}
