package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260813171950, Down_20260813171950)
}

func Up_20260813171950(tx *sql.Tx) error {
	_, err := tx.Exec(`
CREATE TABLE IF NOT EXISTS end_user_notifications (
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
  expires_at      DATETIME(6) NULL DEFAULT NULL,
  created_at      DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  updated_at      DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  PRIMARY KEY (id),
  UNIQUE KEY idx_end_user_notifications_uuid (uuid),
  KEY idx_end_user_notifications_dispatch (status, next_attempt_at),
  KEY idx_end_user_notifications_host (host_id, status),
  KEY idx_end_user_notifications_execution_id (execution_id),
  CONSTRAINT fk_end_user_notifications_host_id FOREIGN KEY (host_id) REFERENCES hosts (id) ON DELETE CASCADE
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci
`)
	if err != nil {
		return fmt.Errorf("creating end_user_notifications table: %w", err)
	}

	return nil
}

func Down_20260813171950(tx *sql.Tx) error {
	return nil
}
