package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260721164157, Down_20260721164157)
}

func Up_20260721164157(tx *sql.Tx) error {
	// Password reset tokens are base64url-encoded, so their alphabet is
	// case-sensitive. The column defaulted to the case-insensitive
	// utf8mb4_unicode_ci collation, which made lookups match case-mutated
	// tokens. Switch to utf8mb4_bin so comparisons are byte-exact.
	if _, err := tx.Exec(`
		ALTER TABLE password_reset_requests
		MODIFY token VARCHAR(1024) COLLATE utf8mb4_bin NOT NULL
	`); err != nil {
		return fmt.Errorf("alter password_reset_requests token to case-sensitive collation: %w", err)
	}
	return nil
}

func Down_20260721164157(tx *sql.Tx) error {
	return nil
}
