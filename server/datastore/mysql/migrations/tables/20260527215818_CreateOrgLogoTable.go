package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260527215818, Down_20260527215818)
}

func Up_20260527215818(tx *sql.Tx) error {
	if _, err := tx.Exec(`CREATE TABLE org_logo (
		mode        VARCHAR(10)  NOT NULL,
		data        MEDIUMBLOB   NOT NULL,
		uploaded_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
		PRIMARY KEY (mode)
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`); err != nil {
		return fmt.Errorf("create org_logo table: %w", err)
	}
	return nil
}

func Down_20260527215818(tx *sql.Tx) error {
	return nil
}
