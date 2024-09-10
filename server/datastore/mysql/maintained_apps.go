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
