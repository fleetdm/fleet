package mysql

import (
	"cmp"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) GetVPPAppMetadataByTeamAndTitleID(ctx context.Context, teamID *uint, titleID uint) (*fleet.VPPAppStoreApp, error) {
	const query = `
SELECT
	vap.adam_id,
	vap.platform,
	vap.name,
	vap.latest_version,
	vat.self_service,
	NULLIF(vap.icon_url, '') AS icon_url
FROM
	vpp_apps vap
	INNER JOIN vpp_apps_teams vat ON vat.adam_id = vap.adam_id AND vat.platform = vap.platform
WHERE
	vap.title_id = ? %s`

	// when team id is not nil, we need to filter by the global or team id given.
	args := []any{titleID}
	teamFilter := ""
	if teamID != nil {
		args = append(args, *teamID)
		teamFilter = "AND vat.global_or_team_id = ?"
	}

	var app fleet.VPPAppStoreApp
	err := sqlx.GetContext(ctx, ds.reader(ctx), &app, fmt.Sprintf(query, teamFilter), args...)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("VPPApp"), "get VPP app metadata")
		}
		return nil, ctxerr.Wrap(ctx, err, "get VPP app metadata")
	}

	return &app, nil
}

func (ds *Datastore) GetSummaryHostVPPAppInstalls(ctx context.Context, teamID *uint, appID fleet.VPPAppID) (*fleet.VPPAppStatusSummary,
	error,
) {
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
	hvsi.adam_id = :adam_id AND hvsi.platform = :platform AND
	(h.team_id = :team_id OR (h.team_id IS NULL AND :team_id = 0)) AND
	hvsi.id IN (
		SELECT
			max(hvsi2.id) -- ensure we use only the most recently created install attempt for each host
		FROM
			host_vpp_software_installs hvsi2
		WHERE
			hvsi2.adam_id = :adam_id AND hvsi2.platform = :platform
		GROUP BY
			hvsi2.host_id
	)
) s`, vppAppHostStatusNamedQuery("hvsi", "ncr", "status"))

	var tmID uint
	if teamID != nil {
		tmID = *teamID
	}

	query, args, err := sqlx.Named(stmt, map[string]interface{}{
		"adam_id":                   appID.AdamID,
		"platform":                  appID.Platform,
		"team_id":                   tmID,
		"mdm_status_acknowledged":   fleet.MDMAppleStatusAcknowledged,
		"mdm_status_error":          fleet.MDMAppleStatusError,
		"mdm_status_format_error":   fleet.MDMAppleStatusCommandFormatError,
		"software_status_pending":   fleet.SoftwareInstallPending,
		"software_status_failed":    fleet.SoftwareInstallFailed,
		"software_status_installed": fleet.SoftwareInstalled,
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

func (ds *Datastore) SetTeamVPPApps(ctx context.Context, teamID *uint, appFleets []fleet.VPPAppTeam) error {
	existingApps, err := ds.GetAssignedVPPApps(ctx, teamID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "SetTeamVPPApps getting list of existing apps")
	}

	// if we're batch-setting apps and replacing the ones installed during setup
	// in the same go, no need to validate that we don't delete one marked as
	// install during setup (since we're overwriting those). This is always
	// called from fleetctl gitops, so it should always be the case anyway.
	var replacingInstallDuringSetup bool
	if len(appFleets) == 0 || appFleets[0].InstallDuringSetup != nil {
		replacingInstallDuringSetup = true
	}

	var toAddApps []fleet.VPPAppTeam
	var toRemoveApps []fleet.VPPAppID

	for existingApp, appTeamInfo := range existingApps {
		var found bool
		for _, appFleet := range appFleets {
			// Self service value doesn't matter for removing app from team
			if existingApp == appFleet.VPPAppID {
				found = true
			}
		}
		if !found {
			// if app is marked as install during setup, prevent deletion unless we're replacing those.
			if !replacingInstallDuringSetup && appTeamInfo.InstallDuringSetup != nil && *appTeamInfo.InstallDuringSetup {
				return errDeleteInstallerInstalledDuringSetup
			}
			toRemoveApps = append(toRemoveApps, existingApp)
		}
	}

	for _, appFleet := range appFleets {
		// upsert it if it does not exist or SelfService or InstallDuringSetup flags are changed
		if existingFleet, ok := existingApps[appFleet.VPPAppID]; !ok || existingFleet.SelfService != appFleet.SelfService ||
			appFleet.InstallDuringSetup != nil &&
				existingFleet.InstallDuringSetup != nil && *appFleet.InstallDuringSetup != *existingFleet.InstallDuringSetup {
			toAddApps = append(toAddApps, appFleet)
		}
	}

	var vppToken *fleet.VPPTokenDB
	if len(appFleets) > 0 {
		vppToken, err = ds.GetVPPTokenByTeamID(ctx, teamID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "SetTeamVPPApps retrieve VPP token ID")
		}
	}

	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		for _, toAdd := range toAddApps {
			if err := insertVPPAppTeams(ctx, tx, toAdd, teamID, vppToken.ID); err != nil {
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
	vppToken, err := ds.GetVPPTokenByTeamID(ctx, teamID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "InsertVPPAppWithTeam unable to get VPP Token ID")
	}

	err = ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		titleID, err := ds.getOrInsertSoftwareTitleForVPPApp(ctx, tx, app)
		if err != nil {
			return err
		}

		app.TitleID = titleID

		if err := insertVPPApps(ctx, tx, []*fleet.VPPApp{app}); err != nil {
			return ctxerr.Wrap(ctx, err, "InsertVPPAppWithTeam insertVPPApps transaction")
		}

		if err := insertVPPAppTeams(ctx, tx, app.VPPAppTeam, teamID, vppToken.ID); err != nil {
			return ctxerr.Wrap(ctx, err, "InsertVPPAppWithTeam insertVPPAppTeams transaction")
		}

		return nil
	})
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "InsertVPPAppWithTeam")
	}

	return app, nil
}

func (ds *Datastore) GetAssignedVPPApps(ctx context.Context, teamID *uint) (map[fleet.VPPAppID]fleet.VPPAppTeam, error) {
	stmt := `
SELECT
	adam_id, platform, self_service, install_during_setup
FROM
	vpp_apps_teams vat
WHERE
	vat.global_or_team_id = ?
	`
	var tmID uint
	if teamID != nil {
		tmID = *teamID
	}

	var results []fleet.VPPAppTeam
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &results, stmt, tmID); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get assigned VPP apps")
	}

	appSet := make(map[fleet.VPPAppID]fleet.VPPAppTeam)
	for _, r := range results {
		appSet[r.VPPAppID] = r
	}

	return appSet, nil
}

func insertVPPApps(ctx context.Context, tx sqlx.ExtContext, apps []*fleet.VPPApp) error {
	stmt := `
INSERT INTO vpp_apps
	(adam_id, bundle_identifier, icon_url, name, latest_version, title_id, platform)
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
		insertVals.WriteString(`(?, ?, ?, ?, ?, ?, ?),`)
		args = append(args, a.AdamID, a.BundleIdentifier, a.IconURL, a.Name, a.LatestVersion, a.TitleID, a.Platform)
	}

	stmt = fmt.Sprintf(stmt, strings.TrimSuffix(insertVals.String(), ","))

	_, err := tx.ExecContext(ctx, stmt, args...)

	return ctxerr.Wrap(ctx, err, "insert VPP apps")
}

func insertVPPAppTeams(ctx context.Context, tx sqlx.ExtContext, appID fleet.VPPAppTeam, teamID *uint, vppTokenID uint) error {
	stmt := `
INSERT INTO vpp_apps_teams
	(adam_id, global_or_team_id, team_id, platform, self_service, vpp_token_id, install_during_setup)
VALUES
	(?, ?, ?, ?, ?, ?, COALESCE(?, false))
ON DUPLICATE KEY UPDATE
	self_service = VALUES(self_service),
	install_during_setup = COALESCE(?, install_during_setup)
`

	var globalOrTmID uint
	if teamID != nil {
		globalOrTmID = *teamID

		if *teamID == 0 {
			teamID = nil
		}
	}

	_, err := tx.ExecContext(ctx, stmt, appID.AdamID, globalOrTmID, teamID, appID.Platform, appID.SelfService, vppTokenID, appID.InstallDuringSetup, appID.InstallDuringSetup)
	if IsDuplicate(err) {
		err = &existsError{
			Identifier:   fmt.Sprintf("%s %s self_service: %v", appID.AdamID, appID.Platform, appID.SelfService),
			TeamID:       teamID,
			ResourceType: "VPPAppID",
		}
	}

	return ctxerr.Wrap(ctx, err, "writing vpp app team mapping to db")
}

func removeVPPAppTeams(ctx context.Context, tx sqlx.ExtContext, appID fleet.VPPAppID, teamID *uint) error {
	stmt := `
DELETE FROM
  vpp_apps_teams
WHERE
  adam_id = ?
AND
  team_id = ?
AND
  platform = ?
`
	_, err := tx.ExecContext(ctx, stmt, appID.AdamID, teamID, appID.Platform)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "deleting vpp app from team")
	}

	return nil
}

func (ds *Datastore) getOrInsertSoftwareTitleForVPPApp(ctx context.Context, tx sqlx.ExtContext, app *fleet.VPPApp) (uint, error) {
	// NOTE: it was decided to populate "apps" as the source for VPP apps for now, TBD
	// if this needs to change to better map to how software titles are reported
	// back by osquery. Since it may change, we're using a variable for the source.
	var source string
	switch app.Platform {
	case fleet.IOSPlatform:
		source = "ios_apps"
	case fleet.IPadOSPlatform:
		source = "ipados_apps"
	default:
		source = "apps"
	}

	selectStmt := `SELECT id FROM software_titles WHERE name = ? AND source = ? AND browser = ''`
	selectArgs := []any{app.Name, source}
	insertStmt := `INSERT INTO software_titles (name, source, browser) VALUES (?, ?, '')`
	insertArgs := []any{app.Name, source}

	if app.BundleIdentifier != "" {
		// match by bundle identifier first, or standard matching if we
		// don't have a bundle identifier match
		switch source {
		case "ios_apps", "ipados_apps":
			selectStmt = `
				    SELECT id
				    FROM software_titles
				    WHERE (bundle_identifier = ? AND source = ?) OR (name = ? AND source = ? AND browser = '')
				    ORDER BY bundle_identifier = ? DESC
				    LIMIT 1`
			selectArgs = []any{app.BundleIdentifier, source, app.Name, source, app.BundleIdentifier}
		default:
			selectStmt = `
				    SELECT id
				    FROM software_titles
				    WHERE (bundle_identifier = ? OR (name = ? AND browser = ''))
				      AND source NOT IN ('ios_apps', 'ipados_apps')
				    ORDER BY bundle_identifier = ? DESC
				    LIMIT 1`
			selectArgs = []any{app.BundleIdentifier, app.Name, app.BundleIdentifier}
		}
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
		return 0, ctxerr.Wrap(ctx, err, "optimistic get or insert VPP app")
	}

	return titleID, nil
}

func (ds *Datastore) DeleteVPPAppFromTeam(ctx context.Context, teamID *uint, appID fleet.VPPAppID) error {
	// allow delete only if install_during_setup is false
	const stmt = `DELETE FROM vpp_apps_teams WHERE global_or_team_id = ? AND adam_id = ? AND platform = ? AND install_during_setup = 0`

	var globalOrTeamID uint
	if teamID != nil {
		globalOrTeamID = *teamID
	}
	res, err := ds.writer(ctx).ExecContext(ctx, stmt, globalOrTeamID, appID.AdamID, appID.Platform)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "delete VPP app from team")
	}

	rows, _ := res.RowsAffected()
	if rows == 0 {
		// could be that the VPP app does not exist, or it is installed during
		// setup, do additional check.
		var installDuringSetup bool
		if err := sqlx.GetContext(ctx, ds.reader(ctx), &installDuringSetup,
			`SELECT install_during_setup FROM vpp_apps_teams WHERE global_or_team_id = ? AND adam_id = ? AND platform = ?`, globalOrTeamID, appID.AdamID, appID.Platform); err != nil && !errors.Is(err, sql.ErrNoRows) {
			return ctxerr.Wrap(ctx, err, "check if vpp app is installed during setup")
		}
		if installDuringSetup {
			return errDeleteInstallerInstalledDuringSetup
		}
		return notFound("VPPApp").WithMessage(fmt.Sprintf("adam id %s platform %s for team id %d", appID.AdamID, appID.Platform,
			globalOrTeamID))
	}
	return nil
}

func (ds *Datastore) GetVPPAppByTeamAndTitleID(ctx context.Context, teamID *uint, titleID uint) (*fleet.VPPApp, error) {
	stmt := `
SELECT
  va.adam_id,
  va.bundle_identifier,
  va.icon_url,
  va.name,
  va.title_id,
  va.platform,
  va.created_at,
  va.updated_at,
  vat.self_service
FROM vpp_apps va
JOIN vpp_apps_teams vat ON va.adam_id = vat.adam_id AND va.platform = vat.platform
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

func (ds *Datastore) InsertHostVPPSoftwareInstall(ctx context.Context, hostID uint, appID fleet.VPPAppID,
	commandUUID, associatedEventID string, selfService bool,
) error {
	stmt := `
INSERT INTO host_vpp_software_installs
  (host_id, adam_id, platform, command_uuid, user_id, associated_event_id, self_service)
VALUES
  (?,?,?,?,?,?,?)
	`

	var userID *uint
	if ctxUser := authz.UserFromContext(ctx); ctxUser != nil {
		userID = &ctxUser.ID
	}

	if _, err := ds.writer(ctx).ExecContext(ctx, stmt, hostID, appID.AdamID, appID.Platform, commandUUID, userID,
		associatedEventID, selfService); err != nil {
		return ctxerr.Wrap(ctx, err, "insert into host_vpp_software_installs")
	}

	return nil
}

func (ds *Datastore) GetActivityDataForVPPAppUnInstall(ctx context.Context, commandResults *mdm.CommandResults) (*fleet.User, *fleet.ActivityUnInstalledAppStoreApp, error) {
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
	hvsi.command_uuid AS command_uuid,
	hvsi.self_service AS self_service
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
		HostID          uint    `db:"host_id"`
		HostDisplayName string  `db:"host_display_name"`
		SoftwareTitle   string  `db:"software_title"`
		AppStoreID      string  `db:"app_store_id"`
		CommandUUID     string  `db:"command_uuid"`
		UserName        *string `db:"user_name"`
		UserID          *uint   `db:"user_id"`
		UserEmail       *string `db:"user_email"`
		SelfService     bool    `db:"self_service"`
	}

	listStmt, args, err := sqlx.Named(stmt, map[string]any{
		"command_uuid":              commandResults.CommandUUID,
		"software_status_failed":    string(fleet.SoftwareInstallFailed),
		"software_status_installed": string(fleet.SoftwareInstalled),
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

	var user *fleet.User
	if res.UserID != nil {
		user = &fleet.User{
			ID:    *res.UserID,
			Name:  *res.UserName,
			Email: *res.UserEmail,
		}
	}

	var status string
	switch commandResults.Status {
	case fleet.MDMAppleStatusAcknowledged:
		status = string(fleet.SoftwareInstalled)
	case fleet.MDMAppleStatusCommandFormatError:
	case fleet.MDMAppleStatusError:
		status = string(fleet.SoftwareInstallFailed)
	default:
		// This case shouldn't happen (we should only be doing this check if the command is in a
		// "terminal" state, but adding it so we have a default
		status = string(fleet.SoftwareInstallPending)
	}

	act := &fleet.ActivityUnInstalledAppStoreApp{
		HostID:          res.HostID,
		HostDisplayName: res.HostDisplayName,
		SoftwareTitle:   res.SoftwareTitle,
		AppStoreID:      res.AppStoreID,
		CommandUUID:     res.CommandUUID,
		SelfService:     res.SelfService,
		Status:          status,
	}

	return user, act, nil
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
	hvsi.command_uuid AS command_uuid,
	hvsi.self_service AS self_service
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
		HostID          uint    `db:"host_id"`
		HostDisplayName string  `db:"host_display_name"`
		SoftwareTitle   string  `db:"software_title"`
		AppStoreID      string  `db:"app_store_id"`
		CommandUUID     string  `db:"command_uuid"`
		UserName        *string `db:"user_name"`
		UserID          *uint   `db:"user_id"`
		UserEmail       *string `db:"user_email"`
		SelfService     bool    `db:"self_service"`
	}

	listStmt, args, err := sqlx.Named(stmt, map[string]any{
		"command_uuid":              commandResults.CommandUUID,
		"software_status_failed":    string(fleet.SoftwareInstallFailed),
		"software_status_installed": string(fleet.SoftwareInstalled),
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

	var user *fleet.User
	if res.UserID != nil {
		user = &fleet.User{
			ID:    *res.UserID,
			Name:  *res.UserName,
			Email: *res.UserEmail,
		}
	}

	var status string
	switch commandResults.Status {
	case fleet.MDMAppleStatusAcknowledged:
		status = string(fleet.SoftwareInstalled)
	case fleet.MDMAppleStatusCommandFormatError:
	case fleet.MDMAppleStatusError:
		status = string(fleet.SoftwareInstallFailed)
	default:
		// This case shouldn't happen (we should only be doing this check if the command is in a
		// "terminal" state, but adding it so we have a default
		status = string(fleet.SoftwareInstallPending)
	}

	act := &fleet.ActivityInstalledAppStoreApp{
		HostID:          res.HostID,
		HostDisplayName: res.HostDisplayName,
		SoftwareTitle:   res.SoftwareTitle,
		AppStoreID:      res.AppStoreID,
		CommandUUID:     res.CommandUUID,
		SelfService:     res.SelfService,
		Status:          status,
	}

	return user, act, nil
}

func (ds *Datastore) GetVPPTokenByLocation(ctx context.Context, loc string) (*fleet.VPPTokenDB, error) {
	stmt := `SELECT id FROM vpp_tokens WHERE location = ?`
	var tokenID uint
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &tokenID, stmt, loc); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ctxerr.Wrap(ctx, notFound("VPPToken"), "retrieve vpp token by location")
		}
		return nil, ctxerr.Wrap(ctx, err, "retrieve vpp token by location")
	}
	return ds.GetVPPToken(ctx, tokenID)
}

func (ds *Datastore) InsertVPPToken(ctx context.Context, tok *fleet.VPPTokenData) (*fleet.VPPTokenDB, error) {
	insertStmt := `
	INSERT INTO
		vpp_tokens (
			organization_name,
			location,
			renew_at,
			token
		)
	VALUES (?, ?, ?, ?)
`

	vppTokenDB, err := vppTokenDataToVppTokenDB(ctx, tok)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "translating vpp token to db representation")
	}

	tokEnc, err := encrypt([]byte(vppTokenDB.Token), ds.serverPrivateKey)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "encrypt token with datastore.serverPrivateKey")
	}

	res, err := ds.writer(ctx).ExecContext(
		ctx,
		insertStmt,
		vppTokenDB.OrgName,
		vppTokenDB.Location,
		vppTokenDB.RenewDate,
		tokEnc,
	)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "inserting vpp token")
	}

	id, _ := res.LastInsertId()

	vppTokenDB.ID = uint(id) //nolint:gosec // dismiss G115

	return vppTokenDB, nil
}

func (ds *Datastore) UpdateVPPToken(ctx context.Context, tokenID uint, tok *fleet.VPPTokenData) (*fleet.VPPTokenDB, error) {
	stmt := `
	UPDATE vpp_tokens
	SET
		organization_name = ?,
		location = ?,
		renew_at = ?,
		token = ?
	WHERE
		id = ?
`

	vppTokenDB, err := vppTokenDataToVppTokenDB(ctx, tok)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "translating vpp token to db representation")
	}

	tokEnc, err := encrypt([]byte(vppTokenDB.Token), ds.serverPrivateKey)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "encrypt token with datastore.serverPrivateKey")
	}

	_, err = ds.writer(ctx).ExecContext(
		ctx,
		stmt,
		vppTokenDB.OrgName,
		vppTokenDB.Location,
		vppTokenDB.RenewDate,
		tokEnc,
		tokenID,
	)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "inserting vpp token")
	}

	return ds.GetVPPToken(ctx, tokenID)
}

func vppTokenDataToVppTokenDB(ctx context.Context, tok *fleet.VPPTokenData) (*fleet.VPPTokenDB, error) {
	tokRawBytes, err := base64.StdEncoding.DecodeString(tok.Token)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "decoding raw vpp token data")
	}

	var tokRaw fleet.VPPTokenRaw
	if err := json.Unmarshal(tokRawBytes, &tokRaw); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "unmarshalling raw vpp token")
	}

	exp, err := time.Parse(fleet.VPPTimeFormat, tokRaw.ExpDate)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "parsing vpp token expiration date")
	}
	exp = exp.UTC()

	vppTokenDB := &fleet.VPPTokenDB{
		OrgName:   tokRaw.OrgName,
		Location:  tok.Location,
		RenewDate: exp,
		Token:     tok.Token,
	}

	return vppTokenDB, nil
}

func (ds *Datastore) GetVPPToken(ctx context.Context, tokenID uint) (*fleet.VPPTokenDB, error) {
	stmt := `
	SELECT
		id,
		organization_name,
		location,
		renew_at,
		token
	FROM
		vpp_tokens v
	WHERE
		id = ?
`
	stmtTeams := `
	SELECT
		vt.team_id,
		vt.null_team_type,
		COALESCE(t.name, '') AS name
	FROM
		vpp_token_teams vt
	LEFT OUTER JOIN
		teams t
	ON t.id = vt.team_id
	WHERE
		vpp_token_id = ?
`

	var tokEnc fleet.VPPTokenDB

	var tokTeams []struct {
		TeamID   *uint              `db:"team_id"`
		NullTeam fleet.NullTeamType `db:"null_team_type"`
		Name     string             `db:"name"`
	}

	if err := sqlx.GetContext(ctx, ds.reader(ctx), &tokEnc, stmt, tokenID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ctxerr.Wrap(ctx, notFound("VPPToken"), "selecting vpp token from db")
		}
		return nil, ctxerr.Wrap(ctx, err, "selecting vpp token from db")
	}

	tokDec, err := decrypt([]byte(tokEnc.Token), ds.serverPrivateKey)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "decrypting vpp token with serverPrivateKey")
	}

	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &tokTeams, stmtTeams, tokenID); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "selecting vpp token teams from db")
	}

	tok := &fleet.VPPTokenDB{
		ID:        tokEnc.ID,
		OrgName:   tokEnc.OrgName,
		Location:  tokEnc.Location,
		RenewDate: tokEnc.RenewDate,
		Token:     string(tokDec),
	}

	if tokTeams == nil {
		// Not assigned, no need to loop over teams
		return tok, nil
	}

TEAMLOOP:
	for _, team := range tokTeams {
		switch team.NullTeam {
		case fleet.NullTeamAllTeams:
			// This should only be possible if there are no other teams
			// Make sure something array is non-nil
			if len(tokTeams) != 1 {
				return nil, ctxerr.Errorf(ctx, "team \"%s\" belongs to All teams, and %d other team(s)", tok.OrgName, len(tokTeams)-1)
			}
			tok.Teams = []fleet.TeamTuple{}
			break TEAMLOOP
		case fleet.NullTeamNoTeam:
			tok.Teams = append(tok.Teams, fleet.TeamTuple{
				ID:   0,
				Name: fleet.TeamNameNoTeam,
			})
		case fleet.NullTeamNone:
			// Regular team
			tok.Teams = append(tok.Teams, fleet.TeamTuple{
				ID:   *team.TeamID,
				Name: team.Name,
			})
		}
	}

	return tok, nil
}

func (ds *Datastore) UpdateVPPTokenTeams(ctx context.Context, id uint, teams []uint) (*fleet.VPPTokenDB, error) {
	stmtTeamName := `SELECT name FROM teams WHERE id = ?`
	stmtRemove := `DELETE FROM vpp_token_teams WHERE vpp_token_id = ?`
	stmtInsert := `
	INSERT INTO
		vpp_token_teams (
			vpp_token_id,
			team_id,
			null_team_type
	) VALUES `
	stmtValues := `(?, ?, ?)`
	// Delete all apps associated with a token if we change its team
	stmtDeleteApps := `DELETE FROM vpp_apps_teams WHERE vpp_token_id = ?`
	deleteArgs := []any{id}

	var values string
	var args []any
	// No DB constraint for null_team_type, if no team or all teams
	// comes up we have to check it in go
	var nullTeamCheck fleet.NullTeamType

	if len(teams) > 0 {
		for _, team := range teams {
			team := team
			if values == "" {
				values = stmtValues
			} else {
				values = strings.Join([]string{values, stmtValues}, ",")
			}
			var teamptr *uint
			nullTeam := fleet.NullTeamNone
			if team != 0 {
				// Regular team
				teamptr = &team
			} else {
				// NoTeam team
				nullTeam = fleet.NullTeamNoTeam
				nullTeamCheck = fleet.NullTeamNoTeam
			}
			args = append(args, id, teamptr, nullTeam)
		}
	} else if teams != nil {
		// Empty but not nil, All Teams!
		values = stmtValues
		args = append(args, id, nil, fleet.NullTeamAllTeams)
		nullTeamCheck = fleet.NullTeamAllTeams
	}

	stmtInsertFull := stmtInsert + values

	err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		// NOTE This is not optimal, and has the potential to
		// introduce race conditions. Ideally we would insert and
		// check the constraints in a single query.
		if err := checkVPPNullTeam(ctx, tx, &id, nullTeamCheck); err != nil {
			return ctxerr.Wrap(ctx, err, "vpp token null team check")
		}

		if _, err := tx.ExecContext(ctx, stmtDeleteApps, deleteArgs...); err != nil {
			return ctxerr.Wrap(ctx, err, "deleting old vpp team apps associations")
		}

		if _, err := tx.ExecContext(ctx, stmtRemove, id); err != nil {
			return ctxerr.Wrap(ctx, err, "removing old vpp team associations")
		}

		if len(args) > 0 {
			if _, err := tx.ExecContext(ctx, stmtInsertFull, args...); err != nil {
				if isChildForeignKeyError(err) {
					return foreignKey("team", fmt.Sprintf("(team_id)=(%v)", values))
				}

				return ctxerr.Wrap(ctx, err, "updating vpp token team")
			}
		}

		return nil
	})
	if err != nil {
		var mysqlErr *mysql.MySQLError
		// https://dev.mysql.com/doc/mysql-errors/8.4/en/server-error-reference.html#error_er_dup_entry
		if errors.As(err, &mysqlErr) && IsDuplicate(err) {
			var dupeTeamID uint
			var dupeTeamName string
			_, _ = fmt.Sscanf(mysqlErr.Message, "Duplicate entry '%d' for", &dupeTeamID)
			if err := sqlx.GetContext(ctx, ds.reader(ctx), &dupeTeamName, stmtTeamName, dupeTeamID); err != nil {
				return nil, ctxerr.Wrap(ctx, err, "getting team name for vpp token conflict error")
			}
			return nil, ctxerr.Wrap(ctx, fleet.ErrVPPTokenTeamConstraint{Name: dupeTeamName, ID: &dupeTeamID})
		}
		return nil, ctxerr.Wrap(ctx, err, "modifying vpp token team associations")
	}

	return ds.GetVPPToken(ctx, id)
}

func (ds *Datastore) DeleteVPPToken(ctx context.Context, tokenID uint) error {
	_, err := ds.writer(ctx).ExecContext(ctx, `DELETE FROM vpp_tokens WHERE id = ?`, tokenID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "deleting vpp token")
	}

	return nil
}

func (ds *Datastore) ListVPPTokens(ctx context.Context) ([]*fleet.VPPTokenDB, error) {
	// linter false positive on the word "token" (gosec G101)
	//nolint:gosec
	stmtTokens := `
	SELECT
		id,
		organization_name,
		location,
		renew_at,
		token
	FROM
		vpp_tokens v
`

	stmtTeams := `
	SELECT
		vt.id,
		vt.vpp_token_id,
		vt.team_id,
		vt.null_team_type,
		COALESCE(t.name, '') AS name
	FROM
		vpp_token_teams vt
	LEFT OUTER JOIN
		teams t
	ON vt.team_id = t.id
`
	var tokEncs []fleet.VPPTokenDB

	var teams []struct {
		ID         string             `db:"id"`
		VPPTokenID uint               `db:"vpp_token_id"`
		TeamID     *uint              `db:"team_id"`
		TeamName   string             `db:"name"`
		NullTeam   fleet.NullTeamType `db:"null_team_type"`
	}

	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &tokEncs, stmtTokens); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "selecting vpp tokens from db")
	}

	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &teams, stmtTeams); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "selecting vpp token teams from db")
	}

	tokens := map[uint]*fleet.VPPTokenDB{}

	for _, tokEnc := range tokEncs {
		tokDec, err := decrypt([]byte(tokEnc.Token), ds.serverPrivateKey)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "decrypting vpp token with serverPrivateKey")
		}

		tokens[tokEnc.ID] = &fleet.VPPTokenDB{
			ID:        tokEnc.ID,
			OrgName:   tokEnc.OrgName,
			Location:  tokEnc.Location,
			RenewDate: tokEnc.RenewDate,
			Token:     string(tokDec),
		}
	}

	for _, team := range teams {
		token := tokens[team.VPPTokenID]
		if token.Teams != nil && len(token.Teams) == 0 {
			// Token was already assigned to All Teams, we should not
			// see it again in a loop
			return nil, fmt.Errorf("vpp token \"%s\" has been assigned to All teams, and another team", token.OrgName)
		}
		switch team.NullTeam {
		case fleet.NullTeamAllTeams:
			// All teams, there should be no other teams.
			// Make sure array is non-nil
			if token.Teams != nil {
				// This team has already been assigned something, this
				// should not happen
				return nil, fmt.Errorf("vpp token \"%s\" has been asssigned to All teams, and another team", token.OrgName)
			}
			token.Teams = []fleet.TeamTuple{}
		case fleet.NullTeamNoTeam:
			token.Teams = append(token.Teams, fleet.TeamTuple{ID: 0, Name: fleet.TeamNameNoTeam})
		case fleet.NullTeamNone:
			// Regular team
			token.Teams = append(token.Teams, fleet.TeamTuple{ID: *team.TeamID, Name: team.TeamName})
		}
	}

	var outTokens []*fleet.VPPTokenDB
	for _, token := range tokens {
		outTokens = append(outTokens, token)
	}

	slices.SortFunc(outTokens, func(a, b *fleet.VPPTokenDB) int {
		return cmp.Compare(a.OrgName, b.OrgName)
	})

	return outTokens, nil
}

func (ds *Datastore) GetVPPTokenByTeamID(ctx context.Context, teamID *uint) (*fleet.VPPTokenDB, error) {
	stmtTeam := `
	SELECT
		v.id,
		v.organization_name,
		v.location,
		v.renew_at,
		v.token
	FROM
		vpp_token_teams vt
	INNER JOIN
		vpp_tokens v
	ON vt.vpp_token_id = v.id
	WHERE
		vt.team_id = ?
`
	stmtTeamNames := `
	SELECT
		vt.team_id,
		vt.null_team_type,
		COALESCE(t.name, '') AS name
	FROM
		vpp_token_teams vt
	LEFT OUTER JOIN
		teams t
	ON t.id = vt.team_id
	WHERE
		vt.vpp_token_id = ?
`
	stmtNullTeam := `
	SELECT
		v.id,
		v.organization_name,
		v.location,
		v.renew_at,
		v.token
	FROM
		vpp_tokens v
	INNER JOIN
		vpp_token_teams vt
	ON v.id = vt.vpp_token_id
	WHERE
		vt.team_id IS NULL
	AND
		vt.null_team_type = ?
`

	var tokEnc fleet.VPPTokenDB

	var tokTeams []struct {
		TeamID   *uint              `db:"team_id"`
		NullTeam fleet.NullTeamType `db:"null_team_type"`
		Name     string             `db:"name"`
	}

	var err error
	if teamID != nil && *teamID != 0 {
		err = sqlx.GetContext(ctx, ds.reader(ctx), &tokEnc, stmtTeam, teamID)
	} else {
		err = sqlx.GetContext(ctx, ds.reader(ctx), &tokEnc, stmtNullTeam, fleet.NullTeamNoTeam)
	}
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			if err := sqlx.GetContext(ctx, ds.reader(ctx), &tokEnc, stmtNullTeam, fleet.NullTeamAllTeams); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return nil, ctxerr.Wrap(ctx, notFound("VPPToken"), "retrieving vpp token by team")
				}
				return nil, ctxerr.Wrap(ctx, err, "retrieving vpp token by team")
			}
		} else {
			return nil, ctxerr.Wrap(ctx, err, "retrieving vpp token by team")
		}
	}

	tokDec, err := decrypt([]byte(tokEnc.Token), ds.serverPrivateKey)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "decrypting vpp token with serverPrivateKey")
	}

	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &tokTeams, stmtTeamNames, tokEnc.ID); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "retrieving vpp token team information")
	}

	tok := &fleet.VPPTokenDB{
		ID:        tokEnc.ID,
		OrgName:   tokEnc.OrgName,
		Location:  tokEnc.Location,
		RenewDate: tokEnc.RenewDate,
		Token:     string(tokDec),
	}

	if tokTeams == nil {
		// Not assigned, no need to loop over teams
		return tok, nil
	}

TEAMLOOP:
	for _, team := range tokTeams {
		switch team.NullTeam {
		case fleet.NullTeamAllTeams:
			// This should only be possible if there are no other teams
			// Make sure something array is non-nil
			if len(tokTeams) != 1 {
				return nil, ctxerr.Errorf(ctx, "team \"%s\" belongs to All teams, and %d other team(s)", tok.OrgName, len(tokTeams)-1)
			}
			tok.Teams = []fleet.TeamTuple{}
			break TEAMLOOP
		case fleet.NullTeamNoTeam:
			tok.Teams = append(tok.Teams, fleet.TeamTuple{
				ID:   0,
				Name: fleet.TeamNameNoTeam,
			})
		case fleet.NullTeamNone:
			// Regular team
			tok.Teams = append(tok.Teams, fleet.TeamTuple{
				ID:   *team.TeamID,
				Name: team.Name,
			})
		}
	}

	return tok, nil
}

func checkVPPNullTeam(ctx context.Context, tx sqlx.ExtContext, currentID *uint, nullTeam fleet.NullTeamType) error {
	nullTeamStmt := `SELECT vpp_token_id FROM vpp_token_teams WHERE null_team_type = ?`
	anyTeamStmt := `SELECT vpp_token_id FROM vpp_token_teams WHERE null_team_type = 'allteams' OR null_team_type = 'noteam' OR team_id IS NOT NULL`

	if nullTeam == fleet.NullTeamAllTeams {
		var ids []uint
		if err := sqlx.SelectContext(ctx, tx, &ids, anyTeamStmt); err != nil {
			return ctxerr.Wrap(ctx, err, "scanning row in check vpp token null team")
		}

		if len(ids) > 0 {
			if len(ids) > 1 {
				return ctxerr.Wrap(ctx, errors.New("Cannot assign token to All teams, other teams have tokens"))
			}
			if currentID == nil || ids[0] != *currentID {
				return ctxerr.Wrap(ctx, errors.New("Cannot assign token to All teams, other teams have tokens"))
			}
		}
	}

	var id uint
	allTeamsFound := true
	if err := sqlx.GetContext(ctx, tx, &id, nullTeamStmt, fleet.NullTeamAllTeams); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			allTeamsFound = false
		} else {
			return ctxerr.Wrap(ctx, err, "scanning row in check vpp token null team")
		}
	}

	if allTeamsFound && currentID != nil && *currentID != id {
		return ctxerr.Wrap(ctx, fleet.ErrVPPTokenTeamConstraint{Name: fleet.ReservedNameAllTeams})
	}

	if nullTeam != fleet.NullTeamNone {
		var id uint
		if err := sqlx.GetContext(ctx, tx, &id, nullTeamStmt, nullTeam); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil
			}
			return ctxerr.Wrap(ctx, err, "scanning row in check vpp token null team")
		}
		if currentID == nil || *currentID != id {
			return ctxerr.Wrap(ctx, fleet.ErrVPPTokenTeamConstraint{Name: nullTeam.PrettyName()})
		}
	}

	return nil
}
