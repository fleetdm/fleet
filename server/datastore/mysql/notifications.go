package mysql

import (
	"context"
	"database/sql"
	"errors"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

// UpsertNotification creates or updates-in-place by dedupe_key.
//
// Behavior:
//   - New row: insert with current fields; resolved_at = NULL.
//   - Existing row matching dedupe_key: update all producer-owned fields and
//     clear resolved_at so a re-triggered condition re-surfaces to users. The
//     per-user state (read/dismissed) is NOT cleared — admins who already
//     acknowledged it stay acknowledged unless they explicitly restore it.
//     That keeps dismissals sticky across cron re-runs while still surfacing
//     genuine re-occurrences via refreshed title/body/severity.
func (ds *Datastore) UpsertNotification(ctx context.Context, u fleet.NotificationUpsert) (*fleet.Notification, error) {
	if u.DedupeKey == "" {
		return nil, ctxerr.New(ctx, "notification dedupe_key is required")
	}
	if u.Audience == "" {
		u.Audience = fleet.NotificationAudienceAdmin
	}

	const upsert = `
		INSERT INTO notifications (
			type, severity, title, body, cta_url, cta_label, metadata, dedupe_key, audience, resolved_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, NULL)
		ON DUPLICATE KEY UPDATE
			type = VALUES(type),
			severity = VALUES(severity),
			title = VALUES(title),
			body = VALUES(body),
			cta_url = VALUES(cta_url),
			cta_label = VALUES(cta_label),
			metadata = VALUES(metadata),
			audience = VALUES(audience),
			resolved_at = NULL,
			updated_at = CURRENT_TIMESTAMP(6)
	`
	if _, err := ds.writer(ctx).ExecContext(ctx, upsert,
		u.Type, u.Severity, u.Title, u.Body, u.CTAURL, u.CTALabel, u.Metadata, u.DedupeKey, u.Audience,
	); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "upsert notification")
	}

	// Read the canonical row back (ON DUPLICATE KEY UPDATE does not give us
	// a reliable LAST_INSERT_ID for updates).
	var n fleet.Notification
	if err := sqlx.GetContext(ctx, ds.writer(ctx), &n,
		`SELECT id, type, severity, title, body, cta_url, cta_label, metadata, dedupe_key, audience,
		        resolved_at, created_at, updated_at
		   FROM notifications WHERE dedupe_key = ?`,
		u.DedupeKey,
	); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "read back upserted notification")
	}
	return &n, nil
}

// ResolveNotification marks a notification row as resolved by dedupe_key.
// Idempotent: no error if no matching row exists.
func (ds *Datastore) ResolveNotification(ctx context.Context, dedupeKey string) error {
	if dedupeKey == "" {
		return ctxerr.New(ctx, "dedupe_key required")
	}
	_, err := ds.writer(ctx).ExecContext(ctx,
		`UPDATE notifications SET resolved_at = CURRENT_TIMESTAMP(6)
		 WHERE dedupe_key = ? AND resolved_at IS NULL`,
		dedupeKey,
	)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "resolve notification")
	}
	return nil
}

// audienceForUser returns the list of notification audiences a user should
// see. For v1 this is just "admin" if the user is a global admin, else empty.
func audienceForUser(user *fleet.User) []fleet.NotificationAudience {
	if user == nil || user.GlobalRole == nil {
		return nil
	}
	if *user.GlobalRole == fleet.RoleAdmin {
		return []fleet.NotificationAudience{fleet.NotificationAudienceAdmin}
	}
	return nil
}

// ListNotificationsForUser returns notifications for the given user joined
// with per-user state. The service layer has already authorized the caller;
// this method filters by audience based on the user's global role.
func (ds *Datastore) ListNotificationsForUser(
	ctx context.Context, userID uint, filter fleet.NotificationListFilter,
) ([]*fleet.Notification, error) {
	user, err := ds.UserByID(ctx, userID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "load user for notification list")
	}
	audiences := audienceForUser(user)
	if len(audiences) == 0 {
		return []*fleet.Notification{}, nil
	}

	query := `
		SELECT n.id, n.type, n.severity, n.title, n.body, n.cta_url, n.cta_label,
		       n.metadata, n.dedupe_key, n.audience, n.resolved_at, n.created_at, n.updated_at,
		       uns.read_at, uns.dismissed_at
		  FROM notifications n
		  LEFT JOIN user_notification_state uns
		    ON uns.notification_id = n.id AND uns.user_id = ?
		 WHERE n.audience IN (?)`
	args := []interface{}{userID, audiences}

	if !filter.IncludeResolved {
		query += ` AND n.resolved_at IS NULL`
	}
	if !filter.IncludeDismissed {
		query += ` AND uns.dismissed_at IS NULL`
	}
	query += ` ORDER BY
		FIELD(n.severity, 'error', 'warning', 'info'),
		n.created_at DESC`

	stmt, args, err := sqlx.In(query, args...)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "build notifications query")
	}

	var rows []*fleet.Notification
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &rows, stmt, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list notifications")
	}
	if rows == nil {
		rows = []*fleet.Notification{}
	}
	return rows, nil
}

// NotificationByIDForUser loads one notification joined with the user's
// per-user state. Returns a notFound error if the user's role does not match
// the notification's audience.
func (ds *Datastore) NotificationByIDForUser(
	ctx context.Context, notificationID, userID uint,
) (*fleet.Notification, error) {
	user, err := ds.UserByID(ctx, userID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "load user for notification lookup")
	}
	audiences := audienceForUser(user)
	if len(audiences) == 0 {
		return nil, ctxerr.Wrap(ctx, notFound("Notification").WithID(notificationID))
	}

	query := `
		SELECT n.id, n.type, n.severity, n.title, n.body, n.cta_url, n.cta_label,
		       n.metadata, n.dedupe_key, n.audience, n.resolved_at, n.created_at, n.updated_at,
		       uns.read_at, uns.dismissed_at
		  FROM notifications n
		  LEFT JOIN user_notification_state uns
		    ON uns.notification_id = n.id AND uns.user_id = ?
		 WHERE n.id = ? AND n.audience IN (?)`
	stmt, args, err := sqlx.In(query, userID, notificationID, audiences)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "build notification lookup query")
	}

	var n fleet.Notification
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &n, stmt, args...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ctxerr.Wrap(ctx, notFound("Notification").WithID(notificationID))
		}
		return nil, ctxerr.Wrap(ctx, err, "lookup notification")
	}
	return &n, nil
}

// DismissNotification upserts per-user state with dismissed_at = NOW.
func (ds *Datastore) DismissNotification(ctx context.Context, notificationID, userID uint) error {
	return ds.setNotificationStateTimestamp(ctx, notificationID, userID, "dismissed_at", true)
}

// RestoreNotification clears dismissed_at on the per-user state row.
func (ds *Datastore) RestoreNotification(ctx context.Context, notificationID, userID uint) error {
	return ds.setNotificationStateTimestamp(ctx, notificationID, userID, "dismissed_at", false)
}

// MarkNotificationRead upserts per-user state with read_at = NOW.
func (ds *Datastore) MarkNotificationRead(ctx context.Context, notificationID, userID uint) error {
	return ds.setNotificationStateTimestamp(ctx, notificationID, userID, "read_at", true)
}

// setNotificationStateTimestamp is a helper that upserts the (user,
// notification) row in user_notification_state with either NOW or NULL for
// the given column. The column name is not user-supplied so no SQL injection
// risk.
func (ds *Datastore) setNotificationStateTimestamp(
	ctx context.Context, notificationID, userID uint, column string, set bool,
) error {
	// Guard column against typos — only two valid values.
	if column != "dismissed_at" && column != "read_at" {
		return ctxerr.New(ctx, "invalid per-user state column")
	}

	// When clearing, simply update if the row exists (no-op if it doesn't).
	if !set {
		_, err := ds.writer(ctx).ExecContext(ctx,
			"UPDATE user_notification_state SET "+column+" = NULL WHERE user_id = ? AND notification_id = ?",
			userID, notificationID,
		)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "clear notification state timestamp")
		}
		return nil
	}

	// Setting: upsert with NOW(6). All non-targeted columns keep existing
	// values or default to NULL.
	stmt := `
		INSERT INTO user_notification_state (user_id, notification_id, ` + column + `)
		VALUES (?, ?, CURRENT_TIMESTAMP(6))
		ON DUPLICATE KEY UPDATE ` + column + ` = CURRENT_TIMESTAMP(6)
	`
	if _, err := ds.writer(ctx).ExecContext(ctx, stmt, userID, notificationID); err != nil {
		return ctxerr.Wrap(ctx, err, "set notification state timestamp")
	}
	return nil
}

// MarkAllNotificationsRead marks every active, non-dismissed, unread
// notification visible to the user as read.
func (ds *Datastore) MarkAllNotificationsRead(ctx context.Context, userID uint) error {
	user, err := ds.UserByID(ctx, userID)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "load user for mark-all-read")
	}
	audiences := audienceForUser(user)
	if len(audiences) == 0 {
		return nil
	}

	// For each active notification without an existing state row, insert one
	// with read_at = NOW. For existing rows where read_at IS NULL, set it.
	// A single INSERT ... SELECT ... ON DUPLICATE KEY UPDATE handles both.
	stmt, args, err := sqlx.In(`
		INSERT INTO user_notification_state (user_id, notification_id, read_at)
		SELECT ?, n.id, CURRENT_TIMESTAMP(6)
		  FROM notifications n
		  LEFT JOIN user_notification_state uns
		    ON uns.notification_id = n.id AND uns.user_id = ?
		 WHERE n.audience IN (?)
		   AND n.resolved_at IS NULL
		   AND (uns.dismissed_at IS NULL OR uns.dismissed_at IS NULL)
		   AND (uns.read_at IS NULL)
		ON DUPLICATE KEY UPDATE read_at = CURRENT_TIMESTAMP(6)
	`, userID, userID, audiences)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "build mark-all-read query")
	}
	if _, err := ds.writer(ctx).ExecContext(ctx, stmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "mark all notifications read")
	}
	return nil
}

// CountActiveNotificationsForUser returns (unread, active). Active = not
// dismissed, not resolved. Unread = active and not yet read.
func (ds *Datastore) CountActiveNotificationsForUser(
	ctx context.Context, userID uint,
) (int, int, error) {
	user, err := ds.UserByID(ctx, userID)
	if err != nil {
		return 0, 0, ctxerr.Wrap(ctx, err, "load user for notification count")
	}
	audiences := audienceForUser(user)
	if len(audiences) == 0 {
		return 0, 0, nil
	}

	stmt, args, err := sqlx.In(`
		SELECT
			SUM(CASE WHEN uns.read_at IS NULL THEN 1 ELSE 0 END) AS unread,
			COUNT(*) AS active
		  FROM notifications n
		  LEFT JOIN user_notification_state uns
		    ON uns.notification_id = n.id AND uns.user_id = ?
		 WHERE n.audience IN (?)
		   AND n.resolved_at IS NULL
		   AND (uns.dismissed_at IS NULL)
	`, userID, audiences)
	if err != nil {
		return 0, 0, ctxerr.Wrap(ctx, err, "build notification count query")
	}

	var row struct {
		Unread sql.NullInt64 `db:"unread"`
		Active sql.NullInt64 `db:"active"`
	}
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &row, stmt, args...); err != nil {
		return 0, 0, ctxerr.Wrap(ctx, err, "count notifications")
	}
	return int(row.Unread.Int64), int(row.Active.Int64), nil
}
