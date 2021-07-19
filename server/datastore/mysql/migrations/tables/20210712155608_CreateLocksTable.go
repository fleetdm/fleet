package tables

import (
	"database/sql"
	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20210712155608, Down_20210712155608)
}

func Up_20210712155608(tx *sql.Tx) error {
	if _, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS locks (
		id INT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
		name VARCHAR(255),
		owner VARCHAR(255),
		expires_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		UNIQUE KEY idx_name (name)
	)`); err != nil {
		return errors.Wrap(err, "create locks")
	}
	return nil
}

func Down_20210712155608(tx *sql.Tx) error {
	return nil
}
