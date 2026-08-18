package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260817214627, Down_20260817214627)
}

func Up_20260817214627(tx *sql.Tx) error {
	_, err := tx.Exec(`
CREATE TABLE IF NOT EXISTS notifications_end_user (
  id              INT UNSIGNED NOT NULL AUTO_INCREMENT,
  uuid            VARCHAR(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  host_id         INT UNSIGNED NOT NULL,
  status          VARCHAR(31) COLLATE utf8mb4_unicode_ci NOT NULL,
  kind            VARCHAR(63) COLLATE utf8mb4_unicode_ci NOT NULL,
  payload         JSON NOT NULL,
  attempt_count   INT UNSIGNED NOT NULL DEFAULT 0,
  next_attempt_at DATETIME(6) NULL DEFAULT NULL,
  displayed_at    DATETIME(6) NULL DEFAULT NULL,
  execution_id    VARCHAR(255) COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  last_exit_code  INT NULL DEFAULT NULL,
  last_reason     VARCHAR(63) COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
  expires_at      DATETIME(6) NOT NULL,
  created_at      DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  updated_at      DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  PRIMARY KEY (id),
  UNIQUE KEY idx_notifications_end_user_uuid (uuid),
  KEY idx_notifications_end_user_dispatch (status, next_attempt_at),
  KEY idx_notifications_end_user_host (host_id, status),
  KEY idx_notifications_end_user_execution_id (execution_id),
  -- for the two halves of the expiry sweep, which runs every minute
  KEY idx_notifications_end_user_expires_at (expires_at),
  KEY idx_notifications_end_user_stuck (status, updated_at),
  CONSTRAINT fk_notifications_end_user_host_id FOREIGN KEY (host_id) REFERENCES hosts (id) ON DELETE CASCADE
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci
`)
	if err != nil {
		return fmt.Errorf("creating notifications_end_user table: %w", err)
	}

	return nil
}

func Down_20260817214627(tx *sql.Tx) error {
	return nil
}
