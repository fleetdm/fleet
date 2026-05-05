package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20250408133233, Down_20250408133233)
}

func Up_20250408133233(tx *sql.Tx) error {
	_, err := tx.Exec(`
	CREATE TABLE IF NOT EXISTS scim_last_request (
	    -- dummy column as a hint that this table should only have one row
	    id TINYINT(1) UNSIGNED NOT NULL PRIMARY KEY DEFAULT 1,
	    status VARCHAR(31) NOT NULL,
	    details VARCHAR(255) NOT NULL,
	    created_at DATETIME(6) NOT NULL DEFAULT NOW(6),
	    updated_at DATETIME(6) NOT NULL DEFAULT NOW(6) ON UPDATE NOW(6)
	) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci;
	`)

	if err != nil {
		return fmt.Errorf("failed to create scim last request table: %s", err)
	}

	return nil
}

func Down_20250408133233(tx *sql.Tx) error {
	return nil
}
