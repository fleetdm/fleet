package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260702013057, Down_20260702013057)
}

func Up_20260702013057(tx *sql.Tx) error {
	_, err := tx.Exec(`
CREATE TABLE software_title_team_pins (
  team_id INT UNSIGNED NOT NULL,
  title_id          INT UNSIGNED NOT NULL,
  pinned_version    VARCHAR(255) NOT NULL,
  updated_at        TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  PRIMARY KEY (team_id, title_id),
  CONSTRAINT fk_pin_title FOREIGN KEY (title_id)
    REFERENCES software_titles(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
`)
	if err != nil {
		return fmt.Errorf("creating software_title_team_pins table: %w", err)
	}
	return nil
}

func Down_20260702013057(tx *sql.Tx) error {
	return nil
}
