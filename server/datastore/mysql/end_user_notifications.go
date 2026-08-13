package mysql

import (
	"context"
	"database/sql"
	"errors"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

const endUserNotificationColumns = `
	eun.id,
	eun.uuid,
	eun.host_id,
	eun.status,
	eun.kind,
	eun.payload,
	eun.attempt_count,
	eun.next_attempt_at,
	eun.displayed_at,
	eun.execution_id,
	eun.last_exit_code,
	eun.last_reason,
	eun.expires_at,
	eun.created_at,
	eun.updated_at
`

func (ds *Datastore) NewEndUserNotification(ctx context.Context, notification *fleet.EndUserNotification) (*fleet.EndUserNotification, error) {
	const insertStmt = `
INSERT INTO end_user_notifications (
	uuid, host_id, status, kind, payload, next_attempt_at, expires_at
) VALUES (?, ?, ?, ?, ?, ?, ?)
`

	notificationUUID := notification.UUID
	if notificationUUID == "" {
		notificationUUID = uuid.NewString()
	}

	status := notification.Status
	if status == "" {
		status = fleet.EndUserNotificationPending
	}

	if _, err := ds.writer(ctx).ExecContext(ctx, insertStmt,
		notificationUUID,
		notification.HostID,
		status,
		notification.Kind,
		notification.Payload,
		notification.NextAttemptAt,
		notification.ExpiresAt,
	); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "insert end user notification")
	}

	const getStmt = `SELECT ` + endUserNotificationColumns + ` FROM end_user_notifications eun WHERE eun.uuid = ?`

	var created fleet.EndUserNotification
	if err := sqlx.GetContext(ctx, ds.writer(ctx), &created, getStmt, notificationUUID); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "load created end user notification")
	}
	return &created, nil
}

func (ds *Datastore) GetEndUserNotificationByUUID(ctx context.Context, notificationUUID string) (*fleet.EndUserNotification, error) {
	const getStmt = `SELECT ` + endUserNotificationColumns + ` FROM end_user_notifications eun WHERE eun.uuid = ?`

	var notification fleet.EndUserNotification
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &notification, getStmt, notificationUUID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ctxerr.Wrap(ctx, notFound("EndUserNotification").WithName(notificationUUID))
		}
		return nil, ctxerr.Wrap(ctx, err, "get end user notification by uuid")
	}
	return &notification, nil
}

func (ds *Datastore) GetEndUserNotificationByExecutionID(ctx context.Context, executionID string) (*fleet.EndUserNotification, error) {
	const getStmt = `SELECT ` + endUserNotificationColumns + ` FROM end_user_notifications eun WHERE eun.execution_id = ?`

	var notification fleet.EndUserNotification
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &notification, getStmt, executionID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ctxerr.Wrap(ctx, notFound("EndUserNotification").WithName(executionID))
		}
		return nil, ctxerr.Wrap(ctx, err, "get end user notification by execution id")
	}
	return &notification, nil
}

func (ds *Datastore) ListEndUserNotificationIDsForHost(ctx context.Context, hostID uint) ([]uint, error) {
	const listStmt = `
SELECT id
FROM end_user_notifications
WHERE host_id = ? AND status IN (?, ?)
ORDER BY id
`

	var ids []uint
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &ids, listStmt,
		hostID, fleet.EndUserNotificationPending, fleet.EndUserNotificationDispatched,
	); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list end user notification ids for host")
	}
	return ids, nil
}

// ListEndUserNotificationsToDispatch returns the oldest due notification for each
// of up to limit hosts, so the limit counts hosts rather than notifications and
// the caller never has to remember which hosts it has already dispatched for.
//
// Only macOS hosts running Fleet Desktop can display a notification, so the rest
// are filtered out here instead of queuing a script that could only fail.
//
// A host with a dispatched notification is skipped until that one finishes, so
// Fleet only ever has one notification in flight per host.
func (ds *Datastore) ListEndUserNotificationsToDispatch(ctx context.Context, limit int) ([]*fleet.EndUserNotification, error) {
	const listStmt = `
SELECT ` + endUserNotificationColumns + `
FROM end_user_notifications eun
	JOIN (
		SELECT MIN(due.id) AS id
		FROM end_user_notifications due
			JOIN hosts h ON h.id = due.host_id
			JOIN host_orbit_info hoi ON hoi.host_id = due.host_id
		WHERE due.status = ?
			AND (due.next_attempt_at IS NULL OR due.next_attempt_at <= NOW(6))
			AND (due.expires_at IS NULL OR due.expires_at > NOW(6))
			AND h.platform = 'darwin'
			AND hoi.desktop_version IS NOT NULL AND hoi.desktop_version != ''
			AND NOT EXISTS (
				SELECT 1 FROM end_user_notifications dispatched
				WHERE dispatched.host_id = due.host_id AND dispatched.status = ?
			)
		GROUP BY due.host_id
		ORDER BY id
		LIMIT ?
	) one_per_host ON one_per_host.id = eun.id
ORDER BY eun.id
`

	var notifications []*fleet.EndUserNotification
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &notifications, listStmt,
		fleet.EndUserNotificationPending, fleet.EndUserNotificationDispatched, limit,
	); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list end user notifications to dispatch")
	}
	return notifications, nil
}

// SetEndUserNotificationsDispatched records the execution each notification was
// queued as, in one statement. Each notification carries its own uuid and
// execution ID, so the pairing needs no per-row SQL.
//
// It is an upsert only to write many different execution IDs at once, never to
// create a notification. uuid is unique, so every row here matches an existing
// one. The one thing that deletes a notification is its host being deleted, and
// that would fail the host_id foreign key rather than insert an orphan.
func (ds *Datastore) SetEndUserNotificationsDispatched(ctx context.Context, notifications []*fleet.EndUserNotification) error {
	if len(notifications) == 0 {
		return nil
	}

	const upsertStmt = `
INSERT INTO end_user_notifications
	(uuid, host_id, kind, payload, status, execution_id, attempt_count)
VALUES
	(:uuid, :host_id, :kind, :payload, :status, :execution_id, :attempt_count)
ON DUPLICATE KEY UPDATE
	status = VALUES(status),
	execution_id = VALUES(execution_id),
	attempt_count = VALUES(attempt_count)
`

	args := make([]map[string]any, 0, len(notifications))
	for _, notification := range notifications {
		args = append(args, map[string]any{
			"uuid":         notification.UUID,
			"host_id":      notification.HostID,
			"kind":         notification.Kind,
			"payload":      notification.Payload,
			"status":       fleet.EndUserNotificationDispatched,
			"execution_id": notification.ExecutionID,
			// the dispatcher is the only writer of attempt_count, and one cron
			// runs at a time, so counting up from the value just read is safe
			"attempt_count": notification.AttemptCount + 1,
		})
	}

	if _, err := sqlx.NamedExecContext(ctx, ds.writer(ctx), upsertStmt, args); err != nil {
		return ctxerr.Wrap(ctx, err, "set end user notifications dispatched")
	}
	return nil
}

// ExpireEndUserNotifications sets notifications past their expiry to expired so
// they are never dispatched again, and returns how many it expired.
func (ds *Datastore) ExpireEndUserNotifications(ctx context.Context) (int64, error) {
	const updateStmt = `
UPDATE end_user_notifications
SET status = ?
WHERE expires_at IS NOT NULL AND expires_at <= NOW(6) AND status IN (?, ?)
`

	res, err := ds.writer(ctx).ExecContext(ctx, updateStmt,
		fleet.EndUserNotificationExpired, fleet.EndUserNotificationPending, fleet.EndUserNotificationDispatched,
	)
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "expire end user notifications")
	}

	expired, err := res.RowsAffected()
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "count expired end user notifications")
	}
	return expired, nil
}

// UpdateEndUserNotification applies what an end user's device reported about one
// of their notifications. The fields the device set say which action it took, and
// each action has its own query.
func (ds *Datastore) UpdateEndUserNotification(ctx context.Context, notificationUUID string, action fleet.EndUserNotificationAction) error {
	switch {
	case action.DisplayedAt != nil:
		return ds.verifyEndUserNotification(ctx, notificationUUID, action)
	case action.NextAttemptAt != nil:
		return ds.snoozeEndUserNotification(ctx, notificationUUID, action)
	}
	return nil
}

// the device confirmed the notification reached the end user, so Fleet stops
// trying to deliver it
func (ds *Datastore) verifyEndUserNotification(ctx context.Context, notificationUUID string, action fleet.EndUserNotificationAction) error {
	const updateStmt = `
UPDATE end_user_notifications
SET status = ?, displayed_at = ?
WHERE uuid = ?
`

	if _, err := ds.writer(ctx).ExecContext(ctx, updateStmt,
		fleet.EndUserNotificationDisplayed, action.DisplayedAt, notificationUUID,
	); err != nil {
		return ctxerr.Wrap(ctx, err, "verify end user notification")
	}
	return nil
}

// the notification goes back in the queue for a later attempt, so it drops the
// execution it was dispatched with
func (ds *Datastore) snoozeEndUserNotification(ctx context.Context, notificationUUID string, action fleet.EndUserNotificationAction) error {
	const updateStmt = `
UPDATE end_user_notifications
SET status = ?, next_attempt_at = ?, last_reason = ?, execution_id = NULL
WHERE uuid = ?
`

	if _, err := ds.writer(ctx).ExecContext(ctx, updateStmt,
		fleet.EndUserNotificationPending, action.NextAttemptAt, action.LastReason, notificationUUID,
	); err != nil {
		return ctxerr.Wrap(ctx, err, "snooze end user notification")
	}
	return nil
}
