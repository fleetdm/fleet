package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20250905090000, Down_20250905090000)
}

func Up_20250905090000(tx *sql.Tx) error {
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS software_title_icons (
			id INT AUTO_INCREMENT PRIMARY KEY,
			team_id INT UNSIGNED NOT NULL,
			software_title_id INT UNSIGNED NOT NULL,
			storage_id VARCHAR(64) COLLATE utf8mb4_unicode_ci NOT NULL,
			filename varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,
			created_at timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
			UNIQUE KEY idx_unique_team_id_title_id_storage_id (team_id, software_title_id),
			INDEX idx_storage_id_team_id (storage_id, team_id),
			FOREIGN KEY (software_title_id) REFERENCES software_titles(id) ON DELETE CASCADE ON UPDATE CASCADE
		)
	`)
	return err
}

func Down_20250905090000(tx *sql.Tx) error {
	_, err := tx.Exec("DROP TABLE IF EXISTS `software_title_icons`;")
	return err
}
