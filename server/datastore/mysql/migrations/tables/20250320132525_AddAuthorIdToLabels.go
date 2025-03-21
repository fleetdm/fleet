package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20250311132525, Down_20250311132525)
}

func Up_20250311132525(tx *sql.Tx) error {
	_, err := tx.Exec(
		"ALTER TABLE `labels` " +
			"ADD COLUMN `author_id` int unsigned NULL DEFAULT NULL, " +
			"ADD CONSTRAINT FOREIGN KEY (`author_id`) REFERENCES `users` (`id`) ON DELETE SET NULL; ",
	)
	if err != nil {
		return fmt.Errorf("failed to add author_id column to labels table: %w", err)
	}

	_, err = tx.Exec(`ALTER TABLE labels ADD INDEX (author_id)`)
	if err != nil {
		return fmt.Errorf("failed to add author_id index to labels table: %w", err)
	}

	return nil
}

func Down_20250311132525(tx *sql.Tx) error {
	return nil
}
