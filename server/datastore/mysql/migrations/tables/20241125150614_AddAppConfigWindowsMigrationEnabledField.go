package tables

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20241125150614, Down_20241125150614)
}

func Up_20241125150614(tx *sql.Tx) error {
	var raw json.RawMessage
	var id uint
	row := tx.QueryRow(`SELECT id, json_value FROM app_config_json LIMIT 1;`)
	if err := row.Scan(&id, &raw); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("select app_config_json: %w", err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal(raw, &config); err != nil {
		return fmt.Errorf("unmarshal appconfig: %w", err)
	}

	mdm, ok := config["mdm"]
	if !ok {
		return errors.New("missing mdm section")
	}
	mdmMap, ok := mdm.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid type for mdm: %T", mdm)
	}
	mdmMap["windows_migration_enabled"] = false

	b, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("marshal updated appconfig: %w", err)
	}
	if _, err := tx.Exec(`UPDATE app_config_json SET json_value = ? WHERE id = ?;`, b, id); err != nil {
		return fmt.Errorf("update app_config_json: %w", err)
	}

	return nil
}

func Down_20241125150614(tx *sql.Tx) error {
	return nil
}
