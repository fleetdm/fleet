package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260227011550, Down_20260227011550)
}

func Up_20260227011550(tx *sql.Tx) error {
	// Drop the unique constraint on organization_name so that multiple tokens from
	// the same Apple Business Manager organization can be added.
	if _, err := tx.Exec(`ALTER TABLE abm_tokens DROP INDEX idx_abm_tokens_organization_name`); err != nil {
		return fmt.Errorf("drop unique index on abm_tokens.organization_name: %w", err)
	}

	// Add dep_name: the unique per-token identifier used as the key in nano_dep_names.
	// The ConsumerKey from the OAuth1 token is unique per ABM server token.
	if _, err := tx.Exec(`ALTER TABLE abm_tokens ADD COLUMN dep_name VARCHAR(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' AFTER organization_name`); err != nil {
		return fmt.Errorf("add dep_name column to abm_tokens: %w", err)
	}

	// Backfill: existing tokens used organization_name as the nano_dep_names key,
	// so set dep_name = organization_name to preserve that mapping.
	if _, err := tx.Exec(`UPDATE abm_tokens SET dep_name = organization_name`); err != nil {
		return fmt.Errorf("backfill dep_name in abm_tokens: %w", err)
	}

	// Add a unique constraint on dep_name to prevent uploading the same ABM token twice.
	if _, err := tx.Exec(`ALTER TABLE abm_tokens ADD UNIQUE KEY idx_abm_tokens_dep_name (dep_name)`); err != nil {
		return fmt.Errorf("add unique index on abm_tokens.dep_name: %w", err)
	}

	return nil
}

func Down_20260227011550(tx *sql.Tx) error {
	return nil
}
