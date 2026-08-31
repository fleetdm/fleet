package tables

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUp_20260831184623(t *testing.T) {
	db := applyUpToPrev(t)

	hostID := execNoErrLastID(t, db,
		`INSERT INTO hosts (osquery_host_id, node_key, hostname, uuid, platform) VALUES (?, ?, ?, ?, 'darwin')`,
		"osquery-1", "node-key-1", "host1", "uuid-1",
	)
	policyID := execNoErrLastID(t, db,
		`INSERT INTO policies (name, query, description, checksum) VALUES (?, ?, ?, ?)`,
		"policy1", "", "", "checksum1",
	)
	execNoErr(t, db,
		`INSERT INTO notifications_end_user (uuid, host_id, status, kind, payload, expires_at)
			VALUES (?, ?, 'pending', 'patch', '{}', NOW(6) + INTERVAL 1 HOUR)`,
		"notification-1", hostID,
	)

	applyNext(t, db)

	execNoErr(t, db, `INSERT INTO patch_notifications (notification_uuid) VALUES (?)`, "notification-1")
	execNoErr(t, db,
		`INSERT INTO patch_notification_apps (notification_uuid, policy_id, software_title_id, software_installer_id)
			VALUES (?, ?, ?, ?)`,
		"notification-1", policyID, 42, 7,
	)

	// install_at and reminder_sent_at start empty; the countdown sub-issue writes them.
	var installAt, reminderSentAt *string
	require.NoError(t, db.QueryRow(
		`SELECT install_at, reminder_sent_at FROM patch_notifications WHERE notification_uuid = ?`,
		"notification-1").Scan(&installAt, &reminderSentAt))
	assert.Nil(t, installAt)
	assert.Nil(t, reminderSentAt)

	// (notification_uuid, software_title_id) is the dedup key, so a repeat of the same app is rejected.
	_, err := db.Exec(
		`INSERT INTO patch_notification_apps (notification_uuid, software_title_id) VALUES (?, ?)`,
		"notification-1", 42)
	require.Error(t, err)

	// Deleting the policy leaves the app row behind so the notification still lists the app.
	execNoErr(t, db, `DELETE FROM policies WHERE id = ?`, policyID)
	var policyIDAfter *uint
	require.NoError(t, db.QueryRow(
		`SELECT policy_id FROM patch_notification_apps WHERE notification_uuid = ? AND software_title_id = ?`,
		"notification-1", 42).Scan(&policyIDAfter))
	assert.Nil(t, policyIDAfter)

	// Both tables hang off the notification.
	execNoErr(t, db, `DELETE FROM notifications_end_user WHERE uuid = ?`, "notification-1")

	var count int
	require.NoError(t, db.GetContext(context.Background(), &count,
		`SELECT COUNT(*) FROM patch_notifications WHERE notification_uuid = ?`, "notification-1"))
	assert.Zero(t, count)
	require.NoError(t, db.GetContext(context.Background(), &count,
		`SELECT COUNT(*) FROM patch_notification_apps WHERE notification_uuid = ?`, "notification-1"))
	assert.Zero(t, count)
}
