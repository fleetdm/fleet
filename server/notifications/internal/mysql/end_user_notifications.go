// Package mysql is the notifications context's own datastore.
package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
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

type Datastore struct {
	primary *sqlx.DB
	replica *sqlx.DB
	logger  *slog.Logger
}

func NewDatastore(conns *platform_mysql.DBConnections, logger *slog.Logger) *Datastore {
	return &Datastore{primary: conns.Primary, replica: conns.Replica, logger: logger} // nolint:nilaway // conns is never nil by the time serve.go builds the notifications bounded context
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

// ListEndUserNotificationsToDispatch returns the oldest due notifications, one
// per host, skipping hosts that already have one in flight. limit caps the rows
// read, so it isn't the number of hosts returned.
//
// Only macOS hosts running fleetd qualify. Whether Fleet Desktop is installed
// and new enough is left to the script, which reports it as exit 100 or 101.
//
// The one-per-host cut happens in Go rather than a GROUP BY, which would have to
// group the whole backlog before LIMIT applied.
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

// SetEndUserNotificationsDispatched records the execution each notification was
// queued as.
func (ds *Datastore) SetEndUserNotificationsDispatched(ctx context.Context, notifications []*api.EndUserNotification) error {
	if len(notifications) == 0 {
		return nil
	}

	// Each notification has its own execution id, so they ride in as a derived
	// table to be set in one statement. Looking them up in upcoming_activities
	// instead would miss a host whose script already finished, since finishing
	// deletes that row. Only the number of SELECTs varies with the input; every
	// value is bound.
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
	eun.attempt_count = eun.attempt_count + 1,
	eun.last_reason = NULL
`, strings.Join(selectParts, " UNION ALL "))

	if _, err := ds.primary.ExecContext(ctx, updateStmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "set end user notifications dispatched")
	}
	return nil
}

// DeferEndUserNotificationsForHosts records that the notifications still
// waiting on these hosts are waiting because one of theirs is already going
// out. Called after marking that one dispatched, so it isn't pending anymore
// and doesn't mark itself.
func (ds *Datastore) DeferEndUserNotificationsForHosts(ctx context.Context, hostIDs []uint) error {
	if len(hostIDs) == 0 {
		return nil
	}

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
// dispatched ones that have gone unanswered for
// EndUserNotificationStuckDispatchTimeout, returning how many of both. One
// already displayed is left alone whatever its expiry says.
//
// Two statements rather than one OR across expires_at and updated_at, which
// could use neither index and scanned the table on every run. Expiry goes first
// so a row that qualifies for both is only counted once.
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

// VerifyEndUserNotification records the first time a notification reached its
// end user. Calling it again doesn't move the timestamp.
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

// DelayEndUserNotification returns a notification to the queue for a later
// attempt. A non-nil payload replaces its content, so a reminder stays the same
// notification instead of becoming a second one.
//
// execution_id stays, since the script already on the host still has a result to
// report. displayed_at is cleared, so the next send records its own display, and
// because a notification counts as in flight only while displayed_at is null:
// leaving it set would let a second one go out alongside this one.
//
// Expired and failed notifications are left alone, so a delay can't revive one.
func (ds *Datastore) DelayEndUserNotification(ctx context.Context, notificationUUID string, nextAttemptAt time.Time, payload json.RawMessage) error {
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

// SetEndUserNotificationOutcome records how an attempt to display ended. The
// exit code and reason are always recorded. A non-nil nextAttemptAt returns the
// notification to the queue, otherwise that was its last attempt.
//
// A successful display goes through VerifyEndUserNotification, the only place
// displayed_at is written. The status only moves while the notification is still
// within its expiry, so a late result can't revive an expired one.
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
