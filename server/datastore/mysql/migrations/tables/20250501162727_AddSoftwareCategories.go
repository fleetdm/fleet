package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20250501162727, Down_20250501162727)
}

func Up_20250501162727(tx *sql.Tx) error {
	_, err := tx.Exec(`CREATE TABLE IF NOT EXISTS software_categories  (
    		id INT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    		name VARCHAR(63) NOT NULL,
			UNIQUE KEY idx_software_categories_name (name)
			) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`)
	if err != nil {
		return fmt.Errorf("failed to create software_categories table: %w", err)
	}

	_, err = tx.Exec(`CREATE TABLE IF NOT EXISTS software_installer_software_categories  (
    		id INT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    		software_category_id INT UNSIGNED NOT NULL,
			software_installer_id INT UNSIGNED NOT NULL,
			created_at DATETIME(6) NULL DEFAULT NOW(6),
			FOREIGN KEY (software_installer_id) REFERENCES software_installers (id) ON DELETE CASCADE,
			FOREIGN KEY (software_category_id) REFERENCES software_categories (id) ON DELETE CASCADE,
			UNIQUE INDEX idx_unique_software_installer_id_software_category_id (software_installer_id,software_category_id)
			)`)
	if err != nil {
		return fmt.Errorf("failed to create software_installer_software_categories table: %w", err)
	}

	_, err = tx.Exec(`CREATE TABLE IF NOT EXISTS vpp_app_team_software_categories  (
    		id INT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    		software_category_id INT UNSIGNED NOT NULL,
			vpp_app_team_id INT UNSIGNED NOT NULL,
			created_at DATETIME(6) NULL DEFAULT NOW(6),
			FOREIGN KEY (vpp_app_team_id) REFERENCES vpp_apps_teams (id) ON DELETE CASCADE,
			FOREIGN KEY (software_category_id) REFERENCES software_categories (id) ON DELETE CASCADE,
			UNIQUE INDEX idx_unique_vpp_app_team_id_software_category_id (vpp_app_team_id,software_category_id)
			)`)
	if err != nil {
		return fmt.Errorf("failed to create vpp_app_team_software_categories table: %w", err)
	}

	// insert categories
	_, err = tx.Exec(`INSERT INTO software_categories (name) VALUES ('Productivity'), ('Browsers'), ('Communication'), ('Developer tools')`)
	if err != nil {
		return fmt.Errorf("inserting default categories into software_categories table: %w", err)
	}

	return nil
}

func Down_20250501162727(tx *sql.Tx) error {
	return nil
}
