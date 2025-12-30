package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20251219201524, Down_20251219201524)
}

func Up_20251219201524(tx *sql.Tx) error {
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS software_update_schedules (
			id INT UNSIGNED NOT NULL AUTO_INCREMENT,
			team_id INT UNSIGNED NOT NULL,
			title_id INT UNSIGNED NOT NULL,
			enabled BOOLEAN NOT NULL DEFAULT FALSE,
			start_time char(5) NOT NULL,
			end_time char(5) NOT NULL,

			PRIMARY KEY (id),
			UNIQUE KEY idx_team_title (team_id, title_id),
			FOREIGN KEY (title_id) REFERENCES software_titles (id) ON DELETE CASCADE
		) DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci;
	`)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`
				ALTER TABLE hosts ADD COLUMN timezone VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci
	`)
	if err != nil {
		return err
	}
	return nil
}

func Down_20251219201524(tx *sql.Tx) error {
	return nil
}
