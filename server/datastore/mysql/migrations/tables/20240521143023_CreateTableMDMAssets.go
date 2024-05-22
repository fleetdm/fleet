package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20240521143023, Down_20240521143023)
}

func Up_20240521143023(tx *sql.Tx) error {
	_, err := tx.Exec(`
CREATE TABLE mdm_config_assets (
    id int(10) unsigned NOT NULL AUTO_INCREMENT PRIMARY KEY, 

    -- name is used for humans to identify what value is stored in this row
    name varchar(256) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',

    -- value holds the raw value of the asset
    value blob NOT NULL,

    -- this table does soft deletes, and the application logic is in charge of
    -- preventing INSERTs of two rows with the same name that are not deleted
    deleted_at timestamp NULL DEFAULT NULL,

    created_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP
)`)
	if err != nil {
		return fmt.Errorf("creating mdm_config_assets table: %w", err)
	}

	return nil
}

func Down_20240521143023(tx *sql.Tx) error {
	return nil
}
