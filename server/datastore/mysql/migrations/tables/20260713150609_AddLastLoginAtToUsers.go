package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20260713150609, Down_20260713150609)
}

func Up_20260713150609(tx *sql.Tx) error {
	if _, err := tx.Exec(`ALTER TABLE users ADD COLUMN last_login_at TIMESTAMP NULL DEFAULT NULL`); err != nil {
		return errors.Wrap(err, "add last_login_at to users")
	}

	// Users' last activity is computed from sessions (MAX(accessed_at) per
	// user), which needs an index on user_id.
	if _, err := tx.Exec(`ALTER TABLE sessions ADD INDEX idx_sessions_user_id (user_id)`); err != nil {
		return errors.Wrap(err, "add user_id index to sessions")
	}

	// Best-effort backfill from live sessions so existing active users aren't
	// reported as never having logged in. A session is created at login, so
	// the newest session's created_at is the user's most recent login.
	// Sessions are deleted on logout and expiry, so users without a live
	// session keep a NULL last_login_at.
	// updated_at is explicitly preserved (it has ON UPDATE CURRENT_TIMESTAMP).
	if _, err := tx.Exec(`
		UPDATE users u
		JOIN (
			SELECT user_id, MAX(created_at) AS last_session_created_at
			FROM sessions
			GROUP BY user_id
		) s ON s.user_id = u.id
		SET u.last_login_at = s.last_session_created_at,
			u.updated_at = u.updated_at
	`); err != nil {
		return errors.Wrap(err, "backfill users.last_login_at from sessions")
	}

	return nil
}

func Down_20260713150609(tx *sql.Tx) error {
	return nil
}
