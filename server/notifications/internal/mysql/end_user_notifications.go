// Package mysql provides the MySQL datastore implementation for the
// notifications bounded context.
package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/notifications/api"
	"github.com/fleetdm/fleet/v4/server/notifications/internal/types"
	platform_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Datastore is the MySQL implementation of the notifications datastore.
type Datastore struct {
	primary *sqlx.DB
	replica *sqlx.DB
	logger  *slog.Logger
}

// NewDatastore creates a new MySQL datastore for notifications.
func NewDatastore(conns *platform_mysql.DBConnections, logger *slog.Logger) *Datastore {
	return &Datastore{primary: conns.Primary, replica: conns.Replica, logger: logger} // nolint:nilaway // conns is never nil by the time serve.go builds the notifications bounded context
}

func (ds *Datastore) reader(ctx context.Context) *sqlx.DB {
	return ds.replica
}

// Ensure Datastore implements types.Datastore
var _ types.Datastore = (*Datastore)(nil)

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

func (ds *Datastore) NewEndUserNotification(ctx context.Context, notification *api.EndUserNotification) (*api.EndUserNotification, error) {
	const insertStmt = `
INSERT INTO notifications_end_user (
	uuid, host_id, status, kind, payload, next_attempt_at, expires_at
) VALUES (?, ?, ?, ?, ?, ?, ?)
`

	notificationUUID := notification.UUID
	if notificationUUID == "" {
		notificationUUID = uuid.NewString()
	}

	status := notification.Status
	if status == "" {
		status = api.EndUserNotificationPending
	}

	if _, err := ds.primary.ExecContext(ctx, insertStmt,
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

	const getStmt = `SELECT ` + endUserNotificationColumns + ` FROM notifications_end_user eun WHERE eun.uuid = ?`

	var created api.EndUserNotification
	if err := sqlx.GetContext(ctx, ds.primary, &created, getStmt, notificationUUID); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "load created end user notification")
	}
	return &created, nil
}

func (ds *Datastore) GetEndUserNotificationByUUID(ctx context.Context, notificationUUID string) (*api.EndUserNotification, error) {
	const getStmt = `SELECT ` + endUserNotificationColumns + ` FROM notifications_end_user eun WHERE eun.uuid = ?`

	var notification api.EndUserNotification
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &notification, getStmt, notificationUUID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ctxerr.Wrap(ctx, &types.NotFoundError{Identifier: notificationUUID})
		}
		return nil, ctxerr.Wrap(ctx, err, "get end user notification by uuid")
	}
	return &notification, nil
}

// GetEndUserNotificationByExecutionID reads from primary because callers look
// up an execution the dispatcher may have written a moment ago.
func (ds *Datastore) GetEndUserNotificationByExecutionID(ctx context.Context, executionID string) (*api.EndUserNotification, error) {
	const getStmt = `SELECT ` + endUserNotificationColumns + ` FROM notifications_end_user eun WHERE eun.execution_id = ?`

	var notification api.EndUserNotification
	if err := sqlx.GetContext(ctx, ds.primary, &notification, getStmt, executionID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ctxerr.Wrap(ctx, &types.NotFoundError{Identifier: executionID})
		}
		return nil, ctxerr.Wrap(ctx, err, "get end user notification by execution id")
	}
	return &notification, nil
}

// ListEndUserNotificationsToDispatch returns the oldest due notifications, at most
// one per host; limit bounds notifications read, not hosts returned.
//
// Only macOS hosts running fleetd are considered; whether Fleet Desktop itself is
// installed and new enough is the script's own check (exit 100/101). A host with
// an undelivered dispatch is skipped, so only one notification is ever in flight
// per host.
//
// Dedupes to one per host in Go rather than with GROUP BY: grouping has to finish
// over the whole backlog before LIMIT applies (323ms at 100k rows vs 4ms here).
func (ds *Datastore) ListEndUserNotificationsToDispatch(ctx context.Context, limit int) ([]*api.EndUserNotification, error) {
	const listStmt = `
SELECT ` + endUserNotificationColumns + `
FROM notifications_end_user eun
	JOIN hosts h ON h.id = eun.host_id
	JOIN host_orbit_info hoi ON hoi.host_id = eun.host_id
WHERE eun.status = ?
	AND (eun.next_attempt_at IS NULL OR eun.next_attempt_at <= NOW(6))
	AND (eun.expires_at IS NULL OR eun.expires_at > NOW(6))
	AND h.platform = 'darwin'
	AND NOT EXISTS (
		SELECT 1 FROM notifications_end_user dispatched
		WHERE dispatched.host_id = eun.host_id AND dispatched.status = ?
			AND dispatched.displayed_at IS NULL
	)
ORDER BY eun.id
LIMIT ?
`

	var due []*api.EndUserNotification
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &due, listStmt,
		api.EndUserNotificationPending, api.EndUserNotificationDispatched, limit,
	); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "list end user notifications to dispatch")
	}

	notifications := make([]*api.EndUserNotification, 0, len(due))
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

// SetEndUserNotificationsDispatched records the execution each notification
// was queued as. The pairs are joined in as a literal derived table rather
// than looked up in upcoming_activities, since a script that finishes before
// this runs has already had its queue row deleted.
func (ds *Datastore) SetEndUserNotificationsDispatched(ctx context.Context, notifications []*api.EndUserNotification) error {
	if len(notifications) == 0 {
		return nil
	}

	// Only the number of SELECTs varies with the input: each one is the same
	// literal, and every uuid and execution id is bound as an argument.
	selectParts := make([]string, 0, len(notifications))
	args := make([]any, 0, len(notifications)*2+1)
	for _, notification := range notifications {
		if notification.ExecutionID == nil {
			return ctxerr.New(ctx, "end user notification dispatched without an execution id")
		}
		selectParts = append(selectParts, "SELECT ? AS uuid, ? AS execution_id")
		args = append(args, notification.UUID, *notification.ExecutionID)
	}
	args = append(args, api.EndUserNotificationDispatched)

	updateStmt := fmt.Sprintf(`
UPDATE notifications_end_user eun
	JOIN (%s) queued ON queued.uuid = eun.uuid
SET
	eun.status = ?,
	eun.execution_id = queued.execution_id,
	eun.attempt_count = eun.attempt_count + 1
`, strings.Join(selectParts, " UNION ALL "))

	if _, err := ds.primary.ExecContext(ctx, updateStmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "set end user notifications dispatched")
	}
	return nil
}

// ExpireEndUserNotifications gives up on notifications past their expiry, and
// on dispatched ones unanswered for EndUserNotificationStuckDispatchTimeout,
// returning how many of both. One already displayed is left alone regardless
// of expiry.
//
// The two are separate statements rather than one OR: an OR across expires_at
// and updated_at can use neither index, which turns this once-a-minute sweep
// into a full table scan. Expiry runs first, so a row that qualifies for both
// is counted once.
func (ds *Datastore) ExpireEndUserNotifications(ctx context.Context) (int64, error) {
	const expiryStmt = `
UPDATE notifications_end_user
SET status = ?
WHERE displayed_at IS NULL
	AND status IN (?, ?)
	AND expires_at IS NOT NULL AND expires_at <= NOW(6)
`

	const stuckStmt = `
UPDATE notifications_end_user
SET status = ?
WHERE displayed_at IS NULL
	AND status = ?
	AND updated_at <= ?
`

	res, err := ds.primary.ExecContext(ctx, expiryStmt,
		api.EndUserNotificationExpired, api.EndUserNotificationPending, api.EndUserNotificationDispatched,
	)
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "expire end user notifications")
	}
	expired, err := res.RowsAffected()
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "count expired end user notifications")
	}

	stuckBefore := time.Now().UTC().Add(-api.EndUserNotificationStuckDispatchTimeout)
	res, err = ds.primary.ExecContext(ctx, stuckStmt,
		api.EndUserNotificationExpired, api.EndUserNotificationDispatched, stuckBefore,
	)
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "expire stuck end user notifications")
	}
	stuck, err := res.RowsAffected()
	if err != nil {
		return 0, ctxerr.Wrap(ctx, err, "count stuck end user notifications")
	}

	return expired + stuck, nil
}

// VerifyEndUserNotification records the first time the notification reached the
// end user; a later call doesn't move the timestamp.
func (ds *Datastore) VerifyEndUserNotification(ctx context.Context, notificationUUID string, displayedAt time.Time) error {
	const updateStmt = `
UPDATE notifications_end_user
SET displayed_at = IF(displayed_at IS NULL, ?, displayed_at)
WHERE uuid = ?
`

	if _, err := ds.primary.ExecContext(ctx, updateStmt,
		displayedAt, notificationUUID,
	); err != nil {
		return ctxerr.Wrap(ctx, err, "verify end user notification")
	}
	return nil
}

// DelayEndUserNotification puts a notification back in the queue for a later
// attempt. It keeps execution_id, since the script it was dispatched with is
// still running and its result has to land somewhere. Expired and failed
// notifications are left alone so a delay can't revive one.
func (ds *Datastore) DelayEndUserNotification(ctx context.Context, notificationUUID string, nextAttemptAt time.Time) error {
	const updateStmt = `
UPDATE notifications_end_user
SET status = ?, next_attempt_at = ?, last_reason = ?
WHERE uuid = ?
	AND status NOT IN (?, ?)
	AND (expires_at IS NULL OR expires_at > NOW(6))
`

	if _, err := ds.primary.ExecContext(ctx, updateStmt,
		api.EndUserNotificationPending, nextAttemptAt, api.EndUserNotificationReasonDelayed, notificationUUID,
		api.EndUserNotificationExpired, api.EndUserNotificationFailed,
	); err != nil {
		return ctxerr.Wrap(ctx, err, "delay end user notification")
	}
	return nil
}

// SetEndUserNotificationOutcome records how a display attempt ended. A
// successful display is delegated to VerifyEndUserNotification, the only place
// displayed_at is written. A non-nil nextAttemptAt returns it to the queue;
// otherwise this was the last attempt, and either way the transition is skipped
// once past expires_at, so a late result can't revive an expired notification.
func (ds *Datastore) SetEndUserNotificationOutcome(ctx context.Context, notificationUUID string, outcome api.NotificationOutcome, nextAttemptAt *time.Time) error {
	const recordStmt = `
UPDATE notifications_end_user
SET last_exit_code = ?, last_reason = ?
WHERE uuid = ?
`

	var reason *string
	if outcome.Reason != "" {
		reason = &outcome.Reason
	}

	if _, err := ds.primary.ExecContext(ctx, recordStmt, outcome.ExitCode, reason, notificationUUID); err != nil {
		return ctxerr.Wrap(ctx, err, "record end user notification outcome")
	}

	if outcome.Displayed {
		return ds.VerifyEndUserNotification(ctx, notificationUUID, time.Now().UTC())
	}

	status := api.EndUserNotificationFailed
	if nextAttemptAt != nil {
		status = api.EndUserNotificationPending
	}

	const transitionStmt = `
UPDATE notifications_end_user
SET status = ?, next_attempt_at = ?
WHERE uuid = ? AND (expires_at IS NULL OR expires_at > NOW(6))
`

	if _, err := ds.primary.ExecContext(ctx, transitionStmt, status, nextAttemptAt, notificationUUID); err != nil {
		return ctxerr.Wrap(ctx, err, "set end user notification status")
	}
	return nil
}
