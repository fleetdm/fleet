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

// ListEndUserNotificationsToDispatch returns the oldest due notifications, at most
// one per host, so the caller never has to remember which hosts it has already
// dispatched for. limit bounds how many notifications are read, so a batch holds
// fewer hosts than that when a host has several due at once; the rest of that
// host's wait for a later pass.
//
// Only macOS hosts running Fleet Desktop can display a notification, so the rest
// are filtered out here instead of queuing a script that could only fail.
//
// A host with a dispatched notification is skipped until that one finishes, so
// Fleet only ever has one notification in flight per host.
//
// One per host is applied in Go rather than by a GROUP BY, because grouping has to
// finish over every dispatchable notification before LIMIT applies: measured at
// 10ms for a batch against a backlog of 2k and 323ms against 100k, where this form
// stays at 4ms. The map it costs holds one batch, not one entry per host in the
// fleet.
func (ds *Datastore) ListEndUserNotificationsToDispatch(ctx context.Context, limit int) ([]*fleet.EndUserNotification, error) {
	const listStmt = `
SELECT ` + endUserNotificationColumns + `
FROM end_user_notifications eun
	JOIN hosts h ON h.id = eun.host_id
	JOIN host_orbit_info hoi ON hoi.host_id = eun.host_id
WHERE eun.status = ?
	AND (eun.next_attempt_at IS NULL OR eun.next_attempt_at <= NOW(6))
	AND (eun.expires_at IS NULL OR eun.expires_at > NOW(6))
	AND h.platform = 'darwin'
	AND hoi.desktop_version IS NOT NULL AND hoi.desktop_version != ''
	AND NOT EXISTS (
		SELECT 1 FROM end_user_notifications dispatched
		WHERE dispatched.host_id = eun.host_id AND dispatched.status = ?
	)
ORDER BY eun.id
LIMIT ?
`

	var due []*fleet.EndUserNotification
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &due, listStmt,
		fleet.EndUserNotificationPending, fleet.EndUserNotificationDispatched, limit,
	); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list end user notifications to dispatch")
	}

	notifications := make([]*fleet.EndUserNotification, 0, len(due))
	hostsSeen := make(map[uint]struct{}, len(due))
	for _, notification := range due {
		if _, seen := hostsSeen[notification.HostID]; seen {
			continue
		}
		hostsSeen[notification.HostID] = struct{}{}
		notifications = append(notifications, notification)
	}
	return notifications, nil
}

// SetEndUserNotificationsDispatched records the execution each notification was
// queued as, in one statement.
//
// The execution IDs differ per row, so rather than send a value per row it joins
// the executions that were just queued and reads each one from there. One host
// has one of them, and the uuid list picks one notification per host, so the join
// matches a single row on both sides.
func (ds *Datastore) SetEndUserNotificationsDispatched(ctx context.Context, notifications []*fleet.EndUserNotification) error {
	if len(notifications) == 0 {
		return nil
	}

	const updateStmt = `
UPDATE end_user_notifications eun
	JOIN upcoming_activities ua
		ON ua.host_id = eun.host_id AND ua.execution_id IN (?)
SET
	eun.status = ?,
	eun.execution_id = ua.execution_id,
	eun.attempt_count = eun.attempt_count + 1
WHERE
	eun.uuid IN (?)
`

	executionIDs := make([]string, 0, len(notifications))
	uuids := make([]string, 0, len(notifications))
	for _, notification := range notifications {
		if notification.ExecutionID == nil {
			return ctxerr.New(ctx, "end user notification dispatched without an execution id")
		}
		executionIDs = append(executionIDs, *notification.ExecutionID)
		uuids = append(uuids, notification.UUID)
	}

	stmt, args, err := sqlx.In(updateStmt, executionIDs, fleet.EndUserNotificationDispatched, uuids)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "build end user notifications dispatched update")
	}

	if _, err := ds.writer(ctx).ExecContext(ctx, stmt, args...); err != nil {
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
