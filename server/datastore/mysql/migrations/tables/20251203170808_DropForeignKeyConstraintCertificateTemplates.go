package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20251203170808, Down_20251203170808)
}

func Up_20251203170808(tx *sql.Tx) error {
	_, err := tx.Exec(
		`ALTER TABLE certificate_templates DROP FOREIGN KEY certificate_templates_ibfk_1;`)
	if err != nil {
		return fmt.Errorf("failed to drop foreign key constraint on table certificate_templates: %w", err)
	}
	return nil
}

func Down_20251203170808(tx *sql.Tx) error {
	return nil
}
