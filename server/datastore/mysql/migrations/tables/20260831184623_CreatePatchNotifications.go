package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260831184623, Down_20260831184623)
}

func Up_20260831184623(tx *sql.Tx) error {
	_, err := tx.Exec(`
CREATE TABLE IF NOT EXISTS patch_notifications (
  notification_uuid VARCHAR(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  install_at        DATETIME(6) NULL DEFAULT NULL,
  reminder_sent_at  DATETIME(6) NULL DEFAULT NULL,
  created_at        DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  updated_at        DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
  PRIMARY KEY (notification_uuid),
  -- for the reminder and install passes, which run every minute
  KEY idx_patch_notifications_install_at (install_at),
  CONSTRAINT fk_patch_notifications_notification_uuid FOREIGN KEY (notification_uuid)
    REFERENCES notifications_end_user (uuid) ON DELETE CASCADE
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci
`)
	if err != nil {
		return fmt.Errorf("creating patch_notifications table: %w", err)
	}

	// The apps live in columns rather than in the notification's JSON payload
	// because the dedup guard and the Automation runs query both filter by app
	// and by policy, and the dedup guard runs on every policy pass.
	//
	// software_title_id and software_installer_id carry no foreign key: an
	// installer can be deleted while a notification is still on screen, and the
	// activity that names its apps outlives it.
	_, err = tx.Exec(`
CREATE TABLE IF NOT EXISTS patch_notification_apps (
  notification_uuid     VARCHAR(36) COLLATE utf8mb4_unicode_ci NOT NULL,
  policy_id             INT UNSIGNED NULL DEFAULT NULL,
  software_title_id     INT UNSIGNED NOT NULL,
  software_installer_id INT UNSIGNED NULL DEFAULT NULL,
  created_at            DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  PRIMARY KEY (notification_uuid, software_title_id),
  -- for the Automation runs table, which reads by policy
  KEY idx_patch_notification_apps_policy (policy_id),
  CONSTRAINT fk_patch_notification_apps_notification_uuid FOREIGN KEY (notification_uuid)
    REFERENCES notifications_end_user (uuid) ON DELETE CASCADE,
  CONSTRAINT fk_patch_notification_apps_policy_id FOREIGN KEY (policy_id)
    REFERENCES policies (id) ON DELETE SET NULL
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci
`)
	if err != nil {
		return fmt.Errorf("creating patch_notification_apps table: %w", err)
	}

	return nil
}

func Down_20260831184623(tx *sql.Tx) error {
	return nil
}
