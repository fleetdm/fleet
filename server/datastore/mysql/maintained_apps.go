package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"

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
		if err != nil {
			return ctxerr.Wrap(ctx, err, "upsert maintained app")
		}
		id, _ := res.LastInsertId()
		appID = uint(id) //nolint:gosec // dismiss G115
		return nil
	})
	if err != nil {
		return nil, err
	}

	app.ID = appID
	return app, nil
}

const teamFMATitlesJoin = `
			team_titles.id software_title_id FROM fleet_library_apps fla
			LEFT JOIN (
				SELECT DISTINCT st.id, st.bundle_identifier, st.name
				FROM software_titles st
				LEFT JOIN
					software_installers si
					ON si.title_id = st.id AND si.global_or_team_id = ?
					AND si.platform IN ('darwin','windows')
				LEFT JOIN
					vpp_apps va
					ON va.title_id = st.id
					AND va.platform = 'darwin'
				LEFT JOIN
					vpp_apps_teams vat
					ON vat.adam_id = va.adam_id
					AND vat.platform = va.platform
					AND vat.global_or_team_id = ?
				WHERE si.id IS NOT NULL OR vat.id IS NOT NULL
			) team_titles ON (
				team_titles.bundle_identifier != '' AND team_titles.bundle_identifier = fla.bundle_identifier
			) OR (
				team_titles.bundle_identifier = '' AND team_titles.name = fla.name
			)`

func (ds *Datastore) GetMaintainedAppByID(ctx context.Context, appID uint, teamID *uint) (*fleet.MaintainedApp, error) {
	stmt := `SELECT fla.id, fla.name, fla.token, fla.version, fla.platform, fla.installer_url, fla.sha256, fla.bundle_identifier,
		sc1.contents AS install_script, sc2.contents AS uninstall_script, `
	var args []any

	if teamID != nil {
		stmt += teamFMATitlesJoin
		args = []any{teamID, teamID}
	} else {
		stmt += `NULL software_title_id FROM fleet_library_apps fla`
	}

	stmt += `
JOIN script_contents sc1 ON sc1.id = fla.install_script_content_id
JOIN script_contents sc2 ON sc2.id = fla.uninstall_script_content_id
WHERE
	fla.id = ?
	`
	args = append(args, appID)

	var app fleet.MaintainedApp
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &app, stmt, args...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ctxerr.Wrap(ctx, notFound("MaintainedApp"), "no matching maintained app found")
		}

		return nil, ctxerr.Wrap(ctx, err, "getting maintained app by id")
	}

	return &app, nil
}

// NoMaintainedAppsInDatabase is the error type for no Fleet Maintained Apps in the database
type NoMaintainedAppsInDatabase struct {
	fleet.ErrorWithUUID
}

// Error implements the error interface.
func (e *NoMaintainedAppsInDatabase) Error() string {
	return `Fleet was unable to ingest the maintained apps list. Run fleetctl trigger name=maintained_apps to try repopulating the apps list.`
}

// StatusCode implements the go-kit http StatusCoder interface.
func (e *NoMaintainedAppsInDatabase) StatusCode() int {
	return http.StatusNotFound
}

func (ds *Datastore) ListAvailableFleetMaintainedApps(ctx context.Context, teamID *uint, opt fleet.ListOptions) ([]fleet.MaintainedApp, *fleet.PaginationMetadata, error) {
	stmt := `SELECT fla.id, fla.name, fla.platform, `
	var args []any

	if teamID != nil {
		stmt += teamFMATitlesJoin + ` WHERE TRUE`
		args = []any{teamID, teamID}
	} else {
		stmt += `NULL software_title_id FROM fleet_library_apps fla`
	}

	if match := opt.MatchQuery; match != "" {
		match = likePattern(match)
		stmt += ` AND (fla.name LIKE ?)`
		args = append(args, match)
	}

	// perform a second query to grab the filtered count. Build the count statement before
	// adding the pagination constraints to the stmt but after including the
	// MatchQuery option sql.
	dbReader := ds.reader(ctx)
	getAppsCountStmt := fmt.Sprintf(`SELECT COUNT(DISTINCT s.id) FROM (%s) AS s`, stmt)
	var filteredCount int
	if err := sqlx.GetContext(ctx, dbReader, &filteredCount, getAppsCountStmt, args...); err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "get fleet maintained apps count")
	}

	if filteredCount == 0 { // check if we have nothing in the full apps list, in which case provide an error back
		var totalCount int
		if err := sqlx.GetContext(ctx, dbReader, &totalCount, `SELECT COUNT(id) FROM fleet_library_apps`); err != nil {
			return nil, nil, ctxerr.Wrap(ctx, err, "get fleet maintained apps total count")
		}

		if totalCount == 0 {
			return nil, nil, &NoMaintainedAppsInDatabase{}
		}
	}

	stmtPaged, args := appendListOptionsWithCursorToSQL(stmt, args, &opt)

	var avail []fleet.MaintainedApp
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &avail, stmtPaged, args...); err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "selecting available fleet managed apps")
	}

	return avail, &fleet.PaginationMetadata{
		HasPreviousResults: opt.Page > 0,
		HasNextResults:     uint(filteredCount) > (opt.Page+1)*opt.PerPage,
		TotalResults:       uint(filteredCount), //nolint:gosec // dismiss G115
	}, nil
}
