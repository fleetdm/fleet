package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260501133045, Down_20260501133045)
}

func Up_20260501133045(tx *sql.Tx) error {
	if _, err := tx.Exec(`
CREATE TABLE IF NOT EXISTS windows_mdm_default_team (
  id         INT(10) UNSIGNED NOT NULL DEFAULT 1,
  team_id    INT(10) UNSIGNED DEFAULT NULL,
  created_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  updated_at TIMESTAMP(6) NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  PRIMARY KEY (id),
  CONSTRAINT fk_windows_mdm_default_team_team_id
    FOREIGN KEY (team_id) REFERENCES teams (id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`); err != nil {
		return fmt.Errorf("creating windows_mdm_default_team table: %w", err)
	}

	if _, err := tx.Exec(`INSERT INTO windows_mdm_default_team (id, team_id, created_at, updated_at) VALUES (1, NULL, '2020-01-01 00:00:00', '2020-01-01 00:00:00')`); err != nil {
		return fmt.Errorf("inserting default row into windows_mdm_default_team: %w", err)
	}

	return nil
}

func Down_20260501133045(_ *sql.Tx) error {
	return nil
}
