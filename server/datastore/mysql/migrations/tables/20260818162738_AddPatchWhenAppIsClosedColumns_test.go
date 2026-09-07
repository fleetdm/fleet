package tables

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUp_20260818162738(t *testing.T) {
	db := applyUpToPrev(t)

	// Seed a policy and an install that pre-date the migration, with timestamps far enough back
	// that any rewrite of the rows would be obvious.
	policyID := execNoErrLastID(
		t, db,
		"INSERT INTO policies (name, query, description, checksum, created_at, updated_at) VALUES (?,?,?,?,?,?)",
		"policy1", "", "", "checksum1", "2020-01-01 00:00:00", "2020-01-02 03:04:05",
	)
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
	installID := execNoErrLastID(t, db, `
		INSERT INTO host_software_installs (host_id, execution_id, software_installer_id, created_at, updated_at)
		VALUES (?, 'exec-before-migration', ?, '2020-01-01 00:00:00', '2020-01-02 03:04:05')`,
		hostID, installerID)

	timestamps := func(table string, id int64) (time.Time, time.Time) {
		var row struct {
			CreatedAt time.Time `db:"created_at"`
			UpdatedAt time.Time `db:"updated_at"`
		}
		require.NoError(t, db.GetContext(context.Background(), &row,
			`SELECT created_at, updated_at FROM `+table+` WHERE id = ?`, id))
		return row.CreatedAt, row.UpdatedAt
	}
	policyCreatedBefore, policyUpdatedBefore := timestamps("policies", policyID)
	installCreatedBefore, installUpdatedBefore := timestamps("host_software_installs", installID)

	applyNext(t, db)

	// Timestamps survive the migration. They matter because both tables are
	// ON UPDATE CURRENT_TIMESTAMP: the API serves policies.updated_at, so rewriting rows would make
	// every policy look freshly edited, and host_software_installs.updated_at throttles continuous
	// policy automation re-installs.
	policyCreatedAfter, policyUpdatedAfter := timestamps("policies", policyID)
	assert.Equal(t, policyCreatedBefore, policyCreatedAfter, "migration must not touch policies.created_at")
	assert.Equal(t, policyUpdatedBefore, policyUpdatedAfter, "migration must not touch policies.updated_at")

	installCreatedAfter, installUpdatedAfter := timestamps("host_software_installs", installID)
	assert.Equal(t, installCreatedBefore, installCreatedAfter, "migration must not touch host_software_installs.created_at")
	assert.Equal(t, installUpdatedBefore, installUpdatedAfter, "migration must not touch host_software_installs.updated_at")

	// Existing rows get the default.
	var notifyBeforePatching bool
	require.NoError(t, db.GetContext(context.Background(), &notifyBeforePatching,
		`SELECT notify_before_patching FROM policies WHERE id = ?`, policyID))
	assert.False(t, notifyBeforePatching)

	// An install queued before the migration keeps the installer's own pre-install query.
	var overridePreInstallQuery bool
	require.NoError(t, db.GetContext(context.Background(), &overridePreInstallQuery,
		`SELECT override_pre_install_query FROM host_software_installs WHERE id = ?`, installID))
	assert.False(t, overridePreInstallQuery)

	// New rows can set the columns.
	policy2 := execNoErrLastID(
		t, db, "INSERT INTO policies (name, query, description, checksum, notify_before_patching) VALUES (?,?,?,?,?)",
		"policy2", "", "", "checksum2", 1,
	)
	require.NoError(t, db.GetContext(context.Background(), &notifyBeforePatching,
		`SELECT notify_before_patching FROM policies WHERE id = ?`, policy2))
	assert.True(t, notifyBeforePatching)

	newInstall := execNoErrLastID(t, db, `
		INSERT INTO host_software_installs (host_id, execution_id, software_installer_id, override_pre_install_query)
		VALUES (?, 'exec-after-migration', ?, 1)`, hostID, installerID)
	require.NoError(t, db.GetContext(context.Background(), &overridePreInstallQuery,
		`SELECT override_pre_install_query FROM host_software_installs WHERE id = ?`, newInstall))
	assert.True(t, overridePreInstallQuery)
}
