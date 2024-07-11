package mysql

import (
	"context"
	"fmt"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) BatchInsertVPPApps(ctx context.Context, apps []*fleet.VPPApp) error {
	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		if err := insertVPPApps(ctx, tx, apps); err != nil {
			return ctxerr.Wrap(ctx, err, "BatchInsertVPPApps insertVPPApps transaction")
		}

		return nil
	})
}

func (ds *Datastore) InsertVPPAppWithTeam(ctx context.Context, app *fleet.VPPApp, teamID *uint) error {
	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		titleID, err := insertSoftwareTitleForVPPApp(ctx, tx, app)
		if err != nil {
			return err
		}

		app.TitleID = titleID

		if err := insertVPPApps(ctx, tx, []*fleet.VPPApp{app}); err != nil {
			return ctxerr.Wrap(ctx, err, "InsertVPPAppWithTeam insertVPPApps transaction")
		}

		if err := insertVPPAppTeams(ctx, tx, app.AdamID, teamID); err != nil {
			return ctxerr.Wrap(ctx, err, "InsertVPPAppWithTeam insertVPPAppTeams transaction")
		}

		return nil
	})
}

func (ds *Datastore) GetAssignedVPPApps(ctx context.Context, teamID *uint) (map[string]struct{}, error) {
	stmt := `
SELECT 
	adam_id
FROM 
	vpp_apps_teams vat
WHERE
	vat.global_or_team_id = ?
	`
	var tmID uint
	if teamID != nil {
		tmID = *teamID
	}

	var results []string
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &results, stmt, tmID); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get assigned VPP apps")
	}

	appSet := make(map[string]struct{})
	for _, r := range results {
		appSet[r] = struct{}{}
	}

	return appSet, nil
}

func insertVPPApps(ctx context.Context, tx sqlx.ExtContext, apps []*fleet.VPPApp) error {
	stmt := `
INSERT INTO vpp_apps
	(adam_id, available_count, bundle_identifier, icon_url, name, latest_version, title_id)
VALUES
%s
ON DUPLICATE KEY UPDATE
	updated_at = CURRENT_TIMESTAMP,
	latest_version = VALUES(latest_version),
	icon_url = VALUES(icon_url),
	name = VALUES(name)
	`
	var args []any
	var insertVals strings.Builder

	for _, a := range apps {
		insertVals.WriteString(`(?, ?, ?, ?, ?, ?, ?),`)
		args = append(args, a.AdamID, a.AvailableCount, a.BundleIdentifier, a.IconURL, a.Name, a.LatestVersion, a.TitleID)
	}

	stmt = fmt.Sprintf(stmt, strings.TrimSuffix(insertVals.String(), ","))

	_, err := tx.ExecContext(ctx, stmt, args...)

	return ctxerr.Wrap(ctx, err, "insert VPP apps")
}

func insertVPPAppTeams(ctx context.Context, tx sqlx.ExtContext, adamID string, teamID *uint) error {
	stmt := `
INSERT INTO vpp_apps_teams
	(adam_id, global_or_team_id, team_id)
VALUES
	(?, ?, ?)
	`

	var globalOrTmID uint
	if teamID != nil {
		globalOrTmID = *teamID
	}

	_, err := tx.ExecContext(ctx, stmt, adamID, globalOrTmID, teamID)

	return ctxerr.Wrap(ctx, err, "writing vpp app team mapping to db")
}

func insertSoftwareTitleForVPPApp(ctx context.Context, tx sqlx.ExtContext, app *fleet.VPPApp) (uint, error) {
	stmt := `INSERT INTO software_titles (name, source, bundle_identifier, browser) VALUES (?, '', ?, '')`

	result, err := tx.ExecContext(ctx, stmt, app.Name, app.BundleIdentifier)
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "writing vpp app software title")
	}

	id, _ := result.LastInsertId()

	return uint(id), nil
}
