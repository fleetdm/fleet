package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/jmoiron/sqlx"
)

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
		for _, app := range apps {
			titleID, err := ds.getOrInsertSoftwareTitleForVPPApp(ctx, tx, app)
			if err != nil {
				return err
			}

			app.TitleID = titleID

			if err := insertVPPApps(ctx, tx, []*fleet.VPPApp{app}); err != nil {
				return ctxerr.Wrap(ctx, err, "BatchInsertVPPApps insertVPPApps transaction")
			}
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

func (ds *Datastore) InsertVPPAppWithTeam(ctx context.Context, app *fleet.VPPApp, teamID *uint) (*fleet.VPPApp, error) {
	err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		titleID, err := ds.getOrInsertSoftwareTitleForVPPApp(ctx, tx, app)
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
	if err != nil {
		return nil, err
	}

	return app, nil
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
	(adam_id, bundle_identifier, icon_url, name, latest_version, title_id)
VALUES
%s
ON DUPLICATE KEY UPDATE
	updated_at = CURRENT_TIMESTAMP,
	latest_version = VALUES(latest_version),
	icon_url = VALUES(icon_url),
	name = VALUES(name),
	title_id = VALUES(title_id)
	`
	var args []any
	var insertVals strings.Builder

	for _, a := range apps {
		insertVals.WriteString(`(?, ?, ?, ?, ?, ?),`)
		args = append(args, a.AdamID, a.BundleIdentifier, a.IconURL, a.Name, a.LatestVersion, a.TitleID)
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

func (ds *Datastore) getOrInsertSoftwareTitleForVPPApp(ctx context.Context, tx sqlx.ExtContext, app *fleet.VPPApp) (uint, error) {
	// NOTE: it was decided to populate "apps" as the source for VPP apps for now, TBD
	// if this needs to change to better map to how software titles are reported
	// back by osquery. Since it may change, we're using a variable for the source.
	const source = "apps"

	selectStmt := `SELECT id FROM software_titles WHERE name = ? AND source = ? AND browser = ''`
	selectArgs := []any{app.Name, source}
	insertStmt := `INSERT INTO software_titles (name, source, browser) VALUES (?, ?, '')`
	insertArgs := []any{app.Name, source}

	if app.BundleIdentifier != "" {
		// NOTE: The index `idx_sw_titles` doesn't include the bundle
		// identifier. It's possible for the select to return nothing
		// but for the insert to fail if an app with the same name but
		// no bundle identifier exists in the DB.
		selectStmt = `SELECT id FROM software_titles WHERE bundle_identifier = ?`
		selectArgs = []any{app.BundleIdentifier}
		insertStmt = `INSERT INTO software_titles (name, source, bundle_identifier, browser) VALUES (?, ?, ?, '')`
		insertArgs = append(insertArgs, app.BundleIdentifier)
	}

	titleID, err := ds.optimisticGetOrInsertWithWriter(ctx,
		tx,
		&parameterizedStmt{
			Statement: selectStmt,
			Args:      selectArgs,
		},
		&parameterizedStmt{
			Statement: insertStmt,
			Args:      insertArgs,
		},
	)
	if err != nil {
		return 0, err
	}

	return titleID, nil
}

func (ds *Datastore) DeleteVPPAppFromTeam(ctx context.Context, teamID *uint, adamID string) error {
	const stmt = `DELETE FROM vpp_apps_teams WHERE global_or_team_id = ? AND adam_id = ?`

	var globalOrTeamID uint
	if teamID != nil {
		globalOrTeamID = *teamID
	}
	res, err := ds.writer(ctx).ExecContext(ctx, stmt, globalOrTeamID, adamID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "delete VPP app from team")
	}

	rows, _ := res.RowsAffected()
	if rows == 0 {
		return notFound("VPPApp").WithMessage(fmt.Sprintf("adam id %s for team id %d", adamID, globalOrTeamID))
	}
	return nil
}

func (ds *Datastore) GetVPPAppByTeamAndTitleID(ctx context.Context, teamID *uint, titleID uint, withScriptContents bool) (*fleet.VPPApp, error) {
	stmt := `
SELECT
  va.adam_id,
  va.bundle_identifier,
  va.icon_url,
  va.name,
  va.title_id,
  va.created_at,
  va.updated_at
FROM vpp_apps va
JOIN vpp_apps_teams vat ON va.adam_id = vat.adam_id
WHERE vat.global_or_team_id = ? AND va.title_id = ?
  `

	var tmID uint
	if teamID != nil {
		tmID = *teamID
	}

	var dest fleet.VPPApp
	err := sqlx.GetContext(ctx, ds.reader(ctx), &dest, stmt, tmID, titleID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("VPPApp"), "get VPP app")
		}
		return nil, ctxerr.Wrap(ctx, err, "get VPP app")
	}

	return &dest, nil
}

func (ds *Datastore) InsertHostVPPSoftwareInstall(ctx context.Context, hostID, userID uint, adamID, commandUUID, associatedEventID string) error {
	stmt := `
INSERT INTO host_vpp_software_installs
  (host_id, adam_id, command_uuid, user_id, associated_event_id)
VALUES
  (?,?,?,?,?)
	`

	if _, err := ds.writer(ctx).ExecContext(ctx, stmt, hostID, adamID, commandUUID, userID, associatedEventID); err != nil {
		return ctxerr.Wrap(ctx, err, "insert into host_vpp_software_installs")
	}

	return nil
}

func (ds *Datastore) GetPastActivityDataForVPPAppInstall(ctx context.Context, commandResults *mdm.CommandResults) (*fleet.User, *fleet.ActivityInstalledAppStoreApp, error) {
	if commandResults == nil {
		return nil, nil, nil
	}

	stmt := `
SELECT
	u.name AS user_name,
	u.id AS user_id,
	u.email as user_email,
	hvsi.host_id AS host_id,
	hdn.display_name AS host_display_name,
	st.name AS software_title,
	hvsi.adam_id AS app_store_id,
	hvsi.command_uuid AS command_uuid
FROM
	host_vpp_software_installs hvsi
	LEFT OUTER JOIN users u ON hvsi.user_id = u.id
	LEFT OUTER JOIN host_display_names hdn ON hdn.host_id = hvsi.host_id
	LEFT OUTER JOIN vpp_apps vpa ON hvsi.adam_id = vpa.adam_id
	LEFT OUTER JOIN software_titles st ON st.id = vpa.title_id
WHERE
	hvsi.command_uuid = :command_uuid
	`

	type result struct {
		HostID          uint   `db:"host_id"`
		HostDisplayName string `db:"host_display_name"`
		SoftwareTitle   string `db:"software_title"`
		AppStoreID      string `db:"app_store_id"`
		CommandUUID     string `db:"command_uuid"`
		UserName        string `db:"user_name"`
		UserID          uint   `db:"user_id"`
		UserEmail       string `db:"user_email"`
	}

	listStmt, args, err := sqlx.Named(stmt, map[string]any{
		"command_uuid":              commandResults.CommandUUID,
		"software_status_failed":    string(fleet.SoftwareInstallerFailed),
		"software_status_installed": string(fleet.SoftwareInstallerInstalled),
	})
	if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "build list query from named args")
	}

	var res result
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &res, listStmt, args...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, notFound("install_command")
		}

		return nil, nil, ctxerr.Wrap(ctx, err, "select past activity data for VPP app install")
	}

	user := &fleet.User{
		ID:    res.UserID,
		Name:  res.UserName,
		Email: res.UserEmail,
	}

	var status string
	switch commandResults.Status {
	case fleet.MDMAppleStatusAcknowledged:
		status = string(fleet.SoftwareInstallerInstalled)
	case fleet.MDMAppleStatusCommandFormatError:
	case fleet.MDMAppleStatusError:
		status = string(fleet.SoftwareInstallerFailed)
	default:
		// This case shouldn't happen (we should only be doing this check if the command is in a
		// "terminal" state, but adding it so we have a default
		status = string(fleet.SoftwareInstallerPending)
	}

	act := &fleet.ActivityInstalledAppStoreApp{
		HostID:          res.HostID,
		HostDisplayName: res.HostDisplayName,
		SoftwareTitle:   res.SoftwareTitle,
		AppStoreID:      res.AppStoreID,
		CommandUUID:     res.CommandUUID,
		Status:          status,
	}

	return user, act, nil
}
