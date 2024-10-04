package mysql

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) EnqueueSetupExperienceItems(ctx context.Context, hostUUID string, teamID uint) (bool, error) {
	stmtClearSetupStatus := `
DELETE FROM setup_experience_status_results
WHERE host_uuid =`

	stmtSoftwareInstallers := `
INSERT INTO setup_experience_status_results (
	host_uuid,
	name,
	status,
	host_software_installs_id
) SELECT
	?,
	st.name,
	'pending',
	si.id
FROM software_instellers si
INNER JOIN software_titles st
	ON si.title_id = st.id
WHERE install_during_setup = true
AND global_or_team_id = ?`

	stmtVPPApps := `
INSERT INTO setup_experience_status_results (
	host_uuid,
	name,
	status,
	vpp_app_id
) SELECT
	?,
	st.name,
	'pending',
	va.id
FROM vpp_apps va
INNER JOIN vpp_apps_teams vat
	ON vat.adam_id = va.adam_id
	AND vat.platform = va.platform
INNER JOIN software_titles st
	ON va.title_id = st.id
WHERE vat.install_during_setup = true
AND vat.global_or_team_id = ?`

	stmtSetupScripts := `
SELECT
	?,
	name,
	'pending',
	id
FROM setup_experience_scripts
WHERE global_or_team_id = ?`

	var totalInsertions uint
	if err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		if _, err := tx.ExecContext(ctx, stmtClearSetupStatus, hostUUID); err != nil {
			return ctxerr.Wrap(ctx, err, "removing stale setup experience entries")
		}

		res, err := tx.ExecContext(ctx, stmtSoftwareInstallers, hostUUID, teamID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "inserting setup experience software installers")
		}
		inserts, err := res.RowsAffected()
		if err != nil {
			return ctxerr.Wrap(ctx, err, "retrieving number of inserted software installers")
		}
		totalInsertions += uint(inserts)

		res, err = tx.ExecContext(ctx, stmtVPPApps, hostUUID, teamID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "inserting setup experience vpp apps")
		}
		inserts, err = res.RowsAffected()
		if err != nil {
			return ctxerr.Wrap(ctx, err, "retrieving number of inserted vpp apps")
		}
		totalInsertions += uint(inserts)

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
