package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260422150000, Down_20260422150000)
}

func Up_20260422150000(tx *sql.Tx) error {
	_, err := tx.Exec(`ALTER TABLE vpp_apps ADD COLUMN metadata_region VARCHAR(8) DEFAULT NULL`)
	if err != nil {
		return fmt.Errorf("failed to add metadata_region to vpp_apps: %w", err)
	}
	return nil
}

func Down_20260422150000(tx *sql.Tx) error {
	return nil
}
