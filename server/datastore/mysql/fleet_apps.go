package mysql

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) ListAvailableFleetMaintainedApps(ctx context.Context, teamID uint, opt *fleet.ListOptions) ([]fleet.FleetMaintainedAppAvailable, *fleet.PaginationMetadata, error) {
	stmt := `
SELECT
	fla.id,
	fla.name,
	fla.version,
	fla.platform
FROM
	fleet_library_apps fla
LEFT JOIN
	software_titles st ON fla.name = st.name
LEFT JOIN
	software_installers si ON si.title_id = st.id AND si.global_or_team_id = ?
LEFT JOIN
	vpp_apps va ON va.title_id = st.id
LEFT JOIN
	vpp_apps_teams vat ON va.adam_id = vat.adam_id AND va.platform = vat.platform AND vat.global_or_team_id = ?
WHERE
	st.name IS NULL
`
	stmt, args := appendListOptionsWithCursorToSQL(stmt, []any{teamID, teamID}, opt)

	var avail []fleet.FleetMaintainedAppAvailable
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &avail, stmt, args...); err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "selecting available fleet managed apps")
	}

	var meta *fleet.PaginationMetadata
	meta = &fleet.PaginationMetadata{HasPreviousResults: opt.Page > 0}
	if len(avail) > int(opt.PerPage) {
		meta.HasNextResults = true
		avail = avail[:len(avail)-1]
	}

	return avail, meta, nil
}
