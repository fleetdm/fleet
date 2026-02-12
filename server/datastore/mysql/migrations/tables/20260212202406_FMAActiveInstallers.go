package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260212202406, Down_20260212202406)
}

func Up_20260212202406(tx *sql.Tx) error {
	_, err := tx.Exec(`
		ALTER TABLE software_installers
			DROP INDEX idx_software_installers_team_id_title_id,
			DROP INDEX idx_software_installers_platform_title_id,
			ADD INDEX idx_software_installers_platform_title_id (global_or_team_id,platform,title_id,version) USING BTREE
	`)
	if err != nil {
		return fmt.Errorf("altering software_installers indexes: %w", err)
	}

	_, err = tx.Exec(`
		CREATE TABLE IF NOT EXISTS fma_active_installers (
			id INT UNSIGNED NOT NULL AUTO_INCREMENT,
			team_id INT UNSIGNED DEFAULT NULL,
			global_or_team_id INT UNSIGNED NOT NULL DEFAULT 0,
			fleet_maintained_app_id INT UNSIGNED NOT NULL,
			software_installer_id INT UNSIGNED NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			PRIMARY KEY (id),
			UNIQUE KEY idx_fma_active_team_app (global_or_team_id, fleet_maintained_app_id),
			CONSTRAINT fk_fma_active_team FOREIGN KEY (team_id) REFERENCES teams (id) ON DELETE CASCADE ON UPDATE CASCADE,
			CONSTRAINT fk_fma_active_app FOREIGN KEY (fleet_maintained_app_id) REFERENCES fleet_maintained_apps (id) ON DELETE CASCADE,
			CONSTRAINT fk_fma_active_installer FOREIGN KEY (software_installer_id) REFERENCES software_installers (id) ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
	`)
	if err != nil {
		return fmt.Errorf("creating fma_active_installers table: %w", err)
	}
	return nil
}

func Down_20260212202406(tx *sql.Tx) error {
	return nil
}
