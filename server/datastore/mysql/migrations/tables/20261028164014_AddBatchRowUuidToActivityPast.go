package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20261028164014, Down_20261028164014)
}

func Up_20261028164014(tx *sql.Tx) error {
	// Correlates the rows of a bulk insert with their generated ids. Unindexed on purpose: it is only read back within the
	// inserting transaction, through a primary key range.
	if !columnExists(tx, "activity_past", "batch_row_uuid") {
		_, err := tx.Exec(`ALTER TABLE activity_past ADD COLUMN batch_row_uuid BINARY(16) NULL, ALGORITHM=INSTANT`)
		if err != nil {
			return fmt.Errorf("adding batch_row_uuid to activity_past: %w", err)
		}
	}
	return nil
}

func Down_20261028164014(tx *sql.Tx) error {
	return nil
}
