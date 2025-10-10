package mysql

import (
	"context"

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
