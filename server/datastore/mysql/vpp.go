package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) GetTeamAppleSerialNumbers(ctx context.Context, teamID uint) ([]string, error) {
	stmt := `
SELECT
  hardware_serial
FROM
  hosts
WHERE
  platform = 'darwin'
AND
  hardware_serial != ''
AND
  team_id = ?
`

	var serialNumbers []string
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &serialNumbers, stmt, teamID); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "unable to retrieve team serial numbers")
	}

	return serialNumbers, nil
}

func (ds *Datastore) GetVPPAppMetadataByTeamAndTitleID(ctx context.Context, teamID *uint, titleID uint) (*fleet.VPPAppStoreApp, error) {
	const query = `
SELECT
	vap.adam_id,
	vap.name,
	vap.latest_version,
	NULLIF(vap.icon_url, '') AS icon_url
FROM
	vpp_apps vap
	INNER JOIN vpp_apps_teams vat ON vat.adam_id = vap.adam_id
WHERE
	vap.title_id = ? AND
	vat.global_or_team_id = ?`

	var tmID uint
	if teamID != nil {
		tmID = *teamID
	}

	var app fleet.VPPAppStoreApp
	err := sqlx.GetContext(ctx, ds.reader(ctx), &app, query, titleID, tmID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("VPPApp"), "get VPP app metadata")
		}
		return nil, ctxerr.Wrap(ctx, err, "get VPP app metadata")
	}

	return &app, nil
}

func (ds *Datastore) GetSummaryHostVPPAppInstalls(ctx context.Context, teamID *uint, adamID string) (*fleet.VPPAppStatusSummary, error) {
	var dest fleet.VPPAppStatusSummary

	stmt := fmt.Sprintf(`
SELECT
	COALESCE(SUM( IF(status = :software_status_pending, 1, 0)), 0) AS pending,
	COALESCE(SUM( IF(status = :software_status_failed, 1, 0)), 0) AS failed,
	COALESCE(SUM( IF(status = :software_status_installed, 1, 0)), 0) AS installed
FROM (
SELECT
	%s
FROM
	host_vpp_software_installs hvsi
INNER JOIN
	hosts h ON hvsi.host_id = h.id
LEFT OUTER JOIN
	nano_command_results ncr ON ncr.id = h.uuid AND ncr.command_uuid = hvsi.command_uuid
WHERE
	hvsi.adam_id = :adam_id AND
	(h.team_id = :team_id OR (h.team_id IS NULL AND :team_id = 0)) AND
	hvsi.id IN (
		SELECT
			max(hvsi2.id) -- ensure we use only the most recently created install attempt for each host
		FROM
			host_vpp_software_installs hvsi2
		WHERE
			hvsi2.adam_id = :adam_id
		GROUP BY
			hvsi2.host_id
	)
) s`, vppAppHostStatusNamedQuery("hvsi", "ncr", "status"))

	var tmID uint
	if teamID != nil {
		tmID = *teamID
	}

	query, args, err := sqlx.Named(stmt, map[string]interface{}{
		"adam_id":                   adamID,
		"team_id":                   tmID,
		"mdm_status_acknowledged":   fleet.MDMAppleStatusAcknowledged,
		"mdm_status_error":          fleet.MDMAppleStatusError,
		"mdm_status_format_error":   fleet.MDMAppleStatusCommandFormatError,
		"software_status_pending":   fleet.SoftwareInstallerPending,
		"software_status_failed":    fleet.SoftwareInstallerFailed,
		"software_status_installed": fleet.SoftwareInstallerInstalled,
	})
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get summary host vpp installs: named query")
	}

	err = sqlx.GetContext(ctx, ds.reader(ctx), &dest, query, args...)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get summary host vpp install status")
	}
	return &dest, nil
}

// hvsiAlias is the table alias to use as prefix for the
// host_vpp_software_installs column names, no prefix used if empty.
// ncrAlias is the table alias to use as prefix for the nano_command_results
// column names, no prefix used if empty.
// colAlias is the name to be assigned to the computed status column, pass
// empty to have the value only, no column alias set.
func vppAppHostStatusNamedQuery(hvsiAlias, ncrAlias, colAlias string) string {
	if hvsiAlias != "" {
		hvsiAlias += "."
	}
	if ncrAlias != "" {
		ncrAlias += "."
	}
	if colAlias != "" {
		colAlias = " AS " + colAlias
	}
	return fmt.Sprintf(`
			CASE
				WHEN %[1]sstatus = :mdm_status_acknowledged THEN
					:software_status_installed
				WHEN %[1]sstatus = :mdm_status_error OR %[1]sstatus = :mdm_status_format_error THEN
					:software_status_failed
				WHEN %[2]sid IS NOT NULL THEN
					:software_status_pending
				ELSE
					NULL -- not installed via VPP App
			END %[3]s `, ncrAlias, hvsiAlias, colAlias)
}

func (ds *Datastore) BatchInsertVPPApps(ctx context.Context, apps []*fleet.VPPApp) error {
	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		if err := insertVPPApps(ctx, tx, apps); err != nil {
			return ctxerr.Wrap(ctx, err, "BatchInsertVPPApps insertVPPApps transaction")
		}

		return nil
	})
}

func (ds *Datastore) SetTeamVPPApps(ctx context.Context, teamID *uint, adamIDs []string) error {
	existingApps, err := ds.GetAssignedVPPApps(ctx, teamID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "SetTeamVPPApps getting list of existing apps")
	}

	var missingApps []string
	var toRemoveApps []string

	for existingApp := range existingApps {
		var found bool
		for _, adamID := range adamIDs {
			if adamID == existingApp {
				found = true
			}
		}
		if !found {
			toRemoveApps = append(toRemoveApps, existingApp)
		}
	}

	for _, adamID := range adamIDs {
		if _, ok := existingApps[adamID]; !ok {
			missingApps = append(missingApps, adamID)
		}
	}

	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		for _, toAdd := range missingApps {
			if err := insertVPPAppTeams(ctx, tx, toAdd, teamID); err != nil {
				return ctxerr.Wrap(ctx, err, "SetTeamVPPApps inserting vpp app into team")
			}
		}

		for _, toRemove := range toRemoveApps {
			if err := removeVPPAppTeams(ctx, tx, toRemove, teamID); err != nil {
				return ctxerr.Wrap(ctx, err, "SetTeamVPPApps removing vpp app from team")
			}
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

func removeVPPAppTeams(ctx context.Context, tx sqlx.ExtContext, adamID string, teamID *uint) error {
	stmt := `
DELETE FROM
  vpp_apps_teams
WHERE
  adam_id = ?
AND
  team_id = ?
`
	_, err := tx.ExecContext(ctx, stmt, adamID, teamID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "deleting vpp app from team")
	}

	return nil
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
