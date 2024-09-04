package tables

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUp_20240612150059(t *testing.T) {
	db := applyUpToPrev(t)

	script1 := execNoErrLastID(t, db, "INSERT INTO script_contents(contents, md5_checksum) VALUES ('echo hello', 'a')")

	host := insertHost(t, db, nil)

	hostScript := execNoErrLastID(t, db, `
INSERT INTO host_script_results (
	host_id,
	execution_id,
	output,
	script_content_id
) VALUES (?, ?, '', ?)`, host, "f", script1)

	// Apply current migration.
	applyNext(t, db)

	var hostDeletedAt *time.Time
	err := db.Get(&hostDeletedAt, "SELECT host_deleted_at FROM host_script_results WHERE id = ?", hostScript)
	require.NoError(t, err)
	require.Nil(t, hostDeletedAt)
}
