package tables

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUp_20260416161724(t *testing.T) {
	db := applyUpToPrev(t)

	userResult, err := db.Exec(`
		INSERT INTO users (name, email, password, salt, global_role)
		VALUES (?, ?, ?, ?, ?)`,
		"Admin User", "admin@example.com", []byte("pw"), "salt", "admin",
	)
	require.NoError(t, err)
	userID, err := userResult.LastInsertId()
	require.NoError(t, err)

	applyNext(t, db)

	// Insert a preference row.
	_, err = db.Exec(`
		INSERT INTO user_notification_preferences (user_id, category, channel, enabled)
		VALUES (?, ?, ?, ?)`,
		userID, "vulnerabilities", "in_app", false,
	)
	require.NoError(t, err)

	// PK (user_id, category, channel) rejects exact duplicates.
	_, err = db.Exec(`
		INSERT INTO user_notification_preferences (user_id, category, channel, enabled)
		VALUES (?, ?, ?, ?)`,
		userID, "vulnerabilities", "in_app", true,
	)
	require.Error(t, err, "(user_id, category, channel) must be unique")

	// Different channel is fine.
	_, err = db.Exec(`
		INSERT INTO user_notification_preferences (user_id, category, channel, enabled)
		VALUES (?, ?, ?, ?)`,
		userID, "vulnerabilities", "email", true,
	)
	require.NoError(t, err)

	var rowCount int
	err = db.QueryRow(`SELECT COUNT(*) FROM user_notification_preferences WHERE user_id = ?`, userID).Scan(&rowCount)
	require.NoError(t, err)
	assert.Equal(t, 2, rowCount)

	// Deleting the user cascades.
	_, err = db.Exec(`DELETE FROM users WHERE id = ?`, userID)
	require.NoError(t, err)

	err = db.QueryRow(`SELECT COUNT(*) FROM user_notification_preferences WHERE user_id = ?`, userID).Scan(&rowCount)
	require.NoError(t, err)
	assert.Equal(t, 0, rowCount)
}
