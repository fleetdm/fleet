package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20220627104817, Down_20220627104817)
}

func Up_20220627104817(tx *sql.Tx) error {
	// there may be many batteries per host, so the primary key is an
	// auto-increment, not the host_id.
	_, err := tx.Exec(`
CREATE TABLE host_batteries (
  id            INT(10) UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  host_id       INT(10) UNSIGNED NOT NULL,
  serial_number VARCHAR(255) NOT NULL,
  cycle_count   INT(10) NOT NULL,
  health        VARCHAR(40) NOT NULL,
  created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at    TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

  UNIQUE KEY idx_host_batteries_host_id_serial_number (host_id, serial_number)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`)
	if err != nil {
		return errors.Wrapf(err, "create table")
	}

	return nil
}

func Down_20220627104817(tx *sql.Tx) error {
	return nil
}
