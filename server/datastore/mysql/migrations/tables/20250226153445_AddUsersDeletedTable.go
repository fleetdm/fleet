package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20250226153445, Down_20250226153445)
}

func Up_20250226153445(tx *sql.Tx) error {
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS users_deleted (
		    -- matches users.id, which is an auto-incrementing primary key
  			id int unsigned NOT NULL,
			name varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
			email varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
			created_at DATETIME(6) NULL DEFAULT NOW(6),
			updated_at DATETIME(6) NULL DEFAULT NOW(6) ON UPDATE NOW(6),
			PRIMARY KEY (id)
		)`)
	if err != nil {
		return fmt.Errorf("create users_deleted table: %w", err)
	}

	_, err = tx.Exec(`
		ALTER TABLE android_enterprises
		-- user that created the enterprise
		ADD COLUMN user_id int unsigned NOT NULL DEFAULT 0`)
	if err != nil {
		return fmt.Errorf("add user_id to android_enterprise table: %w", err)
	}

	return nil
}

func Down_20250226153445(_ *sql.Tx) error {
	return nil
}
