package mysql

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) GetSetupExperienceScript(ctx context.Context, teamID uint) (*fleet.Script, error) {
	query := `
SELECT
  id,
  team_id,
  global_or_team_id,
  name,
  script_content_id,
  created_at,
  updated_at
FROM
  setup_experience_scripts
WHERE
  global_or_team_id = ?
`
	var script fleet.Script
	// TODO: Add unique constraint on global_or_team_id to enforce only one SE script per team?
	// If so, what to do if multiple scripts exist?
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &script, query, teamID); err != nil {
		return nil, err
	}

	return &script, nil
}

func (ds *Datastore) SetSetupExperienceScript(ctx context.Context, script *fleet.Script) error {
	// var res sql.Result
	err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		var err error

		// first insert script contents
		scRes, err := insertScriptContents(ctx, tx, script.ScriptContents)
		if err != nil {
			return err
		}
		id, _ := scRes.LastInsertId()

		// then create the script entity
		_, err = insertSetupExperienceScript(ctx, tx, script, uint(id))
		return err
	})

	// // TODO: Do we want to return the script here?
	// if err != nil {
	// 	return err
	// }
	// id, _ := res.LastInsertId()

	// return ds.getScriptDB(ctx, ds.writer(ctx), uint(id))

	return err
}

func insertSetupExperienceScript(ctx context.Context, tx sqlx.ExtContext, script *fleet.Script, scriptContentsID uint) (sql.Result, error) {
	const insertStmt = `
INSERT INTO
  setup_experience_scripts (
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
		// TODO: Add unique constraint on global_or_team_id to enforce only one SE script per team?
		// If so, how to detect/handle that error?
		if IsDuplicate(err) {
			// name already exists for this team/global
			err = alreadyExists("Script", script.Name)
		} else if isChildForeignKeyError(err) {
			// team does not exist
			err = foreignKey("setup_experience_scripts", fmt.Sprintf("team_id=%v", script.TeamID))
		}
		return nil, ctxerr.Wrap(ctx, err, "insert setup experience script")
	}

	return res, nil
}

func (ds *Datastore) DeleteSetupExperienceScript(ctx context.Context, teamID uint) error {
	// TODO: Add unique constraint on global_or_team_id to enforce only one SE script per team?
	// If not, this will delete all SE scripts for a team and may need further work.
	_, err := ds.writer(ctx).ExecContext(ctx, `DELETE FROM setup_experience_scripts WHERE global_or_team_id = ?`, teamID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "delete setup experience script")
	}

	// NOTE: CleanupUnusedScriptContents is responsible for removing any orphaned script_contents
	// for setup experience scripts.

	return nil
}
