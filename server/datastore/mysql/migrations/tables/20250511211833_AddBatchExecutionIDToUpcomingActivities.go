package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20250511211833, Down_20250511211833)
}

func Up_20250511211833(tx *sql.Tx) error {
	const alterStmt = "ALTER TABLE `upcoming_activities` ADD COLUMN `batch_execution_id` varchar(255) NOT NULL DEFAULT FALSE AFTER `execution_id`;"
	_, err := tx.Exec(alterStmt)
	if err != nil {
		return errors.Wrapf(err, "adding column batch_execution_id to upcoming_activities table")
	}
	const indexStmt = `
ALTER TABLE upcoming_activities ADD INDEX idx_batch_execution_id (batch_execution_id), ALGORITHM=INPLACE, LOCK=NONE;`
	_, err = tx.Exec(indexStmt)
	if err != nil {
		return errors.Wrapf(err, "adding index idx_batch_execution_id on batch_execution_id to upcoming_activities table")
	}
	return nil
}

func Down_20250511211833(tx *sql.Tx) error {
	return nil
}
