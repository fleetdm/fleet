package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20261027104022, Down_20261027104022)
}

func Up_20261027104022(tx *sql.Tx) error {
	_, err := tx.Exec(`
CREATE TABLE IF NOT EXISTS google_conditional_access_integrations (
  id                  INT UNSIGNED NOT NULL AUTO_INCREMENT,
  customer_id         VARCHAR(64) COLLATE utf8mb4_unicode_ci NOT NULL,
  proxy_server_secret VARCHAR(64) COLLATE utf8mb4_unicode_ci NOT NULL,
  oauth_client_id     VARCHAR(255) COLLATE utf8mb4_unicode_ci NOT NULL,
  setup_done          TINYINT(1) NOT NULL DEFAULT 0,
  setup_error         TEXT COLLATE utf8mb4_unicode_ci NULL,
  created_at          DATETIME(6) DEFAULT CURRENT_TIMESTAMP(6),
  updated_at          DATETIME(6) DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),

  PRIMARY KEY (id),
  UNIQUE KEY idx_google_conditional_access_customer_id (customer_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`)
	if err != nil {
		return fmt.Errorf("creating google_conditional_access_integrations table: %w", err)
	}
	return nil
}

func Down_20261027104022(tx *sql.Tx) error {
	return nil
}
