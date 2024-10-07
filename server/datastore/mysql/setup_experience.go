package mysql

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

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
