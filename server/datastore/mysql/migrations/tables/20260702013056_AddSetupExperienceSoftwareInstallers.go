package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260702013056, Down_20260702013056)
}

func Up_20260702013056(tx *sql.Tx) error {
	_, err := tx.Exec(`
CREATE TABLE setup_experience_software_installers (
  software_installer_id INT UNSIGNED NOT NULL,
  platform              VARCHAR(32)  NOT NULL,
  global_or_team_id     INT UNSIGNED NOT NULL,
  created_at            TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (software_installer_id, platform),
  KEY idx_seti_team_platform (global_or_team_id, platform),
  CONSTRAINT fk_seti_installer FOREIGN KEY (software_installer_id)
    REFERENCES software_installers(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
`)
	if err != nil {
		return fmt.Errorf("creating setup_experience_software_installers table: %w", err)
	}
	return nil
}

func Down_20260702013056(tx *sql.Tx) error {
	return nil
}
