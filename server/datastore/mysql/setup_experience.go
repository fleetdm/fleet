package mysql

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) SetSetupExperienceSoftwareTitles(ctx context.Context, teamID uint, softwareTitleIDs []uint) error {
	titleIDQuestionMarks := strings.Join(slices.Repeat([]string{"?"}, len(softwareTitleIDs)), ",")

	stmtSelectInstallersIDs := fmt.Sprintf(`
SELECT
	si.id
FROM
	software_titles st
LEFT JOIN
	software_installers si
	ON st.id = si.title_id
WHERE
	global_or_team_id = ?
AND
	st.id IN (%s)
`, softwareTitleIDs)

	stmtSelectVPPAppsTeamsID := fmt.Sprintf(`
SELECT
	vat.id
FROM
	software_titles st
LEFT JOIN
	vpp_apps va
	ON st.id = va.title_id
LEFT JOIN
	vpp_apps_teams vat
	ON va.adam_id = vat.adam_id
WHERE
	global_or_team_id = ?
AND
	st.id IN (%s)
`, titleIDQuestionMarks)

	stmtUnsetInstallers := `
UPDATE software_installers
SET install_during_setup = false
WHERE team_or_global_id = ?`

	stmtUnsetVPPAppsTeams := `
UPDATE vpp_apps_teams vat
SET install_during_setup = false
WHERE team_or_global_id = ?`

	stmtSetInstallers := `
UPDATE software_installers
SET install_during_setup = true
WHERE id IN (%s)`

	stmtSetVPPAppsTeams := `
UPDATE vpp_apps_teams
SET install_during_setup = true
WHERE id IN (%s)`

	if err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		var softwareIDs []uint
		var vppAppTeamIDs []uint
		titleIDs := make([]any, 0, len(softwareTitleIDs))
		for _, id := range softwareTitleIDs {
			titleIDs = append(titleIDs, id)
		}

		if err := sqlx.SelectContext(ctx, tx, &softwareIDs, stmtSelectInstallersIDs, titleIDs...); err != nil {
			return ctxerr.Wrap(ctx, err, "selecting software IDs using title IDs")
		}

		if err := sqlx.SelectContext(ctx, tx, &vppAppTeamIDs, stmtSelectVPPAppsTeamsID, titleIDs...); err != nil {
			return ctxerr.Wrap(ctx, err, "selecting vpp app team IDs using title IDs")
		}

		if _, err := tx.ExecContext(ctx, stmtUnsetInstallers); err != nil {
			return ctxerr.Wrap(ctx, err, "unsetting software installers")
		}

		if _, err := tx.ExecContext(ctx, stmtUnsetVPPAppsTeams); err != nil {
			return ctxerr.Wrap(ctx, err, "unsetting vpp app teams")
		}

		if _, err := tx.ExecContext(ctx, stmtSetInstallers); err != nil {
			return ctxerr.Wrap(ctx, err, "setting software installers")
		}

		if _, err := tx.ExecContext(ctx, stmtSetVPPAppsTeams); err != nil {
			return ctxerr.Wrap(ctx, err, "setting vpp app teams")
		}

		return nil
	}); err != nil {
		return ctxerr.Wrap(ctx, err, "setting setup experience software")
	}

	return nil
}

// func (ds *Datastore) ListSetupExperienceSoftwareTitles(ctx context.Context, teamID uint) ([]string, error) {
// 	return nil, nil
// }
