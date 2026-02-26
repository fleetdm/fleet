package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260226160000, Down_20260226160000)
}

func Up_20260226160000(tx *sql.Tx) error {
	if _, err := tx.Exec(`
		CREATE TABLE host_recovery_key_passwords (
			host_id int unsigned NOT NULL,
			encrypted_password BLOB NOT NULL,
			created_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
			updated_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
			PRIMARY KEY (host_id)
		)
	`); err != nil {
		return fmt.Errorf("creating host_recovery_key_passwords table: %w", err)
	}
	return nil
}

func Down_20260226160000(tx *sql.Tx) error {
	return nil
}
