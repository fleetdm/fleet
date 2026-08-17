// Package mysql is the notifications context's own datastore.
package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/notifications/api"
	"github.com/fleetdm/fleet/v4/server/notifications/internal/types"
	platform_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type Datastore struct {
	primary *sqlx.DB
	replica *sqlx.DB
	logger  *slog.Logger
}

func NewDatastore(conns *platform_mysql.DBConnections, logger *slog.Logger) *Datastore {
	return &Datastore{primary: conns.Primary, replica: conns.Replica, logger: logger} // nolint:nilaway // serve.go always passes real connections
}

func (ds *Datastore) reader(ctx context.Context) *sqlx.DB {
	return ds.replica
}

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

func (ds *Datastore) GetEndUserNotificationByExecutionID(ctx context.Context, executionID string) (*api.EndUserNotification, error) {
	const getStmt = `SELECT ` + endUserNotificationColumns + ` FROM notifications_end_user eun WHERE eun.execution_id = ?`

	var notification api.EndUserNotification
	// primary, not the replica: fleetd can report a result before the write that
	// recorded this execution has replicated
	if err := sqlx.GetContext(ctx, ds.primary, &notification, getStmt, executionID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ctxerr.Wrap(ctx, &types.NotFoundError{Identifier: executionID})
		}
		return nil, ctxerr.Wrap(ctx, err, "get end user notification by execution id")
	}
	return &notification, nil
}

// ListEndUserNotificationsToDispatch returns the oldest notification due on each
// host. limit caps how many rows are read, which is not the same as how many
// notifications come back.
func (ds *Datastore) ListEndUserNotificationsToDispatch(ctx context.Context, limit int) ([]*api.EndUserNotification, error) {
	// The host_orbit_info join is what requires fleetd. Whether Fleet Desktop is
	// installed and new enough is left to the script, which reports it as exit
	// 100 or 101.
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
	// primary, not the replica: the caller keeps calling this until it comes back
	// empty, so it has to see the notifications it just marked dispatched
	if err := sqlx.SelectContext(ctx, ds.primary, &due, listStmt,
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

func (ds *Datastore) SetEndUserNotificationsDispatched(ctx context.Context, notifications []*api.EndUserNotification) error {
	// last_reason is cleared because whatever stopped the previous attempt no
	// longer describes this notification
	const updateStmt = `
UPDATE notifications_end_user
SET status = ?, execution_id = ?, attempt_count = attempt_count + 1, last_reason = NULL
WHERE uuid = ?
`

	for _, notification := range notifications {
		if notification.ExecutionID == nil || *notification.ExecutionID == "" {
			return ctxerr.New(ctx, "end user notification dispatched without an execution id")
		}
	}

	for _, notification := range notifications {
		if _, err := ds.primary.ExecContext(ctx, updateStmt,
			api.EndUserNotificationDispatched, *notification.ExecutionID, notification.UUID,
		); err != nil {
			return ctxerr.Wrap(ctx, err, "set end user notification dispatched")
		}
	}
	return nil
}

// DeferEndUserNotificationsForHosts marks the notifications still pending on
// these hosts as waiting behind the one that just went out to each of them.
func (ds *Datastore) DeferEndUserNotificationsForHosts(ctx context.Context, hostIDs []uint) error {
	if len(hostIDs) == 0 {
		return nil
	}

	// the one that just went out is dispatched by now, so it isn't pending and
	// doesn't mark itself
	stmt, args, err := sqlx.In(`
UPDATE notifications_end_user
SET last_reason = ?
WHERE host_id IN (?) AND status = ?
`, api.EndUserNotificationReasonDeferred, hostIDs, api.EndUserNotificationPending)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "build deferred end user notifications update")
	}
	if _, err := ds.primary.ExecContext(ctx, stmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "defer end user notifications")
	}
	return nil
}

// ExpireEndUserNotifications gives up on notifications past their expiry, and on
// ones sent so long ago that no result is coming, returning how many of both. One
// that already reached its end user is left alone whatever its expiry says.
func (ds *Datastore) ExpireEndUserNotifications(ctx context.Context) (int64, error) {
	// Two statements rather than one OR across expires_at and updated_at, which
	// would use neither index and scan the table on every run.
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

	// expiry runs first so a row that qualifies for both is only counted once
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

// VerifyEndUserNotification records the first time a notification reached its
// end user. Calling it again doesn't move the timestamp.
func (ds *Datastore) VerifyEndUserNotification(ctx context.Context, notificationUUID string, displayedAt time.Time) error {
	// Only while the notification is still out: a result arriving after the end
	// user delayed it belongs to a send that is already over.
	const updateStmt = `
UPDATE notifications_end_user
SET displayed_at = IF(displayed_at IS NULL, ?, displayed_at)
WHERE uuid = ? AND status = ?
`

	if _, err := ds.primary.ExecContext(ctx, updateStmt,
		displayedAt, notificationUUID, api.EndUserNotificationDispatched,
	); err != nil {
		return ctxerr.Wrap(ctx, err, "verify end user notification")
	}
	return nil
}

// DelayEndUserNotification puts a notification back in the queue for a later
// attempt. A non-nil payload replaces its content, so a reminder is the same
// notification rather than a second one.
func (ds *Datastore) DelayEndUserNotification(ctx context.Context, notificationUUID string, nextAttemptAt time.Time, payload json.RawMessage) error {
	// execution_id is left alone: the script already on the host still has a
	// result to report, and the next dispatch overwrites it. displayed_at is
	// cleared so the next send records its own display.
	//
	// Expired and failed notifications are excluded so a delay can't revive one.
	const updateStmt = `
UPDATE notifications_end_user
SET status = ?, next_attempt_at = ?, last_reason = ?, displayed_at = NULL,
	payload = COALESCE(?, payload)
WHERE uuid = ?
	AND status NOT IN (?, ?)
	AND (expires_at IS NULL OR expires_at > NOW(6))
`

	if _, err := ds.primary.ExecContext(ctx, updateStmt,
		api.EndUserNotificationPending, nextAttemptAt, api.EndUserNotificationReasonDelayed, payload, notificationUUID,
		api.EndUserNotificationExpired, api.EndUserNotificationFailed,
	); err != nil {
		return ctxerr.Wrap(ctx, err, "delay end user notification")
	}
	return nil
}

// SetEndUserNotificationOutcome records how an attempt to display ended. A
// non-nil nextAttemptAt puts the notification back in the queue, otherwise that
// was its last attempt.
func (ds *Datastore) SetEndUserNotificationOutcome(ctx context.Context, notificationUUID string, outcome api.NotificationOutcome, nextAttemptAt *time.Time) error {
	// the exit code and reason are recorded whatever state the notification is in,
	// since they describe an attempt that really happened
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

	// Same window as VerifyEndUserNotification above: only a notification that is
	// still out gets its schedule changed, so a late result can't revive an
	// expired one or undo a delay the end user asked for.
	const transitionStmt = `
UPDATE notifications_end_user
SET status = ?, next_attempt_at = ?
WHERE uuid = ? AND status = ? AND (expires_at IS NULL OR expires_at > NOW(6))
`

	if _, err := ds.primary.ExecContext(ctx, transitionStmt,
		status, nextAttemptAt, notificationUUID, api.EndUserNotificationDispatched,
	); err != nil {
		return ctxerr.Wrap(ctx, err, "set end user notification status")
	}
	return nil
}
