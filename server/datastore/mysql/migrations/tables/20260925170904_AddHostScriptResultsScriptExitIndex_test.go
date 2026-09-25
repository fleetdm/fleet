package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260925170904(t *testing.T) {
	db := applyUpToPrev(t)

	execNoErr(t, db, `INSERT INTO script_contents (id, md5_checksum, contents) VALUES (1, UNHEX(MD5('echo hi')), 'echo hi')`)
	execNoErr(t, db, `INSERT INTO scripts (id, global_or_team_id, name, script_content_id) VALUES (1, 0, 'test.sh', 1)`)
	execNoErr(t, db, `
		INSERT INTO host_script_results (host_id, execution_id, output, script_id, exit_code) VALUES
			(1, 'pending-exec', '', 1, NULL),
			(2, 'finished-exec', 'ok', 1, 0)`)

	applyNext(t, db)

	require.Equal(t, []string{"script_id", "exit_code"},
		indexColumns(t, db, "host_script_results", "idx_host_script_results_script_exit"))

	// Seeded rows survive the ALTER, and the pending one is still found by the
	// query shape the index serves.
	var execIDs []string
	require.NoError(t, db.Select(&execIDs, `
		SELECT execution_id FROM host_script_results
		WHERE script_id = 1 AND exit_code IS NULL`))
	require.Equal(t, []string{"pending-exec"}, execIDs)
}

func TestUp_20260925170904_AlreadyApplied(t *testing.T) {
	db := applyUpToPrev(t)
	execNoErr(t, db, `ALTER TABLE host_script_results ADD INDEX idx_host_script_results_script_exit (script_id, exit_code)`)

	applyNext(t, db)

	require.Equal(t, []string{"script_id", "exit_code"},
		indexColumns(t, db, "host_script_results", "idx_host_script_results_script_exit"))
}
