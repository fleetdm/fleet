package tables

import (
	"database/sql"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestUp_20230906152143(t *testing.T) {
	db := applyUpToPrev(t)

	const (
		insertOneOffResultStmt = `INSERT INTO host_script_results (
			host_id, execution_id, script_contents, output
		) VALUES (?, ?, ?, '')`

		insertScriptResultStmt = `INSERT INTO host_script_results (
			host_id, execution_id, script_contents, script_id, output
		) VALUES (?, ?, ?, ?, '')`

		insertScriptStmt = `INSERT INTO scripts (
			team_id, global_or_team_id, name, script_contents
		) VALUES (?, ?, ?, ?)`

		insertTeamStmt = `INSERT INTO teams (name) VALUES (?)`

		deleteScriptStmt = `DELETE FROM scripts WHERE id = ?`

		deleteTeamStmt = `DELETE FROM teams WHERE id = ?`

		loadResultStmt = `SELECT
			id, host_id, execution_id, script_contents, script_id
		FROM host_script_results WHERE id = ?`

		loadScriptStmt = `SELECT id FROM scripts WHERE id = ?`
	)

	type script struct {
		id, globalOrTeamID   int64
		name, scriptContents string
		teamID               *int64
	}
	type scriptResult struct {
		id, hostID                  int64
		executionID, scriptContents string
		scriptID                    *int64
	}

	// create an existing (one-off) host script results (using maps to avoid
	// referencing structs that may change in the future)
	preExistingResult := scriptResult{
		hostID:         123,
		executionID:    uuid.New().String(),
		scriptContents: "a",
	}
	res, err := db.Exec(insertOneOffResultStmt, preExistingResult.hostID, preExistingResult.executionID, preExistingResult.scriptContents)
	require.NoError(t, err)
	preExistingResult.id, _ = res.LastInsertId()

	// Apply current migration.
	applyNext(t, db)

	// create a global script
	globalScript := script{
		globalOrTeamID: 0,
		teamID:         nil,
		name:           "global-script",
		scriptContents: "b",
	}
	res, err = db.Exec(insertScriptStmt, globalScript.teamID, globalScript.globalOrTeamID, globalScript.name, globalScript.scriptContents)
	require.NoError(t, err)
	globalScript.id, _ = res.LastInsertId()

	// create a host script result for that global script
	globalScriptResult := scriptResult{
		hostID:         123,
		executionID:    uuid.New().String(),
		scriptContents: globalScript.scriptContents,
		scriptID:       &globalScript.id,
	}
	res, err = db.Exec(insertScriptResultStmt, globalScriptResult.hostID, globalScriptResult.executionID, globalScriptResult.scriptContents, globalScriptResult.scriptID)
	require.NoError(t, err)
	globalScriptResult.id, _ = res.LastInsertId()

	// delete the global script
	_, err = db.Exec(deleteScriptStmt, globalScript.id)
	require.NoError(t, err)

	// the global host script result is still present but now unlinked to the script
	var result scriptResult
	err = db.QueryRow(loadResultStmt, globalScriptResult.id).Scan(&result.id, &result.hostID, &result.executionID, &result.scriptContents, &result.scriptID)
	require.NoError(t, err)
	require.Nil(t, result.scriptID)
	// clear the script id on globalScriptResult to allow comparing the rest of the fields
	globalScriptResult.scriptID = nil
	require.Equal(t, globalScriptResult, result)

	// create a team-specific script
	res, err = db.Exec(insertTeamStmt, "team1")
	require.NoError(t, err)
	teamID, _ := res.LastInsertId()

	teamScript := script{
		globalOrTeamID: teamID,
		teamID:         &teamID,
		name:           "team-script",
		scriptContents: "c",
	}
	res, err = db.Exec(insertScriptStmt, teamScript.teamID, teamScript.globalOrTeamID, teamScript.name, teamScript.scriptContents)
	require.NoError(t, err)
	teamScript.id, _ = res.LastInsertId()

	// create a host script result for that team script
	teamScriptResult := scriptResult{
		hostID:         123,
		executionID:    uuid.New().String(),
		scriptContents: teamScript.scriptContents,
		scriptID:       &teamScript.id,
	}
	res, err = db.Exec(insertScriptResultStmt, teamScriptResult.hostID, teamScriptResult.executionID, teamScriptResult.scriptContents, teamScriptResult.scriptID)
	require.NoError(t, err)
	teamScriptResult.id, _ = res.LastInsertId()

	// delete the team
	_, err = db.Exec(deleteTeamStmt, teamID)
	require.NoError(t, err)

	// the script is deleted, but the host script result still exists (unlinked to the script)
	var notFoundID int64
	err = db.QueryRow(loadScriptStmt, teamScript.id).Scan(&notFoundID)
	require.Error(t, err)
	require.ErrorIs(t, err, sql.ErrNoRows)

	err = db.QueryRow(loadResultStmt, teamScriptResult.id).Scan(&result.id, &result.hostID, &result.executionID, &result.scriptContents, &result.scriptID)
	require.NoError(t, err)
	require.Nil(t, result.scriptID)
	// clear the script id on teamScriptResult to allow comparing the rest of the fields
	teamScriptResult.scriptID = nil
	require.Equal(t, teamScriptResult, result)

	// the pre-existing host script result is still there, untouched
	err = db.QueryRow(loadResultStmt, preExistingResult.id).Scan(&result.id, &result.hostID, &result.executionID, &result.scriptContents, &result.scriptID)
	require.NoError(t, err)
	require.Nil(t, result.scriptID)
	require.Equal(t, preExistingResult, result)
}
