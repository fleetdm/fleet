package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260415120000, Down_20260415120000)
}

// Up_20260415120000 creates tables backing the in-app notification center.
//
// The notifications table is the canonical source of truth for active/resolved
// system-generated notifications (license expiring, APNs cert expiring, etc.).
// Producers upsert by dedupe_key so a recurring checker (e.g. cron) does not
// create duplicate rows; when the underlying condition clears, the producer
// sets resolved_at so the notification is hidden without losing history.
//
// user_notification_state tracks per-user interactions (read, dismiss) as a
// side table rather than JSON in users.settings, so we can index on
// dismissed_at / read_at for fast unread-count queries.
//
// notification_deliveries is scaffolding for future email/slack delivery — no
// code writes to it yet but the table exists so later work is a pure additive
// change.
func Up_20260415120000(tx *sql.Tx) error {
	if _, err := tx.Exec(`
		CREATE TABLE notifications (
			id           BIGINT(20) UNSIGNED NOT NULL AUTO_INCREMENT,
			type         VARCHAR(64)  COLLATE utf8mb4_unicode_ci NOT NULL,
			severity     VARCHAR(16)  COLLATE utf8mb4_unicode_ci NOT NULL,
			title        VARCHAR(255) COLLATE utf8mb4_unicode_ci NOT NULL,
			body         TEXT         COLLATE utf8mb4_unicode_ci NOT NULL,
			cta_url      VARCHAR(512) COLLATE utf8mb4_unicode_ci DEFAULT NULL,
			cta_label    VARCHAR(64)  COLLATE utf8mb4_unicode_ci DEFAULT NULL,
			metadata     JSON         DEFAULT NULL,
			dedupe_key   VARCHAR(255) COLLATE utf8mb4_unicode_ci NOT NULL,
			audience     VARCHAR(32)  COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'admin',
			resolved_at  TIMESTAMP(6) NULL DEFAULT NULL,
			created_at   TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
			updated_at   TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
			PRIMARY KEY (id),
			UNIQUE KEY idx_notifications_dedupe_key (dedupe_key),
			KEY idx_notifications_resolved (resolved_at),
			KEY idx_notifications_type (type)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
	`); err != nil {
		return fmt.Errorf("creating notifications table: %w", err)
	}

	if _, err := tx.Exec(`
		CREATE TABLE user_notification_state (
			user_id         INT(10) UNSIGNED NOT NULL,
			notification_id BIGINT(20) UNSIGNED NOT NULL,
			read_at         TIMESTAMP(6) NULL DEFAULT NULL,
			dismissed_at    TIMESTAMP(6) NULL DEFAULT NULL,
			created_at      TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
			updated_at      TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
			PRIMARY KEY (user_id, notification_id),
			KEY idx_uns_notification_id (notification_id),
			KEY idx_uns_dismissed (user_id, dismissed_at),
			CONSTRAINT fk_uns_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
			CONSTRAINT fk_uns_notification FOREIGN KEY (notification_id) REFERENCES notifications (id) ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
	`); err != nil {
		return fmt.Errorf("creating user_notification_state table: %w", err)
	}

	if _, err := tx.Exec(`
		CREATE TABLE notification_deliveries (
			id              BIGINT(20) UNSIGNED NOT NULL AUTO_INCREMENT,
			notification_id BIGINT(20) UNSIGNED NOT NULL,
			channel         VARCHAR(32)  COLLATE utf8mb4_unicode_ci NOT NULL,
			target          VARCHAR(512) COLLATE utf8mb4_unicode_ci NOT NULL,
			status          VARCHAR(32)  COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'pending',
			error           TEXT         COLLATE utf8mb4_unicode_ci DEFAULT NULL,
			attempted_at    TIMESTAMP(6) NULL DEFAULT NULL,
			created_at      TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
			updated_at      TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
			PRIMARY KEY (id),
			KEY idx_nd_notification_id (notification_id),
			KEY idx_nd_status (status),
			CONSTRAINT fk_nd_notification FOREIGN KEY (notification_id) REFERENCES notifications (id) ON DELETE CASCADE
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
	`); err != nil {
		return fmt.Errorf("creating notification_deliveries table: %w", err)
	}

	return nil
}

// Down_20260415120000 is a no-op. Fleet convention: down migrations return nil
// because forward-only migrations are safer than attempting rollback DDL.
func Down_20260415120000(tx *sql.Tx) error {
	return nil
}
