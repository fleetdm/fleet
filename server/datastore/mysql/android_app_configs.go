package mysql

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/jmoiron/sqlx"
)

func updateAndroidAppConfiguration(ctx context.Context, tx sqlx.ExtContext, teamID *uint, adamID string, cfg []byte) error {
	err := validateAndroidAppConfiguration(ctx, cfg)
	if err != nil {
		// TODO(JK): Send user error message?
		return ctxerr.Wrap(ctx, err, "validating Android app configuration")
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
		configuration = VALUES(configuration)`

	_, err = tx.ExecContext(ctx, stmt, adamID, tid, globalOrTeamID, cfg)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "UpsertAndroidAppConfiguration")
	}
	return nil
}

func deleteAndroidAppConfiguration(ctx context.Context, teamID *uint, adamID string) error {
	return nil
}

//
// func (ds *Datastore) getAndroidAppConfiguration(ctx context.Context, teamID *uint, adamID string) (fleet.AndroidAppConfig, error) {
// 	return fleet.AndroidAppConfig{}, nil
// }

func validateAndroidAppConfiguration(ctx context.Context, configuration []byte) error {

	type androidAppConfig struct {
		// can be anything, doesn't matter to us as long as it's valid
		// mysql will validate it, so there is no need to do so in Go code
		ManagedConfiguration []byte `json:"managedConfiguration"`
		// TODO(JK): documentation says workProfileWidgets is an enum
		// what type do we actually get here?
		WorkProfileWidgets string `json:"workProfileWidgets"`
	}

	dec := json.NewDecoder(bytes.NewReader(configuration))
	dec.DisallowUnknownFields() // Only allow the two keys

	// TODO(JK): validate both fields for appropriate types
	// handle nil (should be unreachable, right?)

	var cfg androidAppConfig
	if err := dec.Decode(&cfg); err != nil {
		return err
	}

	return nil
}
