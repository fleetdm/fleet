package tables

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUp_20260904125235(t *testing.T) {
	db := applyUpToPrev(t)

	insertDevice := func(id string) {
		_, err := db.Exec(`INSERT INTO nano_devices (id, authenticate) VALUES (?, '<plist/>')`, id)
		require.NoError(t, err)
	}
	insertEnrollment := func(id, deviceID, enrollType string, lastSeenAt time.Time) {
		_, err := db.Exec(`INSERT INTO nano_enrollments
			(id, device_id, type, topic, push_magic, token_hex, last_seen_at)
			VALUES (?, ?, ?, 'topic', 'magic', 'abcd', ?)`, id, deviceID, enrollType, lastSeenAt)
		require.NoError(t, err)
	}

	insertDevice("device-1")
	insertDevice("device-2")
	seenTimes := map[string]time.Time{
		"device-1":          time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		"device-1:USER-abc": time.Date(2026, 2, 3, 4, 5, 6, 0, time.UTC),
		"device-2":          time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC),
	}
	insertEnrollment("device-1", "device-1", "Device", seenTimes["device-1"])
	insertEnrollment("device-1:USER-abc", "device-1", "User", seenTimes["device-1:USER-abc"])
	insertEnrollment("device-2", "device-2", "Device", seenTimes["device-2"])

	applyNext(t, db)

	tx, err := db.Begin()
	require.NoError(t, err)
	require.False(t, columnExists(tx, "nano_enrollments", "last_seen_at"), "last_seen_at must be dropped")
	require.NoError(t, tx.Rollback())

	var rows []struct {
		ID       string    `db:"id"`
		SeenTime time.Time `db:"seen_time"`
	}
	require.NoError(t, db.Select(&rows, `SELECT id, seen_time FROM nano_seen_times`))
	require.Len(t, rows, len(seenTimes))
	for _, row := range rows {
		want, ok := seenTimes[row.ID]
		require.True(t, ok, "unexpected nano_seen_times row %q", row.ID)
		require.True(t, want.Equal(row.SeenTime), "seen_time mismatch for %q: want %s got %s", row.ID, want, row.SeenTime)
	}
}

func TestUp_20260904125235_empty(t *testing.T) {
	db := applyUpToPrev(t)

	applyNext(t, db)

	var count int
	require.NoError(t, db.Get(&count, `SELECT COUNT(*) FROM nano_seen_times`))
	require.Zero(t, count)
}
