package tables

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestUp_20241224000000(t *testing.T) {
	db := applyUpToPrev(t)

	const (
		insertScriptResultStmt = `INSERT INTO host_script_results (
			host_id, execution_id, script_content_id, script_id, output
		) VALUES (?, ?, ?, ?, '')`

		insertScriptStmt = `INSERT INTO scripts (
			team_id, global_or_team_id, name, script_content_id
		) VALUES (?, ?, ?, ?)`

		loadResultStmt = `SELECT
			id, host_id, execution_id, script_id, is_internal
		FROM host_script_results WHERE id = ?`
	)

	type script struct {
		id, globalOrTeamID   int64
		name, scriptContents string
		teamID               *int64
	}
	type scriptResult struct {
		id, hostID  int64
		executionID string
		scriptID    *int64
		isInternal  bool
	}

	// create a global script
	globalScript := script{
		globalOrTeamID: 0,
		teamID:         nil,
		name:           "global-script",
		scriptContents: "b",
	}

	scriptContentsID := execNoErrLastID(t, db, "INSERT INTO script_contents(contents, md5_checksum) VALUES (?, 'a')", globalScript.scriptContents)

	res, err := db.Exec(insertScriptStmt, globalScript.teamID, globalScript.globalOrTeamID, globalScript.name, scriptContentsID)
	require.NoError(t, err)
	globalScript.id, _ = res.LastInsertId()

	// create a host script result for that global script
	globalScriptResult := scriptResult{
		hostID:      123,
		executionID: uuid.New().String(),
		scriptID:    &globalScript.id,
	}
	res, err = db.Exec(insertScriptResultStmt, globalScriptResult.hostID, globalScriptResult.executionID, scriptContentsID, globalScriptResult.scriptID)
	require.NoError(t, err)
	globalScriptResult.id, _ = res.LastInsertId()

	// Apply current migration.
	applyNext(t, db)

	// the global host script result should be set as internal
	var result scriptResult
	err = db.QueryRow(loadResultStmt, globalScriptResult.id).Scan(&result.id, &result.hostID, &result.executionID, &result.scriptID, &result.isInternal)
	require.NoError(t, err)
	require.False(t, result.isInternal)
	require.Equal(t, globalScriptResult, result)
}
