package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"time"
	"unicode/utf8"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) NewHostScriptExecutionRequest(ctx context.Context, request *fleet.HostScriptRequestPayload) (*fleet.HostScriptResult, error) {
	const (
		insStmt = `INSERT INTO host_script_results (host_id, execution_id, script_contents, output, script_id) VALUES (?, ?, ?, '', ?)`
		getStmt = `SELECT id, host_id, execution_id, script_contents, created_at, script_id FROM host_script_results WHERE id = ?`
	)

	execID := uuid.New().String()
	result, err := ds.writer(ctx).ExecContext(ctx, insStmt,
		request.HostID,
		execID,
		request.ScriptContents,
		request.ScriptID,
	)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "new host script execution request")
	}

	var script fleet.HostScriptResult
	id, _ := result.LastInsertId()
	if err := ds.writer(ctx).GetContext(ctx, &script, getStmt, id); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting the created host script result to return")
	}
	return &script, nil
}

func (ds *Datastore) SetHostScriptExecutionResult(ctx context.Context, result *fleet.HostScriptResultPayload) error {
	const updStmt = `
  UPDATE host_script_results SET
    output = ?,
    runtime = ?,
    exit_code = ?
  WHERE
    host_id = ? AND
    execution_id = ?`

	const maxOutputRuneLen = 10000
	output := result.Output
	if len(output) > utf8.UTFMax*maxOutputRuneLen {
		// truncate the bytes as we know the output is too long, no point
		// converting more bytes than needed to runes.
		output = output[len(output)-(utf8.UTFMax*maxOutputRuneLen):]
	}
	if utf8.RuneCountInString(output) > maxOutputRuneLen {
		outputRunes := []rune(output)
		output = string(outputRunes[len(outputRunes)-maxOutputRuneLen:])
	}

	if _, err := ds.writer(ctx).ExecContext(ctx, updStmt,
		output,
		result.Runtime,
		result.ExitCode,
		result.HostID,
		result.ExecutionID,
	); err != nil {
		return ctxerr.Wrap(ctx, err, "update host script result")
	}
	return nil
}

func (ds *Datastore) ListPendingHostScriptExecutions(ctx context.Context, hostID uint, ignoreOlder time.Duration) ([]*fleet.HostScriptResult, error) {
	const listStmt = `
  SELECT
    id,
    host_id,
    execution_id,
    script_id,
    script_contents
  FROM
    host_script_results
  WHERE
    host_id = ? AND
    exit_code IS NULL AND
    created_at >= DATE_SUB(NOW(), INTERVAL ? SECOND)`

	var results []*fleet.HostScriptResult
	seconds := int(ignoreOlder.Seconds())
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &results, listStmt, hostID, seconds); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list pending host script results")
	}
	return results, nil
}

func (ds *Datastore) GetHostScriptExecutionResult(ctx context.Context, execID string) (*fleet.HostScriptResult, error) {
	const getStmt = `
  SELECT
    id,
    host_id,
    execution_id,
    script_contents,
    script_id,
    output,
    runtime,
    exit_code,
    created_at
  FROM
    host_script_results
  WHERE
    execution_id = ?`

	var result fleet.HostScriptResult
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &result, getStmt, execID); err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("HostScriptResult").WithName(execID))
		}
		return nil, ctxerr.Wrap(ctx, err, "get host script result")
	}
	return &result, nil
}

func (ds *Datastore) NewScript(ctx context.Context, script *fleet.Script) (*fleet.Script, error) {
	const insertStmt = `
INSERT INTO
  scripts (
    team_id, global_or_team_id, name, script_contents
  )
VALUES
  (?, ?, ?, ?)
`
	var globalOrTeamID uint
	if script.TeamID != nil {
		globalOrTeamID = *script.TeamID
	}
	res, err := ds.writer(ctx).ExecContext(ctx, insertStmt,
		script.TeamID, globalOrTeamID, script.Name, script.ScriptContents)
	if err != nil {
		if isDuplicate(err) {
			// name already exists for this team/global
			err = alreadyExists("Script", script.Name)
		} else if isChildForeignKeyError(err) {
			// team does not exist
			err = foreignKey("scripts", fmt.Sprintf("team_id=%v", script.TeamID))
		}
		return nil, ctxerr.Wrap(ctx, err, "insert script")
	}
	id, _ := res.LastInsertId()
	return ds.getScriptDB(ctx, ds.writer(ctx), uint(id))
}

func (ds *Datastore) Script(ctx context.Context, id uint) (*fleet.Script, error) {
	return ds.getScriptDB(ctx, ds.reader(ctx), id)
}

func (ds *Datastore) getScriptDB(ctx context.Context, q sqlx.QueryerContext, id uint) (*fleet.Script, error) {
	const getStmt = `
SELECT
  id,
  team_id,
  name,
  created_at,
  updated_at
FROM
  scripts
WHERE
  id = ?
`
	var script fleet.Script
	if err := sqlx.GetContext(ctx, q, &script, getStmt, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, notFound("Script").WithID(id)
		}
		return nil, ctxerr.Wrap(ctx, err, "get script")
	}
	return &script, nil
}

func (ds *Datastore) GetScriptContents(ctx context.Context, id uint) ([]byte, error) {
	const getStmt = `
SELECT
  script_contents
FROM
  scripts
WHERE
  id = ?
`
	var contents []byte
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &contents, getStmt, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, notFound("Script").WithID(id)
		}
		return nil, ctxerr.Wrap(ctx, err, "get script contents")
	}
	return contents, nil
}

func (ds *Datastore) DeleteScript(ctx context.Context, id uint) error {
	_, err := ds.writer(ctx).ExecContext(ctx, `DELETE FROM scripts WHERE id = ?`, id)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "delete script")
	}
	return nil
}

func (ds *Datastore) ListScripts(ctx context.Context, teamID *uint, opt fleet.ListOptions) ([]*fleet.Script, *fleet.PaginationMetadata, error) {
	var scripts []*fleet.Script

	const selectStmt = `
SELECT
  s.id,
  s.team_id,
  s.name,
  s.created_at,
  s.updated_at
FROM
  scripts s
WHERE
  s.global_or_team_id = ?
`
	var globalOrTeamID uint
	if teamID != nil {
		globalOrTeamID = *teamID
	}

	args := []any{globalOrTeamID}
	stmt, args := appendListOptionsWithCursorToSQL(selectStmt, args, &opt)

	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &scripts, stmt, args...); err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "select scripts")
	}

	var metaData *fleet.PaginationMetadata
	if opt.IncludeMetadata {
		metaData = &fleet.PaginationMetadata{HasPreviousResults: opt.Page > 0}
		if len(scripts) > int(opt.PerPage) {
			metaData.HasNextResults = true
			scripts = scripts[:len(scripts)-1]
		}
	}
	return scripts, metaData, nil
}

func (ds *Datastore) GetHostScriptDetails(ctx context.Context, hostID uint, teamID *uint, opt fleet.ListOptions, hostPlatform string) ([]*fleet.HostScriptDetail, *fleet.PaginationMetadata, error) {
	var globalOrTeamID uint
	if teamID != nil {
		globalOrTeamID = *teamID
	}

	var extension string
	switch hostPlatform {
	case "darwin":
		extension = `%.sh`
		break
	case "windows":
		extension = `%.ps1`
	}

	type row struct {
		ScriptID    uint       `db:"script_id"`
		Name        string     `db:"name"`
		HSRID       *uint      `db:"hsr_id"`
		ExecutionID *string    `db:"execution_id"`
		ExecutedAt  *time.Time `db:"executed_at"`
		ExitCode    *int64     `db:"exit_code"`
	}

	sql := `
SELECT
	s.id AS script_id,
	s.name,
	hsr.id AS hsr_id,
	hsr.created_at AS executed_at,
	hsr.execution_id,
	hsr.exit_code
FROM
	scripts s
	LEFT JOIN (
		SELECT
			id,
			host_id,		
			script_id,
			execution_id,
			created_at,
			exit_code
		FROM
			host_script_results r
		WHERE
			host_id = ?
			AND NOT EXISTS (
				SELECT
					1
				FROM
					host_script_results
				WHERE
					host_id = ?
					AND id != r.id
					AND script_id = r.script_id
					AND(created_at > r.created_at
						OR(created_at = r.created_at
							AND id > r.id)))) hsr
	ON s.id = hsr.script_id
WHERE
	(hsr.host_id IS NULL OR hsr.host_id = ?)
	AND s.global_or_team_id = ?
	`

	args := []any{hostID, hostID, hostID, globalOrTeamID}
	if len(extension) > 0 {
		args = append(args, extension)
		sql += `
		AND s.name LIKE ?
		`
	}
	stmt, args := appendListOptionsWithCursorToSQL(sql, args, &opt)

	var rows []*row
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &rows, stmt, args...); err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "get host script details")
	}

	var metaData *fleet.PaginationMetadata
	if opt.IncludeMetadata {
		metaData = &fleet.PaginationMetadata{HasPreviousResults: opt.Page > 0}
		if len(rows) > int(opt.PerPage) {
			metaData.HasNextResults = true
			rows = rows[:len(rows)-1]
		}
	}

	results := make([]*fleet.HostScriptDetail, 0, len(rows))
	for _, r := range rows {
		results = append(results, fleet.NewHostScriptDetail(hostID, r.ScriptID, r.Name, r.ExecutionID, r.ExecutedAt, r.ExitCode, r.HSRID))
	}

	return results, metaData, nil
}

func (ds *Datastore) BatchSetScripts(ctx context.Context, tmID *uint, scripts []*fleet.Script) error {
	const loadExistingScripts = `
SELECT
  name
FROM
  scripts
WHERE
  global_or_team_id = ? AND
  name IN (?)
`
	const deleteAllScriptsInTeam = `
DELETE FROM
  scripts
WHERE
  global_or_team_id = ?
`

	const deleteScriptsNotInList = `
DELETE FROM
  scripts
WHERE
  global_or_team_id = ? AND
  name NOT IN (?)
`

	const insertNewOrEditedScript = `
INSERT INTO
  scripts (
    team_id, global_or_team_id, name, script_contents
  )
VALUES
  (?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
  script_contents = VALUES(script_contents)
`

	// use a team id of 0 if no-team
	var globalOrTeamID uint
	if tmID != nil {
		globalOrTeamID = *tmID
	}

	// build a list of names for the incoming scripts, will keep the
	// existing ones if there's a match and no change
	incomingNames := make([]string, len(scripts))
	// at the same time, index the incoming scripts keyed by name for ease
	// of processing
	incomingScripts := make(map[string]*fleet.Script, len(scripts))
	for i, p := range scripts {
		incomingNames[i] = p.Name
		incomingScripts[p.Name] = p
	}

	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		var existingScripts []*fleet.Script

		if len(incomingNames) > 0 {
			// load existing scripts that match the incoming scripts by names
			stmt, args, err := sqlx.In(loadExistingScripts, globalOrTeamID, incomingNames)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "build query to load existing scripts")
			}
			if err := sqlx.SelectContext(ctx, tx, &existingScripts, stmt, args...); err != nil {
				return ctxerr.Wrap(ctx, err, "load existing scripts")
			}
		}

		// figure out if we need to delete any scripts
		keepNames := make([]string, 0, len(incomingNames))
		for _, p := range existingScripts {
			if newS := incomingScripts[p.Name]; newS != nil {
				keepNames = append(keepNames, p.Name)
			}
		}

		var (
			stmt string
			args []any
			err  error
		)
		if len(keepNames) > 0 {
			// delete the obsolete scripts
			stmt, args, err = sqlx.In(deleteScriptsNotInList, globalOrTeamID, keepNames)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "build statement to delete obsolete scripts")
			}
		} else {
			stmt = deleteAllScriptsInTeam
			args = []any{globalOrTeamID}
		}
		if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
			return ctxerr.Wrap(ctx, err, "delete obsolete scripts")
		}

		// insert the new scripts and the ones that have changed
		for _, s := range incomingScripts {
			if _, err := tx.ExecContext(ctx, insertNewOrEditedScript, tmID, globalOrTeamID, s.Name, s.ScriptContents); err != nil {
				return ctxerr.Wrapf(ctx, err, "insert new/edited script with name %q", s.Name)
			}
		}
		return nil
	})
}
