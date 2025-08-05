package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20250731122715, Down_20250731122715)
}

func Up_20250731122715(tx *sql.Tx) error {
	// Update subject_country column from varchar(2) to varchar(32)
	if _, err := tx.Exec(`ALTER TABLE host_certificates MODIFY COLUMN subject_country varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL`); err != nil {
		return fmt.Errorf("failed to modify subject_country column: %w", err)
	}

	// Update issuer_country column from varchar(2) to varchar(32)
	if _, err := tx.Exec(`ALTER TABLE host_certificates MODIFY COLUMN issuer_country varchar(32) COLLATE utf8mb4_unicode_ci NOT NULL`); err != nil {
		return fmt.Errorf("failed to modify issuer_country column: %w", err)
	}

	return nil
}

func Down_20250731122715(_ *sql.Tx) error {
	return nil
}
