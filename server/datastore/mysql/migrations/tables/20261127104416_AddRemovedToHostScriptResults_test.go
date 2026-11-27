package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20261127104416(t *testing.T) {
	db := applyUpToPrev(t)

	execNoErr(t, db, `INSERT INTO host_script_results (host_id, execution_id, output, exit_code) VALUES (1, 'exec-1', 'ok', 0)`)

	applyNext(t, db)

	var removed bool
	require.NoError(t, db.Get(&removed, `SELECT removed FROM host_script_results WHERE execution_id = 'exec-1'`))
	require.False(t, removed)
}
