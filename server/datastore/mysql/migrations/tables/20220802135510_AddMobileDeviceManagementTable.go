package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20220802135510, Down_20220802135510)
}

func Up_20220802135510(tx *sql.Tx) error {
	_, err := tx.Exec(`
CREATE TABLE mobile_device_management_solutions (
  id            INT(10) UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  name          VARCHAR(100) NOT NULL,
  server_url    VARCHAR(255) DEFAULT '' NOT NULL,
  created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at    TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

  UNIQUE KEY idx_mobile_device_management_solutions_name (name)
)`)
	if err != nil {
		return errors.Wrapf(err, "create table")
	}

	return nil
}

func Down_20220802135510(tx *sql.Tx) error {
	return nil
}
