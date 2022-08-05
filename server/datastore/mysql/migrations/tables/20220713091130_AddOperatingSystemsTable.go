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
    name VARCHAR(191) NOT NULL,
	version VARCHAR(191) NOT NULL,
	arch VARCHAR(191) NOT NULL,
	kernel_version VARCHAR(191) NOT NULL,
    UNIQUE KEY idx_unique_os (name, version, arch, kernel_version)
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
