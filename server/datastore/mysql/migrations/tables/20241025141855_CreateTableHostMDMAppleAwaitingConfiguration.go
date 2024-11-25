package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20241025141855, Down_20241025141855)
}

func Up_20241025141855(tx *sql.Tx) error {
	_, err := tx.Exec(`
CREATE TABLE host_mdm_apple_awaiting_configuration (
	host_uuid           VARCHAR(255) NOT NULL PRIMARY KEY,
	awaiting_configuration TINYINT(1) NOT NULL DEFAULT FALSE
)`)
	if err != nil {
		return fmt.Errorf("creating host_mdm_apple_awaiting_configuration  table: %w", err)
	}

	return nil
}

func Down_20241025141855(tx *sql.Tx) error {
	return nil
}
