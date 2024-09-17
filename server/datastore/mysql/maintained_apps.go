package mysql

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) UpsertMaintainedApp(ctx context.Context, app *fleet.MaintainedApp) error {
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

	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
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
		_, err = tx.ExecContext(ctx, upsertStmt, app.Name, app.Token, app.Version, app.Platform, app.InstallerURL,
			app.SHA256, app.BundleIdentifier, installScriptID, uninstallScriptID)
		return ctxerr.Wrap(ctx, err, "upsert maintained app")
	})
}

func (ds *Datastore) ListAvailableFleetMaintainedApps(ctx context.Context, teamID uint, opt fleet.ListOptions) ([]fleet.FleetMaintainedAppAvailable, *fleet.PaginationMetadata, error) {
	stmt := `
SELECT
	fla.id,
	fla.name,
	fla.version,
	fla.platform
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
		(si.platform = fla.platform AND si.team_id = ?)
		OR
		(va.platform = fla.platform AND vat.team_id = ?)
	)
)`

	stmtPaged, args := appendListOptionsWithCursorToSQL(stmt, []any{teamID, teamID}, &opt)

	var avail []fleet.FleetMaintainedAppAvailable
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &avail, stmtPaged, args...); err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "selecting available fleet managed apps")
	}

	meta := &fleet.PaginationMetadata{HasPreviousResults: opt.Page > 0}
	if len(avail) > int(opt.PerPage) {
		meta.HasNextResults = true
		avail = avail[:len(avail)-1]
	}

	return avail, meta, nil
}
