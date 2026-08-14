package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

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
	AND NOT EXISTS (
		SELECT 1 FROM end_user_notifications dispatched
		WHERE dispatched.host_id = eun.host_id AND dispatched.status = ?
			AND dispatched.displayed_at IS NULL
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
// queued as, joining the just-queued executions rather than sending one value per
// row.
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

// ExpireEndUserNotifications sets notifications past their expiry to expired, and
// returns how many. One already displayed is left alone regardless of expiry.
func (ds *Datastore) ExpireEndUserNotifications(ctx context.Context) (int64, error) {
	const updateStmt = `
UPDATE end_user_notifications
SET status = ?
WHERE expires_at IS NOT NULL AND expires_at <= NOW(6)
	AND status IN (?, ?)
	AND displayed_at IS NULL
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

// VerifyEndUserNotification records the first time the notification reached the
// end user; a later call doesn't move the timestamp.
func (ds *Datastore) VerifyEndUserNotification(ctx context.Context, notificationUUID string, displayedAt time.Time) error {
	const updateStmt = `
UPDATE end_user_notifications
SET displayed_at = IF(displayed_at IS NULL, ?, displayed_at)
WHERE uuid = ?
`

	if _, err := ds.writer(ctx).ExecContext(ctx, updateStmt,
		displayedAt, notificationUUID,
	); err != nil {
		return ctxerr.Wrap(ctx, err, "verify end user notification")
	}
	return nil
}

// DelayEndUserNotification puts a notification back in the queue for a later
// attempt, dropping the execution it was dispatched with.
func (ds *Datastore) DelayEndUserNotification(ctx context.Context, notificationUUID string, nextAttemptAt time.Time) error {
	const updateStmt = `
UPDATE end_user_notifications
SET status = ?, next_attempt_at = ?, last_reason = ?, execution_id = NULL
WHERE uuid = ?
`

	if _, err := ds.writer(ctx).ExecContext(ctx, updateStmt,
		fleet.EndUserNotificationPending, nextAttemptAt, fleet.EndUserNotificationReasonDelayed, notificationUUID,
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
func (ds *Datastore) SetEndUserNotificationOutcome(ctx context.Context, notificationUUID string, outcome fleet.NotificationOutcome, nextAttemptAt *time.Time) error {
	const recordStmt = `
UPDATE end_user_notifications
SET last_exit_code = ?, last_reason = ?
WHERE uuid = ?
`

	var reason *string
	if outcome.Reason != "" {
		reason = &outcome.Reason
	}

	if _, err := ds.writer(ctx).ExecContext(ctx, recordStmt, outcome.ExitCode, reason, notificationUUID); err != nil {
		return ctxerr.Wrap(ctx, err, "record end user notification outcome")
	}

	if outcome.Displayed {
		return ds.VerifyEndUserNotification(ctx, notificationUUID, time.Now().UTC())
	}

	status := fleet.EndUserNotificationFailed
	if nextAttemptAt != nil {
		status = fleet.EndUserNotificationPending
	}

	const transitionStmt = `
UPDATE end_user_notifications
SET status = ?, next_attempt_at = ?
WHERE uuid = ? AND (expires_at IS NULL OR expires_at > NOW(6))
`

	if _, err := ds.writer(ctx).ExecContext(ctx, transitionStmt, status, nextAttemptAt, notificationUUID); err != nil {
		return ctxerr.Wrap(ctx, err, "set end user notification status")
	}
	return nil
}

// activateNextScriptActivitiesForHosts activates the next upcoming activity for
// each host, but only where that activity is a script, in a fixed number of
// statements for the whole set rather than per host. Lives here rather than with
// the rest of the activity queue since it's specific to the notification
// dispatcher, not a general replacement for
// activateNextUpcomingActivityForBatchOfHosts.
//
// A host whose turn belongs to another activity type, or that already has one
// activated, is left for that other path to pick up.
func (ds *Datastore) activateNextScriptActivitiesForHosts(ctx context.Context, tx sqlx.ExtContext, hostIDs []uint) error {
	// Ranks every activity type so a script can't activate ahead of an install
	// queued earlier; only rank 1 matters since scripts never batch.
	findNextScriptsStmt := `
SELECT
	execution_id
FROM (
	SELECT
		execution_id,
		activity_type,
		activated_at,
		ROW_NUMBER() OVER (
			PARTITION BY host_id
			ORDER BY IF(activated_at IS NULL, 0, 1) DESC, priority DESC, created_at ASC
		) AS rank_in_host
	FROM
		upcoming_activities
	WHERE
		host_id IN (?)
		%s
) candidates
WHERE
	rank_in_host = 1 AND
	activity_type = 'script' AND
	activated_at IS NULL
`

	// same columns as activateNextScriptActivity, for many hosts at once
	const insertScriptResultsStmt = `
INSERT INTO
	host_script_results
(host_id, execution_id, script_content_id, output, script_id, policy_id,
	user_id, sync_request, setup_experience_script_id, is_internal)
SELECT
	ua.host_id,
	ua.execution_id,
	sua.script_content_id,
	'',
	sua.script_id,
	sua.policy_id,
	ua.user_id,
	COALESCE(ua.payload->'$.sync_request', 0),
	sua.setup_experience_script_id,
	COALESCE(ua.payload->'$.is_internal', 0)
FROM
	upcoming_activities ua
	INNER JOIN script_upcoming_activities sua
		ON sua.upcoming_activity_id = ua.id
WHERE
	ua.execution_id IN (?)
ORDER BY
	ua.priority DESC, ua.created_at ASC
`

	const markActivatedStmt = `
UPDATE upcoming_activities
SET
	activated_at = NOW()
WHERE
	execution_id IN (?)
`

	var (
		stmt string
		args []any
		err  error
	)
	if len(ds.testActivateSpecificNextActivities) > 0 {
		stmt, args, err = sqlx.In(fmt.Sprintf(findNextScriptsStmt, ` AND execution_id IN (?) `),
			hostIDs, ds.testActivateSpecificNextActivities)
	} else {
		stmt, args, err = sqlx.In(fmt.Sprintf(findNextScriptsStmt, ""), hostIDs)
	}
	if err != nil {
		return ctxerr.Wrap(ctx, err, "prepare statement to find next scripts to activate")
	}

	var execIDs []string
	if err := sqlx.SelectContext(ctx, tx, &execIDs, stmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "find next scripts to activate")
	}
	if len(execIDs) == 0 {
		return nil
	}

	stmt, args, err = sqlx.In(insertScriptResultsStmt, execIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "prepare insert to activate scripts")
	}
	if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "insert to activate scripts")
	}

	stmt, args, err = sqlx.In(markActivatedStmt, execIDs)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "prepare statement to mark upcoming activities as activated")
	}
	if _, err := tx.ExecContext(ctx, stmt, args...); err != nil {
		return ctxerr.Wrap(ctx, err, "mark upcoming activities as activated")
	}
	return nil
}
