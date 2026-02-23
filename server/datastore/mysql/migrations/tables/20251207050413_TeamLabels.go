package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20251207050413, Down_20251207050413)
}

func Up_20251207050413(tx *sql.Tx) error {
	// lack of delete cascade is intentional here as label membership needs to be cleaned up when deleting labels, and
	// there isn't a foreign key relationship back to labels on that table
	_, err := tx.Exec(
		"ALTER TABLE `labels` " +
			"ADD COLUMN `team_id` int unsigned NULL DEFAULT NULL, " +
			"ADD CONSTRAINT FOREIGN KEY (`team_id`) REFERENCES `teams` (`id`);",
	)
	if err != nil {
		return fmt.Errorf("failed to add team_id column to labels table: %w", err)
	}

	return nil
}

func Down_20251207050413(tx *sql.Tx) error {
	return nil
}
