package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20251124135808, Down_20251124135808)
}

func Up_20251124135808(tx *sql.Tx) error {
	_, err := tx.Exec(`
	CREATE TABLE android_app_configurations (
		id INT UNSIGNED NOT NULL AUTO_INCREMENT,
		application_id VARCHAR(255) NOT NULL,
		team_id INT UNSIGNED NULL,
		global_or_team_id INT NOT NULL DEFAULT 0,
		configuration JSON NOT NULL,
		created_at timestamp(6) DEFAULT CURRENT_TIMESTAMP(6),
		updated_at timestamp(6) DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),

		PRIMARY KEY (id),
		FOREIGN KEY (team_id) REFERENCES teams (id) ON DELETE CASCADE,
		UNIQUE KEY idx_global_or_team_id_application_id (global_or_team_id, application_id)
	) DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci`)
	if err != nil {
		return fmt.Errorf("failed to create table android_app_configurations: %w", err)
	}
	return nil
}

func Down_20251124135808(tx *sql.Tx) error {
	return nil
}
