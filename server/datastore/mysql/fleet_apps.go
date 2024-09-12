package mysql

import (
	"context"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) ListAvailableFleetMaintainedApps(ctx context.Context, teamID uint, page int, pageSize int) ([]fleet.FleetMaintainedAppAvailable, *fleet.PaginationMetadata, error) {
	stmtSelect := `
SELECT
	id,
	name,
	version,
	platform
FROM
	fleet_library_apps fla
WHERE
	name NOT IN (%s)
`

	stmtExisting := `
SELECT
	st.name
FROM
	software_titles st
LEFT JOIN
	software_installers si ON si.title_id = st.id
LEFT JOIN
	vpp_apps va ON va.title_id = st.id
LEFT JOIN
	vpp_apps_teams vat ON va.adam_id = vat.adam_id ANd va.platform = vat.platform
WHERE
	si.global_or_team_id = ?
AND
	vat.global_or_team_id = ?
`
	stmt := fmt.Sprintf(stmtSelect, stmtExisting)

	var avail []fleet.FleetMaintainedAppAvailable

	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &avail, stmt, teamID, teamID); err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "selecting available fleet managed apps")
	}

	return avail, nil, nil
}
