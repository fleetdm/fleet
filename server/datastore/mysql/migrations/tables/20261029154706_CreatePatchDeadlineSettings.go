package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20261029154706, Down_20261029154706)
}

func Up_20261029154706(tx *sql.Tx) error {
	_, err := tx.Exec(`
CREATE TABLE IF NOT EXISTS patch_deadline_settings (
  team_id    INT UNSIGNED NOT NULL,
  days       SET('monday','tuesday','wednesday','thursday','friday','saturday','sunday') NOT NULL,
  ` + "`time`" + `     CHAR(5) NOT NULL,
  created_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  updated_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  PRIMARY KEY (team_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
`)
	if err != nil {
		return fmt.Errorf("creating patch_deadline_settings table: %w", err)
	}
	return nil
}

func Down_20261029154706(tx *sql.Tx) error {
	return nil
}
