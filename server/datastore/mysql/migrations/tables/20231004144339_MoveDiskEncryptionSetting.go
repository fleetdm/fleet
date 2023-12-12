package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20231004144339, Down_20231004144339)
}

func Up_20231004144339(tx *sql.Tx) error {
	stmt := `
UPDATE teams
SET
    config = JSON_SET(config, '$.mdm.enable_disk_encryption', 
                           JSON_EXTRACT(config, '$.mdm.macos_settings.enable_disk_encryption')),
    config = JSON_REMOVE(config, '$.mdm.macos_settings.enable_disk_encryption')
WHERE
    JSON_EXTRACT(config, '$.mdm.macos_settings.enable_disk_encryption') IS NOT NULL;
  `

	if _, err := tx.Exec(stmt); err != nil {
		return fmt.Errorf("move team mdm.macos_settings.enable_disk_encryption setting to mdm.enable_disk_encryption: %w", err)
	}

	return nil
}

func Down_20231004144339(tx *sql.Tx) error {
	return nil
}
