package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20220714191431, Down_20220714191431)
}

func Up_20220714191431(tx *sql.Tx) error {
	_, err := tx.Exec(`
CREATE TABLE host_operating_system (
    host_id INT UNSIGNED NOT NULL PRIMARY KEY,
    os_id INT UNSIGNED NOT NULL
)
	`)
	if err != nil {
		return errors.Wrapf(err, "create table")
	}

	return nil
}

func Down_20220714191431(tx *sql.Tx) error {
	return nil
}
