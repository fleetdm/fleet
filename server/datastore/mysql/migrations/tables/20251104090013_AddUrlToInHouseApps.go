package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20251104090013, Down_20251104090013)
}

func Up_20251104090013(tx *sql.Tx) error {
	_, err := tx.Exec(`
ALTER TABLE in_house_apps
	ADD COLUMN url varchar(4095) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT ''
`)
	if err != nil {
		return fmt.Errorf("failed to alter in_house_apps url: %w", err)
	}
	return nil
}

func Down_20251104090013(tx *sql.Tx) error {
	return nil
}
