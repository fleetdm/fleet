package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260331000000, Down_20260331000000)
}

func Up_20260331000000(tx *sql.Tx) error {
	_, err := tx.Exec(`ALTER TABLE host_certificate_templates ADD COLUMN retry_count INT UNSIGNED NOT NULL DEFAULT 0`)
	if err != nil {
		return fmt.Errorf("adding retry_count to host_certificate_templates: %w", err)
	}
	return nil
}

func Down_20260331000000(tx *sql.Tx) error {
	return nil
}
