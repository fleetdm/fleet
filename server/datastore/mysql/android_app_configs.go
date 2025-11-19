package mysql

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) updateAndroidAppConfiguration(ctx context.Context, tx sqlx.ExtContext, teamID *uint, adamID string, configuration json.RawMessage) error {
	err := validateAndroidAppConfiguration(configuration)
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

func validateAndroidAppConfiguration(configuration json.RawMessage) error {
	type androidAppConfig struct {
		ManagedConfiguration json.RawMessage `json:"managedConfiguration"`
		WorkProfileWidgets   string          `json:"workProfileWidgets"`
	}

	var res androidAppConfig
	if err := fleet.JSONStrictDecode(bytes.NewReader(configuration), &res); err != nil {
		return err
	}

	// TODO(JK): validate both fields for appropriate types
	// handle nil (should be unreachable, right?)

	return nil
}
