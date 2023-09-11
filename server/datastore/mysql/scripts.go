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
		insStmt = `INSERT INTO host_script_results (host_id, execution_id, script_contents, output) VALUES (?, ?, ?, '')`
		getStmt = `SELECT id, host_id, execution_id, script_contents, created_at FROM host_script_results WHERE id = ?`
	)

	execID := uuid.New().String()
	result, err := ds.writer(ctx).ExecContext(ctx, insStmt,
		request.HostID,
		execID,
		request.ScriptContents,
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

func (d *Datastore) NewScript(ctx context.Context, script *fleet.Script) (*fleet.Script, error) {
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
	res, err := d.writer(ctx).ExecContext(ctx, insertStmt,
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
	return d.getScriptDB(ctx, d.writer(ctx), uint(id))
}

func (d *Datastore) getScriptDB(ctx context.Context, q sqlx.QueryerContext, id uint) (*fleet.Script, error) {
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
