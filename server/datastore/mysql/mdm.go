package mysql

import (
	"context"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) GetMDMCommandPlatform(ctx context.Context, commandUUID string) (string, error) {
	stmt := `
SELECT CASE
	WHEN EXISTS (SELECT 1 FROM nano_commands WHERE command_uuid = ?) THEN 'darwin'
	WHEN EXISTS (SELECT 1 FROM windows_mdm_commands WHERE command_uuid = ?) THEN 'windows'
	ELSE ''
END AS platform
`

	var p string
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &p, stmt, commandUUID, commandUUID); err != nil {
		return "", err
	}
	if p == "" {
		return "", ctxerr.Wrap(ctx, notFound("MDMCommand").WithName(commandUUID))
	}

	return p, nil
}

func (ds *Datastore) ListMDMCommands(
	ctx context.Context,
	tmFilter fleet.TeamFilter,
	listOpts *fleet.MDMCommandListOptions,
) ([]*fleet.MDMCommand, error) {
	appleStmt := `
SELECT
    nvq.id as host_uuid,
    nvq.command_uuid,
    COALESCE(NULLIF(nvq.status, ''), 'Pending') as status,
    COALESCE(nvq.result_updated_at, nvq.created_at) as updated_at,
    nvq.request_type,
    h.hostname,
    h.team_id
FROM
    nano_view_queue nvq
INNER JOIN
    hosts h
ON
    nvq.id = h.uuid
WHERE
   nvq.active = 1
`

	// TODO(roberto): note how we're not filtering by `active = 1` on windows.
	// Nano behaves like this:
	// - When a device is unenrolled, all pending commands are set `active = 0`
	// - When a command is delivered, the entry is removed from the table
	//
	// In Windows, we set `active = 0` when we get the results of the command. What we should do?
	windowsStmt := `
SELECT
    mwe.host_uuid,
    wmc.command_uuid,
    COALESCE(NULLIF(wmcr.status_code, ''), 'Pending') as status,
    COALESCE(wmc.updated_at, wmc.created_at) as updated_at,
    wmc.target_loc_uri as request_type,
    h.hostname,
    h.team_id
FROM windows_mdm_command_queue wmcq
INNER JOIN mdm_windows_enrollments mwe ON wmcq.enrollment_id = mwe.id
INNER JOIN hosts h ON h.uuid = mwe.host_uuid
INNER JOIN windows_mdm_commands wmc ON wmc.command_uuid = wmcq.command_uuid
LEFT JOIN windows_mdm_command_results wmcr ON wmcr.command_uuid = wmcq.command_uuid
`

	jointStmt := fmt.Sprintf(
		`SELECT * FROM ((%s) UNION ALL (%s)) as combined_commands WHERE %s`,
		appleStmt, windowsStmt, ds.whereFilterHostsByTeams(tmFilter, "h"),
	)
	jointStmt, params := appendListOptionsWithCursorToSQL(jointStmt, nil, &listOpts.ListOptions)
	var results []*fleet.MDMCommand
	fmt.Println(jointStmt)
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &results, jointStmt, params...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list commands")
	}
	return results, nil
}
