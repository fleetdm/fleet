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
	host_uuid, 
	type, 
	name, 
	status, 
	host_software_installs_id, 
	nano_command_uuid, 
	script_execution_id, 
	error
FROM setup_experience_status_results
WHERE host_uuid = ?
	`
	var results []*fleet.SetupExperienceStatusResult
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &results, stmt, hostUUID); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "select setup experience status results by host uuid")
	}
	return results, nil
}
