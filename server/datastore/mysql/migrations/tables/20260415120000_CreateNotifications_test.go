package tables

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUp_20260415120000(t *testing.T) {
	db := applyUpToPrev(t)

	// Create a user first so user_notification_state FK passes.
	userResult, err := db.Exec(`
		INSERT INTO users (name, email, password, salt, global_role)
		VALUES (?, ?, ?, ?, ?)`,
		"Admin User", "admin@example.com", []byte("pw"), "salt", "admin",
	)
	require.NoError(t, err)
	userID, err := userResult.LastInsertId()
	require.NoError(t, err)

	applyNext(t, db)

	// Insert a notification.
	res, err := db.Exec(`
		INSERT INTO notifications (type, severity, title, body, cta_url, cta_label, metadata, dedupe_key, audience)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"apns_cert_expiring", "warning",
		"APNs certificate expiring soon",
		"Your Apple Push Notification service certificate expires in 12 days.",
		"/settings/integrations/mdm/apple",
		"Renew certificate",
		`{"days_until":12}`,
		"apns_cert_expiring",
		"admin",
	)
	require.NoError(t, err)
	notificationID, err := res.LastInsertId()
	require.NoError(t, err)

	// Dedupe key is unique.
	_, err = db.Exec(`
		INSERT INTO notifications (type, severity, title, body, dedupe_key, audience)
		VALUES (?, ?, ?, ?, ?, ?)`,
		"apns_cert_expiring", "warning", "dup", "dup", "apns_cert_expiring", "admin",
	)
	require.Error(t, err, "dedupe_key must be unique")

	// Insert per-user state.
	_, err = db.Exec(`
		INSERT INTO user_notification_state (user_id, notification_id, read_at)
		VALUES (?, ?, NOW(6))`,
		userID, notificationID,
	)
	require.NoError(t, err)

	// Dismiss it.
	_, err = db.Exec(`
		UPDATE user_notification_state
		SET dismissed_at = NOW(6)
		WHERE user_id = ? AND notification_id = ?`,
		userID, notificationID,
	)
	require.NoError(t, err)

	var dismissedCount int
	err = db.QueryRow(`
		SELECT COUNT(*) FROM user_notification_state
		WHERE user_id = ? AND dismissed_at IS NOT NULL`,
		userID,
	).Scan(&dismissedCount)
	require.NoError(t, err)
	assert.Equal(t, 1, dismissedCount)

	// Deliveries insert (future email/slack).
	_, err = db.Exec(`
		INSERT INTO notification_deliveries (notification_id, channel, target, status)
		VALUES (?, ?, ?, ?)`,
		notificationID, "email", "admin@example.com", "pending",
	)
	require.NoError(t, err)

	// FK cascade: deleting notification removes delivery + state rows.
	_, err = db.Exec(`DELETE FROM notifications WHERE id = ?`, notificationID)
	require.NoError(t, err)

	var remaining int
	err = db.QueryRow(`SELECT COUNT(*) FROM notification_deliveries WHERE notification_id = ?`, notificationID).Scan(&remaining)
	require.NoError(t, err)
	assert.Equal(t, 0, remaining)

	err = db.QueryRow(`SELECT COUNT(*) FROM user_notification_state WHERE notification_id = ?`, notificationID).Scan(&remaining)
	require.NoError(t, err)
	assert.Equal(t, 0, remaining)
}
