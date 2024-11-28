package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) EnqueueSetupExperienceItems(ctx context.Context, hostUUID string, teamID uint) (bool, error) {
	stmtClearSetupStatus := `
DELETE FROM setup_experience_status_results
WHERE host_uuid = ?`

	stmtSoftwareInstallers := `
INSERT INTO setup_experience_status_results (
	host_uuid,
	name,
	status,
	software_installer_id
) SELECT
	?,
	st.name,
	'pending',
	si.id
FROM software_installers si
INNER JOIN software_titles st
	ON si.title_id = st.id
WHERE install_during_setup = true
AND global_or_team_id = ?`

	stmtVPPApps := `
INSERT INTO setup_experience_status_results (
	host_uuid,
	name,
	status,
	vpp_app_team_id
) SELECT
	?,
	st.name,
	'pending',
	vat.id
FROM vpp_apps va
INNER JOIN vpp_apps_teams vat
	ON vat.adam_id = va.adam_id
	AND vat.platform = va.platform
INNER JOIN software_titles st
	ON va.title_id = st.id
WHERE vat.install_during_setup = true
AND vat.global_or_team_id = ?`

	stmtSetupScripts := `
INSERT INTO setup_experience_status_results (
	host_uuid,
	name,
	status,
	setup_experience_script_id
) SELECT
	?,
	name,
	'pending',
	id
FROM setup_experience_scripts
WHERE global_or_team_id = ?`

	var totalInsertions uint
	if err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		// Clean out old statuses for the host
		if _, err := tx.ExecContext(ctx, stmtClearSetupStatus, hostUUID); err != nil {
			return ctxerr.Wrap(ctx, err, "removing stale setup experience entries")
		}

		// Software installers
		res, err := tx.ExecContext(ctx, stmtSoftwareInstallers, hostUUID, teamID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "inserting setup experience software installers")
		}
		inserts, err := res.RowsAffected()
		if err != nil {
			return ctxerr.Wrap(ctx, err, "retrieving number of inserted software installers")
		}
		totalInsertions += uint(inserts) // nolint: gosec

		// VPP apps
		res, err = tx.ExecContext(ctx, stmtVPPApps, hostUUID, teamID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "inserting setup experience vpp apps")
		}
		inserts, err = res.RowsAffected()
		if err != nil {
			return ctxerr.Wrap(ctx, err, "retrieving number of inserted vpp apps")
		}
		totalInsertions += uint(inserts) // nolint: gosec

		// Scripts
		res, err = tx.ExecContext(ctx, stmtSetupScripts, hostUUID, teamID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "inserting setup experience scripts")
		}
		inserts, err = res.RowsAffected()
		if err != nil {
			return ctxerr.Wrap(ctx, err, "retrieving number of inserted setup experience scripts")
		}
		totalInsertions += uint(inserts) // nolint: gosec

		// Only run setup experience on hosts that have something configured.
		if totalInsertions > 0 {
			if err := setHostAwaitingConfiguration(ctx, tx, hostUUID, true); err != nil {
				return ctxerr.Wrap(ctx, err, "setting host awaiting configuration to true")
			}
		}

		return nil
	}); err != nil {
		return false, ctxerr.Wrap(ctx, err, "enqueue setup experience")
	}

	return totalInsertions > 0, nil
}

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
		// List of title IDs that were sent but aren't in the
		// database. We add everything and then remove them
		// from the list when we validate them below
		missingTitleIDs := make(map[uint]struct{})
		// Arguments used for queries that select vpp apps/installers
		titleIDAndTeam := []any{teamID}
		for _, id := range titleIDs {
			missingTitleIDs[id] = struct{}{}
			titleIDAndTeam = append(titleIDAndTeam, id)
		}

		// Select requested software installers
		if len(titleIDs) > 0 {
			if err := sqlx.SelectContext(ctx, tx, &softwareIDPlatforms, stmtSelectInstallersIDs, titleIDAndTeam...); err != nil {
				return ctxerr.Wrap(ctx, err, "selecting software IDs using title IDs")
			}
		}

		// Validate only macOS software
		for _, tuple := range softwareIDPlatforms {
			delete(missingTitleIDs, tuple.TitleID)
			if tuple.Platform != string(fleet.MacOSPlatform) {
				return ctxerr.Errorf(ctx, "only MacOS supported, unsupported software installer: %d (%s, %s)", tuple.ID, tuple.Name, tuple.Platform)
			}
			softwareIDs = append(softwareIDs, tuple.ID)
		}

		// Select requested VPP apps
		if len(titleIDs) > 0 {
			if err := sqlx.SelectContext(ctx, tx, &vppIDPlatforms, stmtSelectVPPAppsTeamsID, titleIDAndTeam...); err != nil {
				return ctxerr.Wrap(ctx, err, "selecting vpp app team IDs using title IDs")
			}
		}

		// Validate only macOS VPPP apps
		for _, tuple := range vppIDPlatforms {
			delete(missingTitleIDs, tuple.TitleID)
			if tuple.Platform != string(fleet.MacOSPlatform) {
				return ctxerr.Errorf(ctx, "only MacOS supported, unsupported AppStoreApp title: %d (%s, %s)", tuple.ID, tuple.Name, tuple.Platform)
			}
			vppAppTeamIDs = append(vppAppTeamIDs, tuple.ID)
		}

		// If we have any missing titles, return error
		if len(missingTitleIDs) > 0 {
			var keys []string
			for k := range missingTitleIDs {
				keys = append(keys, fmt.Sprintf("%d", k))
			}
			return ctxerr.Errorf(ctx, "title IDs not available: %s", strings.Join(keys, ","))
		}

		// Unset all installers
		if _, err := tx.ExecContext(ctx, stmtUnsetInstallers, teamID); err != nil {
			return ctxerr.Wrap(ctx, err, "unsetting software installers")
		}

		// Unset all vpp apps
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
		AvailableForInstall: true,
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
	sesr.host_software_installs_execution_id,
	sesr.vpp_app_team_id,
	sesr.nano_command_uuid,
	sesr.setup_experience_script_id,
	sesr.script_execution_id,
	sesr.error,
	NULLIF(va.adam_id, '') AS vpp_app_adam_id,
	NULLIF(va.platform, '') AS vpp_app_platform,
	ses.script_content_id,
	COALESCE(si.title_id, COALESCE(va.title_id, NULL)) AS software_title_id
FROM setup_experience_status_results sesr
LEFT JOIN setup_experience_scripts ses ON ses.id = sesr.setup_experience_script_id
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

func (ds *Datastore) UpdateSetupExperienceStatusResult(ctx context.Context, status *fleet.SetupExperienceStatusResult) error {
	const stmt = `
UPDATE setup_experience_status_results
SET
	host_uuid = ?,
	name = ?,
	status = ?,
	software_installer_id = ?,
	host_software_installs_execution_id = ?,
	vpp_app_team_id = ?,
	nano_command_uuid = ?,
	setup_experience_script_id = ?,
	script_execution_id = ?,
	error = ?
WHERE id = ?
`
	if err := status.IsValid(); err != nil {
		return ctxerr.Wrap(ctx, err, "invalid status update")
	}

	if _, err := ds.writer(ctx).ExecContext(
		ctx,
		stmt,
		status.HostUUID,
		status.Name,
		status.Status,
		status.SoftwareInstallerID,
		status.HostSoftwareInstallsExecutionID,
		status.VPPAppTeamID,
		status.NanoCommandUUID,
		status.SetupExperienceScriptID,
		status.ScriptExecutionID,
		status.Error,
		status.ID,
	); err != nil {
		return ctxerr.Wrap(ctx, err, "updating setup experience status result")
	}

	return nil
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

func (ds *Datastore) GetSetupExperienceScriptByID(ctx context.Context, scriptID uint) (*fleet.Script, error) {
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
  id = ?
`

	var script fleet.Script
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &script, query, scriptID); err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("SetupExperienceScript"), "get setup experience script by id")
		}
		return nil, ctxerr.Wrap(ctx, err, "get setup experience script by id")
	}

	return &script, nil
}

func (ds *Datastore) SetSetupExperienceScript(ctx context.Context, script *fleet.Script) error {
	err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		var err error

		// first insert script contents
		scRes, err := insertScriptContents(ctx, tx, script.ScriptContents)
		if err != nil {
			return err
		}
		id, _ := scRes.LastInsertId()

		// then create the script entity
		_, err = insertSetupExperienceScript(ctx, tx, script, uint(id)) // nolint: gosec
		return err
	})

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

func (ds *Datastore) SetHostAwaitingConfiguration(ctx context.Context, hostUUID string, awaitingConfiguration bool) error {
	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		return setHostAwaitingConfiguration(ctx, tx, hostUUID, awaitingConfiguration)
	})
}

func setHostAwaitingConfiguration(ctx context.Context, tx sqlx.ExtContext, hostUUID string, awaitingConfiguration bool) error {
	const stmt = `
INSERT INTO host_mdm_apple_awaiting_configuration (host_uuid, awaiting_configuration)
VALUES (?, ?)
ON DUPLICATE KEY UPDATE
	awaiting_configuration = VALUES(awaiting_configuration)
	`

	_, err := tx.ExecContext(ctx, stmt, hostUUID, awaitingConfiguration)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "setting host awaiting configuration")
	}

	return nil
}

func (ds *Datastore) GetHostAwaitingConfiguration(ctx context.Context, hostUUID string) (bool, error) {
	const stmt = `
SELECT
	awaiting_configuration
FROM host_mdm_apple_awaiting_configuration
WHERE host_uuid = ?
	`
	var awaitingConfiguration bool

	if err := sqlx.GetContext(ctx, ds.reader(ctx), &awaitingConfiguration, stmt, hostUUID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, notFound("HostAwaitingConfiguration")
		}

		return false, ctxerr.Wrap(ctx, err, "getting host awaiting configuration")
	}

	return awaitingConfiguration, nil
}

func (ds *Datastore) MaybeUpdateSetupExperienceVPPStatus(ctx context.Context, hostUUID string, nanoCommandUUID string, status fleet.SetupExperienceStatusResultStatus) (bool, error) {
	selectStmt := "SELECT id FROM setup_experience_status_results WHERE host_uuid = ? AND nano_command_uuid = ?"
	updateStmt := "UPDATE setup_experience_status_results SET status = ? WHERE id = ?"

	var id uint
	if err := ds.writer(ctx).GetContext(ctx, &id, selectStmt, hostUUID, nanoCommandUUID); err != nil {
		// TODO: maybe we can use the reader instead for this query
		if errors.Is(err, sql.ErrNoRows) {
			// return early if no results found
			return false, nil
		}
		return false, err
	}
	res, err := ds.writer(ctx).ExecContext(ctx, updateStmt, status, id)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()

	return n > 0, nil
}

func (ds *Datastore) MaybeUpdateSetupExperienceSoftwareInstallStatus(ctx context.Context, hostUUID string, executionID string, status fleet.SetupExperienceStatusResultStatus) (bool, error) {
	selectStmt := "SELECT id FROM setup_experience_status_results WHERE host_uuid = ? AND host_software_installs_execution_id = ?"
	updateStmt := "UPDATE setup_experience_status_results SET status = ? WHERE id = ?"

	var id uint
	if err := ds.writer(ctx).GetContext(ctx, &id, selectStmt, hostUUID, executionID); err != nil {
		// TODO: maybe we can use the reader instead for this query
		if errors.Is(err, sql.ErrNoRows) {
			// return early if no results found
			return false, nil
		}
		return false, err
	}
	res, err := ds.writer(ctx).ExecContext(ctx, updateStmt, status, id)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()

	return n > 0, nil
}

func (ds *Datastore) MaybeUpdateSetupExperienceScriptStatus(ctx context.Context, hostUUID string, executionID string, status fleet.SetupExperienceStatusResultStatus) (bool, error) {
	selectStmt := "SELECT id FROM setup_experience_status_results WHERE host_uuid = ? AND script_execution_id = ?"
	updateStmt := "UPDATE setup_experience_status_results SET status = ? WHERE id = ?"

	var id uint
	if err := ds.writer(ctx).GetContext(ctx, &id, selectStmt, hostUUID, executionID); err != nil {
		// TODO: maybe we can use the reader instead for this query
		if errors.Is(err, sql.ErrNoRows) {
			// return early if no results found
			return false, nil
		}
		return false, err
	}
	res, err := ds.writer(ctx).ExecContext(ctx, updateStmt, status, id)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()

	return n > 0, nil
}
