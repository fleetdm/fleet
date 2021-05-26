package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20210526113559, Down_20210526113559)
}

func Up_20210526113559(tx *sql.Tx) error {
	sql := `
		CREATE TABLE host_additional (
			host_id int unsigned NOT NULL PRIMARY KEY,
			additional json DEFAULT NULL,
			FOREIGN KEY (host_id) REFERENCES hosts (id) ON DELETE CASCADE ON UPDATE CASCADE
		)
	`
	if _, err := tx.Exec(sql); err != nil {
		return errors.Wrap(err, "create host_additional")
	}

	sql = `
		INSERT INTO host_additional (host_id, additional)
		SELECT id, additional FROM hosts
	`
	if _, err := tx.Exec(sql); err != nil {
		return errors.Wrap(err, "migration additional data")
	}

	sql = `
		ALTER TABLE hosts
		DROP COLUMN additional
	`
	if _, err := tx.Exec(sql); err != nil {
		return errors.Wrap(err, "migration additional data")
	}

	return nil
}

func Down_20210526113559(tx *sql.Tx) error {
	return nil
}
