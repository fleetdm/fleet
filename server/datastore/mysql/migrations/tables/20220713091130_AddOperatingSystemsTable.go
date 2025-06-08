package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20220713091130, Down_20220713091130)
}

func Up_20220713091130(tx *sql.Tx) error {
	// Length of VARCHAR set to conform with max key length limitations for the unique constraint
	_, err := tx.Exec(`
CREATE TABLE operating_systems (
    id INT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
	version VARCHAR(150) NOT NULL,
	arch VARCHAR(150) NOT NULL,
	kernel_version VARCHAR(150) NOT NULL,
	platform VARCHAR(50) NOT NULL,
    UNIQUE KEY idx_unique_os (name, version, arch, kernel_version, platform)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
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
