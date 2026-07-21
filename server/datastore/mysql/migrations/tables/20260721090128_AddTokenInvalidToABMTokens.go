package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20260721090128, Down_20260721090128)
}

func Up_20260721090128(tx *sql.Tx) error {
	_, err := tx.Exec(`ALTER TABLE abm_tokens ADD COLUMN token_invalid TINYINT(1) NOT NULL DEFAULT '0'`)
	return err
}

func Down_20260721090128(tx *sql.Tx) error {
	return nil
}
