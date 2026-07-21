package tables

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUp_20260721173820(t *testing.T) {
	db := applyUpToPrev(t)

	// Seed a policy and a software installer that pre-date the migration.
	policyID := execNoErrLastID(
		t, db, "INSERT INTO policies (name, query, description, checksum) VALUES (?,?,?,?)",
		"policy1", "", "", "checksum1",
	)

	titleID := execNoErrLastID(t, db,
		`INSERT INTO software_titles (name, source, extension_for) VALUES ('App', 'apps', '')`)
	scriptID := execNoErrLastID(t, db,
		`INSERT INTO script_contents (md5_checksum, contents) VALUES (UNHEX(MD5('sc')), '')`)
	installerID := execNoErrLastID(t, db, `
		INSERT INTO software_installers
			(title_id, filename, version, platform, install_script_content_id, uninstall_script_content_id, storage_id, package_ids, patch_query)
		VALUES (?, 'app.pkg', '1.0', 'darwin', ?, ?, 'storage-1', '', '')`,
		titleID, scriptID, scriptID)

	applyNext(t, db)

	// Existing rows get the defaults: patch_when_closed=0 and an empty managed query.
	var patchWhenClosed bool
	require.NoError(t, db.GetContext(context.Background(), &patchWhenClosed,
		`SELECT patch_when_closed FROM policies WHERE id = ?`, policyID))
	assert.False(t, patchWhenClosed)

	var appOpenQuery string
	require.NoError(t, db.GetContext(context.Background(), &appOpenQuery,
		`SELECT app_open_query FROM software_installers WHERE id = ?`, installerID))
	assert.Empty(t, appOpenQuery)

	// New rows can set both columns.
	policy2 := execNoErrLastID(
		t, db, "INSERT INTO policies (name, query, description, checksum, patch_when_closed) VALUES (?,?,?,?,?)",
		"policy2", "", "", "checksum2", 1,
	)
	require.NoError(t, db.GetContext(context.Background(), &patchWhenClosed,
		`SELECT patch_when_closed FROM policies WHERE id = ?`, policy2))
	assert.True(t, patchWhenClosed)

	const managedQuery = "SELECT 1 WHERE NOT EXISTS (SELECT 1 FROM apps a JOIN processes p ON p.path LIKE concat(a.path, '/%') WHERE a.bundle_identifier = 'com.example.app');"
	title2 := execNoErrLastID(t, db,
		`INSERT INTO software_titles (name, source, extension_for) VALUES ('App2', 'apps', '')`)
	installer2 := execNoErrLastID(t, db, `
		INSERT INTO software_installers
			(title_id, filename, version, platform, install_script_content_id, uninstall_script_content_id, storage_id, package_ids, patch_query, app_open_query)
		VALUES (?, 'app2.pkg', '1.0', 'darwin', ?, ?, 'storage-2', '', '', ?)`,
		title2, scriptID, scriptID, managedQuery)
	require.NoError(t, db.GetContext(context.Background(), &appOpenQuery,
		`SELECT app_open_query FROM software_installers WHERE id = ?`, installer2))
	assert.Equal(t, managedQuery, appOpenQuery)
}
