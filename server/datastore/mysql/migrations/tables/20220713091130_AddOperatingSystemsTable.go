package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20220713091130, Down_20220713091130)
}

func Up_20220713091130(tx *sql.Tx) error {
	// VARCHAR(191) is used to conform with max key length limitations for the unique constraint
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
		return errors.Wrapf(err, "create operating_systems table")
	}

	_, err = tx.Exec(`
CREATE TABLE host_operating_system (
    host_id INT UNSIGNED NOT NULL PRIMARY KEY,
    os_id INT UNSIGNED NOT NULL,
	FOREIGN KEY fk_operating_systems_id (os_id) REFERENCES operating_systems(id) ON DELETE CASCADE,
	INDEX idx_host_operating_system_id (os_id)
)`)
	if err != nil {
		return errors.Wrapf(err, "create host_operating_systems table")
	}

	return nil
}

func Down_20220713091130(tx *sql.Tx) error {
	return nil
}
