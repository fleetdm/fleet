package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20220330100659, Down_20220330100659)
}

func Up_20220330100659(tx *sql.Tx) error {
	_, err := tx.Exec(`
CREATE TABLE jobs (
    id INT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    name VARCHAR(255) NOT NULL,
    args JSON,
    state VARCHAR(255) NOT NULL,
    retries INT NOT NULL DEFAULT 0,
    error TEXT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
	`)
	if err != nil {
		return errors.Wrapf(err, "create table")
	}

	return nil
}

func Down_20220330100659(tx *sql.Tx) error {
	return nil
}
