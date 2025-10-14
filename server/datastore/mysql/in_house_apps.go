package mysql

import (
	"context"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) insertInHouseApp(ctx context.Context, payload *fleet.InHouseAppPayload) (uint, uint, error) {

	stmt := `
	INSERT INTO in_house_apps (
		team_id,
		title_id,
		global_or_team_id,
		name,
		storage_id,
		platform,
		version
	)
	VALUES (?, ?, ?, ?, ?, ?, ?)`

	var tid *uint
	var globalOrTeamID uint
	if payload.TeamID != nil {
		globalOrTeamID = *payload.TeamID

		if *payload.TeamID > 0 {
			tid = payload.TeamID
		}
	}

	titleID, err := ds.getOrGenerateSoftwareInstallerTitleID(ctx, &fleet.UploadSoftwareInstallerPayload{
		TeamID:           tid,
		Title:            payload.Name,
		BundleIdentifier: payload.BundleID,
		Source:           "ios_apps"}, // TODO: what about iPad apps
	)
	if err != nil {
		return 0, 0, ctxerr.Wrap(ctx, err, "insertInHouseApp")
	}

	var installerID uint
	err = ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		args := []any{
			tid,
			titleID,
			globalOrTeamID,
			payload.Name,
			payload.StorageID,
			payload.Platform,
			payload.Version,
		}

		res, err := tx.ExecContext(ctx, stmt, args...)
		if err != nil {
			if IsDuplicate(err) {
				// already exists for this team/no team
				err = alreadyExists("InHouseApp", payload.Name)
			}
			return ctxerr.Wrap(ctx, err, "insertInHouseApp")
		}

		id64, err := res.LastInsertId()
		installerID = uint(id64) //nolint:gosec // dismiss G115
		if err != nil {
			return ctxerr.Wrap(ctx, err, "insertInHouseApp")
		}

		if err := setOrUpdateSoftwareInstallerLabelsDB(ctx, tx, installerID, *payload.ValidatedLabels, softwareTypeInHouseApp); err != nil {
			return ctxerr.Wrap(ctx, err, "upsert in house app labels")
		}

		return nil
	})

	return installerID, titleID, ctxerr.Wrap(ctx, err, "insertInHouseApp")
}

// hihsiAlias is the table alias to use as prefix for the
// host_in_house_software_installs column names, no prefix used if empty.
// ncrAlias is the table alias to use as prefix for the nano_command_results
// column names, no prefix used if empty.
// colAlias is the name to be assigned to the computed status column, pass
// empty to have the value only, no column alias set.
func inHouseAppHostStatusNamedQuery(hihsiAlias, ncrAlias, colAlias string) string {
	if hihsiAlias != "" {
		hihsiAlias += "."
	}
	if ncrAlias != "" {
		ncrAlias += "."
	}
	if colAlias != "" {
		colAlias = " AS " + colAlias
	}

	return fmt.Sprintf(`
	CASE
		WHEN %sverification_at IS NOT NULL THEN
			:software_status_installed
		WHEN %sverification_failed_at IS NOT NULL THEN
			:software_status_failed
		WHEN %sstatus = :mdm_status_error OR %sstatus = :mdm_status_format_error THEN
			:software_status_failed
		ELSE
			:software_status_pending
	END %s
	`, hihsiAlias, hihsiAlias, ncrAlias, ncrAlias, colAlias)
}
