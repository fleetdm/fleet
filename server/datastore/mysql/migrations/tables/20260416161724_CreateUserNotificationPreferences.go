package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260416161724, Down_20260416161724)
}

// Up_20260416161724 creates the user_notification_preferences table backing
// per-user opt-in/out for notification categories across delivery channels.
//
// Rows only exist for explicit opt-out (or later re-opt-in), so the SELECT
// side of the filter is a LEFT JOIN that treats a missing row as "enabled".
// This keeps the default behavior "user sees everything in their audience"
// without needing to seed rows for every (user, category, channel) cross
// product at user creation time.
//
// channel is stored even though only in_app is read today so the UI can hold
// user preferences ahead of the email/slack delivery pipelines landing.
func Up_20260416161724(tx *sql.Tx) error {
	if _, err := tx.Exec(`
		CREATE TABLE user_notification_preferences (
			user_id   INT(10) UNSIGNED NOT NULL,
			category  VARCHAR(32) COLLATE utf8mb4_unicode_ci NOT NULL,
			channel   VARCHAR(16) COLLATE utf8mb4_unicode_ci NOT NULL,
			enabled   TINYINT(1) NOT NULL DEFAULT 1,
			created_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
			updated_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
			PRIMARY KEY (user_id, category, channel),
			CONSTRAINT fk_unp_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
	`); err != nil {
		return fmt.Errorf("creating user_notification_preferences table: %w", err)
	}
	return nil
}

func Down_20260416161724(tx *sql.Tx) error {
	return nil
}
