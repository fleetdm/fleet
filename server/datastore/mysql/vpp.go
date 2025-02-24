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

	"github.com/fleetdm/fleet/v4/pkg/automatic_policy"
	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/go-kit/log/level"
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
	vat.id vpp_apps_teams_id,
	vat.created_at added_at,
	NULLIF(vap.icon_url, '') AS icon_url,
	vap.bundle_identifier AS bundle_identifier
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

	labels, err := ds.getVPPAppLabels(ctx, app.VPPAppsTeamsID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get vpp app labels")
	}
	var exclAny, inclAny []fleet.SoftwareScopeLabel
	for _, l := range labels {
		if l.Exclude {
			exclAny = append(exclAny, l)
		} else {
			inclAny = append(inclAny, l)
		}
	}

	if len(inclAny) > 0 && len(exclAny) > 0 {
		// there's a bug somewhere
		level.Warn(ds.logger).Log("msg", "vpp app has both include and exclude labels", "vpp_apps_teams_id", app.VPPAppsTeamsID, "include", fmt.Sprintf("%v", inclAny), "exclude", fmt.Sprintf("%v", exclAny))
	}
	app.LabelsExcludeAny = exclAny
	app.LabelsIncludeAny = inclAny

	policies, err := ds.getPoliciesBySoftwareTitleIDs(ctx, []uint{titleID}, teamID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get policies by software title ID")
	}
	app.AutomaticInstallPolicies = policies

	return &app, nil
}

func (ds *Datastore) getVPPAppLabels(ctx context.Context, vppAppsTeamsID uint) ([]fleet.SoftwareScopeLabel, error) {
	query := `
SELECT
	label_id,
	exclude,
	l.name AS label_name,
	va.title_id AS title_id
FROM
	vpp_app_team_labels vatl
	JOIN vpp_apps_teams vat ON vat.id = vatl.vpp_app_team_id
	JOIN vpp_apps va ON va.adam_id = vat.adam_id
	JOIN labels l ON l.id = vatl.label_id
WHERE
	vatl.vpp_app_team_id = ? AND va.platform = vat.platform
`

	var labels []fleet.SoftwareScopeLabel
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &labels, query, vppAppsTeamsID); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get vpp app labels")
	}

	return labels, nil
}

func (ds *Datastore) GetSummaryHostVPPAppInstalls(ctx context.Context, teamID *uint, appID fleet.VPPAppID) (*fleet.VPPAppStatusSummary,
	error,
) {
	var dest fleet.VPPAppStatusSummary

	// TODO(sarah): do we need to handle host_deleted_at similar to GetSummaryHostSoftwareInstalls?
	// Currently there is no host_deleted_at in host_vpp_software_installs, so
	// not handling it as part of the unified queue work.

	stmt := `
WITH

-- select most recent upcoming activities for each host
upcoming AS (
	SELECT
		ua.host_id,
		:software_status_pending AS status
	FROM
		upcoming_activities ua
		JOIN vpp_app_upcoming_activities vaua ON ua.id = vaua.upcoming_activity_id
		JOIN hosts h ON host_id = h.id
		LEFT JOIN (
			upcoming_activities ua2
			INNER JOIN vpp_app_upcoming_activities vaua2
				ON ua2.id = vaua2.upcoming_activity_id
		) ON ua.host_id = ua2.host_id AND
			vaua.adam_id = vaua2.adam_id AND
			vaua.platform = vaua2.platform AND
			ua.activity_type = ua2.activity_type AND
			(ua2.priority < ua.priority OR ua2.created_at > ua.created_at)
	WHERE
		ua.activity_type = 'vpp_app_install'
		AND ua2.id IS NULL
		AND vaua.adam_id = :adam_id
		AND vaua.platform = :platform
		AND (h.team_id = :team_id OR (h.team_id IS NULL AND :team_id = 0))
),

-- select most recent past activities for each host
past AS (
	SELECT
		hvsi.host_id,
		CASE
			WHEN ncr.status = :mdm_status_acknowledged THEN
				:software_status_installed
			WHEN ncr.status = :mdm_status_error OR ncr.status = :mdm_status_format_error THEN
				:software_status_failed
			ELSE
				NULL -- either pending or not installed via VPP App
		END AS status
	FROM
		host_vpp_software_installs hvsi
		JOIN hosts h ON host_id = h.id
		JOIN nano_command_results ncr ON ncr.id = h.uuid AND ncr.command_uuid = hvsi.command_uuid
		LEFT JOIN host_vpp_software_installs hvsi2
			ON hvsi.host_id = hvsi2.host_id AND
				 hvsi.adam_id = hvsi2.adam_id AND
				 hvsi.platform = hvsi2.platform AND
				 hvsi2.removed = 0 AND
				 (hvsi.created_at < hvsi2.created_at OR (hvsi.created_at = hvsi2.created_at AND hvsi.id < hvsi2.id))
	WHERE
		hvsi2.id IS NULL
		AND hvsi.adam_id = :adam_id
		AND hvsi.platform = :platform
		AND (h.team_id = :team_id OR (h.team_id IS NULL AND :team_id = 0))
		AND hvsi.host_id NOT IN (SELECT host_id FROM upcoming) -- antijoin to exclude hosts with upcoming activities
		AND hvsi.removed = 0
)

-- count each status
SELECT
	COALESCE(SUM( IF(status = :software_status_pending, 1, 0)), 0) AS pending,
	COALESCE(SUM( IF(status = :software_status_failed, 1, 0)), 0) AS failed,
	COALESCE(SUM( IF(status = :software_status_installed, 1, 0)), 0) AS installed
FROM (

-- union most recent past and upcoming activities after joining to get statuses for most recent activities
SELECT
	past.host_id,
	past.status
FROM past
UNION
SELECT
	upcoming.host_id,
	upcoming.status
FROM upcoming 
) t`

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

func (ds *Datastore) getExistingLabels(ctx context.Context, vppAppTeamID uint) (*fleet.LabelIdentsWithScope, error) {
	existingLabels, err := ds.getVPPAppLabels(ctx, vppAppTeamID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting existing labels")
	}

	var labels fleet.LabelIdentsWithScope
	var exclAny, inclAny []fleet.SoftwareScopeLabel
	for _, l := range existingLabels {
		if l.Exclude {
			exclAny = append(exclAny, l)
		} else {
			inclAny = append(inclAny, l)
		}
	}

	if len(inclAny) > 0 && len(exclAny) > 0 {
		// there's a bug somewhere
		return nil, ctxerr.New(ctx, "found both include and exclude labels on a vpp app")
	}

	switch {
	case len(exclAny) > 0:
		labels.LabelScope = fleet.LabelScopeExcludeAny
		labels.ByName = make(map[string]fleet.LabelIdent, len(exclAny))
		for _, l := range exclAny {
			labels.ByName[l.LabelName] = fleet.LabelIdent{LabelName: l.LabelName, LabelID: l.LabelID}
		}
		return &labels, nil

	case len(inclAny) > 0:
		labels.LabelScope = fleet.LabelScopeIncludeAny
		labels.ByName = make(map[string]fleet.LabelIdent, len(inclAny))
		for _, l := range inclAny {
			labels.ByName[l.LabelName] = fleet.LabelIdent{LabelName: l.LabelName, LabelID: l.LabelID}
		}
		return &labels, nil
	default:
		return nil, nil
	}
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

	appsWithChangedLabels := make(map[uint]map[uint]struct{})
	for _, appFleet := range appFleets {
		// upsert it if it does not exist or labels or SelfService or InstallDuringSetup flags are changed
		existingApp, isExistingApp := existingApps[appFleet.VPPAppID]
		appFleet.AppTeamID = existingApp.AppTeamID
		var labelsChanged bool
		if isExistingApp {
			existingLabels, err := ds.getExistingLabels(ctx, appFleet.AppTeamID)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "getting existing labels for vpp app")
			}

			labelsChanged = !existingLabels.Equal(appFleet.ValidatedLabels)

		}

		// Get the hosts that are NOT in label scope currently (before the update happens)
		if labelsChanged {
			hostsNotInScope, err := ds.GetExcludedHostIDMapForVPPApp(ctx, appFleet.AppTeamID)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "getting hosts not in scope for vpp app")
			}
			appsWithChangedLabels[appFleet.AppTeamID] = hostsNotInScope

		}

		if !isExistingApp ||
			existingApp.SelfService != appFleet.SelfService ||
			labelsChanged ||
			appFleet.InstallDuringSetup != nil &&
				existingApp.InstallDuringSetup != nil &&
				*appFleet.InstallDuringSetup != *existingApp.InstallDuringSetup {
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
			vppAppTeamID, err := insertVPPAppTeams(ctx, tx, toAdd, teamID, vppToken.ID)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "SetTeamVPPApps inserting vpp app into team")
			}

			if toAdd.ValidatedLabels != nil {
				if err := setOrUpdateSoftwareInstallerLabelsDB(ctx, tx, vppAppTeamID, *toAdd.ValidatedLabels, softwareTypeVPP); err != nil {
					return ctxerr.Wrap(ctx, err, "failed to update labels on vpp apps batch operation")
				}
			}

			if hostsNotInScope, ok := appsWithChangedLabels[toAdd.AppTeamID]; ok {
				hostsInScope, err := ds.GetIncludedHostIDMapForVPPAppTx(ctx, tx, toAdd.AppTeamID)
				if err != nil {
					return ctxerr.Wrap(ctx, err, "getting hosts in scope for vpp app")
				}

				var hostsToClear []uint
				for id := range hostsInScope {
					if _, ok := hostsNotInScope[id]; ok {
						// it was not in scope but now it is, so we should clear policy status
						hostsToClear = append(hostsToClear, id)
					}
				}

				// We clear the policy status here because otherwise the policy automation machinery
				// won't pick this up and the software won't install.
				if err := ds.ClearVPPAppAutoInstallPolicyStatusForHostsTx(ctx, tx, toAdd.AppTeamID, hostsToClear); err != nil {
					return ctxerr.Wrap(ctx, err, "failed to clear auto install policy status for host")
				}
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

		vppAppTeamID, err := insertVPPAppTeams(ctx, tx, app.VPPAppTeam, teamID, vppToken.ID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "InsertVPPAppWithTeam insertVPPAppTeams transaction")
		}

		app.VPPAppTeam.AppTeamID = vppAppTeamID

		if app.ValidatedLabels != nil {
			if err := setOrUpdateSoftwareInstallerLabelsDB(ctx, tx, vppAppTeamID, *app.ValidatedLabels, softwareTypeVPP); err != nil {
				return ctxerr.Wrap(ctx, err, "InsertVPPAppWithTeam setOrUpdateSoftwareInstallerLabelsDB transaction")
			}
		}

		if app.VPPAppTeam.AddAutoInstallPolicy {
			generatedPolicyData, err := automatic_policy.Generate(automatic_policy.MacInstallerMetadata{
				Title:            app.Name,
				BundleIdentifier: app.BundleIdentifier,
			})
			if err != nil {
				return ctxerr.Wrap(ctx, err, "generate automatic policy query data")
			}

			if err := ds.createAutomaticPolicy(ctx, tx, *generatedPolicyData, teamID, nil, ptr.Uint(vppAppTeamID)); err != nil {
				return ctxerr.Wrap(ctx, err, "create automatic policy")
			}
		}

		return nil
	})
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "InsertVPPAppWithTeam")
	}

	return app, nil
}

func (ds *Datastore) GetVPPApps(ctx context.Context, teamID *uint) ([]fleet.VPPAppResponse, error) {
	var tmID uint
	if teamID != nil {
		tmID = *teamID
	}
	var results []fleet.VPPAppResponse

	// intentionally using writer as this is called right after batch-setting VPP apps
	if err := sqlx.SelectContext(ctx, ds.writer(ctx), &results, `
		SELECT vat.team_id, va.title_id, vat.adam_id app_store_id, vat.platform
		FROM vpp_apps_teams vat
		JOIN vpp_apps va ON va.adam_id = vat.adam_id AND va.platform = vat.platform
		WHERE global_or_team_id = ?`, tmID); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get VPP apps")
	}

	return results, nil
}

func (ds *Datastore) GetAssignedVPPApps(ctx context.Context, teamID *uint) (map[fleet.VPPAppID]fleet.VPPAppTeam, error) {
	stmt := `
SELECT
	adam_id, platform, self_service, install_during_setup, id, created_at added_at
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

func (ds *Datastore) InsertVPPApps(ctx context.Context, apps []*fleet.VPPApp) error {
	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		return insertVPPApps(ctx, tx, apps)
	})
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

func insertVPPAppTeams(ctx context.Context, tx sqlx.ExtContext, appID fleet.VPPAppTeam, teamID *uint, vppTokenID uint) (uint, error) {
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

	res, err := tx.ExecContext(ctx, stmt, appID.AdamID, globalOrTmID, teamID, appID.Platform, appID.SelfService, vppTokenID, appID.InstallDuringSetup, appID.InstallDuringSetup)
	if IsDuplicate(err) {
		err = &existsError{
			Identifier:   fmt.Sprintf("%s %s self_service: %v", appID.AdamID, appID.Platform, appID.SelfService),
			TeamID:       teamID,
			ResourceType: "VPPAppID",
		}
	}

	var id int64
	if insertOnDuplicateDidInsertOrUpdate(res) {
		id, _ = res.LastInsertId()
	} else {
		stmt := `SELECT id FROM vpp_apps_teams WHERE adam_id = ? AND platform = ? AND global_or_team_id = ?`
		if err := sqlx.GetContext(ctx, tx, &id, stmt, appID.AdamID, appID.Platform, globalOrTmID); err != nil {
			return 0, ctxerr.Wrap(ctx, err, "vpp app teams id")
		}
	}

	vatID := uint(id) //nolint:gosec // dismiss G115
	return vatID, ctxerr.Wrap(ctx, err, "writing vpp app team mapping to db")
}

func removeVPPAppTeams(ctx context.Context, tx sqlx.ExtContext, appID fleet.VPPAppID, teamID *uint) error {
	_, err := tx.ExecContext(ctx, `UPDATE policies p
		JOIN vpp_apps_teams vat ON vat.id = p.vpp_apps_teams_id AND vat.adam_id = ? AND vat.team_id = ? AND vat.platform = ?
		SET vpp_apps_teams_id = NULL`, appID.AdamID, teamID, appID.Platform)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "unsetting vpp app policy associations from team")
	}

	_, err = tx.ExecContext(ctx, `DELETE FROM vpp_apps_teams WHERE adam_id = ? AND team_id = ? AND platform = ?`, appID.AdamID, teamID, appID.Platform)
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
	tx := ds.writer(ctx) // make sure we're looking at a consistent vision of the world when deleting
	res, err := tx.ExecContext(ctx, stmt, globalOrTeamID, appID.AdamID, appID.Platform)
	if err != nil {
		if isMySQLForeignKey(err) {
			// Check if the app is referenced by a policy automation.
			var count int
			if err := sqlx.GetContext(ctx, tx, &count, `SELECT COUNT(*) FROM policies p JOIN vpp_apps_teams vat
					ON vat.id = p.vpp_apps_teams_id AND vat.global_or_team_id = ?
				    AND vat.adam_id = ? AND vat.platform = ?`, globalOrTeamID, appID.AdamID, appID.Platform); err != nil {
				return ctxerr.Wrapf(ctx, err, "getting reference from policies")
			}
			if count > 0 {
				return errDeleteInstallerWithAssociatedPolicy
			}
		}
		return ctxerr.Wrap(ctx, err, "delete VPP app from team")
	}

	rows, _ := res.RowsAffected()
	if rows == 0 {
		// could be that the VPP app does not exist, or it is installed during
		// setup, do additional check.
		var installDuringSetup bool
		if err := sqlx.GetContext(ctx, tx, &installDuringSetup,
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

func (ds *Datastore) GetTitleInfoFromVPPAppsTeamsID(ctx context.Context, vppAppsTeamsID uint) (*fleet.PolicySoftwareTitle, error) {
	var info fleet.PolicySoftwareTitle
	err := sqlx.GetContext(ctx, ds.reader(ctx), &info, `SELECT name, title_id FROM vpp_apps va
    	JOIN vpp_apps_teams vat ON vat.adam_id = va.adam_id AND vat.platform = va.platform AND vat.id = ?`, vppAppsTeamsID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("VPPApp"), "get VPP title info from VPP apps teams ID")
		}
		return nil, ctxerr.Wrap(ctx, err, "get VPP title info from VPP apps teams ID")
	}

	return &info, nil
}

func (ds *Datastore) GetVPPAppMetadataByAdamIDPlatformTeamID(ctx context.Context, adamID string, platform fleet.AppleDevicePlatform, teamID *uint) (*fleet.VPPApp, error) {
	stmt := `
	SELECT va.adam_id,
	 va.bundle_identifier,
	 va.icon_url,
	 va.name,
	 va.platform,
	 vat.self_service,
	 va.title_id,
	 va.platform,
	 va.created_at,
	 vat.created_at added_at,
	 va.updated_at,
	 vat.id
	FROM vpp_apps va
	JOIN vpp_apps_teams vat ON va.adam_id = vat.adam_id AND va.platform = vat.platform AND vat.global_or_team_id = ?
	WHERE va.adam_id = ? AND va.platform = ?
  `

	// when team id is not nil, we need to filter by the global or team id given.
	var tmID uint
	if teamID != nil {
		tmID = *teamID
	}

	var dest fleet.VPPApp
	err := sqlx.GetContext(ctx, ds.reader(ctx), &dest, stmt, tmID, adamID, platform)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("VPPApp"), "get VPP app metadata by team")
		}
		return nil, ctxerr.Wrap(ctx, err, "get VPP app metadata by team")
	}

	return &dest, nil
}

func (ds *Datastore) GetVPPAppByTeamAndTitleID(ctx context.Context, teamID *uint, titleID uint) (*fleet.VPPApp, error) {
	stmt := `
SELECT
  vat.id,
  va.adam_id,
  va.bundle_identifier,
  va.icon_url,
  va.name,
  va.title_id,
  va.platform,
  va.created_at,
  va.updated_at,
  vat.self_service,
  vat.created_at added_at
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
	commandUUID, associatedEventID string, opts fleet.HostSoftwareInstallOptions,
) error {
	const (
		insertUAStmt = `
INSERT INTO upcoming_activities
	(host_id, priority, user_id, fleet_initiated, activity_type, execution_id, payload)
VALUES
	(?, ?, ?, ?, 'vpp_app_install', ?,
		JSON_OBJECT(
			'self_service', ?,
			'associated_event_id', ?,
			'user', (SELECT JSON_OBJECT('name', name, 'email', email, 'gravatar_url', gravatar_url) FROM users WHERE id = ?)
		)
	)`

		insertVAUAStmt = `
INSERT INTO vpp_app_upcoming_activities
	(upcoming_activity_id, adam_id, platform, policy_id)
VALUES
	(?, ?, ?, ?)`

		hostExistsStmt = `SELECT 1 FROM hosts WHERE id = ?`
	)

	// we need to explicitly do this check here because we can't set a FK constraint on the schema
	var hostExists bool
	err := sqlx.GetContext(ctx, ds.reader(ctx), &hostExists, hostExistsStmt, hostID)
	if err != nil {
		if err == sql.ErrNoRows {
			return notFound("Host").WithID(hostID)
		}

		return ctxerr.Wrap(ctx, err, "checking if host exists")
	}

	var userID *uint
	if ctxUser := authz.UserFromContext(ctx); ctxUser != nil && opts.PolicyID == nil {
		userID = &ctxUser.ID
	}

	err = ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		res, err := tx.ExecContext(ctx, insertUAStmt,
			hostID,
			opts.Priority(),
			userID,
			opts.IsFleetInitiated(),
			commandUUID,
			opts.SelfService,
			associatedEventID,
			userID,
		)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "insert vpp install request")
		}

		activityID, _ := res.LastInsertId()
		_, err = tx.ExecContext(ctx, insertVAUAStmt,
			activityID,
			appID.AdamID,
			appID.Platform,
			opts.PolicyID,
		)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "insert vpp install request join table")
		}

		if _, err := ds.activateNextUpcomingActivity(ctx, tx, hostID, ""); err != nil {
			return ctxerr.Wrap(ctx, err, "activate next activity")
		}
		return nil
	})
	return err
}

func (ds *Datastore) MapAdamIDsPendingInstall(ctx context.Context, hostID uint) (map[string]struct{}, error) {
	var adamIds []string
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &adamIds, `SELECT hvsi.adam_id
			FROM host_vpp_software_installs hvsi
			JOIN nano_view_queue nvq ON nvq.command_uuid = hvsi.command_uuid AND nvq.status IS NULL
			WHERE hvsi.host_id = ?`, hostID); err != nil && err != sql.ErrNoRows {
		return nil, ctxerr.Wrap(ctx, err, "list pending VPP installs")
	}
	adamMap := map[string]struct{}{}
	for _, id := range adamIds {
		adamMap[id] = struct{}{}
	}

	return adamMap, nil
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
	hvsi.self_service AS self_service,
	hvsi.policy_id AS policy_id,
	p.name AS policy_name
FROM
	host_vpp_software_installs hvsi
	LEFT OUTER JOIN users u ON hvsi.user_id = u.id
	LEFT OUTER JOIN host_display_names hdn ON hdn.host_id = hvsi.host_id
	LEFT OUTER JOIN vpp_apps vpa ON hvsi.adam_id = vpa.adam_id
	LEFT OUTER JOIN software_titles st ON st.id = vpa.title_id
	LEFT OUTER JOIN policies p ON p.id = hvsi.policy_id
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
		PolicyID        *uint   `db:"policy_id"`
		PolicyName      *string `db:"policy_name"`
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
		PolicyID:        res.PolicyID,
		PolicyName:      res.PolicyName,
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
	// Delete all apps, and associated policy automations, associated with a token if we change its team
	stmtRemovePolicyAutomations := `UPDATE policies p
		JOIN vpp_apps_teams vat ON vat.id = p.vpp_apps_teams_id AND vat.vpp_token_id = ?
		SET vpp_apps_teams_id = NULL`
	stmtDeleteApps := `DELETE FROM vpp_apps_teams WHERE vpp_token_id = ? %s`

	var teamsFilter string
	if len(teams) > 0 {
		teamsFilter = "AND global_or_team_id NOT IN (?)"
	}

	stmtDeleteApps = fmt.Sprintf(stmtDeleteApps, teamsFilter)

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

		if _, err := tx.ExecContext(ctx, stmtRemovePolicyAutomations, id); err != nil {
			return ctxerr.Wrap(ctx, err, "deleting old vpp team apps policy automations")
		}

		delArgs := []any{id}
		if len(teams) > 0 {
			inStmt, inArgs, err := sqlx.In(stmtDeleteApps, id, teams)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "building IN statement for deleting old vpp apps teams associations")
			}

			stmtDeleteApps = inStmt
			delArgs = inArgs
		}

		if _, err := tx.ExecContext(ctx, stmtDeleteApps, delArgs...); err != nil {
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
	return ds.withTx(ctx, func(tx sqlx.ExtContext) error {
		_, err := tx.ExecContext(ctx, `UPDATE policies p
			JOIN vpp_apps_teams vat ON vat.id = p.vpp_apps_teams_id AND vat.vpp_token_id = ?
			SET vpp_apps_teams_id = NULL`, tokenID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "removing policy automations associated with vpp token")
		}

		_, err = tx.ExecContext(ctx, `DELETE FROM vpp_tokens WHERE id = ?`, tokenID)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "deleting vpp token")
		}

		return nil
	})
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

func (ds *Datastore) GetIncludedHostIDMapForVPPApp(ctx context.Context, vppAppTeamID uint) (map[uint]struct{}, error) {
	return ds.getIncludedHostIDMapForSoftware(ctx, ds.writer(ctx), vppAppTeamID, softwareTypeVPP)
}

func (ds *Datastore) GetIncludedHostIDMapForVPPAppTx(ctx context.Context, tx sqlx.ExtContext, vppAppTeamID uint) (map[uint]struct{}, error) {
	return ds.getIncludedHostIDMapForSoftware(ctx, tx, vppAppTeamID, softwareTypeVPP)
}

func (ds *Datastore) GetExcludedHostIDMapForVPPApp(ctx context.Context, vppAppTeamID uint) (map[uint]struct{}, error) {
	return ds.getExcludedHostIDMapForSoftware(ctx, vppAppTeamID, softwareTypeVPP)
}

func (ds *Datastore) GetAllVPPApps(ctx context.Context) ([]*fleet.VPPApp, error) {
	query := `
SELECT 
    adam_id, 
	title_id, 
	bundle_identifier, 
	icon_url, 
	name, 
	latest_version, 
	platform
FROM vpp_apps`

	var apps []*fleet.VPPApp
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &apps, query); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting all VPP apps")
	}

	return apps, nil
}
