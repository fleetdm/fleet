package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20261111132618, Down_20261111132618)
}

func Up_20261111132618(tx *sql.Tx) error {
	// --- invites table ---

	// Add token_hash column if it doesn't exist yet.
	if !columnExists(tx, "invites", "token_hash") {
		if _, err := tx.Exec(`ALTER TABLE invites ADD COLUMN token_hash BINARY(32) NULL`); err != nil {
			return fmt.Errorf("add invites.token_hash: %w", err)
		}
	}

	// Backfill hashes from cleartext tokens (only rows that still have a token column).
	if columnExists(tx, "invites", "token") {
		if _, err := tx.Exec(`UPDATE invites SET token_hash = UNHEX(SHA2(token, 256)) WHERE token_hash IS NULL`); err != nil {
			return fmt.Errorf("backfill invites.token_hash: %w", err)
		}
	}

	// Make token_hash NOT NULL now that all rows have a value.
	// (We need the column to be NOT NULL before adding the unique index.)
	if columnExists(tx, "invites", "token_hash") {
		if _, err := tx.Exec(`ALTER TABLE invites MODIFY COLUMN token_hash BINARY(32) NOT NULL`); err != nil {
			return fmt.Errorf("set invites.token_hash NOT NULL: %w", err)
		}
	}

	// Add unique index on token_hash.
	if !indexExistsTx(tx, "invites", "idx_invite_token_hash") {
		if _, err := tx.Exec(`ALTER TABLE invites ADD UNIQUE INDEX idx_invite_token_hash (token_hash)`); err != nil {
			return fmt.Errorf("add idx_invite_token_hash: %w", err)
		}
	}

	// Drop old unique index on token (if it still exists).
	if indexExistsTx(tx, "invites", "idx_invite_unique_key") {
		if _, err := tx.Exec(`ALTER TABLE invites DROP INDEX idx_invite_unique_key`); err != nil {
			return fmt.Errorf("drop idx_invite_unique_key: %w", err)
		}
	}

	// Drop old cleartext token column.
	if columnExists(tx, "invites", "token") {
		if _, err := tx.Exec(`ALTER TABLE invites DROP COLUMN token`); err != nil {
			return fmt.Errorf("drop invites.token: %w", err)
		}
	}

	// --- password_reset_requests table ---

	// Add token_hash column if it doesn't exist yet.
	if !columnExists(tx, "password_reset_requests", "token_hash") {
		if _, err := tx.Exec(`ALTER TABLE password_reset_requests ADD COLUMN token_hash BINARY(32) NULL`); err != nil {
			return fmt.Errorf("add password_reset_requests.token_hash: %w", err)
		}
	}

	// Backfill hashes from cleartext tokens.
	if columnExists(tx, "password_reset_requests", "token") {
		if _, err := tx.Exec(`UPDATE password_reset_requests SET token_hash = UNHEX(SHA2(token, 256)) WHERE token_hash IS NULL`); err != nil {
			return fmt.Errorf("backfill password_reset_requests.token_hash: %w", err)
		}
	}

	// Make token_hash NOT NULL.
	if columnExists(tx, "password_reset_requests", "token_hash") {
		if _, err := tx.Exec(`ALTER TABLE password_reset_requests MODIFY COLUMN token_hash BINARY(32) NOT NULL`); err != nil {
			return fmt.Errorf("set password_reset_requests.token_hash NOT NULL: %w", err)
		}
	}

	// Add index on token_hash.
	if !indexExistsTx(tx, "password_reset_requests", "idx_password_reset_token_hash") {
		if _, err := tx.Exec(`ALTER TABLE password_reset_requests ADD INDEX idx_password_reset_token_hash (token_hash)`); err != nil {
			return fmt.Errorf("add idx_password_reset_token_hash: %w", err)
		}
	}

	// Drop old cleartext token column.
	if columnExists(tx, "password_reset_requests", "token") {
		if _, err := tx.Exec(`ALTER TABLE password_reset_requests DROP COLUMN token`); err != nil {
			return fmt.Errorf("drop password_reset_requests.token: %w", err)
		}
	}

	return nil
}

func Down_20261111132618(tx *sql.Tx) error {
	return nil
}
