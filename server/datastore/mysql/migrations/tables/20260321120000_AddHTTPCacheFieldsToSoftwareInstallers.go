package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260321120000, Down_20260321120000)
}

func Up_20260321120000(tx *sql.Tx) error {
	_, err := tx.Exec(`ALTER TABLE software_installers
		ADD COLUMN http_etag VARCHAR(255) DEFAULT NULL,
		ADD COLUMN http_last_modified VARCHAR(255) DEFAULT NULL`)
	if err != nil {
		return fmt.Errorf("failed to add http cache columns to software_installers: %w", err)
	}
	return nil
}

func Down_20260321120000(tx *sql.Tx) error {
	return nil
}
