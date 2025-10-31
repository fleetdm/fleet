package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20251030174959, Down_20251030174959)
}

func Up_20251030174959(tx *sql.Tx) error {
	_, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS software_title_display_names (
			id INT AUTO_INCREMENT PRIMARY KEY,
			team_id INT UNSIGNED NOT NULL,
			software_title_id INT UNSIGNED NOT NULL,
			display_name varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
			created_at timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
			UNIQUE KEY idx_unique_team_id_title_id (team_id, software_title_id),
			FOREIGN KEY (software_title_id) REFERENCES software_titles(id) ON DELETE CASCADE ON UPDATE CASCADE
		)
`)
	if err != nil {
		return errors.Wrapf(err, "create software_title_display_names table")
	}

	return nil
}

func Down_20251030174959(tx *sql.Tx) error {
	return nil
}
