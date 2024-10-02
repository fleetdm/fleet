package mysql

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) SetSetupExperienceSoftwareTitles(ctx context.Context, teamID uint, titleIDs []uint) error {
	titleIDQuestionMarks := strings.Join(slices.Repeat([]string{"?"}, len(titleIDs)), ",")

	stmtSelectInstallersIDs := fmt.Sprintf(`
SELECT
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
		titleIDArgs := make([]any, 0, len(titleIDs))
		titleIDAndTeam := []any{teamID}
		for _, id := range titleIDs {
			titleIDArgs = append(titleIDArgs, id)
			titleIDAndTeam = append(titleIDAndTeam, id)
		}

		if err := sqlx.SelectContext(ctx, tx, &softwareIDPlatforms, stmtSelectInstallersIDs, titleIDAndTeam...); err != nil {
			return ctxerr.Wrap(ctx, err, "selecting software IDs using title IDs")
		}

		for _, tuple := range softwareIDPlatforms {
			if tuple.Platform != string(fleet.MacOSPlatform) {
				return ctxerr.Errorf(ctx, "only MacOS supported, unsupported software installer: %d (%s, %s)", tuple.ID, tuple.Name, tuple.Platform)
			}
			softwareIDs = append(softwareIDs, tuple.ID)
		}

		if err := sqlx.SelectContext(ctx, tx, &vppIDPlatforms, stmtSelectVPPAppsTeamsID, titleIDAndTeam...); err != nil {
			return ctxerr.Wrap(ctx, err, "selecting vpp app team IDs using title IDs")
		}

		for _, tuple := range vppIDPlatforms {
			if tuple.Platform != string(fleet.MacOSPlatform) {
				return ctxerr.Errorf(ctx, "only MacOS supported, unsupported AppStoreApp title: %d (%s, %s)", tuple.ID, tuple.Name, tuple.Platform)
			}
			vppAppTeamIDs = append(vppAppTeamIDs, tuple.ID)
		}

		if _, err := tx.ExecContext(ctx, stmtUnsetInstallers, teamID); err != nil {
			return ctxerr.Wrap(ctx, err, "unsetting software installers")
		}

		if _, err := tx.ExecContext(ctx, stmtUnsetVPPAppsTeams, teamID); err != nil {
			return ctxerr.Wrap(ctx, err, "unsetting vpp app teams")
		}

		if len(softwareIDs) == 0 {
			// 0 is not a valid ID, but it stops sql syntax error
			softwareIDs = append(softwareIDs, 0)
		}
		stmtSetInstallersLoop := fmt.Sprintf(stmtSetInstallers, questionMarks(len(softwareIDs)))
		if _, err := tx.ExecContext(ctx, stmtSetInstallersLoop, softwareIDs...); err != nil {
			return ctxerr.Wrap(ctx, err, "setting software installers")
		}

		if len(vppAppTeamIDs) == 0 {
			// 0 is not a valid ID, but it stops sql syntax error
			vppAppTeamIDs = append(vppAppTeamIDs, 0)
		}
		stmtSetVPPAppsTeamsLoop := fmt.Sprintf(stmtSetVPPAppsTeams, questionMarks(len(vppAppTeamIDs)))
		if _, err := tx.ExecContext(ctx, stmtSetVPPAppsTeamsLoop, vppAppTeamIDs...); err != nil {
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

	titles, count, meta, err := ds.ListSoftwareTitles(ctx, fleet.SoftwareTitleListOptions{
		TeamID:              &teamID,
		ListOptions:         opts,
		Platform:            string(fleet.MacOSPlatform),
		SetupExperienceOnly: true,
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
	Name     string `db:"name"`
	Platform string `db:"platform"`
}

func questionMarks(number int) string {
	return strings.Join(slices.Repeat([]string{"?"}, number), ",")
}
