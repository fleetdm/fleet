package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20250629131032, Down_20250629131032)
}

func Up_20250629131032(tx *sql.Tx) error {
	_, err := tx.Exec(
		"ALTER TABLE `labels` " +
			"ADD COLUMN `criteria` json DEFAULT NULL; ",
	)
	if err != nil {
		return fmt.Errorf("failed to add criteria column to labels table: %w", err)
	}
	return nil
}

func Down_20250629131032(tx *sql.Tx) error {
	return nil
}
