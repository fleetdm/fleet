package tables

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/fleet"
)

func init() {
	MigrationClient.AddMigration(Up_20251209221730, Down_20251209221730)
}

func Up_20251209221730(tx *sql.Tx) error {
	// Update global config
	if err := updateAppConfigJSON(tx, func(config *fleet.AppConfig) error {
		if config != nil {
			config.MDM.MacOSUpdates.UpdateNewHosts = optjson.SetBool(config.MDM.MacOSUpdates.Configured())
		}
		return nil
	}); err != nil {
		return err
	}

	// Update team config
	rows, err := tx.Query("SELECT config, id FROM teams WHERE config IS NOT NULL")
	if err != nil {
		return fmt.Errorf("selecting team configs: %w", err)
	}
	defer rows.Close()

	type teamData struct {
		id  uint
		raw []byte
	}
	var teams []teamData
	for rows.Next() {
		var t teamData
		if err := rows.Scan(&t.raw, &t.id); err != nil {
			return fmt.Errorf("scanning teams row: %w", err)
		}
		teams = append(teams, t)
	}
	if err := rows.Close(); err != nil {
		return err
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterating team configs: %w", err)
	}

	for _, t := range teams {
		var config fleet.TeamConfig
		if err := json.Unmarshal(t.raw, &config); err != nil {
			return fmt.Errorf("unmarshalling team config: %w", err)
		}
		config.MDM.MacOSUpdates.UpdateNewHosts = optjson.SetBool(config.MDM.MacOSUpdates.Configured())

		b, err := json.Marshal(config)
		if err != nil {
			return fmt.Errorf("marshaling team config: %w", err)
		}

		if _, err := tx.Exec(`UPDATE teams SET config = ? WHERE id = ?`, b, t.id); err != nil {
			return fmt.Errorf("updating team config: %w", err)
		}
	}
	return nil
}

func Down_20251209221730(tx *sql.Tx) error {
	return nil
}
