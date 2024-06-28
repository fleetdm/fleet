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
    value longblob NOT NULL,

    -- deleted_at is used to track the date in which the row was marked as
    -- deleted for auditing/debugging purposes.
    deleted_at timestamp NULL DEFAULT NULL,

    -- deletion_uuid is used as part of an UNIQUE KEY to guarantee that only
    -- one non-deleted row with a given name exists. This value should be filled
    -- along with deleted_at
    deletion_uuid varchar(127) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',

    -- md5_checksum holds the binary checksum of the value column.
    md5_checksum  BINARY(16) NOT NULL,

    created_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,

    UNIQUE KEY idx_mdm_config_assets_name_deletion_uuid (name, deletion_uuid)
)`)
	if err != nil {
		return fmt.Errorf("creating mdm_config_assets table: %w", err)
	}

	return nil
}

func Down_20240521143023(tx *sql.Tx) error {
	return nil
}
