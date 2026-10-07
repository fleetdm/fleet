package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20261005130632, Down_20261005130632)
}

func Up_20261005130632(tx *sql.Tx) error {
	if columnExists(tx, "acme_challenges", "attested_public_key") {
		return nil
	}
	if _, err := tx.Exec(`
		ALTER TABLE acme_challenges
		ADD COLUMN attested_public_key VARBINARY(255) NULL DEFAULT NULL,
		ALGORITHM=INSTANT
	`); err != nil {
		return fmt.Errorf("adding attested_public_key to acme_challenges table: %w", err)
	}
	return nil
}

func Down_20261005130632(tx *sql.Tx) error {
	return nil
}
