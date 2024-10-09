package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) SetSetupExperienceSoftwareTitles(ctx context.Context, teamID uint, titleIDs []uint) error {
	titleIDQuestionMarks := strings.Join(slices.Repeat([]string{"?"}, len(titleIDs)), ",")

	stmtSelectInstallersIDs := fmt.Sprintf(`
SELECT
	st.id AS title_id,
	si.id,
	st.name,
	si.platform
FROM
	software_titles st
LEFT JOIN
	software_installers si
	ON st.id = si.title_id
WHERE
	si.global_or_team_id = ?
AND
	st.id IN (%s)
`, titleIDQuestionMarks)

	stmtSelectVPPAppsTeamsID := fmt.Sprintf(`
SELECT
	st.id AS title_id,
	vat.id,
	st.name,
	vat.platform
FROM
	software_titles st
LEFT JOIN
	vpp_apps va
	ON st.id = va.title_id
LEFT JOIN
	vpp_apps_teams vat
	ON va.adam_id = vat.adam_id
WHERE
	vat.global_or_team_id = ?
AND
	st.id IN (%s)
`, titleIDQuestionMarks)

	stmtUnsetInstallers := `
UPDATE software_installers
SET install_during_setup = false
WHERE global_or_team_id = ?`

	stmtUnsetVPPAppsTeams := `
UPDATE vpp_apps_teams vat
SET install_during_setup = false
WHERE global_or_team_id = ?`

	stmtSetInstallers := `
UPDATE software_installers
SET install_during_setup = true
WHERE id IN (%s)`

	stmtSetVPPAppsTeams := `
UPDATE vpp_apps_teams
SET install_during_setup = true
WHERE id IN (%s)`

	if err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		var softwareIDPlatforms []idPlatformTuple
		var softwareIDs []any
		var vppIDPlatforms []idPlatformTuple
		var vppAppTeamIDs []any
		missingTitleIDs := make(map[uint]struct{})
		titleIDArgs := make([]any, 0, len(titleIDs))
		titleIDAndTeam := []any{teamID}
		for _, id := range titleIDs {
			missingTitleIDs[id] = struct{}{}
			titleIDArgs = append(titleIDArgs, id)
			titleIDAndTeam = append(titleIDAndTeam, id)
		}

		if err := sqlx.SelectContext(ctx, tx, &softwareIDPlatforms, stmtSelectInstallersIDs, titleIDAndTeam...); err != nil {
			return ctxerr.Wrap(ctx, err, "selecting software IDs using title IDs")
		}

		for _, tuple := range softwareIDPlatforms {
			delete(missingTitleIDs, tuple.TitleID)
			if tuple.Platform != string(fleet.MacOSPlatform) {
				return ctxerr.Errorf(ctx, "only MacOS supported, unsupported software installer: %d (%s, %s)", tuple.ID, tuple.Name, tuple.Platform)
			}
			softwareIDs = append(softwareIDs, tuple.ID)
		}

		if err := sqlx.SelectContext(ctx, tx, &vppIDPlatforms, stmtSelectVPPAppsTeamsID, titleIDAndTeam...); err != nil {
			return ctxerr.Wrap(ctx, err, "selecting vpp app team IDs using title IDs")
		}

		for _, tuple := range vppIDPlatforms {
			delete(missingTitleIDs, tuple.TitleID)
			if tuple.Platform != string(fleet.MacOSPlatform) {
				return ctxerr.Errorf(ctx, "only MacOS supported, unsupported AppStoreApp title: %d (%s, %s)", tuple.ID, tuple.Name, tuple.Platform)
			}
			vppAppTeamIDs = append(vppAppTeamIDs, tuple.ID)
		}

		if len(missingTitleIDs) > 0 {
			var keys []string
			for k := range missingTitleIDs {
				keys = append(keys, fmt.Sprintf("%d", k))
			}
			return ctxerr.Errorf(ctx, "title IDs not available: %s", strings.Join(keys, ","))
		}

		if _, err := tx.ExecContext(ctx, stmtUnsetInstallers, teamID); err != nil {
			return ctxerr.Wrap(ctx, err, "unsetting software installers")
		}

		if _, err := tx.ExecContext(ctx, stmtUnsetVPPAppsTeams, teamID); err != nil {
			return ctxerr.Wrap(ctx, err, "unsetting vpp app teams")
		}

		if len(softwareIDs) > 0 {
			stmtSetInstallersLoop := fmt.Sprintf(stmtSetInstallers, questionMarks(len(softwareIDs)))
			if _, err := tx.ExecContext(ctx, stmtSetInstallersLoop, softwareIDs...); err != nil {
				return ctxerr.Wrap(ctx, err, "setting software installers")
			}
		}

		if len(vppAppTeamIDs) > 0 {
			stmtSetVPPAppsTeamsLoop := fmt.Sprintf(stmtSetVPPAppsTeams, questionMarks(len(vppAppTeamIDs)))
			if _, err := tx.ExecContext(ctx, stmtSetVPPAppsTeamsLoop, vppAppTeamIDs...); err != nil {
				return ctxerr.Wrap(ctx, err, "setting vpp app teams")
			}
		}

		return nil
	}); err != nil {
		return ctxerr.Wrap(ctx, err, "setting setup experience software")
	}

	return nil
}

func (ds *Datastore) ListSetupExperienceSoftwareTitles(ctx context.Context, teamID uint, opts fleet.ListOptions) ([]fleet.SoftwareTitleListResult, int, *fleet.PaginationMetadata, error) {
	opts.IncludeMetadata = true
	opts.After = ""

	titles, count, meta, err := ds.ListSoftwareTitles(ctx, fleet.SoftwareTitleListOptions{
		TeamID:              &teamID,
		ListOptions:         opts,
		Platform:            string(fleet.MacOSPlatform),
		SetupExperienceOnly: true,
	}, fleet.TeamFilter{
		IncludeObserver: true,
		TeamID:          &teamID,
	})
	if err != nil {
		return nil, 0, nil, ctxerr.Wrap(ctx, err, "calling list software titles")
	}

	return titles, count, meta, nil
}

type idPlatformTuple struct {
	ID       uint   `db:"id"`
	TitleID  uint   `db:"title_id"`
	Name     string `db:"name"`
	Platform string `db:"platform"`
}

func questionMarks(number int) string {
	return strings.Join(slices.Repeat([]string{"?"}, number), ",")
}

func (ds *Datastore) ListSetupExperienceResultsByHostUUID(ctx context.Context, hostUUID string) ([]*fleet.SetupExperienceStatusResult, error) {
	const stmt = `
SELECT
	sesr.id,
	sesr.host_uuid,
	sesr.name,
	sesr.status,
	sesr.software_installer_id,
	sesr.host_software_installs_id,
	sesr.vpp_app_team_id,
	sesr.nano_command_uuid,
	sesr.setup_experience_script_id,
	sesr.script_execution_id,
	sesr.error,
	COALESCE(si.title_id, COALESCE(va.title_id, NULL)) AS software_title_id
FROM setup_experience_status_results sesr
LEFT JOIN software_installers si ON si.id = sesr.software_installer_id
LEFT JOIN vpp_apps_teams vat ON vat.id = sesr.vpp_app_team_id
LEFT JOIN vpp_apps va ON vat.adam_id = va.adam_id
WHERE host_uuid = ?
	`
	var results []*fleet.SetupExperienceStatusResult
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &results, stmt, hostUUID); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "select setup experience status results by host uuid")
	}
	return results, nil
}

func (ds *Datastore) GetSetupExperienceScript(ctx context.Context, teamID *uint) (*fleet.Script, error) {
	query := `
SELECT
  id,
  team_id,
  name,
  script_content_id,
  created_at,
  updated_at
FROM
  setup_experience_scripts
WHERE
  global_or_team_id = ?
`
	var globalOrTeamID uint
	if teamID != nil {
		globalOrTeamID = *teamID
	}

	var script fleet.Script
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &script, query, globalOrTeamID); err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("SetupExperienceScript"), "get setup experience script")
		}
		return nil, ctxerr.Wrap(ctx, err, "get setup experience script")
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

		if IsDuplicate(err) {
			// already exists for this team/no team
			err = &existsError{ResourceType: "SetupExperienceScript", TeamID: &globalOrTeamID}
		} else if isChildForeignKeyError(err) {
			// team does not exist
			err = foreignKey("setup_experience_scripts", fmt.Sprintf("team_id=%v", script.TeamID))
		}
		return nil, ctxerr.Wrap(ctx, err, "insert setup experience script")
	}

	return res, nil
}

func (ds *Datastore) DeleteSetupExperienceScript(ctx context.Context, teamID *uint) error {
	var globalOrTeamID uint
	if teamID != nil {
		globalOrTeamID = *teamID
	}

	_, err := ds.writer(ctx).ExecContext(ctx, `DELETE FROM setup_experience_scripts WHERE global_or_team_id = ?`, globalOrTeamID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "delete setup experience script")
	}

	// NOTE: CleanupUnusedScriptContents is responsible for removing any orphaned script_contents
	// for setup experience scripts.

	return nil
}
