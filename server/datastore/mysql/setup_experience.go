package mysql

import (
	"context"

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
		totalInsertions += uint(inserts)

		// VPP apps
		res, err = tx.ExecContext(ctx, stmtVPPApps, hostUUID, teamID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "inserting setup experience vpp apps")
		}
		inserts, err = res.RowsAffected()
		if err != nil {
			return ctxerr.Wrap(ctx, err, "retrieving number of inserted vpp apps")
		}
		totalInsertions += uint(inserts)

		// Scripts
		res, err = tx.ExecContext(ctx, stmtSetupScripts, hostUUID, teamID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "inserting setup experience scripts")
		}
		inserts, err = res.RowsAffected()
		if err != nil {
			return ctxerr.Wrap(ctx, err, "retrieving number of inserted setup experience scripts")
		}
		totalInsertions += uint(inserts)

		return nil
	}); err != nil {
		return false, ctxerr.Wrap(ctx, err, "enqueue setup experience")
	}

	return totalInsertions > 0, nil
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
