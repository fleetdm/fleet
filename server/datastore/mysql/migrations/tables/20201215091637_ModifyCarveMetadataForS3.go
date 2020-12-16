package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20201215091637, Down_20201215091637)
}

func Up_20201215091637(tx *sql.Tx) error {
	query := `
		ALTER TABLE carve_metadata
		ADD max_block INT DEFAULT -1,
		MODIFY session_id VARCHAR(255) NOT NULL;
	`
	if _, err := tx.Exec(query); err != nil {
		return errors.Wrap(err, "alter carve session_id size")
	}

	return nil
}

func Down_20201215091637(tx *sql.Tx) error {
	query := `
		ALTER TABLE carve_metadata
		DROP max_block,
		MODIFY session_id VARCHAR(64) NOT NULL;
	`
	if _, err := tx.Exec(query); err != nil {
		return errors.Wrap(err, "revert carve session_id size")
	}

	return nil
}
