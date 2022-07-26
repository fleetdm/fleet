package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20220713091130, Down_20220713091130)
}

func Up_20220713091130(tx *sql.Tx) error {
	_, err := tx.Exec(`
CREATE TABLE operating_systems (
    id INT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
	version VARCHAR(255) NOT NULL,
	arch VARCHAR(255) NOT NULL,
	kernel_version VARCHAR(255) NOT NULL
)
	`)
	if err != nil {
		return errors.Wrapf(err, "create table")
	}

	return nil
}

func Down_20220713091130(tx *sql.Tx) error {
	return nil
}
