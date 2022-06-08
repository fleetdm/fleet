package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20220518124708, Down_20220518124708)
}

func Up_20220518124708(tx *sql.Tx) error {
	_, err := tx.Exec(
		"ALTER TABLE `software_cve` ADD COLUMN `source` int DEFAULT '0'",
	)
	if err != nil {
		return fmt.Errorf("add 'source' column to 'software_cve': %w", err)
	}
	return nil
}

func Down_20220518124708(tx *sql.Tx) error {
	return nil
}
