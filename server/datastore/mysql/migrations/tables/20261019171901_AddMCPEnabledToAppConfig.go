package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20261019171901, Down_20261019171901)
}

func Up_20261019171901(tx *sql.Tx) error {
	if _, err := tx.Exec(`UPDATE app_config_json SET json_value = JSON_SET(json_value, '$.server_settings.mcp_enabled', CAST('true' AS JSON)) WHERE id = 1`); err != nil {
		return fmt.Errorf("set server_settings.mcp_enabled: %w", err)
	}
	return nil
}

func Down_20261019171901(tx *sql.Tx) error {
	return nil
}
