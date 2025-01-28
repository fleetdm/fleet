package migrations

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20250101000000, Down_20250101000000)
}

func Up_20250101000000(tx *sql.Tx) error {
	_, err := tx.Exec(`CREATE TABLE android_enterprises (
    		id INT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    		signup_name VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
    		enterprise_id VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
			created_at DATETIME(6) NULL DEFAULT NOW(6),
  			updated_at DATETIME(6) NULL DEFAULT NOW(6) ON UPDATE NOW(6))`)
	if err != nil {
		return fmt.Errorf("failed to create android_enterprise table: %w", err)
	}
	logger.Info.Println("Done with initial migration.")
	return nil
}

func Down_20250101000000(_ *sql.Tx) error {
	return nil
}
