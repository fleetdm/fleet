package mysql

import (
	"context"
	"crypto/md5" //nolint:gosec
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/fleetdm/fleet/v4/pkg/scripts"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) NewHostScriptExecutionRequest(ctx context.Context, request *fleet.HostScriptRequestPayload) (*fleet.HostScriptResult, error) {
	var res *fleet.HostScriptResult
	return res, ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		var err error
		if request.ScriptContentID == 0 {
			// then we are doing a sync execution, so create the contents first
			scRes, err := insertScriptContents(ctx, request.ScriptContents, tx)
			if err != nil {
				return err
			}

			id, _ := scRes.LastInsertId()
			request.ScriptContentID = uint(id)
		}
		res, err = newHostScriptExecutionRequest(ctx, request, tx)
		return err
	})
}

func newHostScriptExecutionRequest(ctx context.Context, request *fleet.HostScriptRequestPayload, tx sqlx.ExtContext) (*fleet.HostScriptResult, error) {
	const (
		insStmt = `INSERT INTO host_script_results (host_id, execution_id, script_content_id, output, script_id, user_id, sync_request) VALUES (?, ?, ?, '', ?, ?, ?)`
		getStmt = `SELECT hsr.id, hsr.host_id, hsr.execution_id, hsr.created_at, hsr.script_id, hsr.user_id, hsr.sync_request, sc.contents as script_contents FROM host_script_results hsr JOIN script_contents sc WHERE sc.id = hsr.script_content_id AND hsr.id = ?`
	)

	execID := uuid.New().String()
	result, err := tx.ExecContext(ctx, insStmt,
		request.HostID,
		execID,
		request.ScriptContentID,
		request.ScriptID,
		request.UserID,
		request.SyncRequest,
	)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "new host script execution request")
	}

	var script fleet.HostScriptResult
	id, _ := result.LastInsertId()
	err = sqlx.GetContext(ctx, tx, &script, getStmt, id)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting the created host script result to return")
	}

	return &script, nil
}

func truncateScriptResult(output string) string {
	const maxOutputRuneLen = 10000
	if len(output) > utf8.UTFMax*maxOutputRuneLen {
		// truncate the bytes as we know the output is too long, no point
		// converting more bytes than needed to runes.
		output = output[len(output)-(utf8.UTFMax*maxOutputRuneLen):]
	}
	if utf8.RuneCountInString(output) > maxOutputRuneLen {
		outputRunes := []rune(output)
		output = string(outputRunes[len(outputRunes)-maxOutputRuneLen:])
	}
	return output
}

func (ds *Datastore) SetHostScriptExecutionResult(ctx context.Context, result *fleet.HostScriptResultPayload) (*fleet.HostScriptResult, error) {
	const resultExistsStmt = `
	SELECT
		1
	FROM
		host_script_results
	WHERE
	 	host_id = ? AND
		execution_id = ? AND
		exit_code IS NOT NULL
`

	const updStmt = `
  UPDATE host_script_results SET
    output = ?,
    runtime = ?,
    exit_code = ?
  WHERE
    host_id = ? AND
    execution_id = ?`

	const hostMDMActionsStmt = `
  SELECT
    CASE
      WHEN lock_ref = ? THEN 'lock_ref'
      WHEN unlock_ref = ? THEN 'unlock_ref'
      WHEN wipe_ref = ? THEN 'wipe_ref'
      ELSE ''
    END AS ref_col
  FROM
    host_mdm_actions
  WHERE
    host_id = ?
`

	output := truncateScriptResult(result.Output)

	var hsr *fleet.HostScriptResult
	err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		var resultExists bool
		err := sqlx.GetContext(ctx, tx, &resultExists, resultExistsStmt, result.HostID, result.ExecutionID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return ctxerr.Wrap(ctx, err, "check if host script result exists")
		}
		if resultExists {
			// succeed but leave hsr nil
			return nil
		}

		res, err := tx.ExecContext(ctx, updStmt,
			output,
			result.Runtime,
			// Windows error codes are signed 32-bit integers, but are
			// returned as unsigned integers by the windows API. The
			// software that receives them is responsible for casting
			// it to a 32-bit signed integer.
			// See /orbit/pkg/scripts/exec_windows.go
			int32(result.ExitCode),
			result.HostID,
			result.ExecutionID,
		)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "update host script result")
		}

		if n, _ := res.RowsAffected(); n > 0 {
			// it did update, so return the updated result
			hsr, err = ds.getHostScriptExecutionResultDB(ctx, tx, result.ExecutionID)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "load updated host script result")
			}

			// look up if that script was a lock/unlock/wipe script for that host,
			// and if so update the host_mdm_actions table accordingly.
			var refCol string
			err = sqlx.GetContext(ctx, tx, &refCol, hostMDMActionsStmt, result.ExecutionID, result.ExecutionID, result.ExecutionID, result.HostID)
			if err != nil && !errors.Is(err, sql.ErrNoRows) { // ignore ErrNoRows, refCol will be empty
				return ctxerr.Wrap(ctx, err, "lookup host script corresponding mdm action")
			}
			if refCol != "" {
				err = updateHostLockWipeStatusFromResult(ctx, tx, result.HostID, refCol, result.ExitCode == 0)
				if err != nil {
					return ctxerr.Wrap(ctx, err, "update host mdm action based on script result")
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return hsr, nil
}

func (ds *Datastore) ListPendingHostScriptExecutions(ctx context.Context, hostID uint) ([]*fleet.HostScriptResult, error) {
	const listStmt = `
  SELECT
    id,
    host_id,
    execution_id,
    script_id
  FROM
    host_script_results
  WHERE
    host_id = ? AND
    exit_code IS NULL
    -- async requests + sync requests created within the given interval
    AND (
      sync_request = 0
      OR created_at >= DATE_SUB(NOW(), INTERVAL ? SECOND)
    )
  ORDER BY
    created_at ASC`

	var results []*fleet.HostScriptResult
	seconds := int(scripts.MaxServerWaitTime.Seconds())
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &results, listStmt, hostID, seconds); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list pending host script executions")
	}
	return results, nil
}

func (ds *Datastore) IsExecutionPendingForHost(ctx context.Context, hostID uint, scriptID uint) ([]*uint, error) {
	const getStmt = `
		SELECT
		  1
		FROM
		  host_script_results
		WHERE
		  host_id = ? AND
		  script_id = ? AND
		  exit_code IS NULL
	`

	var results []*uint
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &results, getStmt, hostID, scriptID); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "is execution pending for host")
	}
	return results, nil
}

func (ds *Datastore) GetHostScriptExecutionResult(ctx context.Context, execID string) (*fleet.HostScriptResult, error) {
	return ds.getHostScriptExecutionResultDB(ctx, ds.reader(ctx), execID)
}

func (ds *Datastore) getHostScriptExecutionResultDB(ctx context.Context, q sqlx.QueryerContext, execID string) (*fleet.HostScriptResult, error) {
	const getStmt = `
  SELECT
    hsr.id,
    hsr.host_id,
    hsr.execution_id,
    sc.contents as script_contents,
    hsr.script_id,
    hsr.output,
    hsr.runtime,
    hsr.exit_code,
    hsr.created_at,
    hsr.user_id,
    hsr.sync_request
  FROM
    host_script_results hsr
  JOIN
	script_contents sc
  WHERE
    hsr.execution_id = ?
  AND
	hsr.script_content_id = sc.id
`

	var result fleet.HostScriptResult
	if err := sqlx.GetContext(ctx, q, &result, getStmt, execID); err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("HostScriptResult").WithName(execID))
		}
		return nil, ctxerr.Wrap(ctx, err, "get host script result")
	}
	return &result, nil
}

func (ds *Datastore) NewScript(ctx context.Context, script *fleet.Script) (*fleet.Script, error) {
	var res sql.Result
	err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		var err error

		// first insert script contents
		scRes, err := insertScriptContents(ctx, script.ScriptContents, tx)
		if err != nil {
			return err
		}
		id, _ := scRes.LastInsertId()

		// then create the script entity
		res, err = insertScript(ctx, script, uint(id), tx)
		return err
	})
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return ds.getScriptDB(ctx, ds.writer(ctx), uint(id))
}

func insertScript(ctx context.Context, script *fleet.Script, scriptContentsID uint, tx sqlx.ExtContext) (sql.Result, error) {
	const insertStmt = `
INSERT INTO
  scripts (
    team_id, global_or_team_id, name, script_content_id
  )
VALUES
  (?, ?, ?, ?)
`
	var globalOrTeamID uint
	if script.TeamID != nil {
		globalOrTeamID = *script.TeamID
	}
	res, err := tx.ExecContext(ctx, insertStmt,
		script.TeamID, globalOrTeamID, script.Name, scriptContentsID)
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
	return res, nil
}

func insertScriptContents(ctx context.Context, contents string, tx sqlx.ExtContext) (sql.Result, error) {
	const insertStmt = `
INSERT INTO
  script_contents (
	  md5_checksum, contents
  )
VALUES (UNHEX(?),?)
ON DUPLICATE KEY UPDATE
  id=LAST_INSERT_ID(id)
	`

	md5Checksum := md5ChecksumScriptContent(contents)
	res, err := tx.ExecContext(ctx, insertStmt, md5Checksum, contents)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "insert script contents")
	}

	return res, nil
}

func md5ChecksumScriptContent(s string) string {
	rawChecksum := md5.Sum([]byte(s)) //nolint:gosec
	return strings.ToUpper(hex.EncodeToString(rawChecksum[:]))
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
  updated_at,
  script_content_id
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
  sc.contents
FROM
  script_contents sc
  JOIN scripts s
WHERE
  s.script_content_id = sc.id
  AND s.id = ?;
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

func (ds *Datastore) GetScriptIDByName(ctx context.Context, name string, teamID *uint) (uint, error) {
	const selectStmt = `
SELECT
  id
FROM
  scripts
WHERE
  global_or_team_id = ?
  AND name = ?
`
	var globalOrTeamID uint
	if teamID != nil {
		globalOrTeamID = *teamID
	}

	var id uint
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &id, selectStmt, globalOrTeamID, name); err != nil {
		if err == sql.ErrNoRows {
			return 0, notFound("Script").WithName(name)
		}
		return 0, ctxerr.Wrap(ctx, err, "get script by name")
	}
	return id, nil
}

func (ds *Datastore) GetHostScriptDetails(ctx context.Context, hostID uint, teamID *uint, opt fleet.ListOptions, hostPlatform string) ([]*fleet.HostScriptDetail, *fleet.PaginationMetadata, error) {
	var globalOrTeamID uint
	if teamID != nil {
		globalOrTeamID = *teamID
	}

	var extension string
	switch {
	case hostPlatform == "windows":
		// filter by .ps1 extension
		extension = `%.ps1`
	case fleet.IsUnixLike(hostPlatform):
		// filter by .sh extension
		extension = `%.sh`
	default:
		// no extension filter
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
    team_id, global_or_team_id, name, script_content_id
  )
VALUES
  (?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
  script_content_id = VALUES(script_content_id)
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
			scRes, err := insertScriptContents(ctx, s.ScriptContents, tx)
			if err != nil {
				return ctxerr.Wrapf(ctx, err, "inserting script contents for script with name %q", s.Name)
			}
			id, _ := scRes.LastInsertId()
			if _, err := tx.ExecContext(ctx, insertNewOrEditedScript, tmID, globalOrTeamID, s.Name, uint(id)); err != nil {
				return ctxerr.Wrapf(ctx, err, "insert new/edited script with name %q", s.Name)
			}
		}
		return nil
	})
}

func (ds *Datastore) GetHostLockWipeStatus(ctx context.Context, host *fleet.Host) (*fleet.HostLockWipeStatus, error) {
	const stmt = `
		SELECT
			lock_ref,
			wipe_ref,
			unlock_ref,
			unlock_pin,
			fleet_platform
		FROM
			host_mdm_actions
		WHERE
			host_id = ?
`

	var mdmActions struct {
		LockRef       *string `db:"lock_ref"`
		WipeRef       *string `db:"wipe_ref"`
		UnlockRef     *string `db:"unlock_ref"`
		UnlockPIN     *string `db:"unlock_pin"`
		FleetPlatform string  `db:"fleet_platform"`
	}
	fleetPlatform := host.FleetPlatform()
	status := &fleet.HostLockWipeStatus{
		HostFleetPlatform: fleetPlatform,
	}

	if err := sqlx.GetContext(ctx, ds.reader(ctx), &mdmActions, stmt, host.ID); err != nil {
		if err == sql.ErrNoRows {
			// do not return a Not Found error, return the zero-value status, which
			// will report the correct states.
			return status, nil
		}
		return nil, ctxerr.Wrap(ctx, err, "get host lock/wipe status")
	}

	// if we have a fleet platform stored in host_mdm_actions, use it instead of
	// the host.FleetPlatform() because the platform can be overwritten with an
	// unknown OS name when a Wipe gets executed.
	if mdmActions.FleetPlatform != "" {
		fleetPlatform = mdmActions.FleetPlatform
		status.HostFleetPlatform = fleetPlatform
	}

	switch fleetPlatform {
	case "darwin":
		if mdmActions.UnlockPIN != nil {
			status.UnlockPIN = *mdmActions.UnlockPIN
		}
		if mdmActions.UnlockRef != nil {
			var err error
			status.UnlockRequestedAt, err = time.Parse(time.DateTime, *mdmActions.UnlockRef)
			if err != nil {
				// if the format is unexpected but there's something in UnlockRef, just
				// replace it with the current timestamp, it should still indicate that
				// an unlock was requested (e.g. in case someone plays with the data
				// directly in the DB and messes up the format).
				status.UnlockRequestedAt = time.Now().UTC()
			}
		}

		if mdmActions.LockRef != nil {
			// the lock reference is an MDM command
			cmd, cmdRes, err := ds.getHostMDMAppleCommand(ctx, *mdmActions.LockRef, host.UUID)
			if err != nil {
				return nil, ctxerr.Wrap(ctx, err, "get lock reference")
			}
			status.LockMDMCommand = cmd
			status.LockMDMCommandResult = cmdRes
		}

		if mdmActions.WipeRef != nil {
			// the wipe reference is an MDM command
			cmd, cmdRes, err := ds.getHostMDMAppleCommand(ctx, *mdmActions.WipeRef, host.UUID)
			if err != nil {
				return nil, ctxerr.Wrap(ctx, err, "get wipe reference")
			}
			status.WipeMDMCommand = cmd
			status.WipeMDMCommandResult = cmdRes
		}

	case "windows", "linux":
		// lock and unlock references are scripts
		if mdmActions.LockRef != nil {
			hsr, err := ds.getHostScriptExecutionResultDB(ctx, ds.reader(ctx), *mdmActions.LockRef)
			if err != nil {
				return nil, ctxerr.Wrap(ctx, err, "get lock reference script result")
			}
			status.LockScript = hsr
		}

		if mdmActions.UnlockRef != nil {
			hsr, err := ds.getHostScriptExecutionResultDB(ctx, ds.reader(ctx), *mdmActions.UnlockRef)
			if err != nil {
				return nil, ctxerr.Wrap(ctx, err, "get unlock reference script result")
			}
			status.UnlockScript = hsr
		}

		// wipe is an MDM command on Windows, a script on Linux
		if mdmActions.WipeRef != nil {
			if fleetPlatform == "windows" {
				cmd, cmdRes, err := ds.getHostMDMWindowsCommand(ctx, *mdmActions.WipeRef, host.UUID)
				if err != nil {
					return nil, ctxerr.Wrap(ctx, err, "get wipe reference")
				}
				status.WipeMDMCommand = cmd
				status.WipeMDMCommandResult = cmdRes
			} else {
				hsr, err := ds.getHostScriptExecutionResultDB(ctx, ds.reader(ctx), *mdmActions.WipeRef)
				if err != nil {
					return nil, ctxerr.Wrap(ctx, err, "get wipe reference script result")
				}
				status.WipeScript = hsr
			}
		}
	}

	return status, nil
}

func (ds *Datastore) getHostMDMWindowsCommand(ctx context.Context, cmdUUID, hostUUID string) (*fleet.MDMCommand, *fleet.MDMCommandResult, error) {
	cmd, err := ds.getMDMCommand(ctx, ds.reader(ctx), cmdUUID)
	if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "get Windows MDM command")
	}

	// get the MDM command result, which may be not found (indicating the command
	// is pending). Note that it doesn't return ErrNoRows if not found, it
	// returns success and an empty cmdRes slice.
	cmdResults, err := ds.GetMDMWindowsCommandResults(ctx, cmdUUID)
	if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "get Windows MDM command result")
	}

	// each item in the slice returned by GetMDMWindowsCommandResults is
	// potentially a result for a different host, we need to find the one for
	// that specific host.
	var cmdRes *fleet.MDMCommandResult
	for _, r := range cmdResults {
		if r.HostUUID != hostUUID {
			continue
		}
		// all statuses for Windows indicate end of processing of the command
		// (there is no equivalent of "NotNow" or "Idle" as for Apple).
		cmdRes = r
		break
	}
	return cmd, cmdRes, nil
}

func (ds *Datastore) getHostMDMAppleCommand(ctx context.Context, cmdUUID, hostUUID string) (*fleet.MDMCommand, *fleet.MDMCommandResult, error) {
	cmd, err := ds.getMDMCommand(ctx, ds.reader(ctx), cmdUUID)
	if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "get Apple MDM command")
	}

	// get the MDM command result, which may be not found (indicating the command
	// is pending). Note that it doesn't return ErrNoRows if not found, it
	// returns success and an empty cmdRes slice.
	cmdResults, err := ds.GetMDMAppleCommandResults(ctx, cmdUUID)
	if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "get Apple MDM command result")
	}

	// each item in the slice returned by GetMDMAppleCommandResults is
	// potentially a result for a different host, we need to find the one for
	// that specific host.
	var cmdRes *fleet.MDMCommandResult
	for _, r := range cmdResults {
		if r.HostUUID != hostUUID {
			continue
		}
		if r.Status == fleet.MDMAppleStatusAcknowledged || r.Status == fleet.MDMAppleStatusError || r.Status == fleet.MDMAppleStatusCommandFormatError {
			cmdRes = r
			break
		}
	}
	return cmd, cmdRes, nil
}

// LockHostViaScript will create the script execution request and update
// host_mdm_actions in a single transaction.
func (ds *Datastore) LockHostViaScript(ctx context.Context, request *fleet.HostScriptRequestPayload, hostFleetPlatform string) error {
	var res *fleet.HostScriptResult
	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		var err error

		scRes, err := insertScriptContents(ctx, request.ScriptContents, tx)
		if err != nil {
			return err
		}

		id, _ := scRes.LastInsertId()
		request.ScriptContentID = uint(id)

		res, err = newHostScriptExecutionRequest(ctx, request, tx)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "lock host via script create execution")
		}

		// on duplicate we don't clear any other existing state because at this
		// point in time, this is just a request to lock the host that is recorded,
		// it is pending execution. The host's state should be updated to "locked"
		// only when the script execution is successfully completed, and then any
		// unlock or wipe references should be cleared.
		const stmt = `
	INSERT INTO host_mdm_actions
	(
		host_id,
		lock_ref,
		fleet_platform
	)
	VALUES (?,?,?)
	ON DUPLICATE KEY UPDATE
		lock_ref = VALUES(lock_ref)
	`

		_, err = tx.ExecContext(ctx, stmt,
			request.HostID,
			res.ExecutionID,
			hostFleetPlatform,
		)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "lock host via script update mdm actions")
		}

		return nil
	})
}

// UnlockHostViaScript will create the script execution request and update
// host_mdm_actions in a single transaction.
func (ds *Datastore) UnlockHostViaScript(ctx context.Context, request *fleet.HostScriptRequestPayload, hostFleetPlatform string) error {
	var res *fleet.HostScriptResult
	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		var err error

		scRes, err := insertScriptContents(ctx, request.ScriptContents, tx)
		if err != nil {
			return err
		}

		id, _ := scRes.LastInsertId()
		request.ScriptContentID = uint(id)

		res, err = newHostScriptExecutionRequest(ctx, request, tx)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "unlock host via script create execution")
		}

		// on duplicate we don't clear any other existing state because at this
		// point in time, this is just a request to unlock the host that is
		// recorded, it is pending execution. The host's state should be updated to
		// "unlocked" only when the script execution is successfully completed, and
		// then any lock or wipe references should be cleared.
		const stmt = `
	INSERT INTO host_mdm_actions
	(
		host_id,
		unlock_ref,
		fleet_platform
	)
	VALUES (?,?,?)
	ON DUPLICATE KEY UPDATE
		unlock_ref = VALUES(unlock_ref),
		unlock_pin = NULL
	`

		_, err = tx.ExecContext(ctx, stmt,
			request.HostID,
			res.ExecutionID,
			hostFleetPlatform,
		)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "unlock host via script update mdm actions")
		}

		return err
	})
}

// WipeHostViaScript creates the script execution request and updates the
// host_mdm_actions table in a single transaction.
func (ds *Datastore) WipeHostViaScript(ctx context.Context, request *fleet.HostScriptRequestPayload, hostFleetPlatform string) error {
	var res *fleet.HostScriptResult
	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		var err error

		scRes, err := insertScriptContents(ctx, request.ScriptContents, tx)
		if err != nil {
			return err
		}

		id, _ := scRes.LastInsertId()
		request.ScriptContentID = uint(id)

		res, err = newHostScriptExecutionRequest(ctx, request, tx)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "wipe host via script create execution")
		}

		// on duplicate we don't clear any other existing state because at this
		// point in time, this is just a request to wipe the host that is recorded,
		// it is pending execution, so if it was locked, it is still locked (so the
		// lock_ref info must still be there).
		const stmt = `
	INSERT INTO host_mdm_actions
	(
		host_id,
		wipe_ref,
		fleet_platform
	)
	VALUES (?,?,?)
	ON DUPLICATE KEY UPDATE
		wipe_ref = VALUES(wipe_ref)
	`

		_, err = tx.ExecContext(ctx, stmt,
			request.HostID,
			res.ExecutionID,
			hostFleetPlatform,
		)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "wipe host via script update mdm actions")
		}

		return err
	})
}

func (ds *Datastore) UnlockHostManually(ctx context.Context, hostID uint, hostFleetPlatform string, ts time.Time) error {
	const stmt = `
	INSERT INTO host_mdm_actions
	(
		host_id,
		unlock_ref,
		fleet_platform
	)
	VALUES (?, ?, ?)
	ON DUPLICATE KEY UPDATE
		-- do not overwrite if a value is already set
		unlock_ref = IF(unlock_ref IS NULL, VALUES(unlock_ref), unlock_ref)
	`
	// for macOS, the unlock_ref is just the timestamp at which the user first
	// requested to unlock the host. This then indicates in the host's status
	// that it's pending an unlock (which requires manual intervention by
	// entering a PIN on the device). The /unlock endpoint can be called multiple
	// times, so we record the timestamp of the first time it was requested and
	// from then on, the host is marked as "pending unlock" until the device is
	// actually unlocked with the PIN. The actual unlocking happens when the
	// device sends an Idle MDM request.
	unlockRef := ts.Format(time.DateTime)
	_, err := ds.writer(ctx).ExecContext(ctx, stmt, hostID, unlockRef, hostFleetPlatform)
	return ctxerr.Wrap(ctx, err, "record manual unlock host request")
}

func buildHostLockWipeStatusUpdateStmt(refCol string, succeeded bool, joinPart string) string {
	var alias string

	stmt := `UPDATE host_mdm_actions `
	if joinPart != "" {
		stmt += `hma ` + joinPart
		alias = "hma."
	}
	stmt += ` SET `

	if succeeded {
		switch refCol {
		case "lock_ref":
			// Note that this must not clear the unlock_pin, because recording the
			// lock request does generate the PIN and store it there to be used by an
			// eventual unlock.
			stmt += fmt.Sprintf("%sunlock_ref = NULL, %[1]swipe_ref = NULL", alias)
		case "unlock_ref":
			// a successful unlock clears itself as well as the lock ref, because
			// unlock is the default state so we don't need to keep its unlock_ref
			// around once it's confirmed.
			stmt += fmt.Sprintf("%slock_ref = NULL, %[1]sunlock_ref = NULL, %[1]sunlock_pin = NULL, %[1]swipe_ref = NULL", alias)
		case "wipe_ref":
			stmt += fmt.Sprintf("%slock_ref = NULL, %[1]sunlock_ref = NULL, %[1]sunlock_pin = NULL", alias)
		}
	} else {
		// if the action failed, then we clear the reference to that action itself so
		// the host stays in the previous state (it doesn't transition to the new
		// state).
		stmt += fmt.Sprintf("%s"+refCol+" = NULL", alias)
	}
	return stmt
}

func (ds *Datastore) UpdateHostLockWipeStatusFromAppleMDMResult(ctx context.Context, hostUUID, cmdUUID, requestType string, succeeded bool) error {
	// a bit of MDM protocol leaking in the mysql layer, but it's either that or
	// the other way around (MDM protocol would translate to database column)
	var refCol string
	switch requestType {
	case "EraseDevice":
		refCol = "wipe_ref"
	case "DeviceLock":
		refCol = "lock_ref"
	default:
		return nil
	}
	return updateHostLockWipeStatusFromResultAndHostUUID(ctx, ds.writer(ctx), hostUUID, refCol, cmdUUID, succeeded)
}

func updateHostLockWipeStatusFromResultAndHostUUID(ctx context.Context, tx sqlx.ExtContext, hostUUID, refCol, cmdUUID string, succeeded bool) error {
	stmt := buildHostLockWipeStatusUpdateStmt(refCol, succeeded, `JOIN hosts h ON hma.host_id = h.id`)
	stmt += ` WHERE h.uuid = ? AND hma.` + refCol + ` = ?`
	_, err := tx.ExecContext(ctx, stmt, hostUUID, cmdUUID)
	return ctxerr.Wrap(ctx, err, "update host lock/wipe status from result via host uuid")
}

func updateHostLockWipeStatusFromResult(ctx context.Context, tx sqlx.ExtContext, hostID uint, refCol string, succeeded bool) error {
	stmt := buildHostLockWipeStatusUpdateStmt(refCol, succeeded, "")
	stmt += ` WHERE host_id = ?`
	_, err := tx.ExecContext(ctx, stmt, hostID)
	return ctxerr.Wrap(ctx, err, "update host lock/wipe status from result")
}

func (ds *Datastore) CleanupUnusedScriptContents(ctx context.Context) error {
	deleteStmt := `
DELETE FROM
  script_contents
WHERE
  NOT EXISTS (
    SELECT 1 FROM host_script_results WHERE script_content_id = script_contents.id)
  AND NOT EXISTS (
    SELECT 1 FROM scripts WHERE script_content_id = script_contents.id)
		`
	_, err := ds.writer(ctx).ExecContext(ctx, deleteStmt)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "cleaning up unused script contents")
	}
	return nil
}

func (ds *Datastore) getOrGenerateScriptContentsID(ctx context.Context, contents string) (uint, error) {
	csum := md5ChecksumScriptContent(contents)
	scriptContentsID, err := ds.optimisticGetOrInsert(ctx,
		&parameterizedStmt{
			Statement: `SELECT id FROM script_contents WHERE md5_checksum = UNHEX(?)`,
			Args:      []interface{}{csum},
		},
		&parameterizedStmt{
			Statement: `INSERT INTO script_contents (md5_checksum, contents) VALUES (UNHEX(?), ?)`,
			Args:      []interface{}{csum, contents},
		},
	)
	if err != nil {
		return 0, err
	}
	return scriptContentsID, nil
}
