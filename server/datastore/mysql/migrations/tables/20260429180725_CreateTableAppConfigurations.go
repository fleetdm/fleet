package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260429180725, Down_20260429180725)
}

func Up_20260429180725(tx *sql.Tx) error {
	_, err := tx.Exec(`
	CREATE TABLE vpp_app_configurations (
		id INT UNSIGNED NOT NULL AUTO_INCREMENT,
		application_id VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
		team_id INT UNSIGNED NOT NULL,
		platform VARCHAR(10) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
		configuration MEDIUMTEXT NOT NULL,
		created_at timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
		updated_at timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),

		PRIMARY KEY (id),
		UNIQUE KEY idx_vpp_app_config_team_app_platform (team_id, application_id, platform),
		CONSTRAINT fk_vpp_app_configurations_app
			FOREIGN KEY (application_id, platform)
			REFERENCES vpp_apps (adam_id, platform)
			ON DELETE CASCADE
	) DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci`)
	if err != nil {
		return fmt.Errorf("failed to create table vpp_app_configurations: %w", err)
	}

	_, err = tx.Exec(`
	CREATE TABLE in_house_app_configurations (
		id INT UNSIGNED NOT NULL AUTO_INCREMENT,
		in_house_app_id INT UNSIGNED NOT NULL,
		configuration MEDIUMTEXT NOT NULL,
		created_at timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
		updated_at timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),

		PRIMARY KEY (id),
		UNIQUE KEY idx_in_house_app_config_app (in_house_app_id),
		CONSTRAINT fk_in_house_app_configurations_app
			FOREIGN KEY (in_house_app_id)
			REFERENCES in_house_apps (id)
			ON DELETE CASCADE
	) DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci`)
	if err != nil {
		return fmt.Errorf("failed to create table in_house_app_configurations: %w", err)
	}

	return nil
}

func Down_20260429180725(tx *sql.Tx) error {
	return nil
}
