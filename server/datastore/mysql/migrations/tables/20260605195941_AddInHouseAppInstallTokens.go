package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20260605195941, Down_20260605195941)
}

func Up_20260605195941(tx *sql.Tx) error {
	_, err := tx.Exec(`
CREATE TABLE in_house_app_install_tokens (
  token             VARCHAR(36)  COLLATE utf8mb4_unicode_ci NOT NULL,
  software_title_id INT UNSIGNED NOT NULL,
  team_id           INT UNSIGNED NOT NULL,
  host_id           INT UNSIGNED NOT NULL,
  expires_at        DATETIME(6)  NOT NULL,
  created_at        DATETIME(6)  NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  PRIMARY KEY (token),
  KEY idx_in_house_app_install_tokens_expires_at (expires_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
`)
	if err != nil {
		return errors.Wrap(err, "create in_house_app_install_tokens")
	}
	return nil
}

func Down_20260605195941(tx *sql.Tx) error {
	return nil
}
