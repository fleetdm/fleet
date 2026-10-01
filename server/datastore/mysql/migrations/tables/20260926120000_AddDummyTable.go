package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20260926120000, Down_20260926120000)
}

func Up_20260926120000(tx *sql.Tx) error {
	_, err := tx.Exec(`
CREATE TABLE IF NOT EXISTS dummy (
  id INT UNSIGNED NOT NULL AUTO_INCREMENT,
  PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`)
	return err
}

func Down_20260926120000(tx *sql.Tx) error {
	return nil
}
