package tables

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20260121151237, Down_20260121151237)
}

func Up_20260121151237(tx *sql.Tx) error {
	// Update global app config
	var raw json.RawMessage
	var id uint
	row := tx.QueryRow(`SELECT id, json_value FROM app_config_json LIMIT 1`)
	if err := row.Scan(&id, &raw); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// No config exists, skip
		} else {
			return fmt.Errorf("select app_config_json: %w", err)
		}
	} else {
		var config map[string]interface{}
		if err := json.Unmarshal(raw, &config); err != nil {
			return fmt.Errorf("unmarshal app_config_json: %w", err)
		}

		// Navigate to mdm.macos_setup and add lock_primary_account_info if not present
		if mdm, ok := config["mdm"].(map[string]interface{}); ok {
			if macosSetup, ok := mdm["macos_setup"].(map[string]interface{}); ok {
				if _, exists := macosSetup["lock_primary_account_info"]; !exists {
					macosSetup["lock_primary_account_info"] = false
					mdm["macos_setup"] = macosSetup
					config["mdm"] = mdm

					b, err := json.Marshal(config)
					if err != nil {
						return fmt.Errorf("marshal updated app_config_json: %w", err)
					}
					if _, err := tx.Exec(`UPDATE app_config_json SET json_value = ? WHERE id = ?`, b, id); err != nil {
						return fmt.Errorf("update app_config_json: %w", err)
					}
				}
			}
		}
	}

	// Update default team config
	var defaultConfigJSON json.RawMessage
	defaultRow := tx.QueryRow(`SELECT json_value FROM default_team_config_json WHERE id = 1`)
	if err := defaultRow.Scan(&defaultConfigJSON); err != nil {
		if err == sql.ErrNoRows {
			// Table might not exist yet, skip
			return nil
		}
		return fmt.Errorf("selecting default_team_config_json: %w", err)
	}

	var defaultConfig map[string]interface{}
	if err := json.Unmarshal(defaultConfigJSON, &defaultConfig); err != nil {
		return fmt.Errorf("unmarshaling default_team_config_json: %w", err)
	}

	// Navigate to mdm.macos_setup and add lock_primary_account_info if not present
	if mdm, ok := defaultConfig["mdm"].(map[string]interface{}); ok {
		if macosSetup, ok := mdm["macos_setup"].(map[string]interface{}); ok {
			if _, exists := macosSetup["lock_primary_account_info"]; !exists {
				macosSetup["lock_primary_account_info"] = false
				mdm["macos_setup"] = macosSetup
				defaultConfig["mdm"] = mdm

				updatedJSON, err := json.Marshal(defaultConfig)
				if err != nil {
					return fmt.Errorf("marshaling updated default_team_config_json: %w", err)
				}

				if _, err = tx.Exec(`UPDATE default_team_config_json SET json_value = ? WHERE id = 1`, updatedJSON); err != nil {
					return fmt.Errorf("updating default_team_config_json: %w", err)
				}
			}
		}
	}

	// Update all team configs
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
		var teamConfig map[string]interface{}
		if err := json.Unmarshal(t.raw, &teamConfig); err != nil {
			return fmt.Errorf("unmarshalling team config: %w", err)
		}

		// Navigate to mdm.macos_setup and add lock_primary_account_info if not present
		if mdm, ok := teamConfig["mdm"].(map[string]interface{}); ok {
			if macosSetup, ok := mdm["macos_setup"].(map[string]interface{}); ok {
				if _, exists := macosSetup["lock_primary_account_info"]; !exists {
					macosSetup["lock_primary_account_info"] = false
					mdm["macos_setup"] = macosSetup
					teamConfig["mdm"] = mdm

					b, err := json.Marshal(teamConfig)
					if err != nil {
						return fmt.Errorf("marshaling team config: %w", err)
					}

					if _, err := tx.Exec(`UPDATE teams SET config = ? WHERE id = ?`, b, t.id); err != nil {
						return fmt.Errorf("updating team config: %w", err)
					}
				}
			}
		}
	}

	return nil
}

func Down_20260121151237(tx *sql.Tx) error {
	// No down migration needed - we can leave the field in place
	return nil
}

