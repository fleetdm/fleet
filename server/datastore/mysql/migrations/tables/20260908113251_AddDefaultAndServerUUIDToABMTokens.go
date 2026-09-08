package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260908113251, Down_20260908113251)
}

func Up_20260908113251(tx *sql.Tx) error {
	if _, err := tx.Exec(`ALTER TABLE abm_tokens
		ADD COLUMN server_uuid VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
		ADD COLUMN is_default TINYINT(1) NOT NULL DEFAULT 0;`); err != nil {
		return fmt.Errorf("adding server_uuid and is_default to abm_tokens: %w", err)
	}

	var count int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM abm_tokens`).Scan(&count); err != nil {
		return fmt.Errorf("counting abm_tokens: %w", err)
	}
	if count == 1 {
		if _, err := tx.Exec(`UPDATE abm_tokens SET is_default = 1`); err != nil {
			return fmt.Errorf("marking sole abm_token default: %w", err)
		}
	}

	return nil
}

func Down_20260908113251(tx *sql.Tx) error {
	return nil
}
