package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20250828122736, Down_20250828122736)
}

func Up_20250828122736(tx *sql.Tx) error {
	_, err := tx.Exec(`
		CREATE TABLE software_title_icons (
			id INT AUTO_INCREMENT PRIMARY KEY,
			team_id INT UNSIGNED NOT NULL,
			software_title_id INT UNSIGNED NOT NULL,
			storage_id VARCHAR(64) COLLATE utf8mb4_unicode_ci NOT NULL,
			filename VARCHAR(255) NOT NULL,
			created_at timestamp(6) DEFAULT CURRENT_TIMESTAMP(6),
			UNIQUE KEY idx_unique_team_id_title_id_storage_id (team_id, software_title_id, storage_id),
			FOREIGN KEY (software_title_id) REFERENCES software_titles(id),
			FOREIGN KEY (team_id) REFERENCES teams(id)
		)
	`)
	return err
}

func Down_20250828122736(tx *sql.Tx) error {
	_, err := tx.Exec("DROP TABLE IF EXISTS `software_title_icons`;")
	return err
}
