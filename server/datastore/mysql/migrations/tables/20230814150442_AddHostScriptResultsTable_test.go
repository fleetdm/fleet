package tables

import (
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestUp_20230814150442(t *testing.T) {
	db := applyUpToPrev(t)

	// Apply current migration.
	applyNext(t, db)

	// NOTE: output field must be provided explicitly (even if empty), because TEXT fields
	// cannot have a default value.
	insertStmt := `INSERT INTO host_script_results (
		host_id, execution_id, script_contents, output
	) VALUES (?, ?, ?, '')`

	hostID := 123
	execID := uuid.New().String()
	scriptContents := "echo 'hello world'"
	res, err := db.Exec(insertStmt, hostID, execID, scriptContents)
	require.NoError(t, err)

	id, _ := res.LastInsertId()
	require.Greater(t, id, int64(0))

	type hostScriptResult struct {
		ID             int           `db:"id"`
		HostID         int           `db:"host_id"`
		ExecutionID    string        `db:"execution_id"`
		ScriptContents string        `db:"script_contents"`
		Output         string        `db:"output"`
		Runtime        int           `db:"runtime"`
		ExitCode       sql.NullInt64 `db:"exit_code"`
		CreatedAt      time.Time     `db:"created_at"`
		UpdatedAt      time.Time     `db:"updated_at"`
	}

	// load the host we just created
	var scriptResult hostScriptResult
	selectStmt := `SELECT id, host_id, execution_id, script_contents, output, runtime, exit_code, created_at, updated_at
	FROM host_script_results
	WHERE id = ?`
	err = db.Get(&scriptResult, selectStmt, id)
	require.NoError(t, err)

	require.Equal(t, int(id), scriptResult.ID)
	require.Equal(t, hostID, scriptResult.HostID)
	require.Equal(t, execID, scriptResult.ExecutionID)
	require.Equal(t, scriptContents, scriptResult.ScriptContents)
	require.Empty(t, scriptResult.Output)
	require.Zero(t, scriptResult.Runtime)
	require.False(t, scriptResult.ExitCode.Valid)
	require.NotZero(t, scriptResult.CreatedAt)
	require.NotZero(t, scriptResult.UpdatedAt)

	// check pending executions for a given host
	var countPending int
	countPendingStmt := `SELECT COUNT(*)
	FROM host_script_results
	WHERE host_id = ? AND exit_code IS NULL`
	err = db.Get(&countPending, countPendingStmt, hostID)
	require.NoError(t, err)
	require.Equal(t, 1, countPending)

	// update the host we just created
	output := `hello world`
	runtime := 10
	exitCode := int64(0)
	updateStmt := `UPDATE host_script_results SET output = ?, runtime = ?, exit_code = ? WHERE host_id = ? AND execution_id = ?`
	_, err = db.Exec(updateStmt, output, runtime, exitCode, hostID, execID)
	require.NoError(t, err)

	// reload the updated host result
	err = db.Get(&scriptResult, selectStmt, id)
	require.NoError(t, err)

	require.Equal(t, output, scriptResult.Output)
	require.Equal(t, runtime, scriptResult.Runtime)
	require.True(t, scriptResult.ExitCode.Valid)
	require.Equal(t, exitCode, scriptResult.ExitCode.Int64)
}
