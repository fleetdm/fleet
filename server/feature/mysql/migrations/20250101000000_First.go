package feature_migrations

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20250101000000, Down_20250101000000)
}

func Up_20250101000000(tx *sql.Tx) error {
	_, err := tx.Exec(`CREATE TABLE feature_sample (
    		id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    		name VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '')`)
	if err != nil {
		return fmt.Errorf("failed to create feature_sample table: %w", err)
	}
	logger.Info.Println("Done with sample migration.")
	return nil
}

func Down_20250101000000(_ *sql.Tx) error {
	return nil
}
