package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260514220719, Down_20260514220719)
}

// Up_20260514220719 adds the encoding_type column that discriminates between
// the legacy dense bitmap format (encoding_type = 0) and the new roaring
// bitmap format (encoding_type = 1). ALGORITHM=INSTANT is a metadata-only
// change on MySQL 8.0+; existing rows are not rewritten and read back with
// encoding_type = 0 via the column DEFAULT, correctly identifying them as
// dense. New writes always set encoding_type = 1.
func Up_20260514220719(tx *sql.Tx) error {
	if columnExists(tx, "host_scd_data", "encoding_type") {
		return nil
	}
	if _, err := tx.Exec(`
		ALTER TABLE host_scd_data
		ADD COLUMN encoding_type TINYINT NOT NULL DEFAULT 0,
		ALGORITHM=INSTANT
	`); err != nil {
		return fmt.Errorf("add encoding_type to host_scd_data: %w", err)
	}
	return nil
}

func Down_20260514220719(tx *sql.Tx) error {
	return nil
}
