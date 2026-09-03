package tables

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUp_20260903190409(t *testing.T) {
	db := applyUpToPrev(t)

	// Seed a host and an install that pre-date the migration.
	hostID := execNoErrLastID(t, db,
		`INSERT INTO hosts (hostname, node_key, osquery_host_id) VALUES ('host1', 'nk1', 'oh1')`)
	titleID := execNoErrLastID(t, db,
		`INSERT INTO software_titles (name, source, extension_for) VALUES ('App', 'apps', '')`)
	scriptID := execNoErrLastID(t, db,
		`INSERT INTO script_contents (md5_checksum, contents) VALUES (UNHEX(MD5('sc')), '')`)
	installerID := execNoErrLastID(t, db, `
		INSERT INTO software_installers
			(title_id, filename, version, platform, install_script_content_id, uninstall_script_content_id, storage_id, package_ids, patch_query)
		VALUES (?, 'app.pkg', '1.0', 'darwin', ?, ?, 'storage-1', '', '')`,
		titleID, scriptID, scriptID)
	existingInstall := execNoErrLastID(t, db, `
		INSERT INTO host_software_installs (host_id, execution_id, software_installer_id)
		VALUES (?, 'exec-before-migration', ?)`, hostID, installerID)

	applyNext(t, db)

	// An install queued before the migration keeps the installer's own pre-install query.
	var overridePreInstallQuery bool
	require.NoError(t, db.GetContext(context.Background(), &overridePreInstallQuery,
		`SELECT override_pre_install_query FROM host_software_installs WHERE id = ?`, existingInstall))
	assert.False(t, overridePreInstallQuery)

	// A new install can ask for the app open query instead.
	newInstall := execNoErrLastID(t, db, `
		INSERT INTO host_software_installs (host_id, execution_id, software_installer_id, override_pre_install_query)
		VALUES (?, 'exec-after-migration', ?, 1)`, hostID, installerID)
	require.NoError(t, db.GetContext(context.Background(), &overridePreInstallQuery,
		`SELECT override_pre_install_query FROM host_software_installs WHERE id = ?`, newInstall))
	assert.True(t, overridePreInstallQuery)
}
