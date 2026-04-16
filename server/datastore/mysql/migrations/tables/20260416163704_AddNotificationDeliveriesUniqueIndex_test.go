package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260416163704(t *testing.T) {
	db := applyUpToPrev(t)

	// A notification to hang deliveries off of.
	_, err := db.Exec(`
		INSERT INTO notifications (type, severity, title, body, dedupe_key, audience)
		VALUES (?, ?, ?, ?, ?, ?)`,
		"license_expiring", "warning", "lic", "body", "license_expiring", "admin",
	)
	require.NoError(t, err)
	var notifID int64
	require.NoError(t, db.QueryRow(`SELECT id FROM notifications WHERE dedupe_key = ?`, "license_expiring").Scan(&notifID))

	applyNext(t, db)

	// Two inserts with the same (notification_id, channel, target) — second
	// should fail the unique key, proving idempotent fanout works.
	_, err = db.Exec(`
		INSERT INTO notification_deliveries (notification_id, channel, target, status)
		VALUES (?, ?, ?, ?)`,
		notifID, "slack", "https://hooks.slack.com/services/AAA", "pending",
	)
	require.NoError(t, err)

	_, err = db.Exec(`
		INSERT INTO notification_deliveries (notification_id, channel, target, status)
		VALUES (?, ?, ?, ?)`,
		notifID, "slack", "https://hooks.slack.com/services/AAA", "pending",
	)
	require.Error(t, err, "duplicate (notification_id, channel, target) must be rejected")

	// Different target is allowed.
	_, err = db.Exec(`
		INSERT INTO notification_deliveries (notification_id, channel, target, status)
		VALUES (?, ?, ?, ?)`,
		notifID, "slack", "https://hooks.slack.com/services/BBB", "pending",
	)
	require.NoError(t, err)
}
