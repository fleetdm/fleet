package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20240307122512, Down_20240307122512)
}

func Up_20240307122512(tx *sql.Tx) error {
	stmt := `
ALTER TABLE host_script_results
	-- host_deleted_at is the time the host was deleted
	ADD COLUMN host_deleted_at TIMESTAMP NULL,

	ADD INDEX idx_hsr_host_deleted_at (host_deleted_at);`

	if _, err := tx.Exec(stmt); err != nil {
		return errors.Wrap(err, "alter host_script_results table")
	}

	return nil
}

func Down_20240307122512(tx *sql.Tx) error {
	return nil
}
