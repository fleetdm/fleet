package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260202151756, Down_20260202151756)
}

func Up_20260202151756(tx *sql.Tx) error {
	_, err := tx.Exec(`ALTER TABLE host_certificate_templates
		CHANGE COLUMN host_uuid host_uuid VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL;`)
	if err != nil {
		return fmt.Errorf("altering host_certificate_templates host_uuid column: %w", err)
	}
	return nil
}

func Down_20260202151756(tx *sql.Tx) error {
	return nil
}
