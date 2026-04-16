package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

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
		`SELECT id, type, severity, title, body, cta_url, cta_label,
		        COALESCE(metadata, CAST('null' AS JSON)) AS metadata,
		        dedupe_key, audience, resolved_at, created_at, updated_at
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

// disabledTypesForUserInApp returns the set of NotificationTypes the user has
// silenced for the in-app channel. It looks up the user's opt-out rows and
// expands them to types via fleet.NotificationTypeCategory. An unknown type
// (one not in the registry) is never filtered here — we default to showing
// unknown notifications so a newly-added producer without a category mapping
// is still visible.
func (ds *Datastore) disabledTypesForUserInApp(ctx context.Context, userID uint) ([]fleet.NotificationType, error) {
	prefs, err := ds.ListUserNotificationPreferences(ctx, userID)
	if err != nil {
		return nil, err
	}
	disabledCats := make(map[fleet.NotificationCategory]struct{}, len(prefs))
	for _, p := range prefs {
		if p.Channel == fleet.NotificationChannelInApp && !p.Enabled {
			disabledCats[p.Category] = struct{}{}
		}
	}
	if len(disabledCats) == 0 {
		return nil, nil
	}
	var disabledTypes []fleet.NotificationType
	for t, c := range fleet.NotificationTypeCategory {
		if _, ok := disabledCats[c]; ok {
			disabledTypes = append(disabledTypes, t)
		}
	}
	return disabledTypes, nil
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

	disabledTypes, err := ds.disabledTypesForUserInApp(ctx, userID)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "load user notification preferences")
	}

	query := `
		SELECT n.id, n.type, n.severity, n.title, n.body, n.cta_url, n.cta_label,
		       COALESCE(n.metadata, CAST('null' AS JSON)) AS metadata,
		       n.dedupe_key, n.audience, n.resolved_at, n.created_at, n.updated_at,
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
	if len(disabledTypes) > 0 {
		query += ` AND n.type NOT IN (?)`
		args = append(args, disabledTypes)
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
		       COALESCE(n.metadata, CAST('null' AS JSON)) AS metadata,
		       n.dedupe_key, n.audience, n.resolved_at, n.created_at, n.updated_at,
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

	disabledTypes, err := ds.disabledTypesForUserInApp(ctx, userID)
	if err != nil {
		return 0, 0, ctxerr.Wrap(ctx, err, "load user notification preferences")
	}

	q := `
		SELECT
			SUM(CASE WHEN uns.read_at IS NULL THEN 1 ELSE 0 END) AS unread,
			COUNT(*) AS active
		  FROM notifications n
		  LEFT JOIN user_notification_state uns
		    ON uns.notification_id = n.id AND uns.user_id = ?
		 WHERE n.audience IN (?)
		   AND n.resolved_at IS NULL
		   AND (uns.dismissed_at IS NULL)`
	qArgs := []interface{}{userID, audiences}
	if len(disabledTypes) > 0 {
		q += ` AND n.type NOT IN (?)`
		qArgs = append(qArgs, disabledTypes)
	}
	stmt, args, err := sqlx.In(q, qArgs...)
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

// ListUserNotificationPreferences returns the opt-out rows for the given user.
// Rows not present default to "enabled" so the caller fills defaults itself
// when it needs a complete grid.
func (ds *Datastore) ListUserNotificationPreferences(
	ctx context.Context, userID uint,
) ([]fleet.UserNotificationPreference, error) {
	var rows []fleet.UserNotificationPreference
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &rows,
		`SELECT user_id, category, channel, enabled
		   FROM user_notification_preferences
		  WHERE user_id = ?`,
		userID,
	); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list user notification preferences")
	}
	return rows, nil
}

// UpsertUserNotificationPreferences writes the provided preferences for the
// given user. To keep storage minimal, an Enabled=true row is deleted rather
// than stored — the default is enabled, so the absence of a row encodes it.
// Enabled=false rows are upserted.
func (ds *Datastore) UpsertUserNotificationPreferences(
	ctx context.Context, userID uint, prefs []fleet.UserNotificationPreference,
) error {
	return ds.withTx(ctx, func(tx sqlx.ExtContext) error {
		for _, p := range prefs {
			if p.Enabled {
				if _, err := tx.ExecContext(ctx,
					`DELETE FROM user_notification_preferences
					  WHERE user_id = ? AND category = ? AND channel = ?`,
					userID, p.Category, p.Channel,
				); err != nil {
					return ctxerr.Wrap(ctx, err, "delete user notification preference")
				}
				continue
			}
			if _, err := tx.ExecContext(ctx,
				`INSERT INTO user_notification_preferences (user_id, category, channel, enabled)
				 VALUES (?, ?, ?, 0)
				 ON DUPLICATE KEY UPDATE enabled = 0, updated_at = CURRENT_TIMESTAMP(6)`,
				userID, p.Category, p.Channel,
			); err != nil {
				return ctxerr.Wrap(ctx, err, "upsert user notification preference")
			}
		}
		return nil
	})
}

// EnqueueNotificationDelivery inserts a pending delivery row. The unique
// index uq_nd_notification_channel_target makes this a no-op for duplicates,
// so producers can fan out on every cron tick without double-scheduling.
func (ds *Datastore) EnqueueNotificationDelivery(
	ctx context.Context, notificationID uint, channel fleet.NotificationChannel, target string,
) error {
	_, err := ds.writer(ctx).ExecContext(ctx,
		`INSERT IGNORE INTO notification_deliveries (notification_id, channel, target, status)
		 VALUES (?, ?, ?, 'pending')`,
		notificationID, channel, target,
	)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "enqueue notification delivery")
	}
	return nil
}

// ClaimPendingDeliveries atomically marks up to `limit` pending delivery rows
// for the given channel as in-flight ("sending") and returns them along with
// the referenced notification rows. A separate "sending" state is used so
// multiple worker instances running this cron in parallel don't grab the
// same rows — whoever wins the UPDATE owns the rows until they mark them
// sent/failed.
func (ds *Datastore) ClaimPendingDeliveries(
	ctx context.Context, channel fleet.NotificationChannel, limit int,
) ([]*fleet.NotificationDelivery, map[uint]*fleet.Notification, error) {
	if limit <= 0 {
		limit = 50
	}
	claimID := time.Now().UnixNano()

	// Stage 1: claim a batch by stamping a unique marker into `error` so we
	// can re-select exactly what we just took. `error` is otherwise NULL for
	// pending rows; this piggybacks on the existing column without needing
	// another schema change. The marker is cleared on MarkDeliveryResult.
	marker := fmt.Sprintf("claim:%d", claimID)
	if _, err := ds.writer(ctx).ExecContext(ctx, `
		UPDATE notification_deliveries
		   SET status = 'sending', error = ?, attempted_at = CURRENT_TIMESTAMP(6)
		 WHERE channel = ? AND status = 'pending'
		 ORDER BY id
		 LIMIT ?`,
		marker, channel, limit,
	); err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "claim pending deliveries")
	}

	// Stage 2: read back the claimed rows.
	var rows []*fleet.NotificationDelivery
	if err := sqlx.SelectContext(ctx, ds.writer(ctx), &rows, `
		SELECT id, notification_id, channel, target, status, error, attempted_at, created_at, updated_at
		  FROM notification_deliveries
		 WHERE channel = ? AND status = 'sending' AND error = ?`,
		channel, marker,
	); err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "load claimed deliveries")
	}
	if len(rows) == 0 {
		return nil, nil, nil
	}

	// Stage 3: bulk-fetch the notifications those deliveries reference so the
	// caller doesn't issue N+1 queries.
	ids := make([]uint, 0, len(rows))
	seen := make(map[uint]struct{}, len(rows))
	for _, r := range rows {
		if _, ok := seen[r.NotificationID]; ok {
			continue
		}
		seen[r.NotificationID] = struct{}{}
		ids = append(ids, r.NotificationID)
	}
	stmt, args, err := sqlx.In(`
		SELECT id, type, severity, title, body, cta_url, cta_label,
		       COALESCE(metadata, CAST('null' AS JSON)) AS metadata,
		       dedupe_key, audience, resolved_at, created_at, updated_at
		  FROM notifications
		 WHERE id IN (?)`, ids)
	if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "build notifications lookup for deliveries")
	}
	var notifs []*fleet.Notification
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &notifs, stmt, args...); err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "load notifications for deliveries")
	}
	byID := make(map[uint]*fleet.Notification, len(notifs))
	for _, n := range notifs {
		byID[n.ID] = n
	}
	return rows, byID, nil
}

// MarkDeliveryResult records the outcome of a send attempt. status == sent
// clears the error column; status == failed stores the message truncated to
// a safe length so a long error payload doesn't blow up the row.
func (ds *Datastore) MarkDeliveryResult(
	ctx context.Context, deliveryID uint, status fleet.NotificationDeliveryStatus, errMsg string,
) error {
	const maxErrLen = 4000
	if len(errMsg) > maxErrLen {
		errMsg = errMsg[:maxErrLen]
	}
	var errCol interface{}
	if errMsg == "" {
		errCol = nil
	} else {
		errCol = errMsg
	}
	if _, err := ds.writer(ctx).ExecContext(ctx, `
		UPDATE notification_deliveries
		   SET status = ?, error = ?, attempted_at = CURRENT_TIMESTAMP(6)
		 WHERE id = ?`,
		status, errCol, deliveryID,
	); err != nil {
		return ctxerr.Wrap(ctx, err, "mark delivery result")
	}
	return nil
}
