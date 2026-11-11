package tables

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUp_20261110211109(t *testing.T) {
	db := applyUpToPrev(t)

	const details = `{"host_id": 7, "host_display_name": "Anna's MacBook Pro"}`
	insert := func(activityType string) int64 {
		return execNoErrLastID(t, db,
			`INSERT INTO activity_past (activity_type, details, streamed, user_email) VALUES (?, ?, true, 'a@example.com')`,
			activityType, details)
	}
	want := map[int64]string{ // row id to expected type
		insert("viewed_host_recovery_lock_password"): "read_host_recovery_lock_password",
		insert("viewed_host_recovery_lock_password"): "read_host_recovery_lock_password",
		insert("read_host_recovery_lock_password"):   "read_host_recovery_lock_password",
	}
	for _, unchanged := range []string{
		"set_host_recovery_lock_password",
		"rotated_host_recovery_lock_password",
		"enabled_recovery_lock_passwords",
		"disabled_recovery_lock_passwords",
		"read_host_disk_encryption_key",
	} {
		want[insert(unchanged)] = unchanged
	}

	applyNext(t, db)

	var rows []struct {
		ID           int64  `db:"id"`
		ActivityType string `db:"activity_type"`
		Details      string `db:"details"`
		Streamed     bool   `db:"streamed"`
	}
	require.NoError(t, db.Select(&rows, `SELECT id, activity_type, details, streamed FROM activity_past`))
	require.Len(t, rows, len(want))
	for _, row := range rows {
		assert.Equal(t, want[row.ID], row.ActivityType)
		assert.JSONEq(t, details, row.Details, "details are not touched")
		assert.True(t, row.Streamed, "streamed is not touched")
	}
}
