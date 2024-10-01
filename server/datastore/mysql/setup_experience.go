package mysql

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) SetSetupExperienceSoftwareTitles(ctx context.Context, teamID uint, titleIDs []uint) error {
	titleIDQuestionMarks := strings.Join(slices.Repeat([]string{"?"}, len(titleIDs)), ",")

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
`, titleIDQuestionMarks)

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
		titleIDArgs := make([]any, 0, len(titleIDs))
		for _, id := range titleIDs {
			titleIDs = append(titleIDs, id)
		}

		if err := sqlx.SelectContext(ctx, tx, &softwareIDs, stmtSelectInstallersIDs, titleIDArgs...); err != nil {
			return ctxerr.Wrap(ctx, err, "selecting software IDs using title IDs")
		}

		if err := sqlx.SelectContext(ctx, tx, &vppAppTeamIDs, stmtSelectVPPAppsTeamsID, titleIDArgs...); err != nil {
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

func (ds *Datastore) ListSetupExperienceSoftwareTitles(ctx context.Context, teamID uint, opts fleet.ListOptions) ([]fleet.SoftwareTitleListResult, int, *fleet.PaginationMetadata, error) {
	opts.IncludeMetadata = true
	opts.After = ""

	vc, ok := viewer.FromContext(ctx)
	if !ok {
		return nil, 0, nil, fleet.ErrNoContext
	}

	titles, count, meta, err := ds.ListSoftwareTitles(ctx, fleet.SoftwareTitleListOptions{
		TeamID:              &teamID,
		ListOptions:         opts,
		Platform:            string(fleet.MacOSPlatform),
		SetupExperienceOnly: true,
	}, fleet.TeamFilter{
		User:            vc.User,
		IncludeObserver: true,
		TeamID:          &teamID,
	})

	if err != nil {
		return nil, 0, nil, ctxerr.Wrap(ctx, err, "calling list software titles")
	}

	return titles, count, meta, nil
}
