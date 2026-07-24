package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260721090128, Down_20260721090128)
}

func Up_20260721090128(tx *sql.Tx) error {
	if _, err := tx.Exec(`ALTER TABLE abm_tokens ADD COLUMN token_invalid TINYINT(1) NOT NULL DEFAULT '0'`); err != nil {
		return fmt.Errorf("adding token_invalid column to abm_tokens table: %w", err)
	}
	return nil
}

func Down_20260721090128(tx *sql.Tx) error {
	return nil
}
