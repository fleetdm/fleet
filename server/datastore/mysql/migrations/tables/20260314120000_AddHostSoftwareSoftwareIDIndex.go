package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260314120000, Down_20260314120000)
}

func Up_20260314120000(tx *sql.Tx) error {
	// Add a secondary index on software_id to speed up queries that join or filter host_software by software_id.
	if _, err := tx.Exec(`ALTER TABLE host_software ADD INDEX idx_host_software_software_id (software_id)`); err != nil {
		return fmt.Errorf("adding software_id index to host_software: %w", err)
	}
	return nil
}

func Down_20260314120000(_ *sql.Tx) error {
	return nil
}
