package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20260908192830, Down_20260908192830)
}

func Up_20260908192830(tx *sql.Tx) error {
	_, err := tx.Exec(`
CREATE TABLE IF NOT EXISTS android_zero_touch_tokens (
  id                      INT UNSIGNED NOT NULL AUTO_INCREMENT,
  team_id                 INT UNSIGNED DEFAULT NULL,
  global_or_team_id       INT UNSIGNED NOT NULL DEFAULT 0,
  token_name              VARCHAR(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  token_value             VARCHAR(1024) COLLATE utf8mb4_unicode_ci NOT NULL,
  embedded_enroll_secret  VARCHAR(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  expires_at              DATETIME(6) NOT NULL,
  created_at              DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  updated_at              DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  PRIMARY KEY (id),
  UNIQUE KEY idx_zt_global_or_team_id (global_or_team_id),
  KEY fk_zt_team_id (team_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`)
	return err
}

func Down_20260908192830(tx *sql.Tx) error {
	return nil
}
