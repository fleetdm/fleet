package tables

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUp_20240528150059(t *testing.T) {
	db := applyUpToPrev(t)

	script1 := execNoErrLastID(t, db, "INSERT INTO script_contents(contents, md5_checksum) VALUES ('echo hello', 'a')")
	script2 := execNoErrLastID(t, db, "INSERT INTO script_contents(contents, md5_checksum) VALUES ('echo world', 'b')")

	installer := execNoErrLastID(t, db, `
INSERT INTO software_installers (
  filename,
  version,
  platform,
  install_script_content_id,
  storage_id
) VALUES (
  'fleet',
  '1.0.0',
  'windows',
  ?,
  'a'
)`, script1)

	host := insertHost(t, db, nil)
	hostInstall := execNoErrLastID(t, db, `
INSERT INTO host_software_installs (
  host_id,
  execution_id,
  software_installer_id
) VALUES (?, ?, ?)`, host, "e", installer)

	hostScript := execNoErrLastID(t, db, `
INSERT INTO host_script_results (
	host_id,
	execution_id,
	output,
	script_content_id
) VALUES (?, ?, '', ?)`, host, "f", script2)

	// Apply current migration.
	applyNext(t, db)

	var hostDeletedAt *time.Time
	err := db.Get(&hostDeletedAt, "SELECT host_deleted_at FROM host_software_installs WHERE id = ?", hostInstall)
	require.NoError(t, err)
	require.Nil(t, hostDeletedAt)

	err = db.Get(&hostDeletedAt, "SELECT host_deleted_at FROM host_script_results WHERE id = ?", hostScript)
	require.NoError(t, err)
	require.Nil(t, hostDeletedAt)
}
