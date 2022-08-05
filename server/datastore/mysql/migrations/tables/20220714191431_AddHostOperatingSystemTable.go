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
    os_id INT UNSIGNED NOT NULL,
	FOREIGN KEY fk_operating_systems_id (os_id) REFERENCES operating_systems(id) ON DELETE CASCADE,
	INDEX idx_host_operating_system_id (os_id)
)`)
	if err != nil {
		return errors.Wrapf(err, "create table")
	}

	return nil
}

func Down_20220714191431(tx *sql.Tx) error {
	return nil
}
