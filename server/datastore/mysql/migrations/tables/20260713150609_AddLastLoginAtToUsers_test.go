package tables

import (
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func TestUp_20260713150609(t *testing.T) {
	db := applyUpToPrev(t)

	// user with live sessions: backfilled from the most recent session
	userWithSession := execNoErrLastID(t, db,
		`INSERT INTO users (name, email, password, salt) VALUES (?, ?, ?, ?)`,
		"with-session", "with-session@example.com", "p", "s",
	)
	// user without sessions: last_login_at stays NULL
	userWithoutSession := execNoErrLastID(t, db,
		`INSERT INTO users (name, email, password, salt) VALUES (?, ?, ?, ?)`,
		"without-session", "without-session@example.com", "p", "s",
	)

	older := time.Now().UTC().Add(-48 * time.Hour).Truncate(time.Second)
	newer := time.Now().UTC().Add(-1 * time.Hour).Truncate(time.Second)
	// accessed_at is more recent than created_at; the backfill must use
	// created_at (login time), not accessed_at (last activity).
	execNoErr(t, db, "INSERT INTO sessions (user_id, `key`, created_at, accessed_at) VALUES (?, ?, ?, ?)", userWithSession, "session-key-1", older, newer)
	execNoErr(t, db, "INSERT INTO sessions (user_id, `key`, created_at, accessed_at) VALUES (?, ?, ?, ?)", userWithSession, "session-key-2", newer, time.Now().UTC())

	var updatedAtBefore time.Time
	require.NoError(t, sqlx.Get(db, &updatedAtBefore, `SELECT updated_at FROM users WHERE id = ?`, userWithSession))

	applyNext(t, db)

	var lastLoginAt *time.Time
	require.NoError(t, sqlx.Get(db, &lastLoginAt, `SELECT last_login_at FROM users WHERE id = ?`, userWithSession))
	require.NotNil(t, lastLoginAt)
	require.WithinDuration(t, newer, *lastLoginAt, time.Second)

	// the backfill must not bump updated_at (ON UPDATE CURRENT_TIMESTAMP)
	var updatedAtAfter time.Time
	require.NoError(t, sqlx.Get(db, &updatedAtAfter, `SELECT updated_at FROM users WHERE id = ?`, userWithSession))
	require.Equal(t, updatedAtBefore, updatedAtAfter)

	lastLoginAt = nil
	require.NoError(t, sqlx.Get(db, &lastLoginAt, `SELECT last_login_at FROM users WHERE id = ?`, userWithoutSession))
	require.Nil(t, lastLoginAt)
}
