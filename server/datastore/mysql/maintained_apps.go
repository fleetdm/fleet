package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) UpsertMaintainedApp(ctx context.Context, app *fleet.MaintainedApp) (*fleet.MaintainedApp, error) {
	const upsertStmt = `
INSERT INTO
	fleet_library_apps (
		name, token, version, platform, installer_url,
		sha256, bundle_identifier, install_script_content_id, uninstall_script_content_id
	)
VALUES
	( ?, ?, ?, ?, ?,
	  ?, ?, ?, ? )
ON DUPLICATE KEY UPDATE
	name = VALUES(name),
	version = VALUES(version),
	platform = VALUES(platform),
	installer_url = VALUES(installer_url),
	sha256 = VALUES(sha256),
	bundle_identifier = VALUES(bundle_identifier),
	install_script_content_id = VALUES(install_script_content_id),
	uninstall_script_content_id = VALUES(uninstall_script_content_id)
`

	var appID uint
	err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		var err error

		// ensure the install script exists
		installRes, err := insertScriptContents(ctx, tx, app.InstallScript)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "insert install script content")
		}
		installScriptID, _ := installRes.LastInsertId()

		// ensure the uninstall script exists
		uninstallRes, err := insertScriptContents(ctx, tx, app.UninstallScript)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "insert uninstall script content")
		}
		uninstallScriptID, _ := uninstallRes.LastInsertId()

		// upsert the maintained app
		res, err := tx.ExecContext(ctx, upsertStmt, app.Name, app.Token, app.Version, app.Platform, app.InstallerURL,
			app.SHA256, app.BundleIdentifier, installScriptID, uninstallScriptID)
		id, _ := res.LastInsertId()
		appID = uint(id) //nolint:gosec // dismiss G115
		return ctxerr.Wrap(ctx, err, "upsert maintained app")
	})
	if err != nil {
		return nil, err
	}

	app.ID = appID
	return app, nil
}

func (ds *Datastore) GetMaintainedAppByID(ctx context.Context, appID uint) (*fleet.MaintainedApp, error) {
	const stmt = `
SELECT
	fla.id,
	fla.name,
	fla.token,
	fla.version,
	fla.platform,
	fla.installer_url,
	fla.sha256,
	fla.bundle_identifier,
	sc1.contents AS install_script,
	sc2.contents AS uninstall_script
FROM fleet_library_apps fla
JOIN script_contents sc1 ON sc1.id = fla.install_script_content_id
JOIN script_contents sc2 ON sc2.id = fla.uninstall_script_content_id
WHERE
	fla.id = ?
	`

	var app fleet.MaintainedApp
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &app, stmt, appID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ctxerr.Wrap(ctx, notFound("MaintainedApp"), "no matching maintained app found")
		}

		return nil, ctxerr.Wrap(ctx, err, "getting maintained app by id")
	}

	return &app, nil
}

func (ds *Datastore) ListAvailableFleetMaintainedApps(ctx context.Context, teamID uint, opt fleet.ListOptions) ([]fleet.MaintainedApp, *fleet.PaginationMetadata, error) {
	stmt := `
SELECT
	fla.id,
	fla.name,
	fla.version,
	fla.platform,
	fla.updated_at
FROM
	fleet_library_apps fla
WHERE NOT EXISTS (
	SELECT
		1
	FROM
		software_titles st
	LEFT JOIN
		software_installers si
		ON si.title_id = st.id
	LEFT JOIN
		vpp_apps va
		ON va.title_id = st.id
	LEFT JOIN
		vpp_apps_teams vat
		ON vat.adam_id = va.adam_id
	WHERE
		st.bundle_identifier = fla.bundle_identifier
	AND (
		(si.platform = fla.platform AND si.global_or_team_id = ?)
		OR
		(va.platform = fla.platform AND vat.global_or_team_id = ?)
	)
)`

	args := []any{teamID, teamID}

	if match := opt.MatchQuery; match != "" {
		match = likePattern(match)
		stmt += ` AND (fla.name LIKE ?)`
		args = append(args, match)
	}

	// perform a second query to grab the counts. Build the count statement before
	// adding the pagination constraints to the stmt but after including the
	// MatchQuery option sql.
	dbReader := ds.reader(ctx)
	getAppsCountStmt := fmt.Sprintf(`SELECT COUNT(DISTINCT s.id) FROM (%s) AS s`, stmt)
	var counts int
	if err := sqlx.GetContext(ctx, dbReader, &counts, getAppsCountStmt, args...); err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "get fleet maintained apps count")
	}

	stmtPaged, args := appendListOptionsWithCursorToSQL(stmt, args, &opt)

	var avail []fleet.MaintainedApp
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &avail, stmtPaged, args...); err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "selecting available fleet managed apps")
	}

	meta := &fleet.PaginationMetadata{HasPreviousResults: opt.Page > 0, TotalResults: uint(counts)} //nolint:gosec // dismiss G115
	if len(avail) > int(opt.PerPage) {                                                              //nolint:gosec // dismiss G115
		meta.HasNextResults = true
		avail = avail[:len(avail)-1]
	}

	return avail, meta, nil
}

// GetSoftwareTitleIDByAppID returns the software title ID related to a given fleet library app ID.
func (ds *Datastore) GetSoftwareTitleIDByMaintainedAppID(ctx context.Context, appID uint, teamID *uint) (uint, error) {
	stmt := `
	SELECT
		st.id
	FROM software_titles st
	JOIN software_installers si ON si.title_id = st.id
	JOIN fleet_library_apps fla ON fla.id = si.fleet_library_app_id
	WHERE fla.id = ? AND si.global_or_team_id = ?`

	var globalOrTeamID uint
	if teamID != nil {
		globalOrTeamID = *teamID
	}

	var titleID uint
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &titleID, stmt, appID, globalOrTeamID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ctxerr.Wrap(ctx, notFound("SoftwareInstaller"), "no matching software installer found")
		}

		return 0, ctxerr.Wrap(ctx, err, "getting software title id by app id")
	}

	return titleID, nil
}
