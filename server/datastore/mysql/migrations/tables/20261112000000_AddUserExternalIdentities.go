package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20261112000000, Down_20261112000000)
}

func Up_20261112000000(tx *sql.Tx) error {
	// The primary key is (issuer, subject) rather than including user_id so that
	// one external identity can never resolve to two users.
	if _, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS user_external_identities (
			user_id    INT UNSIGNED NOT NULL,
			issuer     VARCHAR(255) COLLATE utf8mb4_unicode_ci NOT NULL,
			subject    VARCHAR(255) COLLATE utf8mb4_unicode_ci NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (issuer, subject),
			KEY idx_user_external_identities_user_id (user_id),
			CONSTRAINT user_external_identities_ibfk_1
				FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
	`); err != nil {
		return fmt.Errorf("creating user_external_identities table: %w", err)
	}
	return nil
}

func Down_20261112000000(_ *sql.Tx) error {
	return nil
}
