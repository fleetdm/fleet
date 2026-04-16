package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260416163704, Down_20260416163704)
}

// Up_20260416163704 adds a unique index on
// (notification_id, channel, target) to notification_deliveries.
//
// Producers upsert notifications on a cron and then fan out one delivery row
// per destination (channel+target). Without this index, running the same cron
// twice would double-send to Slack. The index lets the fanout use INSERT
// IGNORE and be naturally idempotent per destination.
//
// For future per-user channels (email), the target will be the user's email
// so the same index protects against duplicates too.
func Up_20260416163704(tx *sql.Tx) error {
	// target is VARCHAR(512) utf8mb4 — 512*4 = 2048 bytes which exceeds
	// InnoDB's default 767-byte index key limit without innodb_large_prefix.
	// We index a prefix of target that still uniquely identifies the
	// destination for realistic webhook URLs and email addresses. 191 chars
	// is the well-known safe prefix length for utf8mb4 column indexes.
	if _, err := tx.Exec(`
		ALTER TABLE notification_deliveries
		ADD UNIQUE KEY uq_nd_notification_channel_target (notification_id, channel, target(191))
	`); err != nil {
		return fmt.Errorf("adding unique index on notification_deliveries: %w", err)
	}
	return nil
}

func Down_20260416163704(tx *sql.Tx) error {
	return nil
}
