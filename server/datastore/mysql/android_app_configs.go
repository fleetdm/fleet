package mysql

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) UpsertAndroidAppConfigurationTx(ctx context.Context, tx sqlx.ExtContext, teamID *uint, adamID string, configuration json.RawMessage) error {
	err := validateAndroidAppConfiguration(ctx, configuration)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "validating android app configuration")
	}

	var tid *uint
	var globalOrTeamID uint
	if teamID != nil {
		globalOrTeamID = *teamID

		if *teamID > 0 {
			tid = teamID
		}
	}

	stmt := `
		INSERT INTO 
			android_app_configurations (adam_id, team_id, global_or_team_id, configuration)
		VALUES (?, ?, ?, ?) 
		ON DUPLICATE KEY UPDATE 
			configuration = VALUES(configuration)
	`

	_, err = tx.ExecContext(ctx, stmt, adamID, tid, globalOrTeamID, configuration)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "UpsertAndroidAppConfiguration")
	}
	return nil
}

func (ds *Datastore) DeleteAndroidAppConfiguration(ctx context.Context, teamID *uint, adamID string) error {
	var globalOrTeamID uint
	if teamID != nil {
		globalOrTeamID = *teamID
	}

	// TODO(JK): should this use team ID or global?
	stmt := `
		DELETE FROM android_app_configurations
		WHERE adam_id = ? AND global_or_team_id = ?
	`

	result, err := ds.writer(ctx).ExecContext(ctx, stmt, adamID, globalOrTeamID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "deleting android app configuration")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting rows affected")
	}

	if rowsAffected == 0 {
		return ctxerr.Wrap(ctx, notFound("AndroidAppConfiguration"), "android app configuration not found")
	}

	return nil
}

func (ds *Datastore) GetAndroidAppConfiguration(ctx context.Context, teamID *uint, adamID string) (cfg json.RawMessage, err error) {
	var globalOrTeamID uint
	if teamID != nil {
		globalOrTeamID = *teamID
	}

	stmt := `
		SELECT configuration FROM android_app_configurations
		WHERE adam_id = ? AND global_or_team_id = ?
	`

	err = sqlx.GetContext(ctx, ds.reader(ctx), &cfg, stmt, adamID, globalOrTeamID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "getting android app configuration")
	}

	return cfg, nil
}

func validateAndroidAppConfiguration(ctx context.Context, configuration json.RawMessage) error {
	type androidAppConfig struct {
		// can be anything, doesn't matter to us as long as it's valid
		// mysql will validate it, so there is no need to do so in Go code
		ManagedConfiguration []byte `json:"managedConfiguration"`
		// TODO(JK): documentation says workProfileWidgets is an enum
		// what type do we actually get here?
		WorkProfileWidgets string `json:"workProfileWidgets"`
	}

	var res androidAppConfig
	if err := fleet.JSONStrictDecode(bytes.NewReader(configuration), &res); err != nil {
		return err
	}

	// TODO(JK): validate both fields for appropriate types
	// handle nil (should be unreachable, right?)

	return nil
}
