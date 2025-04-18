package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20250418162727, Down_20250418162727)
}

func Up_20250418162727(tx *sql.Tx) error {
	_, err := tx.Exec(`CREATE TABLE IF NOT EXISTS software_categories  (
    		id INT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    		name VARCHAR(63) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
			created_at DATETIME(6) NULL DEFAULT NOW(6),
  			updated_at DATETIME(6) NULL DEFAULT NOW(6) ON UPDATE NOW(6))`)
	if err != nil {
		return fmt.Errorf("failed to create software_categories table: %w", err)
	}

	_, err = tx.Exec(`CREATE TABLE IF NOT EXISTS software_categories_software_installers  (
    		id INT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    		category_id INT UNSIGNED NOT NULL,
			software_installer_id INT UNSIGNED NOT NULL,
			created_at DATETIME(6) NULL DEFAULT NOW(6),
  			updated_at DATETIME(6) NULL DEFAULT NOW(6) ON UPDATE NOW(6))`)
	if err != nil {
		return fmt.Errorf("failed to create software_categories_software_installers table: %w", err)
	}

	_, err = tx.Exec(`CREATE TABLE IF NOT EXISTS software_categories_vpp_apps_teams  (
    		id INT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    		category_id INT UNSIGNED NOT NULL,
			vpp_apps_teams_id INT UNSIGNED NOT NULL,
			created_at DATETIME(6) NULL DEFAULT NOW(6),
  			updated_at DATETIME(6) NULL DEFAULT NOW(6) ON UPDATE NOW(6))`)
	if err != nil {
		return fmt.Errorf("failed to create software_categories_vpp_apps_teams table: %w", err)
	}

	return nil
}

func Down_20250418162727(tx *sql.Tx) error {
	return nil
}
