package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20241203130032, Down_20241203130032)
}

func Up_20241203130032(tx *sql.Tx) error {
	_, err := tx.Exec(`ALTER TABLE users ADD COLUMN mfa_enabled BOOLEAN NOT NULL DEFAULT FALSE`)
	if err != nil {
		return fmt.Errorf("failed to add mfa_enabled column to users: %w", err)
	}

	_, err = tx.Exec(`ALTER TABLE invites ADD COLUMN mfa_enabled BOOLEAN NOT NULL DEFAULT FALSE`)
	if err != nil {
		return fmt.Errorf("failed to add mfa_enabled column to invites: %w", err)
	}

	_, err = tx.Exec(`CREATE TABLE verification_tokens (
  id INT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  user_id INT UNSIGNED NOT NULL,
  token varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL UNIQUE,
  created_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  CONSTRAINT verification_tokens_users FOREIGN KEY (user_id) REFERENCES users (id) ON UPDATE CASCADE ON DELETE CASCADE
)`)
	if err != nil {
		return fmt.Errorf("failed to craete verification_tokens table: %w", err)
	}

	return nil
}

func Down_20241203130032(tx *sql.Tx) error {
	return nil
}
