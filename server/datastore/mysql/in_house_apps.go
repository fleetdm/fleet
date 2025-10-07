package mysql

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) InsertInHouseApp(ctx context.Context, payload *fleet.InHouseAppPayload) error {

	stmt := `
	INSERT INTO in_house_apps (
		team_id,
		global_or_team_id,
		name,
		storage_id,
		platform
	)
	VALUES (?, ?, ?, ?, ?)
		`

	var tid *uint
	var globalOrTeamID uint
	if payload.TeamID != nil {
		globalOrTeamID = *payload.TeamID

		if *payload.TeamID > 0 {
			tid = payload.TeamID
		}
	}

	err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		args := []any{
			tid,
			globalOrTeamID,
			payload.Name,
			payload.StorageID,
			payload.Platform,
		}

		_, err := tx.ExecContext(ctx, stmt, args...)
		if err != nil {
			if IsDuplicate(err) {
				// already exists for this team/no team
				err = alreadyExists("InHouseApp", payload.Name)
			}
			return err
		}

		return nil
	})

	return ctxerr.Wrap(ctx, err, "insert in house app")
}
